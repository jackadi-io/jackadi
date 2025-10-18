package inventory

import (
	"errors"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jackadi-io/jackadi/internal/agent"
)

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

func TestRegister(t *testing.T) {
	tests := []struct {
		name          string
		agent         AgentIdentity
		candidates    []AgentIdentity
		rejected      []AgentIdentity
		accepted      map[agent.ID]AgentIdentity
		allowRejected bool
		wantErr       bool
		checkErr      func(error) bool
	}{
		{
			name:  "register candidate successfully",
			agent: AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			candidates: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			rejected:      []AgentIdentity{},
			accepted:      map[agent.ID]AgentIdentity{},
			allowRejected: false,
			wantErr:       false,
		},
		{
			name:          "register agent not in candidates or rejected",
			agent:         AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			candidates:    []AgentIdentity{},
			rejected:      []AgentIdentity{},
			accepted:      map[agent.ID]AgentIdentity{},
			allowRejected: false,
			wantErr:       true,
			checkErr:      func(err error) bool { return errors.Is(err, ErrAgentNotFound) },
		},
		{
			name:  "register already registered agent with same details",
			agent: AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			candidates: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			rejected: []AgentIdentity{},
			accepted: map[agent.ID]AgentIdentity{
				"agent1": {ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			allowRejected: false,
			wantErr:       true,
			checkErr:      func(err error) bool { return errors.Is(err, ErrAgentAlreadyRegistered) },
		},
		{
			name:  "register rogue agent with different details",
			agent: AgentIdentity{ID: "agent1", Address: "addr2", Certificate: "cert2"},
			candidates: []AgentIdentity{
				{ID: "agent1", Address: "addr2", Certificate: "cert2"},
			},
			rejected: []AgentIdentity{},
			accepted: map[agent.ID]AgentIdentity{
				"agent1": {ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			allowRejected: false,
			wantErr:       true,
			checkErr: func(err error) bool {
				var rogueErr *RogueAgentError
				return errors.As(err, &rogueErr)
			},
		},
		{
			name:  "register rejected agent without allowRejected",
			agent: AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			candidates: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			rejected: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			accepted:      map[agent.ID]AgentIdentity{},
			allowRejected: false,
			wantErr:       true,
			checkErr:      func(err error) bool { return err != nil && err.Error() == "agent is both candidate and rejected" },
		},
		{
			name:  "register rejected agent with allowRejected",
			agent: AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			candidates: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			rejected: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			accepted:      map[agent.ID]AgentIdentity{},
			allowRejected: true,
			wantErr:       true,
			checkErr:      func(err error) bool { return err != nil && err.Error() == "agent is both candidate and rejected" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agents := New()
			agents.DisableRegistryFile()
			agents.registry.candidates = tt.candidates
			agents.registry.Rejected = tt.rejected
			agents.registry.Accepted = tt.accepted

			err := agents.Register(tt.agent, tt.allowRejected)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Register() expected error, got nil")
				} else if tt.checkErr != nil && !tt.checkErr(err) {
					t.Errorf("Register() error check failed: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Register() unexpected error: %v", err)
				}
				if _, ok := agents.registry.Accepted[tt.agent.ID]; !ok {
					t.Errorf("Register() agent not in accepted list")
				}
			}
		})
	}
}

func TestUnregister(t *testing.T) {
	tests := []struct {
		name     string
		agent    AgentIdentity
		accepted map[agent.ID]AgentIdentity
		wantErr  bool
	}{
		{
			name:  "unregister existing agent",
			agent: AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			accepted: map[agent.ID]AgentIdentity{
				"agent1": {ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			wantErr: false,
		},
		{
			name:     "unregister non-existing agent",
			agent:    AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			accepted: map[agent.ID]AgentIdentity{},
			wantErr:  true,
		},
		{
			name:  "unregister agent with wrong details",
			agent: AgentIdentity{ID: "agent1", Address: "addr2", Certificate: "cert2"},
			accepted: map[agent.ID]AgentIdentity{
				"agent1": {ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agents := New()
			agents.DisableRegistryFile()
			agents.registry.Accepted = tt.accepted

			err := agents.Unregister(tt.agent)

			if tt.wantErr && err == nil {
				t.Errorf("Unregister() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unregister() unexpected error: %v", err)
			}
			if !tt.wantErr {
				if _, ok := agents.registry.Accepted[tt.agent.ID]; ok {
					t.Errorf("Unregister() agent still in accepted list")
				}
			}
		})
	}
}

func TestReject(t *testing.T) {
	tests := []struct {
		name       string
		agent      AgentIdentity
		candidates []AgentIdentity
		accepted   map[agent.ID]AgentIdentity
		wantErr    bool
	}{
		{
			name:  "reject candidate",
			agent: AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			candidates: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			accepted: map[agent.ID]AgentIdentity{},
			wantErr:  false,
		},
		{
			name:       "reject accepted agent",
			agent:      AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			candidates: []AgentIdentity{},
			accepted: map[agent.ID]AgentIdentity{
				"agent1": {ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			wantErr: false,
		},
		{
			name:  "reject candidate and accepted",
			agent: AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			candidates: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			accepted: map[agent.ID]AgentIdentity{
				"agent1": {ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agents := New()
			agents.DisableRegistryFile()
			agents.registry.candidates = tt.candidates
			agents.registry.Accepted = tt.accepted

			err := agents.Reject(tt.agent)

			if tt.wantErr && err == nil {
				t.Errorf("Reject() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Reject() unexpected error: %v", err)
			}
			if !tt.wantErr {
				found := slices.Contains(agents.registry.Rejected, tt.agent)
				if !found {
					t.Errorf("Reject() agent not in rejected list")
				}
				if _, ok := agents.registry.Accepted[tt.agent.ID]; ok {
					t.Errorf("Reject() agent still in accepted list")
				}
			}
		})
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		name       string
		agent      AgentIdentity
		candidates []AgentIdentity
		rejected   []AgentIdentity
		accepted   map[agent.ID]AgentIdentity
		wantErr    bool
	}{
		{
			name:  "remove from candidates only",
			agent: AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			candidates: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			rejected: []AgentIdentity{},
			accepted: map[agent.ID]AgentIdentity{},
			wantErr:  false,
		},
		{
			name:       "remove from rejected only",
			agent:      AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			candidates: []AgentIdentity{},
			rejected: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			accepted: map[agent.ID]AgentIdentity{},
			wantErr:  false,
		},
		{
			name:       "remove from accepted only",
			agent:      AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			candidates: []AgentIdentity{},
			rejected:   []AgentIdentity{},
			accepted: map[agent.ID]AgentIdentity{
				"agent1": {ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			wantErr: false,
		},
		{
			name:  "remove from all lists",
			agent: AgentIdentity{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			candidates: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			rejected: []AgentIdentity{
				{ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			accepted: map[agent.ID]AgentIdentity{
				"agent1": {ID: "agent1", Address: "addr1", Certificate: "cert1"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agents := New()
			agents.DisableRegistryFile()
			agents.registry.candidates = tt.candidates
			agents.registry.Rejected = tt.rejected
			agents.registry.Accepted = tt.accepted

			err := agents.Remove(tt.agent)

			if tt.wantErr && err == nil {
				t.Errorf("Remove() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Remove() unexpected error: %v", err)
			}
			if !tt.wantErr {
				if _, ok := agents.registry.Accepted[tt.agent.ID]; ok {
					t.Errorf("Remove() agent still in accepted list")
				}
				for _, candidate := range agents.registry.candidates {
					if candidate == tt.agent {
						t.Errorf("Remove() agent still in candidates list")
					}
				}
				for _, rejected := range agents.registry.Rejected {
					if rejected == tt.agent {
						t.Errorf("Remove() agent still in rejected list")
					}
				}
			}
		})
	}
}
