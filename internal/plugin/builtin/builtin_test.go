package builtin

import (
	"context"
	"testing"
)

func TestPing(t *testing.T) {
	result, err := ping()
	if err != nil {
		t.Errorf("ping() unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("ping() = %v, want true", result)
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		wantErr bool
	}{
		{
			name:    "simple echo command",
			args:    "echo test",
			wantErr: false,
		},
		{
			name:    "command with arguments",
			args:    "echo hello world",
			wantErr: false,
		},
		{
			name:    "invalid command",
			args:    "nonexistentcommand12345",
			wantErr: true,
		},
		{
			name:    "command with quotes",
			args:    "echo 'quoted string'",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := run(ctx, tt.args)

			if tt.wantErr && err == nil {
				t.Errorf("run() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("run() unexpected error: %v", err)
			}
		})
	}
}

func TestParseNames(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantPlugin string
		wantTask   string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:       "plugin only",
			input:      "cmd",
			wantPlugin: "cmd",
			wantTask:   "",
			wantErr:    false,
		},
		{
			name:       "plugin and task",
			input:      "cmd:run",
			wantPlugin: "cmd",
			wantTask:   "run",
			wantErr:    false,
		},
		{
			name:       "plugin with multiple colons",
			input:      "cmd:run:extra",
			wantPlugin: "cmd",
			wantTask:   "run",
			wantErr:    false,
		},
		{
			name:       "empty string",
			input:      "",
			wantPlugin: "",
			wantTask:   "",
			wantErr:    true,
			wantErrMsg: "missing plugin or plugin:task",
		},
		{
			name:       "colon only",
			input:      ":",
			wantPlugin: "",
			wantTask:   "",
			wantErr:    true,
			wantErrMsg: "missing plugin or plugin:task",
		},
		{
			name:       "task without plugin",
			input:      ":task",
			wantPlugin: "",
			wantTask:   "",
			wantErr:    true,
			wantErrMsg: "missing plugin or plugin:task",
		},
		{
			name:       "plugin with empty task",
			input:      "cmd:",
			wantPlugin: "cmd",
			wantTask:   "",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin, task, err := parseNames(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseNames() expected error, got nil")
					return
				}
				if tt.wantErrMsg != "" && err.Error() != tt.wantErrMsg {
					t.Errorf("parseNames() error = %v, want %v", err.Error(), tt.wantErrMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("parseNames() unexpected error: %v", err)
				return
			}

			if plugin != tt.wantPlugin {
				t.Errorf("parseNames() plugin = %v, want %v", plugin, tt.wantPlugin)
			}
			if task != tt.wantTask {
				t.Errorf("parseNames() task = %v, want %v", task, tt.wantTask)
			}
		})
	}
}
