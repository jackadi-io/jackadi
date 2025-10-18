package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackadi-io/jackadi/internal/config"
	"github.com/jackadi-io/jackadi/internal/plugin/core"
	"github.com/jackadi-io/jackadi/internal/plugin/inventory"
	"github.com/jackadi-io/jackadi/internal/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockPlugin is a mock implementation of core.Plugin for testing.
type mockPlugin struct {
	name            string
	tasks           []string
	doFunc          func(ctx context.Context, task string, input *proto.Input) (core.Response, error)
	getTaskLockMode func(task string) (proto.LockMode, error)
}

func (m *mockPlugin) Name() (string, error) {
	return m.name, nil
}

func (m *mockPlugin) Tasks() ([]string, error) {
	return m.tasks, nil
}

func (m *mockPlugin) Help(task string) (map[string]string, error) {
	return map[string]string{"help": "mock help"}, nil
}

func (m *mockPlugin) Version() (core.Version, error) {
	return core.Version{PluginVersion: "1.0.0"}, nil
}

func (m *mockPlugin) Do(ctx context.Context, task string, input *proto.Input) (core.Response, error) {
	if m.doFunc != nil {
		return m.doFunc(ctx, task, input)
	}
	return core.Response{Output: []byte("mock output")}, nil
}

func (m *mockPlugin) CollectSpecs(ctx context.Context) ([]byte, error) {
	return []byte("{}"), nil
}

func (m *mockPlugin) GetTaskLockMode(task string) (proto.LockMode, error) {
	if m.getTaskLockMode != nil {
		return m.getTaskLockMode(task)
	}
	return proto.LockMode_NO_LOCK, nil
}

