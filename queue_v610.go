package fdbx

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
)

// PunchSize - размер ожидания в случае отсутствия задач
var PunchSize = time.Minute

func newV610queue(conn *v610Conn, rtp RecordType, prefix []byte) (*v610queue, error) {
	return &v610queue{
		cn:  conn,
		rtp: rtp,
		pf:  prefix,
	}, nil
}

type v610queue struct {
	pf  []byte
	rtp RecordType
	cn  *v610Conn
}

func (q *v610queue) Ack(db DB, ids ...[]byte) error {
	var ok bool
	var db610 *v610db

	if db == nil {
		return ErrNullDB.WithStack()
	}

	if db610, ok = db.(*v610db); !ok {
		return ErrIncompatibleDB.WithStack()
	}

	for i := range ids {
		db610.tx.Clear(q.lostKey(ids[i], []byte{byte(len(ids[i]))}))
	}

	return nil
}

func (q *v610queue) Pub(db DB, when time.Time, ids ...[]byte) (err error) {
	var ok bool
	var db610 *v610db

	if db == nil {
		return ErrNullDB.WithStack()
	}

	if db610, ok = db.(*v610db); !ok {
		return ErrIncompatibleDB.WithStack()
	}

	if when.IsZero() {
		when = time.Now()
	}

	delay := make([]byte, 8)
	binary.BigEndian.PutUint64(delay, uint64(when.UnixNano()))

	// set task
	for i := range ids {
		db610.tx.Set(q.dataKey(delay, ids[i], []byte{byte(len(ids[i]))}), nil)
	}

	// update watch
	db610.tx.Set(q.watchKey(), delay)
	return nil
}

func (q *v610queue) Sub(ctx context.Context) (<-chan Record, <-chan error) {
	modc := make(chan Record)
	errc := make(chan error, 1)

	go func() {
		var m Record
		var err error

		defer close(errc)
		defer close(modc)
		defer func() {
			if rec := recover(); rec != nil {

				if err, ok := rec.(error); ok {
					errc <- ErrQueuePanic.WithReason(err)
				} else {
					errc <- ErrQueuePanic.WithReason(fmt.Errorf("%+v", rec))
				}
			}
		}()

		for {

			if m, err = q.SubOne(ctx); err != nil {
				errc <- err
				return
			}

			if m == nil {
				continue
			}

			select {
			case modc <- m:
			case <-ctx.Done():
				errc <- ctx.Err()
				return
			}
		}
	}()

	return modc, errc
}

func (q *v610queue) SubOne(ctx context.Context) (_ Record, err error) {
	var list []Record

	if list, err = q.SubList(ctx, 1); err != nil {
		return
	}

	if len(list) > 0 {
		return list[0], nil
	}

	return nil, ErrRecordNotFound.WithStack()
}

func (q *v610queue) nextTaskDistance() (d time.Duration, err error) {
	d = PunchSize

	rng := fdb.KeyRange{
		Begin: q.dataKey(),
		End:   q.dataKey(tail),
	}

	_, err = q.cn.fdb.ReadTransact(func(tx fdb.ReadTransaction) (_ interface{}, e error) {
		rows := tx.GetRange(rng, fdb.RangeOptions{Mode: fdb.StreamingModeWantAll, Limit: 1}).GetSliceOrPanic()
		if len(rows) > 0 {
			pflen := 4 + len(q.pf)
			iwhen := int64(binary.BigEndian.Uint64(rows[0].Key[pflen : pflen+8]))
			if wait := time.Unix(0, iwhen).Sub(time.Now()); wait > 0 {
				d = wait + time.Millisecond
			}
		}
		return nil, nil
	})
	return d, err
}

func (q *v610queue) waitTask(ctx context.Context, wait fdb.FutureNil) (err error) {
	var punch time.Duration

	if wait == nil {
		return nil
	}

	if punch, err = q.nextTaskDistance(); err != nil {
		return
	}

	wc := make(chan struct{}, 1)
	go func() {
		defer close(wc)
		wait.BlockUntilReady()
		wc <- struct{}{}
	}()

	wctx, cancel := context.WithTimeout(ctx, punch)
	defer cancel()

	select {
	case <-wc:
	case <-wctx.Done():
		wait.Cancel()
	}
	return nil
}

