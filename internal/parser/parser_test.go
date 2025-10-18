package parser

import (
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := map[string]struct {
		args           []string
		wantPositional []any
		wantOptions    map[string]any
		wantErr        bool
	}{
		"simple key-value": {
			args:           []string{"key=value"},
			wantPositional: []any{},
			wantOptions:    map[string]any{"key": "value"},
		},
		"key with hyphen for jackadi tag": {
			args:           []string{"dry-run=true"},
			wantPositional: []any{},
			wantOptions:    map[string]any{"dry-run": "true"},
		},
		"key with underscore": {
			args:           []string{"output_file=/tmp/test.txt"},
			wantPositional: []any{},
			wantOptions:    map[string]any{"output_file": "/tmp/test.txt"},
		},
		"key with multiple hyphens": {
			args:           []string{"some-long-key-name=value"},
			wantPositional: []any{},
			wantOptions:    map[string]any{"some-long-key-name": "value"},
		},
		"mixed hyphens and underscores": {
			args:           []string{"key_with-both=test"},
			wantPositional: []any{},
			wantOptions:    map[string]any{"key_with-both": "test"},
		},
		"hyphenated keys matching jackadi tags": {
			args:           []string{"output-file=/tmp/output.txt", "max-retries=5"},
			wantPositional: []any{},
			wantOptions: map[string]any{
				"output-file": "/tmp/output.txt",
				"max-retries": "5",
			},
		},
		"multiple options": {
			args:           []string{"verbose=true", "timeout=30", "dry-run=false"},
			wantPositional: []any{},
			wantOptions: map[string]any{
				"verbose": "true",
				"timeout": "30",
				"dry-run": "false",
			},
		},
		"positional arguments": {
			args:           []string{"arg1", "arg2", "arg3"},
			wantPositional: []any{"arg1", "arg2", "arg3"},
			wantOptions:    map[string]any{},
		},
		"options then positional should error": {
			args:    []string{"key=value", "positional"},
			wantErr: true,
		},
		"positional then options": {
			args:           []string{"positional", "key=value"},
			wantPositional: []any{"positional"},
			wantOptions:    map[string]any{"key": "value"},
			wantErr:        false,
		},
		"empty args": {
			args:           []string{},
			wantPositional: []any{},
			wantOptions:    map[string]any{},
		},
		"numeric key": {
			args:           []string{"key123=value"},
			wantPositional: []any{},
			wantOptions:    map[string]any{"key123": "value"},
		},
		"boolean values": {
			args:           []string{"enabled=true", "disabled=false"},
			wantPositional: []any{},
			wantOptions: map[string]any{
				"enabled":  "true",
				"disabled": "false",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := ParseArgs(tt.args)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result.Positional) != len(tt.wantPositional) {
				t.Errorf("positional length mismatch: got %d, want %d", len(result.Positional), len(tt.wantPositional))
			}

			for i, pos := range tt.wantPositional {
				if i >= len(result.Positional) {
					break
				}
				if result.Positional[i] != pos {
					t.Errorf("positional[%d]: got %v, want %v", i, result.Positional[i], pos)
				}
			}

			if len(result.Options) != len(tt.wantOptions) {
				t.Errorf("options length mismatch: got %d, want %d", len(result.Options), len(tt.wantOptions))
			}

			for key, wantVal := range tt.wantOptions {
				gotVal, exists := result.Options[key]
				if !exists {
					t.Errorf("option %q not found", key)
					continue
				}
				if gotVal != wantVal {
					t.Errorf("option %q: got %v, want %v", key, gotVal, wantVal)
				}
			}
		})
	}
}
