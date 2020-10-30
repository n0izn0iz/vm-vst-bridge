#ifndef MEMBUF_H
#define MEMBUF_H

#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>

typedef struct membuf {
	uint64_t	challenge;
	uint64_t	answer;
	bool			connected[2];
	uint64_t	readIndices[2];
	uint64_t	writeIndices[2];
	char			data;
} membuf;

#endif // MEMBUF_H
