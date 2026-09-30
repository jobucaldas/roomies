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

func TestAddMemberAsAdmin(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()

	aliceToken := registerUser(t, router, "Alice", "alice4@test.com", "password123")
	bobToken := registerUser(t, router, "Bob", "bob@test.com", "password123")

	bobUserID := getUserID(t, router, bobToken)
	houseID := createHouse(t, router, aliceToken, "Shared House")

	body := map[string]string{"user_id": bobUserID, "role": "member"}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/houses/"+houseID+"/members", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+aliceToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	bobHousesReq := httptest.NewRequest("GET", "/api/houses", nil)
	bobHousesReq.Header.Set("Authorization", "Bearer "+bobToken)
	bobW := httptest.NewRecorder()
	router.ServeHTTP(bobW, bobHousesReq)
	if bobW.Code != http.StatusOK {
		t.Errorf("bob houses list failed: %d", bobW.Code)
	}
	var bobHouses []models.House
	json.NewDecoder(bobW.Body).Decode(&bobHouses)
	if len(bobHouses) != 1 {
		t.Errorf("expected Bob to see 1 house, got %d", len(bobHouses))
	}
}

func TestNonAdminCannotAddMembers(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()

	aliceToken := registerUser(t, router, "Alice", "alice5@test.com", "password123")
	bobToken := registerUser(t, router, "Bob", "bob3@test.com", "password123")

	bobUserID := getUserID(t, router, bobToken)
	charlieToken := registerUser(t, router, "Charlie", "charlie@test.com", "password123")
	charlieUserID := getUserID(t, router, charlieToken)

	houseID := createHouse(t, router, aliceToken, "House")

	addBody := map[string]string{"user_id": bobUserID, "role": "member"}
	addData, _ := json.Marshal(addBody)
	addReq := httptest.NewRequest("POST", "/api/houses/"+houseID+"/members", bytes.NewReader(addData))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Authorization", "Bearer "+aliceToken)
	addW := httptest.NewRecorder()
	router.ServeHTTP(addW, addReq)
	if addW.Code != http.StatusCreated {
		t.Fatalf("add bob failed: %d %s", addW.Code, addW.Body.String())
	}

	charlieAddBody := map[string]string{"user_id": charlieUserID, "role": "member"}
	charlieAddData, _ := json.Marshal(charlieAddBody)
	charlieReq := httptest.NewRequest("POST", "/api/houses/"+houseID+"/members", bytes.NewReader(charlieAddData))
	charlieReq.Header.Set("Content-Type", "application/json")
	charlieReq.Header.Set("Authorization", "Bearer "+bobToken)
	charlieW := httptest.NewRecorder()
	router.ServeHTTP(charlieW, charlieReq)
	if charlieW.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", charlieW.Code, charlieW.Body.String())
	}
}

func TestRemoveMember(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()

	aliceToken := registerUser(t, router, "Alice", "alice6@test.com", "password123")
	bobToken := registerUser(t, router, "Bob", "bob4@test.com", "password123")

	bobUserID := getUserID(t, router, bobToken)
	houseID := createHouse(t, router, aliceToken, "House")

	addBody := map[string]string{"user_id": bobUserID, "role": "member"}
	addData, _ := json.Marshal(addBody)
	addReq := httptest.NewRequest("POST", "/api/houses/"+houseID+"/members", bytes.NewReader(addData))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Authorization", "Bearer "+aliceToken)
	addW := httptest.NewRecorder()
	router.ServeHTTP(addW, addReq)
	if addW.Code != http.StatusCreated {
		t.Fatalf("add bob failed: %d %s", addW.Code, addW.Body.String())
	}

	delReq := httptest.NewRequest("DELETE", "/api/houses/"+houseID+"/members/"+bobUserID, nil)
	delReq.Header.Set("Authorization", "Bearer "+aliceToken)
	delW := httptest.NewRecorder()
	router.ServeHTTP(delW, delReq)
	if delW.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", delW.Code, delW.Body.String())
	}
}
