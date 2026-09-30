package providers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type NotificationDeliveryCipher struct {
	currentID string
	keys      map[string][]byte
}
type EncryptedNotificationSecret struct {
	KeyID      string `json:"key_id"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

func NewNotificationDeliveryCipher(currentID, currentKey, oldKeys string) (*NotificationDeliveryCipher, error) {
	if strings.TrimSpace(currentID) == "" || strings.TrimSpace(currentKey) == "" {
		return nil, errors.New("notification delivery key id and key are required")
	}
	keys := map[string][]byte{}
	add := func(id, encoded string) error {
		key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
		if err != nil || len(key) != 32 {
			return fmt.Errorf("notification delivery key %q must be base64-encoded 32-byte AES-256", id)
		}
		if id == "" {
			return errors.New("notification delivery key id is required")
		}
		keys[id] = key
		return nil
	}
	if err := add(strings.TrimSpace(currentID), currentKey); err != nil {
		return nil, err
	}
	for _, pair := range strings.Split(oldKeys, ",") {
		if strings.TrimSpace(pair) == "" {
			continue
		}
		values := strings.SplitN(pair, ":", 2)
		if len(values) != 2 {
			return nil, errors.New("invalid NOTIFICATION_DELIVERY_OLD_KEYS entry")
		}
		if err := add(strings.TrimSpace(values[0]), values[1]); err != nil {
			return nil, err
		}
	}
	return &NotificationDeliveryCipher{currentID: strings.TrimSpace(currentID), keys: keys}, nil
}
func (c *NotificationDeliveryCipher) CurrentKeyID() string { return c.currentID }
func (c *NotificationDeliveryCipher) Encrypt(subscriptionID string, value any) (string, error) {
	plain, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(c.keys[c.currentID])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nil, nonce, plain, []byte(subscriptionID))
	body, err := json.Marshal(EncryptedNotificationSecret{KeyID: c.currentID, Nonce: base64.RawStdEncoding.EncodeToString(nonce), Ciphertext: base64.RawStdEncoding.EncodeToString(sealed)})
	return string(body), err
}
func (c *NotificationDeliveryCipher) Decrypt(subscriptionID, value string, target any) error {
	var envelope EncryptedNotificationSecret
	if err := json.Unmarshal([]byte(value), &envelope); err != nil {
		return err
	}
	key, ok := c.keys[envelope.KeyID]
	if !ok {
		return errors.New("notification delivery key is unavailable")
	}
	nonce, err := base64.RawStdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return err
	}
	sealed, err := base64.RawStdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	plain, err := gcm.Open(nil, nonce, sealed, []byte(subscriptionID))
	if err != nil {
		return errors.New("notification delivery ciphertext authentication failed")
	}
	return json.Unmarshal(plain, target)
}
