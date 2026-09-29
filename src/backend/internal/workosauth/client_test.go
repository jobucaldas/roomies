package workosauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthorizationURL(t *testing.T) {
	client := &Client{
		APIKey:      "sk_test",
		ClientID:    "client_123",
		RedirectURI: "http://localhost/callback",
	}
	url, err := client.AuthorizationURL("sign-up", "state-1")
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(url,
		"https://api.workos.com/user_management/authorize?",
		"client_id=client_123",
		"provider=authkit",
		"screen_hint=sign-up",
		"state=state-1",
		"redirect_uri=http%3A%2F%2Flocalhost%2Fcallback",
	) {
		t.Fatalf("unexpected authorization url: %s", url)
	}
}

func TestAuthenticateWithCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user_management/authenticate" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["grant_type"] != "authorization_code" || body["code"] != "abc" {
			t.Fatalf("body = %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"user": map[string]any{
				"id":         "user_1",
				"email":      "a@example.test",
				"first_name": "Ada",
				"last_name":  "Lovelace",
			},
			"access_token": "tok",
		})
	}))
	defer server.Close()

	client := &Client{
		APIKey:      "sk_test",
		ClientID:    "client_123",
		RedirectURI: "http://localhost/callback",
		APIBase:     server.URL,
		HTTPClient:  server.Client(),
	}
	result, err := client.AuthenticateWithCode(context.Background(), "abc", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatal(err)
	}
	if result.User.Email != "a@example.test" || DisplayName(result.User) != "Ada Lovelace" {
		t.Fatalf("result = %#v", result)
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !contains(value, part) {
			return false
		}
	}
	return true
}

func contains(value, part string) bool {
	return len(value) >= len(part) && (value == part || len(part) == 0 ||
		(func() bool {
			for i := 0; i+len(part) <= len(value); i++ {
				if value[i:i+len(part)] == part {
					return true
				}
			}
			return false
		})())
}
