package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/roomies/backend/internal/models"
)

var ErrLastAdmin = errors.New("cannot remove or demote the last admin")

type HouseRepository struct {
	db *sqlx.DB
}

func NewHouseRepository(db *sqlx.DB) *HouseRepository {
	return &HouseRepository{db: db}
}

func (r *HouseRepository) Create(ctx context.Context, house *models.House) error {
	query := `INSERT INTO houses (id, name, created_at) VALUES (:id, :name, :created_at)`
	_, err := r.db.NamedExecContext(ctx, query, house)
	return err
}

func (r *HouseRepository) CreateWithAdmin(ctx context.Context, house *models.House, member *models.HouseMember) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.NamedExecContext(ctx,
		`INSERT INTO houses (id, name, created_at) VALUES (:id, :name, :created_at)`, house); err != nil {
		return err
	}
	if _, err := tx.NamedExecContext(ctx,
		`INSERT INTO house_members (id, house_id, user_id, role, joined_at)
		 VALUES (:id, :house_id, :user_id, :role, :joined_at)`, member); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *HouseRepository) GetByID(ctx context.Context, id string) (*models.House, error) {
	var house models.House
	err := r.db.GetContext(ctx, &house, "SELECT * FROM houses WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	return &house, nil
}

func (r *HouseRepository) Update(ctx context.Context, house *models.House) error {
	query := `UPDATE houses SET name = :name WHERE id = :id`
	_, err := r.db.NamedExecContext(ctx, query, house)
	return err
}

func (r *HouseRepository) ListByUser(ctx context.Context, userID string) ([]models.House, error) {
	var houses []models.House
	query := `SELECT h.* FROM houses h
		JOIN house_members hm ON hm.house_id = h.id
		WHERE hm.user_id = $1
		ORDER BY h.created_at DESC`
	err := r.db.SelectContext(ctx, &houses, query, userID)
	return houses, err
}

func (r *HouseRepository) AddMember(ctx context.Context, member *models.HouseMember) error {
	query := `INSERT INTO house_members (id, house_id, user_id, role, joined_at) VALUES (:id, :house_id, :user_id, :role, :joined_at)`
	_, err := r.db.NamedExecContext(ctx, query, member)
	return err
}

func (r *HouseRepository) GetMember(ctx context.Context, houseID, userID string) (*models.HouseMember, error) {
	var member models.HouseMember
	query := `SELECT hm.*, u.name as user_name, u.email as user_email
		FROM house_members hm
		JOIN users u ON u.id = hm.user_id
		WHERE hm.house_id = $1 AND hm.user_id = $2`
	err := r.db.GetContext(ctx, &member, query, houseID, userID)
	if err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *HouseRepository) ListMembers(ctx context.Context, houseID string) ([]models.HouseMember, error) {
	var members []models.HouseMember
	query := `SELECT hm.*, u.name as user_name, u.email as user_email
		FROM house_members hm
		JOIN users u ON u.id = hm.user_id
		WHERE hm.house_id = $1
		ORDER BY hm.joined_at ASC`
	err := r.db.SelectContext(ctx, &members, query, houseID)
	return members, err
}

func (r *HouseRepository) UpdateMemberRole(ctx context.Context, houseID, userID, role string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := r.lockHouse(ctx, tx, houseID); err != nil {
		return err
	}

	var currentRole string
	if err := tx.GetContext(ctx, &currentRole,
		"SELECT role FROM house_members WHERE house_id = $1 AND user_id = $2", houseID, userID); err != nil {
		return fmt.Errorf("member not found: %w", err)
	}
	if currentRole == "admin" && role != "admin" {
		if err := ensureAnotherAdmin(ctx, tx, houseID); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE house_members SET role = $1 WHERE house_id = $2 AND user_id = $3",
		role, houseID, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *HouseRepository) RemoveMember(ctx context.Context, houseID, userID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := r.lockHouse(ctx, tx, houseID); err != nil {
		return err
	}

	var role string
	if err := tx.GetContext(ctx, &role,
		"SELECT role FROM house_members WHERE house_id = $1 AND user_id = $2", houseID, userID); err != nil {
		return fmt.Errorf("member not found: %w", err)
	}
	if role == "admin" {
		if err := ensureAnotherAdmin(ctx, tx, houseID); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM house_members WHERE house_id = $1 AND user_id = $2", houseID, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *HouseRepository) GetMemberCount(ctx context.Context, houseID string) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM house_members WHERE house_id = $1 AND role != 'monitor'", houseID)
	return count, err
}

func (r *HouseRepository) GetAdminCount(ctx context.Context, houseID string) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM house_members WHERE house_id = $1 AND role = 'admin'", houseID)
	return count, err
}

func (r *HouseRepository) lockHouse(ctx context.Context, tx *sqlx.Tx, houseID string) error {
	query := "SELECT id FROM houses WHERE id = $1"
	if r.db.DriverName() == "postgres" {
		query += " FOR UPDATE"
	}
	var id string
	return tx.GetContext(ctx, &id, query, houseID)
}

func ensureAnotherAdmin(ctx context.Context, tx *sqlx.Tx, houseID string) error {
	var count int
	if err := tx.GetContext(ctx, &count,
		"SELECT COUNT(*) FROM house_members WHERE house_id = $1 AND role = 'admin'", houseID); err != nil {
		return err
	}
	if count <= 1 {
		return ErrLastAdmin
	}
	return nil
}
