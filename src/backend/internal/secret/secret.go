// Package secret encrypts personal data and household content before it is
// written to the database (AES-256-GCM) and computes keyed blind indexes for
// values that must stay searchable, such as email addresses.
//
// Columns hold "enc:v1:<key id>:<base64 nonce||ciphertext>". Values without
// that prefix are legacy plaintext; database.EncryptExistingData rewrites them
// (and values under retired keys) at startup.
package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
)

const prefix = "enc:v1:"

// ErrNotConfigured means no key ring was installed with Configure.
var ErrNotConfigured = errors.New("data encryption key is not configured")

// KeyRing holds the current key, retired keys still needed for reading, and
// the blind-index key derived from the current key.
type KeyRing struct {
	currentID string
	aeads     map[string]cipher.AEAD
	indexKey  []byte
}

var active atomic.Pointer[KeyRing]

// Configure installs the key ring used by String and Index.
func Configure(ring *KeyRing) { active.Store(ring) }

// Active returns the installed key ring, or nil.
func Active() *KeyRing { return active.Load() }

// NewKeyRing builds a key ring from base64 32-byte keys. oldKeys is a
// comma-separated list of id:base64-key pairs kept for decryption only.
func NewKeyRing(currentID, currentKey, oldKeys string) (*KeyRing, error) {
	currentID = strings.TrimSpace(currentID)
	if currentID == "" || strings.Contains(currentID, ":") {
		return nil, errors.New("data encryption key id must be non-empty and must not contain ':'")
	}
	ring := &KeyRing{currentID: currentID, aeads: map[string]cipher.AEAD{}}
	master, err := ring.add(currentID, currentKey)
	if err != nil {
		return nil, err
	}
	ring.indexKey, err = hkdf.Key(sha256.New, master, nil, "roomies blind index v1", 32)
	if err != nil {
		return nil, err
	}
	for _, pair := range strings.Split(oldKeys, ",") {
		if strings.TrimSpace(pair) == "" {
			continue
		}
		id, encoded, ok := strings.Cut(pair, ":")
		id = strings.TrimSpace(id)
		if !ok || id == "" {
			return nil, errors.New("DATA_ENCRYPTION_OLD_KEYS entries must be key-id:base64-key")
		}
		if _, exists := ring.aeads[id]; exists {
			return nil, fmt.Errorf("duplicate data encryption key id %q", id)
		}
		if _, err := ring.add(id, encoded); err != nil {
			return nil, err
		}
	}
	return ring, nil
}

func (k *KeyRing) add(id, encoded string) ([]byte, error) {
	master, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil || len(master) != 32 {
		return nil, fmt.Errorf("data encryption key %q must be a base64-encoded 32-byte key", id)
	}
	encKey, err := hkdf.Key(sha256.New, master, nil, "roomies column encryption v1", 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	k.aeads[id] = aead
	return master, nil
}

// CurrentID is the id of the key new values are sealed with.
func (k *KeyRing) CurrentID() string { return k.currentID }

// Encrypt seals plain under the current key.
func (k *KeyRing) Encrypt(plain string) (string, error) {
	aead := k.aeads[k.currentID]
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nonce, nonce, []byte(plain), nil)
	return prefix + k.currentID + ":" + base64.RawStdEncoding.EncodeToString(sealed), nil
}

// Decrypt opens a stored value. Values without the prefix are returned as-is
// (plaintext written before encryption was enabled).
func (k *KeyRing) Decrypt(stored string) (string, error) {
	if !strings.HasPrefix(stored, prefix) {
		return stored, nil
	}
	id, encoded, ok := strings.Cut(stored[len(prefix):], ":")
	if !ok {
		return "", errors.New("malformed encrypted value")
	}
	aead, found := k.aeads[id]
	if !found {
		return "", fmt.Errorf("data encryption key %q is not configured; add it to DATA_ENCRYPTION_OLD_KEYS", id)
	}
	sealed, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil || len(sealed) < aead.NonceSize() {
		return "", errors.New("malformed encrypted value")
	}
	plain, err := aead.Open(nil, sealed[:aead.NonceSize()], sealed[aead.NonceSize():], nil)
	if err != nil {
		return "", errors.New("encrypted value failed authentication")
	}
	return string(plain), nil
}

// NeedsRewrite reports a stored value that is plaintext or sealed under a
// retired key.
func (k *KeyRing) NeedsRewrite(stored string) bool {
	return !strings.HasPrefix(stored, prefix+k.currentID+":")
}

// Index returns a keyed SHA-256 HMAC of value for equality lookups on an
// encrypted column. Callers normalize value first (e.g. lowercase email).
func (k *KeyRing) Index(value string) string {
	mac := hmac.New(sha256.New, k.indexKey)
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

// Index computes a blind index with the installed key ring.
func Index(value string) (string, error) {
	ring := active.Load()
	if ring == nil {
		return "", ErrNotConfigured
	}
	return ring.Index(value), nil
}

// String is a text column encrypted at rest. It reads and marshals to JSON
// as its plaintext; database/sql writes it through Value, which encrypts.
type String string

// Value implements driver.Valuer.
func (s String) Value() (driver.Value, error) {
	ring := active.Load()
	if ring == nil {
		return nil, ErrNotConfigured
	}
	return ring.Encrypt(string(s))
}

// Scan implements sql.Scanner.
func (s *String) Scan(src any) error {
	var stored string
	switch v := src.(type) {
	case nil:
		*s = ""
		return nil
	case string:
		stored = v
	case []byte:
		stored = string(v)
	default:
		return fmt.Errorf("cannot scan %T into secret.String", src)
	}
	ring := active.Load()
	if ring == nil {
		if strings.HasPrefix(stored, prefix) {
			return ErrNotConfigured
		}
		*s = String(stored)
		return nil
	}
	plain, err := ring.Decrypt(stored)
	if err != nil {
		return err
	}
	*s = String(plain)
	return nil
}

// Plain returns the plaintext as a string.
func (s String) Plain() string { return string(s) }
