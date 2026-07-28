package credstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
