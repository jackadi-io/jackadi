package inventory

import (
	"testing"

	"github.com/jackadi-io/jackadi/internal/agent"
)

func TestGetMatchingAccepted(t *testing.T) {
	addr1 := "addr1"
	addr2 := "addr2"
	cert1 := "cert1"
	cert2 := "cert2"

	tests := []struct {
		name        string
		accepted    map[agent.ID]AgentIdentity
		agentID     agent.ID
		address     *string
		certificate *string
		wantCount   int
		wantAgents  []AgentIdentity
	}{
		{
			name: "match by ID only",
			accepted: map[agent.ID]AgentIdentity{
				"agent1": {ID: "agent1", Address: "addr1", Certificate: "cert1"},
				"agent2": {ID: "agent2", Address: "addr2", Certificate: "cert2"},
			},
			agentID:   "agent1",
			wantCount: 1,
			wantAgents: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
		},
		{
			name: "match by ID and address",
			accepted: map[agent.ID]AgentIdentity{
				"agent1": {ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agentID:   "agent1",
			address:   &addr1,
			wantCount: 1,
			wantAgents: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
		},
		{
			name: "match by ID and certificate",
			accepted: map[agent.ID]AgentIdentity{
				"agent1": {ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agentID:     "agent1",
			certificate: &cert1,
			wantCount:   1,
			wantAgents: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
		},
		{
			name: "match all fields",
			accepted: map[agent.ID]AgentIdentity{
				"agent1": {ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agentID:     "agent1",
			address:     &addr1,
			certificate: &cert1,
			wantCount:   1,
			wantAgents: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
		},
		{
			name: "no match wrong ID",
			accepted: map[agent.ID]AgentIdentity{
				"agent1": {ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agentID:    "agent2",
			wantCount:  0,
			wantAgents: []AgentIdentity{},
		},
		{
			name: "no match wrong address",
			accepted: map[agent.ID]AgentIdentity{
				"agent1": {ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agentID:    "agent1",
			address:    &addr2,
			wantCount:  0,
			wantAgents: []AgentIdentity{},
		},
		{
			name: "no match wrong certificate",
			accepted: map[agent.ID]AgentIdentity{
				"agent1": {ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agentID:     "agent1",
			certificate: &cert2,
			wantCount:   0,
			wantAgents:  []AgentIdentity{},
		},
		{
			name:       "empty accepted list",
			accepted:   map[agent.ID]AgentIdentity{},
			agentID:    "agent1",
			wantCount:  0,
			wantAgents: []AgentIdentity{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New()
			a.DisableRegistryFile()
			a.registry.Accepted = tt.accepted

			got := a.GetMatchingAccepted(tt.agentID, tt.address, tt.certificate)

			if len(got) != tt.wantCount {
				t.Errorf("GetMatchingAccepted() returned %d agents, want %d", len(got), tt.wantCount)
			}

			for i, want := range tt.wantAgents {
				if i >= len(got) {
					t.Errorf("missing agent at index %d", i)
					continue
				}
				if got[i] != want {
					t.Errorf("GetMatchingAccepted()[%d] = %v, want %v", i, got[i], want)
				}
			}
		})
	}
}

func TestGetMatchingCandidates(t *testing.T) {
	addr1 := "addr1"
	addr2 := "addr2"
	cert1 := "cert1"
	cert2 := "cert2"

	tests := []struct {
		name        string
		candidates  []AgentIdentity
		agentID     agent.ID
		address     *string
		certificate *string
		wantCount   int
		wantAgents  []AgentIdentity
	}{
		{
			name: "match by ID only",
			candidates: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
				{ID: "agent2", Address: "addr2", Certificate: "cert2"},
			},
			agentID:   "agent1",
			wantCount: 1,
			wantAgents: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
		},
		{
			name: "match by ID and address",
			candidates: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agentID:   "agent1",
			address:   &addr1,
			wantCount: 1,
			wantAgents: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
		},
		{
			name: "match by ID and certificate",
			candidates: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agentID:     "agent1",
			certificate: &cert1,
			wantCount:   1,
			wantAgents: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
		},
		{
			name: "match all fields",
			candidates: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agentID:     "agent1",
			address:     &addr1,
			certificate: &cert1,
			wantCount:   1,
			wantAgents: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
		},
		{
			name: "no match wrong ID",
			candidates: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agentID:    "agent2",
			wantCount:  0,
			wantAgents: []AgentIdentity{},
		},
		{
			name: "no match wrong address",
			candidates: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agentID:    "agent1",
			address:    &addr2,
			wantCount:  0,
			wantAgents: []AgentIdentity{},
		},
		{
			name: "no match wrong certificate",
			candidates: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agentID:     "agent1",
			certificate: &cert2,
			wantCount:   0,
			wantAgents:  []AgentIdentity{},
		},
		{
			name:       "empty candidates list",
			candidates: []AgentIdentity{},
			agentID:    "agent1",
			wantCount:  0,
			wantAgents: []AgentIdentity{},
		},
		{
			name: "multiple matches same ID",
			candidates: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
				{ID: "agent1", Address: "addr2", Certificate: "cert2"},
			},
			agentID:   "agent1",
			wantCount: 2,
			wantAgents: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
				{ID: "agent1", Address: "addr2", Certificate: "cert2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New()
			a.DisableRegistryFile()
			a.registry.candidates = tt.candidates

			got := a.GetMatchingCandidates(tt.agentID, tt.address, tt.certificate)

			if len(got) != tt.wantCount {
				t.Errorf("GetMatchingCandidates() returned %d agents, want %d", len(got), tt.wantCount)
			}

			for i, want := range tt.wantAgents {
				if i >= len(got) {
					t.Errorf("missing agent at index %d", i)
					continue
				}
				if got[i] != want {
					t.Errorf("GetMatchingCandidates()[%d] = %v, want %v", i, got[i], want)
				}
			}
		})
	}
}

func TestGetMatchingRejected(t *testing.T) {
	addr1 := "addr1"
	addr2 := "addr2"
	cert1 := "cert1"
	cert2 := "cert2"

	tests := []struct {
		name        string
		rejected    []AgentIdentity
		agentID     agent.ID
		address     *string
		certificate *string
		wantCount   int
		wantAgents  []AgentIdentity
	}{
		{
			name: "match by ID only",
			rejected: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
				{ID: "agent2", Address: "addr2", Certificate: "cert2"},
			},
			agentID:   "agent1",
			wantCount: 1,
			wantAgents: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
		},
		{
			name: "match by ID and address",
			rejected: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agentID:   "agent1",
			address:   &addr1,
			wantCount: 1,
			wantAgents: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
		},
		{
			name: "match by ID and certificate",
			rejected: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agentID:     "agent1",
			certificate: &cert1,
			wantCount:   1,
			wantAgents: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
		},
		{
			name: "match all fields",
			rejected: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agentID:     "agent1",
			address:     &addr1,
			certificate: &cert1,
			wantCount:   1,
			wantAgents: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
		},
		{
			name: "no match wrong ID",
			rejected: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agentID:    "agent2",
			wantCount:  0,
			wantAgents: []AgentIdentity{},
		},
		{
			name: "no match wrong address",
			rejected: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agentID:    "agent1",
			address:    &addr2,
			wantCount:  0,
			wantAgents: []AgentIdentity{},
		},
		{
			name: "no match wrong certificate",
			rejected: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			agentID:     "agent1",
			certificate: &cert2,
			wantCount:   0,
			wantAgents:  []AgentIdentity{},
		},
		{
			name:       "empty rejected list",
			rejected:   []AgentIdentity{},
			agentID:    "agent1",
			wantCount:  0,
			wantAgents: []AgentIdentity{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New()
			a.DisableRegistryFile()
			a.registry.Rejected = tt.rejected

			got := a.GetMatchingRejected(tt.agentID, tt.address, tt.certificate)

			if len(got) != tt.wantCount {
				t.Errorf("GetMatchingRejected() returned %d agents, want %d", len(got), tt.wantCount)
			}

			for i, want := range tt.wantAgents {
				if i >= len(got) {
					t.Errorf("missing agent at index %d", i)
					continue
				}
				if got[i] != want {
					t.Errorf("GetMatchingRejected()[%d] = %v, want %v", i, got[i], want)
				}
			}
		})
	}
}
