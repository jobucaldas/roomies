package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/roomies/backend/internal/models"
)

func authenticatedRequest(t *testing.T, router *chi.Mux, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var data []byte
	if body != nil {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	return response
}

func createExpense(t *testing.T, router *chi.Mux, token, houseID string, body models.CreateExpenseRequest) models.Expense {
	t.Helper()
	response := authenticatedRequest(t, router, http.MethodPost, "/api/houses/"+houseID+"/expenses", token, body)
	if response.Code != http.StatusCreated {
		t.Fatalf("create expense failed: %d %s", response.Code, response.Body.String())
	}
	var expense models.Expense
	if err := json.NewDecoder(response.Body).Decode(&expense); err != nil {
		t.Fatal(err)
	}
	return expense
}

func createNote(t *testing.T, router *chi.Mux, token, houseID string, body models.CreateNoteRequest) models.Note {
	t.Helper()
	response := authenticatedRequest(t, router, http.MethodPost, "/api/houses/"+houseID+"/notes", token, body)
	if response.Code != http.StatusCreated {
		t.Fatalf("create note failed: %d %s", response.Code, response.Body.String())
	}
	var note models.Note
	if err := json.NewDecoder(response.Body).Decode(&note); err != nil {
		t.Fatal(err)
	}
	return note
}

func addMember(t *testing.T, router *chi.Mux, adminToken, houseID, userID, role string) {
	t.Helper()
	response := authenticatedRequest(t, router, http.MethodPost, "/api/houses/"+houseID+"/members", adminToken,
		models.AddMemberRequest{UserID: userID, Role: role})
	if response.Code != http.StatusCreated {
		t.Fatalf("add member failed: %d %s", response.Code, response.Body.String())
	}
}

func TestExpenseObjectEndpointsAreScopedToRouteHouse(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()
	token := registerUser(t, router, "Scoped Expense", "scoped-expense@test.com", "password123")
	houseA := createHouse(t, router, token, "House A")
	houseB := createHouse(t, router, token, "House B")
	expense := createExpense(t, router, token, houseB, models.CreateExpenseRequest{
		Amount: 12.34, Description: "belongs to B", Date: "2025-02-01", Visibility: "shared",
	})

	checks := []struct {
		method string
		path   string
		body   interface{}
	}{
		{http.MethodGet, fmt.Sprintf("/api/houses/%s/expenses/%s", houseA, expense.ID), nil},
		{http.MethodPut, fmt.Sprintf("/api/houses/%s/expenses/%s", houseA, expense.ID), map[string]string{"description": "tampered"}},
		{http.MethodPost, fmt.Sprintf("/api/houses/%s/expenses/%s/visibility", houseA, expense.ID), models.SetVisibilityRequest{Visibility: "private"}},
		{http.MethodDelete, fmt.Sprintf("/api/houses/%s/expenses/%s", houseA, expense.ID), nil},
	}
	for _, check := range checks {
		response := authenticatedRequest(t, router, check.method, check.path, token, check.body)
		if response.Code != http.StatusNotFound {
			t.Errorf("%s %s: expected 404, got %d: %s", check.method, check.path, response.Code, response.Body.String())
		}
	}
	response := authenticatedRequest(t, router, http.MethodGet,
		fmt.Sprintf("/api/houses/%s/expenses/%s", houseB, expense.ID), token, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("expense was mutated through wrong house route: %d %s", response.Code, response.Body.String())
	}
}

func TestNoteObjectEndpointsAreScopedToRouteHouse(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()
	token := registerUser(t, router, "Scoped Note", "scoped-note@test.com", "password123")
	houseA := createHouse(t, router, token, "House A")
	houseB := createHouse(t, router, token, "House B")
	note := createNote(t, router, token, houseB, models.CreateNoteRequest{Title: "B", Content: "belongs to B"})

	checks := []struct {
		method string
		path   string
		body   interface{}
	}{
		{http.MethodGet, fmt.Sprintf("/api/houses/%s/notes/%s", houseA, note.ID), nil},
		{http.MethodPut, fmt.Sprintf("/api/houses/%s/notes/%s", houseA, note.ID), map[string]string{"title": "tampered"}},
		{http.MethodDelete, fmt.Sprintf("/api/houses/%s/notes/%s", houseA, note.ID), nil},
	}
	for _, check := range checks {
		response := authenticatedRequest(t, router, check.method, check.path, token, check.body)
		if response.Code != http.StatusNotFound {
			t.Errorf("%s %s: expected 404, got %d: %s", check.method, check.path, response.Code, response.Body.String())
		}
	}
	response := authenticatedRequest(t, router, http.MethodGet,
		fmt.Sprintf("/api/houses/%s/notes/%s", houseB, note.ID), token, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("note was mutated through wrong house route: %d %s", response.Code, response.Body.String())
	}
}

func TestCreateExpenseRejectsInvalidParticipantsWithoutPartialWrite(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()
	adminToken := registerUser(t, router, "Admin", "participant-admin@test.com", "password123")
	outsiderToken := registerUser(t, router, "Outsider", "participant-outsider@test.com", "password123")
	outsiderID := getUserID(t, router, outsiderToken)
	houseID := createHouse(t, router, adminToken, "Participant House")

	invalidBodies := []models.CreateExpenseRequest{
		{Amount: 10, Description: "bad split member", Date: "2025-02-01", Split: []models.SplitEntry{{UserID: outsiderID, Amount: 10}}},
		{Amount: 10, Description: "bad split total", Date: "2025-02-01", Split: []models.SplitEntry{{UserID: getUserID(t, router, adminToken), Amount: 9}}},
		{Amount: 10, Description: "bad visibility", Date: "2025-02-01", Visibility: "private", VisibleTo: []string{outsiderID}},
	}
	for _, body := range invalidBodies {
		response := authenticatedRequest(t, router, http.MethodPost, "/api/houses/"+houseID+"/expenses", adminToken, body)
		if response.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d: %s", response.Code, response.Body.String())
		}
	}
	response := authenticatedRequest(t, router, http.MethodGet, "/api/houses/"+houseID+"/expenses", adminToken, nil)
	var expenses []models.Expense
	if err := json.NewDecoder(response.Body).Decode(&expenses); err != nil {
		t.Fatal(err)
	}
	if len(expenses) != 0 {
		t.Fatalf("invalid aggregate left %d partial expenses", len(expenses))
	}
}

