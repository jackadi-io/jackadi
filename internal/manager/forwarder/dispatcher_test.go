package forwarder

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jackadi-io/jackadi/internal/agent"
	"github.com/jackadi-io/jackadi/internal/manager/inventory"
	"github.com/jackadi-io/jackadi/internal/proto"
)

func TestTargetedAgentsList(t *testing.T) {
	inv := &inventory.Agents{}
	dispatcher := NewDispatcher[string, string](inv)

	agents := []agent.ID{agent.ID("agent1"), agent.ID("agent2"), agent.ID("agent3")}
	for _, agentID := range agents {
		_ = dispatcher.RegisterAgent(agentID)
	}

	result, err := dispatcher.TargetedAgents("agent1,agent2", proto.TargetMode_LIST)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := map[string]bool{
		"agent1": true,
		"agent2": true,
	}
	if diff := cmp.Diff(result, expected); diff != "" {
		t.Errorf("Mismatch (-got +want):\n%s", diff)
	}
}

func TestTargetedAgentsGlob(t *testing.T) {
	inv := &inventory.Agents{}
	dispatcher := NewDispatcher[string, string](inv)

	agents := []agent.ID{agent.ID("web-1"), agent.ID("web-2"), agent.ID("db-1"), agent.ID("cache-1")}
	for _, agentID := range agents {
		_ = dispatcher.RegisterAgent(agentID)
	}

	tests := []struct {
		pattern  string
		expected map[string]bool
	}{
		{
			pattern: "web-*",
			expected: map[string]bool{
				"web-1": true,
				"web-2": true,
			},
		},
		{
			pattern: "*-1",
			expected: map[string]bool{
				"web-1":   true,
				"db-1":    true,
				"cache-1": true,
			},
		},
		{
			pattern: "db-*",
			expected: map[string]bool{
				"db-1": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			result, err := dispatcher.TargetedAgents(tt.pattern, proto.TargetMode_GLOB)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if diff := cmp.Diff(result, tt.expected); diff != "" {
				t.Errorf("Mismatch for pattern %q (-got +want):\n%s", tt.pattern, diff)
			}
		})
	}
}

func TestTargetedAgentsRegex(t *testing.T) {
	inv := &inventory.Agents{}
	dispatcher := NewDispatcher[string, string](inv)

	agents := []agent.ID{agent.ID("web-01"), agent.ID("web-02"), agent.ID("db-01"), agent.ID("cache-01")}
	for _, agentID := range agents {
		_ = dispatcher.RegisterAgent(agentID)
	}

	tests := []struct {
		pattern     string
		expected    map[string]bool
		expectError bool
	}{
		{
			pattern: "web-\\d+",
			expected: map[string]bool{
				"web-01": true,
				"web-02": true,
			},
		},
		{
			pattern: ".*-01",
			expected: map[string]bool{
				"web-01":   true,
				"db-01":    true,
				"cache-01": true,
			},
		},
		{
			pattern: "db-.*",
			expected: map[string]bool{
				"db-01": true,
			},
		},
		{
			pattern:     "[invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			result, err := dispatcher.TargetedAgents(tt.pattern, proto.TargetMode_REGEX)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error for invalid regex")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if diff := cmp.Diff(result, tt.expected); diff != "" {
				t.Errorf("Mismatch for pattern %q (-got +want):\n%s", tt.pattern, diff)
			}
		})
	}
}

func TestTargetedAgentsQuery(t *testing.T) {
	inv := inventory.New()
	inv.DisableRegistryFile()
	dispatcher := NewDispatcher[string, string](&inv)

	agents := []agent.ID{agent.ID("web-1"), agent.ID("web-2"), agent.ID("db-1")}
	for _, agentID := range agents {
		_ = dispatcher.RegisterAgent(agentID)
	}

	// Set up agent states with specs in the inventory
	for _, agentID := range agents {
		inv.MarkAgentStateChange(agentID, true)
	}

	// Set specs for each agent
	specs1 := map[string]any{
		"os":   "linux",
		"role": "webserver",
		"system": map[string]any{
			"cpu":    4,
			"memory": 8,
		},
	}
	specs2 := map[string]any{
		"os":   "linux",
		"role": "webserver",
		"system": map[string]any{
			"cpu":    8,
			"memory": 16,
		},
	}
	specs3 := map[string]any{
		"os":   "linux",
		"role": "database",
		"system": map[string]any{
			"cpu":    8,
			"memory": 32,
		},
	}

	_ = inv.SetSpec(agent.ID("web-1"), specs1)
	_ = inv.SetSpec(agent.ID("web-2"), specs2)
	_ = inv.SetSpec(agent.ID("db-1"), specs3)

	tests := []struct {
		name        string
		query       string
		expected    map[string]bool
		expectError bool
	}{
		{
			name:  "specs exact match",
			query: "specs.os==linux",
			expected: map[string]bool{
				"web-1": true,
				"web-2": true,
				"db-1":  true,
			},
		},
		{
			name:  "specs glob match",
			query: "specs.role=~web*",
			expected: map[string]bool{
				"web-1": true,
				"web-2": true,
			},
		},
		{
			name:  "specs regex match",
			query: "specs.role=~/web.*/",
			expected: map[string]bool{
				"web-1": true,
				"web-2": true,
			},
		},
		{
			name:  "AND condition",
			query: "specs.os==linux and specs.role==webserver",
			expected: map[string]bool{
				"web-1": true,
				"web-2": true,
			},
		},
		{
			name:  "OR condition",
			query: "specs.role==webserver or specs.role==database",
			expected: map[string]bool{
				"web-1": true,
				"web-2": true,
				"db-1":  true,
			},
		},
		{
			name:        "empty query",
			query:       "",
			expectError: true,
		},
		{
			name:        "invalid operator",
			query:       "hostname!=web-1",
			expectError: true,
		},
		{
			name:        "invalid field",
			query:       "invalid.field==value",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := dispatcher.TargetedAgents(tt.query, proto.TargetMode_QUERY)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if diff := cmp.Diff(result, tt.expected); diff != "" {
				t.Errorf("Mismatch for query %q (-got +want):\n%s", tt.query, diff)
			}
		})
	}
}

func TestTargetedAgentsUnknownMode(t *testing.T) {
	inv := &inventory.Agents{}
	dispatcher := NewDispatcher[string, string](inv)

	_, err := dispatcher.TargetedAgents("test", proto.TargetMode_UNKNOWN)
	if err == nil {
		t.Error("Expected error for unknown target mode")
	}
}
