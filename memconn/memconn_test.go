package memconn

// #include "membuf.h"
import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMemconn(t *testing.T) {
	sizes := []int{4, 8, 16, 32, 42, 64, 128, 138, 256, 420, 512, 1024, 2048, 4096, 1024 * 1024}

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

		clientConn, serverConn, closeConn := testingConnPair(t, "/dev/shm/ivshmem", ringSize, 0, logger)

		runs := 50000

		done := make(chan struct{})
		go func() {
			testConnWrite(t, runs, serverConn, clientConn, fmt.Sprint("Hello conn 1-", ringSize), logger.Named("server"))
			close(done)
		}()

		testConnWrite(t, runs, clientConn, serverConn, fmt.Sprint("Hello conn 2-", ringSize), logger.Named("client"))

		<-done
		closeConn()
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

func testConnWrite(t *testing.T, runs int, in net.Conn, out net.Conn, testStr string, logger *zap.Logger) {
	t.Helper()

	done := make(chan struct{})

	go func() {
		for i := 0; i < runs; i++ {
			testStr := fmt.Sprint(testStr, "-", i)
			logger.Debug("new test", zap.Int("i", i), zap.String("testStr", testStr))
			buf := make([]byte, len(testStr))
			logger.Debug("waiting on read")
			n, err := io.ReadFull(out, buf)
			logger.Debug("read done")
			require.NoError(t, err)
			require.Equal(t, n, len(testStr))
			require.Equal(t, testStr, string(buf))
		}

		close(done)
	}()

	for i := 0; i < runs; i++ {
		testStr := fmt.Sprint(testStr, "-", i)
		n, err := in.Write([]byte(testStr))
		require.NoError(t, err)
		require.Equal(t, len(testStr), n)
	}

	<-done
}

func testingConnPair(t *testing.T, path string, ringSize, offset int, logger *zap.Logger) (net.Conn, net.Conn, func()) {
	t.Helper()

	l := Listen(path, ringSize, offset, logger.Named("server"))

	ch := make(chan net.Conn)
	defer close(ch)

	go func() {
		serverConn, err := l.Accept()
		require.NotNil(t, serverConn)
		require.NoError(t, err)
		ch <- serverConn
	}()

	dial := Dialer(path, ringSize, offset, logger.Named("client"))

	ctx, cancel := context.WithCancel(context.Background())

	clientConn, err := dial(ctx, "memconn")
	require.NotNil(t, clientConn)
	require.NoError(t, err)

	serverConn := <-ch

	close := func() {
		require.NoError(t, serverConn.Close())
		require.NoError(t, clientConn.Close())
		require.NoError(t, l.Close())
		cancel()
	}

	return clientConn, serverConn, close
}
