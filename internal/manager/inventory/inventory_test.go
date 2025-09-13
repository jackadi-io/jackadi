package inventory

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jackadi-io/jackadi/internal/agent"
)

// TODO:
// - TestAccept
// - TestGetMatchingXXX
// - TestRegister
// - TestUnregister
// - TestReject
// - TestRemove

func TestAddCandidate(t *testing.T) {
	tests := []struct {
		name          string
		agent         AgentIdentity
		inventory     []AgentIdentity
		want          []AgentIdentity
		expectedError error
	}{
		{
			name:      "empty list",
			agent:     AgentIdentity{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
			inventory: []AgentIdentity{},
			want: []AgentIdentity{
				{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
			},
			expectedError: nil,
		},
		{
			name:  "agent is not known",
			agent: AgentIdentity{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
			inventory: []AgentIdentity{
				{ID: agent.ID("agent2"), Address: "127.0.0.2", Certificate: "certificate2"},
			},
			want: []AgentIdentity{
				{ID: agent.ID("agent2"), Address: "127.0.0.2", Certificate: "certificate2"},
				{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
			},
		},
		{
			name:  "agent is already known",
			agent: AgentIdentity{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
			inventory: []AgentIdentity{
				{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
				{ID: agent.ID("agent2"), Address: "127.0.0.2", Certificate: "certificate1"},
			},
			want: []AgentIdentity{
				{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
				{ID: agent.ID("agent2"), Address: "127.0.0.2", Certificate: "certificate1"},
			},
			expectedError: ErrAgentAlreadyCandidate,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			agents := New()
			agents.DisableRegistryFile()
			agents.registry.candidates = test.inventory
			err := agents.AddCandidate(test.agent)

			if diff := cmp.Diff(agents.registry.candidates, test.want); diff != "" {
				t.Errorf("Mismatch for '%s' test:\n%s", test.name, diff)
			}
			if !errors.Is(err, test.expectedError) {
				t.Errorf("Mismatch error: wants '%s', got test:\n%s", test.expectedError, err)
			}
		})
	}
}

func TestRemoveCandidates(t *testing.T) {
	tests := []struct {
		name          string
		agent         AgentIdentity
		inventory     []AgentIdentity
		want          []AgentIdentity
		expectedError error
	}{
		{
			name:          "empty list",
			agent:         AgentIdentity{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
			inventory:     []AgentIdentity{},
			want:          []AgentIdentity{},
			expectedError: ErrAgentNotFound,
		},
		{
			name:  "list with only one agent - matching",
			agent: AgentIdentity{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
			inventory: []AgentIdentity{
				{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
			},
			want:          []AgentIdentity{},
			expectedError: nil,
		},
		{
			name:  "list with only one agent - not matching",
			agent: AgentIdentity{ID: agent.ID("agent2"), Address: "127.0.0.1", Certificate: "certificate1"},
			inventory: []AgentIdentity{
				{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
			},
			want: []AgentIdentity{
				{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
			},
			expectedError: ErrAgentNotFound,
		},
		{
			name:  "list with multiple one agent - matching",
			agent: AgentIdentity{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
			inventory: []AgentIdentity{
				{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
				{ID: agent.ID("agent1"), Address: "127.0.0.2", Certificate: "certificate1"},
				{ID: agent.ID("agent3"), Address: "127.0.0.3", Certificate: "certificate1"},
				{ID: agent.ID("agent4"), Address: "127.0.0.4", Certificate: "certificate1"},
			},
			want: []AgentIdentity{
				{ID: agent.ID("agent1"), Address: "127.0.0.2", Certificate: "certificate1"},
				{ID: agent.ID("agent3"), Address: "127.0.0.3", Certificate: "certificate1"},
				{ID: agent.ID("agent4"), Address: "127.0.0.4", Certificate: "certificate1"},
			},
			expectedError: nil,
		},
		{
			name:  "list with multiple one agent - not matching",
			agent: AgentIdentity{ID: agent.ID("agent1"), Address: "127.0.0.3", Certificate: "certificate1"},
			inventory: []AgentIdentity{
				{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
				{ID: agent.ID("agent1"), Address: "127.0.0.2", Certificate: "certificate1"},
				{ID: agent.ID("agent3"), Address: "127.0.0.3", Certificate: "certificate1"},
				{ID: agent.ID("agent4"), Address: "127.0.0.4", Certificate: "certificate1"},
			},
			want: []AgentIdentity{
				{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
				{ID: agent.ID("agent1"), Address: "127.0.0.2", Certificate: "certificate1"},
				{ID: agent.ID("agent3"), Address: "127.0.0.3", Certificate: "certificate1"},
				{ID: agent.ID("agent4"), Address: "127.0.0.4", Certificate: "certificate1"},
			},
			expectedError: ErrAgentNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			agents := New()
			agents.DisableRegistryFile()
			agents.registry.candidates = test.inventory
			err := agents.RemoveCandidate(test.agent)

			if diff := cmp.Diff(agents.registry.candidates, test.want); diff != "" {
				t.Errorf("Mismatch for '%s' test:\n%s", test.name, diff)
			}
			if !errors.Is(err, test.expectedError) {
				t.Errorf("Mismatch error: wants '%s', got test:\n%s", test.expectedError, err)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		name   string
		agent1 AgentIdentity
		agent2 AgentIdentity
		want   []diff
	}{
		{
			name:   "same agent",
			agent1: AgentIdentity{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
			agent2: AgentIdentity{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
			want:   []diff{},
		},
		{
			name:   "different ID",
			agent1: AgentIdentity{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
			agent2: AgentIdentity{ID: agent.ID("agent2"), Address: "127.0.0.1", Certificate: "certificate1"},
			want: []diff{
				{"ID", agent.ID("agent1"), agent.ID("agent2")},
			},
		},
		{
			name:   "different address",
			agent1: AgentIdentity{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
			agent2: AgentIdentity{ID: agent.ID("agent1"), Address: "::1", Certificate: "certificate1"},
			want: []diff{
				{"address", "127.0.0.1", "::1"},
			},
		},
		{
			name:   "different certificate",
			agent1: AgentIdentity{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
			agent2: AgentIdentity{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate2"},
			want: []diff{
				{"certificate", "hidden", "hidden"},
			},
		},
		{
			name:   "completely different",
			agent1: AgentIdentity{ID: agent.ID("agent1"), Address: "127.0.0.1", Certificate: "certificate1"},
			agent2: AgentIdentity{ID: agent.ID("agent2"), Address: "::1", Certificate: "certificate2"},
			want: []diff{
				{"ID", agent.ID("agent1"), agent.ID("agent2")},
				{"address", "127.0.0.1", "::1"},
				{"certificate", "hidden", "hidden"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diffs := Compare(test.agent1, test.agent2)

			if diff := cmp.Diff(diffs, test.want); diff != "" {
				t.Errorf("Mismatch for '%s' test:\n%s", test.name, diff)
			}
		})
	}
}
