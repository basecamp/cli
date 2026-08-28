package credstore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
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
	assert.ErrorIs(t, store.ProbeError(), context.DeadlineExceeded)
	assert.Contains(t, store.FallbackWarning(), "system keyring unavailable (context deadline exceeded)")
}

// Regression: a probe failure silently demoted reads to the plaintext file,
// and a miss there reported "credentials not found" — indistinguishable from
// never having logged in, when the credentials sat safely in the keyring
// this process merely failed to reach. The store must keep the probe error
// and name it wherever the fallback shows: the warning and Load's error.
func TestProbeFailureIsReportedOnLoad(t *testing.T) {
	dir := t.TempDir()
	probeErr := errors.New("User interaction is not allowed. (exit status 36)")
	stubProbe(t, func(string, time.Duration) error { return probeErr })

	store := NewStore(StoreOptions{ServiceName: "test", FallbackDir: dir})
	credentialsPath := filepath.Join(dir, "credentials.json")

	assert.Same(t, probeErr, store.ProbeError())
	assert.Equal(t, "system keyring unavailable ("+probeErr.Error()+"), credentials stored in plaintext at "+credentialsPath,
		store.FallbackWarning())

	_, err := store.Load("profile:work")
	require.Error(t, err)
	assert.ErrorIs(t, err, probeErr)
	assert.ErrorContains(t, err, "credentials not found for profile:work")
	assert.ErrorContains(t, err, "system keyring unavailable ("+probeErr.Error()+")")
	assert.ErrorContains(t, err, "fell back to "+credentialsPath)
}

// The file fallback keeps working after a failed probe — it is the only
// storage on hosts with no keyring at all — and a hit there is not an error.
func TestProbeFailureStillReadsFallbackFile(t *testing.T) {
	dir := t.TempDir()
	stubProbe(t, func(string, time.Duration) error { return errors.New("no keyring") })

	store := NewStore(StoreOptions{ServiceName: "test", FallbackDir: dir})

	require.NoError(t, store.Save("mykey", []byte(`{"token":"abc123"}`)))
	data, err := store.Load("mykey")
	require.NoError(t, err)
	assert.JSONEq(t, `{"token":"abc123"}`, string(data))
}

// File storage the caller asked for is not a fallback: no probe ran, so
// there is no probe error to report and nothing to warn about.
func TestRequestedFileStorageReportsNoProbeFailure(t *testing.T) {
	dir := t.TempDir()
	stubProbe(t, func(string, time.Duration) error {
		t.Error("probe should not run when file storage is requested")
		return nil
	})

	store := NewStore(StoreOptions{ServiceName: "test", ForceFile: true, FallbackDir: dir})

	assert.NoError(t, store.ProbeError())
	assert.Empty(t, store.FallbackWarning())
	_, err := store.Load("mykey")
	assert.EqualError(t, err, "credentials not found for mykey")
}

// The probe entry is unique per process, not per probe, so stores built
// concurrently within one process must not probe at the same time — they
// would share the entry and reintroduce the cross-process race in-process.
func TestNewStoreSerializesProbes(t *testing.T) {
	var inFlight, maxInFlight atomic.Int32
	stubProbe(t, func(string, time.Duration) error {
		n := inFlight.Add(1)
		defer inFlight.Add(-1)
		for {
			seen := maxInFlight.Load()
			if n <= seen || maxInFlight.CompareAndSwap(seen, n) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		return nil
	})

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() { NewStore(StoreOptions{ServiceName: "test", FallbackDir: t.TempDir()}) })
	}
	wg.Wait()

	assert.Equal(t, int32(1), maxInFlight.Load(), "probes must run one at a time within a process")
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
