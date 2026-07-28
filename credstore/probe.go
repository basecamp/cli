package credstore

import (
	"context"
	"time"

	"github.com/zalando/go-keyring"
)

// probeKeyring checks system keyring availability. Overridable in tests.
var probeKeyring = probe

// probeKey names the throwaway entry every probe writes and removes. It is
// deliberately deterministic, not random: go-keyring has no list API, so an
// entry leaked by an abandoned probe (a timed-out probe whose blocked Set
// completes after the process exits, or a darwin cleanup cut short) would be
// permanently unfindable under a random name. Under a fixed name, the next
// probe's Set overwrites the leftover and its Delete removes it — leaks
// self-heal on the following run. Concurrent probes sharing the name are
// harmless: Set results are unaffected, and the losing Delete just fails,
// which is ignored. The name is reserved within the service's namespace.
const probeKey = "__probe__"

// probe writes and removes a throwaway keyring entry to check availability.
// A zero or negative timeout probes unbounded, matching historical behavior.
// A positive timeout bounds the probe; on platforms where the probe runs a
// child process (darwin), the child is killed when the timeout expires.
func probe(serviceName string, timeout time.Duration) error {
	if timeout <= 0 {
		return probeDirect(serviceName, probeKey)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return probeBounded(ctx, serviceName, probeKey)
}

// probeDirect probes via go-keyring, which has no cancellation path.
func probeDirect(serviceName, key string) error {
	if err := keyring.Set(serviceName, key, "probe"); err != nil {
		return err
	}
	_ = keyring.Delete(serviceName, key)
	return nil
}
