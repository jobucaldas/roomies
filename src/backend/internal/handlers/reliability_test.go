package handlers_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/roomies/backend/internal/models"
)

func TestInvitationCreateReplaysIdempotentResponse(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()
	adminToken := registerUser(t, env.Router, "Admin", "admin-invite@test.com", "password123")
	houseID := createHouse(t, env.Router, adminToken, "Invite House")

	first := createInvitation(t, env.Router, adminToken, houseID, models.CreateInvitationRequest{Email: "invitee@example.com", Role: "member"}, "invite-create-1")
	secondReq := invitationRequest(t, http.MethodPost, "/api/houses/"+houseID+"/invites", adminToken, models.CreateInvitationRequest{Email: "invitee@example.com", Role: "member"}, "invite-create-1")
	secondRes := httptest.NewRecorder()
	env.Router.ServeHTTP(secondRes, secondReq)
	if secondRes.Code != http.StatusCreated {
		t.Fatalf("expected replayed 201, got %d: %s", secondRes.Code, secondRes.Body.String())
	}
	if secondRes.Header().Get("X-Idempotent-Replay") != "true" {
		t.Fatalf("expected replay header, got %q", secondRes.Header().Get("X-Idempotent-Replay"))
	}
	var replayed models.HouseInvitation
	if err := json.NewDecoder(secondRes.Body).Decode(&replayed); err != nil {
		t.Fatal(err)
	}
	if replayed.ID != first.ID || replayed.ManualAcceptanceURL != first.ManualAcceptanceURL {
		t.Fatalf("expected same replayed invitation, got first=%+v replayed=%+v", first, replayed)
	}

	var inviteCount, jobCount, outboxCount int
	if err := env.DB.Get(&inviteCount, `SELECT COUNT(*) FROM house_invitations`); err != nil {
		t.Fatal(err)
	}
	if err := env.DB.Get(&jobCount, `SELECT COUNT(*) FROM durable_jobs`); err != nil {
		t.Fatal(err)
	}
	if err := env.DB.Get(&outboxCount, `SELECT COUNT(*) FROM outbox_messages`); err != nil {
		t.Fatal(err)
	}
	if inviteCount != 1 || jobCount != 1 || outboxCount != 1 {
		t.Fatalf("expected one persisted invite/job/outbox, got invites=%d jobs=%d outbox=%d", inviteCount, jobCount, outboxCount)
	}
}

func TestInvitationAcceptanceIsEnumerationSafeAndListOmitsManualURL(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()
	adminToken := registerUser(t, env.Router, "Admin", "admin-privacy@test.com", "password123")
	houseID := createHouse(t, env.Router, adminToken, "Privacy House")
	invite := createInvitation(t, env.Router, adminToken, houseID, models.CreateInvitationRequest{Email: "invitee@example.com", Role: "member"}, "")
	token := manualInvitationToken(t, invite.ManualAcceptanceURL)

	listReq := httptest.NewRequest(http.MethodGet, "/api/houses/"+houseID+"/invites", nil)
	listReq.Header.Set("Authorization", "Bearer "+adminToken)
	listRes := httptest.NewRecorder()
	env.Router.ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("list invites failed: %d %s", listRes.Code, listRes.Body.String())
	}
	if strings.Contains(listRes.Body.String(), "manual_acceptance_url") || strings.Contains(listRes.Body.String(), token) {
		t.Fatalf("list response leaked manual acceptance token: %s", listRes.Body.String())
	}

	wrongUserToken := registerUser(t, env.Router, "Wrong", "wrong-user@example.com", "password123")
	wrongAccept := acceptInvitation(t, env.Router, wrongUserToken, token)
	invalidTokenUser := registerUser(t, env.Router, "Invitee", "invitee@example.com", "password123")
	invalidAccept := acceptInvitation(t, env.Router, invalidTokenUser, "invalid-token")
	if wrongAccept.Code != http.StatusNotFound || invalidAccept.Code != http.StatusNotFound {
		t.Fatalf("expected enumeration-safe 404s, got wrong=%d invalid=%d", wrongAccept.Code, invalidAccept.Code)
	}
	if wrongAccept.Body.String() != invalidAccept.Body.String() {
		t.Fatalf("expected matching enumeration-safe errors, got wrong=%q invalid=%q", wrongAccept.Body.String(), invalidAccept.Body.String())
	}

	success := acceptInvitation(t, env.Router, invalidTokenUser, token)
	if success.Code != http.StatusOK {
		t.Fatalf("accept invitation failed: %d %s", success.Code, success.Body.String())
	}
	repeat := acceptInvitation(t, env.Router, invalidTokenUser, token)
	if repeat.Code != http.StatusOK {
		t.Fatalf("repeat accept should be idempotent: %d %s", repeat.Code, repeat.Body.String())
	}
}

