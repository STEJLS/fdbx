package fdbx_test

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/shestakovda/fdbx"
	"github.com/stretchr/testify/assert"
)

// current test settings
var (
	TestVersion    = fdbx.ConnVersion610
	TestDatabase   = uint16(0x0102)
	TestCollection = uint16(0x0304)
)

func TestDB(t *testing.T) {
	conn, err := fdbx.NewConn(TestDatabase, TestVersion)
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	defer conn.ClearDB()

	rec1 := newTestRecord()
	rec2 := &testRecord{ID: rec1.ID}

	rec3 := newTestRecord()
	rec4 := &testRecord{ID: rec3.ID}

	// ******** Key/Value ********

	uid := uuid.New()
	key := uid[:8]
	val := uid[8:16]

	assert.NoError(t, conn.Tx(func(db fdbx.DB) error {
		v, e := db.Get(TestCollection, key)
		assert.Empty(t, v)
		return e
	}))

	assert.NoError(t, conn.Tx(func(db fdbx.DB) error { return db.Set(TestCollection, key, val) }))

	assert.NoError(t, conn.Tx(func(db fdbx.DB) error {
		v, e := db.Get(TestCollection, key)
		assert.Equal(t, val, v)
		return e
	}))

	assert.NoError(t, conn.Tx(func(db fdbx.DB) error { return db.Del(TestCollection, key) }))

	assert.NoError(t, conn.Tx(func(db fdbx.DB) error {
		v, e := db.Get(TestCollection, key)
		assert.Empty(t, v)
		return e
	}))

	// ******** Record ********

	assert.True(t, errors.Is(conn.Tx(func(db fdbx.DB) error { return db.Load(rec1, rec3) }), fdbx.ErrRecordNotFound))
	assert.True(t, errors.Is(conn.Tx(func(db fdbx.DB) error { return db.Load(rec3, rec1) }), fdbx.ErrRecordNotFound))

	assert.NoError(t, conn.Tx(func(db fdbx.DB) error { return db.Save(rec1, rec3) }))

	assert.NoError(t, conn.Tx(func(db fdbx.DB) error { return db.Load(rec2, rec4) }))
	assert.NoError(t, conn.Tx(func(db fdbx.DB) error { return db.Load(rec4, rec2) }))

	assert.NoError(t, conn.Tx(func(db fdbx.DB) error {
		list, err := db.Select(TestCollection, testRecordFabric)
		assert.NoError(t, err)
		assert.Len(t, list, 2)
		return err
	}))

	assert.NoError(t, conn.Tx(func(db fdbx.DB) error { return db.Drop(rec1, rec3) }))

	assert.True(t, errors.Is(conn.Tx(func(db fdbx.DB) error { return db.Load(rec2, rec4) }), fdbx.ErrRecordNotFound))
	assert.True(t, errors.Is(conn.Tx(func(db fdbx.DB) error { return db.Load(rec4, rec2) }), fdbx.ErrRecordNotFound))

	assert.Equal(t, rec1, rec2)
	assert.Equal(t, rec3, rec4)
}

