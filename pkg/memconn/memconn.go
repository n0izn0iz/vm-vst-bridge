package memconn

// #include "membuf.h"
import "C"
import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

type uint64_t = C.uint64_t

type Conn struct {
	size         C.size_t
	membuf       *C.membuf
	readerOffset C.size_t
	writerOffset C.size_t
	readerIndex  int
	writerIndex  int
	logger       *zap.Logger
	closed       bool
	closeLock    *sync.Mutex
	muLock       *sync.Mutex
}

var _ net.Conn = (*Conn)(nil)

func Connect(membuf *C.membuf, ringSize int, offset int, invert bool, logger *zap.Logger) *Conn {
	var readerOffset C.size_t
	var writerOffset C.size_t
	var readerIndex int
	var writerIndex int
	if !invert {
		readerOffset = C.size_t(offset)
		writerOffset = C.size_t(offset + (ringSize / 2))
		readerIndex = 0
		writerIndex = 1
	} else {
		readerOffset = C.size_t(offset + (ringSize / 2))
		writerOffset = C.size_t(offset)
		readerIndex = 1
		writerIndex = 0
	}

	if os.Getenv("DEBUG_MEMCONN") == "true" {
		logger = logger.Named("conn")
	} else {
		logger = zap.NewNop()
	}

	return &Conn{
		size:         C.size_t(ringSize / 2),
		membuf:       membuf,
		readerOffset: readerOffset,
		writerOffset: writerOffset,
		readerIndex:  readerIndex,
		writerIndex:  writerIndex,
		logger:       logger,
		closed:       false,
		closeLock:    &sync.Mutex{},
	}
}

func (c *Conn) Close() error {
	c.logger.Debug("Conn.Close", zap.Stack("trace"))
	c.closeLock.Lock()
	defer c.closeLock.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	c.membuf.connected[c.writerIndex] = false
	return nil
}

func (c *Conn) Read(buf []byte) (int, error) {
	index := c.membuf.readIndices[c.readerIndex]
	c.logger.Debug("Conn.Read", zap.Int("len", len(buf)), zap.Uint64("readIndex", uint64(index)), zap.Stack("trace"))

	rbuf := C.malloc(c.size)
	defer C.free(rbuf)

	var toRead C.uint64_t
	if len(buf) <= 0 {
		return 0, nil
	}
	for {
		if c.closed {
			return 0, io.ErrClosedPipe
		}
		toRead = readLen(c.membuf.writeIndices[c.readerIndex], index, c.size)
		if toRead > 0 {
			break
		}
		if !c.membuf.connected[c.readerIndex] {
			return 0, io.EOF
		}
		time.Sleep(memTiming)
	}

	if toRead > C.uint64_t(len(buf)) {
		toRead = C.uint64_t(len(buf))
	}

	newIndex := (index + toRead) % c.size
	rs := c.readerOffset + index
	if newIndex > index {
		readBytes(c.membuf, rbuf, 0, rs, toRead)
	} else {
		fLen := c.size - index
		readBytes(c.membuf, rbuf, 0, rs, fLen)
		readBytes(c.membuf, rbuf, fLen, c.readerOffset, newIndex)
	}
	c.membuf.readIndices[c.readerIndex] = newIndex
	rbytes := C.GoBytes(rbuf, C.int(toRead))
	copy(buf, rbytes)
	c.logger.Debug("did read", zap.String("data", string(buf[:toRead])), zap.Uint64("readStart", uint64(rs)), zap.Uint64("len", uint64(toRead)), zap.Uint64("newIndex", uint64(newIndex)))

	return int(toRead), nil
}

func (c *Conn) Write(data []byte) (int, error) {
	index := c.membuf.writeIndices[c.writerIndex]
	c.logger.Debug("Conn.Write", zap.String("data", string(data)), zap.Int("len", len(data)), zap.Uint64("writeIndex", uint64(index)), zap.Stack("trace"))

	for i := 0; i < len(data); {
		currentIndex := (index + C.uint64_t(i)) % c.size
		var canWrite C.uint64_t
		for {
			if c.closed {
				return int(i), io.ErrClosedPipe
			}
			if !c.membuf.connected[c.writerIndex] {
				return int(i), io.ErrClosedPipe
			}
			canWrite = writeLen(currentIndex, c.membuf.readIndices[c.writerIndex], c.size)
			if canWrite > 0 {
				break
			}
			time.Sleep(memTiming)
		}
		ocw := canWrite
		if canWrite > C.uint64_t(len(data)-i) {
			canWrite = C.uint64_t(len(data) - i)
		}
		newIndex := (currentIndex + canWrite) % c.size
		ws := c.writerOffset + currentIndex
		c.logger.Debug("will write", zap.String("data", string(data[i:])), zap.Uint64("writeStart", uint64(ws)), zap.Uint64("couldWrite", uint64(ocw)), zap.Uint64("canWrite", uint64(canWrite)), zap.Uint64("newIndex", uint64(newIndex)))
		cbuf := C.CBytes(data[i:])
		if newIndex > currentIndex {
			writeBytes(c.membuf, cbuf, 0, ws, canWrite)
		} else {
			fLen := c.size - currentIndex
			writeBytes(c.membuf, cbuf, 0, ws, fLen)
			writeBytes(c.membuf, cbuf, fLen, c.writerOffset, newIndex)
		}
		c.membuf.writeIndices[c.writerIndex] = newIndex
		C.free(cbuf)
		i += int(canWrite)
	}

	return len(data), nil
}

func (c *Conn) SetDeadline(t time.Time) error {
	err := c.SetReadDeadline(t)
	if err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}

func (c *Conn) SetReadDeadline(time.Time) error {
	return nil //FIXME
}

func (c *Conn) SetWriteDeadline(time.Time) error {
	return nil //FIXME
}

func (l *Conn) LocalAddr() net.Addr  { return addr{} }
func (l *Conn) RemoteAddr() net.Addr { return addr{} }

func readLen(writeIndex C.uint64_t, readIndex C.uint64_t, size C.size_t) C.uint64_t {
	if writeIndex >= size || readIndex >= size || size < 2 {
		panic(fmt.Sprint("invalid arguments, wi:", writeIndex, ", ri:", readIndex, ", size:", size))
	}
	if readIndex == writeIndex {
		return 0
	} else if readIndex < writeIndex {
		return writeIndex - readIndex
	} else {
		return writeIndex + (size - readIndex)
	}
}

func writeLen(writeIndex C.uint64_t, readIndex C.uint64_t, size C.size_t) C.uint64_t {
	if writeIndex >= size || readIndex >= size || size < 2 {
		panic(fmt.Sprint("invalid arguments, wi:", writeIndex, ", ri:", readIndex, ", size:", size))
	}
	if readIndex == writeIndex {
		return size - 1
	} else if readIndex < writeIndex {
		return (size - writeIndex) + readIndex - 1
	} else {
		return readIndex - writeIndex - 1
	}
}
