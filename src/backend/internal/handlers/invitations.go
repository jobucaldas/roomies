package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	roomiesclock "github.com/roomies/backend/internal/clock"
	"github.com/roomies/backend/internal/middleware"
	"github.com/roomies/backend/internal/models"
	"github.com/roomies/backend/internal/repository"
)

type InvitationHandler struct {
	houseRepo       *repository.HouseRepository
	reliabilityRepo *repository.ReliabilityRepository
	clock           roomiesclock.Clock
	publicBaseURL   string
	invitationTTL   time.Duration
}

func NewInvitationHandler(houseRepo *repository.HouseRepository, reliabilityRepo *repository.ReliabilityRepository, clk roomiesclock.Clock, publicBaseURL string, invitationTTL time.Duration) *InvitationHandler {
	if clk == nil {
		clk = roomiesclock.RealClock{}
	}
	return &InvitationHandler{
		houseRepo:       houseRepo,
		reliabilityRepo: reliabilityRepo,
		clock:           clk,
		publicBaseURL:   strings.TrimRight(publicBaseURL, "/"),
		invitationTTL:   invitationTTL,
	}
}

func (h *InvitationHandler) Create(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())
	member, err := loadHouseMember(r.Context(), h.houseRepo, houseID, userID)
	if err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}
	if !isHouseAdmin(member) {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "only admins can invite members"})
		return
	}

	var req models.CreateInvitationRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Role == "" {
		req.Role = "member"
	}
	if req.Role != "admin" && req.Role != "member" && req.Role != "monitor" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid role: must be admin, member, or monitor"})
		return
	}
	address, err := mail.ParseAddress(req.Email)
	if err != nil || address.Address != req.Email {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "email must be valid"})
		return
	}

	invite, token, err := h.reliabilityRepo.CreateInvitation(r.Context(), repository.CreateInvitationParams{
		HouseID:       houseID,
		ActorID:       userID,
		Email:         req.Email,
		Role:          req.Role,
		ExpiresAt:     h.clock.Now().Add(h.invitationTTL),
		PublicBaseURL: h.publicBaseURL,
	})
	if err == repository.ErrInvitationPendingExists {
		writeJSON(w, http.StatusConflict, models.ErrorResponse{Error: err.Error()})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to create invitation"})
		return
	}
	invite.ManualAcceptanceURL, err = repository.BuildInvitationAcceptanceURL(h.publicBaseURL, token)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to construct invitation link"})
		return
	}
	// The URL contains a bearer token. It is returned only in this initial response;
	// idempotent replays retain the invitation result but deliberately omit the URL.
	w.Header().Set("X-Invitation-Token-Response", "one-time")
	writeJSON(w, http.StatusCreated, invite)
}

func (h *InvitationHandler) List(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())
	member, err := loadHouseMember(r.Context(), h.houseRepo, houseID, userID)
	if err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}
	if !isHouseAdmin(member) {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "only admins can list invitations"})
		return
	}
	invites, err := h.reliabilityRepo.ListInvitations(r.Context(), houseID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to list invitations"})
		return
	}
	if invites == nil {
		invites = []models.HouseInvitation{}
	}
	writeJSON(w, http.StatusOK, invites)
}

func (h *InvitationHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	inviteID := chi.URLParam(r, "inviteId")
	userID := middleware.GetUserID(r.Context())
	member, err := loadHouseMember(r.Context(), h.houseRepo, houseID, userID)
	if err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}
	if !isHouseAdmin(member) {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "only admins can revoke invitations"})
		return
	}
	invite, err := h.reliabilityRepo.RevokeInvitation(r.Context(), houseID, inviteID, userID)
	if err == repository.ErrInvitationUnavailable {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: "invitation unavailable"})
		return
	}
	if err == repository.ErrInvitationStateConflict {
		writeJSON(w, http.StatusConflict, models.ErrorResponse{Error: "invitation state change rejected"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to revoke invitation"})
		return
	}
	writeJSON(w, http.StatusOK, invite)
}

func (h *InvitationHandler) Accept(w http.ResponseWriter, r *http.Request) {
	var req models.AcceptInvitationRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}
	req.Token = strings.TrimSpace(req.Token)
	if req.Token == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "token is required"})
		return
	}
	invite, err := h.reliabilityRepo.AcceptInvitation(r.Context(), req.Token, middleware.GetUserID(r.Context()), middleware.GetUserEmail(r.Context()))
	if err == repository.ErrInvitationUnavailable {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: "invitation unavailable"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to accept invitation"})
		return
	}
	writeJSON(w, http.StatusOK, models.InvitationAcceptanceResponse{Invitation: *invite})
}

