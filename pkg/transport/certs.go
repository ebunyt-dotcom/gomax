package transport

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"errors"
	"fmt"
	"sync"
)

//go:embed certs/rootca_ssl_rsa2022.crt
var rootCACertPEM []byte

var (
	rootCACertPool *x509.CertPool
	certPoolOnce   sync.Once
	certPoolErr    error
)

// GetEmbeddedRootCAPEM returns the raw PEM-encoded bytes of the embedded Russian Trusted Root CA.
func GetEmbeddedRootCAPEM() []byte {
	return rootCACertPEM
}

// GetRootCACertPool returns an *x509.CertPool initialized with the system's root certificates
// combined with the embedded Russian Trusted Root CA.
// If the system certificate pool cannot be loaded (e.g. minimal scratch containers),
// a new standalone pool containing only the embedded root CA is returned.
func GetRootCACertPool() (*x509.CertPool, error) {
	certPoolOnce.Do(func() {
		if len(rootCACertPEM) == 0 {
			certPoolErr = errors.New("embedded root CA certificate PEM is empty")
			return
		}

		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}

		if ok := pool.AppendCertsFromPEM(rootCACertPEM); !ok {
			certPoolErr = errors.New("failed to parse embedded root CA certificate PEM")
			return
		}

		rootCACertPool = pool
	})

	return rootCACertPool, certPoolErr
}

// NewTLSConfig constructs a production-ready *tls.Config configured with the Russian Trusted Root CA
// and the specified serverName for SNI and hostname verification.
func NewTLSConfig(serverName string) (*tls.Config, error) {
	pool, err := GetRootCACertPool()
	if err != nil {
		return nil, fmt.Errorf("load root CA cert pool: %w", err)
	}

	return &tls.Config{
		RootCAs:    pool,
		ServerName: serverName,
		MinVersion: tls.VersionTLS12,
	}, nil
}
