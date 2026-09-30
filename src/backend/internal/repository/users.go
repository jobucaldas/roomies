package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/roomies/backend/internal/models"
)

var ErrLocalPasswordAccount = errors.New("email belongs to a local password account")

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `INSERT INTO users (id, name, email, password_hash, workos_user_id, created_at)
		VALUES (:id, :name, :email, :password_hash, :workos_user_id, :created_at)`
	_, err := r.db.NamedExecContext(ctx, query, user)
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	err := r.db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.GetContext(ctx, &user, "SELECT * FROM users WHERE email = $1", email)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByWorkOSUserID(ctx context.Context, workosUserID string) (*models.User, error) {
	var user models.User
	err := r.db.GetContext(ctx, &user, "SELECT * FROM users WHERE workos_user_id = $1", workosUserID)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM users WHERE email = $1", email)
	return count > 0, err
}

func (r *UserRepository) LinkWorkOSUser(ctx context.Context, userID, workosUserID, name string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET workos_user_id = $1, name = $2 WHERE id = $3`,
		workosUserID, name, userID,
	)
	return err
}

func (r *UserRepository) UpsertFromWorkOS(ctx context.Context, workosUserID, email, name string) (*models.User, error) {
	if user, err := r.GetByWorkOSUserID(ctx, workosUserID); err == nil {
		if user.Name != name || user.Email != email {
			_, updateErr := r.db.ExecContext(ctx,
				`UPDATE users SET name = $1, email = $2 WHERE id = $3`,
				name, email, user.ID,
			)
			if updateErr != nil {
				return nil, updateErr
			}
			user.Name = name
			user.Email = email
		}
		return user, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if user, err := r.GetByEmail(ctx, email); err == nil {
		// Local passwords are not proof of email ownership. Linking would let
		// whoever registered the address keep password access to the AuthKit user.
		if strings.TrimSpace(user.PasswordHash) != "" {
			return nil, ErrLocalPasswordAccount
		}
		if err := r.LinkWorkOSUser(ctx, user.ID, workosUserID, name); err != nil {
			return nil, err
		}
		user.WorkOSUserID = workosUserID
		user.Name = name
		return user, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	user := &models.User{
		ID:           models.NewID(),
		Name:         name,
		Email:        email,
		PasswordHash: "",
		WorkOSUserID: workosUserID,
		CreatedAt:    time.Now(),
	}
	if err := r.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}
