package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/roomies/backend/internal/middleware"
	"github.com/roomies/backend/internal/models"
	"github.com/roomies/backend/internal/repository"
)

type HouseholdHandler struct {
	repo   *repository.HouseholdRepository
	houses *repository.HouseRepository
}

func NewHouseholdHandler(repo *repository.HouseholdRepository, houses *repository.HouseRepository) *HouseholdHandler {
	return &HouseholdHandler{repo: repo, houses: houses}
}
func (h *HouseholdHandler) member(w http.ResponseWriter, r *http.Request) (*models.HouseMember, bool) {
	m, err := loadHouseMember(r.Context(), h.houses, chi.URLParam(r, "id"), middleware.GetUserID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not an active house member"})
		return nil, false
	}
	return m, true
}
func (h *HouseholdHandler) mutator(w http.ResponseWriter, r *http.Request) (*models.HouseMember, bool) {
	m, ok := h.member(w, r)
	if !ok {
		return nil, false
	}
	if isHouseMonitor(m) {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "monitors have read-only access"})
		return nil, false
	}
	return m, true
}
func householdError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrHouseholdNotFound):
		writeJSON(w, 404, models.ErrorResponse{Error: "resource not found"})
	case errors.Is(err, repository.ErrVersionConflict), errors.Is(err, repository.ErrAlreadyCompleted):
		writeJSON(w, 409, models.ErrorResponse{Error: err.Error()})
	case errors.Is(err, repository.ErrInvalidOccurrence):
		writeJSON(w, 400, models.ErrorResponse{Error: err.Error()})
	default:
		writeJSON(w, 400, models.ErrorResponse{Error: err.Error()})
	}
}
func versionParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	v, err := strconv.ParseInt(r.URL.Query().Get("version"), 10, 64)
	if err != nil || v < 1 {
		writeJSON(w, 400, models.ErrorResponse{Error: "version query parameter must be a positive integer"})
		return 0, false
	}
	return v, true
}

func (h *HouseholdHandler) ListGroceries(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.member(w, r); !ok {
		return
	}
	values, err := h.repo.ListGroceries(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, 500, models.ErrorResponse{Error: "failed to list groceries"})
		return
	}
	if values == nil {
		values = []models.GroceryItem{}
	}
	writeJSON(w, 200, values)
}
func (h *HouseholdHandler) GetGrocery(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.member(w, r); !ok {
		return
	}
	item, err := h.repo.GetGrocery(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "groceryId"))
	if err != nil {
		householdError(w, err)
		return
	}
	writeJSON(w, 200, item)
}
func groceryRequest(w http.ResponseWriter, r *http.Request) (models.GroceryRequest, bool) {
	var req models.GroceryRequest
	if decodeJSON(w, r, &req) != nil {
		writeJSON(w, 400, models.ErrorResponse{Error: "invalid request body"})
		return req, false
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Quantity = strings.TrimSpace(req.Quantity)
	req.Unit = strings.TrimSpace(req.Unit)
	req.Note = strings.TrimSpace(req.Note)
	if req.Name == "" || len([]rune(req.Name)) > 160 || len([]rune(req.Quantity)) > 40 || len([]rune(req.Unit)) > 40 || len([]rune(req.Note)) > 1000 {
		writeJSON(w, 400, models.ErrorResponse{Error: "invalid grocery field length"})
		return req, false
	}
	return req, true
}
func (h *HouseholdHandler) CreateGrocery(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.mutator(w, r); !ok {
		return
	}
	req, ok := groceryRequest(w, r)
	if !ok {
		return
	}
	item := &models.GroceryItem{HouseID: chi.URLParam(r, "id"), Name: req.Name, Quantity: req.Quantity, Unit: req.Unit, Note: req.Note, AssigneeID: req.AssigneeID, Position: req.Position}
	if err := h.repo.CreateGrocery(r.Context(), item, middleware.GetUserID(r.Context())); err != nil {
		householdError(w, err)
		return
	}
	writeJSON(w, 201, item)
}
func (h *HouseholdHandler) UpdateGrocery(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.mutator(w, r); !ok {
		return
	}
	req, ok := groceryRequest(w, r)
	if !ok {
		return
	}
	if req.Version < 1 {
		writeJSON(w, 400, models.ErrorResponse{Error: "version is required"})
		return
	}
	item := &models.GroceryItem{ID: chi.URLParam(r, "groceryId"), HouseID: chi.URLParam(r, "id"), Name: req.Name, Quantity: req.Quantity, Unit: req.Unit, Note: req.Note, AssigneeID: req.AssigneeID, Position: req.Position, Version: req.Version}
	if err := h.repo.UpdateGrocery(r.Context(), item, middleware.GetUserID(r.Context())); err != nil {
		householdError(w, err)
		return
	}
	writeJSON(w, 200, item)
}
func (h *HouseholdHandler) ToggleGrocery(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.mutator(w, r); !ok {
		return
	}
	var req models.GroceryToggleRequest
	if decodeJSON(w, r, &req) != nil || req.Version < 1 {
		writeJSON(w, 400, models.ErrorResponse{Error: "checked and version are required"})
		return
	}
	item, err := h.repo.ToggleGrocery(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "groceryId"), middleware.GetUserID(r.Context()), req.Checked, req.Version)
	if err != nil {
		householdError(w, err)
		return
	}
	writeJSON(w, 200, item)
}
func (h *HouseholdHandler) DeleteGrocery(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.mutator(w, r); !ok {
		return
	}
	v, ok := versionParam(w, r)
	if !ok {
		return
	}
	if err := h.repo.DeleteGrocery(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "groceryId"), middleware.GetUserID(r.Context()), v); err != nil {
		householdError(w, err)
		return
	}
	w.WriteHeader(204)
}

