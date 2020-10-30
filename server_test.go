package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/n0izn0iz/vm-vst-bridge-host/memconn"
)

func TestEcho(t *testing.T) {
	sizes := []int{4, 8, 16, 32, 42, 64, 128, 138, 256, 420, 512, 1024, 2048, 4096, 1024 * 1024}
	const shmemPath = "/dev/shm/ivshmem"

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

	rand.Seed(time.Now().UnixNano())

	for _, ringSize := range sizes {
		tLog.Debug("ringSize", zap.Int("value", ringSize))
		sLog := logger.Named("server")
		gLog := sLog.Named("grpc")
		lis := memconn.Listen(shmemPath, ringSize, 0, sLog)
		opts := []grpc_zap.Option{
			grpc_zap.WithLevels(grpc_zap.DefaultCodeToLevel),
		}
		s := grpc.NewServer(grpc_middleware.WithUnaryServerChain(
			grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.UnaryServerInterceptor(gLog, opts...),
		), grpc_middleware.WithStreamServerChain(
			grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.StreamServerInterceptor(gLog, opts...),
		))
		RegisterVSTBridgeServer(s, &server{logger: sLog.Named("impl")})
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Serve(lis); err != nil {
				require.Equal(t, memconn.ErrClosed, err)
			}
		}()

		dialer := memconn.Dialer(shmemPath, ringSize, 0, logger.Named("client"))

		for i := 0; i < 100; i++ {
			tLog.Debug("test", zap.Int("index", i))
			ctx, cancel := context.WithCancel(context.Background())
			conn, err := grpc.DialContext(ctx, "memconn", grpc.WithContextDialer(dialer), grpc.WithInsecure())
			require.NoError(t, err)
			client := NewVSTBridgeClient(conn)

			testStr := fmt.Sprint("test ", i)

			for j := 0; j < 100; j++ {
				tLog.Debug("calling echo", zap.Int("i", i), zap.Int("j", j))
				resp, err := client.Echo(ctx, &Echo_Request{Str: testStr})
				tLog.Debug("called echo", zap.Int("i", i), zap.Int("j", j), zap.Error(err))
				require.NoError(t, err)
				require.Equal(t, testStr, resp.GetStr())
			}

			require.NoError(t, conn.Close())
			cancel()
		}

		// stop server
		require.NoError(t, lis.Close())
		wg.Wait()
	}
}
