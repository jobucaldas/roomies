package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/roomies/backend/internal/models"
)

var (
	ErrExpenseNotFound           = errors.New("expense not found")
	ErrInvalidExpenseParticipant = errors.New("expense participant is not an eligible house member")
)

const expenseSelect = `SELECT e.id, e.house_id, e.payer_id, e.amount, e.amount_cents, e.description,
	e.category, CAST(e.date AS TEXT) AS date, e.visibility, e.created_at,
	u.name AS payer_name
	FROM expenses e
	JOIN users u ON u.id = e.payer_id`

type ExpenseRepository struct {
	db *sqlx.DB
}

func NewExpenseRepository(db *sqlx.DB) *ExpenseRepository {
	return &ExpenseRepository{db: db}
}

// CreateAggregate persists the expense and all of its dependent rows atomically.
func (r *ExpenseRepository) CreateAggregate(ctx context.Context, expense *models.Expense, visibleTo []string, splits []models.ExpenseSplit) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := r.lockHouse(ctx, tx, expense.HouseID); err != nil {
		return err
	}
	if err := validateParticipants(ctx, tx, expense.HouseID, visibleTo, splits); err != nil {
		return err
	}
	if err := validateMoneyAmounts(expense, splits); err != nil {
		return err
	}

	query := `INSERT INTO expenses (id, house_id, payer_id, amount, amount_cents, description, category, date, visibility, created_at)
		VALUES (:id, :house_id, :payer_id, :amount, :amount_cents, :description, :category, :date, :visibility, :created_at)`
	if _, err := tx.NamedExecContext(ctx, query, expense); err != nil {
		return err
	}
	if err := insertVisibility(ctx, tx, expense.ID, visibleTo); err != nil {
		return err
	}
	if err := insertSplits(ctx, tx, splits); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ExpenseRepository) GetByHouseAndID(ctx context.Context, houseID, id string) (*models.Expense, error) {
	var expense models.Expense
	query := expenseSelect + ` WHERE e.house_id = $1 AND e.id = $2`
	if err := r.db.GetContext(ctx, &expense, query, houseID, id); err != nil {
		return nil, err
	}
	return &expense, nil
}

func (r *ExpenseRepository) ListByHouse(ctx context.Context, houseID, userID string) ([]models.Expense, error) {
	var expenses []models.Expense
	query := expenseSelect + ` WHERE e.house_id = $1
		AND (e.visibility = 'shared'
			OR e.payer_id = $2
			OR EXISTS (
				SELECT 1 FROM expense_visibility ev
				WHERE ev.expense_id = e.id AND ev.user_id = $2
			))
		ORDER BY e.created_at DESC`
	err := r.db.SelectContext(ctx, &expenses, query, houseID, userID)
	return expenses, err
}

// UpdateAggregate scopes the mutation to the URL house and optionally replaces
// all splits in the same transaction as the expense update.
func (r *ExpenseRepository) UpdateAggregate(ctx context.Context, expense *models.Expense, splits *[]models.ExpenseSplit) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := r.lockHouse(ctx, tx, expense.HouseID); err != nil {
		return err
	}
	if splits != nil {
		if err := validateParticipants(ctx, tx, expense.HouseID, nil, *splits); err != nil {
			return err
		}
		if err := validateMoneyAmounts(expense, *splits); err != nil {
			return err
		}
	}

	query := `UPDATE expenses SET amount = :amount, amount_cents = :amount_cents, description = :description,
		category = :category, date = :date WHERE house_id = :house_id AND id = :id`
	result, err := tx.NamedExecContext(ctx, query, expense)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return ErrExpenseNotFound
	}
	if splits != nil {
		if _, err := tx.ExecContext(ctx, "DELETE FROM expense_splits WHERE expense_id = $1", expense.ID); err != nil {
			return err
		}
		if err := insertSplits(ctx, tx, *splits); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *ExpenseRepository) Delete(ctx context.Context, houseID, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM expenses WHERE house_id = $1 AND id = $2", houseID, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return ErrExpenseNotFound
	}
	return nil
}

func (r *ExpenseRepository) SetVisibility(ctx context.Context, houseID, expenseID, visibility string, visibleTo []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := r.lockHouse(ctx, tx, houseID); err != nil {
		return err
	}
	if err := validateParticipants(ctx, tx, houseID, visibleTo, nil); err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx, "UPDATE expenses SET visibility = $1 WHERE house_id = $2 AND id = $3", visibility, houseID, expenseID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return ErrExpenseNotFound
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM expense_visibility WHERE expense_id = $1", expenseID); err != nil {
		return err
	}
	if err := insertVisibility(ctx, tx, expenseID, visibleTo); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ExpenseRepository) AddVisibility(ctx context.Context, expenseID, userID string) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO expense_visibility (expense_id, user_id) VALUES ($1, $2) ON CONFLICT (expense_id, user_id) DO NOTHING",
		expenseID, userID)
	return err
}

