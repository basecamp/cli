// Package auth provides authentication using shared CLI infrastructure.
//
// This is a seed template. Customize ServiceName, DisableEnvVar, and
// the auth flow for your application.
package auth

import (
	"github.com/basecamp/cli/credstore"
	"github.com/basecamp/cli/pkce"
)

// NewCredentialStore creates a credential store for this CLI.
// Replace "your-app" with your application name and "APP_NO_KEYRING"
// with your application's env var.
func NewCredentialStore(configDir string) *credstore.Store {
	return credstore.NewStore(credstore.StoreOptions{
		ServiceName:   "your-app",       // TODO: Replace with your app name
		DisableEnvVar: "APP_NO_KEYRING", // TODO: Replace with your env var
		FallbackDir:   configDir,
	})
}

// GeneratePKCE generates PKCE verifier and challenge for OAuth flows.
func GeneratePKCE() (verifier, challenge string) {
	verifier = pkce.GenerateVerifier()
	challenge = pkce.GenerateChallenge(verifier)
	return
}