func TestCursor(t *testing.T) {
	conn, err := fdbx.NewConn(TestDatabase, TestVersion)
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	defer conn.ClearDB()

	records := make([]fdbx.Record, 10)
	for i := range records {
		records[i] = newTestRecord()
	}

	assert.NoError(t, conn.Tx(func(db fdbx.DB) error { return db.Save(records...) }))

	cur, err := conn.Cursor(TestCollection, testRecordFabric, nil, 3)
	assert.NoError(t, err)
	assert.NotNil(t, cur)
	assert.False(t, cur.Empty())

	defer func() { assert.NoError(t, cur.Close()) }()

	// ********* all in *********

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	recc, errc := cur.Select(ctx)

	recs := make([]fdbx.Record, 0, 10)
	for rec := range recc {
		recs = append(recs, rec)
	}

	errs := make([]error, 0)
	for err := range errc {
		errs = append(errs, err)
	}

	assert.Len(t, errs, 0)
	assert.Len(t, recs, 10)
	assert.True(t, cur.Empty())
	assert.NoError(t, cur.Close())

	// ********* steps *********

	recl := make([]fdbx.Record, 0, 10)
	rect := make([]fdbx.Record, 0, 10)

	// page size = 3
	cur, err = conn.Cursor(TestCollection, testRecordFabric, nil, 3)
	assert.NoError(t, err)
	assert.NotNil(t, cur)
	assert.False(t, cur.Empty())

	// pos: 0 -> (load 3) -> 3
	assert.NoError(t, conn.Tx(func(db fdbx.DB) error { recl, err = cur.Next(db, 0); return err }))
	assert.Len(t, recl, 3)
	assert.False(t, cur.Empty())
	rect = append(rect, recl...)

	// pos: 3 -> (skip 3) -> 6 -> (load 3) -> 9
	assert.NoError(t, conn.Tx(func(db fdbx.DB) error { recl, err = cur.Next(db, 1); return err }))
	assert.Len(t, recl, 3)
	assert.False(t, cur.Empty())

	// pos: 9 -> (skip -6) -> 3 -> (load 3) -> 6
	assert.NoError(t, conn.Tx(func(db fdbx.DB) error { recl, err = cur.Prev(db, 1); return err }))
	assert.Len(t, recl, 3)
	assert.False(t, cur.Empty())
	rect = append(rect, recl...)

	// pos: 6 -> (load 3) -> 9
	assert.NoError(t, conn.Tx(func(db fdbx.DB) error { recl, err = cur.Next(db, 0); return err }))
	assert.Len(t, recl, 3)
	assert.False(t, cur.Empty())
	rect = append(rect, recl...)

	// pos: 9 -> (load 3) -> 10
	assert.NoError(t, conn.Tx(func(db fdbx.DB) error { recl, err = cur.Next(db, 0); return err }))
	assert.Len(t, recl, 1)
	assert.True(t, cur.Empty())
	rect = append(rect, recl...)

	assert.Equal(t, recs, rect)
}

func testRecordFabric(id []byte) (fdbx.Record, error) { return &testRecord{ID: id}, nil }

func newTestRecord() *testRecord {
	uid := uuid.New()
	str := uid.String()
	num := binary.BigEndian.Uint64(uid[:8])
	flt := float64(binary.BigEndian.Uint64(uid[8:16]))

	return &testRecord{
		ID:      uid[:],
		Name:    str,
		Number:  num,
		Decimal: flt,
		Logic:   flt > float64(num),
		Data:    uid[:],
		Strs:    []string{str, str, str},
	}
}

type testRecord struct {
	ID      []byte   `json:"id"`
	Name    string   `json:"name"`
	Number  uint64   `json:"number"`
	Decimal float64  `json:"decimal"`
	Logic   bool     `json:"logic"`
	Data    []byte   `json:"data"`
	Strs    []string `json:"strs"`
}

func (r *testRecord) FdbxID() []byte               { return r.ID }
func (r *testRecord) FdbxType() uint16             { return TestCollection }
func (r *testRecord) FdbxMarshal() ([]byte, error) { return json.Marshal(r) }
func (r *testRecord) FdbxUnmarshal(b []byte) error { return json.Unmarshal(b, r) }

// func TestConn(t *testing.T) {
// 	const db = 1
// 	const skey = "test key"
// 	const skey2 = "test key 2"
// 	const skey3 = "test key 3"
// 	const skey4 = "test key 4"
// 	const cid = 2
// 	const qtype = 3

// 	var buf []byte
// 	var tkey = []byte(skey)
// 	var tdata = []byte("test buf")
// 	var tdata2 = []byte("test buf 2")
// 	var tdata3 = []byte("test buf 3")
// 	var tdata4 = []byte("test buf 4")
// 	var terr = errors.New("test err")

// 	// ************ MockConn ************

// 	mc, err := fdbx.NewConn(db, fdbx.ConnVersionMock)
// 	assert.NoError(t, err)
// 	assert.NotNil(t, mc)
// 	assert.NoError(t, mc.Tx(func(db fdbx.DB) error { return nil }))
// 	q, err := mc.Queue(0, nil)
// 	assert.NoError(t, err)
// 	assert.NotNil(t, q)

