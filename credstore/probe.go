package credstore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/zalando/go-keyring"
)

// probeKeyring checks system keyring availability. Overridable in tests.
var probeKeyring = probe

// probe writes and removes a throwaway keyring entry to check availability.
// A zero timeout probes unbounded, matching historical behavior. A positive
// timeout bounds the probe; on platforms where the probe runs a child
// process (darwin), the child is killed when the timeout expires.
func probe(serviceName string, timeout time.Duration) error {
	// Probe with a random key to avoid collisions.
	probeKey := probeKeyName()
	if timeout <= 0 {
		return probeDirect(serviceName, probeKey)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return probeBounded(ctx, serviceName, probeKey)
}

// probeDirect probes via go-keyring, which has no cancellation path.
func probeDirect(serviceName, probeKey string) error {
	if err := keyring.Set(serviceName, probeKey, "probe"); err != nil {
		return err
	}
	_ = keyring.Delete(serviceName, probeKey)
	return nil
}

func probeKeyName() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "__probe_" + hex.EncodeToString(b)
}