func choreRequest(w http.ResponseWriter, r *http.Request) (models.ChoreRequest, bool) {
	var req models.ChoreRequest
	if decodeJSON(w, r, &req) != nil {
		writeJSON(w, 400, models.ErrorResponse{Error: "invalid request body"})
		return req, false
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	if req.Title == "" || len([]rune(req.Title)) > 160 || len([]rune(req.Description)) > 4000 {
		writeJSON(w, 400, models.ErrorResponse{Error: "invalid chore field length"})
		return req, false
	}
	return req, true
}
func (h *HouseholdHandler) ListChores(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.member(w, r); !ok {
		return
	}
	out, err := h.repo.ListChores(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, 500, models.ErrorResponse{Error: "failed to list chores"})
		return
	}
	if out == nil {
		out = []models.Chore{}
	}
	writeJSON(w, 200, out)
}
func (h *HouseholdHandler) GetChore(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.member(w, r); !ok {
		return
	}
	out, err := h.repo.GetChore(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "choreId"))
	if err != nil {
		householdError(w, err)
		return
	}
	writeJSON(w, 200, out)
}
func (h *HouseholdHandler) CreateChore(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.mutator(w, r); !ok {
		return
	}
	req, ok := choreRequest(w, r)
	if !ok {
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	item := &models.Chore{HouseID: chi.URLParam(r, "id"), Title: req.Title, Description: req.Description, AssigneeID: req.AssigneeID, Timezone: req.Timezone, DueLocal: req.DueLocal, RRule: req.RRule, ExDates: req.ExDates, Enabled: enabled}
	if err := h.repo.SaveChore(r.Context(), item, middleware.GetUserID(r.Context()), true); err != nil {
		householdError(w, err)
		return
	}
	writeJSON(w, 201, item)
}
func (h *HouseholdHandler) UpdateChore(w http.ResponseWriter, r *http.Request) {
	member, ok := h.mutator(w, r)
	if !ok {
		return
	}
	existing, err := h.repo.GetChore(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "choreId"))
	if err != nil {
		householdError(w, err)
		return
	}
	if !canMutateOwnedResource(member, existing.CreatorID, middleware.GetUserID(r.Context())) {
		writeJSON(w, 403, models.ErrorResponse{Error: "only the creator or an admin can edit chores"})
		return
	}
	req, ok := choreRequest(w, r)
	if !ok {
		return
	}
	if req.Version < 1 {
		writeJSON(w, 400, models.ErrorResponse{Error: "version is required"})
		return
	}
	enabled := existing.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	item := &models.Chore{ID: existing.ID, HouseID: existing.HouseID, CreatorID: existing.CreatorID, Title: req.Title, Description: req.Description, AssigneeID: req.AssigneeID, Timezone: req.Timezone, DueLocal: req.DueLocal, RRule: req.RRule, ExDates: req.ExDates, Enabled: enabled, Version: req.Version}
	if err = h.repo.SaveChore(r.Context(), item, middleware.GetUserID(r.Context()), false); err != nil {
		householdError(w, err)
		return
	}
	writeJSON(w, 200, item)
}
func (h *HouseholdHandler) DeleteChore(w http.ResponseWriter, r *http.Request) {
	member, ok := h.mutator(w, r)
	if !ok {
		return
	}
	existing, err := h.repo.GetChore(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "choreId"))
	if err != nil {
		householdError(w, err)
		return
	}
	if !canMutateOwnedResource(member, existing.CreatorID, middleware.GetUserID(r.Context())) {
		writeJSON(w, 403, models.ErrorResponse{Error: "only the creator or an admin can delete chores"})
		return
	}
	v, ok := versionParam(w, r)
	if !ok {
		return
	}
	if err = h.repo.DeleteChore(r.Context(), existing.HouseID, existing.ID, middleware.GetUserID(r.Context()), v); err != nil {
		householdError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (h *HouseholdHandler) CompleteChore(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.mutator(w, r); !ok {
		return
	}
	chore, err := h.repo.GetChore(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "choreId"))
	if err != nil {
		householdError(w, err)
		return
	}
	var req models.CompleteChoreRequest
	if decodeJSON(w, r, &req) != nil || req.OccurrenceAt.IsZero() {
		writeJSON(w, 400, models.ErrorResponse{Error: "occurrence_at is required"})
		return
	}
	out, err := h.repo.CompleteChore(r.Context(), chore, middleware.GetUserID(r.Context()), req.OccurrenceAt)
	if err != nil {
		householdError(w, err)
		return
	}
	writeJSON(w, 201, out)
}
func (h *HouseholdHandler) ListChoreCompletions(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.member(w, r); !ok {
		return
	}
	if _, err := h.repo.GetChore(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "choreId")); err != nil {
		householdError(w, err)
		return
	}
	out, err := h.repo.ListChoreCompletions(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "choreId"))
	if err != nil {
		writeJSON(w, 500, models.ErrorResponse{Error: "failed to list completions"})
		return
	}
	if out == nil {
		out = []models.ChoreCompletion{}
	}
	writeJSON(w, 200, out)
}

