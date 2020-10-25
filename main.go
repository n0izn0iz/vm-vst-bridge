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

typedef struct membuf {
	uint64_t writer_id;
	uint64_t reader_id;
	uint64_t write_idx;
  uint64_t read_idx;
	void		 data;
} membuf;

int initShmem(_GoString_ _shmem_device_file, membuf** mb)
{
	size_t len = _GoStringLen(_shmem_device_file);
	char shmem_device_file[len + 1];
	memcpy(shmem_device_file, _GoStringPtr(_shmem_device_file), len);
	shmem_device_file[len] = '\0';

  struct stat st;
  if (stat(shmem_device_file, &st) < 0)  {
    fprintf(stderr, "Failed to stat the shared memory file: %s\n", shmem_device_file);
    return 2;
  }

  int shmFD = open(shmem_device_file, O_RDONLY);
  if (shmFD < 0) {
    fprintf(stderr, "Failed to open the shared memory file: %s\n", shmem_device_file);
    return 3;
  }

  *mb = mmap(0, st.st_size, PROT_READ, MAP_SHARED, shmFD, 0);
  if (*mb == MAP_FAILED) {
    fprintf(stderr, "Failed to map the shared memory file: %s\n", shmem_device_file);
    close(shmFD);
    return 4;
  }

  return 0;
}

receiver_data_t recvShmem(rctx_shmem_t* rctx_shmem)
{
	receiver_data_t _receiver_data;
	receiver_data_t* receiver_data = &_receiver_data;

  struct shmheader *header = (struct shmheader*)rctx_shmem->mmap;

  int valid = 0;
  do {
    if (header->magic != 0x11112014) {
			printf("waiting for magic\n");
      while (header->magic != 0x11112014) {
        usleep(10000);//10ms
      }
			printf("magic arrived\n");
      rctx_shmem->read_idx = header->write_idx;
      continue;
    }

    if (rctx_shmem->read_idx == header->write_idx) {
      usleep(10000);//10ms
      continue;
	  }

    valid = 1;
  } while (!valid);

  if (++(rctx_shmem->read_idx) == header->max_chunks) {
    rctx_shmem->read_idx = 0;
  }

  receiver_data->data_size = header->chunk_size;
  receiver_data->data = &rctx_shmem->mmap[header->offset+header->chunk_size*rctx_shmem->read_idx];

	return _receiver_data;
}
*/
import "C"
import (
	"fmt"
)

func main() {
	fmt.Println("Hello")

	var mb *C.membuf
	ret := C.initShmem("/dev/shm/ivshmem", &mb)
	if ret != 0 {
		panic(fmt.Sprint("failed to init shmem: ret=", ret))
	}

	fmt.Println("Listening..")
	for {
		d := C.recvShmem(&rctxShmem)
		fmt.Print(C.GoStringN(d.data, d.data_size))
	}
}
