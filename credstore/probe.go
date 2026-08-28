package credstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/zalando/go-keyring"
)

// probeKeyring checks system keyring availability. Overridable in tests.
var probeKeyring = probe

// The probe entry lives in this package's own service namespace —
// probeServicePrefix plus the caller's service — publicly documented on
// StoreOptions.ProbeTimeout as reserved by credstore, so probing never
// touches the caller's real service and a colliding consumer would have to
// deliberately adopt this package's declared namespace.
//
// Within that namespace the account is per process: probeKeyPrefix plus the
// pid. Concurrent invocations must never share a keychain item, because on
// darwin `security add-generic-password -U` is find-then-create inside the
// security tool, and a peer's delete/add landing in that window fails the
// add with errSecDuplicateItem on a perfectly healthy keychain — twenty
// concurrent probes of one shared item lost 190 of 200. Distinct pids give
// each in-flight process its own item, and NewStore serializes probes within
// a process (probeMu), so no two probes ever touch the same entry.
//
// The account is still deterministic for a given pid rather than random:
// go-keyring has no list API, so an entry leaked by an abandoned probe (a
// timed-out probe whose blocked Set completes after the process exits, or a
// darwin cleanup cut short) would be permanently unfindable under a random
// name. Under the pid-derived name, the next process to reuse that pid
// overwrites the leftover with its own probe and removes it — leaks still
// self-heal, on pid reuse instead of on the very next run.
const (
	probeServicePrefix = "credstore.probe."
	probeKeyPrefix     = "__probe__."
)

// probeMu serializes probes within a process. The probe account is unique
// per process, not per probe, so two stores constructed concurrently in one
// process would otherwise share an item and reintroduce the race above.
var probeMu sync.Mutex

// keyring operations, extracted as vars so tests can observe the entry
// probeDirect writes and removes without a live keyring.
var (
	keyringSet    = keyring.Set
	keyringDelete = keyring.Delete
)

// probeService derives the reserved namespace the probe entry lives in.
func probeService(serviceName string) string {
	return probeServicePrefix + serviceName
}

// probeKey derives this process's probe account.
func probeKey() string {
	return probeKeyPrefix + strconv.Itoa(os.Getpid())
}

// probe writes and removes a throwaway keyring entry to check availability.
// A zero or negative timeout probes unbounded, matching historical behavior.
// A positive timeout bounds the probe; on platforms where the probe runs a
// child process (darwin), the child is killed when the timeout expires. A
// probe that hits the bound reports the timeout by name, since that reason
// reaches users through Store.FallbackWarning and Load errors.
func probe(serviceName string, timeout time.Duration) error {
	service, key := probeService(serviceName), probeKey()
	if timeout <= 0 {
		return probeDirect(service, key)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := probeBounded(ctx, service, key)
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("keyring probe timed out after %s: %w", timeout, err)
	}
	return err
}

// probeDirect probes via go-keyring, which has no cancellation path.
func probeDirect(serviceName, key string) error {
	if err := keyringSet(serviceName, key, "probe"); err != nil {
		return err
	}
	_ = keyringDelete(serviceName, key)
	return nil
}