func calendarRequest(w http.ResponseWriter, r *http.Request) (models.CalendarEventRequest, bool) {
	var req models.CalendarEventRequest
	if decodeJSON(w, r, &req) != nil {
		writeJSON(w, 400, models.ErrorResponse{Error: "invalid request body"})
		return req, false
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	if req.Title == "" || len([]rune(req.Title)) > 160 || len([]rune(req.Description)) > 4000 {
		writeJSON(w, 400, models.ErrorResponse{Error: "invalid calendar field length"})
		return req, false
	}
	return req, true
}
func (h *HouseholdHandler) ListCalendar(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.member(w, r); !ok {
		return
	}
	out, err := h.repo.ListCalendar(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, 500, models.ErrorResponse{Error: "failed to list calendar events"})
		return
	}
	if out == nil {
		out = []models.CalendarEvent{}
	}
	writeJSON(w, 200, out)
}
func (h *HouseholdHandler) GetCalendar(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.member(w, r); !ok {
		return
	}
	out, err := h.repo.GetCalendar(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "calendarId"))
	if err != nil {
		householdError(w, err)
		return
	}
	writeJSON(w, 200, out)
}
func (h *HouseholdHandler) CreateCalendar(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.mutator(w, r); !ok {
		return
	}
	req, ok := calendarRequest(w, r)
	if !ok {
		return
	}
	event := &models.CalendarEvent{HouseID: chi.URLParam(r, "id"), Title: req.Title, Description: req.Description, Timezone: req.Timezone, StartLocal: req.StartLocal, EndLocal: req.EndLocal, AllDay: req.AllDay, RRule: req.RRule, ExDates: req.ExDates}
	if err := h.repo.SaveCalendar(r.Context(), event, middleware.GetUserID(r.Context()), true); err != nil {
		householdError(w, err)
		return
	}
	writeJSON(w, 201, event)
}
func (h *HouseholdHandler) UpdateCalendar(w http.ResponseWriter, r *http.Request) {
	member, ok := h.mutator(w, r)
	if !ok {
		return
	}
	existing, err := h.repo.GetCalendar(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "calendarId"))
	if err != nil {
		householdError(w, err)
		return
	}
	if !canMutateOwnedResource(member, existing.CreatorID, middleware.GetUserID(r.Context())) {
		writeJSON(w, 403, models.ErrorResponse{Error: "only the creator or an admin can edit calendar events"})
		return
	}
	req, ok := calendarRequest(w, r)
	if !ok {
		return
	}
	if req.Version < 1 {
		writeJSON(w, 400, models.ErrorResponse{Error: "version is required"})
		return
	}
	event := &models.CalendarEvent{ID: existing.ID, HouseID: existing.HouseID, CreatorID: existing.CreatorID, Title: req.Title, Description: req.Description, Timezone: req.Timezone, StartLocal: req.StartLocal, EndLocal: req.EndLocal, AllDay: req.AllDay, RRule: req.RRule, ExDates: req.ExDates, Version: req.Version}
	if err = h.repo.SaveCalendar(r.Context(), event, middleware.GetUserID(r.Context()), false); err != nil {
		householdError(w, err)
		return
	}
	writeJSON(w, 200, event)
}
func (h *HouseholdHandler) DeleteCalendar(w http.ResponseWriter, r *http.Request) {
	member, ok := h.mutator(w, r)
	if !ok {
		return
	}
	existing, err := h.repo.GetCalendar(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "calendarId"))
	if err != nil {
		householdError(w, err)
		return
	}
	if !canMutateOwnedResource(member, existing.CreatorID, middleware.GetUserID(r.Context())) {
		writeJSON(w, 403, models.ErrorResponse{Error: "only the creator or an admin can delete calendar events"})
		return
	}
	v, ok := versionParam(w, r)
	if !ok {
		return
	}
	if err = h.repo.DeleteCalendar(r.Context(), existing.HouseID, existing.ID, middleware.GetUserID(r.Context()), v); err != nil {
		householdError(w, err)
		return
	}
	w.WriteHeader(204)
}

