package database

import (
	"testing"
)

func TestStringToKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		want    Key
		wantErr bool
	}{
		{
			name: "valid result key",
			key:  "res:123",
			want: Key{Prefix: "res", ID: "123"},
		},
		{
			name: "valid request key",
			key:  "req:456",
			want: Key{Prefix: "req", ID: "456"},
		},
		{
			name: "valid key with long ID",
			key:  "res:1234567890",
			want: Key{Prefix: "res", ID: "1234567890"},
		},
		{
			name:    "invalid key no colon",
			key:     "res123",
			wantErr: true,
		},
		{
			name:    "invalid key empty",
			key:     "",
			wantErr: true,
		},
		{
			name:    "invalid key multiple colons",
			key:     "res:123:456",
			wantErr: true,
		},
		{
			name: "key with empty prefix",
			key:  ":123",
			want: Key{Prefix: "", ID: "123"},
		},
		{
			name: "key with empty ID",
			key:  "res:",
			want: Key{Prefix: "res", ID: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StringToKey(tt.key)
			if tt.wantErr {
				if err == nil {
					t.Errorf("StringToKey() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("StringToKey() unexpected error: %v", err)
				return
			}
			if got.Prefix != tt.want.Prefix {
				t.Errorf("StringToKey() Prefix = %v, want %v", got.Prefix, tt.want.Prefix)
			}
			if got.ID != tt.want.ID {
				t.Errorf("StringToKey() ID = %v, want %v", got.ID, tt.want.ID)
			}
		})
	}
}

func TestGenerateResultKey(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{
			name: "simple ID",
			id:   "123",
			want: "res:123",
		},
		{
			name: "long ID",
			id:   "1234567890",
			want: "res:1234567890",
		},
		{
			name: "empty ID",
			id:   "",
			want: "res:",
		},
		{
			name: "ID with special characters",
			id:   "abc-123",
			want: "res:abc-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateResultKey(tt.id)
			if string(got) != tt.want {
				t.Errorf("GenerateResultKey() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestGenerateRequestKey(t *testing.T) {
	tests := []struct {
		name string
		id   int64
		want string
	}{
		{
			name: "positive ID",
			id:   123,
			want: "req:123",
		},
		{
			name: "zero ID",
			id:   0,
			want: "req:0",
		},
		{
			name: "negative ID",
			id:   -456,
			want: "req:-456",
		},
		{
			name: "large ID",
			id:   9223372036854775807,
			want: "req:9223372036854775807",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateRequestKey(tt.id)
			if string(got) != tt.want {
				t.Errorf("GenerateRequestKey() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestGenerateRequestKeyFromString(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{
			name: "numeric string",
			id:   "123",
			want: "req:123",
		},
		{
			name: "empty string",
			id:   "",
			want: "req:",
		},
		{
			name: "alphanumeric string",
			id:   "abc123",
			want: "req:abc123",
		},
		{
			name: "string with special characters",
			id:   "test-id-456",
			want: "req:test-id-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateRequestKeyFromString(tt.id)
			if string(got) != tt.want {
				t.Errorf("GenerateRequestKeyFromString() = %v, want %v", string(got), tt.want)
			}
		})
	}
}
