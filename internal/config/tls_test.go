package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"
)

func TestGetAPITLSCertificate(t *testing.T) {
	tests := []struct {
		name      string
		setupFn   func(t *testing.T) (string, string)
		cleanupFn func(t *testing.T, certFile, keyFile string)
		wantErr   bool
	}{
		{
			name: "valid certificate and key files",
			setupFn: func(t *testing.T) (string, string) {
				t.Helper()
				return createTempTLSFiles(t)
			},
			cleanupFn: cleanupTLSFiles,
			wantErr:   false,
		},
		{
			name: "non-existent certificate file",
			setupFn: func(t *testing.T) (string, string) {
				t.Helper()
				_, keyFile := createTempTLSFiles(t)
				return "/non/existent/cert.pem", keyFile
			},
			cleanupFn: func(t *testing.T, certFile, keyFile string) {
				t.Helper()
				// Only cleanup keyFile since certFile doesn't exist
				if err := os.Remove(keyFile); err != nil {
					t.Errorf("Failed to cleanup key file: %v", err)
				}
			},
			wantErr: true,
		},
		{
			name: "non-existent key file",
			setupFn: func(t *testing.T) (string, string) {
				t.Helper()
				certFile, _ := createTempTLSFiles(t)
				return certFile, "/non/existent/key.pem"
			},
			cleanupFn: func(t *testing.T, certFile, keyFile string) {
				t.Helper()
				// Only cleanup certFile since keyFile doesn't exist
				if err := os.Remove(certFile); err != nil {
					t.Errorf("Failed to cleanup cert file: %v", err)
				}
			},
			wantErr: true,
		},
		{
			name: "empty file paths",
			setupFn: func(t *testing.T) (string, string) {
				t.Helper()
				return "", ""
			},
			cleanupFn: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			certFile, keyFile := tt.setupFn(t)

			if tt.cleanupFn != nil {
				t.Helper()
				defer tt.cleanupFn(t, certFile, keyFile)
			}

			certs, err := GetAPITLSCertificate(certFile, keyFile)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetAPITLSCertificate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(certs) != 1 {
					t.Errorf("Expected 1 certificate, got %d", len(certs))
				}

				// Verify the certificate is valid
				cert := certs[0]
				if cert.Certificate == nil {
					t.Error("Certificate is nil")
				}
				if cert.PrivateKey == nil {
					t.Error("Private key is nil")
				}
			}
		})
	}
}

func TestGetMTLSCertificate(t *testing.T) {
	tests := []struct {
		name      string
		setupFn   func(t *testing.T) (string, string, string)
		cleanupFn func(t *testing.T, certFile, keyFile, caFile string)
		wantErr   bool
	}{
		{
			name: "valid mTLS configuration",
			setupFn: func(t *testing.T) (string, string, string) {
				t.Helper()
				certFile, keyFile := createTempTLSFiles(t)
				caFile := createTempCAFile(t)
				return certFile, keyFile, caFile
			},
			cleanupFn: func(t *testing.T, certFile, keyFile, caFile string) {
				t.Helper()
				cleanupTLSFiles(t, certFile, keyFile)
				if err := os.Remove(caFile); err != nil {
					t.Errorf("Failed to cleanup CA file: %v", err)
				}
			},
			wantErr: false,
		},
		{
			name: "non-existent CA file",
			setupFn: func(t *testing.T) (string, string, string) {
				t.Helper()
				certFile, keyFile := createTempTLSFiles(t)
				return certFile, keyFile, "/non/existent/ca.pem"
			},
			cleanupFn: func(t *testing.T, certFile, keyFile, caFile string) {
				t.Helper()
				cleanupTLSFiles(t, certFile, keyFile)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			certFile, keyFile, caFile := tt.setupFn(t)

			if tt.cleanupFn != nil {
				defer tt.cleanupFn(t, certFile, keyFile, caFile)
			}

			certs, caPool, err := GetMTLSCertificate(certFile, keyFile, caFile)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetMTLSCertificate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(certs) != 1 {
					t.Errorf("Expected 1 certificate, got %d", len(certs))
				}
				if caPool == nil {
					t.Error("CA pool is nil")
				}
			}
		})
	}
}

// createTempTLSFiles creates temporary certificate and key files for testing.
func createTempTLSFiles(t *testing.T) (certFile, keyFile string) {
	t.Helper()

	// Generate a private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Jackadi"},
			Country:       []string{"FR"},
			Province:      []string{""},
			Locality:      []string{"Not Paris"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: nil,
		DNSNames:    []string{"localhost"},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	// Create temp certificate file
	certTempFile, err := os.CreateTemp("", "test_cert_*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp cert file: %v", err)
	}

	certPEM := &pem.Block{Type: "CERTIFICATE", Bytes: certDER}
	if err := pem.Encode(certTempFile, certPEM); err != nil {
		t.Fatalf("Failed to encode certificate: %v", err)
	}
	certTempFile.Close()

	// Create temp key file
	keyTempFile, err := os.CreateTemp("", "test_key_*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp key file: %v", err)
	}

	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to marshal private key: %v", err)
	}

	keyPEM := &pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyDER}
	if err := pem.Encode(keyTempFile, keyPEM); err != nil {
		t.Fatalf("Failed to encode private key: %v", err)
	}
	keyTempFile.Close()

	return certTempFile.Name(), keyTempFile.Name()
}

// createTempCAFile creates a temporary CA certificate file for testing.
func createTempCAFile(t *testing.T) string {
	t.Helper()

	// Generate a CA private key
	caPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate CA private key: %v", err)
	}

	// Create CA certificate template
	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization:  []string{"Test CA"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test City"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create CA certificate
	caCertDER, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caPrivateKey.PublicKey, caPrivateKey)
	if err != nil {
		t.Fatalf("Failed to create CA certificate: %v", err)
	}

	// create temp CA file
	caTempFile, err := os.CreateTemp("", "test_ca_*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp CA file: %v", err)
	}

	caPEM := &pem.Block{Type: "CERTIFICATE", Bytes: caCertDER}
	if err := pem.Encode(caTempFile, caPEM); err != nil {
		t.Fatalf("Failed to encode CA certificate: %v", err)
	}
	caTempFile.Close()

	return caTempFile.Name()
}

// cleanupTLSFiles removes temporary certificate and key files.
func cleanupTLSFiles(t *testing.T, certFile, keyFile string) {
	t.Helper()
	if err := os.Remove(certFile); err != nil {
		t.Errorf("Failed to cleanup cert file %s: %v", certFile, err)
	}
	if err := os.Remove(keyFile); err != nil {
		t.Errorf("Failed to cleanup key file %s: %v", keyFile, err)
	}
}
