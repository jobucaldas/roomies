package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/roomies/backend/internal/middleware"
	"github.com/roomies/backend/internal/models"
	"github.com/roomies/backend/internal/repository"
)

type HouseHandler struct {
	houseRepo *repository.HouseRepository
	userRepo  *repository.UserRepository
}

func NewHouseHandler(houseRepo *repository.HouseRepository, userRepo *repository.UserRepository) *HouseHandler {
	return &HouseHandler{houseRepo: houseRepo, userRepo: userRepo}
}

func (h *HouseHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	houses, err := h.houseRepo.ListByUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to list houses"})
		return
	}
	if houses == nil {
		houses = []models.House{}
	}
	writeJSON(w, http.StatusOK, houses)
}

func (h *HouseHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req models.CreateHouseRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "house name is required"})
		return
	}

	house := &models.House{
		ID:        models.NewID(),
		Name:      req.Name,
		CreatedAt: time.Now(),
	}

	member := &models.HouseMember{
		ID:       models.NewID(),
		HouseID:  house.ID,
		UserID:   userID,
		Role:     "admin",
		JoinedAt: time.Now(),
	}

	if err := h.houseRepo.CreateWithAdmin(r.Context(), house, member); err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to create house"})
		return
	}

	writeJSON(w, http.StatusCreated, house)
}

func (h *HouseHandler) Get(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	house, err := h.houseRepo.GetByID(r.Context(), houseID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: "house not found"})
		return
	}

	userID := middleware.GetUserID(r.Context())
	if _, err := h.houseRepo.GetMember(r.Context(), houseID, userID); err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member of this house"})
		return
	}

	writeJSON(w, http.StatusOK, house)
}

func (h *HouseHandler) Update(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())

	member, err := h.houseRepo.GetMember(r.Context(), houseID, userID)
	if err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}
	if member.Role != "admin" {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "only admins can update the house"})
		return
	}

	var req models.CreateHouseRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "house name is required"})
		return
	}

	house, err := h.houseRepo.GetByID(r.Context(), houseID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: "house not found"})
		return
	}
	house.Name = req.Name

	if err := h.houseRepo.Update(r.Context(), house); err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to update house"})
		return
	}

	writeJSON(w, http.StatusOK, house)
}

func (h *HouseHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())

	if _, err := h.houseRepo.GetMember(r.Context(), houseID, userID); err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}

	members, err := h.houseRepo.ListMembers(r.Context(), houseID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to list members"})
		return
	}
	if members == nil {
		members = []models.HouseMember{}
	}
	writeJSON(w, http.StatusOK, members)
}

func (h *HouseHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())

	member, err := h.houseRepo.GetMember(r.Context(), houseID, userID)
	if err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}
	if member.Role != "admin" {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "only admins can add members"})
		return
	}

	var req models.AddMemberRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Role == "" {
		req.Role = "member"
	}
	if req.Role != "admin" && req.Role != "member" && req.Role != "monitor" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid role: must be admin, member, or monitor"})
		return
	}

	targetUser, err := h.userRepo.GetByID(r.Context(), req.UserID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: "user not found"})
		return
	}

	newMember := &models.HouseMember{
		ID:       models.NewID(),
		HouseID:  houseID,
		UserID:   targetUser.ID,
		Role:     req.Role,
		JoinedAt: time.Now(),
	}

	if err := h.houseRepo.AddMember(r.Context(), newMember); err != nil {
		writeJSON(w, http.StatusConflict, models.ErrorResponse{Error: "user is already a member"})
		return
	}

	writeJSON(w, http.StatusCreated, newMember)
}

func (h *HouseHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	targetUserID := chi.URLParam(r, "userId")
	userID := middleware.GetUserID(r.Context())

	member, err := h.houseRepo.GetMember(r.Context(), houseID, userID)
	if err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}
	if member.Role != "admin" {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "only admins can update roles"})
		return
	}

	var req models.UpdateMemberRoleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Role != "admin" && req.Role != "member" && req.Role != "monitor" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid role"})
		return
	}

	if err := h.houseRepo.UpdateMemberRole(r.Context(), houseID, targetUserID, req.Role); err != nil {
		if errors.Is(err, repository.ErrLastAdmin) {
			writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: "member not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "role updated"})
}

func (h *HouseHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	targetUserID := chi.URLParam(r, "userId")
	userID := middleware.GetUserID(r.Context())

	member, err := h.houseRepo.GetMember(r.Context(), houseID, userID)
	if err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}
	if member.Role != "admin" {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "only admins can remove members"})
		return
	}

	if err := h.houseRepo.RemoveMember(r.Context(), houseID, targetUserID); err != nil {
		if errors.Is(err, repository.ErrLastAdmin) {
			writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: "member not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "member removed"})
}
