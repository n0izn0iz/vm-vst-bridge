package memconn

import (
	"context"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testConnWrite(t *testing.T, in net.Conn, out net.Conn, testStr string, logger *zap.Logger) {
	t.Helper()

	ch := make(chan []byte)
	defer close(ch)

	go func() {
		buf := make([]byte, len(testStr))
		logger.Debug("waiting on read")
		n, err := io.ReadFull(out, buf)
		logger.Debug("read done")
		require.NoError(t, err)
		require.Equal(t, n, len(testStr))
		ch <- buf
	}()

	n, err := in.Write([]byte(testStr))
	require.NoError(t, err)
	require.Equal(t, len(testStr), n)

	buf := <-ch
	require.Equal(t, testStr, string(buf))
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

	ctx := context.TODO()

	clientConn, err := dial(ctx, "memconn")
	require.NotNil(t, clientConn)
	require.NoError(t, err)

	serverConn := <-ch

	close := func() {
		require.NoError(t, serverConn.Close())
		require.NoError(t, clientConn.Close())
		require.NoError(t, l.Close())
	}

	return clientConn, serverConn, close
}
