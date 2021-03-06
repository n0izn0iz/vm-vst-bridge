package memconn

/*
#include "membuf.h"
*/
import "C"
import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Listener struct {
	membuf    *C.membuf
	ringSize  int
	offset    int
	closed    bool
	closeLock *sync.Mutex
	logger    *zap.Logger
}

var _ net.Listener = (*Listener)(nil)

func Listen(shmemPath string, ringSize int, offset int, logger *zap.Logger) *Listener {
	mem, _ := newMembuf(shmemPath, ringSize, offset, logger)
	mem.connected[0] = false
	mem.connected[1] = false
	mem.writeIndices[0] = 0
	mem.writeIndices[1] = 0
	return &Listener{
		membuf:    mem,
		ringSize:  ringSize,
		offset:    offset,
		logger:    logger,
		closeLock: &sync.Mutex{},
	}
}

var ErrClosed = fmt.Errorf("closed")

func (l *Listener) Accept() (net.Conn, error) {
	l.logger.Debug("Listener.Accept", zap.Stack("trace"))

	if l.closed {
		return nil, ErrClosed
	}

	if l.membuf.connected[0] || l.membuf.connected[1] {
		l.logger.Debug("handshake: wait for end of previous conn")
		for l.membuf.connected[0] || l.membuf.connected[1] {
			time.Sleep(memTiming)
			if l.closed {
				return nil, ErrClosed
			}
		}
	}

	challenge := C.uint64_t(rand.Uint32())
	answer := challenge + 42
	l.logger.Debug("handshake: posting challenge", zap.Uint("challenge", uint(challenge)))
	mem := l.membuf
	mem.challenge = challenge
	for mem.answer != answer {
		time.Sleep(memTiming)
		if l.closed {
			return nil, ErrClosed
		}
	}
	l.logger.Debug("handshake: got challenge answer", zap.Uint("challenge", uint(challenge)))
	mem.readIndices[0] = mem.writeIndices[0]
	mem.readIndices[1] = mem.writeIndices[1]
	mem.connected[0] = true
	mem.connected[1] = true

	return Connect(mem, l.ringSize, l.offset, false, l.logger), nil
}

func Dialer(shmemPath string, ringSize int, offset int, logger *zap.Logger) func(ctx context.Context, address string) (net.Conn, error) {
	mem, _ := newMembuf(shmemPath, ringSize, offset, logger)
	Dial := func(ctx context.Context, address string) (net.Conn, error) {
		logger.Debug("Dialer.Dial", zap.Stack("trace"))
		if mem.connected[0] || mem.connected[1] {
			logger.Debug("handshake: wait for end of previous conn")
			for mem.connected[0] || mem.connected[1] {
				time.Sleep(memTiming)
				select {
				case <-ctx.Done():
					return nil, ErrClosed
				default:
				}
			}
		}

		logger.Debug("handshake: answer challenge and wait for connected")
		var challenge, answer C.uint64_t
		for !(mem.connected[0] && mem.connected[1]) {
			challenge = mem.challenge
			answer = challenge + 42
			mem.answer = answer
			time.Sleep(memTiming)
			select {
			case <-ctx.Done():
				return nil, ErrClosed
			default:
			}
		}
		logger.Debug("handshake: done", zap.Uint("challenge", uint(challenge)))

		return Connect(mem, ringSize, offset, true, logger), nil
	}
	return Dial
}

func (l *Listener) Close() error {
	l.logger.Debug("Listener.Close", zap.Stack("trace"))
	l.closeLock.Lock()
	defer l.closeLock.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true
	return nil
}

func (l *Listener) Addr() net.Addr { return addr{} }
