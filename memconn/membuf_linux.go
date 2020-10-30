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

void readBytes(membuf* mb, void* dst, uint64_t dstOffset, uint64_t start, size_t len) {
	memcpy(((char*)dst) + dstOffset, (&mb->data) + start, len);
}

void writeBytes(membuf* mb, void* src, uint64_t srcOffset, uint64_t start, size_t len) {
	memcpy((&mb->data) + start, ((char*)src) + srcOffset, len);
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
	"unsafe"

	"go.uber.org/zap"
)

func initShmem(shmem_device_file string, size *C.size_t, err *C.int) *C.membuf {
	return C.initShmem(shmem_device_file, size, err)
}

func readBytes(mb *C.membuf, dst unsafe.Pointer, dstOffset C.uint64_t, start C.uint64_t, len C.size_t) {
	C.readBytes(mb, dst, dstOffset, start, len)
}

func writeBytes(mb *C.membuf, src unsafe.Pointer, srcOffset C.uint64_t, start C.uint64_t, len C.size_t) {
	C.writeBytes(mb, src, srcOffset, start, len)
}

func newMembuf(shmemPath string, ringSize int, offset int, logger *zap.Logger) (*C.membuf, C.size_t) {
	if len(shmemPath) <= 0 || offset < 0 || ringSize < 4 || ringSize%2 != 0 {
		panic(fmt.Sprint("invalid parameter(s): shmemPath: ", shmemPath, ", ringSize: ", ringSize, ", offset: ", offset))
	}

	var ret C.int
	var memSize C.size_t
	mem := initShmem(shmemPath, &memSize, &ret)
	if ret != 0 {
		panic(fmt.Sprint("failed to init shmem: ret=", ret))
	}
	logger.Debug("ivshmem:",
		zap.Int("memSizeMiB", int(memSize/1024/1024)),
		zap.Int("offsetB", offset),
		zap.Int("ringSizeB", ringSize),
	)

	// FIXME: compute real maximum
	if C.size_t(ringSize) > (memSize/2)-C.size_t(offset) {
		panic("ring too big for ivshmem")
	}

	return mem, memSize
}
