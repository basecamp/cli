package credstore

import (
	"context"
	"errors"
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
// therefore treats a failed write against evidence of a responsive keyring
// (a duplicate-item error, or a read the keyring answers) as availability,
// not as grounds for the file fallback — see probeDirect and the darwin
// probeBounded.
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
// error with no output — so disambiguate with a read: a keyring that
// answers Get, with the entry present (the peer's probe) or cleanly absent
// (ErrNotFound — the peer already cleaned up), is demonstrably responsive
// and usable. Only a keyring that fails both the write and the read is
// reported unavailable.
func probeDirect(serviceName, key string) error {
	if err := keyringSet(serviceName, key, "probe"); err != nil {
		if _, getErr := keyringGet(serviceName, key); getErr == nil || errors.Is(getErr, keyring.ErrNotFound) {
			_ = keyringDelete(serviceName, key)
			return nil
		}
		return err
	}
	_ = keyringDelete(serviceName, key)
	return nil
}
