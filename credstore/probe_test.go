package credstore

import (
	"errors"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Pins the probe-entry contract on every platform, not just darwin.
//
// Per probe: concurrent probes must never share a keychain item — on darwin
// `add-generic-password -U` is find-then-create, and a peer's delete/add
// landing in that window fails the add with errSecDuplicateItem on a
// healthy keychain, silently demoting the loser to the file fallback. The
// pid separates processes; the sequence separates probes within one,
// including a timed-out probe's abandoned worker from a later probe.
//
// Deterministic, not random: a probe abandoned mid-flight can leave at most
// one entry — this account, reserved service — which the next process
// reusing the pid overwrites and removes. The names are publicly documented
// on StoreOptions.ProbeTimeout.
func TestProbeContractIsPerProbeAndReserved(t *testing.T) {
	assert.Equal(t, "credstore.probe.svc", probeService("svc"))

	first, second := probeKey(), probeKey()
	assert.Regexp(t, probeKeyPattern(), first)
	assert.Regexp(t, probeKeyPattern(), second)
	assert.NotEqual(t, first, second, "each probe gets its own account")
}

// probeKeyPattern matches any probe account this process derives.
func probeKeyPattern() string {
	return `^__probe__\.` + strconv.Itoa(os.Getpid()) + `\.\d+$`
}

// recordKeyringOps replaces probeDirect's keyring operations for one test,
// recording the entry each touched.
func recordKeyringOps(t *testing.T, setErr error) (set, deleted *[2]string) {
	t.Helper()
	set, deleted = new([2]string), new([2]string)
	restoreSet, restoreDelete := keyringSet, keyringDelete
	keyringSet = func(service, key, _ string) error {
		*set = [2]string{service, key}
		return setErr
	}
	keyringDelete = func(service, key string) error {
		*deleted = [2]string{service, key}
		return nil
	}
	t.Cleanup(func() { keyringSet, keyringDelete = restoreSet, restoreDelete })
	return set, deleted
}

// The unbounded probe goes through go-keyring rather than the security
// binary, so it must derive the same per-probe entry as the bounded one: a
// fixed account on either path would put that path back in the race.
func TestProbeDirectUsesPerProbeEntry(t *testing.T) {
	set, deleted := recordKeyringOps(t, nil)

	assert.NoError(t, probe("svc", 0))
	assert.Equal(t, "credstore.probe.svc", set[0])
	assert.Regexp(t, probeKeyPattern(), set[1])
	assert.Equal(t, *set, *deleted, "cleanup should remove exactly the entry the probe wrote")
}

func TestProbeDirectFailureSkipsCleanup(t *testing.T) {
	setErr := errors.New("no keyring provider")
	_, deleted := recordKeyringOps(t, setErr)

	assert.ErrorIs(t, probe("svc", 0), setErr)
	assert.Zero(t, *deleted, "a failed write has nothing to clean up")
}
