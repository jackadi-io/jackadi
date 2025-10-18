package sdk

import (
	"context"
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		pluginName string
	}{
		{
			name:       "simple name",
			pluginName: "test-plugin",
		},
		{
			name:       "empty name",
			pluginName: "",
		},
		{
			name:       "name with special characters",
			pluginName: "my-plugin_v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.pluginName)
			if p == nil {
				t.Fatal("New() returned nil")
			}
			if p.name != tt.pluginName {
				t.Errorf("New() name = %v, want %v", p.name, tt.pluginName)
			}
			if p.tasks == nil {
				t.Error("New() tasks map is nil")
			}
			if p.specs == nil {
				t.Error("New() specs map is nil")
			}
		})
	}
}

func TestPluginName(t *testing.T) {
	tests := []struct {
		name       string
		pluginName string
	}{
		{
			name:       "valid name",
			pluginName: "test-plugin",
		},
		{
			name:       "empty name",
			pluginName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.pluginName)
			name, err := p.Name()
			if err != nil {
				t.Errorf("Name() unexpected error: %v", err)
			}
			if name != tt.pluginName {
				t.Errorf("Name() = %v, want %v", name, tt.pluginName)
			}
		})
	}
}

func TestPluginTasks(t *testing.T) {
	tests := []struct {
		name      string
		taskNames []string
	}{
		{
			name:      "no tasks",
			taskNames: []string{},
		},
		{
			name:      "single task",
			taskNames: []string{"task1"},
		},
		{
			name:      "multiple tasks",
			taskNames: []string{"task1", "task2", "task3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New("test-plugin")
			p.taskNames = tt.taskNames

			tasks, err := p.Tasks()
			if err != nil {
				t.Errorf("Tasks() unexpected error: %v", err)
			}
			if len(tasks) != len(tt.taskNames) {
				t.Errorf("Tasks() count = %v, want %v", len(tasks), len(tt.taskNames))
			}
			for i, task := range tt.taskNames {
				if i >= len(tasks) || tasks[i] != task {
					t.Errorf("Tasks()[%d] = %v, want %v", i, tasks[i], task)
				}
			}
		})
	}
}

func TestGetVersion(t *testing.T) {
	v := getVersion()
	if v.PluginVersion == "" && v.Commit == "" && v.BuildTime == "" && v.GoVersion == "" {
		t.Log("getVersion() returned empty version (expected when no build info available)")
	}
}

func TestSpecCollectorWithSummary(t *testing.T) {
	tests := []struct {
		name    string
		summary string
	}{
		{
			name:    "simple summary",
			summary: "Test summary",
		},
		{
			name:    "empty summary",
			summary: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &SpecCollector{}
			result := sc.WithSummary(tt.summary)
			if result != sc {
				t.Error("WithSummary() did not return same instance")
			}
			if sc.summary != tt.summary {
				t.Errorf("WithSummary() summary = %v, want %v", sc.summary, tt.summary)
			}
		})
	}
}

func TestSpecCollectorWithDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "simple description",
			description: "Test description",
		},
		{
			name:        "empty description",
			description: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &SpecCollector{}
			result := sc.WithDescription(tt.description)
			if result != sc {
				t.Error("WithDescription() did not return same instance")
			}
			if sc.description != tt.description {
				t.Errorf("WithDescription() description = %v, want %v", sc.description, tt.description)
			}
		})
	}
}

func TestSpecCollectorWithFlags(t *testing.T) {
	tests := []struct {
		name  string
		flags []Flag
	}{
		{
			name:  "single flag",
			flags: []Flag{Deprecated},
		},
		{
			name:  "multiple flags",
			flags: []Flag{Deprecated, Experimental},
		},
		{
			name:  "no flags",
			flags: []Flag{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &SpecCollector{}
			result := sc.WithFlags(tt.flags...)
			if result != sc {
				t.Error("WithFlags() did not return same instance")
			}
			if len(sc.flags) != len(tt.flags) {
				t.Errorf("WithFlags() flags count = %v, want %v", len(sc.flags), len(tt.flags))
			}
		})
	}
}

func TestCollectSpecs(t *testing.T) {
	type testSpec struct {
		Value string
	}

	tests := []struct {
		name    string
		specs   map[string]any
		wantErr bool
	}{
		{
			name: "valid spec returning struct",
			specs: map[string]any{
				"test": func() (testSpec, error) {
					return testSpec{Value: "test"}, nil
				},
			},
			wantErr: false,
		},
		{
			name: "valid spec returning map",
			specs: map[string]any{
				"test": func() (map[string]string, error) {
					return map[string]string{"key": "value"}, nil
				},
			},
			wantErr: false,
		},
		{
			name: "spec with context",
			specs: map[string]any{
				"test": func(ctx context.Context) (testSpec, error) {
					return testSpec{Value: "test"}, nil
				},
			},
			wantErr: false,
		},
		{
			name: "spec returning error",
			specs: map[string]any{
				"test": func() (testSpec, error) {
					return testSpec{}, errors.New("test error")
				},
			},
			wantErr: true,
		},
		{
			name:    "no specs",
			specs:   map[string]any{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New("test-plugin")
			for name, fn := range tt.specs {
				p.specs[name] = &SpecCollector{function: fn, name: name}
			}

			ctx := context.Background()
			data, err := p.CollectSpecs(ctx)

			if tt.wantErr {
				if err == nil {
					t.Error("CollectSpecs() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("CollectSpecs() unexpected error: %v", err)
				return
			}

			if data == nil {
				t.Error("CollectSpecs() returned nil data")
			}
		})
	}
}
