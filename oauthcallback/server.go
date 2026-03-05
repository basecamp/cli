package oauthcallback

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// WaitForCallback starts a local HTTP server on listener and waits for an
// OAuth callback. It returns the authorization code from the callback.
//
// If listener is nil, one is created on listenAddr. Passing a pre-bound
// listener (e.g., from net.Listen("tcp", "127.0.0.1:0")) is preferred
// for tests to avoid port conflicts.
func WaitForCallback(ctx context.Context, expectedState string, listener net.Listener, listenAddr string) (string, error) {
	if listener == nil {
		lc := net.ListenConfig{}
		var err error
		listener, err = lc.Listen(ctx, "tcp", listenAddr)
		if err != nil {
			return "", fmt.Errorf("failed to start callback server: %w", err)
		}
	}
	defer func() { _ = listener.Close() }()

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	var once sync.Once

	send := func(ch chan<- string, val string) {
		select {
		case ch <- val:
		default:
		}
	}
	sendErr := func(ch chan<- error, val error) {
		select {
		case ch <- val:
		default:
		}
	}

	server := &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	shutdown := func() {
		once.Do(func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			go func() { defer cancel(); _ = server.Shutdown(shutdownCtx) }()
		})
	}

	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		code := r.URL.Query().Get("code")
		errParam := r.URL.Query().Get("error")

		if errParam != "" {
			sendErr(errCh, fmt.Errorf("OAuth error: %s", errParam))
			_, _ = fmt.Fprint(w, "<html><body><h1>Authentication failed</h1><p>You can close this window.</p></body></html>")
			shutdown()
			return
		}

		if state != expectedState {
			sendErr(errCh, fmt.Errorf("state mismatch: CSRF protection failed"))
			_, _ = fmt.Fprint(w, "<html><body><h1>Authentication failed</h1><p>State mismatch.</p></body></html>")
			shutdown()
			return
		}

		if code == "" {
			sendErr(errCh, fmt.Errorf("OAuth callback missing authorization code"))
			_, _ = fmt.Fprint(w, "<html><body><h1>Authentication failed</h1><p>Missing authorization code.</p></body></html>")
			shutdown()
			return
		}

		send(codeCh, code)
		_, _ = fmt.Fprint(w, "<html><body><h1>Authentication successful!</h1><p>You can close this window.</p></body></html>")
		shutdown()
	})

	go func() { _ = server.Serve(listener) }()

	select {
	case code := <-codeCh:
		return code, nil
	case err := <-errCh:
		return "", err
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(5 * time.Minute):
		return "", fmt.Errorf("authentication timeout waiting for callback on %s", listener.Addr())
	}
}
