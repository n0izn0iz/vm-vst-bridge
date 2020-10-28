package memconn

/*
#include "membuf.h"
*/
import "C"
import (
	"context"
	"errors"
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

func newMembuf(shmemPath string, ringSize int, offset int) (*C.membuf, C.size_t) {
	if len(shmemPath) <= 0 || offset < 0 || ringSize < 4 || ringSize%2 != 0 {
		panic("invalid parameter(s)")
	}

	var ret C.int
	var memSize C.size_t
	mem := C.initShmem(shmemPath, &memSize, &ret)
	if ret != 0 {
		panic(fmt.Sprint("failed to init shmem: ret=", ret))
	}
	fmt.Println("ivshmem size:", memSize/1024/1024, "MiB")
	fmt.Println("ring offset:", offset, "B")
	fmt.Println("ring size:", ringSize, "B")

	// FIXME: compute real maximum
	if C.size_t(ringSize) > (memSize/2)-C.size_t(offset) {
		panic("ring too big for ivshmem")
	}

	return mem, memSize
}

func Listen(shmemPath string, ringSize int, offset int) *Listener {
	mem, _ := newMembuf(shmemPath, ringSize, offset)
	mem.connected[0] = false
	mem.connected[1] = false
	mem.writeIndices[0] = 0
	mem.writeIndices[1] = 0
	return &Listener{
		membuf:   mem,
		ringSize: ringSize,
		offset:   offset,
	}
}

var errClosed = fmt.Errorf("closed")

func (l *Listener) Accept() (net.Conn, error) {
	fmt.Println("Accept called")

	if l.membuf.connected[0] || l.membuf.connected[1] {
		time.Sleep(10 * time.Millisecond)
	}

	challenge := C.uint64_t(rand.Uint32())
	//fmt.Println("challenge: ", challenge)
	answer := challenge + 42

	mem := l.membuf
	mem.challenge = challenge
	//fmt.Println("Wait for challenge answer")
	for mem.answer != answer {
		time.Sleep(10 * time.Millisecond)
	}
	fmt.Println("Got answer")
	mem.readIndices[0] = mem.writeIndices[0]
	mem.readIndices[1] = mem.writeIndices[1]
	mem.connected[0] = true
	mem.connected[1] = true

	return Connect(mem, l.ringSize, l.offset, false), nil
}

func Dialer(shmemPath string, ringSize int, offset int) func(ctx context.Context, address string) (net.Conn, error) {
	mem, _ := newMembuf(shmemPath, ringSize, offset)
	return func(ctx context.Context, address string) (net.Conn, error) {
		fmt.Println("Dial called")

		if mem.connected[0] || mem.connected[1] {
			return nil, errors.New("already dialed")
		}

		fmt.Println("Wait for connected")
		for !(mem.connected[0] && mem.connected[1]) {
			mem.answer = mem.challenge + 42
			time.Sleep(10 * time.Millisecond)
		}
		fmt.Println("Got connected")

		return Connect(mem, ringSize, offset, true), nil
	}
}

func (l *Listener) Close() error {
	l.membuf.connected[0] = false
	l.membuf.connected[1] = false
	return nil
}

func (l *Listener) Addr() net.Addr { return addr{} }
