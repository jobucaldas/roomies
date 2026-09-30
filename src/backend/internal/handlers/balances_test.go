package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/roomies/backend/internal/models"
)

func TestGetBalances(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()

	aliceToken := registerUser(t, router, "Alice", "bal-alice@test.com", "password123")
	bobToken := registerUser(t, router, "Bob", "bal-bob@test.com", "password123")
	bobUserID := getUserID(t, router, bobToken)

	houseID := createHouse(t, router, aliceToken, "Balance House")

	addBody := map[string]string{"user_id": bobUserID, "role": "member"}
	addData, _ := json.Marshal(addBody)
	addReq := httptest.NewRequest("POST", "/api/houses/"+houseID+"/members", bytes.NewReader(addData))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Authorization", "Bearer "+aliceToken)
	addW := httptest.NewRecorder()
	router.ServeHTTP(addW, addReq)
	if addW.Code != http.StatusCreated {
		t.Fatalf("add bob failed: %d", addW.Code)
	}

	expBody := models.CreateExpenseRequest{
		Amount: 200.00, Description: "Dinner", Visibility: "shared", Date: "2025-01-15",
	}
	expData, _ := json.Marshal(expBody)
	expReq := httptest.NewRequest("POST", "/api/houses/"+houseID+"/expenses", bytes.NewReader(expData))
	expReq.Header.Set("Content-Type", "application/json")
	expReq.Header.Set("Authorization", "Bearer "+aliceToken)
	expW := httptest.NewRecorder()
	router.ServeHTTP(expW, expReq)
	if expW.Code != http.StatusCreated {
		t.Fatalf("create expense failed: %d %s", expW.Code, expW.Body.String())
	}

	balReq := httptest.NewRequest("GET", "/api/houses/"+houseID+"/balances", nil)
	balReq.Header.Set("Authorization", "Bearer "+aliceToken)
	balW := httptest.NewRecorder()
	router.ServeHTTP(balW, balReq)
	if balW.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", balW.Code, balW.Body.String())
	}

	var result models.BalanceResponse
	json.NewDecoder(balW.Body).Decode(&result)
	if len(result.Balances) != 2 {
		t.Errorf("expected 2 balance entries, got %d", len(result.Balances))
	}
	if len(result.Settlements) == 0 {
		t.Errorf("expected at least 1 settlement when Alice paid 200 split 2 ways")
	}
}

func TestBalancesWithSettlements(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()

	aliceToken := registerUser(t, router, "Alice", "bal2-alice@test.com", "password123")
	bobToken := registerUser(t, router, "Bob", "bal2-bob@test.com", "password123")
	bobUserID := getUserID(t, router, bobToken)

	houseID := createHouse(t, router, aliceToken, "Settlement House")

	addBody := map[string]string{"user_id": bobUserID, "role": "member"}
	addData, _ := json.Marshal(addBody)
	addReq := httptest.NewRequest("POST", "/api/houses/"+houseID+"/members", bytes.NewReader(addData))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Authorization", "Bearer "+aliceToken)
	addW := httptest.NewRecorder()
	router.ServeHTTP(addW, addReq)
	if addW.Code != http.StatusCreated {
		t.Fatalf("add bob failed: %d", addW.Code)
	}

	exp1 := models.CreateExpenseRequest{Amount: 100.00, Description: "Groceries by Alice", Visibility: "shared", Date: "2025-01-15"}
	exp1Data, _ := json.Marshal(exp1)
	req1 := httptest.NewRequest("POST", "/api/houses/"+houseID+"/expenses", bytes.NewReader(exp1Data))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+aliceToken)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("create expense 1 failed: %d", w1.Code)
	}

	exp2 := models.CreateExpenseRequest{Amount: 60.00, Description: "Utilities by Bob", Visibility: "shared", Date: "2025-01-16"}
	exp2Data, _ := json.Marshal(exp2)
	req2 := httptest.NewRequest("POST", "/api/houses/"+houseID+"/expenses", bytes.NewReader(exp2Data))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+bobToken)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("create expense 2 failed: %d", w2.Code)
	}

	balReq := httptest.NewRequest("GET", "/api/houses/"+houseID+"/balances", nil)
	balReq.Header.Set("Authorization", "Bearer "+aliceToken)
	balW := httptest.NewRecorder()
	router.ServeHTTP(balW, balReq)
	if balW.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", balW.Code, balW.Body.String())
	}

	var result models.BalanceResponse
	json.NewDecoder(balW.Body).Decode(&result)

	for _, b := range result.Balances {
		t.Logf("  %s: paid=%.2f owed=%.2f net=%.2f", b.UserName, b.Paid, b.Owed, b.Net)
	}
	for _, s := range result.Settlements {
		t.Logf("  %s -> %s: %.2f", s.FromUserName, s.ToUserName, s.Amount)
	}

	if len(result.Settlements) == 0 {
		t.Error("expected settlements")
	}
}
