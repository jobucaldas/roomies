package handlers_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/roomies/backend/internal/handlers"
	"github.com/roomies/backend/internal/models"
)

func TestPrivateExpenseIsHiddenFromAdminsItWasNotSharedWith(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()
	adminToken := registerUser(t, router, "Admin", "private-admin@test.com", "password123")
	payerToken := registerUser(t, router, "Payer", "private-payer@test.com", "password123")
	houseID := createHouse(t, router, adminToken, "Private House")
	addMember(t, router, adminToken, houseID, payerToken, "member")
	expense := createExpense(t, router, payerToken, houseID, models.CreateExpenseRequest{
		Amount: 42, Description: "therapy session", Date: "2025-02-01", Visibility: "private",
	})
	path := fmt.Sprintf("/api/houses/%s/expenses/%s", houseID, expense.ID)

	for _, attempt := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, path, nil},
		{http.MethodPut, path, map[string]any{}},
		{http.MethodPost, path + "/visibility", models.SetVisibilityRequest{Visibility: "shared"}},
		{http.MethodDelete, path, nil},
	} {
		response := authenticatedRequest(t, router, attempt.method, attempt.path, adminToken, attempt.body)
		if response.Code != http.StatusNotFound || strings.Contains(response.Body.String(), "therapy") {
			t.Fatalf("%s %s: expected 404 without details, got %d: %s", attempt.method, attempt.path, response.Code, response.Body.String())
		}
	}
	response := authenticatedRequest(t, router, http.MethodGet, path, payerToken, nil)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "therapy") {
		t.Fatalf("payer lost access to the private expense: %d %s", response.Code, response.Body.String())
	}
}

func TestLogoutRevokesTheSessionToken(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()
	token := registerUser(t, router, "Leaver", "logout-revoke@test.com", "password123")
	if response := authenticatedRequest(t, router, http.MethodGet, "/api/auth/me", token, nil); response.Code != http.StatusOK {
		t.Fatalf("fresh session rejected: %d", response.Code)
	}
	if response := authenticatedRequest(t, router, http.MethodPost, "/api/auth/logout", token, nil); response.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d body = %s", response.Code, response.Body.String())
	}
	if response := authenticatedRequest(t, router, http.MethodGet, "/api/auth/me", token, nil); response.Code != http.StatusUnauthorized {
		t.Fatalf("signed-out token still works: %d %s", response.Code, response.Body.String())
	}
}

func TestSessionTokensWithoutIDOrExpiryAreRejected(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()
	userID := getUserID(t, env.Router, registerUser(t, env.Router, "Claims", "claims@test.com", "password123"))
	for name, claims := range map[string]jwt.MapClaims{
		"no jti":    {"user_id": userID, "exp": time.Now().Add(time.Hour).Unix()},
		"no expiry": {"user_id": userID, "jti": "abc"},
	} {
		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(env.JWTSecret))
		if err != nil {
			t.Fatal(err)
		}
		if response := authenticatedRequest(t, env.Router, http.MethodGet, "/api/auth/me", token, nil); response.Code != http.StatusUnauthorized {
			t.Fatalf("%s: expected 401, got %d", name, response.Code)
		}
	}
}

func TestLoginIsRateLimitedPerAccount(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()
	handler := handlers.NewAuthHandler(env.UserRepo, env.JWTSecret, nil)
	handler.SetRateLimit(1000)
	status := 0
	for attempt := 0; attempt < 11; attempt++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader([]byte(`{"email":"Target@Example.test","password":"guess"}`)))
		// A different client address every time: the per-IP budget never trips.
		req.RemoteAddr = fmt.Sprintf("198.51.100.%d:1234", attempt)
		w := httptest.NewRecorder()
		handler.Login(w, req)
		status = w.Code
	}
	if status != http.StatusTooManyRequests {
		t.Fatalf("eleventh guess against one account: status = %d", status)
	}
}

func TestInvitationLinksCarry256BitTokens(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()
	adminToken := registerUser(t, router, "Admin", "token-admin@test.com", "password123")
	houseID := createHouse(t, router, adminToken, "Token House")
	response := authenticatedRequest(t, router, http.MethodPost, "/api/houses/"+houseID+"/invites", adminToken,
		models.CreateInvitationRequest{Email: "guest@test.com", Role: "member"})
	if response.Code != http.StatusCreated {
		t.Fatalf("invite failed: %d %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	start := strings.Index(body, "https://")
	link, err := url.Parse(strings.SplitN(body[start:], `"`, 2)[0])
	if err != nil {
		t.Fatal(err)
	}
	if token := link.Query().Get("token"); len(token) < 43 {
		t.Fatalf("invitation token too short for 256 bits: %q", token)
	}
}

func TestAPIResponsesAreNotCacheable(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()
	token := registerUser(t, router, "Cache", "cache@test.com", "password123")
	response := authenticatedRequest(t, router, http.MethodGet, "/api/auth/me", token, nil)
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
}
