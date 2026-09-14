package handlers

import (
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/roomies/backend/internal/middleware"
	"github.com/roomies/backend/internal/models"
	"github.com/roomies/backend/internal/repository"
)

type BalanceHandler struct {
	expenseRepo *repository.ExpenseRepository
	houseRepo   *repository.HouseRepository
}

func NewBalanceHandler(expenseRepo *repository.ExpenseRepository, houseRepo *repository.HouseRepository) *BalanceHandler {
	return &BalanceHandler{expenseRepo: expenseRepo, houseRepo: houseRepo}
}

func (h *BalanceHandler) GetBalances(w http.ResponseWriter, r *http.Request) {
	houseID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())
	if _, err := loadHouseMember(r.Context(), h.houseRepo, houseID, userID); err != nil {
		writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "not a member"})
		return
	}

	members, err := h.houseRepo.ListMembers(r.Context(), houseID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to get members"})
		return
	}
	activeMembers := make([]models.HouseMember, 0, len(members))
	for _, member := range members {
		if member.Role != "monitor" {
			activeMembers = append(activeMembers, member)
		}
	}
	if len(activeMembers) == 0 {
		writeJSON(w, http.StatusConflict, models.ErrorResponse{Error: "house has no eligible balance members"})
		return
	}

	expenses, err := h.expenseRepo.GetExpensesForBalance(r.Context(), houseID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to get expenses"})
		return
	}
	paid := make(map[string]int64, len(activeMembers))
	owed := make(map[string]int64, len(activeMembers))
	participantNames := make(map[string]string, len(activeMembers))
	for _, member := range activeMembers {
		participantNames[member.UserID] = member.UserName
	}
	for _, expense := range expenses {
		amountCents := expense.AmountCents
		participantNames[expense.PayerID] = expense.PayerName
		paid[expense.PayerID] += amountCents

		splits, err := h.expenseRepo.GetSplits(r.Context(), expense.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to get expense splits"})
			return
		}
		if len(splits) == 0 {
			writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "expense has no splits"})
			return
		}
		var splitTotal int64
		for _, split := range splits {
			shareCents := split.ShareAmountCents
			participantNames[split.UserID] = split.UserName
			owed[split.UserID] += shareCents
			splitTotal += shareCents
		}
		if splitTotal != amountCents {
			writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "expense splits do not equal expense amount"})
			return
		}
	}

	participantIDs := make([]string, 0, len(participantNames))
	for participantID := range participantNames {
		participantIDs = append(participantIDs, participantID)
	}
	sort.Slice(participantIDs, func(i, j int) bool {
		leftName := participantNames[participantIDs[i]]
		rightName := participantNames[participantIDs[j]]
		if leftName == rightName {
			return participantIDs[i] < participantIDs[j]
		}
		return leftName < rightName
	})

	balances := make([]models.BalanceEntry, 0, len(participantIDs))
	balanceCents := make([]balanceInCents, 0, len(participantIDs))
	for _, participantID := range participantIDs {
		netCents := paid[participantID] - owed[participantID]
		balances = append(balances, models.BalanceEntry{
			UserID: participantID, UserName: participantNames[participantID],
			Paid: centsToMoney(paid[participantID]), Owed: centsToMoney(owed[participantID]), Net: centsToMoney(netCents),
		})
		balanceCents = append(balanceCents, balanceInCents{
			userID: participantID, userName: participantNames[participantID], net: netCents,
		})
	}
	settlements := calculateSettlements(balanceCents)
	writeJSON(w, http.StatusOK, models.BalanceResponse{Balances: balances, Settlements: settlements})
}

type balanceInCents struct {
	userID   string
	userName string
	net      int64
}

func calculateSettlements(balances []balanceInCents) []models.BalanceSettlement {
	creditors := make([]balanceInCents, 0)
	debtors := make([]balanceInCents, 0)
	for _, balance := range balances {
		if balance.net > 0 {
			creditors = append(creditors, balance)
		} else if balance.net < 0 {
			debtors = append(debtors, balance)
		}
	}
	sort.Slice(creditors, func(i, j int) bool { return creditors[i].net > creditors[j].net })
	sort.Slice(debtors, func(i, j int) bool { return debtors[i].net < debtors[j].net })

	settlements := make([]models.BalanceSettlement, 0)
	for i, j := 0, 0; i < len(creditors) && j < len(debtors); {
		amount := creditors[i].net
		if debt := -debtors[j].net; debt < amount {
			amount = debt
		}
		settlements = append(settlements, models.BalanceSettlement{
			FromUserID: debtors[j].userID, FromUserName: debtors[j].userName,
			ToUserID: creditors[i].userID, ToUserName: creditors[i].userName,
			Amount: centsToMoney(amount),
		})
		creditors[i].net -= amount
		debtors[j].net += amount
		if creditors[i].net == 0 {
			i++
		}
		if debtors[j].net == 0 {
			j++
		}
	}
	return settlements
}
