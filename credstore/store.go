package credstore

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

// StoreOptions configures credential storage.
type StoreOptions struct {
	// ServiceName is the keyring service name (e.g., "basecamp", "fizzy").
	ServiceName string

	// DisableEnvVar is the env var name that disables keyring (e.g., "BASECAMP_NO_KEYRING").
	// When set to any non-empty value, forces file-based storage.
	DisableEnvVar string

	// FallbackDir is the directory for file-based credential storage.
	FallbackDir string
}

// Store handles credential storage with keyring preference and file fallback.
type Store struct {
	serviceName     string
	useKeyring      bool
	fallbackDir     string
	fallbackWarning string
}

// NewStore creates a credential store. It probes the system keyring
// and falls back to file storage if unavailable.
func NewStore(opts StoreOptions) *Store {
	if opts.DisableEnvVar != "" && os.Getenv(opts.DisableEnvVar) != "" {
		return &Store{serviceName: opts.ServiceName, useKeyring: false, fallbackDir: opts.FallbackDir}
	}

	// Probe keyring with a random key to avoid collisions.
	probeKey := probeKeyName()
	err := keyring.Set(opts.ServiceName, probeKey, "probe")
	if err == nil {
		_ = keyring.Delete(opts.ServiceName, probeKey)
		return &Store{serviceName: opts.ServiceName, useKeyring: true, fallbackDir: opts.FallbackDir}
	}

	return &Store{
		serviceName:     opts.ServiceName,
		useKeyring:      false,
		fallbackDir:     opts.FallbackDir,
		fallbackWarning: fmt.Sprintf("system keyring unavailable, credentials stored in plaintext at %s", filepath.Join(opts.FallbackDir, "credentials.json")),
	}
}

// FallbackWarning returns a warning message if the store fell back to file
// storage, or empty string if using keyring.
func (s *Store) FallbackWarning() string {
	return s.fallbackWarning
}

func probeKeyName() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "__probe_" + hex.EncodeToString(b)
}

func (s *Store) key(name string) string {
	return fmt.Sprintf("%s::%s", s.serviceName, name)
}

// Load retrieves credentials for the given key.
func (s *Store) Load(key string) ([]byte, error) {
	if s.useKeyring {
		data, err := keyring.Get(s.serviceName, s.key(key))
		if err != nil {
			return nil, fmt.Errorf("credentials not found: %w", err)
		}
		return []byte(data), nil
	}
	return s.loadFromFile(key)
}

// Save stores credentials for the given key.
func (s *Store) Save(key string, data []byte) error {
	if s.useKeyring {
		return keyring.Set(s.serviceName, s.key(key), string(data))
	}
	return s.saveToFile(key, data)
}

// Delete removes credentials for the given key.
func (s *Store) Delete(key string) error {
	if s.useKeyring {
		return keyring.Delete(s.serviceName, s.key(key))
	}
	return s.deleteFromFile(key)
}

// UsingKeyring returns true if the store is using the system keyring.
func (s *Store) UsingKeyring() bool {
	return s.useKeyring
}