func (h *HouseholdHandler) ListChat(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.member(w, r); !ok {
		return
	}
	before := int64(0)
	var err error
	if raw := r.URL.Query().Get("before"); raw != "" {
		before, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || before < 1 {
			writeJSON(w, 400, models.ErrorResponse{Error: "invalid before cursor"})
			return
		}
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			writeJSON(w, 400, models.ErrorResponse{Error: "limit must be between 1 and 100"})
			return
		}
	}
	page, err := h.repo.ListChat(r.Context(), chi.URLParam(r, "id"), before, limit)
	if err != nil {
		writeJSON(w, 500, models.ErrorResponse{Error: "failed to list chat"})
		return
	}
	writeJSON(w, 200, page)
}
func (h *HouseholdHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.mutator(w, r); !ok {
		return
	}
	var req models.ChatMessageRequest
	if decodeJSON(w, r, &req) != nil {
		writeJSON(w, 400, models.ErrorResponse{Error: "invalid request body"})
		return
	}
	out, err := h.repo.CreateChat(r.Context(), chi.URLParam(r, "id"), middleware.GetUserID(r.Context()), req.Body)
	if err != nil {
		householdError(w, err)
		return
	}
	writeJSON(w, 201, out)
}
func (h *HouseholdHandler) EditChat(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.mutator(w, r); !ok {
		return
	}
	var req models.ChatMessageRequest
	if decodeJSON(w, r, &req) != nil {
		writeJSON(w, 400, models.ErrorResponse{Error: "invalid request body"})
		return
	}
	out, err := h.repo.EditChat(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "messageId"), middleware.GetUserID(r.Context()), req.Body)
	if err != nil {
		householdError(w, err)
		return
	}
	writeJSON(w, 200, out)
}
func (h *HouseholdHandler) DeleteChat(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.mutator(w, r); !ok {
		return
	}
	if err := h.repo.DeleteChat(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "messageId"), middleware.GetUserID(r.Context())); err != nil {
		householdError(w, err)
		return
	}
	w.WriteHeader(204)
}
