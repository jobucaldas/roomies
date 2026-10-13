package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/roomies/backend/internal/models"
	"github.com/roomies/backend/internal/repository"
)

func setupTest(t *testing.T) (http.Handler, *repository.UserRepository, string, func()) {
	env := newTestEnv(t)
	return env.Router, env.UserRepo, env.JWTSecret, env.Cleanup
}

func registerUser(t *testing.T, router http.Handler, name, email, password string) string {
	body := map[string]string{"name": name, "email": email, "password": password}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register failed: %d %s", w.Code, w.Body.String())
	}
	var resp models.AuthResponse
	json.NewDecoder(w.Body).Decode(&resp)
	return resp.Token
}

func getUserID(t *testing.T, router http.Handler, token string) string {
	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get user failed: %d %s", w.Code, w.Body.String())
	}
	var user models.User
	json.NewDecoder(w.Body).Decode(&user)
	return user.ID
}

func createHouse(t *testing.T, router http.Handler, token, name string) string {
	body := map[string]string{"name": name}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/houses", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create house failed: %d %s", w.Code, w.Body.String())
	}
	var house models.House
	json.NewDecoder(w.Body).Decode(&house)
	return house.ID
}

func TestCreateHouse(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()

	token := registerUser(t, router, "Alice", "alice@test.com", "password123")

	body := map[string]string{"name": "Test House"}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/houses", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var house models.House
	json.NewDecoder(w.Body).Decode(&house)
	if house.Name != "Test House" {
		t.Errorf("expected 'Test House', got '%s'", house.Name)
	}
	if house.ID == "" {
		t.Error("expected non-empty house ID")
	}
}

func TestCreateHouseNoName(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()
	token := registerUser(t, router, "Alice", "alice2@test.com", "password123")

	body := map[string]string{"name": ""}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/houses", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestListHouses(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()
	token := registerUser(t, router, "Alice", "alice3@test.com", "password123")
	createHouse(t, router, token, "House 1")
	createHouse(t, router, token, "House 2")

	req := httptest.NewRequest("GET", "/api/houses", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var houses []models.House
	json.NewDecoder(w.Body).Decode(&houses)
	if len(houses) != 2 {
		t.Errorf("expected 2 houses, got %d", len(houses))
	}
}

func TestMembersJoinOnlyByAcceptingAnInvitation(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()

	aliceToken := registerUser(t, router, "Alice", "alice4@test.com", "password123")
	bobToken := registerUser(t, router, "Bob", "bob@test.com", "password123")
	houseID := createHouse(t, router, aliceToken, "Shared House")

	// Adding someone by user id would expose their name and email to the
	// house without their consent, so the endpoint no longer exists.
	direct := authenticatedRequest(t, router, http.MethodPost, "/api/houses/"+houseID+"/members", aliceToken,
		map[string]string{"user_id": getUserID(t, router, bobToken), "role": "member"})
	if direct.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected direct add to be unavailable, got %d: %s", direct.Code, direct.Body.String())
	}

	addMember(t, router, aliceToken, houseID, bobToken, "member")
	bobW := authenticatedRequest(t, router, http.MethodGet, "/api/houses", bobToken, nil)
	var bobHouses []models.House
	json.NewDecoder(bobW.Body).Decode(&bobHouses)
	if len(bobHouses) != 1 {
		t.Errorf("expected Bob to see 1 house after accepting, got %d", len(bobHouses))
	}
}

func TestRemoveMember(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()

	aliceToken := registerUser(t, router, "Alice", "alice6@test.com", "password123")
	bobToken := registerUser(t, router, "Bob", "bob4@test.com", "password123")

	bobUserID := getUserID(t, router, bobToken)
	houseID := createHouse(t, router, aliceToken, "House")

	addMember(t, router, aliceToken, houseID, bobToken, "member")

	delReq := httptest.NewRequest("DELETE", "/api/houses/"+houseID+"/members/"+bobUserID, nil)
	delReq.Header.Set("Authorization", "Bearer "+aliceToken)
	delW := httptest.NewRecorder()
	router.ServeHTTP(delW, delReq)
	if delW.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", delW.Code, delW.Body.String())
	}
}
