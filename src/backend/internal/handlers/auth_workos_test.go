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
	if body["access_token_ttl_s"] == nil {
		t.Fatalf("missing access_token_ttl_s: %#v", body)
	}
}

func TestWorkOSAuthorizeRequiresPKCE(t *testing.T) {
	env := newTestEnv(t)
	handler := handlers.NewAuthHandler(env.UserRepo, env.JWTSecret, enabledWorkOS(t, "unused"))
	req := httptest.NewRequest(http.MethodGet, "/api/auth/workos/authorize?screen_hint=sign-in", nil)
	w := httptest.NewRecorder()
	handler.WorkOSAuthorize(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
}

func TestWorkOSAuthorizeReturnsSignedState(t *testing.T) {
	env := newTestEnv(t)
	handler := handlers.NewAuthHandler(env.UserRepo, env.JWTSecret, enabledWorkOS(t, "unused"))
	pair, err := workosauth.GeneratePKCEPair()
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/workos/authorize?screen_hint=sign-in&code_challenge="+pair.Challenge+"&code_challenge_method=S256", nil)
	w := httptest.NewRecorder()
	handler.WorkOSAuthorize(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["url"] == "" || body["state"] == "" {
		t.Fatalf("body = %#v", body)
	}
	if err := workosauth.VerifyOAuthState(env.JWTSecret, body["state"], pair.Verifier, time.Now()); err != nil {
		t.Fatalf("state verify: %v", err)
	}
}

func TestWorkOSCallbackUpsertsUser(t *testing.T) {
	env := newTestEnv(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["code_verifier"] == "" {
			t.Fatalf("missing code_verifier: %#v", body)
		}
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
	payload := signedCallbackPayload(t, env.JWTSecret, "abc")
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

func TestWorkOSCallbackRejectsInvalidState(t *testing.T) {
	env := newTestEnv(t)
	handler := handlers.NewAuthHandler(env.UserRepo, env.JWTSecret, enabledWorkOS(t, "user_workos_1"))
	payload, _ := json.Marshal(map[string]string{
		"code":          "abc",
		"state":         "not.a.valid.state",
		"code_verifier": "verifier",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/workos/callback", bytes.NewReader(payload))
	w := httptest.NewRecorder()
	handler.WorkOSCallback(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
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
	payload := signedCallbackPayload(t, env.JWTSecret, "abc")
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

func signedCallbackPayload(t *testing.T, secret, code string) []byte {
	t.Helper()
	pair, err := workosauth.GeneratePKCEPair()
	if err != nil {
		t.Fatal(err)
	}
	state, err := workosauth.SignOAuthState(secret, pair.Challenge, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]string{
		"code":          code,
		"state":         state,
		"code_verifier": pair.Verifier,
	})
	if err != nil {
		t.Fatal(err)
	}
	return payload
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
