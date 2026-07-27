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

func TestProbeBoundedKillsChildOnTimeout(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "pid")
	// exec replaces the shell so the recorded PID is the hung process itself,
	// mirroring a `security -i` blocked on a locked keychain.
	stubSecurity(t, "#!/bin/sh\necho $$ > "+pidFile+".tmp && mv "+pidFile+".tmp "+pidFile+"\nexec sleep 60\n")

	// Expire the context once the child is demonstrably hung (PID recorded),
	// so the test exercises exactly the timeout path without racing startup.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		for {
			if _, err := os.Stat(pidFile); err == nil {
				cancel()
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
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

func TestProbeBoundedSuccess(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args")
	stubSecurity(t, "#!/bin/sh\necho \"$@\" >> "+argsFile+"\ncat > /dev/null\nexit 0\n")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, probeBounded(ctx, "test", "__probe_ok"))

	raw, err := os.ReadFile(argsFile)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	require.Len(t, lines, 2, "probe should add then delete the probe entry")
	assert.Equal(t, "-i", lines[0])
	assert.Equal(t, "delete-generic-password -s test -a __probe_ok", lines[1])
}

func TestQuoteSecurityArg(t *testing.T) {
	assert.Equal(t, "basecamp", quoteSecurityArg("basecamp"))
	assert.Equal(t, "''", quoteSecurityArg(""))
	assert.Equal(t, "'my service'", quoteSecurityArg("my service"))
	assert.Equal(t, `'it'"'"'s'`, quoteSecurityArg("it's"))
}
