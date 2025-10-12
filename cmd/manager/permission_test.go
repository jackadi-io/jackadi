package main

import (
	"testing"
)

func TestPermissionMatch(t *testing.T) {
	tests := []struct {
		name            string
		permission      Permission
		pluginRequested string
		taskRequested   string
		want            bool
	}{
		{
			name:            "exact match",
			permission:      Permission{Resource: "plugin1", Action: "task1"},
			pluginRequested: "plugin1",
			taskRequested:   "task1",
			want:            true,
		},
		{
			name:            "*:* match",
			permission:      Permission{Resource: "*", Action: "*"},
			pluginRequested: "anyplugin",
			taskRequested:   "anytask",
			want:            true,
		},
		{
			name:            "exact:* match",
			permission:      Permission{Resource: "plugin1", Action: "*"},
			pluginRequested: "plugin1",
			taskRequested:   "anytask",
			want:            true,
		},
		{
			name:            "*:exact match",
			permission:      Permission{Resource: "*", Action: "task1"},
			pluginRequested: "anyplugin",
			taskRequested:   "task1",
			want:            true,
		},
		{
			name:            "plugin unmatch",
			permission:      Permission{Resource: "plugin1", Action: "task1"},
			pluginRequested: "plugin2",
			taskRequested:   "task1",
			want:            false,
		},
		{
			name:            "task unmatch",
			permission:      Permission{Resource: "plugin1", Action: "task1"},
			pluginRequested: "plugin1",
			taskRequested:   "task2",
			want:            false,
		},
		{
			name:            "both plugin and task unmatch",
			permission:      Permission{Resource: "plugin1", Action: "task1"},
			pluginRequested: "plugin2",
			taskRequested:   "task2",
			want:            false,
		},
		{
			name:            "wildcard plugin doesn't match specific task",
			permission:      Permission{Resource: "*", Action: "task1"},
			pluginRequested: "anyplugin",
			taskRequested:   "task2",
			want:            false,
		},
		{
			name:            "specific plugin doesn't match with wildcard task",
			permission:      Permission{Resource: "plugin1", Action: "*"},
			pluginRequested: "plugin2",
			taskRequested:   "anytask",
			want:            false,
		},
		{
			name:            "empty plugin requested",
			permission:      Permission{Resource: "plugin1", Action: "task1"},
			pluginRequested: "",
			taskRequested:   "task1",
			want:            false,
		},
		{
			name:            "empty task requested",
			permission:      Permission{Resource: "plugin1", Action: "task1"},
			pluginRequested: "plugin1",
			taskRequested:   "",
			want:            false,
		},
		{
			name:            "wildcard matches empty plugin",
			permission:      Permission{Resource: "*", Action: "task1"},
			pluginRequested: "",
			taskRequested:   "task1",
			want:            true,
		},
		{
			name:            "wildcard matches empty task",
			permission:      Permission{Resource: "plugin1", Action: "*"},
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

func TestCanAccessEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		config   ParsedAuthConfig
		username string
		resource string
		action   string
		want     bool
	}{
		{
			name: "user has exact permission",
			config: ParsedAuthConfig{
				Users: map[User][]Role{
					"user1": {"role1"},
				},
				Roles: map[string]Permissions{
					"role1": {
						Endpoints: []Permission{
							{Resource: "workflow", Action: "list"},
						},
					},
				},
			},
			username: "user1",
			resource: "workflow",
			action:   "list",
			want:     true,
		},
		{
			name: "user has wildcard resource permission",
			config: ParsedAuthConfig{
				Users: map[User][]Role{
					"user1": {"role1"},
				},
				Roles: map[string]Permissions{
					"role1": {
						Endpoints: []Permission{
							{Resource: "*", Action: "list"},
						},
					},
				},
			},
			username: "user1",
			resource: "workflow",
			action:   "list",
			want:     true,
		},
		{
			name: "user has wildcard action permission",
			config: ParsedAuthConfig{
				Users: map[User][]Role{
					"user1": {"role1"},
				},
				Roles: map[string]Permissions{
					"role1": {
						Endpoints: []Permission{
							{Resource: "workflow", Action: "*"},
						},
					},
				},
			},
			username: "user1",
			resource: "workflow",
			action:   "list",
			want:     true,
		},
		{
			name: "user has full wildcard permission",
			config: ParsedAuthConfig{
				Users: map[User][]Role{
					"user1": {"role1"},
				},
				Roles: map[string]Permissions{
					"role1": {
						Endpoints: []Permission{
							{Resource: "*", Action: "*"},
						},
					},
				},
			},
			username: "user1",
			resource: "workflow",
			action:   "list",
			want:     true,
		},
		{
			name: "user not in config",
			config: ParsedAuthConfig{
				Users: map[User][]Role{
					"user1": {"role1"},
				},
				Roles: map[string]Permissions{
					"role1": {
						Endpoints: []Permission{
							{Resource: "workflow", Action: "list"},
						},
					},
				},
			},
			username: "user2",
			resource: "workflow",
			action:   "list",
			want:     false,
		},
		{
			name: "user has no matching permission",
			config: ParsedAuthConfig{
				Users: map[User][]Role{
					"user1": {"role1"},
				},
				Roles: map[string]Permissions{
					"role1": {
						Endpoints: []Permission{
							{Resource: "workflow", Action: "list"},
						},
					},
				},
			},
			username: "user1",
			resource: "workflow",
			action:   "create",
			want:     false,
		},
		{
			name: "user has multiple roles with matching permission",
			config: ParsedAuthConfig{
				Users: map[User][]Role{
					"user1": {"role1", "role2"},
				},
				Roles: map[string]Permissions{
					"role1": {
						Endpoints: []Permission{
							{Resource: "workflow", Action: "list"},
						},
					},
					"role2": {
						Endpoints: []Permission{
							{Resource: "task", Action: "exec"},
						},
					},
				},
			},
			username: "user1",
			resource: "task",
			action:   "exec",
			want:     true,
		},
		{
			name: "user role not found in config",
			config: ParsedAuthConfig{
				Users: map[User][]Role{
					"user1": {"role1"},
				},
				Roles: map[string]Permissions{},
			},
			username: "user1",
			resource: "workflow",
			action:   "list",
			want:     false,
		},
		{
			name: "empty username",
			config: ParsedAuthConfig{
				Users: map[User][]Role{
					"user1": {"role1"},
				},
				Roles: map[string]Permissions{
					"role1": {
						Endpoints: []Permission{
							{Resource: "workflow", Action: "list"},
						},
					},
				},
			},
			username: "",
			resource: "workflow",
			action:   "list",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Authorizer{
				config:    tt.config,
				configDir: "",
			}
			got := a.canAccessEndpoint(tt.username, tt.resource, tt.action)
			if got != tt.want {
				t.Errorf("Authorizer.canAccessEndpoint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanAccessTask(t *testing.T) {
	tests := []struct {
		name     string
		config   ParsedAuthConfig
		username string
		plugin   string
		task     string
		want     bool
	}{
		{
			name: "user has exact permission",
			config: ParsedAuthConfig{
				Users: map[User][]Role{
					"user1": {"role1"},
				},
				Roles: map[string]Permissions{
					"role1": {
						Tasks: []Permission{
							{Resource: "git", Action: "clone"},
						},
					},
				},
			},
			username: "user1",
			plugin:   "git",
			task:     "clone",
			want:     true,
		},
		{
			name: "user has wildcard plugin permission",
			config: ParsedAuthConfig{
				Users: map[User][]Role{
					"user1": {"role1"},
				},
				Roles: map[string]Permissions{
					"role1": {
						Tasks: []Permission{
							{Resource: "*", Action: "clone"},
						},
					},
				},
			},
			username: "user1",
			plugin:   "git",
			task:     "clone",
			want:     true,
		},
		{
			name: "user has wildcard task permission",
			config: ParsedAuthConfig{
				Users: map[User][]Role{
					"user1": {"role1"},
				},
				Roles: map[string]Permissions{
					"role1": {
						Tasks: []Permission{
							{Resource: "git", Action: "*"},
						},
					},
				},
			},
			username: "user1",
			plugin:   "git",
			task:     "clone",
			want:     true,
		},
		{
			name: "user has full wildcard permission",
			config: ParsedAuthConfig{
				Users: map[User][]Role{
					"user1": {"role1"},
				},
				Roles: map[string]Permissions{
					"role1": {
						Tasks: []Permission{
							{Resource: "*", Action: "*"},
						},
					},
				},
			},
			username: "user1",
			plugin:   "git",
			task:     "clone",
			want:     true,
		},
		{
			name: "user not in config",
			config: ParsedAuthConfig{
				Users: map[User][]Role{
					"user1": {"role1"},
				},
				Roles: map[string]Permissions{
					"role1": {
						Tasks: []Permission{
							{Resource: "git", Action: "clone"},
						},
					},
				},
			},
			username: "user2",
			plugin:   "git",
			task:     "clone",
			want:     false,
		},
		{
			name: "user has no matching permission",
			config: ParsedAuthConfig{
				Users: map[User][]Role{
					"user1": {"role1"},
				},
				Roles: map[string]Permissions{
					"role1": {
						Tasks: []Permission{
							{Resource: "git", Action: "clone"},
						},
					},
				},
			},
			username: "user1",
			plugin:   "git",
			task:     "push",
			want:     false,
		},
		{
			name: "user has multiple roles with matching permission",
			config: ParsedAuthConfig{
				Users: map[User][]Role{
					"user1": {"role1", "role2"},
				},
				Roles: map[string]Permissions{
					"role1": {
						Tasks: []Permission{
							{Resource: "git", Action: "clone"},
						},
					},
					"role2": {
						Tasks: []Permission{
							{Resource: "docker", Action: "build"},
						},
					},
				},
			},
			username: "user1",
			plugin:   "docker",
			task:     "build",
			want:     true,
		},
		{
			name: "user role not found in config",
			config: ParsedAuthConfig{
				Users: map[User][]Role{
					"user1": {"role1"},
				},
				Roles: map[string]Permissions{},
			},
			username: "user1",
			plugin:   "git",
			task:     "clone",
			want:     false,
		},
		{
			name: "empty username",
			config: ParsedAuthConfig{
				Users: map[User][]Role{
					"user1": {"role1"},
				},
				Roles: map[string]Permissions{
					"role1": {
						Tasks: []Permission{
							{Resource: "git", Action: "clone"},
						},
					},
				},
			},
			username: "",
			plugin:   "git",
			task:     "clone",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Authorizer{
				config:    tt.config,
				configDir: "",
			}
			got := a.canAccessTask(tt.username, tt.plugin, tt.task)
			if got != tt.want {
				t.Errorf("Authorizer.canAccessTask() = %v, want %v", got, tt.want)
			}
		})
	}
}