// 	// ************ v610Conn ************

// 	c1, err := fdbx.NewConn(db, fdbx.ConnVersion610)
// 	assert.NoError(t, err)
// 	assert.NotNil(t, c1)

// 	// ************ Clear All ************

// 	assert.NoError(t, c1.ClearDB())

// 	// ************ Key ************

// 	_, err = c1.Key(0, nil)
// 	assert.True(t, errors.Is(err, fdbx.ErrEmptyID))

// 	key, err := c1.Key(cid, tkey)
// 	assert.NoError(t, err)
// 	assert.Equal(t, fdb.Key(append([]byte{0, 1, 0, 2}, tkey...)), key)

// 	// ************ MKey ************

// 	_, err = c1.MKey(nil)
// 	assert.True(t, errors.Is(err, fdbx.ErrNullModel))

// 	key, err = c1.MKey(&testModel{key: skey, cid: cid})
// 	assert.NoError(t, err)
// 	assert.Equal(t, fdb.Key(append([]byte{0, 1, 0, 2}, tkey...)), key)

// 	// ************ DB.Set ************

// 	err = c1.Tx(func(db fdbx.DB) (e error) { return db.Set(0, nil, nil) })
// 	assert.True(t, errors.Is(err, fdbx.ErrEmptyID))

// 	err = c1.Tx(func(db fdbx.DB) (e error) { return db.Set(cid, tkey, tdata) })
// 	assert.NoError(t, err)

// 	// ************ DB.Get ************

// 	err = c1.Tx(func(db fdbx.DB) (e error) { _, e = db.Get(0, nil); return e })
// 	assert.True(t, errors.Is(err, fdbx.ErrEmptyID))

// 	err = c1.Tx(func(db fdbx.DB) (e error) { buf, e = db.Get(cid, tkey); return e })
// 	assert.NoError(t, err)
// 	assert.Equal(t, tdata, buf)

// 	// ************ DB.Del ************

// 	err = c1.Tx(func(db fdbx.DB) (e error) { return db.Del(0, nil) })
// 	assert.True(t, errors.Is(err, fdbx.ErrEmptyID))

// 	err = c1.Tx(func(db fdbx.DB) (e error) { return db.Del(cid, tkey) })
// 	assert.NoError(t, err)

// 	// ************ DB.Save ************

// 	err = c1.Tx(func(db fdbx.DB) (e error) { return db.Save(nil) })
// 	assert.True(t, errors.Is(err, fdbx.ErrNullModel))

// 	m := &testModel{key: skey, cid: cid, err: terr}
// 	err = c1.Tx(func(db fdbx.DB) (e error) { return db.Save(m) })
// 	assert.True(t, errors.Is(err, terr))

// 	// ************ DB.Load ************

// 	err = c1.Tx(func(db fdbx.DB) (e error) { return db.Load(nil) })
// 	assert.True(t, errors.Is(err, fdbx.ErrNullModel))

// 	// ************ DB.Save/DB.Load ************

// 	m1 := &testModel{key: skey, cid: cid, buf: tdata}
// 	m2 := &testModel{key: skey2, cid: cid, buf: tdata2}
// 	m5 := &testModel{key: skey, cid: cid}
// 	m6 := &testModel{key: skey2, cid: cid}

// 	k1, err := c1.MKey(m1)
// 	assert.NoError(t, err)
// 	k2, err := c1.MKey(m2)
// 	assert.NoError(t, err)
// 	k3, err := c1.MKey(m5)
// 	assert.NoError(t, err)
// 	k4, err := c1.MKey(m6)
// 	assert.NoError(t, err)
// 	assert.Equal(t, k1, k3)
// 	assert.Equal(t, k2, k4)

// 	fdbx.GZipSize = 0
// 	fdbx.ChunkSize = 2

// 	assert.NoError(t, c1.Tx(func(db fdbx.DB) (e error) {
// 		if e = db.Save(m1, m2); e != nil {
// 			return
// 		}

