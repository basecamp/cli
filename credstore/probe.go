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
// self-heal on the following run.
//
// The fixed name makes concurrent probes race on one shared item. The
// losing Delete just fails, which is ignored — but on darwin the losing Set
// can fail too: `security add-generic-password -U` is find-then-create
// inside the security tool, so a peer's delete/add interleaving surfaces
// errSecDuplicateItem even though the keychain is healthy. Every probe
// therefore treats a lost write race backed by write evidence — a
// duplicate-item error, a successful retry, or the peer's completed write —
// as availability, not as grounds for the file fallback; see probeDirect
// and the darwin probeBounded.
const (
	probeServicePrefix = "credstore.probe."
	probeKey           = "__probe__"
)

// keyring operations, extracted as vars so tests can exercise probeDirect's
// write-race disambiguation (go-keyring's mock cannot fail Set while
// answering Get).
var (
	keyringSet    = keyring.Set
	keyringGet    = keyring.Get
	keyringDelete = keyring.Delete
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
//
// A failed Set is not yet an unavailable keyring: concurrent probes share
// one fixed-name entry, and on darwin a peer's delete/add interleaving makes
// the write lose with errSecDuplicateItem (see the probeKey comment). The
// write error alone cannot be classified — go-keyring returns a bare exit
// error with no output — so recovery demands fresh evidence the keyring
// accepts writes, never a mere read answer (a read-only keyring cleanly
// misses a Get of the absent probe entry, and reporting it available would
// break every later Save). Two forms of write evidence qualify:
//
//   - An immediate retry of the Set succeeds — the contended entry has
//     settled (present, so darwin's -U updates in place; absent, so a plain
//     create lands) and this process demonstrably wrote.
//   - The retry also loses, but Get finds the entry — a peer process of the
//     same uid completed exactly this write moments ago, which is what
//     sustained churn from concurrent probes looks like.
//
// A keyring that fails both writes and cannot show a peer's is reported
// unavailable with the original write error.
func probeDirect(serviceName, key string) error {
	err := keyringSet(serviceName, key, "probe")
	if err != nil {
		if retryErr := keyringSet(serviceName, key, "probe"); retryErr != nil {
			if _, getErr := keyringGet(serviceName, key); getErr != nil {
				return err
			}
		}
	}
	_ = keyringDelete(serviceName, key)
	return nil
}
