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
	"net"
)

type Listener struct {
	mb *C.membuf
}

var _ net.Listener = (*Listener)(nil)

func Listen(shmemPath string, ringSize int, offset int) Listener {
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
	size := C.size_t(ringSize)
	fmt.Println("ring size:", size, "B")

	return &Listener{mb}
}

func (l *Listener) Close() error {
	return nil
}

func (l *Listener) LocalAddr() net.Addr  { return addr{} }
func (l *Listener) RemoteAddr() net.Addr { return addr{} }

type addr struct{}

func (a addr) Network() string { return "memconn" }
func (a addr) String() string  { return "memconn" }
