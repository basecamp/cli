package credstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stubProbe(t *testing.T, fn func(serviceName string, timeout time.Duration) error) {
	t.Helper()
	restore := probeKeyring
	probeKeyring = fn
	t.Cleanup(func() { probeKeyring = restore })
}

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

	require.NoError(t, store.Save("key1", []byte(`{"a":1}`)))
	require.NoError(t, store.Save("key2", []byte(`{"b":2}`)))

	d1, _ := store.Load("key1")
	d2, _ := store.Load("key2")
	assert.JSONEq(t, `{"a":1}`, string(d1))
	assert.JSONEq(t, `{"b":2}`, string(d2))

	// Delete one, other persists
	require.NoError(t, store.Delete("key1"))
	_, err := store.Load("key1")
	assert.Error(t, err)
	d2, _ = store.Load("key2")
	assert.JSONEq(t, `{"b":2}`, string(d2))
}

func TestForceFileSkipsProbe(t *testing.T) {
	dir := t.TempDir()
	stubProbe(t, func(string, time.Duration) error {
		t.Error("probe should not run when ForceFile is set")
		return nil
	})

	store := NewStore(StoreOptions{
		ServiceName: "test",
		ForceFile:   true,
		FallbackDir: dir,
	})

	assert.False(t, store.UsingKeyring())
	assert.Empty(t, store.FallbackWarning())

	require.NoError(t, store.Save("mykey", []byte(`{"token":"abc123"}`)))
	data, err := store.Load("mykey")
	require.NoError(t, err)
	assert.JSONEq(t, `{"token":"abc123"}`, string(data))
}

func TestProbeTimeoutFallsBackToFile(t *testing.T) {
	dir := t.TempDir()
	stubProbe(t, func(serviceName string, timeout time.Duration) error {
		assert.Equal(t, "test", serviceName)
		assert.Equal(t, 50*time.Millisecond, timeout)
		return context.DeadlineExceeded
	})

	store := NewStore(StoreOptions{
		ServiceName:  "test",
		ProbeTimeout: 50 * time.Millisecond,
		FallbackDir:  dir,
	})

	assert.False(t, store.UsingKeyring())
	assert.Contains(t, store.FallbackWarning(), "system keyring unavailable")
}

func TestZeroValueOptionsProbeUnbounded(t *testing.T) {
	dir := t.TempDir()
	probed := false
	stubProbe(t, func(serviceName string, timeout time.Duration) error {
		probed = true
		assert.Zero(t, timeout)
		return nil
	})

	store := NewStore(StoreOptions{
		ServiceName: "test",
		FallbackDir: dir,
	})

	assert.True(t, probed)
	assert.True(t, store.UsingKeyring())
	assert.Empty(t, store.FallbackWarning())
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
