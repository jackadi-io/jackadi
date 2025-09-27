package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger/v4"
	"github.com/jackadi-io/jackadi/internal/config"
	"github.com/jackadi-io/jackadi/internal/manager/forwarder"
	"github.com/jackadi-io/jackadi/internal/manager/management"
	"github.com/jackadi-io/jackadi/internal/manager/server"
	"github.com/jackadi-io/jackadi/internal/proto"
	"google.golang.org/grpc"
)

func startCLIHandler(clusterServer *server.Server, dis forwarder.Dispatcher[*proto.TaskRequest, *proto.TaskResponse], db *badger.DB) (*grpc.Server, net.Listener, error) {
	_ = os.MkdirAll(filepath.Dir(config.CLISocket), 0755)

	lisCLI, err := net.Listen("unix", config.CLISocket)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to CLI socket: %w", err)
	}

	if err := os.Chmod(config.CLISocket, 0700); err != nil {
		return nil, lisCLI, fmt.Errorf("failed to secure CLI socket: %w", err)
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	fwd := forwarder.New(dis, db)
	proto.RegisterForwarderServer(grpcServer, &fwd)

	apiServer := management.New(clusterServer, db)
	proto.RegisterAPIServer(grpcServer, &apiServer)

	slog.Info("starting local gRPC server for CLI")
	go func() {
		if err = grpcServer.Serve(lisCLI); err != nil {
			slog.Error("gRPC local server stopped", "reason", err)
		}
	}()

	go func() {
		err := startHTTPProxy()
		if err != nil {
			slog.Error("HTTP server stopped", "reason", err)
		}
	}()

	return grpcServer, lisCLI, nil
}