func TestExpenseMoneyValidationAndDeterministicCentSplit(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()
	adminToken := registerUser(t, router, "Cent Admin", "cent-admin@test.com", "password123")
	bobToken := registerUser(t, router, "Cent Bob", "cent-bob@test.com", "password123")
	carolToken := registerUser(t, router, "Cent Carol", "cent-carol@test.com", "password123")
	houseID := createHouse(t, router, adminToken, "Cent House")
	addMember(t, router, adminToken, houseID, getUserID(t, router, bobToken), "member")
	addMember(t, router, adminToken, houseID, getUserID(t, router, carolToken), "member")

	for _, amount := range []float64{-1, 1.001, 100000000} {
		response := authenticatedRequest(t, router, http.MethodPost, "/api/houses/"+houseID+"/expenses", adminToken,
			models.CreateExpenseRequest{Amount: amount, Description: "invalid money", Date: "2025-02-01"})
		if response.Code != http.StatusBadRequest {
			t.Errorf("amount %v: expected 400, got %d", amount, response.Code)
		}
	}
	expense := createExpense(t, router, adminToken, houseID, models.CreateExpenseRequest{
		Amount: 0.01, Description: "one cent", Date: "2025-02-01",
	})
	response := authenticatedRequest(t, router, http.MethodGet,
		fmt.Sprintf("/api/houses/%s/expenses/%s", houseID, expense.ID), adminToken, nil)
	var detail struct {
		Splits []models.ExpenseSplit `json:"splits"`
	}
	if err := json.NewDecoder(response.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if len(detail.Splits) != 3 {
		t.Fatalf("expected 3 splits, got %d", len(detail.Splits))
	}
	var total float64
	for _, split := range detail.Splits {
		total += split.ShareAmount
	}
	if total != 0.01 {
		t.Fatalf("expected exact one-cent total, got %.4f", total)
	}
}

func TestExpenseAmountUpdateRequiresValidReplacementSplits(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()
	adminToken := registerUser(t, router, "Update Admin", "update-split-admin@test.com", "password123")
	memberToken := registerUser(t, router, "Update Member", "update-split-member@test.com", "password123")
	adminID := getUserID(t, router, adminToken)
	memberID := getUserID(t, router, memberToken)
	houseID := createHouse(t, router, adminToken, "Update Split House")
	addMember(t, router, adminToken, houseID, memberID, "member")
	expense := createExpense(t, router, adminToken, houseID, models.CreateExpenseRequest{
		Amount: 10, Description: "before", Date: "2025-02-01",
	})

	response := authenticatedRequest(t, router, http.MethodPut,
		fmt.Sprintf("/api/houses/%s/expenses/%s", houseID, expense.ID), adminToken, map[string]float64{"amount": 12})
	if response.Code != http.StatusBadRequest {
		t.Fatalf("missing replacement split: expected 400, got %d: %s", response.Code, response.Body.String())
	}
	update := models.UpdateExpenseRequest{
		Amount: floatPointer(12),
		Split:  &[]models.SplitEntry{{UserID: adminID, Amount: 8}, {UserID: memberID, Amount: 4}},
	}
	response = authenticatedRequest(t, router, http.MethodPut,
		fmt.Sprintf("/api/houses/%s/expenses/%s", houseID, expense.ID), adminToken, update)
	if response.Code != http.StatusOK {
		t.Fatalf("valid replacement split failed: %d %s", response.Code, response.Body.String())
	}
	response = authenticatedRequest(t, router, http.MethodGet,
		fmt.Sprintf("/api/houses/%s/expenses/%s", houseID, expense.ID), adminToken, nil)
	var detail struct {
		Expense models.Expense        `json:"expense"`
		Splits  []models.ExpenseSplit `json:"splits"`
	}
	if err := json.NewDecoder(response.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if detail.Expense.Amount != 12 || len(detail.Splits) != 2 || detail.Splits[0].ShareAmount+detail.Splits[1].ShareAmount != 12 {
		t.Fatalf("expense and replacement splits were not updated atomically: %+v", detail)
	}
}

func TestLastAdminCannotBeDemotedOrRemoved(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()
	token := registerUser(t, router, "Last Admin", "last-admin@test.com", "password123")
	userID := getUserID(t, router, token)
	houseID := createHouse(t, router, token, "Admin House")

	response := authenticatedRequest(t, router, http.MethodPut,
		fmt.Sprintf("/api/houses/%s/members/%s", houseID, userID), token,
		models.UpdateMemberRoleRequest{Role: "member"})
	if response.Code != http.StatusBadRequest {
		t.Errorf("demote last admin: expected 400, got %d: %s", response.Code, response.Body.String())
	}
	response = authenticatedRequest(t, router, http.MethodDelete,
		fmt.Sprintf("/api/houses/%s/members/%s", houseID, userID), token, nil)
	if response.Code != http.StatusBadRequest {
		t.Errorf("remove last admin: expected 400, got %d: %s", response.Code, response.Body.String())
	}
}

func TestRemovedMemberRemainsInHistoricalBalancesWithoutAccess(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()
	adminToken := registerUser(t, router, "History Admin", "history-admin@test.com", "password123")
	formerToken := registerUser(t, router, "Former Member", "history-former@test.com", "password123")
	formerID := getUserID(t, router, formerToken)
	houseID := createHouse(t, router, adminToken, "History House")
	addMember(t, router, adminToken, houseID, formerID, "member")
	createExpense(t, router, adminToken, houseID, models.CreateExpenseRequest{
		Amount: 20, Description: "historical dinner", Date: "2025-02-01",
	})

	response := authenticatedRequest(t, router, http.MethodDelete,
		fmt.Sprintf("/api/houses/%s/members/%s", houseID, formerID), adminToken, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("remove referenced member failed: %d %s", response.Code, response.Body.String())
	}

	response = authenticatedRequest(t, router, http.MethodGet, "/api/houses/"+houseID+"/members", adminToken, nil)
	var activeMembers []models.HouseMember
	if err := json.NewDecoder(response.Body).Decode(&activeMembers); err != nil {
		t.Fatal(err)
	}
	for _, member := range activeMembers {
		if member.UserID == formerID {
			t.Fatal("removed member still appears in active member management")
		}
	}

	response = authenticatedRequest(t, router, http.MethodGet, "/api/houses/"+houseID+"/balances", adminToken, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("active member could not read historical balances: %d %s", response.Code, response.Body.String())
	}
	var balances models.BalanceResponse
	if err := json.NewDecoder(response.Body).Decode(&balances); err != nil {
		t.Fatal(err)
	}
	var formerBalance *models.BalanceEntry
	for i := range balances.Balances {
		if balances.Balances[i].UserID == formerID {
			formerBalance = &balances.Balances[i]
			break
		}
	}
	if formerBalance == nil || formerBalance.UserName != "Former Member" || formerBalance.Owed != 10 || formerBalance.Net != -10 {
		t.Fatalf("former member's historical debt is incomplete: %+v", formerBalance)
	}
	if len(balances.Settlements) != 1 || balances.Settlements[0].FromUserID != formerID || balances.Settlements[0].Amount != 10 {
		t.Fatalf("former member settlement is incomplete: %+v", balances.Settlements)
	}

	response = authenticatedRequest(t, router, http.MethodGet, "/api/houses/"+houseID+"/balances", formerToken, nil)
	if response.Code != http.StatusForbidden {
		t.Fatalf("removed member retained API access: expected 403, got %d", response.Code)
	}
}

func floatPointer(value float64) *float64 { return &value }
