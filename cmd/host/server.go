package main

import (
	"context"

	"go.uber.org/zap"
	"pipelined.dev/audio/vst2"
	"pipelined.dev/signal"

	"github.com/n0izn0iz/vm-vst-bridge/pkg/vstbridge"
)

type server struct {
	vstbridge.UnimplementedVSTBridgeServer
	logger *zap.Logger
	p      *vst2.Plugin
}

var _ vstbridge.VSTBridgeServer = (*server)(nil)

func newServer(logger *zap.Logger, plugin *vst2.Plugin) vstbridge.VSTBridgeServer {
	return &server{
		logger: logger.Named("VSTBridgeServer"),
		p:      plugin,
	}
}

func (s *server) Echo(ctx context.Context, req *vstbridge.Echo_Request) (*vstbridge.Echo_Reply, error) {
	s.logger.Debug("VSTBridge.Echo", zap.String("str", req.GetStr()))
	return &vstbridge.Echo_Reply{Str: req.GetStr()}, nil
}

func floatBufferToFloatArray(buf vst2.FloatBuffer, channels int, sf int) []*vstbridge.FloatArray {
	sig := signal.Allocator{Channels: channels, Length: sf, Capacity: sf}.Float32()
	buf.CopyTo(sig)

	arr := make([]*vstbridge.FloatArray, channels)

	for ci := 0; ci < channels; ci++ {
		arr[ci] = &vstbridge.FloatArray{Data: make([]float32, sf)}

		for si := 0; si < sf; si++ {
			arr[ci].Data[si] = float32(sig.Sample(sig.BufferIndex(ci, si)))
		}
	}

	return arr
}

func floatArrayToFloatBuffer(arr []*vstbridge.FloatArray, buf vst2.FloatBuffer) {
	channels := len(arr)
	if channels <= 0 {
		return
	}

	cap := len(arr[0].GetData())

	sig := signal.Allocator{Channels: channels, Length: cap, Capacity: cap}.Float32()

	for ci := 0; ci < channels; ci++ {
		data := arr[ci].GetData()
		for si := 0; si < cap; si++ {
			sig.SetSample(sig.BufferIndex(ci, si), float64(data[si]))
		}
	}

	buf.CopyFrom(sig)
}

func (s *server) ProcessReplacing(ctx context.Context, req *vstbridge.ProcessReplacing_Request) (*vstbridge.ProcessReplacing_Reply, error) {
	channels := len(req.GetInputs())

	sf := int(req.GetSampleFrames())

	in := vst2.NewFloatBuffer(channels, sf)
	floatArrayToFloatBuffer(req.GetInputs(), in)

	out := vst2.NewFloatBuffer(channels, sf)

	s.p.ProcessFloat(in, out)

	return &vstbridge.ProcessReplacing_Reply{Outputs: floatBufferToFloatArray(out, channels, sf)}, nil
}

func doubleBufferToDoubleArray(buf vst2.DoubleBuffer, channels int, cap int) []*vstbridge.DoubleArray {
	sig := signal.Allocator{Channels: channels, Length: cap, Capacity: cap}.Float64()
	buf.CopyTo(sig)

	arr := make([]*vstbridge.DoubleArray, channels)

	for ci := 0; ci < channels; ci++ {
		arr[ci] = &vstbridge.DoubleArray{Data: make([]float64, cap)}

		for si := 0; si < cap; si++ {
			arr[ci].Data[si] = sig.Sample(sig.BufferIndex(ci, si))
		}
	}

	return arr
}

func doubleArrayToDoubleBuffer(arr []*vstbridge.DoubleArray, buf vst2.DoubleBuffer) {
	channels := len(arr)
	if channels <= 0 {
		return
	}

	cap := len(arr[0].GetData())

	sig := signal.Allocator{Channels: channels, Length: cap, Capacity: cap}.Float64()

	for ci := 0; ci < channels; ci++ {
		data := arr[ci].GetData()
		for si := 0; si < cap; si++ {
			sig.SetSample(sig.BufferIndex(ci, si), data[si])
		}
	}

	buf.CopyFrom(sig)
}

func (s *server) ProcessDoubleReplacing(ctx context.Context, req *vstbridge.ProcessDoubleReplacing_Request) (*vstbridge.ProcessDoubleReplacing_Reply, error) {
	channels := len(req.GetInputs())

	sf := int(req.GetSampleFrames())

	in := vst2.NewDoubleBuffer(channels, sf)
	doubleArrayToDoubleBuffer(req.GetInputs(), in)

	out := vst2.NewDoubleBuffer(channels, sf)

	s.p.ProcessDouble(in, out)

	return &vstbridge.ProcessDoubleReplacing_Reply{Outputs: doubleBufferToDoubleArray(out, channels, sf)}, nil
}

func (s *server) GetParameter(ctx context.Context, req *vstbridge.GetParameter_Request) (*vstbridge.GetParameter_Reply, error) {
	val := s.p.ParamValue(int(req.GetIndex()))
	return &vstbridge.GetParameter_Reply{Value: val}, nil
}

func (s *server) SetParameter(ctx context.Context, req *vstbridge.SetParameter_Request) (*vstbridge.SetParameter_Reply, error) {
	s.p.SetParamValue(int(req.GetIndex()), req.GetValue())
	return &vstbridge.SetParameter_Reply{}, nil
}

func (s *server) SetSampleRate(ctx context.Context, req *vstbridge.SetSampleRate_Request) (*vstbridge.SetSampleRate_Reply, error) {
	s.p.SetSampleRate(int(req.GetSampleRate()))
	return &vstbridge.SetSampleRate_Reply{}, nil
}
