package credstore

import (
	"context"
	"encoding/base64"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// securityPath is the macOS keychain tool go-keyring shells out to.
// Variable so tests can substitute a stub.
var securityPath = "/usr/bin/security"

// probeCleanupTimeout bounds removal of the probe entry. Cleanup runs
// synchronously under its own budget rather than the probe's: a successful
// add may have consumed the probe deadline, and a delete skipped or killed
// by the exhausted probe context would leave a stray __probe_* entry in the
// keychain. Synchronous because Go does not wait for goroutines at process
// exit — an unjoined cleanup in a short-lived CLI would leak the entry. The
// tail is theoretical in practice: cleanup only runs when the keychain
// proved responsive milliseconds earlier. StoreOptions.ProbeTimeout
// documents this additive bound.
const probeCleanupTimeout = 5 * time.Second

// probeBounded mirrors go-keyring's darwin Set — `security -i` fed an
// add-generic-password command over stdin — via exec.CommandContext so the
// child is killed when ctx expires. go-keyring's own exec has no
// cancellation path: on a locked keychain with no TTY or GUI it blocks
// forever and the child cannot be reclaimed from outside.
func probeBounded(ctx context.Context, serviceName, key string) error {
	// go-keyring base64-encodes every password it stores; mirror that so
	// probe success genuinely predicts go-keyring usability.
	password := "go-keyring-base64:" + base64.StdEncoding.EncodeToString([]byte("probe"))
	command := fmt.Sprintf("add-generic-password -U -s %s -a %s -w %s\n",
		quoteSecurityArg(serviceName), quoteSecurityArg(key), quoteSecurityArg(password))

	cmd := exec.CommandContext(ctx, securityPath, "-i")
	cmd.Stdin = strings.NewReader(command)
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}

	afterProbeAdd()

	// Best-effort by design: the add already proved availability, and a
	// failed delete must not fail the probe.
	cleanupCtx, cancel := context.WithTimeout(context.Background(), probeCleanupTimeout)
	defer cancel()
	_ = exec.CommandContext(cleanupCtx, securityPath, "delete-generic-password", "-s", serviceName, "-a", key).Run()
	return nil
}

// afterProbeAdd runs between the probe's add and its cleanup delete. No-op in
// production; tests use it to expire the probe context in that window and pin
// cleanup's independence from it.
var afterProbeAdd = func() {}

var securityArgUnsafe = regexp.MustCompile(`[^\w@%+=:,./-]`)

// quoteSecurityArg mirrors go-keyring's internal shellescape.Quote so the
// probe entry is written exactly as go-keyring would write it.
func quoteSecurityArg(s string) string {
	if s == "" {
		return "''"
	}
	if securityArgUnsafe.MatchString(s) {
		return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
	}
	return s
}
