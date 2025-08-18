package main

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"os/user"
	"path/filepath"
	"strings"
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
			return nil, err
		}

		caRoot = filepath.Join(usr.HomeDir, caRoot[2:])
	}

	_, err = os.Stat(caRoot)

	if err != nil {
		return nil, err
	}

	rootCAPath := filepath.Join(caRoot, "rootCA.pem")
	rootCAKeyPath := filepath.Join(caRoot, "rootCA-key.pem")

	_, err = os.Stat(rootCAPath)

	if err != nil {
		return nil, err
	}

	_, err = os.Stat(rootCAKeyPath)

	if err != nil {
		return nil, err
	}

	caCert, err = os.ReadFile(rootCAPath)

	if err != nil {
		return nil, err
	}

	caCertKey, err = os.ReadFile(rootCAKeyPath)

	if err != nil {
		return nil, err
	}

	parsedCert, err = tls.X509KeyPair(caCert, caCertKey)

	if err != nil {
		return nil, err
	}

	if parsedCert.Leaf, err = x509.ParseCertificate(parsedCert.Certificate[0]); err != nil {
		return nil, err
	}

	return &parsedCert, nil
}
