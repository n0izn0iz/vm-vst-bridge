package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/n0izn0iz/vm-vst-bridge-host/memconn"
)

func TestEcho(t *testing.T) {
	const ringSize = 16

	rand.Seed(time.Now().UnixNano())

	lis := memconn.Listen("/dev/shm/ivshmem", ringSize, 0)
	s := grpc.NewServer()
	RegisterVSTBridgeServer(s, &server{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()

	dialer := memconn.Dialer("/dev/shm/ivshmem", ringSize, 0)
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "memconn", grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		require.NoError(t, err)
	}
	//defer conn.Close()

	client := NewVSTBridgeClient(conn)
	fmt.Println("will call echo")
	resp, err := client.Echo(ctx, &Echo_Request{Str: "test"})
	if err != nil {
		require.NoError(t, err)
	}

	if resp.GetStr() != "test" {
		t.Fatal("echo reply must be 'test'")
	}

	log.Println("TEST DONE")

	wg.Wait()
}
