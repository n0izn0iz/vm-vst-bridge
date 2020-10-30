package main

/*
#cgo CPPFLAGS: -I/usr/include/vst36 -D__cdecl=""
*/
import "C"

import (
	"context"
	"unsafe"

	"github.com/n0izn0iz/vm-vst-bridge/memconn"
	"github.com/n0izn0iz/vm-vst-bridge/vstbridge"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// lol
const arrayLength = 1 << 30

type bridge struct {
	c   vstbridge.VSTBridgeClient
	l   *zap.Logger
	ctx context.Context
}

var b *bridge

//export NewBridge
func NewBridge() {
	if b != nil {
		panic("bridge already allocated")
	}
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	dialer := memconn.Dialer("/dev/shmem/ivshmem", 4096, 0, logger)
	ctx := context.TODO()
	conn, err := grpc.DialContext(ctx, "memconn", grpc.WithContextDialer(dialer), grpc.WithInsecure())
	// TODO: close conn properly
	if err != nil {
		panic(err)
	}
	b = &bridge{
		c:   vstbridge.NewVSTBridgeClient(conn),
		l:   logger,
		ctx: ctx,
	}
}

//export GetParameter
func GetParameter(index int32) float32 {
	rep, err := b.c.GetParameter(b.ctx, &vstbridge.GetParameter_Request{
		Index: index,
	})
	if err != nil {
		b.l.Error("GetParameter", zap.Error(err))
		return 0.5
	}
	return rep.GetValue()
}

//export SetParameter
func SetParameter(index int32, value float32) {
	_, err := b.c.SetParameter(b.ctx, &vstbridge.SetParameter_Request{
		Index: index,
		Value: value,
	})
	if err != nil {
		b.l.Error("SetParameter", zap.Error(err))
	}
}

//export ProcessReplacing
func ProcessReplacing(inputs **float32, outputs **float32, sampleFrames int32) {
	ins := (*[arrayLength]*float32)(unsafe.Pointer(inputs))
	in1 := (*[arrayLength]C.float)(unsafe.Pointer(ins[0]))
	in2 := (*[arrayLength]C.float)(unsafe.Pointer(ins[1]))

	data1 := make([]float32, sampleFrames)
	data2 := make([]float32, sampleFrames)

	for i := int32(0); i < sampleFrames; i++ {
		data1[i] = float32(in1[i])
		data2[i] = float32(in2[i])
	}

	rep, err := b.c.ProcessReplacing(b.ctx, &vstbridge.ProcessReplacing_Request{
		Inputs: []*vstbridge.FloatArray{
			&vstbridge.FloatArray{Data: data1},
			&vstbridge.FloatArray{Data: data2},
		},
		SampleFrames: sampleFrames,
	})
	if err != nil {
		b.l.Error("ProcessReplacing", zap.Error(err))
		return
	}

	rdata1 := rep.GetOutputs()[0].GetData()
	rdata2 := rep.GetOutputs()[1].GetData()

	outs := (*[arrayLength]*float32)(unsafe.Pointer(outputs))
	out1 := (*[arrayLength]C.float)(unsafe.Pointer(outs[0]))
	out2 := (*[arrayLength]C.float)(unsafe.Pointer(outs[1]))

	for i := int32(0); i < sampleFrames; i++ {
		out1[i] = C.float(rdata1[i])
		out2[i] = C.float(rdata2[i])
	}
}

//export ProcessDoubleReplacing
func ProcessDoubleReplacing(inputs **float64, outputs **float64, sampleFrames int32) {
	ins := (*[arrayLength]*float64)(unsafe.Pointer(inputs))
	in1 := (*[arrayLength]C.double)(unsafe.Pointer(ins[0]))
	in2 := (*[arrayLength]C.double)(unsafe.Pointer(ins[1]))

	data1 := make([]float64, sampleFrames)
	data2 := make([]float64, sampleFrames)

	for i := int32(0); i < sampleFrames; i++ {
		data1[i] = float64(in1[i])
		data2[i] = float64(in2[i])
	}

	rep, err := b.c.ProcessDoubleReplacing(b.ctx, &vstbridge.ProcessDoubleReplacing_Request{
		Inputs: []*vstbridge.DoubleArray{
			&vstbridge.DoubleArray{Data: data1},
			&vstbridge.DoubleArray{Data: data2},
		},
		SampleFrames: sampleFrames,
	})
	if err != nil {
		b.l.Error("ProcessReplacing", zap.Error(err))
		return
	}

	rdata1 := rep.GetOutputs()[0].GetData()
	rdata2 := rep.GetOutputs()[1].GetData()

	outs := (*[arrayLength]*float64)(unsafe.Pointer(outputs))
	out1 := (*[arrayLength]C.double)(unsafe.Pointer(outs[0]))
	out2 := (*[arrayLength]C.double)(unsafe.Pointer(outs[1]))

	for i := int32(0); i < sampleFrames; i++ {
		out1[i] = C.double(rdata1[i])
		out2[i] = C.double(rdata2[i])
	}
}

func main() {}
