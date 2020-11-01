package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"pipelined.dev/audio/vst2"

	"github.com/n0izn0iz/vm-vst-bridge/pkg/memconn"
	"github.com/n0izn0iz/vm-vst-bridge/pkg/vstbridge"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	var offset int
	flag.IntVar(&offset, "offset", 0, "offset of the ring buffer in the ivshmem")
	var ringSize int
	flag.IntVar(&ringSize, "size", 16, "size of the ring buffer")
	var shmemPath string
	flag.StringVar(&shmemPath, "shmem-path", "/dev/shm/ivshmem", "path to the shared memory file")
	var pluginPath string
	flag.StringVar(&pluginPath, "plugin-path", "C:\\VST\\64\\Synth1 VST64.dll", "path to the plugin")

	flag.Parse()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync() // flushes buffer, if any

	vst, err := vst2.Open(pluginPath)
	if err != nil {
		panic(err)
	}
	defer vst.Close()

	fmt.Println("Will load")

	plugin := vst.Load(func(code vst2.HostOpcode, _ vst2.Index, _ vst2.Value, _ vst2.Ptr, _ vst2.Opt) vst2.Return {
		fmt.Printf("Received opcode: %v\n", code)
		return 0
	})
	defer plugin.Close()

	fmt.Println("Loaded")

	lis := memconn.Listen(shmemPath, ringSize, offset, logger)
	/*opts := []grpc_zap.Option{
		grpc_zap.WithLevels(grpc_zap.DefaultCodeToLevel),
	}*/
	s := grpc.NewServer( /*grpc_middleware.WithUnaryServerChain(
		grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
		grpc_zap.UnaryServerInterceptor(logger, opts...),
	), grpc_middleware.WithStreamServerChain(
		grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
		grpc_zap.StreamServerInterceptor(logger, opts...),
	)*/)
	sImpl := newServer(logger, plugin)
	vstbridge.RegisterVSTBridgeServer(s, sImpl)
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := s.Serve(lis); err != nil {
			panic(err)
		}
	}()

	uiMain(plugin)

	<-done
	fmt.Println("done")
}
