package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/dgraph-io/badger/v4"
	"github.com/jackadi-io/jackadi/internal/config"
	"github.com/jackadi-io/jackadi/internal/manager/forwarder"
	"github.com/jackadi-io/jackadi/internal/manager/inventory"
	"github.com/jackadi-io/jackadi/internal/manager/server"
	"github.com/jackadi-io/jackadi/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

type ManagerInstance struct {
	ClusterServer *server.Server
	grpcServer    *grpc.Server
	listener      net.Listener
	addr          string
}

func (m *ManagerInstance) Serve() error {
	if m.grpcServer == nil {
		return errors.New("grpc server is nil")
	}

	slog.Info("starting gRPC server")
	slog.Info("listening", "socket", m.addr)
	return m.grpcServer.Serve(m.listener)
}

func (m *ManagerInstance) Close() {
	if m.listener != nil {
		_ = m.listener.Close()
	}

	if m.grpcServer != nil {
		m.grpcServer.Stop()
	}
}

func (m *ManagerInstance) CollectAgentsSpecs(ctx context.Context) error {
	if m.ClusterServer == nil {
		return errors.New("server not initialized properly")
	}

	m.ClusterServer.CollectAgentsSpecs(ctx)
	return nil
}

func newManager(cfg managerConfig, agentsInventory *inventory.Agents, dis forwarder.Dispatcher[*proto.TaskRequest, *proto.TaskResponse], db *badger.DB) (*ManagerInstance, error) {
	target := fmt.Sprint(cfg.listenAddress, ":", cfg.listenPort)
	lis, err := net.Listen("tcp", target)
	if err != nil {
		return nil, fmt.Errorf("failed to start TCP listener: %w", err)
	}

	var opts []grpc.ServerOption
	if cfg.mTLS {
		certs, ca, err := config.GetMTLSCertificate(cfg.mTLSCert, cfg.mTLSKey, cfg.mTLSAgentCA)
		if err != nil {
			return nil, err
		}
		tlsCfg := &tls.Config{
			MinVersion:   tls.VersionTLS12,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			Certificates: certs,
			ClientCAs:    ca,
		}
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsCfg)))
	} else {
		slog.Warn("mTLS is disabled, connections to agents are unsafe")
	}

	opts = append(opts,
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             config.KeepaliveMinTime, // If a client pings more than once every 5 seconds, terminate the connection
			PermitWithoutStream: true,                    // Allow pings even when there are no active streams
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    config.KeepaliveTime,    // Ping the client if it is idle for 5 seconds to ensure the connection is still active
			Timeout: config.KeepaliveTimeout, // Wait 1 second for the ping ack before assuming the connection is dead
		}),
	)

	grpcServer := grpc.NewServer(opts...)
	clusterServer := server.New(
		server.ServerConfig{
			AutoAccept:  cfg.autoAcceptAgent,
			MTLSEnabled: cfg.mTLS,
			ConfigDir:   cfg.configDir,
			PluginDir:   cfg.pluginDir,
		},
		agentsInventory,
		dis,
		db,
	)
	proto.RegisterClusterServer(grpcServer, &clusterServer)

	m := ManagerInstance{
		ClusterServer: &clusterServer,
		grpcServer:    grpcServer,
		listener:      lis,
		addr:          target,
	}

	return &m, nil
}
