// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package models

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type RowT struct {
	State *RowStateT
	Hash  uint64
	Blob  bool
	GZip  bool
	Data  []byte
}

func (t *RowT) Pack(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	if t == nil {
		return 0
	}
	DataOffset := flatbuffers.UOffsetT(0)
	if t.Data != nil {
		DataOffset = builder.CreateByteString(t.Data)
	}
	RowStart(builder)
	StateOffset := t.State.Pack(builder)
	RowAddState(builder, StateOffset)
	RowAddHash(builder, t.Hash)
	RowAddBlob(builder, t.Blob)
	RowAddGZip(builder, t.GZip)
	RowAddData(builder, DataOffset)
	return RowEnd(builder)
}

func (rcv *Row) UnPackTo(t *RowT) {
	t.State = rcv.State(nil).UnPack()
	t.Hash = rcv.Hash()
	t.Blob = rcv.Blob()
	t.GZip = rcv.GZip()
	t.Data = rcv.DataBytes()
}

func (rcv *Row) UnPack() *RowT {
	if rcv == nil {
		return nil
	}
	t := &RowT{}
	rcv.UnPackTo(t)
	return t
}

type Row struct {
	_tab flatbuffers.Table
}

func GetRootAsRow(buf []byte, offset flatbuffers.UOffsetT) *Row {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &Row{}
	x.Init(buf, n+offset)
	return x
}

func (rcv *Row) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *Row) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *Row) State(obj *RowState) *RowState {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		x := o + rcv._tab.Pos
		if obj == nil {
			obj = new(RowState)
		}
		obj.Init(rcv._tab.Bytes, x)
		return obj
	}
	return nil
}

func (rcv *Row) Hash() uint64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.GetUint64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *Row) MutateHash(n uint64) bool {
	return rcv._tab.MutateUint64Slot(6, n)
}

func (rcv *Row) Blob() bool {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		return rcv._tab.GetBool(o + rcv._tab.Pos)
	}
	return false
}

func (rcv *Row) MutateBlob(n bool) bool {
	return rcv._tab.MutateBoolSlot(8, n)
}

func (rcv *Row) GZip() bool {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(10))
	if o != 0 {
		return rcv._tab.GetBool(o + rcv._tab.Pos)
	}
	return false
}

func (rcv *Row) MutateGZip(n bool) bool {
	return rcv._tab.MutateBoolSlot(10, n)
}

func (rcv *Row) Data(j int) byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(12))
	if o != 0 {
		a := rcv._tab.Vector(o)
		return rcv._tab.GetByte(a + flatbuffers.UOffsetT(j*1))
	}
	return 0
}

func (rcv *Row) DataLength() int {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(12))
	if o != 0 {
		return rcv._tab.VectorLen(o)
	}
	return 0
}

func (rcv *Row) DataBytes() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(12))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *Row) MutateData(j int, n byte) bool {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(12))
	if o != 0 {
		a := rcv._tab.Vector(o)
		return rcv._tab.MutateByte(a+flatbuffers.UOffsetT(j*1), n)
	}
	return false
}

func RowStart(builder *flatbuffers.Builder) {
	builder.StartObject(5)
}
func RowAddState(builder *flatbuffers.Builder, State flatbuffers.UOffsetT) {
	builder.PrependStructSlot(0, flatbuffers.UOffsetT(State), 0)
}
func RowAddHash(builder *flatbuffers.Builder, Hash uint64) {
	builder.PrependUint64Slot(1, Hash, 0)
}
func RowAddBlob(builder *flatbuffers.Builder, Blob bool) {
	builder.PrependBoolSlot(2, Blob, false)
}
func RowAddGZip(builder *flatbuffers.Builder, GZip bool) {
	builder.PrependBoolSlot(3, GZip, false)
}
func RowAddData(builder *flatbuffers.Builder, Data flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(4, flatbuffers.UOffsetT(Data), 0)
}
func RowStartDataVector(builder *flatbuffers.Builder, numElems int) flatbuffers.UOffsetT {
	return builder.StartVector(1, numElems, 1)
}
func RowEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}