func (r *ExpenseRepository) RemoveVisibility(ctx context.Context, expenseID, userID string) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM expense_visibility WHERE expense_id = $1 AND user_id = $2",
		expenseID, userID)
	return err
}

func (r *ExpenseRepository) GetVisibleUsers(ctx context.Context, expenseID string) ([]models.ExpenseVisibility, error) {
	var vis []models.ExpenseVisibility
	err := r.db.SelectContext(ctx, &vis,
		"SELECT expense_id, user_id FROM expense_visibility WHERE expense_id = $1", expenseID)
	return vis, err
}

func (r *ExpenseRepository) GetSplits(ctx context.Context, expenseID string) ([]models.ExpenseSplit, error) {
	var splits []models.ExpenseSplit
	query := `SELECT es.id, es.expense_id, es.user_id, es.share_amount, es.share_amount_cents, u.name AS user_name
		FROM expense_splits es
		JOIN users u ON u.id = es.user_id
		WHERE es.expense_id = $1
		ORDER BY es.id`
	err := r.db.SelectContext(ctx, &splits, query, expenseID)
	return splits, err
}

func (r *ExpenseRepository) GetExpensesForBalance(ctx context.Context, houseID string) ([]models.Expense, error) {
	var expenses []models.Expense
	query := expenseSelect + ` WHERE e.house_id = $1 AND e.visibility = 'shared' ORDER BY e.created_at ASC`
	err := r.db.SelectContext(ctx, &expenses, query, houseID)
	return expenses, err
}

func (r *ExpenseRepository) lockHouse(ctx context.Context, tx *sqlx.Tx, houseID string) error {
	query := "SELECT id FROM houses WHERE id = $1"
	if r.db.DriverName() == "postgres" {
		query += " FOR UPDATE"
	}
	var id string
	return tx.GetContext(ctx, &id, query, houseID)
}

func validateParticipants(ctx context.Context, tx *sqlx.Tx, houseID string, visibleTo []string, splits []models.ExpenseSplit) error {
	for _, userID := range visibleTo {
		var count int
		if err := tx.GetContext(ctx, &count,
			"SELECT COUNT(*) FROM house_members WHERE house_id = $1 AND user_id = $2",
			houseID, userID); err != nil {
			return err
		}
		if count != 1 {
			return fmt.Errorf("%w: visibility user %s", ErrInvalidExpenseParticipant, userID)
		}
	}
	for _, split := range splits {
		var count int
		if err := tx.GetContext(ctx, &count,
			"SELECT COUNT(*) FROM house_members WHERE house_id = $1 AND user_id = $2 AND role != 'monitor'",
			houseID, split.UserID); err != nil {
			return err
		}
		if count != 1 {
			return fmt.Errorf("%w: split user %s", ErrInvalidExpenseParticipant, split.UserID)
		}
	}
	return nil
}

func validateMoneyAmounts(expense *models.Expense, splits []models.ExpenseSplit) error {
	if expense.AmountCents <= 0 {
		return fmt.Errorf("expense amount must be positive")
	}
	if len(splits) == 0 {
		return fmt.Errorf("expense requires at least one split")
	}
	var splitTotal int64
	for _, split := range splits {
		if split.ShareAmountCents < 0 {
			return fmt.Errorf("expense split amount must not be negative")
		}
		splitTotal += split.ShareAmountCents
	}
	if len(splits) != 0 && splitTotal != expense.AmountCents {
		return fmt.Errorf("expense split amounts must equal the expense amount")
	}
	return nil
}

func insertVisibility(ctx context.Context, tx *sqlx.Tx, expenseID string, visibleTo []string) error {
	for _, userID := range visibleTo {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO expense_visibility (expense_id, user_id) VALUES ($1, $2)",
			expenseID, userID); err != nil {
			return err
		}
	}
	return nil
}

func insertSplits(ctx context.Context, tx *sqlx.Tx, splits []models.ExpenseSplit) error {
	for i := range splits {
		query := `INSERT INTO expense_splits (id, expense_id, user_id, share_amount, share_amount_cents)
			VALUES (:id, :expense_id, :user_id, :share_amount, :share_amount_cents)`
		if _, err := tx.NamedExecContext(ctx, query, &splits[i]); err != nil {
			return err
		}
	}
	return nil
}
