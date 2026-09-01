package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/roomies/backend/internal/models"
)

type NoteRepository struct {
	db *sqlx.DB
}

func NewNoteRepository(db *sqlx.DB) *NoteRepository {
	return &NoteRepository{db: db}
}

func (r *NoteRepository) Create(ctx context.Context, note *models.Note) error {
	query := `INSERT INTO notes (id, house_id, author_id, title, content, created_at, updated_at)
		VALUES (:id, :house_id, :author_id, :title, :content, :created_at, :updated_at)`
	_, err := r.db.NamedExecContext(ctx, query, note)
	return err
}

func (r *NoteRepository) GetByHouseAndID(ctx context.Context, houseID, id string) (*models.Note, error) {
	var note models.Note
	query := `SELECT n.*, u.name as author_name FROM notes n
		JOIN users u ON u.id = n.author_id
		WHERE n.house_id = $1 AND n.id = $2`
	err := r.db.GetContext(ctx, &note, query, houseID, id)
	if err != nil {
		return nil, err
	}
	return &note, nil
}

func (r *NoteRepository) ListByHouse(ctx context.Context, houseID string) ([]models.Note, error) {
	var notes []models.Note
	query := `SELECT n.*, u.name as author_name FROM notes n
		JOIN users u ON u.id = n.author_id
		WHERE n.house_id = $1
		ORDER BY n.created_at DESC`
	err := r.db.SelectContext(ctx, &notes, query, houseID)
	return notes, err
}

func (r *NoteRepository) Update(ctx context.Context, note *models.Note) error {
	query := `UPDATE notes SET title = :title, content = :content, updated_at = :updated_at
		WHERE house_id = :house_id AND id = :id`
	_, err := r.db.NamedExecContext(ctx, query, note)
	return err
}

func (r *NoteRepository) Delete(ctx context.Context, houseID, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM notes WHERE house_id = $1 AND id = $2", houseID, id)
	return err
}
