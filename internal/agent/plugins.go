package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync"

	"github.com/jackadi-io/jackadi/internal/collection"
	"github.com/jackadi-io/jackadi/internal/collection/builtin"
	"github.com/jackadi-io/jackadi/internal/collection/hcplugin"
	"github.com/jackadi-io/jackadi/internal/collection/stdplugin"
	"github.com/jackadi-io/jackadi/internal/collection/types"
	"github.com/jackadi-io/jackadi/internal/config"
	"google.golang.org/protobuf/types/known/emptypb"
)

var mu sync.Mutex

func (a *Agent) AgentPlugins(ctx context.Context) (map[string]string, error) {
	res, err := a.taskClient.ListAgentPlugins(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("server query failed: %w", err)
	}

	return res.GetPlugin(), nil
}

func LoadBuiltins(syncReq chan struct{}) chan types.PluginUpdateResponse {
	builtin.MustLoadCmd()
	builtin.MustLoadHealth()
	return builtin.MustLoadPluginMgmt(syncReq)
}

func (a *Agent) KeepPluginsUpToDate(ctxMetadata context.Context, specsSync chan struct{}) {
	defer slog.Info("plugin manager closed")

	// Load Go standard plugin (not prefered)
	stdplugin.Load(a.config.PluginDir)

	// Load hashicorp type plugins
	hcplugins := hcplugin.New()
	hcplugins.Load(a.config.PluginDir)
	slog.Info("loaded plugins", "plugins", collection.Registry.Names())

	mu.Lock()
	a.pluginLoader = hcplugins
	mu.Unlock()

	syncReq := make(chan struct{})
	resp := LoadBuiltins(syncReq)

	for {
		select {
		case <-syncReq:
			changes, changed, err := a.updatePlugins(ctxMetadata)
			ret := types.PluginUpdateResponse{Changes: changes, Error: err}
			resp <- ret
			if changed {
				specsSync <- struct{}{}
			}
		case <-ctxMetadata.Done():
			return
		}
	}
}

func (a *Agent) updatePlugins(ctxMetadata context.Context) ([]types.PluginChanges, bool, error) {
	mu.Lock()
	defer mu.Unlock()

	ctx, cancel := context.WithTimeout(ctxMetadata, config.PluginUpdateTimeout)
	defer cancel()

	agentPlugins, err := a.AgentPlugins(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get plugin list: %w", err)
	}

	tmpDir, err := os.MkdirTemp("/tmp", "jackadi-plugin-tmp")
	if err != nil {
		return nil, false, fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			slog.Error("failed to remove plugin tmp directory", "error", err)
		}
	}()

	// if for some we failed to resolve the manager address during stream connection, we fallback on the configured manager address
	managerHost := net.JoinHostPort(a.connectedManagerAddr, a.config.PluginServerPort)
	if managerHost == "" {
		managerHost = net.JoinHostPort(a.config.ManagerAddress, a.config.PluginServerPort)
	}

	upToDate, err1 := a.pluginLoader.DownloadPlugins(agentPlugins, managerHost, a.config.PluginDir, tmpDir)
	slog.Debug("updating plugins")
	changes, changed, err2 := a.pluginLoader.Update(a.config.PluginDir, tmpDir, upToDate)
	return changes, changed, errors.Join(err1, err2)
}

func (a *Agent) KillPlugins() {
	mu.Lock()
	a.pluginLoader.KillAll()
	mu.Unlock()
}
