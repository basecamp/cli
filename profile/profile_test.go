package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateName(t *testing.T) {
	valid := []string{"personal", "staging", "client-a", "prod_1", "A", "a1"}
	for _, name := range valid {
		assert.NoError(t, ValidateName(name), "should be valid: %q", name)
	}

	invalid := []string{"", "-start", "_start", "has space", "has.dot", "!bang"}
	for _, name := range invalid {
		assert.Error(t, ValidateName(name), "should be invalid: %q", name)
	}
}

func TestCredentialKey(t *testing.T) {
	assert.Equal(t, "profile:staging", CredentialKey("staging", "https://api.example.com"))
	assert.Equal(t, "https://api.example.com", CredentialKey("", "https://api.example.com"))
}

func TestStoreCreateAndList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := NewStore(path)

	// Empty store
	profiles, def, err := store.List()
	require.NoError(t, err)
	assert.Empty(t, profiles)
	assert.Empty(t, def)

	// Create first profile — becomes default
	err = store.Create(&Profile{Name: "personal", BaseURL: "https://api.example.com"})
	require.NoError(t, err)

	profiles, def, err = store.List()
	require.NoError(t, err)
	assert.Len(t, profiles, 1)
	assert.Equal(t, "personal", def)
	assert.Equal(t, "https://api.example.com", profiles["personal"].BaseURL)

	// Create second profile — default unchanged
	err = store.Create(&Profile{Name: "staging", BaseURL: "https://staging.example.com"})
	require.NoError(t, err)

	profiles, def, err = store.List()
	require.NoError(t, err)
	assert.Len(t, profiles, 2)
	assert.Equal(t, "personal", def)
}

func TestStoreCreateDuplicate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := NewStore(path)

	require.NoError(t, store.Create(&Profile{Name: "prod", BaseURL: "https://a.com"}))
	err := store.Create(&Profile{Name: "prod", BaseURL: "https://b.com"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestStoreCreateValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := NewStore(path)

	err := store.Create(&Profile{Name: "-bad", BaseURL: "https://a.com"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid profile name")

	err = store.Create(&Profile{Name: "good", BaseURL: ""})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "base_url is required")
}

func TestStoreGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := NewStore(path)

	require.NoError(t, store.Create(&Profile{Name: "prod", BaseURL: "https://a.com"}))

	p, err := store.Get("prod")
	require.NoError(t, err)
	assert.Equal(t, "prod", p.Name)
	assert.Equal(t, "https://a.com", p.BaseURL)

	_, err = store.Get("nonexistent")
	assert.Error(t, err)
}

func TestStoreDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := NewStore(path)

	require.NoError(t, store.Create(&Profile{Name: "a", BaseURL: "https://a.com"}))
	require.NoError(t, store.Create(&Profile{Name: "b", BaseURL: "https://b.com"}))

	// Delete the default profile — default cleared
	require.NoError(t, store.Delete("a"))

	profiles, def, err := store.List()
	require.NoError(t, err)
	assert.Len(t, profiles, 1)
	assert.Empty(t, def)

	// Delete nonexistent
	err = store.Delete("nonexistent")
	assert.Error(t, err)
}

func TestStoreSetDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := NewStore(path)

	require.NoError(t, store.Create(&Profile{Name: "a", BaseURL: "https://a.com"}))
	require.NoError(t, store.Create(&Profile{Name: "b", BaseURL: "https://b.com"}))

	require.NoError(t, store.SetDefault("b"))
	_, def, _ := store.List()
	assert.Equal(t, "b", def)

	err := store.SetDefault("nonexistent")
	assert.Error(t, err)
}

func TestStoreFilePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := NewStore(path)

	require.NoError(t, store.Create(&Profile{Name: "p", BaseURL: "https://a.com"}))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestStoreExtraFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := NewStore(path)

	extra := map[string]json.RawMessage{
		"account_id": json.RawMessage(`"12345"`),
		"scope":      json.RawMessage(`"full"`),
	}
	require.NoError(t, store.Create(&Profile{Name: "prod", BaseURL: "https://a.com", Extra: extra}))

	p, err := store.Get("prod")
	require.NoError(t, err)
	assert.Equal(t, `"12345"`, string(p.Extra["account_id"]))
	assert.Equal(t, `"full"`, string(p.Extra["scope"]))
}

// Resolution tests

func TestResolveNoProfiles(t *testing.T) {
	name, err := Resolve(ResolveOptions{})
	require.NoError(t, err)
	assert.Empty(t, name)
}

func TestResolveFlagWins(t *testing.T) {
	profiles := map[string]*Profile{
		"a": {Name: "a", BaseURL: "https://a.com"},
		"b": {Name: "b", BaseURL: "https://b.com"},
	}
	name, err := Resolve(ResolveOptions{
		FlagValue:      "b",
		EnvVar:         "a",
		DefaultProfile: "a",
		Profiles:       profiles,
	})
	require.NoError(t, err)
	assert.Equal(t, "b", name)
}

func TestResolveEnvFallback(t *testing.T) {
	profiles := map[string]*Profile{
		"a": {Name: "a", BaseURL: "https://a.com"},
		"b": {Name: "b", BaseURL: "https://b.com"},
	}
	name, err := Resolve(ResolveOptions{
		EnvVar:   "b",
		Profiles: profiles,
	})
	require.NoError(t, err)
	assert.Equal(t, "b", name)
}

func TestResolveDefaultFallback(t *testing.T) {
	profiles := map[string]*Profile{
		"a": {Name: "a", BaseURL: "https://a.com"},
		"b": {Name: "b", BaseURL: "https://b.com"},
	}
	name, err := Resolve(ResolveOptions{
		DefaultProfile: "a",
		Profiles:       profiles,
	})
	require.NoError(t, err)
	assert.Equal(t, "a", name)
}

func TestResolveAutoSelectSingle(t *testing.T) {
	profiles := map[string]*Profile{
		"only": {Name: "only", BaseURL: "https://only.com"},
	}
	name, err := Resolve(ResolveOptions{Profiles: profiles})
	require.NoError(t, err)
	assert.Equal(t, "only", name)
}

func TestResolveMultipleNoSelection(t *testing.T) {
	profiles := map[string]*Profile{
		"a": {Name: "a", BaseURL: "https://a.com"},
		"b": {Name: "b", BaseURL: "https://b.com"},
	}
	_, err := Resolve(ResolveOptions{Profiles: profiles})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple profiles")
}

func TestResolveInteractivePicker(t *testing.T) {
	profiles := map[string]*Profile{
		"a": {Name: "a", BaseURL: "https://a.com"},
		"b": {Name: "b", BaseURL: "https://b.com"},
	}
	name, err := Resolve(ResolveOptions{
		Profiles:    profiles,
		Interactive: true,
		Picker:      func(names []string) (string, error) { return "b", nil },
	})
	require.NoError(t, err)
	assert.Equal(t, "b", name)
}

func TestResolveNotFound(t *testing.T) {
	profiles := map[string]*Profile{
		"a": {Name: "a", BaseURL: "https://a.com"},
	}

	_, err := Resolve(ResolveOptions{FlagValue: "missing", Profiles: profiles})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	_, err = Resolve(ResolveOptions{EnvVar: "missing", Profiles: profiles})
	assert.Error(t, err)

	_, err = Resolve(ResolveOptions{DefaultProfile: "missing", Profiles: profiles})
	assert.Error(t, err)
}
