package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestNewHtpasswd(t *testing.T) {
	got := NewHtpasswd()

	// Check that creds map is initialized
	if got.creds == nil {
		t.Error("Expected creds map to be initialized, got nil")
	}

	// Check that creds map is empty
	if len(got.creds) != 0 {
		t.Errorf("Expected empty creds map, got %d entries", len(got.creds))
	}
}

func TestHtpasswd_Get(t *testing.T) {
	type fields struct {
		creds map[string]string
	}
	type args struct {
		user string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "existing user",
			fields: fields{
				creds: map[string]string{
					"testuser": "$2b$10$hash123",
				},
			},
			args: args{
				user: "testuser",
			},
			want:    "$2b$10$hash123",
			wantErr: false,
		},
		{
			name: "non-existent user",
			fields: fields{
				creds: map[string]string{
					"testuser": "$2b$10$hash123",
				},
			},
			args: args{
				user: "nonexistent",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "user with empty password",
			fields: fields{
				creds: map[string]string{
					"emptyuser": "",
				},
			},
			args: args{
				user: "emptyuser",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "empty credentials map",
			fields: fields{
				creds: map[string]string{},
			},
			args: args{
				user: "anyuser",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Htpasswd{
				creds: tt.fields.creds,
			}
			got, err := h.Get(tt.args.user)
			if (err != nil) != tt.wantErr {
				t.Errorf("Htpasswd.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Htpasswd.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHtpasswd_load(t *testing.T) {
	type fields struct {
		creds map[string]string
	}
	type args struct {
		file string
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		setupFn   func(t *testing.T) string
		cleanupFn func(t *testing.T, path string)
		wantCreds map[string]string
	}{
		{
			name: "valid htpasswd file",
			fields: fields{
				creds: make(map[string]string),
			},
			args:    args{},
			wantErr: false,
			setupFn: func(t *testing.T) string {
				t.Helper()
				return createTempHtpasswdFile(t, "user1:$2b$05$hash1\nuser2:$2b$05$hash2")
			},
			cleanupFn: cleanupTempFile,
			wantCreds: map[string]string{
				"user1": "$2b$05$hash1",
				"user2": "$2b$05$hash2",
			},
		},
		{
			name: "empty file",
			fields: fields{
				creds: make(map[string]string),
			},
			args:    args{},
			wantErr: true,
			setupFn: func(t *testing.T) string {
				t.Helper()
				return createTempHtpasswdFile(t, "")
			},
			cleanupFn: cleanupTempFile,
			wantCreds: map[string]string{},
		},
		{
			name: "file with comments",
			fields: fields{
				creds: make(map[string]string),
			},
			args:    args{},
			wantErr: false,
			setupFn: func(t *testing.T) string {
				t.Helper()
				return createTempHtpasswdFile(t, "# Comment\nuser1:$2b$05$hash1\n# Another comment\nuser2:$2b$05$hash2")
			},
			cleanupFn: cleanupTempFile,
			wantCreds: map[string]string{
				"user1": "$2b$05$hash1",
				"user2": "$2b$05$hash2",
			},
		},
		{
			name: "non-existent file",
			fields: fields{
				creds: make(map[string]string),
			},
			args: args{
				file: "/non/existent/file.htpasswd",
			},
			wantErr:   true,
			wantCreds: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Htpasswd{
				creds: tt.fields.creds,
			}

			var testFile string
			if tt.setupFn != nil {
				testFile = tt.setupFn(t)
				if tt.args.file == "" {
					tt.args.file = testFile
				}
			}

			if tt.cleanupFn != nil && testFile != "" {
				defer tt.cleanupFn(t, testFile)
			}

			if err := h.load(tt.args.file); (err != nil) != tt.wantErr {
				t.Errorf("Htpasswd.load() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Validate loaded credentials
			if !tt.wantErr && tt.wantCreds != nil {
				if len(h.creds) != len(tt.wantCreds) {
					t.Errorf("Expected %d credentials, got %d", len(tt.wantCreds), len(h.creds))
				}
				for user, expectedHash := range tt.wantCreds {
					if actualHash, ok := h.creds[user]; !ok {
						t.Errorf("Expected user %s not found", user)
					} else if actualHash != expectedHash {
						t.Errorf("For user %s, expected hash %s, got %s", user, expectedHash, actualHash)
					}
				}
			}
		})
	}
}

func TestHtpasswd_basicAuthMiddleware(t *testing.T) {
	// Generate proper bcrypt hash for "secret"
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to generate bcrypt hash: %v", err)
	}

	type fields struct {
		creds map[string]string
	}
	type args struct {
		username string
		password string
		hasAuth  bool
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
		wantAuthHeader bool
	}{
		{
			name: "valid authentication",
			fields: fields{
				creds: map[string]string{
					"testuser": string(hash), // "secret"
				},
			},
			args: args{
				username: "testuser",
				password: "secret",
				hasAuth:  true,
			},
			wantStatusCode: http.StatusOK,
			wantAuthHeader: false,
		},
		{
			name: "invalid password",
			fields: fields{
				creds: map[string]string{
					"testuser": string(hash), // "secret"
				},
			},
			args: args{
				username: "testuser",
				password: "wrongpassword",
				hasAuth:  true,
			},
			wantStatusCode: http.StatusUnauthorized,
			wantAuthHeader: true,
		},
		{
			name: "unknown user",
			fields: fields{
				creds: map[string]string{
					"testuser": string(hash),
				},
			},
			args: args{
				username: "unknownuser",
				password: "secret",
				hasAuth:  true,
			},
			wantStatusCode: http.StatusUnauthorized,
			wantAuthHeader: true,
		},
		{
			name: "no authentication",
			fields: fields{
				creds: map[string]string{
					"testuser": string(hash),
				},
			},
			args: args{
				hasAuth: false,
			},
			wantStatusCode: http.StatusUnauthorized,
			wantAuthHeader: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Htpasswd{
				creds: tt.fields.creds,
			}

			// Create a mock handler that just returns 200 OK
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Create the middleware
			middleware := h.basicAuthMiddleware(nextHandler)

			// Create test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.args.hasAuth {
				req.SetBasicAuth(tt.args.username, tt.args.password)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute the middleware
			middleware.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.wantStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.wantStatusCode, rr.Code)
			}

			// Check WWW-Authenticate header
			authHeader := rr.Header().Get("WWW-Authenticate")
			if tt.wantAuthHeader && authHeader == "" {
				t.Error("Expected WWW-Authenticate header, but it was missing")
			}
			if !tt.wantAuthHeader && authHeader != "" {
				t.Errorf("Expected no WWW-Authenticate header, but got: %s", authHeader)
			}
		})
	}
}

func createTempHtpasswdFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "htpasswd_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	return tmpFile.Name()
}

func cleanupTempFile(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Errorf("Failed to cleanup temp file %s: %v", path, err)
	}
}
