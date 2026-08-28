//go:build !darwin

package credstore

import "context"

// probeBounded runs the go-keyring probe with a deadline. Non-darwin
// backends (dbus secret service, Windows credential manager) run in-process,
// so timing out abandons at most a goroutine — there is no child process to
// reclaim. An abandoned probe whose blocked Set later succeeds can leak the
// probe entry if the process exits before Delete runs; the pid-derived
// probeKey makes that self-healing — the next probe from a process reusing
// that pid overwrites and removes the leftover (see probeKey).
func probeBounded(ctx context.Context, serviceName, key string) error {
	done := make(chan error, 1)
	go func() { done <- probeDirect(serviceName, key) }()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// keyringError passes a go-keyring failure through: non-darwin backends run
// in-process and their errors already name the failure.
func keyringError(err error) error { return err }
