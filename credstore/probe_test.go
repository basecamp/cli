package credstore

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zalando/go-keyring"
)

// Pins the leak-containment contract on every platform, not just darwin: a
// probe abandoned mid-flight (a timed-out non-darwin probe whose blocked Set
// completes after process exit, or a darwin cleanup cut short) can leave at
// most the one known entry — deterministic account, reserved service — which
// the next probe's Set overwrites and Delete removes. The names below are
// publicly documented on StoreOptions.ProbeTimeout; changing either breaks
// self-healing across versions and orphans entries written by earlier
// releases.
func TestProbeContractIsDeterministicAndReserved(t *testing.T) {
	assert.Equal(t, "credstore.probe.svc", probeService("svc"))
	assert.Equal(t, "__probe__", probeKey)
}

// stubKeyringOps replaces probeDirect's keyring operations for one test.
// go-keyring's mock cannot fail Set while answering Get, which is exactly
// the shape of the concurrent-probe write race.
func stubKeyringOps(t *testing.T, set func(string, string, string) error, get func(string, string) (string, error)) (deleted *bool) {
	t.Helper()
	deleted = new(bool)
	restoreSet, restoreGet, restoreDelete := keyringSet, keyringGet, keyringDelete
	keyringSet = set
	keyringGet = get
	keyringDelete = func(service, key string) error {
		*deleted = true
		return nil
	}
	t.Cleanup(func() { keyringSet, keyringGet, keyringDelete = restoreSet, restoreGet, restoreDelete })
	return deleted
}

// Regression: concurrent probes share one fixed-name entry, and a losing
// write (darwin surfaces errSecDuplicateItem through go-keyring as a bare
// exit error) must not demote a healthy keyring to the file fallback.
// Recovery requires fresh write evidence — this process's retry landing, or
// a peer's completed write sitting in the keyring — never a bare read
// answer, which a read-only keyring could also give.
func TestProbeDirectWriteRaceRecovery(t *testing.T) {
	setErr := errors.New("exit status 45")

	t.Run("retry lands once the churned entry settles", func(t *testing.T) {
		calls := 0
		deleted := stubKeyringOps(t,
			func(_, _, _ string) error {
				calls++
				if calls == 1 {
					return setErr
				}
				return nil
			},
			func(_, _ string) (string, error) { t.Fatal("no read needed when the retry lands"); return "", nil })

		assert.NoError(t, probeDirect("credstore.probe.svc", probeKey))
		assert.Equal(t, 2, calls)
		assert.True(t, *deleted, "cleanup should remove the retried entry")
	})

	t.Run("retry loses too but the peer's write is present", func(t *testing.T) {
		deleted := stubKeyringOps(t,
			func(_, _, _ string) error { return setErr },
			func(_, _ string) (string, error) { return "probe", nil })

		assert.NoError(t, probeDirect("credstore.probe.svc", probeKey))
		assert.True(t, *deleted, "cleanup should remove the peer's entry")
	})

	t.Run("read-only keyring: writes fail, probe entry absent", func(t *testing.T) {
		stubKeyringOps(t,
			func(_, _, _ string) error { return setErr },
			func(_, _ string) (string, error) { return "", keyring.ErrNotFound })

		assert.ErrorIs(t, probeDirect("credstore.probe.svc", probeKey), setErr)
	})

	t.Run("keyring fails the read too: genuinely unavailable", func(t *testing.T) {
		stubKeyringOps(t,
			func(_, _, _ string) error { return setErr },
			func(_, _ string) (string, error) { return "", errors.New("no keyring provider") })

		assert.ErrorIs(t, probeDirect("credstore.probe.svc", probeKey), setErr)
	})
}

func TestProbeDirectSuccessCleansUp(t *testing.T) {
	deleted := stubKeyringOps(t,
		func(_, _, _ string) error { return nil },
		func(_, _ string) (string, error) { t.Fatal("no read needed when the write succeeds"); return "", nil })

	assert.NoError(t, probeDirect("credstore.probe.svc", probeKey))
	assert.True(t, *deleted)
}
