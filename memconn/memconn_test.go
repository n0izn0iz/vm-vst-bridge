package memconn

// #include "membuf.h"
import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemconn(t *testing.T) {
	sizes := []int{4, 8, 16, 32, 64, 128, 256}
	for _, ringSize := range sizes {
		t.Log("ringSize: ", ringSize)

		clientConn, serverConn, close := testingConnPair(t, "/dev/shm/ivshmem", ringSize, 0)

		t.Log("starting writes")

		for i := 0; i < 5; i++ {
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				testConnWrite(t, serverConn, clientConn, fmt.Sprint("Hello conn 1-", i, ringSize))
			}()
			testConnWrite(t, clientConn, serverConn, fmt.Sprint("Hello conn 2-", i, ringSize))
			wg.Wait()
		}

		close()
	}
}

func TestWriteLen(t *testing.T) {
	require.Equal(t, uint64_t(1), writeLen(0, 0, 2))
	require.Equal(t, uint64_t(0), writeLen(1, 0, 2))
	require.Equal(t, uint64_t(0), writeLen(0, 1, 2))
	require.Equal(t, uint64_t(1), writeLen(1, 1, 2))

	require.Equal(t, uint64_t(7), writeLen(0, 0, 8))
	require.Equal(t, uint64_t(4), writeLen(5, 2, 8))
	require.Equal(t, uint64_t(4), writeLen(1, 6, 8))
	require.Equal(t, uint64_t(6), writeLen(0, 7, 8))
	require.Equal(t, uint64_t(0), writeLen(7, 0, 8))
	require.Equal(t, uint64_t(0), writeLen(0, 1, 8))

	require.Panics(t, func() { _ = writeLen(8, 0, 8) })
	require.Panics(t, func() { _ = writeLen(0, 8, 8) })
	require.Panics(t, func() { _ = writeLen(21, 0, 8) })
	require.Panics(t, func() { _ = writeLen(0, 21, 8) })
	require.Panics(t, func() { _ = writeLen(0, 0, 0) })
	require.Panics(t, func() { _ = writeLen(0, 0, 1) })
	require.Panics(t, func() { _ = writeLen(0, 0, 21) })
}

func TestReadLen(t *testing.T) {
	require.Equal(t, uint64_t(0), readLen(0, 0, 2))
	require.Equal(t, uint64_t(1), readLen(1, 0, 2))
	require.Equal(t, uint64_t(1), readLen(0, 1, 2))
	require.Equal(t, uint64_t(0), readLen(1, 1, 2))

	require.Equal(t, uint64_t(0), readLen(0, 0, 8))
	require.Equal(t, uint64_t(3), readLen(5, 2, 8))
	require.Equal(t, uint64_t(3), readLen(1, 6, 8))
	require.Equal(t, uint64_t(1), readLen(0, 7, 8))
	require.Equal(t, uint64_t(7), readLen(7, 0, 8))
	require.Equal(t, uint64_t(7), readLen(0, 1, 8))

	require.Panics(t, func() { _ = readLen(8, 0, 8) })
	require.Panics(t, func() { _ = readLen(0, 8, 8) })
	require.Panics(t, func() { _ = readLen(21, 0, 8) })
	require.Panics(t, func() { _ = readLen(0, 21, 8) })
	require.Panics(t, func() { _ = readLen(0, 0, 0) })
	require.Panics(t, func() { _ = readLen(0, 0, 1) })
	require.Panics(t, func() { _ = readLen(0, 0, 21) })
}
