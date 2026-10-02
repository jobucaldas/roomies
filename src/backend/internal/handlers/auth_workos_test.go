package handlers_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	req := oauthCallbackRequest(t, handler, "abc")
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
	req := oauthCallbackRequest(t, handler, "abc")
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

func TestWorkOSCallbackRejectsUnboundState(t *testing.T) {
	env := newTestEnv(t)
	handler := handlers.NewAuthHandler(env.UserRepo, env.JWTSecret, enabledWorkOS(t, "user_workos_1"))
	req := oauthCallbackRequest(t, handler, "abc")
	payload, _ := json.Marshal(map[string]string{"code": "abc", "state": "forged-state"})
	forged := httptest.NewRequest(http.MethodPost, "/api/auth/workos/callback", bytes.NewReader(payload))
	for _, cookie := range req.Cookies() {
		forged.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	handler.WorkOSCallback(w, forged)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
}

func TestWebRegisterUsesHttpOnlyCookieWithoutBearer(t *testing.T) {
	env := newTestEnv(t)
	body := []byte(`{"name":"Ada","email":"ada-web@example.test","password":"password123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Roomies-Client", "web")
	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if _, ok := resp["token"]; ok {
		t.Fatalf("web client received a bearer token: %#v", resp)
	}
	var session, hint *http.Cookie
	for _, cookie := range w.Result().Cookies() {
		switch cookie.Name {
		case "roomies_session":
			session = cookie
		case "roomies_session_hint":
			hint = cookie
		}
	}
	if session == nil || !session.HttpOnly || !session.Secure {
		t.Fatalf("session cookie = %#v", session)
	}
	if hint == nil || hint.HttpOnly || hint.Value != "1" {
		t.Fatalf("hint cookie = %#v", hint)
	}
	me := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	me.AddCookie(session)
	meRes := httptest.NewRecorder()
	env.Router.ServeHTTP(meRes, me)
	if meRes.Code != http.StatusOK {
		t.Fatalf("cookie session status = %d body = %s", meRes.Code, meRes.Body.String())
	}
}

func TestLoginIsRateLimited(t *testing.T) {
	env := newTestEnv(t)
	handler := handlers.NewAuthHandler(env.UserRepo, env.JWTSecret, nil)
	handler.SetRateLimit(2)
	for attempt := 0; attempt < 2; attempt++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader([]byte(`{"email":"missing@example.test","password":"password123"}`)))
		w := httptest.NewRecorder()
		handler.Login(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d status = %d body = %s", attempt, w.Code, w.Body.String())
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader([]byte(`{"email":"missing@example.test","password":"password123"}`)))
	w := httptest.NewRecorder()
	handler.Login(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
}

func oauthCallbackRequest(t *testing.T, handler *handlers.AuthHandler, code string) *http.Request {
	t.Helper()
	authReq := httptest.NewRequest(http.MethodGet, "/api/auth/workos/authorize?screen_hint=sign-in", nil)
	authRes := httptest.NewRecorder()
	handler.WorkOSAuthorize(authRes, authReq)
	if authRes.Code != http.StatusOK {
		t.Fatalf("authorize status = %d body = %s", authRes.Code, authRes.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(authRes.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(body["url"])
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("code_challenge_method") != "S256" || parsed.Query().Get("code_challenge") == "" || parsed.Query().Get("state") == "" {
		t.Fatalf("authorization url = %s", body["url"])
	}
	payload, _ := json.Marshal(map[string]string{"code": code, "state": parsed.Query().Get("state")})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/workos/callback", bytes.NewReader(payload))
	for _, cookie := range authRes.Result().Cookies() {
		req.AddCookie(cookie)
	}
	return req
}

func enabledWorkOS(t *testing.T, userID string) *workosauth.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var incoming map[string]string
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			t.Errorf("decode workos body: %v", err)
		} else if incoming["code_verifier"] == "" {
			t.Errorf("code_verifier was not sent")
		}
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

func TestLogoutEndsAuthKitSession(t *testing.T) {
	env := newTestEnv(t)
	claims, _ := json.Marshal(map[string]string{"sid": "session_01ABC"})
	accessToken := "e30." + base64.RawURLEncoding.EncodeToString(claims) + ".sig"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"user": map[string]any{
				"id":             "user_workos_logout",
				"email":          "logout@example.test",
				"email_verified": true,
			},
			"access_token": accessToken,
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
	callback := httptest.NewRecorder()
	handler.WorkOSCallback(callback, oauthCallbackRequest(t, handler, "abc"))
	if callback.Code != http.StatusOK {
		t.Fatalf("callback status = %d body = %s", callback.Code, callback.Body.String())
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	for _, cookie := range callback.Result().Cookies() {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	handler.Logout(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("logout status = %d body = %s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	want := server.URL + "/user_management/sessions/logout?session_id=session_01ABC"
	if body["logout_url"] != want {
		t.Fatalf("logout_url = %q, want %q", body["logout_url"], want)
	}
}

func TestLogoutWithoutAuthKitSessionReturnsNoContent(t *testing.T) {
	env := newTestEnv(t)
	handler := handlers.NewAuthHandler(env.UserRepo, env.JWTSecret, enabledWorkOS(t, "user_workos_x"))
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "roomies_session", Value: "not-a-jwt"})
	w := httptest.NewRecorder()
	handler.Logout(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d body = %s", w.Code, w.Body.String())
	}
}
