package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/roomies/backend/internal/database"
	"github.com/roomies/backend/internal/handlers"
	"github.com/roomies/backend/internal/middleware"
	"github.com/roomies/backend/internal/models"
	"github.com/roomies/backend/internal/repository"
)

func setupTest(t *testing.T) (*chi.Mux, *repository.UserRepository, string, func()) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "sqlite://:memory:"
	}
	db, err := database.Connect(databaseURL)
	if err != nil {
		t.Fatal(err)
	}

	if err := database.RunMigrations(db); err != nil {
		t.Fatal(err)
	}

	userRepo := repository.NewUserRepository(db)
	houseRepo := repository.NewHouseRepository(db)
	expenseRepo := repository.NewExpenseRepository(db)
	noteRepo := repository.NewNoteRepository(db)

	jwtSecret := "test-secret"
	authHandler := handlers.NewAuthHandler(userRepo, jwtSecret)
	houseHandler := handlers.NewHouseHandler(houseRepo, userRepo)
	expenseHandler := handlers.NewExpenseHandler(expenseRepo, houseRepo)
	noteHandler := handlers.NewNoteHandler(noteRepo, houseRepo)
	balanceHandler := handlers.NewBalanceHandler(expenseRepo, houseRepo)

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTAuth(jwtSecret))
		r.Get("/api/auth/me", authHandler.Me)
		r.Route("/api/houses", func(r chi.Router) {
			r.Get("/", houseHandler.List)
			r.Post("/", houseHandler.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", houseHandler.Get)
				r.Put("/", houseHandler.Update)
				r.Route("/members", func(r chi.Router) {
					r.Get("/", houseHandler.ListMembers)
					r.Post("/", houseHandler.AddMember)
					r.Put("/{userId}", houseHandler.UpdateMemberRole)
					r.Delete("/{userId}", houseHandler.RemoveMember)
				})
				r.Route("/expenses", func(r chi.Router) {
					r.Get("/", expenseHandler.List)
					r.Post("/", expenseHandler.Create)
					r.Route("/{eid}", func(r chi.Router) {
						r.Get("/", expenseHandler.Get)
						r.Put("/", expenseHandler.Update)
						r.Delete("/", expenseHandler.Delete)
						r.Post("/visibility", expenseHandler.SetVisibility)
					})
				})
				r.Get("/balances", balanceHandler.GetBalances)
				r.Route("/notes", func(r chi.Router) {
					r.Get("/", noteHandler.List)
					r.Post("/", noteHandler.Create)
					r.Route("/{nid}", func(r chi.Router) {
						r.Get("/", noteHandler.Get)
						r.Put("/", noteHandler.Update)
						r.Delete("/", noteHandler.Delete)
					})
				})
			})
		})
	})

	cleanup := func() {
		db.Close()
	}

	return r, userRepo, jwtSecret, cleanup
}

func registerUser(t *testing.T, router *chi.Mux, name, email, password string) string {
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

func getUserID(t *testing.T, router *chi.Mux, token string) string {
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

func createHouse(t *testing.T, router *chi.Mux, token, name string) string {
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
