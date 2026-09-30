package workosauth

import (
	"testing"
	"time"
)

func TestGenerateAndVerifyOAuthState(t *testing.T) {
	pair, err := GeneratePKCEPair()
	if err != nil {
		t.Fatal(err)
	}
	if pair.ChallengeMethod != "S256" || pair.Verifier == "" || pair.Challenge == "" {
		t.Fatalf("pair = %#v", pair)
	}
	if ChallengeFromVerifier(pair.Verifier) != pair.Challenge {
		t.Fatal("challenge mismatch")
	}

	now := time.Unix(1_700_000_000, 0)
	state, err := SignOAuthState("test-secret", pair.Challenge, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyOAuthState("test-secret", state, pair.Verifier, now.Add(time.Minute)); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := VerifyOAuthState("wrong-secret", state, pair.Verifier, now.Add(time.Minute)); err == nil {
		t.Fatal("expected signature failure")
	}
	if err := VerifyOAuthState("test-secret", state, "not-the-verifier", now.Add(time.Minute)); err == nil {
		t.Fatal("expected pkce mismatch")
	}
	if err := VerifyOAuthState("test-secret", state, pair.Verifier, now.Add(11*time.Minute)); err == nil {
		t.Fatal("expected expiry failure")
	}
}
