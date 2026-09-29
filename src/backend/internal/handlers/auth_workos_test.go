package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/roomies/backend/internal/handlers"
	"github.com/roomies/backend/internal/workosauth"
)

func TestAuthConfigReportsAuthKitDisabledByDefault(t *testing.T) {
	env := newTestEnv(t)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/config", nil)
	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["authkit"] != false {
		t.Fatalf("authkit = %#v", body["authkit"])
	}
	if body["password"] != true {
		t.Fatalf("password = %#v", body["password"])
	}
}

func TestWorkOSCallbackUpsertsUser(t *testing.T) {
	env := newTestEnv(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"user": map[string]any{
				"id":         "user_workos_1",
				"email":      "authkit@example.test",
				"first_name": "Auth",
				"last_name":  "Kit",
			},
			"access_token": "tok",
		})
	}))
	t.Cleanup(server.Close)

	workos := &workosauth.Client{
		APIKey:      "sk_test",
		ClientID:    "client_test",
		RedirectURI: "http://localhost/callback",
		APIBase:     server.URL,
		HTTPClient:  server.Client(),
	}
	handler := handlers.NewAuthHandler(env.UserRepo, env.JWTSecret, workos)
	payload, _ := json.Marshal(map[string]string{"code": "abc"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/workos/callback", bytes.NewReader(payload))
	w := httptest.NewRecorder()
	handler.WorkOSCallback(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	var auth map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &auth); err != nil {
		t.Fatal(err)
	}
	if auth["token"] == nil || auth["token"] == "" {
		t.Fatalf("missing token: %#v", auth)
	}
	user := auth["user"].(map[string]any)
	if user["email"] != "authkit@example.test" {
		t.Fatalf("user = %#v", user)
	}
}
