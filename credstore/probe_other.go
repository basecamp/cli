//go:build !darwin

package credstore

import "context"

// probeBounded runs the go-keyring probe with a deadline. Non-darwin
// backends (dbus secret service, Windows credential manager) run in-process,
// so timing out abandons at most a goroutine — there is no child process to
// reclaim.
func probeBounded(ctx context.Context, serviceName, probeKey string) error {
	done := make(chan error, 1)
	go func() { done <- probeDirect(serviceName, probeKey) }()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
