package inventory

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jackadi-io/jackadi/internal/agent"
)

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
