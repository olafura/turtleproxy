package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

func GetCert(caRoot string) (*tls.Certificate, error) {
	var (
		err        error
		caCert     []byte
		caCertKey  []byte
		parsedCert tls.Certificate
	)

	if caRoot == "" {
		caRoot = os.Getenv("CAROOT")
	}

	if caRoot == "" {
		caRoot = "~/.local/share/mkcert"
	}

	if strings.HasPrefix(caRoot, "~/") {
		usr, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("failed to get current user: %w", err)
		}

		relativePath := caRoot[2:]
		if strings.Contains(relativePath, "..") || strings.HasPrefix(relativePath, "/") {
			return nil, errors.New("invalid path: path traversal detected")
		}

		caRoot = filepath.Join(usr.HomeDir, relativePath)
	}

	caRoot = filepath.Clean(caRoot)

	absPath, err := filepath.Abs(caRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}
	caRoot = absPath

	_, err = os.Stat(caRoot)
	if err != nil {
		return nil, fmt.Errorf("CA root directory not found at %s: %w", caRoot, err)
	}

	rootCAPath := filepath.Join(caRoot, "rootCA.pem")
	rootCAKeyPath := filepath.Join(caRoot, "rootCA-key.pem")

	_, err = os.Stat(rootCAPath)
	if err != nil {
		return nil, fmt.Errorf("CA certificate file not found at %s: %w", rootCAPath, err)
	}

	_, err = os.Stat(rootCAKeyPath)
	if err != nil {
		return nil, fmt.Errorf("CA private key file not found at %s: %w", rootCAKeyPath, err)
	}

	caCert, err = os.ReadFile(rootCAPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate from %s: %w", rootCAPath, err)
	}

	caCertKey, err = os.ReadFile(rootCAKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA private key from %s: %w", rootCAKeyPath, err)
	}

	parsedCert, err = tls.X509KeyPair(caCert, caCertKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate and key pair: %w", err)
	}

	if parsedCert.Leaf, err = x509.ParseCertificate(parsedCert.Certificate[0]); err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	if time.Now().After(parsedCert.Leaf.NotAfter) {
		return nil, fmt.Errorf("certificate has expired on %s", parsedCert.Leaf.NotAfter.Format(time.RFC3339))
	}

	if time.Now().Before(parsedCert.Leaf.NotBefore) {
		return nil, fmt.Errorf("certificate is not yet valid (valid from %s)", parsedCert.Leaf.NotBefore.Format(time.RFC3339))
	}

	return &parsedCert, nil
}
