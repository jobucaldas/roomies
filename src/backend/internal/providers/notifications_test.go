package providers

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	webpush "github.com/marknefedov/go-webpush/v2"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestWebPushStatusClassification(t *testing.T) {
	vapid, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	auth := make([]byte, 16)
	if _, err := rand.Read(auth); err != nil {
		t.Fatal(err)
	}
	target := PushTarget{Platform: "web_push", Endpoint: "https://push.test/subscription", P256DH: base64.RawURLEncoding.EncodeToString(receiver.PublicKey().Bytes()), Auth: base64.RawURLEncoding.EncodeToString(auth)}
	for _, tt := range []struct {
		status          int
		revoke, wantErr bool
	}{{204, false, false}, {410, true, false}, {404, true, false}, {429, false, true}, {503, false, true}, {400, false, true}} {
		client := webpush.NewClient(webpush.Config{HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: tt.status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("provider response"))}, nil
		})}})
		p := &HTTPNotificationProvider{web: client, vapid: vapid, subject: "mailto:test@example.com"}
		result, err := p.Dispatch(context.Background(), target, []byte(`{"notification_id":"id"}`))
		if result.Revoke != tt.revoke || (err != nil) != tt.wantErr {
			t.Fatalf("status %d result=%+v err=%v", tt.status, result, err)
		}
	}
}

func TestFCMStatusClassification(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		revoke  bool
		wantErr bool
	}{
		{"success", 200, `{}`, false, false},
		{"confirmed unregistered not found", 404, `{"error":{"status":"NOT_FOUND","details":[{"@type":"type.googleapis.com/google.firebase.fcm.v1.FcmError","errorCode":"UNREGISTERED"}]}}`, true, false},
		{"unconfirmed not found", 404, `{"error":{"status":"NOT_FOUND"}}`, false, true},
		{"unregistered code", 400, `{"error":{"details":[{"@type":"type.googleapis.com/google.firebase.fcm.v1.FcmError","errorCode":"UNREGISTERED"}]}}`, true, false},
		{"invalid token with valid payload", 400, `{"error":{"details":[{"@type":"type.googleapis.com/google.firebase.fcm.v1.FcmError","errorCode":"INVALID_ARGUMENT"}]}}`, true, false},
		{"unstructured invalid argument", 400, `{"error":{"message":"token says INVALID_ARGUMENT"}}`, false, true},
		{"rate limit", 429, `{}`, false, true},
		{"server", 503, `{}`, false, true},
		{"unauthenticated", 401, `{"error":{"status":"UNAUTHENTICATED"}}`, false, true},
		{"permission", 403, `{"error":{"status":"PERMISSION_DENIED"}}`, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &HTTPNotificationProvider{fcmURL: "https://fcm.test/send", fcmClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tt.status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(tt.body))}, nil
			})}}
			result, err := p.Dispatch(context.Background(), PushTarget{Platform: "android_fcm", Token: "token"}, []byte(`{"notification_id":"id"}`))
			if result.Revoke != tt.revoke || (err != nil) != tt.wantErr {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}