func (q *v610queue) SubList(ctx context.Context, limit uint) (list []Record, err error) {
	var ids [][]byte
	var recs []Record
	var wait fdb.FutureNil

	for len(list) == 0 {

		if err = q.waitTask(ctx, wait); err != nil {
			return
		}

		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// select ids in self tx
		_, err = q.cn.fdb.Transact(func(tx fdb.Transaction) (_ interface{}, e error) {
			var rows []fdb.KeyValue

			now := make([]byte, 8)
			ids = make([][]byte, 0, limit)
			binary.BigEndian.PutUint64(now, uint64(time.Now().UnixNano()))

			rng := fdb.KeyRange{
				Begin: q.dataKey(),
				End:   q.dataKey(now),
			}

			lim := int(limit) - len(list)

			if lim < 1 {
				return nil, nil
			}

			// must lock this range from parallel reads
			if e = tx.AddWriteConflictRange(rng); e != nil {
				return
			}

			opts := fdb.RangeOptions{Mode: fdb.StreamingModeWantAll, Limit: lim}

			if rows = tx.GetRange(rng, opts).GetSliceOrPanic(); len(rows) == 0 {
				wait = tx.Watch(q.watchKey())
				return nil, nil
			}

			for i := range rows {
				rid := getRowID(rows[i].Key)
				ids = append(ids, rid)

				// move to lost
				tx.Set(q.lostKey(rid, []byte{byte(len(rid))}), nil)
				tx.Clear(rows[i].Key)
			}

			return nil, nil
		})
		if err != nil {
			return
		}

		if len(ids) == 0 {
			continue
		}

		if recs, err = q.loadRecs(ids); err != nil {
			return
		}

		list = append(list, recs...)
	}

	return list, nil
}

func (q *v610queue) loadRecs(ids [][]byte) (list []Record, err error) {
	list = make([]Record, len(ids))

	for i := range ids {
		if list[i], err = q.rtp.New(ids[i]); err != nil {
			return
		}
	}

	_, err = q.cn.fdb.ReadTransact(func(rtx fdb.ReadTransaction) (interface{}, error) {
		return nil, loadRecords(q.cn.db, rtx, nil, list...)
	})
	return list, err
}

func (q *v610queue) GetLost(limit uint, filter Predicat) (list []Record, err error) {
	opt := fdb.RangeOptions{Limit: int(limit)}
	rng := fdb.KeyRange{
		Begin: q.lostKey(),
		End:   q.lostKey(tail),
	}

	_, err = q.cn.fdb.ReadTransact(func(rtx fdb.ReadTransaction) (_ interface{}, exp error) {
		list, _, exp = getRange(q.cn.db, rtx, rng, opt, q.rtp, filter)
		return
	})
	return list, err
}

func (q *v610queue) CheckLost(db DB, ids ...[]byte) ([]bool, error) {
	var ok bool
	var db610 *v610db

	res := make([]bool, len(ids))
	fbs := make([]fdb.FutureByteSlice, len(ids))

	if db == nil {
		return nil, ErrNullDB.WithStack()
	}

	if db610, ok = db.(*v610db); !ok {
		return nil, ErrIncompatibleDB.WithStack()
	}

	for i := range ids {
		fbs[i] = db610.tx.Get(q.lostKey(ids[i], []byte{byte(len(ids[i]))}))
	}

	for i := range fbs {
		res[i] = fbs[i].MustGet() != nil
	}

	return res, nil
}

func (q *v610queue) dataKey(pts ...[]byte) fdb.Key { return q.key(0x00, pts...) }
func (q *v610queue) lostKey(pts ...[]byte) fdb.Key { return q.key(0x01, pts...) }
func (q *v610queue) watchKey() fdb.Key             { return q.key(0x02) }

func (q *v610queue) key(prefix byte, pts ...[]byte) fdb.Key {
	parts := append([][]byte{q.pf, {prefix}}, pts...)
	return fdbKey(q.cn.db, q.rtp.ID, parts...)
}

func (q *v610queue) Stat() (wait, lost int, err error) {
	opt := fdb.RangeOptions{Mode: fdb.StreamingModeWantAll}
	dataRng := fdb.KeyRange{
		Begin: q.dataKey(),
		End:   q.dataKey(tail),
	}
	lostRng := fdb.KeyRange{
		Begin: q.lostKey(),
		End:   q.lostKey(tail),
	}

	_, err = q.cn.fdb.ReadTransact(func(rtx fdb.ReadTransaction) (interface{}, error) {
		wait = len(rtx.GetRange(dataRng, opt).GetSliceOrPanic())
		lost = len(rtx.GetRange(lostRng, opt).GetSliceOrPanic())
		return nil, nil
	})
	return wait, lost, err
}