// 		return db.Load(m5, m6)
// 	}))

// 	assert.Equal(t, m1.Dump(), m5.Dump())
// 	assert.Equal(t, m2.Dump(), m6.Dump())

// 	// ************ Queue Pub/Sub ************

// 	fdbx.PunchSize = 50 * time.Millisecond

// 	fab := func(id []byte) (fdbx.Model, error) { return &testModel{key: string(id), cid: cid}, nil }
// 	queue, err := c1.Queue(qtype, fab)
// 	assert.NoError(t, err)

// 	m3 := &testModel{key: skey3, cid: cid, buf: tdata3}
// 	m4 := &testModel{key: skey4, cid: cid, buf: tdata4}

// 	// publish 3 tasks
// 	assert.NoError(t, c1.Tx(func(db fdbx.DB) (e error) {
// 		if e = db.Save(m3, m4); e != nil {
// 			return
// 		}

// 		assert.True(t, errors.Is(queue.Pub(nil, m1, time.Now()), fdbx.ErrNullDB))
// 		assert.True(t, errors.Is(queue.Pub(db, nil, time.Now()), fdbx.ErrNullModel))

// 		if e = queue.Pub(db, m1, time.Now()); e != nil {
// 			return
// 		}

// 		time.Sleep(time.Millisecond)
// 		if e = queue.Pub(db, m2, time.Now()); e != nil {
// 			return
// 		}

// 		time.Sleep(time.Millisecond)
// 		if e = queue.Pub(db, m3, time.Now()); e != nil {
// 			return
// 		}

// 		return nil
// 	}))

// 	var wg sync.WaitGroup
// 	wg.Add(2)

// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
// 	defer cancel()

// 	wrk := func(s string) {
// 		defer wg.Done()
// 		mods, err := queue.SubList(ctx, 2)
// 		assert.NoError(t, err)
// 		assert.Len(t, mods, 2)
// 		assert.NoError(t, c1.Tx(func(db fdbx.DB) error {
// 			assert.True(t, errors.Is(queue.Ack(nil, nil), fdbx.ErrNullDB))
// 			assert.True(t, errors.Is(queue.Ack(db, nil), fdbx.ErrNullModel))

// 			for i := range mods {
// 				assert.NoError(t, queue.Ack(db, mods[i]))
// 			}
// 			return nil
// 		}))
// 	}

// 	// 1 worker get 2 tasks and exit
// 	// 2 worker get 1 task and wait
// 	go wrk(skey2)
// 	go wrk(skey4)

// 	// publish 4 task
// 	assert.NoError(t, c1.Tx(func(db fdbx.DB) (e error) { return queue.Pub(db, m4, time.Now()) }))

// 	// wait all
// 	wg.Wait()

// 	wg.Add(1)
// 	go func() {
// 		defer wg.Done()
// 		m, err := queue.SubOne(ctx)
// 		assert.NoError(t, err)
// 		assert.Equal(t, skey, string(m.ID()))
// 		assert.Equal(t, string(tdata), string(m.Dump()))
// 		assert.NoError(t, c1.Tx(func(db fdbx.DB) (e error) { return queue.Ack(db, m) }))
// 	}()

// 	// publish 1 task
// 	assert.NoError(t, c1.Tx(func(db fdbx.DB) (e error) { return queue.Pub(db, m1, time.Now()) }))

// 	// wait all
// 	wg.Wait()

// 	modc, errc := queue.Sub(ctx)

// 	// publish 1 task
// 	assert.NoError(t, c1.Tx(func(db fdbx.DB) (e error) {
// 		if e = queue.Pub(db, m1, time.Now()); e != nil {
// 			return
// 		}

// 		time.Sleep(time.Millisecond)
// 		if e = queue.Pub(db, m2, time.Now()); e != nil {
// 			return
// 		}

// 		time.Sleep(time.Millisecond)
// 		return queue.Pub(db, m3, time.Now())
// 	}))

// 	mods := make([]fdbx.Model, 0, 3)
// 	errs := make([]error, 0, 3)

// 	for m := range modc {
// 		mods = append(mods, m)
// 	}

