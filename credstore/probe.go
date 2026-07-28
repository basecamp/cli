package credstore

import (
	"context"
	"time"

	"github.com/zalando/go-keyring"
)

// probeKeyring checks system keyring availability. Overridable in tests.
var probeKeyring = probe

// The probe entry lives in this package's own service namespace —
// probeServicePrefix plus the caller's service — publicly documented on
// StoreOptions.ProbeTimeout as reserved by credstore, so probing never
// touches the caller's real service and a colliding consumer would have to
// deliberately adopt this package's declared namespace. Within it, the key
// is deliberately deterministic, not random: go-keyring has no list API, so
// an entry leaked by an abandoned probe (a timed-out probe whose blocked Set
// completes after the process exits, or a darwin cleanup cut short) would be
// permanently unfindable under a random name. Under a fixed name, the next
// probe's Set overwrites the leftover and its Delete removes it — leaks
// self-heal on the following run. Concurrent probes sharing the name are
// harmless: Set results are unaffected, and the losing Delete just fails,
// which is ignored.
const (
	probeServicePrefix = "credstore.probe."
	probeKey           = "__probe__"
)

// probeService derives the reserved namespace the probe entry lives in.
func probeService(serviceName string) string {
	return probeServicePrefix + serviceName
}

// probe writes and removes a throwaway keyring entry to check availability.
// A zero or negative timeout probes unbounded, matching historical behavior.
// A positive timeout bounds the probe; on platforms where the probe runs a
// child process (darwin), the child is killed when the timeout expires.
func probe(serviceName string, timeout time.Duration) error {
	service := probeService(serviceName)
	if timeout <= 0 {
		return probeDirect(service, probeKey)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return probeBounded(ctx, service, probeKey)
}

// probeDirect probes via go-keyring, which has no cancellation path.
func probeDirect(serviceName, key string) error {
	if err := keyring.Set(serviceName, key, "probe"); err != nil {
		return err
	}
	_ = keyring.Delete(serviceName, key)
	return nil
}
