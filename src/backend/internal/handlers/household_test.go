package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/roomies/backend/internal/models"
)

func idempotentRequest(t *testing.T, router http.Handler, method, path, token, key string, body any) *httptest.ResponseRecorder {
	t.Helper()
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", key)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestHouseholdAuthorizationIsolationIdempotencyAndSafeEvents(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()
	admin := registerUser(t, env.Router, "Admin", "domain-admin@test", "password123")
	monitor := registerUser(t, env.Router, "Monitor", "domain-monitor@test", "password123")
	removed := registerUser(t, env.Router, "Removed", "domain-removed@test", "password123")
	houseID := createHouse(t, env.Router, admin, "Domain House")
	otherHouseID := createHouse(t, env.Router, admin, "Other House")
	addMember(t, env.Router, admin, houseID, getUserID(t, env.Router, monitor), "monitor")
	removedID := getUserID(t, env.Router, removed)
	addMember(t, env.Router, admin, houseID, removedID, "member")
	if _, err := env.DB.Exec(`DELETE FROM house_members WHERE house_id=$1 AND user_id=$2`, houseID, removedID); err != nil {
		t.Fatal(err)
	}

	body := models.GroceryRequest{Name: "Milk", Quantity: "2", Unit: "cartons", Note: "oat"}
	first := idempotentRequest(t, env.Router, http.MethodPost, "/api/houses/"+houseID+"/groceries", admin, "grocery-create", body)
	if first.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", first.Code, first.Body.String())
	}
	second := idempotentRequest(t, env.Router, http.MethodPost, "/api/houses/"+houseID+"/groceries", admin, "grocery-create", body)
	if second.Code != http.StatusCreated || second.Header().Get("X-Idempotent-Replay") != "true" || second.Body.String() != first.Body.String() {
		t.Fatalf("replay mismatch: %d %s", second.Code, second.Body.String())
	}
	var item models.GroceryItem
	if err := json.NewDecoder(first.Body).Decode(&item); err != nil {
		t.Fatal(err)
	}

	for name, token := range map[string]string{"monitor": monitor, "removed": removed} {
		response := authenticatedRequest(t, env.Router, http.MethodPost, "/api/houses/"+houseID+"/groceries", token, body)
		if response.Code != http.StatusForbidden {
			t.Fatalf("%s mutate got %d", name, response.Code)
		}
	}
	if response := authenticatedRequest(t, env.Router, http.MethodGet, "/api/houses/"+houseID+"/groceries", monitor, nil); response.Code != http.StatusOK {
		t.Fatalf("monitor list: %d", response.Code)
	}
	if response := authenticatedRequest(t, env.Router, http.MethodGet, "/api/houses/"+houseID+"/groceries", removed, nil); response.Code != http.StatusForbidden {
		t.Fatalf("removed list: %d", response.Code)
	}
	if response := authenticatedRequest(t, env.Router, http.MethodGet, "/api/houses/"+otherHouseID+"/groceries/"+item.ID, admin, nil); response.Code != http.StatusNotFound {
		t.Fatalf("cross-house read: %d", response.Code)
	}

	events := authenticatedRequest(t, env.Router, http.MethodGet, "/api/houses/"+houseID+"/events", admin, nil)
	if events.Code != http.StatusOK || strings.Contains(events.Body.String(), "Milk") || strings.Contains(events.Body.String(), "oat") {
		t.Fatalf("unsafe event payload: %d %s", events.Code, events.Body.String())
	}
	if !strings.Contains(events.Body.String(), `"category":"grocery"`) || !strings.Contains(events.Body.String(), item.ID) {
		t.Fatalf("missing resource-only event: %s", events.Body.String())
	}
}

func TestChatOwnMessageTombstoneAndCursorBounds(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()
	alice := registerUser(t, env.Router, "Alice", "chat-alice@test", "password123")
	bob := registerUser(t, env.Router, "Bob", "chat-bob@test", "password123")
	houseID := createHouse(t, env.Router, alice, "Chat House")
	addMember(t, env.Router, alice, houseID, getUserID(t, env.Router, bob), "member")
	created := authenticatedRequest(t, env.Router, http.MethodPost, "/api/houses/"+houseID+"/chat", alice, models.ChatMessageRequest{Body: "hello private text"})
	if created.Code != http.StatusCreated {
		t.Fatalf("create chat: %d %s", created.Code, created.Body.String())
	}
	var message models.ChatMessage
	_ = json.NewDecoder(created.Body).Decode(&message)
	if response := authenticatedRequest(t, env.Router, http.MethodDelete, "/api/houses/"+houseID+"/chat/"+message.ID, bob, nil); response.Code != http.StatusNotFound {
		t.Fatalf("other delete: %d", response.Code)
	}
	if response := authenticatedRequest(t, env.Router, http.MethodDelete, "/api/houses/"+houseID+"/chat/"+message.ID, alice, nil); response.Code != http.StatusNoContent {
		t.Fatalf("own delete: %d", response.Code)
	}
	page := authenticatedRequest(t, env.Router, http.MethodGet, "/api/houses/"+houseID+"/chat?limit=1", alice, nil)
	if page.Code != http.StatusOK || strings.Contains(page.Body.String(), "hello private text") || !strings.Contains(page.Body.String(), "deleted_at") {
		t.Fatalf("tombstone: %d %s", page.Code, page.Body.String())
	}
	if response := authenticatedRequest(t, env.Router, http.MethodGet, "/api/houses/"+houseID+"/chat?limit=101", alice, nil); response.Code != http.StatusBadRequest {
		t.Fatalf("cursor bound: %d", response.Code)
	}
}
