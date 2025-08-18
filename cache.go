package main

// from https://github.com/elazarl/goproxy/blob/master/examples/certstorage/cache.go

import (
	"crypto/tls"
	"sync"
	"time"
)

type CertEntry struct {
	cert      *tls.Certificate
	timestamp time.Time
}

// CertStorage is a certificate cache with size limits and expiration
type CertStorage struct {
	certs   map[string]*CertEntry
	mtx     sync.RWMutex
	maxSize int
	ttl     time.Duration
}

func (cs *CertStorage) Fetch(hostname string, gen func() (*tls.Certificate, error)) (*tls.Certificate, error) {
	cs.mtx.RLock()
	entry, ok := cs.certs[hostname]
	cs.mtx.RUnlock()

	if ok && time.Since(entry.timestamp) < cs.ttl {
		return entry.cert, nil
	}

	cert, err := gen()
	if err != nil {
		return nil, err
	}

	cs.mtx.Lock()
	cs.evictExpired()

	if len(cs.certs) >= cs.maxSize {
		cs.evictOldest()
	}

	cs.certs[hostname] = &CertEntry{
		cert:      cert,
		timestamp: time.Now(),
	}
	cs.mtx.Unlock()

	return cert, nil
}

func (cs *CertStorage) evictExpired() {
	now := time.Now()
	for hostname, entry := range cs.certs {
		if now.Sub(entry.timestamp) >= cs.ttl {
			delete(cs.certs, hostname)
		}
	}
}

func (cs *CertStorage) evictOldest() {
	if len(cs.certs) == 0 {
		return
	}

	var oldestHost string
	var oldestTime time.Time
	first := true

	for hostname, entry := range cs.certs {
		if first || entry.timestamp.Before(oldestTime) {
			oldestHost = hostname
			oldestTime = entry.timestamp
			first = false
		}
	}

	if oldestHost != "" {
		delete(cs.certs, oldestHost)
	}
}

func NewCertStorage() *CertStorage {
	return &CertStorage{
		certs:   make(map[string]*CertEntry),
		maxSize: 1000,           // Limit to 1000 cached certificates
		ttl:     24 * time.Hour, // Cache certificates for 24 hours
	}
}
