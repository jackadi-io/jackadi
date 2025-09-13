package autocompletion

import (
	"fmt"
	"log/slog"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jackadi-io/jackadi/internal/agent"
	"github.com/jackadi-io/jackadi/internal/collection"
	"github.com/jackadi-io/jackadi/internal/config"
	"github.com/spf13/cobra"
)

type TaskInfo struct {
	Name    string
	Summary string
}

type CollectionInfo struct {
	Name  string
	Tasks map[string]*TaskInfo
}

// GetAvailableCollections returns all available collections from the manager's plugins and built-ins.
func GetAvailableCollections() map[string]*CollectionInfo {
	collections := make(map[string]*CollectionInfo)

	builtinCollections := getBuiltinCollections()
	maps.Copy(collections, builtinCollections)

	pluginCollections := getExternalPluginCollections()
	maps.Copy(collections, pluginCollections)

	return collections
}

// getBuiltinCollections returns built-in collections from the registry.
func getBuiltinCollections() map[string]*CollectionInfo {
	collections := make(map[string]*CollectionInfo)

	agent.LoadBuiltins(nil)
	collectionNames := collection.Registry.Names()

	for _, name := range collectionNames {
		coll, err := collection.Registry.Get(name)
		if err != nil {
			slog.Debug("failed to get collection from registry", "name", name, "error", err)
			continue
		}

		help, err := coll.Help("")
		if err != nil {
			slog.Debug("failed to get help for collection", "name", name, "error", err)
			continue
		}

		collInfo := &CollectionInfo{
			Name:  name,
			Tasks: make(map[string]*TaskInfo),
		}

		for _, helpText := range help {
			parseHelpOutput(helpText, collInfo)
		}

		collections[name] = collInfo
	}

	return collections
}

// getExternalPluginCollections returns collections from external plugin files.
func getExternalPluginCollections() map[string]*CollectionInfo {
	pluginDir := config.DefaultPluginDir
	collections := make(map[string]*CollectionInfo)

	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		slog.Debug("plugin directory does not exist", "path", pluginDir)
		return collections
	}

	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		slog.Debug("failed to read plugin directory", "path", pluginDir, "error", err)
		return collections
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		pluginPath := filepath.Join(pluginDir, entry.Name())

		// Make sure the file is executable
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.Mode()&0111 == 0 {
			continue // Not executable
		}

		collectionInfo := getCollectionInfoFromPlugin(pluginPath)
		if collectionInfo != nil {
			collections[collectionInfo.Name] = collectionInfo
		}
	}

	return collections
}

// getCollectionInfoFromPlugin executes a plugin with --describe to get its information.
func getCollectionInfoFromPlugin(pluginPath string) *CollectionInfo {
	cmd := exec.Command(pluginPath, "--describe")
	cmd.Env = os.Environ()

	output, err := cmd.Output()
	if err != nil {
		slog.Debug("failed to execute plugin describe", "plugin", pluginPath, "error", err)
		return nil
	}

	return parseDescribeOutput(filepath.Base(pluginPath), string(output))
}

// parseDescribeOutput parses the output from plugin --describe command.
func parseDescribeOutput(pluginName string, output string) *CollectionInfo {
	collInfo := &CollectionInfo{
		Name:  pluginName,
		Tasks: make(map[string]*TaskInfo),
	}

	parseHelpOutput(output, collInfo)
	return collInfo
}

// parseHelpOutput parses help output and populates collection info.
func parseHelpOutput(output string, collInfo *CollectionInfo) {
	lines := strings.SplitSeq(output, "\n")

	for line := range lines {
		originalLine := line
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// format: "  taskname  summary  [flags]"
		if strings.HasPrefix(originalLine, "  ") && !strings.HasPrefix(originalLine, "   ") {
			parseTaskLine(originalLine, collInfo)
		}
	}
}

// parseTaskLine parses a single task line from describe output.
func parseTaskLine(line string, collInfo *CollectionInfo) {
	fields := strings.Fields(strings.TrimLeft(line, " "))
	if len(fields) == 0 {
		return
	}
	taskName := fields[0]

	collInfo.Tasks[taskName] = &TaskInfo{
		Name:    taskName,
		Summary: strings.Join(fields, " "),
	}
}

// GetCollectionTaskCompletions returns completion suggestions for collection:task format.
func GetCollectionTaskCompletions(toComplete string) ([]string, cobra.ShellCompDirective) {
	collections := GetAvailableCollections()
	completions := []string{}

	// complete task
	if strings.Contains(toComplete, ":") {
		parts := strings.SplitN(toComplete, ":", 2)
		if len(parts) != 2 {
			return completions, cobra.ShellCompDirectiveDefault
		}

		collectionName := parts[0]
		taskPrefix := parts[1]

		collInfo, exists := collections[collectionName]
		if !exists {
			return completions, cobra.ShellCompDirectiveDefault
		}

		for taskName, taskInfo := range collInfo.Tasks {
			if !strings.HasPrefix(taskName, taskPrefix) {
				continue
			}
			completion := fmt.Sprintf("%s:%s", collectionName, taskName)
			if taskInfo.Summary != "" {
				completion = fmt.Sprintf("%s\t%s", completion, taskInfo.Summary)
			}
			completions = append(completions, completion)
		}
		return completions, cobra.ShellCompDirectiveDefault
	}

	// complete collection only
	for collectionName := range collections {
		if strings.HasPrefix(collectionName, toComplete) {
			completions = append(completions, collectionName+":")
		}
	}

	return completions, cobra.ShellCompDirectiveDefault
}