type HouseEventsHandler struct {
	houseRepo        *repository.HouseRepository
	reliabilityRepo  *repository.ReliabilityRepository
	pollInterval     time.Duration
	maxEventsPerPage int
}

func NewHouseEventsHandler(houseRepo *repository.HouseRepository, reliabilityRepo *repository.ReliabilityRepository, pollInterval time.Duration) *HouseEventsHandler {
	if pollInterval <= 0 {
		pollInterval = 250 * time.Millisecond
	}
	return &HouseEventsHandler{houseRepo: houseRepo, reliabilityRepo: reliabilityRepo, pollInterval: pollInterval, maxEventsPerPage: 100}
}

func (h *HouseEventsHandler) List(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())
	cursor, err := parseEventCursor(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	if acceptsEventStream(r) {
		h.stream(w, r, houseID, userID, cursor)
		return
	}
	if _, err := loadHouseMember(r.Context(), h.houseRepo, houseID, userID); err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}
	events, nextCursor, err := h.reliabilityRepo.ListHouseEvents(r.Context(), houseID, cursor, h.maxEventsPerPage)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to load house events"})
		return
	}
	writeJSON(w, http.StatusOK, models.HouseEventsResponse{Events: events, NextCursor: nextCursor})
}

func (h *HouseEventsHandler) stream(w http.ResponseWriter, r *http.Request, houseID, userID string, cursor int64) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "streaming unsupported"})
		return
	}
	currentCursor := cursor
	started := false
	send := func(events []models.HouseEvent) error {
		for _, event := range events {
			payload, err := json.Marshal(event)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", event.Cursor, event.EventType, payload); err != nil {
				return err
			}
			currentCursor64, _ := parseCursorValue(event.Cursor)
			currentCursor = currentCursor64
		}
		flusher.Flush()
		return nil
	}
	snapshotAndSend := func(initial bool) (int, error) {
		eventCount := 0
		err := h.reliabilityRepo.WithAuthorizedHouseEvents(r.Context(), houseID, userID, currentCursor, h.maxEventsPerPage, func(events []models.HouseEvent) error {
			eventCount = len(events)
			if err := setSSEWriteDeadline(w); err != nil {
				return err
			}
			if initial && !started {
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Connection", "keep-alive")
				w.WriteHeader(http.StatusOK)
				started = true
			}
			if len(events) == 0 {
				if initial {
					flusher.Flush()
				}
				return nil
			}
			return send(events)
		})
		return eventCount, err
	}

	if _, err := snapshotAndSend(true); err != nil {
		if errors.Is(err, repository.ErrHouseMembershipUnavailable) {
			writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		}
		return
	}
	ticker := time.NewTicker(h.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			eventCount, err := snapshotAndSend(false)
			if err != nil {
				return
			}
			if eventCount == 0 {
				if err := setSSEWriteDeadline(w); err != nil {
					return
				}
				if _, err := w.Write([]byte(": keep-alive\n\n")); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}

// setSSEWriteDeadline bounds the time a house revocation barrier can be held by a
// network write. Standard net/http writers support it; test-only wrappers may not.
func setSSEWriteDeadline(w http.ResponseWriter) error {
	err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(5 * time.Second))
	if errors.Is(err, http.ErrNotSupported) {
		return nil
	}
	return err
}

func acceptsEventStream(r *http.Request) bool {
	return strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/event-stream")
}

func parseEventCursor(r *http.Request) (int64, error) {
	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	if cursor == "" {
		cursor = strings.TrimSpace(r.Header.Get("Last-Event-ID"))
	}
	if cursor == "" {
		return 0, nil
	}
	return parseCursorValue(cursor)
}

func parseCursorValue(cursor string) (int64, error) {
	value, err := strconv.ParseInt(cursor, 10, 64)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("cursor must be a non-negative integer")
	}
	return value, nil
}
