package credstore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStore(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TEST_NO_KEYRING", "1")

	store := NewStore(StoreOptions{
		ServiceName:   "test",
		DisableEnvVar: "TEST_NO_KEYRING",
		FallbackDir:   dir,
	})

	assert.False(t, store.UsingKeyring())

	// Save
	err := store.Save("mykey", []byte(`{"token":"abc123"}`))
	require.NoError(t, err)

	// Load
	data, err := store.Load("mykey")
	require.NoError(t, err)
	assert.JSONEq(t, `{"token":"abc123"}`, string(data))

	// Verify file permissions
	info, err := os.Stat(filepath.Join(dir, "credentials.json"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// Delete
	err = store.Delete("mykey")
	require.NoError(t, err)

	_, err = store.Load("mykey")
	assert.Error(t, err)
}

func TestFileStoreMultipleKeys(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TEST_NO_KEYRING", "1")

	store := NewStore(StoreOptions{
		ServiceName:   "test",
		DisableEnvVar: "TEST_NO_KEYRING",
		FallbackDir:   dir,
	})

	store.Save("key1", []byte(`{"a":1}`))
	store.Save("key2", []byte(`{"b":2}`))

	d1, _ := store.Load("key1")
	d2, _ := store.Load("key2")
	assert.JSONEq(t, `{"a":1}`, string(d1))
	assert.JSONEq(t, `{"b":2}`, string(d2))

	// Delete one, other persists
	store.Delete("key1")
	_, err := store.Load("key1")
	assert.Error(t, err)
	d2, _ = store.Load("key2")
	assert.JSONEq(t, `{"b":2}`, string(d2))
}

func TestLoadNonexistent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TEST_NO_KEYRING", "1")

	store := NewStore(StoreOptions{
		ServiceName:   "test",
		DisableEnvVar: "TEST_NO_KEYRING",
		FallbackDir:   dir,
	})

	_, err := store.Load("nonexistent")
	assert.Error(t, err)
}
