package handlers

import (
	"errors"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/roomies/backend/internal/middleware"
	"github.com/roomies/backend/internal/models"
	"github.com/roomies/backend/internal/repository"
)

const maxMoneyCents int64 = 9_999_999_999

type ExpenseHandler struct {
	expenseRepo *repository.ExpenseRepository
	houseRepo   *repository.HouseRepository
}

func NewExpenseHandler(expenseRepo *repository.ExpenseRepository, houseRepo *repository.HouseRepository) *ExpenseHandler {
	return &ExpenseHandler{expenseRepo: expenseRepo, houseRepo: houseRepo}
}

func (h *ExpenseHandler) List(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())

	if _, err := loadHouseMember(r.Context(), h.houseRepo, houseID, userID); err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}

	expenses, err := h.expenseRepo.ListByHouse(r.Context(), houseID, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to list expenses"})
		return
	}
	if expenses == nil {
		expenses = []models.Expense{}
	}
	writeJSON(w, http.StatusOK, expenses)
}

func (h *ExpenseHandler) Create(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())

	member, err := loadHouseMember(r.Context(), h.houseRepo, houseID, userID)
	if err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}
	if !canCreateHouseContent(member) {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "monitors cannot create expenses"})
		return
	}

	var req models.CreateExpenseRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	amountCents, err := validateExpenseFields(req.Amount, req.Description, req.Date)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	req.Description = strings.TrimSpace(req.Description)
	if req.Date == "" {
		req.Date = time.Now().Format("2006-01-02")
	}
	if req.Visibility == "" {
		req.Visibility = "shared"
	}
	if req.Visibility != "shared" && req.Visibility != "private" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "visibility must be 'shared' or 'private'"})
		return
	}
	if req.Visibility == "shared" && len(req.VisibleTo) != 0 {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "visible_to is only valid for private expenses"})
		return
	}

	members, err := h.houseRepo.ListMembers(r.Context(), houseID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to get members"})
		return
	}
	if err := validateVisibleUsers(req.VisibleTo, members); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	expense := &models.Expense{
		ID:          models.NewID(),
		HouseID:     houseID,
		PayerID:     userID,
		Amount:      centsToMoney(amountCents),
		AmountCents: amountCents,
		Description: req.Description,
		Category:    strings.TrimSpace(req.Category),
		Date:        req.Date,
		Visibility:  req.Visibility,
		CreatedAt:   time.Now(),
	}

	splits, err := buildSplits(expense.ID, amountCents, req.Split, members)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	if err := h.expenseRepo.CreateAggregate(r.Context(), expense, req.VisibleTo, splits); err != nil {
		if errors.Is(err, repository.ErrInvalidExpenseParticipant) {
			writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "expense participants must be eligible house members"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to create expense"})
		return
	}

	expense.PayerName = member.UserName
	writeJSON(w, http.StatusCreated, expense)
}

func (h *ExpenseHandler) Get(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	expenseID := chi.URLParam(r, "eid")
	userID := middleware.GetUserID(r.Context())

	if _, err := loadHouseMember(r.Context(), h.houseRepo, houseID, userID); err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}

	expense, err := h.expenseRepo.GetByHouseAndID(r.Context(), houseID, expenseID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: "expense not found"})
		return
	}

	if expense.Visibility == "private" && expense.PayerID != userID {
		visibleUsers, err := h.expenseRepo.GetVisibleUsers(r.Context(), expenseID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "internal error"})
			return
		}
		canSee := false
		for _, visibility := range visibleUsers {
			if visibility.UserID == userID {
				canSee = true
				break
			}
		}
		if !canSee {
			writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: "expense not found"})
			return
		}
	}

	splits, err := h.expenseRepo.GetSplits(r.Context(), expenseID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to get expense splits"})
		return
	}
	if splits == nil {
		splits = []models.ExpenseSplit{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"expense": expense,
		"splits":  splits,
	})
}

