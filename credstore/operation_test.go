package credstore

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
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

func TestKeyringOperationResultRejectsExpiredDeadline(t *testing.T) {
	for _, operation := range []string{"read", "write", "delete"} {
		for _, timerDelivered := range []bool{true, false} {
			name := operation + "/timer-pending"
			if timerDelivered {
				name = operation + "/timer-delivered"
			}
			t.Run(name, func(t *testing.T) {
				stubProbe(t, func(string, time.Duration) error { return nil })
				var calls atomic.Int32
				swapKeyringOperations(t, nil,
					func(string, string, string) error { calls.Add(1); return nil }, nil)
				store := NewStore(StoreOptions{ServiceName: "test", OperationTimeout: time.Second})

				deadline := time.Now().Add(-time.Second)
				var ctx context.Context
				if timerDelivered {
					var cancel context.CancelFunc
					ctx, cancel = context.WithDeadline(context.Background(), deadline)
					t.Cleanup(cancel)
					require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
				} else {
					parent, cancel := context.WithCancel(context.Background())
					t.Cleanup(cancel)
					ctx = pendingDeadlineContext{Context: parent, deadline: deadline}
					require.NoError(t, ctx.Err())
				}

				// Model selecting the provider's successful result after expiry,
				// whether or not the deadline timer has delivered its signal yet.
				store.operationGate <- struct{}{}
				data, err := store.keyringOperationResult(ctx, operation, []byte("late-token"), nil)
				<-store.operationGate
				require.ErrorIs(t, err, context.DeadlineExceeded)
				assert.Empty(t, data)
				assert.ErrorContains(t, err, "may have completed")
				assert.ErrorIs(t, store.Save("work", []byte("retry")), context.DeadlineExceeded)
				assert.Zero(t, calls.Load(), "a late result must not permit another provider call")
				assert.True(t, store.UsingKeyring())
			})
		}
	}
}

// pendingDeadlineContext models an expired deadline whose timer callback has
// not yet closed Done or set Err, as can happen when the scheduler is busy.
type pendingDeadlineContext struct {
	context.Context
	deadline time.Time
}

func (c pendingDeadlineContext) Deadline() (time.Time, bool) {
	return c.deadline, true
}

func TestConcurrentKeyringCallsBehindStalledProvider(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stubProbe(t, func(string, time.Duration) error { return nil })
		file := NewStore(StoreOptions{ForceFile: true, FallbackDir: t.TempDir()})
		require.NoError(t, file.Save("work", []byte(`{"access_token":"stale-file-token"}`)))
		before, err := os.ReadFile(file.credentialsPath())
		require.NoError(t, err)

		started, release := make(chan struct{}), make(chan struct{})
		var startedOnce, releaseOnce sync.Once
		var calls atomic.Int32
		var providers sync.WaitGroup
		block := func() {
			providers.Add(1)
			defer providers.Done()
			calls.Add(1)
			startedOnce.Do(func() { close(started) })
			<-release
		}
		swapKeyringOperations(t,
			func(string, string) (string, error) { block(); return "keyring-token", nil },
			func(string, string, string) error { block(); return nil },
			func(string, string) error { block(); return nil },
		)
		t.Cleanup(func() {
			releaseOnce.Do(func() { close(release) })
			providers.Wait()
		})

		const queued = 64
		const timeout = 40 * time.Millisecond
		store := NewStore(StoreOptions{ServiceName: "test", FallbackDir: file.fallbackDir, OperationTimeout: timeout})
		type result struct {
			data []byte
			err  error
		}
		results := make(chan result, queued+1)
		go func() { results <- result{err: store.Save("work", []byte("new-token"))} }()
		<-started
		synctest.Wait()

		for i := range queued {
			go func() {
				var r result
				switch i % 3 {
				case 0:
					r.data, r.err = store.Load("work")
				case 1:
					r.err = store.Save("work", []byte("queued-token"))
				case 2:
					r.err = store.Delete("work")
				}
				results <- r
			}()
		}
		// All queued callers are blocked behind the actual provider call
		// before fake time advances to their deadlines.
		synctest.Wait()
		assert.Empty(t, results)
		assert.EqualValues(t, 1, calls.Load())
		time.Sleep(timeout)
		synctest.Wait()
		require.Len(t, results, queued+1, "every caller must return while the provider is still blocked")
		for range queued + 1 {
			r := <-results
			assert.ErrorIs(t, r.err, context.DeadlineExceeded)
			assert.Empty(t, r.data, "a queued read must not return stale fallback credentials")
		}
		assert.EqualValues(t, 1, calls.Load())
		assert.True(t, store.UsingKeyring())
		assert.Empty(t, store.FallbackWarning())

		// A late successful write must not revive the store or permit any
		// queued read/write/delete to reach the provider afterward.
		releaseOnce.Do(func() { close(release) })
		providers.Wait()
		synctest.Wait()
		data, err := store.Load("work")
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Empty(t, data)
		assert.ErrorIs(t, store.Save("work", []byte("retry-token")), context.DeadlineExceeded)
		assert.ErrorIs(t, store.Delete("work"), context.DeadlineExceeded)
		assert.EqualValues(t, 1, calls.Load())
		after, err := os.ReadFile(file.credentialsPath())
		require.NoError(t, err)
		assert.Equal(t, before, after, "the fallback file must remain untouched")
	})
}
