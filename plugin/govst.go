package main

/*
#cgo CPPFLAGS: -I/usr/include/vst36 -D__cdecl=""
*/
import "C"

import (
	"context"
	"fmt"
	"unsafe"

	"github.com/n0izn0iz/vm-vst-bridge/memconn"
	"github.com/n0izn0iz/vm-vst-bridge/vstbridge"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// lol
const arrayLength = 1 << 30

type bridge struct {
	c         vstbridge.VSTBridgeClient
	l         *zap.Logger
	ctx       context.Context
	cancelCtx context.CancelFunc
	conn      *grpc.ClientConn
}

var bridges = make(map[uintptr]*bridge)

//export NewBridge
func NewBridge(cplug uintptr) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println("failed to init zap logger", cplug)
		logger = zap.NewNop()
	}
	logger.Debug("NewBridge", zap.Uintptr("cplug", cplug))

	if bridges[cplug] != nil {
		logger.Error("bridge already allocated", zap.Uintptr("cplug", cplug))
		panic("bridge already allocated")
	}

	dialer := memconn.Dialer("/dev/shm/ivshmem", 1000000, 0, logger)

	ctx, cancelCtx := context.WithCancel(context.Background())
	conn, err := grpc.DialContext(ctx, "memconn", grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		logger.Error("failed to dial", zap.Error(err))
		panic(err)
	}

	bridges[cplug] = &bridge{
		c:         vstbridge.NewVSTBridgeClient(conn),
		l:         logger,
		ctx:       ctx,
		cancelCtx: cancelCtx,
		conn:      conn,
	}
}

//export CloseBridge
func CloseBridge(cplug uintptr) {
	b, ok := bridges[cplug]
	if !ok {
		fmt.Println("warning: tried to close unallocated bridge", cplug)
		return
	}
	b.l.Debug("CloseBridge", zap.Uintptr("cplug", cplug))

	err := b.conn.Close()
	if err != nil {
		b.l.Error("failed to close conn", zap.Error(err))
	}

	b.cancelCtx()

	bridges[cplug] = nil
}

//export GetParameter
func GetParameter(cplug uintptr, index int32) float32 {
	b, ok := bridges[cplug]
	if !ok {
		return 0.5
	}

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
func SetParameter(cplug uintptr, index int32, value float32) {
	b, ok := bridges[cplug]
	if !ok {
		return
	}

	_, err := b.c.SetParameter(b.ctx, &vstbridge.SetParameter_Request{
		Index: index,
		Value: value,
	})
	if err != nil {
		b.l.Error("SetParameter", zap.Error(err))
	}
}

//export ProcessReplacing
func ProcessReplacing(cplug uintptr, inputs **float32, outputs **float32, sampleFrames int32) {
	b, ok := bridges[cplug]
	if !ok {
		return
	}

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
func ProcessDoubleReplacing(cplug uintptr, inputs **float64, outputs **float64, sampleFrames int32) {
	b, ok := bridges[cplug]
	if !ok {
		return
	}

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
		b.l.Error("ProcessDoubleReplacing", zap.Error(err))
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