func (h *ExpenseHandler) Update(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	expenseID := chi.URLParam(r, "eid")
	userID := middleware.GetUserID(r.Context())

	member, err := loadHouseMember(r.Context(), h.houseRepo, houseID, userID)
	if err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}
	expense, err := h.expenseRepo.GetByHouseAndID(r.Context(), houseID, expenseID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: "expense not found"})
		return
	}
	if !canMutateOwnedResource(member, expense.PayerID, userID) {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "only the payer or an admin can update this expense"})
		return
	}

	var req models.UpdateExpenseRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	oldAmountCents := expense.AmountCents
	newAmountCents := oldAmountCents
	if req.Amount != nil {
		newAmountCents, err = moneyToCents(*req.Amount)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
			return
		}
		if newAmountCents != oldAmountCents && req.Split == nil {
			writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "replacement split is required when amount changes"})
			return
		}
		expense.AmountCents = newAmountCents
		expense.Amount = centsToMoney(newAmountCents)
	}
	if req.Description != nil {
		description := strings.TrimSpace(*req.Description)
		if description == "" {
			writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "description is required"})
			return
		}
		expense.Description = description
	}
	if req.Category != nil {
		expense.Category = strings.TrimSpace(*req.Category)
	}
	if req.Date != nil {
		if err := validateDate(*req.Date); err != nil {
			writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
			return
		}
		expense.Date = *req.Date
	}

	var replacementSplits *[]models.ExpenseSplit
	if req.Split != nil {
		members, err := h.houseRepo.ListMembers(r.Context(), houseID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to get members"})
			return
		}
		splits, err := buildCustomSplits(expense.ID, newAmountCents, *req.Split, members)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
			return
		}
		replacementSplits = &splits
	}

	if err := h.expenseRepo.UpdateAggregate(r.Context(), expense, replacementSplits); err != nil {
		if errors.Is(err, repository.ErrInvalidExpenseParticipant) {
			writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "split users must be eligible house members"})
			return
		}
		if errors.Is(err, repository.ErrExpenseNotFound) {
			writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: "expense not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to update expense"})
		return
	}

	writeJSON(w, http.StatusOK, expense)
}

func (h *ExpenseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	expenseID := chi.URLParam(r, "eid")
	userID := middleware.GetUserID(r.Context())

	member, err := loadHouseMember(r.Context(), h.houseRepo, houseID, userID)
	if err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}
	expense, err := h.expenseRepo.GetByHouseAndID(r.Context(), houseID, expenseID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: "expense not found"})
		return
	}
	if !canMutateOwnedResource(member, expense.PayerID, userID) {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "only the payer or an admin can delete this expense"})
		return
	}

	if err := h.expenseRepo.Delete(r.Context(), houseID, expenseID); err != nil {
		if errors.Is(err, repository.ErrExpenseNotFound) {
			writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: "expense not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to delete expense"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "expense deleted"})
}

func (h *ExpenseHandler) SetVisibility(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	expenseID := chi.URLParam(r, "eid")
	userID := middleware.GetUserID(r.Context())

	member, err := loadHouseMember(r.Context(), h.houseRepo, houseID, userID)
	if err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}
	expense, err := h.expenseRepo.GetByHouseAndID(r.Context(), houseID, expenseID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: "expense not found"})
		return
	}
	if !canChangeOwnedResourceVisibility(member, expense.PayerID, userID) {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "only the payer can change visibility"})
		return
	}

	var req models.SetVisibilityRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}
	if req.Visibility != "shared" && req.Visibility != "private" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "visibility must be 'shared' or 'private'"})
		return
	}
	if req.Visibility == "shared" && len(req.VisibleTo) != 0 {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "visible_to is only valid for private expenses"})
		return
	}
	members, err := h.houseRepo.ListMembers(r.Context(), houseID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to get members"})
		return
	}
	if err := validateVisibleUsers(req.VisibleTo, members); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.expenseRepo.SetVisibility(r.Context(), houseID, expenseID, req.Visibility, req.VisibleTo); err != nil {
		if errors.Is(err, repository.ErrInvalidExpenseParticipant) {
			writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "visible users must be house members"})
			return
		}
		if errors.Is(err, repository.ErrExpenseNotFound) {
			writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: "expense not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to set visibility"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "visibility updated"})
}

