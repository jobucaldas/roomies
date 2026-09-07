package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func notificationRequest(t *testing.T, router http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var data []byte
	if body != nil {
		data, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}
func TestNotificationPreferencesAreSelfOwnedAndHouseScoped(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()
	admin := registerUser(t, env.Router, "Admin", "notify-admin@test", "password123")
	other := registerUser(t, env.Router, "Other", "notify-other@test", "password123")
	houseID := createHouse(t, env.Router, admin, "House")
	body := map[string]any{"expense_created_enabled": false, "reminder_enabled": true, "cadence": "daily_digest", "timezone": "Europe/Lisbon", "digest_minutes": 540}
	w := notificationRequest(t, env.Router, "PUT", "/api/houses/"+houseID+"/notification-preferences", admin, body)
	if w.Code != 200 {
		t.Fatalf("put preferences: %d %s", w.Code, w.Body.String())
	}
	w = notificationRequest(t, env.Router, "GET", "/api/houses/"+houseID+"/notification-preferences", other, nil)
	if w.Code != 403 {
		t.Fatalf("non-member got %d", w.Code)
	}
}
func TestMonitorCanViewButCannotCreateScheduledEvent(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()
	admin := registerUser(t, env.Router, "Admin", "event-admin@test", "password123")
	monitor := registerUser(t, env.Router, "Monitor", "event-monitor@test", "password123")
	monitorID := getUserID(t, env.Router, monitor)
	houseID := createHouse(t, env.Router, admin, "House")
	if _, err := env.DB.Exec(`INSERT INTO house_members(id,house_id,user_id,role) VALUES('monitor-membership',$1,$2,'monitor')`, houseID, monitorID); err != nil {
		t.Fatal(err)
	}
	body := map[string]any{"title": "Bins", "timezone": "UTC", "dtstart_local": "2025-01-02T09:00:00", "rrule": "FREQ=DAILY;COUNT=2", "exdates": []string{}}
	w := notificationRequest(t, env.Router, "POST", "/api/houses/"+houseID+"/scheduled-events", monitor, body)
	if w.Code != 403 {
		t.Fatalf("monitor create got %d: %s", w.Code, w.Body.String())
	}
	w = notificationRequest(t, env.Router, "POST", "/api/houses/"+houseID+"/scheduled-events", admin, body)
	if w.Code != 201 {
		t.Fatalf("admin create got %d: %s", w.Code, w.Body.String())
	}
	if _, err := env.NotificationRepo.ListScheduledEvents(t.Context(), houseID); err != nil {
		t.Fatalf("repository list: %v", err)
	}
	w = notificationRequest(t, env.Router, "GET", "/api/houses/"+houseID+"/scheduled-events", monitor, nil)
	if w.Code != 200 {
		t.Fatalf("monitor list got %d: %s", w.Code, w.Body.String())
	}
}
func TestSubscriptionListNeverReturnsCapability(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()
	token := registerUser(t, env.Router, "Admin", "sub-admin@test", "password123")
	houseID := createHouse(t, env.Router, token, "House")
	body := map[string]any{"platform": "android_fcm", "token": "secret-registration-token", "device_label": "phone"}
	w := notificationRequest(t, env.Router, "POST", "/api/houses/"+houseID+"/notification-subscriptions", token, body)
	if w.Code != 201 {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	w = notificationRequest(t, env.Router, "GET", "/api/houses/"+houseID+"/notification-subscriptions", token, nil)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	if bytes.Contains(w.Body.Bytes(), []byte("secret-registration-token")) {
		t.Fatalf("capability leaked: %s", w.Body.String())
	}
}

func TestVAPIDPublicKeyEndpointIsPublicAndDoesNotExposePrivateConfiguration(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()
	env.Config.WebPushPublicKey = "public-vapid-key"
	env.Config.WebPushPrivateKey = "private-vapid-key"
	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/notifications/vapid-public-key", nil))
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte(`"public_key":"public-vapid-key"`)) {
		t.Fatalf("public VAPID endpoint: %d %s", w.Code, w.Body.String())
	}
	if bytes.Contains(w.Body.Bytes(), []byte("private-vapid-key")) {
		t.Fatalf("private key leaked: %s", w.Body.String())
	}
}

func TestScheduledEventUpdateIsLimitedToCreatorOrAdmin(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()
	admin := registerUser(t, env.Router, "Admin", "event-update-admin@test", "password123")
	creator := registerUser(t, env.Router, "Creator", "event-update-creator@test", "password123")
	other := registerUser(t, env.Router, "Other", "event-update-other@test", "password123")
	houseID := createHouse(t, env.Router, admin, "House")
	for _, member := range []struct{ token, id string }{{creator, getUserID(t, env.Router, creator)}, {other, getUserID(t, env.Router, other)}} {
		if _, err := env.DB.Exec(`INSERT INTO house_members(id,house_id,user_id,role) VALUES($1,$2,$3,'member')`, "membership-"+member.id, houseID, member.id); err != nil {
			t.Fatal(err)
		}
	}
	body := map[string]any{"title": "Bins", "timezone": "UTC", "dtstart_local": "2025-01-02T09:00:00", "rrule": "FREQ=DAILY;COUNT=2", "exdates": []string{}}
	created := notificationRequest(t, env.Router, "POST", "/api/houses/"+houseID+"/scheduled-events", creator, body)
	if created.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}
	var event struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &event); err != nil {
		t.Fatal(err)
	}
	denied := notificationRequest(t, env.Router, "PUT", "/api/houses/"+houseID+"/scheduled-events/"+event.ID, other, body)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("non-owner update: %d %s", denied.Code, denied.Body.String())
	}
	updated := notificationRequest(t, env.Router, "PUT", "/api/houses/"+houseID+"/scheduled-events/"+event.ID, admin, body)
	if updated.Code != http.StatusOK {
		t.Fatalf("admin update: %d %s", updated.Code, updated.Body.String())
	}
}
