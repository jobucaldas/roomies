package workosauth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const oauthStateTTL = 10 * time.Minute

type oauthStatePayload struct {
	Nonce     string `json:"n"`
	Challenge string `json:"ch"`
	Exp       int64  `json:"exp"`
}

// SignOAuthState binds an authorize request to a PKCE challenge until Exp.
func SignOAuthState(secret, codeChallenge string, now time.Time) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", fmt.Errorf("oauth state signing secret is required")
	}
	if strings.TrimSpace(codeChallenge) == "" {
		return "", fmt.Errorf("code_challenge is required")
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate oauth state nonce: %w", err)
	}
	payload := oauthStatePayload{
		Nonce:     base64.RawURLEncoding.EncodeToString(nonce),
		Challenge: codeChallenge,
		Exp:       now.Add(oauthStateTTL).Unix(),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(body))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return body + "." + sig, nil
}

// VerifyOAuthState checks the HMAC signature, expiry, and PKCE challenge binding.
func VerifyOAuthState(secret, state, codeVerifier string, now time.Time) error {
	if strings.TrimSpace(secret) == "" {
		return fmt.Errorf("oauth state signing secret is required")
	}
	parts := strings.Split(state, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid oauth state")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(parts[0]))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return fmt.Errorf("invalid oauth state signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("invalid oauth state payload")
	}
	var payload oauthStatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("invalid oauth state payload")
	}
	if payload.Exp <= now.Unix() {
		return fmt.Errorf("oauth state expired")
	}
	if payload.Challenge == "" || ChallengeFromVerifier(codeVerifier) != payload.Challenge {
		return fmt.Errorf("oauth state pkce mismatch")
	}
	return nil
}
