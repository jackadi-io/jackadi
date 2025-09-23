package builtin

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"strings"

	"github.com/jackadi-io/jackadi/internal/plugin/core"
	"github.com/jackadi-io/jackadi/internal/plugin/inventory"
	"github.com/jackadi-io/jackadi/internal/plugin/types"
	"github.com/jackadi-io/jackadi/sdk"
)

func parseNames(name string) (string, string, error) {
	splitName := strings.Split(name, ":")
	if len(splitName) == 0 {
		return "", "", errors.New("missing collection or collection:task")
	}
	collectionName := splitName[0]

	if collectionName == "" {
		return "", "", errors.New("missing collection or collection:task")
	}

	taskName := ""
	if len(splitName) > 1 {
		taskName = splitName[1]
	}

	return collectionName, taskName, nil
}

type pluginMgmt struct {
	req  chan struct{}
	resp chan types.PluginUpdateResponse
}

func (s pluginMgmt) help(name string) (map[string]string, error) {
	collectionName, taskName, err := parseNames(name)
	if err != nil {
		return nil, err
	}

	c, err := inventory.Registry.Get(collectionName)
	if err != nil {
		return nil, fmt.Errorf("unknown collection: %s", collectionName)
	}

	return c.Help(taskName)
}

func (s pluginMgmt) version(collectionName string) (*core.Version, error) {
	c, err := inventory.Registry.Get(collectionName)
	if err != nil {
		return nil, fmt.Errorf("unknown collection: %w", err)
	}

	info, err := c.Version()
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	out := core.Version{
		PluginVersion: info.PluginVersion,
		Commit:        info.Commit,
		BuildTime:     info.BuildTime,
		GoVersion:     info.GoVersion,
	}

	return &out, nil
}

func (s pluginMgmt) list() ([]string, error) {
	return inventory.Registry.Names(), nil
}

type pluginInfo struct {
	Name string
	File *string `jackadi:"File,omitempty"`
}

type diff struct {
	Added     []pluginInfo `jackadi:"Added,omitempty"`
	Unchanged []pluginInfo `jackadi:"Unchanged,omitempty"`
	Deleted   []pluginInfo `jackadi:"Deleted,omitempty"`
	Updated   []pluginInfo `jackadi:"Updated,omitempty"`
}

func (s pluginMgmt) sync() (*diff, error) {
	s.req <- struct{}{} // request update to KeepPluginsUpToDate goroutine (internal/agent/plugins.go)
	changes := <-s.resp

	out := diff{}
	for _, p := range changes.Changes {
		info := pluginInfo{Name: p.Name}
		if p.Name != p.FileName {
			info.File = &p.FileName
		}

		switch {
		case p.New:
			out.Added = append(out.Added, info)
		case p.Updated:
			out.Updated = append(out.Updated, info)
		case p.Deleted:
			out.Deleted = append(out.Deleted, info)
		default:
			out.Unchanged = append(out.Unchanged, info)
		}
	}

	return &out, nil
}

func MustLoadPluginMgmt(req chan struct{}) chan types.PluginUpdateResponse {
	resp := make(chan types.PluginUpdateResponse)
	plugingMgmt := pluginMgmt{req: req, resp: resp}

	c := sdk.New("plugins")
	c.MustRegisterTask("help", plugingMgmt.help).
		WithSummary("Provide help for the given collection or collection:task.").
		WithDescription("Help for 'collection': gives the list of task with their summary.\nHelp for 'collection:task': gives the full details of the task.").
		WithArg("name", "collection[:task]", "cmd or cmd:run")
	c.MustRegisterTask("version", plugingMgmt.version).
		WithSummary("Provide version of the given collection.").
		WithDescription("Info for 'collection': gives version, commit id, build time, go version.").
		WithArg("name", "collection", "cmd")
	c.MustRegisterTask("list", plugingMgmt.list).
		WithSummary("List of task in the given collection.").
		WithArg("name", "collection", "cmd")
	c.MustRegisterTask("sync", plugingMgmt.sync).
		WithSummary("Sync plugin with the manager.").
		WithDescription("The agent sync its plugins with the manager.\nIt adds, updates and removes the collections following the manager configuration.").
		WithLockMode(sdk.ExclusiveLock)

	if err := inventory.Registry.Register(c); err != nil {
		name, _ := c.Name()
		slog.Error("could not load builtin task", "error", err, "task", name)
		log.Fatal(err)
	}

	return resp
}
