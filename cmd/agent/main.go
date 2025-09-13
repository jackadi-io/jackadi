package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackadi-io/jackadi/internal/agent"
	_ "github.com/jackadi-io/jackadi/internal/collection/builtin"
	"github.com/jackadi-io/jackadi/internal/config"
	_ "github.com/jackadi-io/jackadi/internal/logs"
	flag "github.com/spf13/pflag"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

type agentConfig struct {
	reconnectDelay int
	agent.Config
}

func run(cfg agentConfig) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	client, ctxMetadata, err := agent.New(cfg.Config)
	if err != nil {
		slog.Error("failed to initialize the agent", "error", err)
		return err
	}

	ctx, cancel := context.WithCancel(ctxMetadata)
	if err := client.Connect(ctx); err != nil {
		cancel()
		slog.Error("failed to connect the manager", "error", err)
		return err
	}

	slog.Debug("initializing", "agent-id", cfg.AgentID)

	wg := sync.WaitGroup{}
	defer func() {
		cancel()
		if err := client.Close(); err != nil {
			slog.Error("failed to close connection", "error", err)
		}
		slog.Info("waiting gracefully for all tasks to finish")
		done := make(chan struct{})
		go func() {
			wg.Wait()
			done <- struct{}{}
		}()

		select {
		case <-done:
			slog.Warn("all tasks stopped")
		case <-time.After(config.GracefulShutdownTimeout):
			slog.Warn("some tasks are still pending, force quit")
			client.KillPlugins()
		}
		slog.Warn("bye")
	}()

	specsSync := make(chan struct{}) // when a plugin is synced, the spec should be synced too
	wg.Add(1)
	go func() {
		defer wg.Done()
		client.KeepPluginsUpToDate(ctx, specsSync)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		client.SpecManager.StartSpecCollector(ctx, specsSync)
	}()

	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
			slog.Info("task listener closed")
		}()
		for {
			slog.Debug("handshake with manager in progress")
			err := func() error {
				ctxHandshake, cancel := context.WithTimeout(ctx, time.Minute)
				defer cancel()
				return client.Handshake(ctxHandshake)
			}()
			if err != nil {
				slog.Error("handshake failed", "error", err)

				select {
				case <-time.After(time.Duration(cfg.reconnectDelay) * time.Second):
					continue
				case <-ctx.Done():
					slog.Debug("context shutdown", "component", "main", "step", "wait before retrying handshake")
					break
				}
			}
			slog.Debug("handshake with manager successful")

			slog.Debug("'task' stream: connecting to the manager", "address", cfg.ManagerAddress, "port", cfg.ManagerPort)
			if err := client.ListenTaskRequest(ctx); err != nil {
				if status.Code(err) == codes.Canceled {
					slog.Info("closing task listener")
					break
				}
				slog.Error(
					"connection to the manager failed",
					"error", err,
					"component", "TaskRequest listener",
					"manager", cfg.ManagerAddress,
				)

				select {
				case <-time.After(time.Duration(cfg.reconnectDelay) * time.Second):
				case <-ctx.Done():
					slog.Debug("context shutdown", "component", "main", "step", "wait before reconnect")
				}
			}
		}
	}()

	<-sig
	slog.Warn("shutting down")

	return nil
}

func main() {
	versionCmd := flag.BoolP("version", "v", false, "print version")

	// Setup flags using the new config module
	config.SetupAgentFlags()

	flag.CommandLine.SortFlags = false
	flag.Parse()

	if *versionCmd {
		printVersion()
		os.Exit(0)
	}

	// Load configuration using Viper
	configFile := flag.Lookup("config").Value.String()
	agentCfg, err := config.LoadAgentConfig(configFile)

	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	cfg := agentConfig{
		reconnectDelay: agentCfg.ReconnectDelay,
		Config: agent.Config{
			AgentID:          agentCfg.AgentID,
			ManagerAddress:   agentCfg.ManagerAddress,
			ManagerPort:      agentCfg.ManagerPort,
			PluginDir:        agentCfg.PluginDir,
			PluginServerPort: agentCfg.PluginServerPort,
			MTLS:             agentCfg.MTLS,
			TLSKey:           agentCfg.TLSKey,
			TLSCert:          agentCfg.TLSCert,
			TLSManagerCA:     agentCfg.TLSManagerCA,
			CustomResolvers:  agentCfg.CustomResolvers,
		},
	}

	slog.Info("jackadi agent", "version", version, "commit", commit, "build date", date)

	if err := run(cfg); err != nil {
		slog.Error("shutdown", "error", err)
		os.Exit(1)
	}
}
