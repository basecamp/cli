package pkce

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateVerifier(t *testing.T) {
	v := GenerateVerifier()
	assert.NotEmpty(t, v)

	// Should be base64url encoded 32 bytes = 43 chars
	assert.Len(t, v, 43)

	// Should be valid base64url
	_, err := base64.RawURLEncoding.DecodeString(v)
	assert.NoError(t, err)

	// Should be unique
	v2 := GenerateVerifier()
	assert.NotEqual(t, v, v2)
}

func TestGenerateChallenge(t *testing.T) {
	v := GenerateVerifier()
	c := GenerateChallenge(v)
	assert.NotEmpty(t, c)

	// Should be base64url encoded SHA256 = 43 chars
	assert.Len(t, c, 43)

	// Deterministic for same input
	c2 := GenerateChallenge(v)
	assert.Equal(t, c, c2)

	// Different input -> different output
	v2 := GenerateVerifier()
	c3 := GenerateChallenge(v2)
	assert.NotEqual(t, c, c3)
}

func TestGenerateState(t *testing.T) {
	s := GenerateState()
	assert.NotEmpty(t, s)

	// Should be base64url encoded 16 bytes = 22 chars
	assert.Len(t, s, 22)

	// Should be unique
	s2 := GenerateState()
	assert.NotEqual(t, s, s2)
}
