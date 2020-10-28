package memconn

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
	"fmt"
	"io"
	"net"
	//"runtime/debug"
	"time"
)

// fixme: add ids to prevent misuse

type uint64_t = C.uint64_t

type Conn struct {
	size         C.size_t
	membuf       *C.membuf
	readerOffset C.size_t
	writerOffset C.size_t
	readerIndex  int
	writerIndex  int
}

var _ net.Conn = (*Conn)(nil)

func Connect(membuf *C.membuf, ringSize int, offset int, invert bool) *Conn {
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
		size:         C.size_t(ringSize / 2),
		membuf:       membuf,
		readerOffset: readerOffset,
		writerOffset: writerOffset,
		readerIndex:  readerIndex,
		writerIndex:  writerIndex,
	}
}

func (c *Conn) Close() error {
	//fmt.Println("Close called", c.readerIndex)
	//debug.PrintStack()
	//c.membuf.connected[c.readerIndex] = false
	c.membuf.connected[c.writerIndex] = false
	//c.closed = true
	return nil // FIXME
}

func (c *Conn) Read(buf []byte) (int, error) {
	//fmt.Println("R______________________")

	//fmt.Println("readIndex: ", c.membuf.readIndices[c.readerIndex])
	var index C.uint64_t
	var toRead C.uint64_t
	if len(buf) <= 0 {
		return 0, nil
	}
	for {
		if !c.membuf.connected[c.writerIndex] && !c.membuf.connected[c.readerIndex] {
			return 0, io.ErrClosedPipe
		}
		if !c.membuf.connected[c.readerIndex] {
			//fmt.Println("not connected #", c.readerIndex)
			return 0, io.EOF
		}
		index = c.membuf.readIndices[c.readerIndex]
		toRead = readLen(c.membuf.writeIndices[c.readerIndex], index, c.size)
		if toRead > 0 {
			break
		}
		//fmt.Println("waiting for write")
		time.Sleep(10 * time.Millisecond)
	}

	if toRead > C.uint64_t(len(buf)) {
		toRead = C.uint64_t(len(buf))
	}
	//fmt.Println("toRead: ", toRead)
	for i := C.uint64_t(0); i < toRead; i++ {
		b := C.readByte(c.membuf, c.readerOffset+((index+i)%c.size))
		buf[i] = byte(b)
	}
	//fmt.Println("Data: ", string(buf))
	//fmt.Print(string(buf))
	c.membuf.readIndices[c.readerIndex] = (index + toRead) % c.size
	//fmt.Println("readIndex: ", c.membuf.readIndices[c.readerIndex])

	return int(toRead), nil
}

func (c *Conn) Write(data []byte) (int, error) {
	//fmt.Println("W______________________")
	//fmt.Println(">>", string(data))

	if !c.membuf.connected[c.writerIndex] {
		//fmt.Println("not connected 2")
		return 0, io.ErrClosedPipe
	}

	toWrite := C.uint64_t(len(data))
	//fmt.Println("toWrite: ", toWrite)
	for i := C.uint64_t(0); i < toWrite; i++ {
		var index C.uint64_t
		for {
			if !c.membuf.connected[c.writerIndex] {
				//fmt.Println("not connected 3")
				return 0, io.ErrClosedPipe
			}
			index = c.membuf.writeIndices[c.writerIndex]
			canWrite := writeLen(index, c.membuf.readIndices[c.writerIndex], c.size)
			if canWrite > 0 {
				break
			}
			//fmt.Println("waiting for read at", i)
			time.Sleep(10 * time.Millisecond)
		}
		C.writeByte(c.membuf, c.writerOffset+index, C.char(data[i]))
		c.membuf.writeIndices[c.writerIndex] = (index + 1) % c.size
	}
	//fmt.Println("writeIndex: ", c.membuf.writeIndices[c.writerIndex])

	return len(data), nil
}

func (c *Conn) SetDeadline(t time.Time) error {
	//fmt.Println("SetDeadline called with ", t)
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
	if writeIndex >= size || readIndex >= size || size < 2 || size%2 != 0 {
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
	if writeIndex >= size || readIndex >= size || size < 2 || size%2 != 0 {
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
