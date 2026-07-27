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

// Regression: a failed rand read must not yield a predictable probe key —
// cleanup deletes the probe entry, so a colliding key could delete a real
// credential.
func TestProbeKeyNameRandFailure(t *testing.T) {
	restore := randRead
	randRead = func([]byte) (int, error) { return 0, errors.New("entropy exhausted") }
	t.Cleanup(func() { randRead = restore })

	key := probeKeyName()
	assert.True(t, strings.HasPrefix(key, "__probe_"))
	assert.NotEqual(t, "__probe_"+strings.Repeat("0", 16), key, "fallback must not be the zeroed rand buffer")
	assert.NotEqual(t, key, probeKeyName(), "fallback keys must not repeat")
}
