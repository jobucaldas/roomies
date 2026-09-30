package workosauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// PKCEPair is an OAuth PKCE verifier/challenge pair (S256).
type PKCEPair struct {
	Verifier        string
	Challenge       string
	ChallengeMethod string
}

// GeneratePKCEPair creates a high-entropy verifier and S256 challenge.
func GeneratePKCEPair() (PKCEPair, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return PKCEPair{}, fmt.Errorf("generate pkce verifier: %w", err)
	}
	verifier := base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(verifier))
	return PKCEPair{
		Verifier:        verifier,
		Challenge:       base64.RawURLEncoding.EncodeToString(sum[:]),
		ChallengeMethod: "S256",
	}, nil
}

// ChallengeFromVerifier returns the S256 code_challenge for verifier.
func ChallengeFromVerifier(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
