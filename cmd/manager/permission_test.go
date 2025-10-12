package main_test

import (
	"testing"

	main "github.com/jackadi-io/jackadi/cmd/manager"
)

func TestPermissionMatch(t *testing.T) {
	tests := []struct {
		name            string
		permission      main.Permission
		pluginRequested string
		taskRequested   string
		want            bool
	}{
		{
			name:            "exact match",
			permission:      main.Permission{Resource: "plugin1", Action: "task1"},
			pluginRequested: "plugin1",
			taskRequested:   "task1",
			want:            true,
		},
		{
			name:            "*:* match",
			permission:      main.Permission{Resource: "*", Action: "*"},
			pluginRequested: "anyplugin",
			taskRequested:   "anytask",
			want:            true,
		},
		{
			name:            "exact:* match",
			permission:      main.Permission{Resource: "plugin1", Action: "*"},
			pluginRequested: "plugin1",
			taskRequested:   "anytask",
			want:            true,
		},
		{
			name:            "*:exact match",
			permission:      main.Permission{Resource: "*", Action: "task1"},
			pluginRequested: "anyplugin",
			taskRequested:   "task1",
			want:            true,
		},
		{
			name:            "plugin unmatch",
			permission:      main.Permission{Resource: "plugin1", Action: "task1"},
			pluginRequested: "plugin2",
			taskRequested:   "task1",
			want:            false,
		},
		{
			name:            "task unmatch",
			permission:      main.Permission{Resource: "plugin1", Action: "task1"},
			pluginRequested: "plugin1",
			taskRequested:   "task2",
			want:            false,
		},
		{
			name:            "both plugin and task unmatch",
			permission:      main.Permission{Resource: "plugin1", Action: "task1"},
			pluginRequested: "plugin2",
			taskRequested:   "task2",
			want:            false,
		},
		{
			name:            "wildcard plugin doesn't match specific task",
			permission:      main.Permission{Resource: "*", Action: "task1"},
			pluginRequested: "anyplugin",
			taskRequested:   "task2",
			want:            false,
		},
		{
			name:            "specific plugin doesn't match with wildcard task",
			permission:      main.Permission{Resource: "plugin1", Action: "*"},
			pluginRequested: "plugin2",
			taskRequested:   "anytask",
			want:            false,
		},
		{
			name:            "empty plugin requested",
			permission:      main.Permission{Resource: "plugin1", Action: "task1"},
			pluginRequested: "",
			taskRequested:   "task1",
			want:            false,
		},
		{
			name:            "empty task requested",
			permission:      main.Permission{Resource: "plugin1", Action: "task1"},
			pluginRequested: "plugin1",
			taskRequested:   "",
			want:            false,
		},
		{
			name:            "wildcard matches empty plugin",
			permission:      main.Permission{Resource: "*", Action: "task1"},
			pluginRequested: "",
			taskRequested:   "task1",
			want:            true,
		},
		{
			name:            "wildcard matches empty task",
			permission:      main.Permission{Resource: "plugin1", Action: "*"},
			pluginRequested: "plugin1",
			taskRequested:   "",
			want:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.permission.Match(tt.pluginRequested, tt.taskRequested)
			if got != tt.want {
				t.Errorf("Permission.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}
