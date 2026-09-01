package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/roomies/backend/internal/middleware"
	"github.com/roomies/backend/internal/models"
	"github.com/roomies/backend/internal/repository"
)

type NoteHandler struct {
	noteRepo  *repository.NoteRepository
	houseRepo *repository.HouseRepository
}

func NewNoteHandler(noteRepo *repository.NoteRepository, houseRepo *repository.HouseRepository) *NoteHandler {
	return &NoteHandler{noteRepo: noteRepo, houseRepo: houseRepo}
}

func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())
	if _, err := h.houseRepo.GetMember(r.Context(), houseID, userID); err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}
	notes, err := h.noteRepo.ListByHouse(r.Context(), houseID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to list notes"})
		return
	}
	if notes == nil {
		notes = []models.Note{}
	}
	writeJSON(w, http.StatusOK, notes)
}

func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())
	member, err := h.houseRepo.GetMember(r.Context(), houseID, userID)
	if err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}
	if member.Role == "monitor" {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "monitors cannot create notes"})
		return
	}

	var req models.CreateNoteRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	if req.Title == "" || req.Content == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "title and content are required"})
		return
	}

	now := time.Now()
	note := &models.Note{
		ID: models.NewID(), HouseID: houseID, AuthorID: userID,
		Title: req.Title, Content: req.Content, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.noteRepo.Create(r.Context(), note); err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to create note"})
		return
	}
	note.AuthorName = member.UserName
	writeJSON(w, http.StatusCreated, note)
}

func (h *NoteHandler) Get(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	noteID := chi.URLParam(r, "nid")
	userID := middleware.GetUserID(r.Context())
	if _, err := h.houseRepo.GetMember(r.Context(), houseID, userID); err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}
	note, err := h.noteRepo.GetByHouseAndID(r.Context(), houseID, noteID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: "note not found"})
		return
	}
	writeJSON(w, http.StatusOK, note)
}

func (h *NoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	noteID := chi.URLParam(r, "nid")
	userID := middleware.GetUserID(r.Context())
	member, err := h.houseRepo.GetMember(r.Context(), houseID, userID)
	if err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}
	note, err := h.noteRepo.GetByHouseAndID(r.Context(), houseID, noteID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: "note not found"})
		return
	}
	if member.Role == "monitor" || (note.AuthorID != userID && member.Role != "admin") {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "only the author or an admin can update this note"})
		return
	}

	var req models.UpdateNoteRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}
	if req.Title != nil {
		note.Title = strings.TrimSpace(*req.Title)
		if note.Title == "" {
			writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "title is required"})
			return
		}
	}
	if req.Content != nil {
		note.Content = strings.TrimSpace(*req.Content)
		if note.Content == "" {
			writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "content is required"})
			return
		}
	}
	note.UpdatedAt = time.Now()
	if err := h.noteRepo.Update(r.Context(), note); err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to update note"})
		return
	}
	writeJSON(w, http.StatusOK, note)
}

func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	noteID := chi.URLParam(r, "nid")
	userID := middleware.GetUserID(r.Context())
	member, err := h.houseRepo.GetMember(r.Context(), houseID, userID)
	if err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}
	note, err := h.noteRepo.GetByHouseAndID(r.Context(), houseID, noteID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: "note not found"})
		return
	}
	if member.Role == "monitor" || (note.AuthorID != userID && member.Role != "admin") {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "only the author or an admin can delete this note"})
		return
	}
	if err := h.noteRepo.Delete(r.Context(), houseID, noteID); err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to delete note"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "note deleted"})
}
