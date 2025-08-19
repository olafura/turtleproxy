package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func createTestCertificate(t *testing.T, notBefore, notAfter time.Time) ([]byte, []byte) {
	// Generate a private key
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Test CA"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// Encode private key to PEM
	privKeyDER, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		t.Fatalf("Failed to marshal private key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privKeyDER})

	return certPEM, keyPEM
}

func createTestCAFiles(t *testing.T, caRoot string, notBefore, notAfter time.Time) {
	err := os.MkdirAll(caRoot, 0o755)
	if err != nil {
		t.Fatalf("Failed to create CA root directory: %v", err)
	}

	certPEM, keyPEM := createTestCertificate(t, notBefore, notAfter)

	certPath := filepath.Join(caRoot, "rootCA.pem")
	keyPath := filepath.Join(caRoot, "rootCA-key.pem")

	err = os.WriteFile(certPath, certPEM, 0o644)
	if err != nil {
		t.Fatalf("Failed to write certificate file: %v", err)
	}

	err = os.WriteFile(keyPath, keyPEM, 0o600)
	if err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}
}

func TestGetCertSuccess(t *testing.T) {
	// Create temporary directory for test CA
	tempDir := t.TempDir()

	// Create valid certificate (valid for 1 hour from now)
	notBefore := time.Now().Add(-1 * time.Minute)
	notAfter := time.Now().Add(1 * time.Hour)
	createTestCAFiles(t, tempDir, notBefore, notAfter)

	cert, err := GetCert(tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cert == nil {
		t.Fatal("Expected certificate, got nil")
	}

	if cert.Leaf == nil {
		t.Fatal("Expected parsed certificate leaf, got nil")
	}
}

func TestGetCertDirectoryNotFound(t *testing.T) {
	nonExistentDir := "/nonexistent/directory"

	_, err := GetCert(nonExistentDir)
	if err == nil {
		t.Fatal("Expected error for non-existent directory, got nil")
	}
}

func TestGetCertMissingCertFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create only the key file, not the cert file
	keyPath := filepath.Join(tempDir, "rootCA-key.pem")
	err := os.WriteFile(keyPath, []byte("dummy key"), 0o600)
	if err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	_, err = GetCert(tempDir)
	if err == nil {
		t.Fatal("Expected error for missing certificate file, got nil")
	}
}

func TestGetCertMissingKeyFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create only the cert file, not the key file
	certPath := filepath.Join(tempDir, "rootCA.pem")
	err := os.WriteFile(certPath, []byte("dummy cert"), 0o644)
	if err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}

	_, err = GetCert(tempDir)
	if err == nil {
		t.Fatal("Expected error for missing key file, got nil")
	}
}

func TestGetCertExpiredCertificate(t *testing.T) {
	tempDir := t.TempDir()

	// Create expired certificate (expired 1 hour ago)
	notBefore := time.Now().Add(-2 * time.Hour)
	notAfter := time.Now().Add(-1 * time.Hour)
	createTestCAFiles(t, tempDir, notBefore, notAfter)

	_, err := GetCert(tempDir)
	if err == nil {
		t.Fatal("Expected error for expired certificate, got nil")
	}
}

func TestGetCertNotYetValidCertificate(t *testing.T) {
	tempDir := t.TempDir()

	// Create certificate that's not yet valid (valid in 1 hour)
	notBefore := time.Now().Add(1 * time.Hour)
	notAfter := time.Now().Add(2 * time.Hour)
	createTestCAFiles(t, tempDir, notBefore, notAfter)

	_, err := GetCert(tempDir)
	if err == nil {
		t.Fatal("Expected error for not-yet-valid certificate, got nil")
	}
}

func TestGetCertInvalidCertificateFormat(t *testing.T) {
	tempDir := t.TempDir()

	certPath := filepath.Join(tempDir, "rootCA.pem")
	keyPath := filepath.Join(tempDir, "rootCA-key.pem")

	// Write invalid certificate and key data
	err := os.WriteFile(certPath, []byte("invalid cert data"), 0o644)
	if err != nil {
		t.Fatalf("Failed to write invalid cert file: %v", err)
	}

	err = os.WriteFile(keyPath, []byte("invalid key data"), 0o600)
	if err != nil {
		t.Fatalf("Failed to write invalid key file: %v", err)
	}

	_, err = GetCert(tempDir)
	if err == nil {
		t.Fatal("Expected error for invalid certificate format, got nil")
	}
}

func TestGetCertEmptyCARoot(t *testing.T) {
	tempDir := t.TempDir()

	// Create valid certificate
	notBefore := time.Now().Add(-1 * time.Minute)
	notAfter := time.Now().Add(1 * time.Hour)
	createTestCAFiles(t, tempDir, notBefore, notAfter)

	// Set CAROOT environment variable
	originalCARoot := os.Getenv("CAROOT")
	defer func() {
		if err := os.Setenv("CAROOT", originalCARoot); err != nil {
			t.Errorf("Failed to restore CAROOT: %v", err)
		}
	}()
	if err := os.Setenv("CAROOT", tempDir); err != nil {
		t.Fatalf("Failed to set CAROOT: %v", err)
	}

	// Test with empty caRoot parameter (should use CAROOT env var)
	cert, err := GetCert("")
	if err != nil {
		t.Fatalf("Expected no error when using CAROOT env var, got %v", err)
	}

	if cert == nil {
		t.Fatal("Expected certificate, got nil")
	}
}

func TestGetCertPathTraversalProtection(t *testing.T) {
	// Test that path traversal attempts are blocked
	_, err := GetCert("~/../etc/passwd")
	if err == nil {
		t.Fatal("Expected error for path traversal attempt, got nil")
	}

	_, err = GetCert("~/./../../etc")
	if err == nil {
		t.Fatal("Expected error for path traversal attempt, got nil")
	}
}