// 	for e := range errc {
// 		errs = append(errs, e)
// 	}

// 	assert.Len(t, mods, 3)
// 	assert.Len(t, errs, 1)
// 	assert.True(t, errors.Is(errs[0], context.DeadlineExceeded))
// 	assert.Equal(t, skey, string(mods[0].ID()))
// 	assert.Equal(t, skey2, string(mods[1].ID()))
// 	assert.Equal(t, skey3, string(mods[2].ID()))

// 	// ************ DB.Select ************

// 	var list []fdbx.Model

// 	predicat := func(buf []byte) (bool, error) {
// 		if string(buf) == string(tdata2) {
// 			return false, nil
// 		}
// 		return true, nil
// 	}

// 	assert.NoError(t, c1.Tx(func(db fdbx.DB) (e error) {
// 		list, e = db.Select(
// 			cid, fab,
// 			fdbx.Limit(3),
// 			fdbx.PrefixLen(4),
// 			fdbx.Filter(predicat),
// 			fdbx.GTE([]byte{0x00}),
// 			fdbx.LT([]byte{0xFF}),
// 		)
// 		return
// 	}))
// 	assert.Len(t, list, 2)
// 	assert.Equal(t, skey, string(list[0].ID()))
// 	assert.Equal(t, skey3, string(list[1].ID()))
// 	assert.Equal(t, string(tdata), string(list[0].Dump()))
// 	assert.Equal(t, string(tdata3), string(list[1].Dump()))

// 	// ************ DB.Drop ************

// 	assert.NoError(t, c1.Tx(func(db fdbx.DB) (e error) { return db.Drop(m1, m2, m3, m4) }))

// 	// assert.False(t, true)
// }

// func BenchmarkSaveOneBig(b *testing.B) {
// 	b.StopTimer()

// 	const db = 1
// 	const cid = 2

// 	// overvalue for disable gzipping
// 	fdbx.GZipSize = 10000000
// 	fdbx.ChunkSize = fdbx.MaxChunkSize

// 	c, err := fdbx.NewConn(db, fdbx.ConnVersion610)
// 	assert.NoError(b, err)
// 	assert.NotNil(b, c)

// 	// 9 Mb no gzipped records
// 	uid := uuid.New()
// 	m := &testModel{key: uid.String(), cid: cid, buf: bytes.Repeat(uid[:9], 1024*1024)}

// 	b.StartTimer()

// 	for i := 0; i < b.N; i++ {
// 		m.key = uuid.New().String()
// 		assert.NoError(b, c.Tx(func(db fdbx.DB) (e error) { return db.Save(m) }))
// 	}
// }

// func BenchmarkSaveMultiSmalls(b *testing.B) {
// 	b.StopTimer()

// 	const db = 1
// 	const cid = 2

// 	// overvalue for disable gzipping
// 	fdbx.GZipSize = 10000000
// 	fdbx.ChunkSize = fdbx.MaxChunkSize

// 	c, err := fdbx.NewConn(db, fdbx.ConnVersion610)
// 	assert.NoError(b, err)
// 	assert.NotNil(b, c)

// 	// 8K with 1Kb no gzipped models
// 	count := 8000
// 	models := make([]fdbx.Model, count)

// 	for i := 0; i < count; i++ {
// 		uid := uuid.New()
// 		models[i] = &testModel{key: uid.String(), cid: cid, buf: bytes.Repeat(uid[:8], 128)}
// 	}

// 	b.StartTimer()

// 	for i := 0; i < b.N; i++ {
// 		assert.NoError(b, c.Tx(func(db fdbx.DB) (e error) { return db.Save(models...) }))
// 	}
// }

// type testModel struct {
// 	err error
// 	cid uint16
// 	key string
// 	buf []byte
// }

// func (m *testModel) ID() []byte                   { return []byte(m.key) }
// func (m *testModel) Collection() uint16           { return m.cid }
// func (m *testModel) MarshalFdbx() ([]byte, error) { return m.buf, m.err }
// func (m *testModel) UnmarshalFdbx(d []byte) error { m.buf = d; return m.err }
