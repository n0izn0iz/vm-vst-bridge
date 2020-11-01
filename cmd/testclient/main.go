package main

import (
	"bufio"
	"context"
	"flag"
	"math/rand"
	"os"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

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

	flag.Parse()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync() // flushes buffer, if any

	dialer := memconn.Dialer(shmemPath, ringSize, offset, logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn, err := grpc.DialContext(ctx, "memconn", grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	client := vstbridge.NewVSTBridgeClient(conn)

	reader := bufio.NewReader(os.Stdin)
	for {
		text, _ := reader.ReadString('\n')
		resp, err := client.Echo(ctx, &vstbridge.Echo_Request{Str: text})
		logger.Debug("response", zap.String("str", resp.GetStr()), zap.Error(err))
	}
}
