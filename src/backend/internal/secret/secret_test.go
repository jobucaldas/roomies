package secret

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func testKey(b byte) string {
	return base64.StdEncoding.EncodeToString([]byte(strings.Repeat(string(b), 32)))
}

func TestStringRoundTripsThroughDatabaseInterfaces(t *testing.T) {
	ring, err := NewKeyRing("k1", testKey('a'), "")
	if err != nil {
		t.Fatal(err)
	}
	Configure(ring)
	stored, err := String("Alice Example").Value()
	if err != nil {
		t.Fatal(err)
	}
	if s := stored.(string); !strings.HasPrefix(s, "enc:v1:k1:") || strings.Contains(s, "Alice") {
		t.Fatalf("value not encrypted: %q", s)
	}
	other, _ := String("Alice Example").Value()
	if other == stored {
		t.Fatal("encryption must be randomized")
	}
	var back String
	if err := back.Scan(stored); err != nil || back != "Alice Example" {
		t.Fatalf("scan = %q, %v", back, err)
	}
	var legacy String
	if err := legacy.Scan([]byte("written before encryption")); err != nil || legacy != "written before encryption" {
		t.Fatalf("legacy plaintext = %q, %v", legacy, err)
	}
	encoded, _ := json.Marshal(struct{ Name String }{back})
	if string(encoded) != `{"Name":"Alice Example"}` {
		t.Fatalf("json = %s", encoded)
	}
}

func TestKeyRotationAndTamperDetection(t *testing.T) {
	oldRing, _ := NewKeyRing("old", testKey('o'), "")
	sealed, _ := oldRing.Encrypt("note body")
	rotated, err := NewKeyRing("new", testKey('n'), "old:"+testKey('o'))
	if err != nil {
		t.Fatal(err)
	}
	if plain, err := rotated.Decrypt(sealed); err != nil || plain != "note body" {
		t.Fatalf("retired key unreadable: %q %v", plain, err)
	}
	if !rotated.NeedsRewrite(sealed) || rotated.NeedsRewrite(mustEncrypt(t, rotated, "x")) {
		t.Fatal("NeedsRewrite must flag only values under other keys")
	}
	tampered := sealed[:len(sealed)-2] + "AA"
	if _, err := rotated.Decrypt(tampered); err == nil {
		t.Fatal("tampered ciphertext must fail authentication")
	}
	withoutOld, _ := NewKeyRing("new", testKey('n'), "")
	if _, err := withoutOld.Decrypt(sealed); err == nil || !strings.Contains(err.Error(), `"old"`) {
		t.Fatalf("missing key should be named: %v", err)
	}
}

func TestIndexIsKeyedAndDeterministic(t *testing.T) {
	a, _ := NewKeyRing("k", testKey('a'), "")
	b, _ := NewKeyRing("k", testKey('b'), "")
	if a.Index("bob@example.test") != a.Index("bob@example.test") {
		t.Fatal("index must be deterministic")
	}
	if a.Index("bob@example.test") == b.Index("bob@example.test") {
		t.Fatal("index must depend on the key")
	}
}

func TestNewKeyRingRejectsBadKeys(t *testing.T) {
	for _, tc := range []struct{ id, key, old string }{
		{"", testKey('a'), ""},
		{"a:b", testKey('a'), ""},
		{"o'brien", testKey('a'), ""},
		{"k", testKey('a'), "bad id:" + testKey('b')},
		{"k", "short", ""},
		{"k", testKey('a'), "nocolon"},
		{"k", testKey('a'), "k:" + testKey('b')},
	} {
		if _, err := NewKeyRing(tc.id, tc.key, tc.old); err == nil {
			t.Fatalf("expected %+v to be rejected", tc)
		}
	}
}

func mustEncrypt(t *testing.T, ring *KeyRing, plain string) string {
	t.Helper()
	sealed, err := ring.Encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	return sealed
}
