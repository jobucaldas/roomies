package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/roomies/backend/internal/models"
)

func TestCreateNote(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()

	token := registerUser(t, router, "Alice", "note-test@test.com", "password123")
	houseID := createHouse(t, router, token, "Note House")

	body := models.CreateNoteRequest{Title: "Welcome", Content: "Welcome to the house!"}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/houses/"+houseID+"/notes", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var note models.Note
	json.NewDecoder(w.Body).Decode(&note)
	if note.Title != "Welcome" || note.Content != "Welcome to the house!" {
		t.Errorf("note content mismatch")
	}
}

func TestListNotes(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()

	token := registerUser(t, router, "Alice", "list-notes@test.com", "password123")
	houseID := createHouse(t, router, token, "Notes House")

	for i := 0; i < 2; i++ {
		body := models.CreateNoteRequest{Title: "Note", Content: "Content"}
		data, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/api/houses/"+houseID+"/notes", bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create note failed: %d", w.Code)
		}
	}

	req := httptest.NewRequest("GET", "/api/houses/"+houseID+"/notes", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var notes []models.Note
	json.NewDecoder(w.Body).Decode(&notes)
	if len(notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(notes))
	}
}

func TestUpdateNote(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()

	token := registerUser(t, router, "Alice", "upd-note@test.com", "password123")
	houseID := createHouse(t, router, token, "Update Note House")

	body := models.CreateNoteRequest{Title: "Original", Content: "Original content"}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/houses/"+houseID+"/notes", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var note models.Note
	json.NewDecoder(w.Body).Decode(&note)

	newTitle := "Updated"
	newContent := "Updated content"
	updateBody := models.UpdateNoteRequest{Title: &newTitle, Content: &newContent}
	updateData, _ := json.Marshal(updateBody)
	updateReq := httptest.NewRequest("PUT", "/api/houses/"+houseID+"/notes/"+note.ID, bytes.NewReader(updateData))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+token)
	updateW := httptest.NewRecorder()
	router.ServeHTTP(updateW, updateReq)
	if updateW.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", updateW.Code, updateW.Body.String())
	}

	var updated models.Note
	json.NewDecoder(updateW.Body).Decode(&updated)
	if updated.Title != "Updated" {
		t.Errorf("expected 'Updated', got '%s'", updated.Title)
	}
}

func TestDeleteNote(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()

	token := registerUser(t, router, "Alice", "del-note@test.com", "password123")
	houseID := createHouse(t, router, token, "Delete Note House")

	body := models.CreateNoteRequest{Title: "Delete Me", Content: "To be deleted"}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/houses/"+houseID+"/notes", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var note models.Note
	json.NewDecoder(w.Body).Decode(&note)

	delReq := httptest.NewRequest("DELETE", "/api/houses/"+houseID+"/notes/"+note.ID, nil)
	delReq.Header.Set("Authorization", "Bearer "+token)
	delW := httptest.NewRecorder()
	router.ServeHTTP(delW, delReq)
	if delW.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", delW.Code, delW.Body.String())
	}
}

func TestMonitorCannotCreateNote(t *testing.T) {
	router, _, _, cleanup := setupTest(t)
	defer cleanup()

	aliceToken := registerUser(t, router, "Alice", "mon-note@test.com", "password123")
	monitorToken := registerUser(t, router, "Mon", "mon-note2@test.com", "password123")
	monitorUserID := getUserID(t, router, monitorToken)

	houseID := createHouse(t, router, aliceToken, "Monitor Notes")

	addBody := map[string]string{"user_id": monitorUserID, "role": "monitor"}
	addData, _ := json.Marshal(addBody)
	addReq := httptest.NewRequest("POST", "/api/houses/"+houseID+"/members", bytes.NewReader(addData))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Authorization", "Bearer "+aliceToken)
	addW := httptest.NewRecorder()
	router.ServeHTTP(addW, addReq)
	if addW.Code != http.StatusCreated {
		t.Fatalf("add monitor failed: %d", addW.Code)
	}

	body := models.CreateNoteRequest{Title: "Test", Content: "Test content"}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/houses/"+houseID+"/notes", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+monitorToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}
