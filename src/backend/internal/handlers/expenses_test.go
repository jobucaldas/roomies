package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/roomies/backend/internal/models"
)

func TestCreateExpense(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()

	token := registerUser(t, router, "Alice", "exp-test@test.com", "password123")
	houseID := createHouse(t, router, token, "Expense House")

	body := models.CreateExpenseRequest{
		Amount:      100.00,
		Description: "Groceries",
		Category:    "Food",
		Date:        "2025-01-15",
		Visibility:  "shared",
	}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/houses/"+houseID+"/expenses", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var expense models.Expense
	json.NewDecoder(w.Body).Decode(&expense)
	if expense.Amount != 100.00 {
		t.Errorf("expected 100.00, got %f", expense.Amount)
	}
	if expense.Visibility != "shared" {
		t.Errorf("expected 'shared', got '%s'", expense.Visibility)
	}
}

func TestCreatePrivateExpense(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()

	token := registerUser(t, router, "Alice", "priv-test@test.com", "password123")
	houseID := createHouse(t, router, token, "Private House")

	body := models.CreateExpenseRequest{
		Amount:      50.00,
		Description: "Private Dinner",
		Visibility:  "private",
		VisibleTo:   []string{},
	}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/houses/"+houseID+"/expenses", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMonitorCannotCreateExpense(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()

	aliceToken := registerUser(t, router, "Alice", "monitor-exp@test.com", "password123")
	monitorToken := registerUser(t, router, "Mon", "mon-exp@test.com", "password123")
	monitorUserID := getUserID(t, router, monitorToken)

	houseID := createHouse(t, router, aliceToken, "Monitor House")

	addBody := map[string]string{"user_id": monitorUserID, "role": "monitor"}
	addData, _ := json.Marshal(addBody)
	addReq := httptest.NewRequest("POST", "/api/houses/"+houseID+"/members", bytes.NewReader(addData))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Authorization", "Bearer "+aliceToken)
	addW := httptest.NewRecorder()
	router.ServeHTTP(addW, addReq)
	if addW.Code != http.StatusCreated {
		t.Fatalf("add monitor failed: %d %s", addW.Code, addW.Body.String())
	}

	body := models.CreateExpenseRequest{
		Amount:      30.00,
		Description: "Monitor attempt",
		Visibility:  "shared",
	}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/houses/"+houseID+"/expenses", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+monitorToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListExpenses(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()

	token := registerUser(t, router, "Alice", "list-exp@test.com", "password123")
	houseID := createHouse(t, router, token, "List House")

	createExp := func(desc string, amount float64) {
		body := models.CreateExpenseRequest{Amount: amount, Description: desc, Visibility: "shared", Date: "2025-01-15"}
		data, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/api/houses/"+houseID+"/expenses", bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create expense failed: %d %s", w.Code, w.Body.String())
		}
	}

	createExp("Rent", 1000.00)
	createExp("Utilities", 200.00)

	req := httptest.NewRequest("GET", "/api/houses/"+houseID+"/expenses", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var expenses []models.Expense
	json.NewDecoder(w.Body).Decode(&expenses)
	if len(expenses) != 2 {
		t.Errorf("expected 2 expenses, got %d", len(expenses))
	}
}

func TestDeleteExpenseAsAuthor(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()

	token := registerUser(t, router, "Alice", "del-exp@test.com", "password123")
	houseID := createHouse(t, router, token, "Delete House")

	body := models.CreateExpenseRequest{Amount: 50.00, Description: "To Delete", Visibility: "shared", Date: "2025-01-15"}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/houses/"+houseID+"/expenses", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var expense models.Expense
	json.NewDecoder(w.Body).Decode(&expense)

	delReq := httptest.NewRequest("DELETE", "/api/houses/"+houseID+"/expenses/"+expense.ID, nil)
	delReq.Header.Set("Authorization", "Bearer "+token)
	delW := httptest.NewRecorder()
	router.ServeHTTP(delW, delReq)
	if delW.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", delW.Code, delW.Body.String())
	}
}

func TestNonAuthorCannotDeleteExpense(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()

	aliceToken := registerUser(t, router, "Alice", "non-auth@test.com", "password123")
	bobToken := registerUser(t, router, "Bob", "non-auth-bob@test.com", "password123")
	bobUserID := getUserID(t, router, bobToken)

	houseID := createHouse(t, router, aliceToken, "Non-Auth House")

	body := models.CreateExpenseRequest{Amount: 75.00, Description: "Alice expense", Visibility: "shared", Date: "2025-01-15"}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/houses/"+houseID+"/expenses", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+aliceToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var expense models.Expense
	json.NewDecoder(w.Body).Decode(&expense)

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

	delReq := httptest.NewRequest("DELETE", "/api/houses/"+houseID+"/expenses/"+expense.ID, nil)
	delReq.Header.Set("Authorization", "Bearer "+bobToken)
	delW := httptest.NewRecorder()
	router.ServeHTTP(delW, delReq)
	if delW.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", delW.Code, delW.Body.String())
	}
}
