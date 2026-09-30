package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/roomies/backend/internal/handlers"
	"github.com/roomies/backend/internal/models"
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
				"id":             "user_workos_1",
				"email":          "authkit@example.test",
				"email_verified": true,
				"first_name":     "Auth",
				"last_name":      "Kit",
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

func TestAuthConfigDisablesPasswordWhenAuthKitEnabled(t *testing.T) {
	env := newTestEnv(t)
	handler := handlers.NewAuthHandler(env.UserRepo, env.JWTSecret, enabledWorkOS(t, "unused"))
	req := httptest.NewRequest(http.MethodGet, "/api/auth/config", nil)
	w := httptest.NewRecorder()
	handler.Config(w, req)
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["authkit"] != true || body["password"] != false {
		t.Fatalf("config = %#v", body)
	}
}

func TestPasswordAuthRejectedWhenAuthKitEnabled(t *testing.T) {
	env := newTestEnv(t)
	handler := handlers.NewAuthHandler(env.UserRepo, env.JWTSecret, enabledWorkOS(t, "unused"))
	for _, path := range []string{"/api/auth/register", "/api/auth/login"} {
		body := []byte(`{"name":"A","email":"a@example.test","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		w := httptest.NewRecorder()
		if path == "/api/auth/register" {
			handler.Register(w, req)
		} else {
			handler.Login(w, req)
		}
		if w.Code != http.StatusForbidden {
			t.Fatalf("%s status = %d body = %s", path, w.Code, w.Body.String())
		}
	}
}

func TestWorkOSCallbackDoesNotLinkLocalPasswordAccount(t *testing.T) {
	env := newTestEnv(t)
	existing := &models.User{
		ID:           models.NewID(),
		Name:         "Local",
		Email:        "authkit@example.test",
		PasswordHash: "local-password-hash",
		CreatedAt:    time.Now().UTC(),
	}
	if err := env.UserRepo.Create(t.Context(), existing); err != nil {
		t.Fatal(err)
	}
	handler := handlers.NewAuthHandler(env.UserRepo, env.JWTSecret, enabledWorkOS(t, "user_workos_1"))
	payload, _ := json.Marshal(map[string]string{"code": "abc"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/workos/callback", bytes.NewReader(payload))
	w := httptest.NewRecorder()
	handler.WorkOSCallback(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	user, err := env.UserRepo.GetByEmail(t.Context(), existing.Email)
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != existing.ID || user.WorkOSUserID != "" || user.PasswordHash != existing.PasswordHash {
		t.Fatalf("local account was linked: %#v", user)
	}
}

func enabledWorkOS(t *testing.T, userID string) *workosauth.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"user": map[string]any{
				"id":             userID,
				"email":          "authkit@example.test",
				"email_verified": true,
				"first_name":     "Auth",
				"last_name":      "Kit",
			},
		})
	}))
	t.Cleanup(server.Close)
	return &workosauth.Client{
		APIKey:      "sk_test",
		ClientID:    "client_test",
		RedirectURI: "http://localhost/callback",
		APIBase:     server.URL,
		HTTPClient:  server.Client(),
	}
}
