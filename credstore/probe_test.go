package credstore

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProbeKeyName(t *testing.T) {
	assert.True(t, strings.HasPrefix(probeKeyName(), "__probe_"))
	assert.NotEqual(t, probeKeyName(), probeKeyName(), "keys must not repeat")
}

// Regression: a failed or short rand read must not yield a predictable probe
// key — cleanup deletes the probe entry, so a colliding key could delete a
// real credential.
func TestProbeKeyNameRandFailure(t *testing.T) {
	restore := randRead
	randRead = func([]byte) (int, error) { return 0, errors.New("entropy exhausted") }
	t.Cleanup(func() { randRead = restore })

	key := probeKeyName()
	assert.Regexp(t, `^__probe_\d+_\d+_\d+$`, key, "fallback should key on PID + time + sequence, not the zeroed rand buffer")
	assert.NotEqual(t, key, probeKeyName(), "fallback keys must not repeat")
}

func TestProbeKeyNameShortRead(t *testing.T) {
	restore := randRead
	randRead = func(b []byte) (int, error) { return len(b) / 2, nil }
	t.Cleanup(func() { randRead = restore })

	key := probeKeyName()
	assert.Regexp(t, `^__probe_\d+_\d+_\d+$`, key, "a short read must fall back, not leak a half-zeroed key")
}