func TestInvitationRevokeIsHouseScoped(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()
	adminAToken := registerUser(t, env.Router, "Admin A", "admin-a@example.com", "password123")
	adminBToken := registerUser(t, env.Router, "Admin B", "admin-b@example.com", "password123")
	houseA := createHouse(t, env.Router, adminAToken, "House A")
	houseB := createHouse(t, env.Router, adminBToken, "House B")
	invite := createInvitation(t, env.Router, adminAToken, houseA, models.CreateInvitationRequest{Email: "invitee@example.com", Role: "member"}, "")

	deleteReq := invitationRequest(t, http.MethodDelete, "/api/houses/"+houseB+"/invites/"+invite.ID, adminBToken, nil, "")
	deleteRes := httptest.NewRecorder()
	env.Router.ServeHTTP(deleteRes, deleteReq)
	if deleteRes.Code != http.StatusNotFound {
		t.Fatalf("expected cross-house revoke to miss invite, got %d: %s", deleteRes.Code, deleteRes.Body.String())
	}
}

func TestHouseEventsSSEReconnectUsesCursorAndPollingFallback(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()
	adminToken := registerUser(t, env.Router, "Admin", "admin-events@example.com", "password123")
	houseID := createHouse(t, env.Router, adminToken, "Events House")
	ts := httptest.NewServer(env.Router)
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	streamReq, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/houses/"+houseID+"/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	streamReq.Header.Set("Authorization", "Bearer "+adminToken)
	streamReq.Header.Set("Accept", "text/event-stream")
	streamRes, err := http.DefaultClient.Do(streamReq)
	if err != nil {
		t.Fatal(err)
	}
	defer streamRes.Body.Close()
	if streamRes.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(streamRes.Body)
		t.Fatalf("expected stream 200, got %d: %s", streamRes.StatusCode, string(body))
	}

	firstInvite := createInvitation(t, env.Router, adminToken, houseID, models.CreateInvitationRequest{Email: "first@example.com", Role: "member"}, "")
	firstEvent := readSSEEvent(t, streamRes.Body, 5*time.Second)
	if firstEvent.ResourceID != firstInvite.ID || firstEvent.EventType != "house.invitation.created" {
		t.Fatalf("unexpected first event: %+v", firstEvent)
	}
	cancel()
	streamRes.Body.Close()

	secondInvite := createInvitation(t, env.Router, adminToken, houseID, models.CreateInvitationRequest{Email: "second@example.com", Role: "member"}, "")
	pollReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/houses/%s/events?cursor=%s", houseID, firstEvent.Cursor), nil)
	pollReq.Header.Set("Authorization", "Bearer "+adminToken)
	pollRes := httptest.NewRecorder()
	env.Router.ServeHTTP(pollRes, pollReq)
	if pollRes.Code != http.StatusOK {
		t.Fatalf("poll fallback failed: %d %s", pollRes.Code, pollRes.Body.String())
	}
	var pollBody models.HouseEventsResponse
	if err := json.NewDecoder(pollRes.Body).Decode(&pollBody); err != nil {
		t.Fatal(err)
	}
	if len(pollBody.Events) != 1 || pollBody.Events[0].ResourceID != secondInvite.ID {
		t.Fatalf("expected polling cursor to return only second event, got %+v", pollBody.Events)
	}

	reconnectReq, err := http.NewRequest(http.MethodGet, ts.URL+"/api/houses/"+houseID+"/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	reconnectReq.Header.Set("Authorization", "Bearer "+adminToken)
	reconnectReq.Header.Set("Accept", "text/event-stream")
	reconnectReq.Header.Set("Last-Event-ID", firstEvent.Cursor)
	reconnectRes, err := http.DefaultClient.Do(reconnectReq)
	if err != nil {
		t.Fatal(err)
	}
	defer reconnectRes.Body.Close()
	secondEvent := readSSEEvent(t, reconnectRes.Body, 5*time.Second)
	if secondEvent.ResourceID != secondInvite.ID || secondEvent.Cursor == firstEvent.Cursor {
		t.Fatalf("expected reconnect to resume after cursor, got %+v", secondEvent)
	}
}

