package providers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/roomies/backend/internal/config"
)

// InvitationDeliveryPayload is safe to persist in an outbox: it contains only
// AES-GCM ciphertext and the key identifier, never a bearer token or URL.
type InvitationDeliveryPayload struct {
	KeyID      string `json:"key_id"`
	Ciphertext string `json:"ciphertext"`
}

type InvitationDeliveryCipher struct {
	activeKeyID string
	keys        map[string][]byte
}

func NewInvitationDeliveryCipher(cfg *config.Config) (*InvitationDeliveryCipher, error) {
	if cfg == nil || cfg.InvitationDeliveryKeyID == "" || cfg.InvitationDeliveryKey == "" {
		return nil, errors.New("invitation delivery key and key ID are required")
	}
	keys := map[string][]byte{}
	if err := addInvitationDeliveryKey(keys, cfg.InvitationDeliveryKeyID, cfg.InvitationDeliveryKey); err != nil {
		return nil, err
	}
	for _, entry := range strings.Split(cfg.InvitationDeliveryOldKeys, ",") {
		if entry = strings.TrimSpace(entry); entry == "" {
			continue
		}
		id, encoded, ok := strings.Cut(entry, ":")
		if !ok {
			return nil, fmt.Errorf("INVITATION_DELIVERY_OLD_KEYS entries must be key-id:base64-key")
		}
		if err := addInvitationDeliveryKey(keys, strings.TrimSpace(id), strings.TrimSpace(encoded)); err != nil {
			return nil, err
		}
	}
	return &InvitationDeliveryCipher{activeKeyID: cfg.InvitationDeliveryKeyID, keys: keys}, nil
}

func addInvitationDeliveryKey(keys map[string][]byte, keyID, encoded string) error {
	if keyID == "" {
		return errors.New("invitation delivery key ID is required")
	}
	if _, exists := keys[keyID]; exists {
		return fmt.Errorf("duplicate invitation delivery key ID %q", keyID)
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(key) != 32 {
		return fmt.Errorf("invitation delivery key %q must be a base64-encoded 32-byte AES-256 key", keyID)
	}
	keys[keyID] = key
	return nil
}

func invitationDeliveryAAD(invitationID, houseID, jobID string) []byte {
	return []byte(invitationID + "\x00" + houseID + "\x00" + jobID)
}

func (c *InvitationDeliveryCipher) Encrypt(invitationID, houseID, jobID, acceptanceURL string) (*InvitationDeliveryPayload, error) {
	if c == nil {
		return nil, errors.New("invitation delivery cipher is not configured")
	}
	block, err := aes.NewCipher(c.keys[c.activeKeyID])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate invitation delivery nonce: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, []byte(acceptanceURL), invitationDeliveryAAD(invitationID, houseID, jobID))
	return &InvitationDeliveryPayload{KeyID: c.activeKeyID, Ciphertext: base64.StdEncoding.EncodeToString(sealed)}, nil
}

func (c *InvitationDeliveryCipher) Decrypt(payload *InvitationDeliveryPayload, invitationID, houseID, jobID string) (string, error) {
	if c == nil || payload == nil || payload.KeyID == "" || payload.Ciphertext == "" {
		return "", errors.New("invitation delivery payload is unavailable")
	}
	key, ok := c.keys[payload.KeyID]
	if !ok {
		return "", fmt.Errorf("invitation delivery key %q is unavailable", payload.KeyID)
	}
	sealed, err := base64.StdEncoding.DecodeString(payload.Ciphertext)
	if err != nil {
		return "", errors.New("invalid invitation delivery ciphertext")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(sealed) < gcm.NonceSize() {
		return "", errors.New("invalid invitation delivery ciphertext")
	}
	plain, err := gcm.Open(nil, sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():], invitationDeliveryAAD(invitationID, houseID, jobID))
	if err != nil {
		return "", errors.New("invitation delivery ciphertext authentication failed")
	}
	return string(plain), nil
}
