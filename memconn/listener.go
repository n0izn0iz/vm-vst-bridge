package memconn

/*
#include "membuf.h"
*/
import "C"
import (
	"fmt"
	"math/rand"
	"net"
	"time"
)

type Listener struct {
	membuf   *C.membuf
	ringSize int
	offset   int
}

var _ net.Listener = (*Listener)(nil)

func Listen(shmemPath string, ringSize int, offset int) *Listener {
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

	mem.connected = false

	return &Listener{
		membuf:   mem,
		ringSize: ringSize,
		offset:   offset,
	}
}

var errClosed = fmt.Errorf("closed")

func (l *Listener) Accept() (net.Conn, error) {
	fmt.Println("Accept called")

	for l.membuf.connected == true {
		time.Sleep(10 * time.Millisecond)
	}

	challenge := C.uint64_t(rand.Uint32())
	fmt.Println("challenge: ", challenge)
	answer := challenge + 42

	mem := l.membuf
	mem.readIndices[0] = mem.writeIndices[0]
	mem.readIndices[1] = mem.writeIndices[1]
	mem.challenge = challenge
	fmt.Println("Wait for challenge answer")
	for mem.answer != answer {
		time.Sleep(10 * time.Millisecond)
	}
	fmt.Println("Got answer")
	mem.connected = true

	return Connect(mem, l.ringSize, l.offset, false), nil
}

func (l *Listener) Close() error {
	return nil
}

func (l *Listener) Addr() net.Addr { return addr{} }
