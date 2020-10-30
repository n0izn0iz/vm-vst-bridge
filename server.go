package main

import (
	"context"

	"go.uber.org/zap"
)

type server struct {
	UnimplementedVSTBridgeServer
	logger *zap.Logger
}

var _ VSTBridgeServer = (*server)(nil)

func (s *server) Echo(ctx context.Context, in *Echo_Request) (*Echo_Reply, error) {
	s.logger.Debug("VSTBridge.Echo", zap.String("str", in.Str))
	return &Echo_Reply{Str: in.Str}, nil
}
