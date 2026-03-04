package oauthcallback

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func listen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })
	return ln
}

func TestWaitForCallback_Success(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ln := listen(t)
	addr := ln.Addr().String()
	state := "test-state-123"

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		code, err := WaitForCallback(ctx, state, ln, "")
		if err != nil {
			errCh <- err
		} else {
			codeCh <- code
		}
	}()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://%s/callback?state=test-state-123&code=auth-code-456", addr))
	require.NoError(t, err)
	resp.Body.Close()

	select {
	case code := <-codeCh:
		assert.Equal(t, "auth-code-456", code)
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for callback")
	}
}

func TestWaitForCallback_MissingCode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ln := listen(t)
	addr := ln.Addr().String()
	errCh := make(chan error, 1)

	go func() {
		_, err := WaitForCallback(ctx, "state", ln, "")
		errCh <- err
	}()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://%s/callback?state=state", addr))
	require.NoError(t, err)
	resp.Body.Close()

	select {
	case err := <-errCh:
		assert.Contains(t, err.Error(), "missing authorization code")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}

func TestWaitForCallback_StateMismatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ln := listen(t)
	addr := ln.Addr().String()
	errCh := make(chan error, 1)

	go func() {
		_, err := WaitForCallback(ctx, "expected-state", ln, "")
		errCh <- err
	}()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://%s/callback?state=wrong-state&code=abc", addr))
	require.NoError(t, err)
	resp.Body.Close()

	select {
	case err := <-errCh:
		assert.Contains(t, err.Error(), "state mismatch")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}

func TestWaitForCallback_OAuthError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ln := listen(t)
	addr := ln.Addr().String()
	errCh := make(chan error, 1)

	go func() {
		_, err := WaitForCallback(ctx, "state", ln, "")
		errCh <- err
	}()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://%s/callback?error=access_denied", addr))
	require.NoError(t, err)
	resp.Body.Close()

	select {
	case err := <-errCh:
		assert.Contains(t, err.Error(), "access_denied")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}

func TestWaitForCallback_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	ln := listen(t)
	errCh := make(chan error, 1)

	go func() {
		_, err := WaitForCallback(ctx, "state", ln, "")
		errCh <- err
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		assert.Error(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}
