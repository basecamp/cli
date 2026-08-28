package credstore

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stubSecurity(t *testing.T, script string) {
	t.Helper()
	stub := filepath.Join(t.TempDir(), "security")
	require.NoError(t, os.WriteFile(stub, []byte(script), 0755))
	restore := securityPath
	securityPath = stub
	t.Cleanup(func() { securityPath = restore })
}

// stubDir returns a scratch directory whose path contains a space, so any
// stub script that embeds a path without quoting fails loudly here rather
// than only on machines with space-containing TMPDIRs.
func stubDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "dir with space")
	require.NoError(t, os.Mkdir(dir, 0755))
	return dir
}

// shQuote single-quotes s for safe embedding in generated stub scripts.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func TestProbeBoundedKillsChildOnTimeout(t *testing.T) {
	pidFile := filepath.Join(stubDir(t), "pid")
	// exec replaces the shell so the recorded PID is the hung process itself,
	// mirroring a `security -i` blocked on a locked keychain.
	stubSecurity(t, "#!/bin/sh\nPF="+shQuote(pidFile)+"\necho $$ > \"$PF.tmp\" && mv \"$PF.tmp\" \"$PF\"\nexec sleep 60\n")

	// Expire the context once the child is demonstrably hung (PID recorded),
	// so the test exercises exactly the timeout path without racing startup.
	// The watcher observes a deadline: a broken fixture cancels anyway and
	// fails fast on the missing PID file instead of hanging the test.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deadline := time.Now().Add(5 * time.Second)
	go func() {
		for time.Now().Before(deadline) {
			if _, err := os.Stat(pidFile); err == nil {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		cancel()
	}()

	err := probeBounded(ctx, "test", "__probe_timeout")
	assert.ErrorIs(t, err, context.Canceled)

	raw, err := os.ReadFile(pidFile)
	require.NoError(t, err)
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		return syscall.Kill(pid, 0) == syscall.ESRCH
	}, 5*time.Second, 10*time.Millisecond, "hung child should be killed when the context expires")
}

// argsStub emits a stub security that records each invocation's argv and
// consumes stdin, mirroring a healthy keychain.
func argsStub(t *testing.T) (argsFile string) {
	t.Helper()
	argsFile = filepath.Join(stubDir(t), "args")
	stubSecurity(t, "#!/bin/sh\nAF="+shQuote(argsFile)+"\necho \"$@\" >> \"$AF\"\ncat > /dev/null\nexit 0\n")
	return argsFile
}

// requireCleanupDelete asserts the stub recorded the add followed by the
// cleanup delete. Both are synchronous: cleanup is deliberately not detached
// (see probeCleanupTimeout), so it has completed by the time probeBounded
// returns.
func requireCleanupDelete(t *testing.T, argsFile, service, key string) {
	t.Helper()
	raw, err := os.ReadFile(argsFile)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	require.Len(t, lines, 2, "probe should add then delete the probe entry")
	assert.Equal(t, "-i", lines[0])
	assert.Equal(t, "delete-generic-password -s "+service+" -a "+key, lines[1])
}

func TestProbeBoundedSuccess(t *testing.T) {
	argsFile := argsStub(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, probeBounded(ctx, "test", "__probe_ok"))
	requireCleanupDelete(t, argsFile, "test", "__probe_ok")
}

// The probe must operate in its own service namespace so it can never touch
// a credential in the caller's real service, whatever its name — and under
// its own per-process account, so concurrent invocations never contend for
// one keychain item (see probeKey).
func TestProbeUsesIsolatedPerProcessEntry(t *testing.T) {
	argsFile := argsStub(t)

	require.NoError(t, probe("svc", 5*time.Second))
	requireCleanupDelete(t, argsFile, probeServicePrefix+"svc", "__probe__."+strconv.Itoa(os.Getpid()))
}

// A probe that hits its bound must say so: the reason reaches users through
// the fallback warning and Load errors, where a bare "context deadline
// exceeded" explains nothing.
func TestProbeTimeoutIsNamed(t *testing.T) {
	stubSecurity(t, "#!/bin/sh\nexec sleep 60\n")

	err := probe("svc", 20*time.Millisecond)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.ErrorContains(t, err, "keyring probe timed out after 20ms")
}

// Regression: the probe deadline expiring immediately after a successful add
// must not skip or kill the cleanup delete — cleanup runs synchronously
// under its own budget (the documented additive bound), so no stray probe
// entry is left in the keychain.
func TestProbeBoundedCleanupSurvivesProbeExpiry(t *testing.T) {
	argsFile := argsStub(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	restore := afterProbeAdd
	afterProbeAdd = cancel // probe context is dead before cleanup starts
	t.Cleanup(func() { afterProbeAdd = restore })

	require.NoError(t, probeBounded(ctx, "test", "__probe_expiry"))
	requireCleanupDelete(t, argsFile, "test", "__probe_expiry")
}

// A failed add reports the keyring unavailable with the security tool's own
// diagnostic folded in — that reason is what users see when the store
// explains its fallback — and cleanup is not attempted.
func TestProbeBoundedFailureCarriesDiagnostic(t *testing.T) {
	argsFile := filepath.Join(stubDir(t), "args")
	stubSecurity(t, "#!/bin/sh\nAF="+shQuote(argsFile)+"\necho \"$@\" >> \"$AF\"\ncat > /dev/null\n"+
		"echo 'security: SecKeychainItemCreateFromContent (<default>): User interaction is not allowed.' >&2\nexit 36\n")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := probeBounded(ctx, "test", "__probe_fail")
	require.Error(t, err)
	assert.ErrorContains(t, err, "User interaction is not allowed.")
	assert.ErrorContains(t, err, "exit status 36")

	raw, err := os.ReadFile(argsFile)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	require.Len(t, lines, 1, "failed probe must not attempt cleanup")
	assert.Equal(t, "-i", lines[0])
}

func TestQuoteSecurityArg(t *testing.T) {
	assert.Equal(t, "basecamp", quoteSecurityArg("basecamp"))
	assert.Equal(t, "''", quoteSecurityArg(""))
	assert.Equal(t, "'my service'", quoteSecurityArg("my service"))
	assert.Equal(t, `'it'"'"'s'`, quoteSecurityArg("it's"))
}
