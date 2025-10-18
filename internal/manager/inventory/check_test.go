package inventory

import (
	"testing"

	"github.com/jackadi-io/jackadi/internal/agent"
)

func TestIsCandidate(t *testing.T) {
	tests := []struct {
		name       string
		candidates []AgentIdentity
		agent      AgentIdentity
		wantIndex  int
		wantFound  bool
	}{
		{
			name: "agent is candidate",
			candidates: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
				{ID: "agent2", Address: "addr2", Certificate: "cert2"},
			},
			agent:     AgentIdentity{ID: "agent2", Address: "addr2", Certificate: "cert2"},
			wantIndex: 1,
			wantFound: true,
		},
		{
			name: "agent not in candidates",
			candidates: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agent:     AgentIdentity{ID: "agent2", Address: "addr2", Certificate: "cert2"},
			wantIndex: -1,
			wantFound: false,
		},
		{
			name:       "empty candidates list",
			candidates: []AgentIdentity{},
			agent:      AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			wantIndex:  -1,
			wantFound:  false,
		},
		{
			name: "first agent in list",
			candidates: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
				{ID: "agent2", Address: "addr2", Certificate: "cert2"},
			},
			agent:     AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			wantIndex: 0,
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New()
			a.DisableRegistryFile()
			a.registry.candidates = tt.candidates

			gotIndex, gotFound := a.isCandidate(tt.agent)

			if gotIndex != tt.wantIndex {
				t.Errorf("isCandidate() index = %v, want %v", gotIndex, tt.wantIndex)
			}
			if gotFound != tt.wantFound {
				t.Errorf("isCandidate() found = %v, want %v", gotFound, tt.wantFound)
			}
		})
	}
}

func TestIsRejected(t *testing.T) {
	tests := []struct {
		name      string
		rejected  []AgentIdentity
		agent     AgentIdentity
		wantIndex int
		wantFound bool
	}{
		{
			name: "agent is rejected",
			rejected: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
				{ID: "agent2", Address: "addr2", Certificate: "cert2"},
			},
			agent:     AgentIdentity{ID: "agent2", Address: "addr2", Certificate: "cert2"},
			wantIndex: 1,
			wantFound: true,
		},
		{
			name: "agent not in rejected list",
			rejected: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agent:     AgentIdentity{ID: "agent2", Address: "addr2", Certificate: "cert2"},
			wantIndex: -1,
			wantFound: false,
		},
		{
			name:      "empty rejected list",
			rejected:  []AgentIdentity{},
			agent:     AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			wantIndex: -1,
			wantFound: false,
		},
		{
			name: "first agent in list",
			rejected: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
				{ID: "agent2", Address: "addr2", Certificate: "cert2"},
			},
			agent:     AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			wantIndex: 0,
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New()
			a.DisableRegistryFile()
			a.registry.Rejected = tt.rejected

			gotIndex, gotFound := a.isRejected(tt.agent)

			if gotIndex != tt.wantIndex {
				t.Errorf("isRejected() index = %v, want %v", gotIndex, tt.wantIndex)
			}
			if gotFound != tt.wantFound {
				t.Errorf("isRejected() found = %v, want %v", gotFound, tt.wantFound)
			}
		})
	}
}

func TestIsRegistered(t *testing.T) {
	tests := []struct {
		name       string
		accepted   map[agent.ID]AgentIdentity
		agent      AgentIdentity
		wantResult bool
	}{
		{
			name: "agent is registered with matching details",
			accepted: map[agent.ID]AgentIdentity{
				"agent1": {ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agent:      AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			wantResult: true,
		},
		{
			name: "agent ID exists but details differ",
			accepted: map[agent.ID]AgentIdentity{
				"agent1": {ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agent:      AgentIdentity{ID: "agent1", Address: "different", Certificate: "cert1"},
			wantResult: false,
		},
		{
			name:       "agent not registered",
			accepted:   map[agent.ID]AgentIdentity{},
			agent:      AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			wantResult: false,
		},
		{
			name: "agent registered among multiple agents",
			accepted: map[agent.ID]AgentIdentity{
				"agent1": {ID: "agent1", Address: "addr1", Certificate: "cert1"},
				"agent2": {ID: "agent2", Address: "addr2", Certificate: "cert2"},
			},
			agent:      AgentIdentity{ID: "agent2", Address: "addr2", Certificate: "cert2"},
			wantResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New()
			a.DisableRegistryFile()
			a.registry.Accepted = tt.accepted

			got := a.isRegistered(tt.agent)

			if got != tt.wantResult {
				t.Errorf("isRegistered() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}

func TestIsRegisteredPublic(t *testing.T) {
	tests := []struct {
		name       string
		accepted   map[agent.ID]AgentIdentity
		agent      AgentIdentity
		wantResult bool
	}{
		{
			name: "agent is registered",
			accepted: map[agent.ID]AgentIdentity{
				"agent1": {ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agent:      AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			wantResult: true,
		},
		{
			name:       "agent not registered",
			accepted:   map[agent.ID]AgentIdentity{},
			agent:      AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New()
			a.DisableRegistryFile()
			a.registry.Accepted = tt.accepted

			got := a.IsRegistered(tt.agent)

			if got != tt.wantResult {
				t.Errorf("IsRegistered() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}
