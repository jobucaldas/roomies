package database

import (
	"bytes"
	"encoding/base64"
	"os"
	"testing"

	"github.com/roomies/backend/internal/secret"
)

// TestMain installs a data key so encrypted columns can be read and written.
func TestMain(m *testing.M) {
	ring, err := secret.NewKeyRing("test", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32)), "")
	if err != nil {
		panic(err)
	}
	secret.Configure(ring)
	os.Exit(m.Run())
}
