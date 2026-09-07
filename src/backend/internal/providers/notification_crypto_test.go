package providers

import (
	"encoding/base64"
	"strings"
	"testing"
)

func notificationTestKey(value byte) string {
	return base64.StdEncoding.EncodeToString([]byte(strings.Repeat(string([]byte{value}), 32)))
}
func TestNotificationCipherTamperAndRotation(t *testing.T) {
	old, err := NewNotificationDeliveryCipher("old", notificationTestKey(1), "")
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := old.Encrypt("sub-1", map[string]string{"token": "secret-token"})
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := NewNotificationDeliveryCipher("new", notificationTestKey(2), "old:"+notificationTestKey(1))
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]string
	if err := rotated.Decrypt("sub-1", encrypted, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["token"] != "secret-token" {
		t.Fatal("rotation failed")
	}
	tampered := encrypted[:len(encrypted)-2] + "xx"
	if err := rotated.Decrypt("sub-1", tampered, &decoded); err == nil {
		t.Fatal("tampered ciphertext was accepted")
	}
	if strings.Contains(encrypted, "secret-token") {
		t.Fatal("plaintext secret stored in envelope")
	}
}
