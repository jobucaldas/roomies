package handlers

import (
	"context"

	"github.com/roomies/backend/internal/models"
)

type houseMemberGetter interface {
	GetMember(ctx context.Context, houseID, userID string) (*models.HouseMember, error)
}

func loadHouseMember(ctx context.Context, repo houseMemberGetter, houseID, userID string) (*models.HouseMember, error) {
	return repo.GetMember(ctx, houseID, userID)
}

func isHouseAdmin(member *models.HouseMember) bool {
	return member != nil && member.Role == "admin"
}

func isHouseMonitor(member *models.HouseMember) bool {
	return member != nil && member.Role == "monitor"
}

func canCreateHouseContent(member *models.HouseMember) bool {
	return member != nil && member.Role != "monitor"
}

func canMutateOwnedResource(member *models.HouseMember, ownerID, userID string) bool {
	return member != nil && member.Role != "monitor" && (member.Role == "admin" || ownerID == userID)
}

func canChangeOwnedResourceVisibility(member *models.HouseMember, ownerID, userID string) bool {
	return member != nil && member.Role != "monitor" && ownerID == userID
}
