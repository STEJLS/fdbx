// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package models

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type File struct {
	_tab flatbuffers.Table
}

func GetRootAsFile(buf []byte, offset flatbuffers.UOffsetT) *File {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &File{}
	x.Init(buf, n+offset)
	return x
}

func (rcv *File) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *File) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *File) Path() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *File) MTime() uint64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.GetUint64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *File) MutateMTime(n uint64) bool {
	return rcv._tab.MutateUint64Slot(6, n)
}

func (rcv *File) Data(j int) byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		a := rcv._tab.Vector(o)
		return rcv._tab.GetByte(a + flatbuffers.UOffsetT(j*1))
	}
	return 0
}

func (rcv *File) DataLength() int {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		return rcv._tab.VectorLen(o)
	}
	return 0
}

func (rcv *File) DataBytes() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *File) MutateData(j int, n byte) bool {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		a := rcv._tab.Vector(o)
		return rcv._tab.MutateByte(a+flatbuffers.UOffsetT(j*1), n)
	}
	return false
}

func FileStart(builder *flatbuffers.Builder) {
	builder.StartObject(3)
}
func FileAddPath(builder *flatbuffers.Builder, Path flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(0, flatbuffers.UOffsetT(Path), 0)
}
func FileAddMTime(builder *flatbuffers.Builder, MTime uint64) {
	builder.PrependUint64Slot(1, MTime, 0)
}
func FileAddData(builder *flatbuffers.Builder, Data flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(2, flatbuffers.UOffsetT(Data), 0)
}
func FileStartDataVector(builder *flatbuffers.Builder, numElems int) flatbuffers.UOffsetT {
	return builder.StartVector(1, numElems, 1)
}
func FileEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}
