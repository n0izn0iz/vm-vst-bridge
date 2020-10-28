package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/n0izn0iz/vm-vst-bridge-host/memconn"
)

func TestEcho(t *testing.T) {
	const ringSize = 512

	rand.Seed(time.Now().UnixNano())

	dialer := memconn.Dialer("/dev/shm/ivshmem", ringSize, 0)
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "memconn", grpc.WithContextDialer(dialer), grpc.WithInsecure())
	require.NoError(t, err)

	client := NewVSTBridgeClient(conn)
	fmt.Println("will call echo")
	resp, err := client.Echo(ctx, &Echo_Request{Str: "test"})
	fmt.Println("called echo")
	if err != nil {
		require.NoError(t, err)
	}

	if resp.GetStr() != "test" {
		t.Fatal("echo reply must be 'test'")
	}

	ret := conn.Close()
	fmt.Println("close ret", ret)
	require.NoError(t, ret)

	time.Sleep(100 * time.Millisecond) // really wait for close

	log.Println("TEST DONE")
}
