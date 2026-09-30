package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/roomies/backend/internal/middleware"
	"github.com/roomies/backend/internal/models"
	"github.com/roomies/backend/internal/repository"
)

type NotificationHandler struct {
	repo   *repository.NotificationRepository
	houses *repository.HouseRepository
}

func NewNotificationHandler(repo *repository.NotificationRepository, houses *repository.HouseRepository) *NotificationHandler {
	return &NotificationHandler{repo: repo, houses: houses}
}
func (h *NotificationHandler) member(w http.ResponseWriter, r *http.Request) (*models.HouseMember, bool) {
	m, err := loadHouseMember(r.Context(), h.houses, chi.URLParam(r, "id"), middleware.GetUserID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return nil, false
	}
	return m, true
}
func (h *NotificationHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.member(w, r); !ok {
		return
	}
	p, err := h.repo.GetPreferences(r.Context(), chi.URLParam(r, "id"), middleware.GetUserID(r.Context()))
	if err != nil {
		writeJSON(w, 500, models.ErrorResponse{Error: "failed to get preferences"})
		return
	}
	writeJSON(w, 200, p)
}
func (h *NotificationHandler) PutPreferences(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.member(w, r); !ok {
		return
	}
	var p models.NotificationPreferences
	if decodeJSON(w, r, &p) != nil {
		writeJSON(w, 400, models.ErrorResponse{Error: "invalid request body"})
		return
	}
	p.HouseID = chi.URLParam(r, "id")
	p.UserID = middleware.GetUserID(r.Context())
	if err := h.repo.PutPreferences(r.Context(), p); err != nil {
		writeJSON(w, 400, models.ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, 200, p)
}
func (h *NotificationHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.member(w, r); !ok {
		return
	}
	var req models.CreateNotificationSubscriptionRequest
	if decodeJSON(w, r, &req) != nil {
		writeJSON(w, 400, models.ErrorResponse{Error: "invalid request body"})
		return
	}
	subscription, err := h.repo.CreateSubscription(r.Context(), chi.URLParam(r, "id"), middleware.GetUserID(r.Context()), req)
	if err != nil {
		writeJSON(w, 400, models.ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, 201, subscription)
}
func (h *NotificationHandler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.member(w, r); !ok {
		return
	}
	values, err := h.repo.ListSubscriptions(r.Context(), chi.URLParam(r, "id"), middleware.GetUserID(r.Context()))
	if err != nil {
		writeJSON(w, 500, models.ErrorResponse{Error: "failed to list subscriptions"})
		return
	}
	if values == nil {
		values = []models.NotificationSubscription{}
	}
	writeJSON(w, 200, values)
}
func (h *NotificationHandler) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.member(w, r); !ok {
		return
	}
	err := h.repo.RevokeSubscription(r.Context(), chi.URLParam(r, "id"), middleware.GetUserID(r.Context()), chi.URLParam(r, "subscriptionId"))
	if errors.Is(err, repository.ErrNotificationNotFound) {
		writeJSON(w, 404, models.ErrorResponse{Error: "subscription not found"})
		return
	}
	if err != nil {
		writeJSON(w, 500, models.ErrorResponse{Error: "failed to revoke subscription"})
		return
	}
	w.WriteHeader(204)
}
func (h *NotificationHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.member(w, r); !ok {
		return
	}
	values, err := h.repo.ListScheduledEvents(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, 500, models.ErrorResponse{Error: "failed to list scheduled events"})
		return
	}
	if values == nil {
		values = []models.ScheduledHouseEvent{}
	}
	writeJSON(w, 200, values)
}
func (h *NotificationHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	member, ok := h.member(w, r)
	if !ok {
		return
	}
	if !canCreateHouseContent(member) {
		writeJSON(w, 403, models.ErrorResponse{Error: "monitors cannot create scheduled events"})
		return
	}
	event, ok := h.eventRequest(w, r)
	if !ok {
		return
	}
	event.HouseID = chi.URLParam(r, "id")
	event.CreatorID = middleware.GetUserID(r.Context())
	if err := h.repo.CreateScheduledEvent(r.Context(), event); err != nil {
		writeJSON(w, 400, models.ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, 201, event)
}
func (h *NotificationHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	member, ok := h.member(w, r)
	if !ok {
		return
	}
	existing, err := h.repo.GetScheduledEvent(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "eventId"))
	if err != nil {
		writeJSON(w, 404, models.ErrorResponse{Error: "scheduled event not found"})
		return
	}
	userID := middleware.GetUserID(r.Context())
	if !canMutateOwnedResource(member, existing.CreatorID, userID) {
		writeJSON(w, 403, models.ErrorResponse{Error: "only the creator or an admin can edit scheduled events"})
		return
	}
	event, ok := h.eventRequest(w, r)
	if !ok {
		return
	}
	event.ID = existing.ID
	event.HouseID = existing.HouseID
	event.CreatorID = existing.CreatorID
	if err := h.repo.UpdateScheduledEvent(r.Context(), event); err != nil {
		writeJSON(w, 400, models.ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, 200, event)
}
func (h *NotificationHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	member, ok := h.member(w, r)
	if !ok {
		return
	}
	existing, err := h.repo.GetScheduledEvent(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "eventId"))
	if err != nil {
		writeJSON(w, 404, models.ErrorResponse{Error: "scheduled event not found"})
		return
	}
	if !canMutateOwnedResource(member, existing.CreatorID, middleware.GetUserID(r.Context())) {
		writeJSON(w, 403, models.ErrorResponse{Error: "only the creator or an admin can delete scheduled events"})
		return
	}
	if err := h.repo.DeleteScheduledEvent(r.Context(), existing.HouseID, existing.ID); err != nil {
		writeJSON(w, 500, models.ErrorResponse{Error: "failed to delete scheduled event"})
		return
	}
	w.WriteHeader(204)
}
func (h *NotificationHandler) eventRequest(w http.ResponseWriter, r *http.Request) (*models.ScheduledHouseEvent, bool) {
	var req models.ScheduledEventRequest
	if decodeJSON(w, r, &req) != nil {
		writeJSON(w, 400, models.ErrorResponse{Error: "invalid request body"})
		return nil, false
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	event := &models.ScheduledHouseEvent{Title: strings.TrimSpace(req.Title), Timezone: req.Timezone, DTStartLocal: req.DTStartLocal, RRule: req.RRule, ExDates: req.ExDates, Enabled: enabled}
	if event.Title == "" || len(event.Title) > 120 {
		writeJSON(w, 400, models.ErrorResponse{Error: "title must contain 1 to 120 characters"})
		return nil, false
	}
	return event, true
}
