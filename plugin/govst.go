package main

/*
#cgo windows CPPFLAGS: -IVST2_SDK -D__cdecl=""
#cgo linux CPPFLAGS: -I/usr/include/vst36 -D__cdecl=""
*/
import "C"

import (
	"context"
	"fmt"
	"unsafe"

	"github.com/n0izn0iz/vm-vst-bridge/pkg/memconn"
	"github.com/n0izn0iz/vm-vst-bridge/pkg/vstbridge"
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

var bridges = make(map[uint64]*bridge)

//export NewBridge
func NewBridge(id uint64) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println("failed to init zap logger", id)
		logger = zap.NewNop()
	}
	logger.Debug("NewBridge", zap.Uint64("id", id))

	if bridges[id] != nil {
		logger.Error("bridge already allocated", zap.Uint64("id", id))
		panic("bridge already allocated")
	}

	dialer := memconn.Dialer("/dev/shm/ivshmem", 1000000, 0, logger)

	ctx, cancelCtx := context.WithCancel(context.Background())
	conn, err := grpc.DialContext(ctx, "memconn", grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		logger.Error("failed to dial", zap.Error(err))
		panic(err)
	}

	bridges[id] = &bridge{
		c:         vstbridge.NewVSTBridgeClient(conn),
		l:         logger,
		ctx:       ctx,
		cancelCtx: cancelCtx,
		conn:      conn,
	}
}

//export CloseBridge
func CloseBridge(id uint64) {
	b, ok := bridges[id]
	if !ok {
		fmt.Println("warning: tried to close unallocated bridge", id)
		return
	}
	b.l.Debug("CloseBridge", zap.Uint64("id", id))

	err := b.conn.Close()
	if err != nil {
		b.l.Error("failed to close conn", zap.Error(err))
	}

	b.cancelCtx()

	bridges[id] = nil
}

//export GetParameter
func GetParameter(id uint64, index int32) float32 {
	b, ok := bridges[id]
	if !ok {
		return 0.5
	}

	b.l.Debug("GetParameter", zap.Uint64("id", id), zap.Int32("index", index))

	rep, err := b.c.GetParameter(b.ctx, &vstbridge.GetParameter_Request{
		Id:    id,
		Index: index,
	})
	if err != nil {
		b.l.Error("GetParameter", zap.Error(err))
		return 0.5
	}
	return rep.GetValue()
}

//export SetParameter
func SetParameter(id uint64, index int32, value float32) {
	b, ok := bridges[id]
	if !ok {
		return
	}

	b.l.Debug("SetSampleRate", zap.Uint64("id", id), zap.Int32("index", index), zap.Float32("value", value))

	_, err := b.c.SetParameter(b.ctx, &vstbridge.SetParameter_Request{
		Id:    id,
		Index: index,
		Value: value,
	})
	if err != nil {
		b.l.Error("SetParameter", zap.Error(err))
	}
}

//export ProcessReplacing
func ProcessReplacing(id uint64, inputs **float32, outputs **float32, sampleFrames int32) {
	b, ok := bridges[id]
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
		Id: id,
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
func ProcessDoubleReplacing(id uint64, inputs **float64, outputs **float64, sampleFrames int32) {
	b, ok := bridges[id]
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
		Id: id,
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

//export SetSampleRate
func SetSampleRate(id uint64, sampleRate float32) bool {
	b, ok := bridges[id]
	if !ok {
		return false
	}

	b.l.Debug("SetSampleRate", zap.Uint64("id", id), zap.Float32("sampleRate", sampleRate))

	_, err := b.c.SetSampleRate(b.ctx, &vstbridge.SetSampleRate_Request{
		Id:         id,
		SampleRate: sampleRate,
	})
	if err != nil {
		b.l.Error("SetSampleRate", zap.Error(err))
		return false
	}

	return true
}

func main() {}
