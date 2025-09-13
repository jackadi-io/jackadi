package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/jackadi-io/jackadi/internal/config"
	"github.com/jackadi-io/jackadi/internal/manager/forwarder"
	"github.com/jackadi-io/jackadi/internal/manager/inventory"
	"github.com/jackadi-io/jackadi/internal/manager/management"
	"github.com/jackadi-io/jackadi/internal/manager/server"
	"github.com/jackadi-io/jackadi/internal/proto"
	flag "github.com/spf13/pflag"

	_ "github.com/jackadi-io/jackadi/internal/logs"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

var version = "dev"
var commit = "N/A"
var date = "N/A"

func printVersion() {
	if version != "dev" {
		version = fmt.Sprintf("v%s", version)
	}
	fmt.Printf("%s (commit: %s, build date: %s)\n", version, commit, date)
}

type managerConfig struct {
	configDir        string
	listenAddress    string
	listenPort       string
	pluginDir        string
	pluginServerPort string
	autoAcceptAgent  bool
	mTLS             bool
	tlsCert          string
	tlsKey           string
	tlsAgentCA       string
}

func dbGC(ctx context.Context, db *badger.DB) {
	ticker := time.NewTicker(config.DatabaseGCInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := db.RunValueLogGC(config.DBGCThreshold)
			if err != nil {
				slog.Warn("database GC failed", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func startAppHandler(cfg managerConfig, agentsInventory *inventory.Agents, dis forwarder.Dispatcher[*proto.TaskRequest, *proto.TaskResponse], db *badger.DB) (*server.Server, *grpc.Server, net.Listener, error) {
	target := fmt.Sprint(cfg.listenAddress, ":", cfg.listenPort)
	lis, err := net.Listen("tcp", target)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to start TCP listener: %w", err)
	}

	var opts []grpc.ServerOption
	if cfg.mTLS {
		certs, ca, err := config.GetMTLSCertificate(cfg.tlsCert, cfg.tlsKey, cfg.tlsAgentCA)
		if err != nil {
			return nil, nil, nil, err
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
	commServer := server.New(
		server.ServerConfig{
			AutoAccept:  cfg.autoAcceptAgent,
			MTLSEnabled: cfg.mTLS,
			ConfigDir:   cfg.configDir,
		},
		agentsInventory,
		dis,
		db,
	)
	proto.RegisterCommServer(grpcServer, &commServer)

	slog.Info("starting gRPC server")
	slog.Info("listening", "address", cfg.listenAddress, "port", target)
	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server stopped", "reason", err)
		}
	}()
	return &commServer, grpcServer, lis, nil
}

func startCLIHandler(commServer *server.Server, dis forwarder.Dispatcher[*proto.TaskRequest, *proto.TaskResponse], db *badger.DB) (*grpc.Server, net.Listener, error) {
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

	mgmtServer := management.New(commServer, db)
	proto.RegisterAPIServer(grpcServer, &mgmtServer)

	slog.Info("starting local gRPC server for CLI")
	go func() {
		if err = grpcServer.Serve(lisCLI); err != nil {
			slog.Error("gRPC local server stopped", "reason", err)
		}
	}()

	return grpcServer, lisCLI, nil
}

func run(cfg managerConfig) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	dbOptions := badger.
		DefaultOptions(config.DatabaseDir).
		WithLogger(slogBadgerAdapter{})

	db, err := badger.Open(dbOptions)
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go dbGC(ctx, db)

	agentsInventory := inventory.New()
	if err := agentsInventory.LoadRegistry(); err != nil {
		slog.Info("unable to load registry", "error", err)
	}
	taskDispatcher := forwarder.NewDispatcher[*proto.TaskRequest, *proto.TaskResponse](&agentsInventory)

	commServer, grpcServer, appListener, err := startAppHandler(cfg, &agentsInventory, taskDispatcher, db)
	defer func() {
		if grpcServer != nil {
			grpcServer.Stop()
		}
		if appListener != nil {
			_ = appListener.Close()
		}
	}()
	if err != nil {
		return err
	}

	go func() {
		commServer.CollectAgentsSpecs(ctx)
	}()

	pluginDir := http.Dir(config.DefaultPluginDir)
	fs := http.FileServer(pluginDir)
	mux := http.NewServeMux()
	mux.Handle("GET "+config.PluginServerPath, http.StripPrefix(config.PluginServerPath, fs))

	socket := net.JoinHostPort(cfg.listenAddress, cfg.pluginServerPort)
	httpServer := http.Server{Addr: socket, Handler: mux, ReadHeaderTimeout: config.HTTPReadHeaderTimeout}
	go func() {
		slog.Info("Starting static webserver", "socket", socket)
		err = httpServer.ListenAndServe()
		if err != nil {
			slog.Error("http server stopped", "error", err)
		}
	}()

	grpcForwarder, cliListener, err := startCLIHandler(commServer, taskDispatcher, db)
	defer func() {
		if grpcForwarder != nil {
			grpcForwarder.Stop()
		}
		if cliListener != nil {
			_ = cliListener.Close()
		}
	}()
	if err != nil {
		return err
	}

	// graceful shutdown
	<-c
	slog.Warn("shutdown")
	if err := httpServer.Shutdown(context.Background()); err != nil {
		slog.Error("plugin server: graceful shutdown failed", "error", err)
	}

	return nil
}

func main() {
	versionCmd := flag.BoolP("version", "v", false, "print version")

	// Setup flags using the new config module
	config.SetupManagerFlags()

	flag.CommandLine.SortFlags = false
	flag.Parse()

	if *versionCmd {
		printVersion()
		os.Exit(0)
	}

	// Load configuration using Viper
	configFile := flag.Lookup("config").Value.String()
	managerCfg, err := config.LoadManagerConfig(configFile)

	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	cfg := managerConfig{
		listenAddress:    managerCfg.ListenAddress,
		listenPort:       managerCfg.ListenPort,
		pluginDir:        managerCfg.PluginDir,
		pluginServerPort: managerCfg.PluginServerPort,
		mTLS:             managerCfg.MTLS,
		tlsKey:           managerCfg.TLSKey,
		tlsCert:          managerCfg.TLSCert,
		tlsAgentCA:       managerCfg.TLSAgentCA,
		autoAcceptAgent:  managerCfg.AutoAcceptAgent,
		configDir:        managerCfg.ConfigDir,
	}

	slog.Info("jackadi manager", "version", version, "commit", commit, "build date", date)

	if err := run(cfg); err != nil {
		slog.Error("shutdown", "error", err)
		os.Exit(1)
	}
}
