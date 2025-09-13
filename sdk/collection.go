package sdk

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"runtime/debug"
	"strings"

	goplugin "github.com/hashicorp/go-plugin"

	"github.com/jackadi-io/jackadi/internal/plugin"
	"github.com/jackadi-io/jackadi/internal/proto"
)

var version = ""
var commit = ""
var date = ""

// getVersion returns plugin information.
//
// LDFLAGS (manually or via goreleaser) are used, with debug.RealBuildInfo as fallback.
func getVersion() plugin.Version {
	v := plugin.Version{
		PluginVersion: version,
		Commit:        commit,
		BuildTime:     date,
	}

	if version != "" && commit != "" && date != "" {
		return v
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return v
	}

	if v.PluginVersion == "" {
		v.PluginVersion = info.Main.Version
	}
	v.GoVersion = info.GoVersion
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if v.Commit == "" {
				v.Commit = setting.Value
			}
		case "vcs.time":
			if v.BuildTime == "" {
				v.BuildTime = setting.Value
			}
		}
	}

	return v
}

type Options interface {
	SetDefaults()
}

type Collection struct {
	name      string
	tasks     map[string]*Task
	taskNames []string // to keep an ordered list

	specs      map[string]*SpecCollector
	specsNames []string
}

func New(name string) *Collection {
	return &Collection{
		name:  name,
		tasks: make(map[string]*Task),
		specs: make(map[string]*SpecCollector),
	}
}

func (t Collection) Name() (string, error) { return t.name, nil }

func (t Collection) Tasks() ([]string, error) {
	return t.taskNames, nil
}

func (t Collection) Help(taskName string) (map[string]string, error) {
	res := make(map[string]string)

	// if no specific task, returns the short description (help) of every tasks
	if taskName == "" {
		var sb strings.Builder
		maxNameLen := 0
		for _, name := range t.taskNames {
			if len(name) > maxNameLen {
				maxNameLen = len(name)
			}
		}

		for _, name := range t.taskNames {
			task := t.tasks[name]
			if task == nil {
				slog.Error("failed to get help for task", "task", name, "error", "not found")
				continue
			}
			line := fmt.Sprintf("  %-*s", maxNameLen, task.name)

			if task.summary != "" {
				line += "  " + task.summary
			}

			if flags := task.flagsString(); flags != "" {
				line += fmt.Sprintf("  [%s]", flags)
			}

			sb.WriteString(line + "\n")
		}
		res[t.name] = sb.String()

		return res, nil
	}

	// if specific task, return the full details
	task, ok := t.tasks[taskName]
	if !ok {
		return nil, fmt.Errorf("unknown task: %s", taskName)
	}

	res[taskName] = task.helpText(t.name)
	return res, nil
}

func (t Collection) Version() (plugin.Version, error) {
	return getVersion(), nil
}

func (t Collection) GetTaskLockMode(taskName string) (proto.LockMode, error) {
	task, ok := t.tasks[taskName]
	if !ok {
		return proto.LockMode_NO_LOCK, fmt.Errorf("unknown task: %s", taskName)
	}
	return task.getLockMode().toProtoLockMode(), nil
}

func MustServe(collection *Collection) {
	if exit := handleFlags(collection); exit {
		return
	}

	tasks, _ := collection.Tasks()
	if len(tasks) == 0 {
		log.Fatalln("colletion must have at least one task")
	}

	cfg := goplugin.ServeConfig{
		HandshakeConfig: plugin.Handshake,
		Plugins: map[string]goplugin.Plugin{
			"collection": &plugin.CollectionPlugin{Impl: collection},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
	}
	goplugin.Serve(&cfg)
}

// handleFlags processes command-line flags for the plugin and returns true if the plugin should exit.
func handleFlags(collection *Collection) bool {
	versionFlag := flag.Bool("version", false, "print plugin information")
	describeFlag := flag.Bool("describe", false, "decribe plugin")
	flag.Parse()

	if *versionFlag {
		fmt.Println(getVersion())
		return true
	}

	if *describeFlag {
		help, _ := collection.Help("")
		for _, h := range help {
			fmt.Println(h)
		}
		return true
	}

	return false
}
