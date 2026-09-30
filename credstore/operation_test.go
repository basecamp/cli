package credstore

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func swapKeyringOperations(t *testing.T, get func(string, string) (string, error), set func(string, string, string) error, del func(string, string) error) {
	t.Helper()
	oldGet, oldSet, oldDelete := keyringGet, keyringSet, keyringDelete
	keyringGet, keyringSet, keyringDelete = get, set, del
	t.Cleanup(func() { keyringGet, keyringSet, keyringDelete = oldGet, oldSet, oldDelete })
}

// A responsive probe does not promise that the next real keyring operation
// will respond. Timing out must not serve a stale file, write plaintext, or
// start another operation alongside a write/delete whose outcome is unknown.
func TestKeyringOperationTimeout(t *testing.T) {
	for _, operation := range []string{"load", "save", "delete", "migrate"} {
		t.Run(operation, func(t *testing.T) {
			dir := t.TempDir()
			file := NewStore(StoreOptions{ForceFile: true, FallbackDir: dir})
			require.NoError(t, file.Save("work", []byte(`{"access_token":"stale-file-token"}`)))
			before, err := os.ReadFile(file.credentialsPath())
			require.NoError(t, err)

			stubProbe(t, func(string, time.Duration) error { return nil })
			var calls atomic.Int32
			release, finished := make(chan struct{}), make(chan struct{})
			block := func() {
				calls.Add(1)
				<-release
				close(finished)
			}
			swapKeyringOperations(t,
				func(string, string) (string, error) { block(); return "keyring-token", nil },
				func(string, string, string) error { block(); return nil },
				func(string, string) error { block(); return nil },
			)
			var releaseOnce sync.Once
			t.Cleanup(func() {
				releaseOnce.Do(func() { close(release) })
				if calls.Load() > 0 {
					<-finished
				}
			})
			store := NewStore(StoreOptions{ServiceName: "test", FallbackDir: dir, OperationTimeout: 40 * time.Millisecond})
			result := make(chan error, 1)
			go func() {
				switch operation {
				case "load":
					data, loadErr := store.Load("work")
					if len(data) != 0 {
						loadErr = errors.New("timed-out read returned credential data")
					}
					result <- loadErr
				case "save":
					result <- store.Save("work", []byte("new-token"))
				case "delete":
					result <- store.Delete("work")
				case "migrate":
					result <- store.MigrateToKeyring()
				}
			}()
			select {
			case operationErr := <-result:
				require.ErrorIs(t, operationErr, context.DeadlineExceeded)
				assert.ErrorContains(t, operationErr, "may still complete")
				assert.NotContains(t, operationErr.Error(), "not found", "a hung keyring is not a missing login")
			case <-time.After(time.Second):
				releaseOnce.Do(func() { close(release) })
				<-result
				t.Fatal("keyring operation ignored OperationTimeout")
			}

			assert.True(t, store.UsingKeyring(), "a timed-out operation must not switch to plaintext")
			assert.Empty(t, store.FallbackWarning())
			_, err = store.Load("work")
			assert.ErrorIs(t, err, context.DeadlineExceeded)
			assert.NotContains(t, err.Error(), "not found", "a hung keyring is not a missing login")
			assert.ErrorIs(t, store.Save("work", []byte("newer-token")), context.DeadlineExceeded)
			assert.ErrorIs(t, store.Delete("work"), context.DeadlineExceeded)
			assert.EqualValues(t, 1, calls.Load(), "a stalled operation poisons this store, not the process")
			after, err := os.ReadFile(file.credentialsPath())
			require.NoError(t, err)
			assert.Equal(t, before, after, "timeout must neither change nor remove the fallback file")

			// A late success is not permission to resume writes whose order
			// relative to the timed-out write/delete cannot be established.
			releaseOnce.Do(func() { close(release) })
			<-finished
			assert.ErrorIs(t, store.Save("work", []byte("late-token")), context.DeadlineExceeded)
			assert.EqualValues(t, 1, calls.Load())
		})
	}
}

func TestBoundedKeyringKeepsHealthyResultsAndErrors(t *testing.T) {
	stubProbe(t, func(string, time.Duration) error { return nil })
	providerErr := errors.New("keyring locked")
	swapKeyringOperations(t,
		func(service, key string) (string, error) {
			assert.Equal(t, "test", service)
			assert.Equal(t, "test::work", key)
			return "token", nil
		},
		func(string, string, string) error { return providerErr },
		func(string, string) error { return nil },
	)
	store := NewStore(StoreOptions{ServiceName: "test", OperationTimeout: time.Second})
	data, err := store.Load("work")
	require.NoError(t, err)
	assert.Equal(t, "token", string(data))
	assert.ErrorIs(t, store.Save("work", data), providerErr)
	assert.NoError(t, store.Delete("work"), "an ordinary provider error must not poison the store")
}

func TestWaitingForKeyringOperationAlsoHasADeadline(t *testing.T) {
	stubProbe(t, func(string, time.Duration) error { return nil })
	var calls atomic.Int32
	swapKeyringOperations(t,
		func(string, string) (string, error) { calls.Add(1); return "token", nil },
		func(string, string, string) error { return nil },
		func(string, string) error { return nil },
	)
	store := NewStore(StoreOptions{ServiceName: "test", OperationTimeout: 40 * time.Millisecond})
	store.operationGate <- struct{}{}
	_, err := store.Load("work")
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.ErrorContains(t, err, "waiting for another operation")
	assert.NotContains(t, err.Error(), "may still complete")
	assert.Zero(t, calls.Load(), "a queued call must not reach the provider after its deadline")
	<-store.operationGate
	data, err := store.Load("work")
	require.NoError(t, err, "timing out before calling the provider must not poison the store")
	assert.Equal(t, "token", string(data))
}

func TestZeroOperationTimeoutPreservesUnboundedCalls(t *testing.T) {
	stubProbe(t, func(string, time.Duration) error { return nil })
	swapKeyringOperations(t,
		func(string, string) (string, error) { return "token", nil },
		func(string, string, string) error { return nil },
		func(string, string) error { return nil },
	)
	store := NewStore(StoreOptions{ServiceName: "test"})
	data, err := store.Load("work")
	require.NoError(t, err)
	assert.Equal(t, "token", string(data))
	assert.NoError(t, store.Save("work", data))
	assert.NoError(t, store.Delete("work"))
}
