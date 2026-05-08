package inventory

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jackadi-io/jackadi/internal/node"
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
		nd            NodeIdentity
		inventory     []NodeIdentity
		want          []NodeIdentity
		expectedError error
	}{
		{
			name:      "empty list",
			nd:        NodeIdentity{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
			inventory: []NodeIdentity{},
			want: []NodeIdentity{
				{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
			},
			expectedError: nil,
		},
		{
			name: "node is not known",
			nd:   NodeIdentity{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
			inventory: []NodeIdentity{
				{ID: node.ID("node2"), Address: "127.0.0.2", Certificate: "certificate2"},
			},
			want: []NodeIdentity{
				{ID: node.ID("node2"), Address: "127.0.0.2", Certificate: "certificate2"},
				{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
			},
		},
		{
			name: "node is already known",
			nd:   NodeIdentity{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
			inventory: []NodeIdentity{
				{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
				{ID: node.ID("node2"), Address: "127.0.0.2", Certificate: "certificate1"},
			},
			want: []NodeIdentity{
				{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
				{ID: node.ID("node2"), Address: "127.0.0.2", Certificate: "certificate1"},
			},
			expectedError: ErrNodeAlreadyCandidate,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nodes := New()
			nodes.DisableRegistryFile()
			nodes.registry.candidates = test.inventory
			err := nodes.AddCandidate(test.nd)

			if diff := cmp.Diff(nodes.registry.candidates, test.want); diff != "" {
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
		nd            NodeIdentity
		inventory     []NodeIdentity
		want          []NodeIdentity
		expectedError error
	}{
		{
			name:          "empty list",
			nd:            NodeIdentity{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
			inventory:     []NodeIdentity{},
			want:          []NodeIdentity{},
			expectedError: ErrNodeNotFound,
		},
		{
			name: "list with only one node - matching",
			nd:   NodeIdentity{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
			inventory: []NodeIdentity{
				{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
			},
			want:          []NodeIdentity{},
			expectedError: nil,
		},
		{
			name: "list with only one node - not matching",
			nd:   NodeIdentity{ID: node.ID("node2"), Address: "127.0.0.1", Certificate: "certificate1"},
			inventory: []NodeIdentity{
				{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
			},
			want: []NodeIdentity{
				{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
			},
			expectedError: ErrNodeNotFound,
		},
		{
			name: "list with multiple nodes - matching",
			nd:   NodeIdentity{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
			inventory: []NodeIdentity{
				{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
				{ID: node.ID("node1"), Address: "127.0.0.2", Certificate: "certificate1"},
				{ID: node.ID("node3"), Address: "127.0.0.3", Certificate: "certificate1"},
				{ID: node.ID("node4"), Address: "127.0.0.4", Certificate: "certificate1"},
			},
			want: []NodeIdentity{
				{ID: node.ID("node1"), Address: "127.0.0.2", Certificate: "certificate1"},
				{ID: node.ID("node3"), Address: "127.0.0.3", Certificate: "certificate1"},
				{ID: node.ID("node4"), Address: "127.0.0.4", Certificate: "certificate1"},
			},
			expectedError: nil,
		},
		{
			name: "list with multiple nodes - not matching",
			nd:   NodeIdentity{ID: node.ID("node1"), Address: "127.0.0.3", Certificate: "certificate1"},
			inventory: []NodeIdentity{
				{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
				{ID: node.ID("node1"), Address: "127.0.0.2", Certificate: "certificate1"},
				{ID: node.ID("node3"), Address: "127.0.0.3", Certificate: "certificate1"},
				{ID: node.ID("node4"), Address: "127.0.0.4", Certificate: "certificate1"},
			},
			want: []NodeIdentity{
				{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
				{ID: node.ID("node1"), Address: "127.0.0.2", Certificate: "certificate1"},
				{ID: node.ID("node3"), Address: "127.0.0.3", Certificate: "certificate1"},
				{ID: node.ID("node4"), Address: "127.0.0.4", Certificate: "certificate1"},
			},
			expectedError: ErrNodeNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nodes := New()
			nodes.DisableRegistryFile()
			nodes.registry.candidates = test.inventory
			err := nodes.RemoveCandidate(test.nd)

			if diff := cmp.Diff(nodes.registry.candidates, test.want); diff != "" {
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
		name  string
		node1 NodeIdentity
		node2 NodeIdentity
		want  []diff
	}{
		{
			name:  "same node",
			node1: NodeIdentity{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
			node2: NodeIdentity{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
			want:  []diff{},
		},
		{
			name:  "different ID",
			node1: NodeIdentity{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
			node2: NodeIdentity{ID: node.ID("node2"), Address: "127.0.0.1", Certificate: "certificate1"},
			want: []diff{
				{"ID", node.ID("node1"), node.ID("node2")},
			},
		},
		{
			name:  "different address",
			node1: NodeIdentity{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
			node2: NodeIdentity{ID: node.ID("node1"), Address: "::1", Certificate: "certificate1"},
			want: []diff{
				{"address", "127.0.0.1", "::1"},
			},
		},
		{
			name:  "different certificate",
			node1: NodeIdentity{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
			node2: NodeIdentity{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate2"},
			want: []diff{
				{"certificate", "hidden", "hidden"},
			},
		},
		{
			name:  "completely different",
			node1: NodeIdentity{ID: node.ID("node1"), Address: "127.0.0.1", Certificate: "certificate1"},
			node2: NodeIdentity{ID: node.ID("node2"), Address: "::1", Certificate: "certificate2"},
			want: []diff{
				{"ID", node.ID("node1"), node.ID("node2")},
				{"address", "127.0.0.1", "::1"},
				{"certificate", "hidden", "hidden"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diffs := Compare(test.node1, test.node2)

			if diff := cmp.Diff(diffs, test.want); diff != "" {
				t.Errorf("Mismatch for '%s' test:\n%s", test.name, diff)
			}
		})
	}
}
