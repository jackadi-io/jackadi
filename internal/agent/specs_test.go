package agent

import (
	"sync"
	"testing"
)

func TestList(t *testing.T) {
	tests := []struct {
		name      string
		specs     map[string]any
		wantCount int
		wantNames []string
	}{
		{
			name:      "empty specs",
			specs:     map[string]any{},
			wantCount: 0,
			wantNames: []string{},
		},
		{
			name: "single spec",
			specs: map[string]any{
				"plugin1": map[string]string{"key": "value"},
			},
			wantCount: 1,
			wantNames: []string{"plugin1"},
		},
		{
			name: "multiple specs sorted",
			specs: map[string]any{
				"plugin3": map[string]string{"key": "value3"},
				"plugin1": map[string]string{"key": "value1"},
				"plugin2": map[string]string{"key": "value2"},
			},
			wantCount: 3,
			wantNames: []string{"plugin1", "plugin2", "plugin3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SpecsManager{
				specs: tt.specs,
				mutex: &sync.RWMutex{},
			}

			names, err := s.List()
			if err != nil {
				t.Errorf("List() unexpected error: %v", err)
				return
			}

			if len(names) != tt.wantCount {
				t.Errorf("List() count = %v, want %v", len(names), tt.wantCount)
			}

			for i, wantName := range tt.wantNames {
				if i >= len(names) || names[i] != wantName {
					t.Errorf("List()[%d] = %v, want %v", i, names[i], wantName)
				}
			}
		})
	}
}

func TestAll(t *testing.T) {
	tests := []struct {
		name  string
		specs map[string]any
	}{
		{
			name:  "empty specs",
			specs: map[string]any{},
		},
		{
			name: "single spec",
			specs: map[string]any{
				"plugin1": map[string]string{"key": "value"},
			},
		},
		{
			name: "multiple specs",
			specs: map[string]any{
				"plugin1": map[string]string{"key1": "value1"},
				"plugin2": map[string]int{"count": 42},
				"plugin3": map[string]bool{"enabled": true},
			},
		},
		{
			name: "nested specs",
			specs: map[string]any{
				"plugin1": map[string]any{
					"nested": map[string]string{"key": "value"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SpecsManager{
				specs: tt.specs,
				mutex: &sync.RWMutex{},
			}

			result, err := s.All()
			if err != nil {
				t.Errorf("All() unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.specs) {
				t.Errorf("All() count = %v, want %v", len(result), len(tt.specs))
			}

			for key := range tt.specs {
				if _, ok := result[key]; !ok {
					t.Errorf("All() missing key: %v", key)
				}
			}
		})
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		name    string
		specs   map[string]any
		path    string
		wantErr bool
	}{
		{
			name: "get simple key",
			specs: map[string]any{
				"plugin1": map[string]string{"key": "value"},
			},
			path:    "plugin1",
			wantErr: false,
		},
		{
			name: "get nested key with dot notation",
			specs: map[string]any{
				"plugin1": map[string]any{
					"nested": map[string]string{"key": "value"},
				},
			},
			path:    "plugin1.nested",
			wantErr: false,
		},
		{
			name: "get deep nested key",
			specs: map[string]any{
				"plugin1": map[string]any{
					"level1": map[string]any{
						"level2": map[string]string{"key": "value"},
					},
				},
			},
			path:    "plugin1.level1.level2",
			wantErr: false,
		},
		{
			name:    "get non-existing key",
			specs:   map[string]any{},
			path:    "nonexistent",
			wantErr: true,
		},
		{
			name: "get invalid nested path",
			specs: map[string]any{
				"plugin1": map[string]string{"key": "value"},
			},
			path:    "plugin1.nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SpecsManager{
				specs: tt.specs,
				mutex: &sync.RWMutex{},
			}

			result, err := s.Get(tt.path)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Get() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Get() unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Get() returned nil result")
			}
		})
	}
}