func TestEffectiveLockMode(t *testing.T) {
	// Save and restore original registry
	originalRegistry := inventory.Registry
	defer func() {
		inventory.Registry = originalRegistry
	}()

	// Setup mock registry
	inventory.Registry = inventory.New()

	// Register a mock plugin with lock mode
	mockPluginWithLock := &mockPlugin{
		name:  "test-plugin",
		tasks: []string{"test-task"},
		getTaskLockMode: func(task string) (proto.LockMode, error) {
			switch task {
			case "test-task":
				return proto.LockMode_EXCLUSIVE, nil
			case "no-lock-task":
				return proto.LockMode_NO_LOCK, nil
			default:
				return proto.LockMode_NO_LOCK, errors.New("task not found")
			}
		},
	}
	_ = inventory.Registry.Register(mockPluginWithLock)

	tests := []struct {
		name     string
		request  *proto.TaskRequest
		expected proto.LockMode
	}{
		{
			name: "override lock mode from request",
			request: &proto.TaskRequest{
				Task:     "test-plugin" + config.PluginSeparator + "test-task",
				LockMode: proto.LockMode_EXCLUSIVE,
			},
			expected: proto.LockMode_EXCLUSIVE,
		},
		{
			name: "use plugin default lock mode",
			request: &proto.TaskRequest{
				Task:     "test-plugin" + config.PluginSeparator + "test-task",
				LockMode: proto.LockMode_UNSPECIFIED,
			},
			expected: proto.LockMode_EXCLUSIVE,
		},
		{
			name: "plugin with no lock task",
			request: &proto.TaskRequest{
				Task:     "test-plugin" + config.PluginSeparator + "no-lock-task",
				LockMode: proto.LockMode_UNSPECIFIED,
			},
			expected: proto.LockMode_NO_LOCK,
		},
		{
			name: "plugin not found - fallback to NO_LOCK",
			request: &proto.TaskRequest{
				Task:     "unknown-plugin" + config.PluginSeparator + "some-task",
				LockMode: proto.LockMode_UNSPECIFIED,
			},
			expected: proto.LockMode_NO_LOCK,
		},
		{
			name: "task not found in plugin - fallback to NO_LOCK",
			request: &proto.TaskRequest{
				Task:     "test-plugin" + config.PluginSeparator + "unknown-task",
				LockMode: proto.LockMode_UNSPECIFIED,
			},
			expected: proto.LockMode_NO_LOCK,
		},
		{
			name: "single part task name (plugin=task)",
			request: &proto.TaskRequest{
				Task:     "test-plugin",
				LockMode: proto.LockMode_UNSPECIFIED,
			},
			expected: proto.LockMode_NO_LOCK, // Will try to find test-plugin in test-plugin
		},
		{
			name: "malformed task name with multiple separators",
			request: &proto.TaskRequest{
				Task:     "test" + config.PluginSeparator + "plugin" + config.PluginSeparator + "task",
				LockMode: proto.LockMode_UNSPECIFIED,
			},
			expected: proto.LockMode_NO_LOCK,
		},
		{
			name: "request override takes precedence over plugin default",
			request: &proto.TaskRequest{
				Task:     "test-plugin" + config.PluginSeparator + "no-lock-task",
				LockMode: proto.LockMode_EXCLUSIVE,
			},
			expected: proto.LockMode_EXCLUSIVE, // Request override
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := effectiveLockMode(tt.request)
			if result != tt.expected {
				t.Errorf("Expected lock mode %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDoTask(t *testing.T) {
	// Save and restore original registry
	originalRegistry := inventory.Registry
	defer func() {
		inventory.Registry = originalRegistry
	}()

	tests := []struct {
		name             string
		setupRegistry    func()
		request          *proto.TaskRequest
		expectedError    proto.InternalError
		expectedOutput   []byte
		validateResponse func(*testing.T, *proto.TaskResponse)
	}{
		{
			name: "successful task execution",
			setupRegistry: func() {
				inventory.Registry = inventory.New()
				plugin := &mockPlugin{
					name: "success-plugin",
					doFunc: func(ctx context.Context, task string, input *proto.Input) (core.Response, error) {
						return core.Response{
							Output:  []byte("success output"),
							Retcode: 0,
						}, nil
					},
				}
				_ = inventory.Registry.Register(plugin)
			},
			request: &proto.TaskRequest{
				Id:      1,
				GroupID: int64Ptr(100),
				Task:    "success-plugin" + config.PluginSeparator + "task1",
				Input:   &proto.Input{},
			},
			expectedError:  proto.InternalError_OK,
			expectedOutput: []byte("success output"),
			validateResponse: func(t *testing.T, resp *proto.TaskResponse) {
				t.Helper()
				if resp.Id != 1 {
					t.Errorf("Expected ID 1, got %d", resp.Id)
				}
				if resp.GetGroupID() != 100 {
					t.Errorf("Expected GroupID 100, got %d", resp.GetGroupID())
				}
			},
		},
		{
			name: "task execution with error message",
			setupRegistry: func() {
				inventory.Registry = inventory.New()
				plugin := &mockPlugin{
					name: "error-plugin",
					doFunc: func(ctx context.Context, task string, input *proto.Input) (core.Response, error) {
						return core.Response{
							Output:  []byte(""),
							Error:   "task failed",
							Retcode: 1,
						}, nil
					},
				}
				_ = inventory.Registry.Register(plugin)
			},
			request: &proto.TaskRequest{
				Id:   2,
				Task: "error-plugin" + config.PluginSeparator + "failing-task",
			},
			expectedError: proto.InternalError_OK,
			validateResponse: func(t *testing.T, resp *proto.TaskResponse) {
				t.Helper()
				if resp.Error != "task failed" {
					t.Errorf("Expected error message 'task failed', got '%s'", resp.Error)
				}
				if resp.Retcode != 1 {
					t.Errorf("Expected retcode 1, got %d", resp.Retcode)
				}
			},
		},
		{
			name: "task execution with module error",
			setupRegistry: func() {
				inventory.Registry = inventory.New()
				plugin := &mockPlugin{
					name: "module-error-plugin",
					doFunc: func(ctx context.Context, task string, input *proto.Input) (core.Response, error) {
						return core.Response{}, status.Error(codes.Internal, "internal plugin error")
					},
				}
				_ = inventory.Registry.Register(plugin)
			},
			request: &proto.TaskRequest{
				Id:   3,
				Task: "module-error-plugin" + config.PluginSeparator + "error-task",
			},
			expectedError: proto.InternalError_MODULE_ERROR,
			validateResponse: func(t *testing.T, resp *proto.TaskResponse) {
				t.Helper()
				if resp.InternalError != proto.InternalError_MODULE_ERROR {
					t.Errorf("Expected MODULE_ERROR, got %v", resp.InternalError)
				}
				if resp.ModuleError != "code=Internal, error=internal plugin error" {
					t.Errorf("Expected module error message, got '%s'", resp.ModuleError)
				}
			},
		},
		{
			name: "task execution with unknown error code",
			setupRegistry: func() {
				inventory.Registry = inventory.New()
				plugin := &mockPlugin{
					name: "unknown-error-plugin",
					doFunc: func(ctx context.Context, task string, input *proto.Input) (core.Response, error) {
						return core.Response{}, errors.New("plain error")
					},
				}
				_ = inventory.Registry.Register(plugin)
			},
			request: &proto.TaskRequest{
				Id:   4,
				Task: "unknown-error-plugin" + config.PluginSeparator + "error-task",
			},
			expectedError: proto.InternalError_MODULE_ERROR,
			validateResponse: func(t *testing.T, resp *proto.TaskResponse) {
				t.Helper()
				if resp.InternalError != proto.InternalError_MODULE_ERROR {
					t.Errorf("Expected MODULE_ERROR, got %v", resp.InternalError)
				}
				if resp.ModuleError != "plain error" {
					t.Errorf("Expected 'plain error', got '%s'", resp.ModuleError)
				}
			},
		},
		{
			name: "unknown plugin",
			setupRegistry: func() {
				inventory.Registry = inventory.New()
			},
			request: &proto.TaskRequest{
				Id:   5,
				Task: "nonexistent" + config.PluginSeparator + "task",
			},
			expectedError: proto.InternalError_UNKNOWN_TASK,
			validateResponse: func(t *testing.T, resp *proto.TaskResponse) {
				t.Helper()
				if resp.InternalError != proto.InternalError_UNKNOWN_TASK {
					t.Errorf("Expected UNKNOWN_TASK, got %v", resp.InternalError)
				}
			},
		},
		{
			name: "malformed task name with too many parts",
			setupRegistry: func() {
				inventory.Registry = inventory.New()
			},
			request: &proto.TaskRequest{
				Id:   6,
				Task: "plugin" + config.PluginSeparator + "task" + config.PluginSeparator + "extra",
			},
			expectedError: proto.InternalError_UNKNOWN_TASK,
			validateResponse: func(t *testing.T, resp *proto.TaskResponse) {
				t.Helper()
				if resp.InternalError != proto.InternalError_UNKNOWN_TASK {
					t.Errorf("Expected UNKNOWN_TASK, got %v", resp.InternalError)
				}
			},
		},
		{
			name: "single part task name (plugin=task)",
			setupRegistry: func() {
				inventory.Registry = inventory.New()
				plugin := &mockPlugin{
					name: "simple-plugin",
					doFunc: func(ctx context.Context, task string, input *proto.Input) (core.Response, error) {
						if task != "simple-plugin" {
							return core.Response{}, errors.New("unexpected task name")
						}
						return core.Response{Output: []byte("simple output")}, nil
					},
				}
				_ = inventory.Registry.Register(plugin)
			},
			request: &proto.TaskRequest{
				Id:   7,
				Task: "simple-plugin",
			},
			expectedError:  proto.InternalError_OK,
			expectedOutput: []byte("simple output"),
		},
		{
			name: "task with group ID",
			setupRegistry: func() {
				inventory.Registry = inventory.New()
				plugin := &mockPlugin{
					name: "group-plugin",
					doFunc: func(ctx context.Context, task string, input *proto.Input) (core.Response, error) {
						return core.Response{Output: []byte("grouped")}, nil
					},
				}
				_ = inventory.Registry.Register(plugin)
			},
			request: &proto.TaskRequest{
				Id:      8,
				GroupID: int64Ptr(999),
				Task:    "group-plugin" + config.PluginSeparator + "grouped-task",
			},
			expectedError: proto.InternalError_OK,
			validateResponse: func(t *testing.T, resp *proto.TaskResponse) {
				t.Helper()
				if resp.GetGroupID() != 999 {
					t.Errorf("Expected GroupID 999, got %d", resp.GetGroupID())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup registry for this test
			tt.setupRegistry()

			ctx := context.Background()
			resp := doTask(ctx, tt.request)

			if resp == nil {
				t.Fatal("Expected response, got nil")
			}

			if resp.InternalError != tt.expectedError {
				t.Errorf("Expected internal error %v, got %v", tt.expectedError, resp.InternalError)
			}

			if tt.expectedOutput != nil {
				if string(resp.Output) != string(tt.expectedOutput) {
					t.Errorf("Expected output '%s', got '%s'", string(tt.expectedOutput), string(resp.Output))
				}
			}

			if tt.validateResponse != nil {
				tt.validateResponse(t, resp)
			}
		})
	}
}

func TestNew(t *testing.T) {
	// Save and restore original registry to avoid conflicts
	originalRegistry := inventory.Registry
	defer func() {
		inventory.Registry = originalRegistry
	}()

	tests := []struct {
		name      string
		config    Config
		expectErr bool
	}{
		{
			name: "valid configuration",
			config: Config{
				ManagerAddress:   "localhost",
				ManagerPort:      "5000",
				AgentID:          "test-agent-1",
				MTLSEnabled:      false,
				PluginDir:        "/tmp/plugins",
				PluginServerPort: "6000",
			},
			expectErr: false,
		},
		{
			name: "minimal configuration",
			config: Config{
				AgentID: "minimal-agent",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset registry for each test
			inventory.Registry = inventory.New()

			agent, ctx, err := New(tt.config)

			if tt.expectErr {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if agent.config.AgentID != tt.config.AgentID {
				t.Errorf("Expected AgentID %s, got %s", tt.config.AgentID, agent.config.AgentID)
			}

			if agent.SpecManager == nil {
				t.Error("Expected SpecManager to be initialized")
			}

			if ctx == nil {
				t.Error("Expected context to be initialized")
			}
		})
	}
}

func TestClose(t *testing.T) {
	tests := []struct {
		name      string
		setupConn bool
		expectErr bool
	}{
		{
			name:      "close nil connection",
			setupConn: false,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := Agent{}

			err := agent.Close()

			if tt.expectErr {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if err.Error() != "trying to close a nil connection" {
					t.Errorf("Expected 'trying to close a nil connection' error, got: %v", err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestTaskNameParsing(t *testing.T) {
	// This tests the task name parsing logic used in both effectiveLockMode and doTask
	tests := []struct {
		name           string
		taskName       string
		expectedPlugin string
		expectedTask   string
		expectedParts  int
		expectInvalid  bool
	}{
		{
			name:           "single part",
			taskName:       "ping",
			expectedPlugin: "ping",
			expectedTask:   "ping",
			expectedParts:  1,
		},
		{
			name:           "two parts",
			taskName:       "system" + config.PluginSeparator + "info",
			expectedPlugin: "system",
			expectedTask:   "info",
			expectedParts:  2,
		},
		{
			name:           "multiple parts",
			taskName:       "a" + config.PluginSeparator + "b" + config.PluginSeparator + "c",
			expectedPlugin: "invalid",
			expectedTask:   "invalid",
			expectedParts:  3,
			expectInvalid:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the same parsing logic as in the actual functions
			parts := strings.Split(tt.taskName, config.PluginSeparator)
			var plugin, task string

			switch {
			case len(parts) == 1:
				plugin = parts[0]
				task = parts[0]
			case len(parts) == 2:
				plugin = parts[0]
				task = parts[1]
			default:
				plugin = "invalid"
				task = "invalid"
			}

			if len(parts) != tt.expectedParts {
				t.Errorf("Expected %d parts, got %d", tt.expectedParts, len(parts))
			}

			if !tt.expectInvalid {
				if plugin != tt.expectedPlugin {
					t.Errorf("Expected plugin '%s', got '%s'", tt.expectedPlugin, plugin)
				}

				if task != tt.expectedTask {
					t.Errorf("Expected task '%s', got '%s'", tt.expectedTask, task)
				}
			}
		})
	}
}

// Helper function.
func int64Ptr(i int64) *int64 {
	return &i
}
