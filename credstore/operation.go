package credstore

import (
	"context"
	"fmt"
	"time"
)

// keyringOperation bounds both the wait behind another operation and the call
// itself. go-keyring has no cancellation API: after a timeout its worker may
// still finish. Refusing subsequent operations avoids overlapping an unknown
// write/delete and bounds abandoned workers to one per store. File storage is
// never substituted for a previously working keyring.
func (s *Store) keyringOperation(operation string, fn func() ([]byte, error)) ([]byte, error) {
	if s.operationTimeout <= 0 {
		return fn()
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.operationTimeout)
	defer cancel()
	select {
	case s.operationGate <- struct{}{}:
		defer func() { <-s.operationGate }()
	case <-ctx.Done():
		return nil, queueTimeoutError(operation, s.operationTimeout, ctx.Err())
	}
	if s.operationErr != nil {
		return nil, fmt.Errorf("keyring unavailable after an earlier timeout: %w", s.operationErr)
	}
	if ctx.Err() != nil {
		return nil, queueTimeoutError(operation, s.operationTimeout, ctx.Err())
	}

	type result struct {
		data []byte
		err  error
	}
	done := make(chan result, 1)
	go func() {
		data, err := fn()
		done <- result{data, err}
	}()
	select {
	case r := <-done:
		return r.data, r.err
	case <-ctx.Done():
		s.operationErr = fmt.Errorf("keyring %s timed out after %s (the operation may still complete): %w", operation, s.operationTimeout, ctx.Err())
		return nil, s.operationErr
	}
}

func queueTimeoutError(operation string, timeout time.Duration, err error) error {
	return fmt.Errorf("keyring %s timed out after %s waiting for another operation (not attempted): %w", operation, timeout, err)
}
