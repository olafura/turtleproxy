package main

import (
	"crypto/tls"
	"errors"
	"testing"
	"time"
)

func TestNewCertStorage(t *testing.T) {
	cs := NewCertStorage()

	if cs.certs == nil {
		t.Error("Expected certs map to be initialized")
	}
	if cs.maxSize != 1000 {
		t.Errorf("Expected maxSize to be 1000, got %d", cs.maxSize)
	}
	if cs.ttl != 24*time.Hour {
		t.Errorf("Expected ttl to be 24 hours, got %v", cs.ttl)
	}
}

func TestCertStorageFetch(t *testing.T) {
	cs := NewCertStorage()
	cs.ttl = 1 * time.Second // Short TTL for testing

	hostname := "example.com"
	mockCert := &tls.Certificate{}

	// Test generator function that returns a mock certificate
	gen := func() (*tls.Certificate, error) {
		return mockCert, nil
	}

	// First fetch should call generator
	cert, err := cs.Fetch(hostname, gen)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if cert != mockCert {
		t.Error("Expected to receive mock certificate")
	}

	// Second fetch should return cached certificate
	cert2, err := cs.Fetch(hostname, func() (*tls.Certificate, error) {
		t.Error("Generator should not be called for cached certificate")
		return nil, errors.New("should not be called")
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if cert2 != mockCert {
		t.Error("Expected to receive cached certificate")
	}

	// Wait for TTL to expire
	time.Sleep(1100 * time.Millisecond)

	// Third fetch should call generator again due to expiration
	cert3, err := cs.Fetch(hostname, gen)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if cert3 != mockCert {
		t.Error("Expected to receive mock certificate after expiration")
	}
}

func TestCertStorageFetchError(t *testing.T) {
	cs := NewCertStorage()

	hostname := "example.com"
	expectedErr := errors.New("generation failed")

	gen := func() (*tls.Certificate, error) {
		return nil, expectedErr
	}

	cert, err := cs.Fetch(hostname, gen)
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
	if cert != nil {
		t.Error("Expected nil certificate on error")
	}
}

func TestCertStorageEviction(t *testing.T) {
	cs := NewCertStorage()
	cs.maxSize = 2 // Small size for testing eviction

	mockCert := &tls.Certificate{}
	gen := func() (*tls.Certificate, error) {
		return mockCert, nil
	}

	// Fill the cache to capacity
	_, err := cs.Fetch("host1.com", gen)
	if err != nil {
		t.Fatalf("Failed to fetch host1.com: %v", err)
	}
	_, err = cs.Fetch("host2.com", gen)
	if err != nil {
		t.Fatalf("Failed to fetch host2.com: %v", err)
	}

	if len(cs.certs) != 2 {
		t.Errorf("Expected 2 certificates in cache, got %d", len(cs.certs))
	}

	// Adding a third should evict the oldest
	_, err = cs.Fetch("host3.com", gen)
	if err != nil {
		t.Fatalf("Failed to fetch host3.com: %v", err)
	}

	if len(cs.certs) != 2 {
		t.Errorf("Expected 2 certificates in cache after eviction, got %d", len(cs.certs))
	}

	// host1.com should be evicted (oldest)
	if _, exists := cs.certs["host1.com"]; exists {
		t.Error("Expected host1.com to be evicted")
	}
	if _, exists := cs.certs["host2.com"]; !exists {
		t.Error("Expected host2.com to remain")
	}
	if _, exists := cs.certs["host3.com"]; !exists {
		t.Error("Expected host3.com to remain")
	}
}

func TestCertStorageEvictExpired(t *testing.T) {
	cs := NewCertStorage()
	cs.ttl = 1 * time.Millisecond // Very short TTL

	mockCert := &tls.Certificate{}
	gen := func() (*tls.Certificate, error) {
		return mockCert, nil
	}

	// Add a certificate
	_, err := cs.Fetch("expired.com", gen)
	if err != nil {
		t.Fatalf("Failed to fetch expired.com: %v", err)
	}

	if len(cs.certs) != 1 {
		t.Errorf("Expected 1 certificate in cache, got %d", len(cs.certs))
	}

	// Wait for expiration
	time.Sleep(2 * time.Millisecond)

	// Trigger eviction by adding another certificate
	_, err = cs.Fetch("new.com", gen)
	if err != nil {
		t.Fatalf("Failed to fetch new.com: %v", err)
	}

	// The expired certificate should be cleaned up
	if _, exists := cs.certs["expired.com"]; exists {
		t.Error("Expected expired.com to be evicted")
	}
	if _, exists := cs.certs["new.com"]; !exists {
		t.Error("Expected new.com to remain")
	}
}

func TestCertStorageEvictOldestEmptyCache(t *testing.T) {
	cs := NewCertStorage()

	// Should not panic when evicting from empty cache
	cs.evictOldest()

	if len(cs.certs) != 0 {
		t.Errorf("Expected empty cache to remain empty, got %d certificates", len(cs.certs))
	}
}
