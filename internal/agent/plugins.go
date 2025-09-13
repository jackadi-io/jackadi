package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jackadi-io/jackadi/internal/collection"
	"github.com/jackadi-io/jackadi/internal/collection/builtin"
	"github.com/jackadi-io/jackadi/internal/collection/hcplugin"
	"github.com/jackadi-io/jackadi/internal/collection/stdplugin"
	"github.com/jackadi-io/jackadi/internal/collection/types"
	"github.com/jackadi-io/jackadi/internal/config"
	"google.golang.org/protobuf/types/known/emptypb"
)

var mu sync.Mutex

func (a *Agent) ListAgentPlugins(ctx context.Context) ([]string, error) {
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
	ctx, cancel := context.WithTimeout(ctxMetadata, config.PluginUpdateTimeout)
	defer cancel()

	list, err := a.ListAgentPlugins(ctx)
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

	slog.Debug("sync", "values", list)
	var errs error
	for _, name := range list {
		url := fmt.Sprintf("http://%s/plugin/%s", managerHost, name)
		slog.Debug("starting plugin sync", "plugin", name, "url", url)
		if err := a.download(name, url, tmpDir); err != nil {
			errs = errors.Join(errs, fmt.Errorf("new plugin not installed: '%s' file not downloaded: %w", name, err))
			slog.Error("failed to download", "plugin", name, "url", url, "error", err)
			continue
		}
		slog.Debug("plugin downloaded", "plugin", name, "url", url)
	}

	mu.Lock()
	slog.Debug("updating plugins")
	changes, changed, err := a.pluginLoader.Update(a.config.PluginDir, tmpDir)
	mu.Unlock()
	errs = errors.Join(errs, err)
	return changes, changed, errs
}

func (a *Agent) KillPlugins() {
	mu.Lock()
	a.pluginLoader.KillAll()
	mu.Unlock()
}

// download the plugin from the provided URL.
// The name is only the identifier of the plugin.
func (a *Agent) download(name, url, tmpDir string) error {
	client := http.Client{Timeout: time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("'%s': %w", url, err)
	}
	if resp == nil {
		return fmt.Errorf("'%s': received nil response", url)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("'%s' http error %d", url, resp.StatusCode)
	}

	localPath := filepath.Join(tmpDir, name)
	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create/access '%s' plugin file: %w", name, err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write '%s' plugin file : %w", name, err)
	}

	if err := os.Chmod(localPath, 0755); err != nil {
		return fmt.Errorf("chmod 644 failed for '%s' plugin file : %w", name, err)
	}
	return nil
}
