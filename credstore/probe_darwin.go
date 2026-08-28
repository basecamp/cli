package credstore

import (
	"context"
	"encoding/base64"
	"errors"
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
// by the exhausted probe context would leave a stray probe entry in the
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
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return securityError(out, err)
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

// securityError folds the security tool's diagnostic into its exit error.
// A bare "exit status 36" tells nobody why the keychain was unavailable;
// the tool's own line ("User interaction is not allowed.") does, and that
// reason reaches users through Store.FallbackWarning and Load errors.
func securityError(out []byte, err error) error {
	diagnostic := strings.Join(strings.Fields(string(out)), " ")
	if diagnostic == "" {
		return err
	}
	return fmt.Errorf("%s (%w)", diagnostic, err)
}

// securityExitReasons names the keychain failures go-keyring's darwin Set
// can surface, by exit status. go-keyring returns cmd.Wait()'s bare "exit
// status N" and discards security's own diagnostic line, and the unbounded
// probe — every session with a terminal — goes through go-keyring, so
// without this an interactive user's fallback read "exit status 36" where
// the bounded (headless) probe's reads "User interaction is not allowed."
// security exits with the low byte of the SecBase.h OSStatus, so the codes
// are stable; the text is what `security error <OSStatus>` prints.
var securityExitReasons = map[int]string{
	36:  "User interaction is not allowed.",                                 // errSecInteractionNotAllowed (-25308)
	37:  "A default keychain could not be found.",                           // errSecNoDefaultKeychain (-25307)
	45:  "The specified item already exists in the keychain.",               // errSecDuplicateItem (-25299)
	50:  "The specified keychain could not be found.",                       // errSecNoSuchKeychain (-25294)
	51:  "The user name or passphrase you entered is not correct.",          // errSecAuthFailed (-25293)
	52:  "This keychain cannot be modified.",                                // errSecReadOnly (-25292)
	53:  "No keychain is available. You may need to restart your computer.", // errSecNotAvailable (-25291)
	128: "User canceled the operation.",                                     // errSecUserCanceled (-128)
}

// keyringError folds the security tool's reason into a go-keyring exit
// error, in the same shape securityError gives the bounded path. Any other
// error — an exit status with no keychain meaning, or no exit at all —
// passes through unchanged rather than being given an invented reason.
func keyringError(err error) error {
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		if reason, ok := securityExitReasons[exit.ExitCode()]; ok {
			return fmt.Errorf("%s (%w)", reason, err)
		}
	}
	return err
}

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
