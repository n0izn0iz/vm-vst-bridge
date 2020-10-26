package main

import (
	"context"
	"fmt"
	"log"
)

type server struct {
	UnimplementedVSTBridgeServer
}

var _ VSTBridgeServer = (*server)(nil)

func (s *server) Echo(ctx context.Context, in *Echo_Request) (*Echo_Reply, error) {
	log.Printf("Received: %v", in.Str)
	fmt.Println("RECEIVED!!!!!")
	return &Echo_Reply{Str: in.Str}, nil
}
