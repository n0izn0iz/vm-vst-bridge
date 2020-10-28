package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/n0izn0iz/vm-vst-bridge-host/memconn"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	fmt.Println("info", info)
	return nil, nil
}

func main() {
	rand.Seed(time.Now().UnixNano())

	var offset int
	flag.IntVar(&offset, "offset", 0, "offset of the ring buffer in the ivshmem")
	var ringSize int
	flag.IntVar(&ringSize, "size", 16, "size of the ring buffer")
	var shmemPath string
	flag.StringVar(&shmemPath, "shmem-path", "/dev/shm/ivshmem", "path to the shared memory file")
	var client bool
	flag.BoolVar(&client, "client", false, "client mode")

	flag.Parse()

	lis := memconn.Listen(shmemPath, ringSize, offset)
	opts := []grpc_zap.Option{
		grpc_zap.WithLevels(grpc_zap.DefaultCodeToLevel),
	}
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	logger.Info("wtf")
	s := grpc.NewServer(grpc_middleware.WithUnaryServerChain(
		//unaryInterceptor,
		grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
		grpc_zap.UnaryServerInterceptor(logger, opts...),
	), grpc_middleware.WithStreamServerChain(
		grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
		grpc_zap.StreamServerInterceptor(logger, opts...),
	))
	RegisterVSTBridgeServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		panic(err)
	}

	fmt.Println("server done")
}