func TestHouseEventsSSEClosesAfterMemberRemoval(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()
	adminToken := registerUser(t, env.Router, "Admin", "admin-events-removal@example.com", "password123")
	memberToken := registerUser(t, env.Router, "Member", "member-events-removal@example.com", "password123")
	memberID := getUserID(t, env.Router, memberToken)
	houseID := createHouse(t, env.Router, adminToken, "Events Removal House")
	addBody, _ := json.Marshal(map[string]string{"user_id": memberID, "role": "member"})
	addReq := httptest.NewRequest(http.MethodPost, "/api/houses/"+houseID+"/members", bytes.NewReader(addBody))
	addReq.Header.Set("Authorization", "Bearer "+adminToken)
	addReq.Header.Set("Content-Type", "application/json")
	addRes := httptest.NewRecorder()
	env.Router.ServeHTTP(addRes, addReq)
	if addRes.Code != http.StatusCreated {
		t.Fatalf("add member failed: %d %s", addRes.Code, addRes.Body.String())
	}

	ts := httptest.NewServer(env.Router)
	defer ts.Close()
	streamReq, err := http.NewRequest(http.MethodGet, ts.URL+"/api/houses/"+houseID+"/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	streamReq.Header.Set("Authorization", "Bearer "+memberToken)
	streamReq.Header.Set("Accept", "text/event-stream")
	streamRes, err := http.DefaultClient.Do(streamReq)
	if err != nil {
		t.Fatal(err)
	}
	defer streamRes.Body.Close()
	if streamRes.StatusCode != http.StatusOK {
		t.Fatalf("expected stream 200, got %d", streamRes.StatusCode)
	}

	removeReq := httptest.NewRequest(http.MethodDelete, "/api/houses/"+houseID+"/members/"+memberID, nil)
	removeReq.Header.Set("Authorization", "Bearer "+adminToken)
	removeRes := httptest.NewRecorder()
	env.Router.ServeHTTP(removeRes, removeReq)
	if removeRes.Code != http.StatusOK {
		t.Fatalf("remove member failed: %d %s", removeRes.Code, removeRes.Body.String())
	}

	streamDone := make(chan error, 1)
	go func() {
		_, readErr := io.Copy(io.Discard, streamRes.Body)
		streamDone <- readErr
	}()
	select {
	case <-streamDone:
	case <-time.After(2 * time.Second):
		t.Fatal("SSE stream remained open after member removal")
	}
}

func createInvitation(t *testing.T, router http.Handler, token, houseID string, body models.CreateInvitationRequest, idempotencyKey string) models.HouseInvitation {
	t.Helper()
	res := httptest.NewRecorder()
	router.ServeHTTP(res, invitationRequest(t, http.MethodPost, "/api/houses/"+houseID+"/invites", token, body, idempotencyKey))
	if res.Code != http.StatusCreated {
		t.Fatalf("create invite failed: %d %s", res.Code, res.Body.String())
	}
	var invite models.HouseInvitation
	if err := json.NewDecoder(res.Body).Decode(&invite); err != nil {
		t.Fatal(err)
	}
	return invite
}

func acceptInvitation(t *testing.T, router http.Handler, token, invitationToken string) *httptest.ResponseRecorder {
	t.Helper()
	res := httptest.NewRecorder()
	router.ServeHTTP(res, invitationRequest(t, http.MethodPost, "/api/invitations/accept", token, models.AcceptInvitationRequest{Token: invitationToken}, ""))
	return res
}

func invitationRequest(t *testing.T, method, path, token string, body any, idempotencyKey string) *http.Request {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	return req
}

func manualInvitationToken(t *testing.T, rawURL string) string {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	token := parsed.Query().Get("token")
	if token == "" {
		t.Fatalf("missing token in %q", rawURL)
	}
	return token
}

func readSSEEvent(t *testing.T, body io.Reader, timeout time.Duration) models.HouseEvent {
	t.Helper()
	reader := bufio.NewReader(body)
	type result struct {
		event models.HouseEvent
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		var dataLine string
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				ch <- result{err: err}
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				if dataLine == "" {
					continue
				}
				var event models.HouseEvent
				if err := json.Unmarshal([]byte(dataLine), &event); err != nil {
					ch <- result{err: err}
					return
				}
				ch <- result{event: event}
				return
			}
			if strings.HasPrefix(line, "data: ") {
				dataLine = strings.TrimPrefix(line, "data: ")
			}
		}
	}()
	select {
	case got := <-ch:
		if got.err != nil {
			t.Fatal(got.err)
		}
		return got.event
	case <-time.After(timeout):
		t.Fatal("timed out waiting for SSE event")
		return models.HouseEvent{}
	}
}
