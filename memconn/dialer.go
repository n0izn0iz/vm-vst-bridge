package memconn

/*
#include "membuf.h"
*/
import "C"
import (
	"context"
	"fmt"
	"net"
	"time"
)

func Dialer(shmemPath string, ringSize int, offset int) func(ctx context.Context, address string) (net.Conn, error) {
	var ret C.int
	var size C.size_t
	mem := C.initShmem(shmemPath, &size, &ret)
	if ret != 0 {
		panic(fmt.Sprint("failed to init shmem: ret=", ret))
	}
	fmt.Println("ivshmem size:", size/1024/1024, "MiB")
	fmt.Println("ring offset:", offset, "B")

	// FIXME: compute real maximum
	if C.size_t(ringSize) > (size / 2) {
		panic("ring too big for ivshmem")
	}
	size = C.size_t(ringSize)
	fmt.Println("ring size:", size, "B")

	return func(ctx context.Context, address string) (net.Conn, error) {
		fmt.Println("Dial called")

		fmt.Println("Wait for connected")
		for !mem.connected {
			mem.answer = mem.challenge + 42
			time.Sleep(10 * time.Millisecond)
		}
		fmt.Println("Got connected")

		return Connect(mem, ringSize, offset, true), nil
	}
}
