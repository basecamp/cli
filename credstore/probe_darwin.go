package credstore

import (
	"context"
	"encoding/base64"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// securityPath is the macOS keychain tool go-keyring shells out to.
// Variable so tests can substitute a stub.
var securityPath = "/usr/bin/security"

// probeBounded mirrors go-keyring's darwin Set — `security -i` fed an
// add-generic-password command over stdin — via exec.CommandContext so the
// child is killed when ctx expires. go-keyring's own exec has no
// cancellation path: on a locked keychain with no TTY or GUI it blocks
// forever and the child cannot be reclaimed from outside.
func probeBounded(ctx context.Context, serviceName, probeKey string) error {
	// go-keyring base64-encodes every password it stores; mirror that so
	// probe success genuinely predicts go-keyring usability.
	password := "go-keyring-base64:" + base64.StdEncoding.EncodeToString([]byte("probe"))
	command := fmt.Sprintf("add-generic-password -U -s %s -a %s -w %s\n",
		quoteSecurityArg(serviceName), quoteSecurityArg(probeKey), quoteSecurityArg(password))

	cmd := exec.CommandContext(ctx, securityPath, "-i")
	cmd.Stdin = strings.NewReader(command)
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}

	_ = exec.CommandContext(ctx, securityPath, "delete-generic-password", "-s", serviceName, "-a", probeKey).Run()
	return nil
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
