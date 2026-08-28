package credstore

import (
	"fmt"
	"os"
	"time"

	"github.com/zalando/go-keyring"
)

// StoreOptions configures credential storage.
type StoreOptions struct {
	// ServiceName is the keyring service name (e.g., "basecamp", "fizzy").
	ServiceName string

	// DisableEnvVar names an environment variable (e.g., "BASECAMP_NO_KEYRING").
	// When that variable is set to any non-empty value in the process
	// environment, the store uses file-based storage without probing the
	// keyring. An empty DisableEnvVar field disables this check entirely.
	DisableEnvVar string

	// ForceFile forces file-based storage without probing the keyring —
	// the programmatic equivalent of the DisableEnvVar environment variable
	// being set.
	ForceFile bool

	// ProbeTimeout bounds the keyring availability probe. Zero or negative
	// means no bound, matching historical behavior. When the probe times
	// out, the store falls back to file storage as if the probe had failed.
	// Probing writes and removes a throwaway entry under the dedicated
	// keyring service "credstore.probe.<ServiceName>" (account
	// "__probe__.<pid>") — a namespace reserved by this package — never
	// under ServiceName itself, so a probe cannot touch real credentials.
	// The account is per process so concurrent invocations never contend
	// for one keychain item, and probes within a process are serialized.
	//
	// On darwin, removal of the throwaway probe entry runs synchronously
	// after a successful probe with a short budget of its own, so worst-case
	// construction there is ProbeTimeout plus that cleanup bound (five
	// seconds; typically milliseconds, since cleanup only runs when the
	// keyring just proved responsive). That cleanup is deliberately not
	// detached: Go does not wait for goroutines at process exit, and a
	// short-lived CLI would leak a probe entry per invocation. On other
	// platforms cleanup is part of the probe itself and adds no extra time.
	//
	// Limitation: on darwin, a positive timeout probes via the security
	// binary directly — the whole point is operating below go-keyring's
	// uncancellable exec layer — so it cannot observe go-keyring's mocked
	// provider (keyring.MockInit), which go-keyring does not expose. Tests
	// that mock the keyring should use a zero ProbeTimeout (the unbounded
	// probe goes through go-keyring and honors the mock) or ForceFile.
	ProbeTimeout time.Duration

	// FallbackDir is the directory for file-based credential storage.
	FallbackDir string
}

// Store handles credential storage with keyring preference and file fallback.
type Store struct {
	serviceName string
	useKeyring  bool
	fallbackDir string

	// probeErr is why the keyring probe failed when the store fell back to
	// file storage against the caller's wishes. Nil when the keyring is in
	// use and when file storage was requested (ForceFile, DisableEnvVar):
	// a requested fallback is not a degradation and warrants no warning.
	probeErr error
}

// NewStore creates a credential store. It probes the system keyring
// and falls back to file storage if unavailable.
func NewStore(opts StoreOptions) *Store {
	if opts.ForceFile || (opts.DisableEnvVar != "" && os.Getenv(opts.DisableEnvVar) != "") {
		return &Store{serviceName: opts.ServiceName, useKeyring: false, fallbackDir: opts.FallbackDir}
	}

	probeMu.Lock()
	err := probeKeyring(opts.ServiceName, opts.ProbeTimeout)
	probeMu.Unlock()

	return &Store{
		serviceName: opts.ServiceName,
		useKeyring:  err == nil,
		fallbackDir: opts.FallbackDir,
		probeErr:    err,
	}
}

// ProbeError returns why the keyring probe failed and the store fell back
// to file storage, or nil when the keyring is in use or file storage was
// requested outright.
func (s *Store) ProbeError() error {
	return s.probeErr
}

// FallbackWarning returns a warning message if the store fell back to file
// storage because the keyring probe failed, or empty string otherwise. The
// message names the probe failure so a fallback is never silent about its
// cause. Callers decide where to surface it; surface it on reads as well as
// writes, since a fallback read may return a stale file left over from an
// earlier fallback rather than the credentials the keyring holds.
func (s *Store) FallbackWarning() string {
	if s.probeErr == nil {
		return ""
	}
	return fmt.Sprintf("system keyring unavailable (%v), credentials stored in plaintext at %s", s.probeErr, s.credentialsPath())
}

func (s *Store) key(name string) string {
	return fmt.Sprintf("%s::%s", s.serviceName, name)
}

// Load retrieves credentials for the given key.
//
// On the file fallback, a miss is reported together with the keyring probe
// failure that caused the fallback: "credentials not found" alone reads as
// "log in again" when the truth may be that the credentials sit safely in
// the keyring and only this process could not reach it.
func (s *Store) Load(key string) ([]byte, error) {
	if s.useKeyring {
		data, err := keyring.Get(s.serviceName, s.key(key))
		if err != nil {
			return nil, fmt.Errorf("credentials not found: %w", err)
		}
		return []byte(data), nil
	}

	data, err := s.loadFromFile(key)
	if err != nil && s.probeErr != nil {
		return nil, fmt.Errorf("%w: system keyring unavailable (%w), fell back to %s", err, s.probeErr, s.credentialsPath())
	}
	return data, err
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
