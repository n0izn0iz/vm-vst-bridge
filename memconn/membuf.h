#ifndef MEMBUF_H
#define MEMBUF_H

#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>

typedef struct membuf {
	uint64_t	challenge;
	uint64_t	answer;
	bool			connected;
	uint64_t	readIndices[2];
	uint64_t	writeIndices[2];
	char			data;
} membuf;

void writeByte(membuf* mb, size_t offset, char val);
char readByte(membuf* mb, size_t offset);
membuf* initShmem(_GoString_ _shmem_device_file, size_t *size, int* err);

#endif // MEMBUF_H
