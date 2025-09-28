package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// GetMTLSCertificate returns the TLS configuration using the provided certificates for mTLS.
//
// Agent configuration:
//
//	GetMTLSCertificate("./tls-example/agent_cert.pem", "./tls-example/agent_key.pem", "./tls-example/manager_ca_cert.pem")
//
// Manager Configuration:
//
//	GetMTLSCertificate("./tls-example/manager_cert.pem", "./tls-example/manager_key.pem", "./tls-example/agent_ca_cert.pem")
func GetMTLSCertificate(localCert, localKey, peerCA string) ([]tls.Certificate, *x509.CertPool, error) {
	cert, err := tls.LoadX509KeyPair(localCert, localKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load local certificate/privkey: %w", err)
	}

	ca := x509.NewCertPool()

	caBytes, err := os.ReadFile(peerCA)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read peer CA certificate '%s': %w", peerCA, err)
	}
	if ok := ca.AppendCertsFromPEM(caBytes); !ok {
		return nil, nil, fmt.Errorf("failed to parse '%s'", peerCA)
	}

	return []tls.Certificate{cert}, ca, nil
}

// GetAPITLSCertificate returns the TLS configuration for API server using the provided certificate and key.
//
// Example usage:
//
//	GetAPITLSCertificate("./tls-example/api_cert.pem", "./tls-example/api_key.pem")
func GetAPITLSCertificate(certFile, keyFile string) ([]tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load API certificate/privkey: %w", err)
	}

	return []tls.Certificate{cert}, nil
}
