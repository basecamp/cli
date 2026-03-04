// Package profile provides named profile management for CLI tools.
//
// A profile bundles a base URL with optional app-specific settings,
// letting users and agents target different environments or accounts
// with --profile or APP_PROFILE without env var juggling.
package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

// Profile is a named environment configuration.
type Profile struct {
	Name    string                     `json:"-"`
	BaseURL string                     `json:"base_url"`
	Extra   map[string]json.RawMessage `json:"extra,omitempty"`
}

// validName matches alphanumeric + hyphen + underscore, must start with alnum.
var validName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// ValidateName checks that a profile name is well-formed.
func ValidateName(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid profile name %q: must match [a-zA-Z0-9][a-zA-Z0-9_-]*", name)
	}
	return nil
}

// CredentialKey returns the credential store key for a profile.
// With a profile: "profile:<name>". Without: the base URL.
func CredentialKey(profileName, baseURL string) string {
	if profileName != "" {
		return "profile:" + profileName
	}
	return baseURL
}

// configFile holds the on-disk JSON structure.
type configFile struct {
	Profiles       map[string]*Profile `json:"profiles,omitempty"`
	DefaultProfile string              `json:"default_profile,omitempty"`
}

// Store manages named profiles in a JSON config file.
type Store struct {
	path string
}

// NewStore creates a profile store backed by configPath (e.g.,
// ~/.config/myapp/config.json). The file and parent directory are
// created on first write.
func NewStore(configPath string) *Store {
	return &Store{path: configPath}
}

// load reads and parses the config file. Returns an empty config if
// the file doesn't exist.
func (s *Store) load() (*configFile, error) {
	data, err := os.ReadFile(s.path) //nolint:gosec // G304: path from trusted config location
	if err != nil {
		if os.IsNotExist(err) {
			return &configFile{Profiles: make(map[string]*Profile)}, nil
		}
		return nil, err
	}

	var cfg configFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("malformed config at %s: %w", s.path, err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]*Profile)
	}
	// Backfill Name field from map key.
	for name, p := range cfg.Profiles {
		p.Name = name
	}
	return &cfg, nil
}

// save writes the config file atomically.
func (s *Store) save(cfg *configFile) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(dir, "config-*.json.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Chmod(0600); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		if runtime.GOOS == "windows" {
			_ = os.Remove(s.path)
			return os.Rename(tmpPath, s.path)
		}
		os.Remove(tmpPath)
		return err
	}
	return nil
}

// List returns all profiles and the default profile name.
func (s *Store) List() (map[string]*Profile, string, error) {
	cfg, err := s.load()
	if err != nil {
		return nil, "", err
	}
	return cfg.Profiles, cfg.DefaultProfile, nil
}

// Get returns a single profile by name.
func (s *Store) Get(name string) (*Profile, error) {
	cfg, err := s.load()
	if err != nil {
		return nil, err
	}
	p, ok := cfg.Profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile %q not found", name)
	}
	return p, nil
}

// Create adds a new profile. Returns an error if it already exists.
func (s *Store) Create(p *Profile) error {
	if err := ValidateName(p.Name); err != nil {
		return err
	}
	if p.BaseURL == "" {
		return fmt.Errorf("profile %q: base_url is required", p.Name)
	}

	cfg, err := s.load()
	if err != nil {
		return err
	}
	if _, exists := cfg.Profiles[p.Name]; exists {
		return fmt.Errorf("profile %q already exists", p.Name)
	}

	cfg.Profiles[p.Name] = p

	// Auto-set default if this is the first profile.
	if len(cfg.Profiles) == 1 {
		cfg.DefaultProfile = p.Name
	}

	return s.save(cfg)
}

// Delete removes a profile by name. Clears default_profile if it
// pointed to the deleted profile.
func (s *Store) Delete(name string) error {
	cfg, err := s.load()
	if err != nil {
		return err
	}
	if _, exists := cfg.Profiles[name]; !exists {
		return fmt.Errorf("profile %q not found", name)
	}

	delete(cfg.Profiles, name)
	if cfg.DefaultProfile == name {
		cfg.DefaultProfile = ""
	}

	return s.save(cfg)
}

// SetDefault sets the default profile.
func (s *Store) SetDefault(name string) error {
	cfg, err := s.load()
	if err != nil {
		return err
	}
	if _, exists := cfg.Profiles[name]; !exists {
		return fmt.Errorf("profile %q not found", name)
	}

	cfg.DefaultProfile = name
	return s.save(cfg)
}
