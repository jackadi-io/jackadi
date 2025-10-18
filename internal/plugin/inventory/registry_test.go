package inventory

import (
	"context"
	"testing"

	"github.com/jackadi-io/jackadi/internal/plugin/core"
	"github.com/jackadi-io/jackadi/internal/proto"
)

type mockPlugin struct {
	name string
	err  error
}

func (m *mockPlugin) Name() (string, error) {
	return m.name, m.err
}

func (m *mockPlugin) Tasks() ([]string, error) {
	return nil, nil
}

func (m *mockPlugin) Help(task string) (map[string]string, error) {
	return nil, nil //nolint: nilnil // test
}

func (m *mockPlugin) Version() (core.Version, error) {
	return core.Version{}, nil
}

func (m *mockPlugin) Do(ctx context.Context, task string, input *proto.Input) (core.Response, error) {
	return core.Response{}, nil
}

func (m *mockPlugin) CollectSpecs(ctx context.Context) ([]byte, error) {
	return nil, nil
}

func (m *mockPlugin) GetTaskLockMode(task string) (proto.LockMode, error) {
	return proto.LockMode_NO_LOCK, nil
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name       string
		plugin     core.Plugin
		existing   map[string]core.Plugin
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:     "register new plugin",
			plugin:   &mockPlugin{name: "test-plugin"},
			existing: map[string]core.Plugin{},
			wantErr:  false,
		},
		{
			name:   "register duplicate plugin",
			plugin: &mockPlugin{name: "test-plugin"},
			existing: map[string]core.Plugin{
				"test-plugin": &mockPlugin{name: "test-plugin"},
			},
			wantErr:    true,
			wantErrMsg: "test-plugin already exists",
		},
		{
			name:     "register multiple plugins",
			plugin:   &mockPlugin{name: "plugin2"},
			existing: map[string]core.Plugin{"plugin1": &mockPlugin{name: "plugin1"}},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()
			r.plugins = tt.existing

			err := r.Register(tt.plugin)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Register() expected error, got nil")
					return
				}
				if tt.wantErrMsg != "" && err.Error() != tt.wantErrMsg {
					t.Errorf("Register() error = %v, want %v", err.Error(), tt.wantErrMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("Register() unexpected error: %v", err)
				return
			}

			name, _ := tt.plugin.Name()
			if _, ok := r.plugins[name]; !ok {
				t.Errorf("Register() plugin not in registry")
			}
		})
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		name       string
		pluginName string
		plugins    map[string]core.Plugin
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:       "get existing plugin",
			pluginName: "test-plugin",
			plugins: map[string]core.Plugin{
				"test-plugin": &mockPlugin{name: "test-plugin"},
			},
			wantErr: false,
		},
		{
			name:       "get non-existing plugin",
			pluginName: "missing-plugin",
			plugins:    map[string]core.Plugin{},
			wantErr:    true,
			wantErrMsg: "'missing-plugin' not registered",
		},
		{
			name:       "get from multiple plugins",
			pluginName: "plugin2",
			plugins: map[string]core.Plugin{
				"plugin1": &mockPlugin{name: "plugin1"},
				"plugin2": &mockPlugin{name: "plugin2"},
				"plugin3": &mockPlugin{name: "plugin3"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()
			r.plugins = tt.plugins

			plugin, err := r.Get(tt.pluginName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Get() expected error, got nil")
					return
				}
				if tt.wantErrMsg != "" && err.Error() != tt.wantErrMsg {
					t.Errorf("Get() error = %v, want %v", err.Error(), tt.wantErrMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("Get() unexpected error: %v", err)
				return
			}

			if plugin == nil {
				t.Errorf("Get() returned nil plugin")
				return
			}

			name, _ := plugin.Name()
			if name != tt.pluginName {
				t.Errorf("Get() plugin name = %v, want %v", name, tt.pluginName)
			}
		})
	}
}

func TestUnregister(t *testing.T) {
	tests := []struct {
		name       string
		pluginName string
		plugins    map[string]core.Plugin
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:       "unregister existing plugin",
			pluginName: "test-plugin",
			plugins: map[string]core.Plugin{
				"test-plugin": &mockPlugin{name: "test-plugin"},
			},
			wantErr: false,
		},
		{
			name:       "unregister non-existing plugin",
			pluginName: "missing-plugin",
			plugins:    map[string]core.Plugin{},
			wantErr:    true,
			wantErrMsg: "missing-plugin does not exist",
		},
		{
			name:       "unregister from multiple plugins",
			pluginName: "plugin2",
			plugins: map[string]core.Plugin{
				"plugin1": &mockPlugin{name: "plugin1"},
				"plugin2": &mockPlugin{name: "plugin2"},
				"plugin3": &mockPlugin{name: "plugin3"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()
			r.plugins = tt.plugins

			err := r.Unregister(tt.pluginName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Unregister() expected error, got nil")
					return
				}
				if tt.wantErrMsg != "" && err.Error() != tt.wantErrMsg {
					t.Errorf("Unregister() error = %v, want %v", err.Error(), tt.wantErrMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("Unregister() unexpected error: %v", err)
				return
			}

			if _, ok := r.plugins[tt.pluginName]; ok {
				t.Errorf("Unregister() plugin still in registry")
			}
		})
	}
}

func TestNames(t *testing.T) {
	tests := []struct {
		name      string
		plugins   map[string]core.Plugin
		wantCount int
		wantNames []string
	}{
		{
			name:      "empty registry",
			plugins:   map[string]core.Plugin{},
			wantCount: 0,
			wantNames: []string{},
		},
		{
			name: "single plugin",
			plugins: map[string]core.Plugin{
				"plugin1": &mockPlugin{name: "plugin1"},
			},
			wantCount: 1,
			wantNames: []string{"plugin1"},
		},
		{
			name: "multiple plugins",
			plugins: map[string]core.Plugin{
				"plugin1": &mockPlugin{name: "plugin1"},
				"plugin2": &mockPlugin{name: "plugin2"},
				"plugin3": &mockPlugin{name: "plugin3"},
			},
			wantCount: 3,
			wantNames: []string{"plugin1", "plugin2", "plugin3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()
			r.plugins = tt.plugins

			names := r.Names()

			if len(names) != tt.wantCount {
				t.Errorf("Names() count = %v, want %v", len(names), tt.wantCount)
			}

			nameMap := make(map[string]bool)
			for _, name := range names {
				nameMap[name] = true
			}

			for _, wantName := range tt.wantNames {
				if !nameMap[wantName] {
					t.Errorf("Names() missing expected name: %v", wantName)
				}
			}
		})
	}
}