func validateExpenseFields(amount float64, description, date string) (int64, error) {
	amountCents, err := moneyToCents(amount)
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(description) == "" {
		return 0, errors.New("description is required")
	}
	if date != "" {
		if err := validateDate(date); err != nil {
			return 0, err
		}
	}
	return amountCents, nil
}

func validateDate(value string) error {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Format("2006-01-02") != value {
		return errors.New("date must be a valid date in YYYY-MM-DD format")
	}
	return nil
}

func moneyToCents(amount float64) (int64, error) {
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount <= 0 {
		return 0, errors.New("amount must be positive")
	}
	scaled := amount * 100
	rounded := math.Round(scaled)
	if math.Abs(scaled-rounded) > 0.000001 {
		return 0, errors.New("amount must have at most two decimal places")
	}
	if rounded > float64(maxMoneyCents) {
		return 0, errors.New("amount exceeds the supported maximum")
	}
	return int64(rounded), nil
}

func centsToMoney(cents int64) float64 {
	return float64(cents) / 100
}

func validateVisibleUsers(userIDs []string, members []models.HouseMember) error {
	memberSet := make(map[string]struct{}, len(members))
	for _, member := range members {
		memberSet[member.UserID] = struct{}{}
	}
	seen := make(map[string]struct{}, len(userIDs))
	for _, userID := range userIDs {
		if _, ok := memberSet[userID]; !ok {
			return errors.New("visible users must be house members")
		}
		if _, duplicate := seen[userID]; duplicate {
			return errors.New("visible_to must not contain duplicate users")
		}
		seen[userID] = struct{}{}
	}
	return nil
}

func buildSplits(expenseID string, amountCents int64, requested []models.SplitEntry, members []models.HouseMember) ([]models.ExpenseSplit, error) {
	if len(requested) != 0 {
		return buildCustomSplits(expenseID, amountCents, requested, members)
	}
	activeMembers := make([]models.HouseMember, 0, len(members))
	for _, member := range members {
		if member.Role != "monitor" {
			activeMembers = append(activeMembers, member)
		}
	}
	if len(activeMembers) == 0 {
		return nil, errors.New("expense requires at least one eligible split member")
	}
	base := amountCents / int64(len(activeMembers))
	remainder := amountCents % int64(len(activeMembers))
	splits := make([]models.ExpenseSplit, 0, len(activeMembers))
	for i, member := range activeMembers {
		share := base
		if int64(i) < remainder {
			share++
		}
		splits = append(splits, models.ExpenseSplit{
			ID:               models.NewID(),
			ExpenseID:        expenseID,
			UserID:           member.UserID,
			ShareAmount:      centsToMoney(share),
			ShareAmountCents: share,
		})
	}
	return splits, nil
}

func buildCustomSplits(expenseID string, amountCents int64, requested []models.SplitEntry, members []models.HouseMember) ([]models.ExpenseSplit, error) {
	if len(requested) == 0 {
		return nil, errors.New("split must contain at least one user")
	}
	eligible := make(map[string]struct{}, len(members))
	for _, member := range members {
		if member.Role != "monitor" {
			eligible[member.UserID] = struct{}{}
		}
	}
	seen := make(map[string]struct{}, len(requested))
	splits := make([]models.ExpenseSplit, 0, len(requested))
	var total int64
	for _, entry := range requested {
		if _, ok := eligible[entry.UserID]; !ok {
			return nil, errors.New("split users must be eligible house members")
		}
		if _, duplicate := seen[entry.UserID]; duplicate {
			return nil, errors.New("split must not contain duplicate users")
		}
		shareCents, err := moneyToCents(entry.Amount)
		if err != nil {
			return nil, errors.New("split amounts must be positive with at most two decimal places")
		}
		total += shareCents
		seen[entry.UserID] = struct{}{}
		splits = append(splits, models.ExpenseSplit{
			ID:               models.NewID(),
			ExpenseID:        expenseID,
			UserID:           entry.UserID,
			ShareAmount:      centsToMoney(shareCents),
			ShareAmountCents: shareCents,
		})
	}
	if total != amountCents {
		return nil, errors.New("split amounts must equal the expense amount")
	}
	return splits, nil
}
