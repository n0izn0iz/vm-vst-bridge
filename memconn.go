package main

/*
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <unistd.h>
#include <fcntl.h>
#include <string.h>

#include <sys/stat.h>
#include <sys/mman.h>

#include "membuf.h"

void writeByte(membuf* mb, size_t offset, char val) {
	(&mb->data)[offset] = val;
}

char readByte(membuf* mb, size_t offset) {
	return (&mb->data)[offset];
}

membuf* initShmem(_GoString_ _shmem_device_file, size_t *size, int* err)
{
	*err = 0;

	size_t len = _GoStringLen(_shmem_device_file);
	char shmem_device_file[len + 1];
	memcpy(shmem_device_file, _GoStringPtr(_shmem_device_file), len);
	shmem_device_file[len] = '\0';

  struct stat st;
  if (stat(shmem_device_file, &st) < 0)  {
    fprintf(stderr, "Failed to stat the shared memory file: %s\n", shmem_device_file);
		*err = 1;
    return NULL;
  }
	*size = st.st_size;

  int shmFD = open(shmem_device_file, O_RDWR);
  if (shmFD < 0) {
    fprintf(stderr, "Failed to open the shared memory file: %s\n", shmem_device_file);
		*err = 2;
    return NULL;
  }

  membuf* mb = mmap(0, st.st_size, PROT_READ | PROT_WRITE, MAP_SHARED, shmFD, 0);
  if (mb == MAP_FAILED) {
    fprintf(stderr, "Failed to map the shared memory file: %s\n", shmem_device_file);
    close(shmFD);
		*err = 3;
    return NULL;
  }

	return mb;
}
*/
import "C"
import (
	"errors"
	"fmt"
	"net"
	"time"
)

// fixme: add ids to prevent misuse

type Listener struct {
	shmemPath string
	ringSize  int
	offset    int
	invert    bool
	accepted  bool
}

var _ net.Listener = (*Listener)(nil)

func Listen(shmemPath string, ringSize int, offset int, invert bool) *Listener {
	return &Listener{shmemPath, ringSize, offset, invert, false}
}

func (l *Listener) Accept() (net.Conn, error) {
	if l.accepted {
		return nil, errors.New("memconn already accepted")
	}
	l.accepted = true
	return Connect(l.shmemPath, l.ringSize, l.offset, l.invert), nil
}

func (l *Listener) Close() error {
	return nil
}

func (l *Listener) Addr() net.Addr { return addr{} }

type Conn struct {
	size         C.size_t
	membuf       *C.membuf
	readerOffset C.size_t
	writerOffset C.size_t
	readerIndex  int
	writerIndex  int
}

var _ net.Conn = (*Conn)(nil)

func Connect(shmemPath string, ringSize int, offset int, invert bool) *Conn {
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

	return &Conn{
		size:         size,
		membuf:       mem,
		readerOffset: readerOffset,
		writerOffset: writerOffset,
		readerIndex:  readerIndex,
		writerIndex:  writerIndex,
	}
}

func (c *Conn) Close() error {
	return nil // FIXME
}

func (c *Conn) Read(buf []byte) (int, error) {
	c.membuf.readIndices[c.readerIndex] = c.membuf.writeIndices[c.readerIndex]
	//fmt.Println("readIndex: ", c.membuf.readIndices[c.readerIndex])

	//fmt.Println("_______________________")
	var readUntil C.uint64_t
	for {
		readUntil = c.membuf.writeIndices[c.readerIndex]
		if c.membuf.readIndices[c.readerIndex] != readUntil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	var toRead C.uint64_t
	if c.membuf.readIndices[c.readerIndex] < readUntil {
		toRead = readUntil - c.membuf.readIndices[c.readerIndex]
	} else {
		toRead = readUntil + (c.size - c.membuf.readIndices[c.readerIndex])
	}
	if toRead > C.uint64_t(len(buf)) {
		toRead = C.uint64_t(len(buf))
	}
	//fmt.Println("toRead: ", toRead)
	for i := C.uint64_t(0); i < toRead; i++ {
		b := C.readByte(c.membuf, c.readerOffset+((c.membuf.readIndices[c.readerIndex]+i)%c.size))
		buf[i] = byte(b)
	}
	//fmt.Println("__")
	//fmt.Println("Text: ", string(buf))
	c.membuf.readIndices[c.readerIndex] = readUntil
	//fmt.Println("readIndex: ", c.membuf.readIndices[c.readerIndex])

	return int(toRead), nil
}

func (c *Conn) Write(data []byte) (int, error) {
	//fmt.Println("_______________________")

	toWrite := C.uint64_t(len(data))
	//fmt.Println("toWrite: ", toWrite)
	baseIndex := c.membuf.writeIndices[c.writerIndex]
	for i := C.uint64_t(0); i < toWrite; i++ {
		for {
			writeUntil := c.membuf.readIndices[c.writerIndex]
			var canWrite C.uint64_t
			if writeUntil == c.membuf.writeIndices[c.writerIndex] {
				canWrite = c.size - 1
			} else if writeUntil < c.membuf.writeIndices[c.writerIndex] {
				canWrite = (c.size - c.membuf.writeIndices[c.writerIndex]) + writeUntil - 1
			} else {
				canWrite = writeUntil - c.membuf.writeIndices[c.writerIndex] - 1
			}
			if canWrite > 0 {
				break
			}
			//fmt.Println("Waiting for read..")
			time.Sleep(10 * time.Millisecond)
		}
		C.writeByte(c.membuf, c.writerOffset+((baseIndex+i)%c.size), C.char(data[i]))
		c.membuf.writeIndices[c.writerIndex] = (c.membuf.writeIndices[c.writerIndex] + 1) % c.size
	}
	//fmt.Println("writeIndex: ", c.membuf.writeIndices[c.writerIndex])

	return len(data), nil
}

func (c *Conn) SetDeadline(time.Time) error {
	return nil //FIXME
}

func (c *Conn) SetReadDeadline(time.Time) error {
	return nil //FIXME
}

func (c *Conn) SetWriteDeadline(time.Time) error {
	return nil //FIXME
}

func (l *Conn) LocalAddr() net.Addr  { return addr{} }
func (l *Conn) RemoteAddr() net.Addr { return addr{} }

type addr struct{}

var _ net.Addr = (*addr)(nil)

func (a addr) Network() string { return "memconn" }
func (a addr) String() string  { return "memconn" }
