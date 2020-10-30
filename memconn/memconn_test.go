package memconn

// #include "membuf.h"
import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMemconn(t *testing.T) {
	sizes := []int{4, 8, 16, 32, 42, 64, 128, 138, 256, 420, 512, 1024, 2048, 4096}

	var logger *zap.Logger
	if os.Getenv("DEBUG") == "true" {
		conf := zap.NewDevelopmentConfig()
		if len(os.Getenv("LOGFILE")) > 0 {
			conf.OutputPaths = []string{os.Getenv("LOGFILE")}
		}
		var err error
		logger, err = conf.Build()
		require.NoError(t, err)
	} else {
		logger = zap.NewNop()
	}
	defer logger.Sync() // flushes buffer, if any

	tLog := logger.Named("test")

	for _, ringSize := range sizes {
		tLog.Debug("new test group", zap.Int("ringSize", ringSize))

		clientConn, serverConn, close := testingConnPair(t, "/dev/shm/ivshmem", ringSize, 0, logger)

		for i := 0; i < 50; i++ {
			tLog.Debug("new test", zap.Int("i", i), zap.Int("ringSize", ringSize))
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				testConnWrite(t, serverConn, clientConn, fmt.Sprint("Hello conn 1-", i, ringSize), logger.Named("test"))
			}()
			testConnWrite(t, clientConn, serverConn, fmt.Sprint("Hello conn 2-", i, ringSize), logger.Named("test"))
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
}
