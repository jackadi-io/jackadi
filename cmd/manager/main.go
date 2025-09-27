package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/jackadi-io/jackadi/internal/config"
	"github.com/jackadi-io/jackadi/internal/manager/forwarder"
	"github.com/jackadi-io/jackadi/internal/manager/inventory"
	"github.com/jackadi-io/jackadi/internal/proto"
	flag "github.com/spf13/pflag"

	_ "github.com/jackadi-io/jackadi/internal/logs"
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

	clusterServer, grpcServer, appListener, err := startManager(cfg, &agentsInventory, taskDispatcher, db)
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
		clusterServer.CollectAgentsSpecs(ctx)
	}()

	pluginDir := http.Dir(cfg.pluginDir)
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

	grpcForwarder, cliListener, err := startCLIHandler(clusterServer, taskDispatcher, db)
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
	configFile := ""
	if flag.Lookup("config") != nil {
		configFile = flag.Lookup("config").Value.String()
	}
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
