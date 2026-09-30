package providers

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/roomies/backend/internal/config"
)

func deliveryTestConfig(keyByte byte) *config.Config {
	return &config.Config{
		InvitationDeliveryKeyID: "test-key",
		InvitationDeliveryKey:   base64.StdEncoding.EncodeToString([]byte(strings.Repeat(string(keyByte), 32))),
	}
}

func TestInvitationDeliveryCipherRejectsTamperingWrongKeyAndAAD(t *testing.T) {
	cipher, err := NewInvitationDeliveryCipher(deliveryTestConfig('a'))
	if err != nil {
		t.Fatal(err)
	}
	payload, err := cipher.Encrypt("invite-1", "house-1", "job-1", "https://roomies.test/accept-invitation?token=secret")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := cipher.Decrypt(payload, "invite-1", "house-1", "job-1"); err != nil || !strings.Contains(got, "token=secret") {
		t.Fatalf("decrypt = %q, %v", got, err)
	}
	tampered := *payload
	tampered.Ciphertext = tampered.Ciphertext[:len(tampered.Ciphertext)-1] + "A"
	if _, err := cipher.Decrypt(&tampered, "invite-1", "house-1", "job-1"); err == nil {
		t.Fatal("tampered ciphertext decrypted")
	}
	if _, err := cipher.Decrypt(payload, "invite-1", "other-house", "job-1"); err == nil {
		t.Fatal("wrong associated data decrypted")
	}
	wrongKey, err := NewInvitationDeliveryCipher(deliveryTestConfig('b'))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wrongKey.Decrypt(payload, "invite-1", "house-1", "job-1"); err == nil {
		t.Fatal("wrong key decrypted")
	}
}
