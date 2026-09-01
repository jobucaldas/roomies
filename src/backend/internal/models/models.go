package models

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

func NewID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}

type User struct {
	ID           string    `db:"id" json:"id"`
	Name         string    `db:"name" json:"name"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type House struct {
	ID        string    `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type HouseMember struct {
	ID        string    `db:"id" json:"id"`
	HouseID   string    `db:"house_id" json:"house_id"`
	UserID    string    `db:"user_id" json:"user_id"`
	Role      string    `db:"role" json:"role"`
	JoinedAt  time.Time `db:"joined_at" json:"joined_at"`
	UserName  string    `db:"user_name" json:"user_name,omitempty"`
	UserEmail string    `db:"user_email" json:"user_email,omitempty"`
}

type Expense struct {
	ID          string    `db:"id" json:"id"`
	HouseID     string    `db:"house_id" json:"house_id"`
	PayerID     string    `db:"payer_id" json:"payer_id"`
	Amount      float64   `db:"amount" json:"amount"`
	Description string    `db:"description" json:"description"`
	Category    string    `db:"category" json:"category"`
	Date        string    `db:"date" json:"date"`
	Visibility  string    `db:"visibility" json:"visibility"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	PayerName   string    `db:"payer_name" json:"payer_name,omitempty"`
}

type ExpenseVisibility struct {
	ExpenseID string `db:"expense_id" json:"expense_id"`
	UserID    string `db:"user_id" json:"user_id"`
}

type ExpenseSplit struct {
	ID          string  `db:"id" json:"id"`
	ExpenseID   string  `db:"expense_id" json:"expense_id"`
	UserID      string  `db:"user_id" json:"user_id"`
	ShareAmount float64 `db:"share_amount" json:"share_amount"`
	UserName    string  `db:"user_name" json:"user_name,omitempty"`
}

type Note struct {
	ID         string    `db:"id" json:"id"`
	HouseID    string    `db:"house_id" json:"house_id"`
	AuthorID   string    `db:"author_id" json:"author_id"`
	Title      string    `db:"title" json:"title"`
	Content    string    `db:"content" json:"content"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
	AuthorName string    `db:"author_name" json:"author_name,omitempty"`
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type CreateHouseRequest struct {
	Name string `json:"name"`
}

type AddMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role"`
}

type CreateExpenseRequest struct {
	Amount      float64      `json:"amount"`
	Description string       `json:"description"`
	Category    string       `json:"category"`
	Date        string       `json:"date"`
	Visibility  string       `json:"visibility"`
	VisibleTo   []string     `json:"visible_to"`
	Split       []SplitEntry `json:"split"`
}

type SplitEntry struct {
	UserID string  `json:"user_id"`
	Amount float64 `json:"amount"`
}

type UpdateExpenseRequest struct {
	Amount      *float64      `json:"amount,omitempty"`
	Description *string       `json:"description,omitempty"`
	Category    *string       `json:"category,omitempty"`
	Date        *string       `json:"date,omitempty"`
	Split       *[]SplitEntry `json:"split,omitempty"`
}

type SetVisibilityRequest struct {
	Visibility string   `json:"visibility"`
	VisibleTo  []string `json:"visible_to"`
}

type CreateNoteRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type UpdateNoteRequest struct {
	Title   *string `json:"title,omitempty"`
	Content *string `json:"content,omitempty"`
}

type BalanceEntry struct {
	UserID   string  `json:"user_id"`
	UserName string  `json:"user_name"`
	Paid     float64 `json:"paid"`
	Owed     float64 `json:"owed"`
	Net      float64 `json:"net"`
}

type BalanceSettlement struct {
	FromUserID   string  `json:"from_user_id"`
	FromUserName string  `json:"from_user_name"`
	ToUserID     string  `json:"to_user_id"`
	ToUserName   string  `json:"to_user_name"`
	Amount       float64 `json:"amount"`
}

type BalanceResponse struct {
	Balances    []BalanceEntry      `json:"balances"`
	Settlements []BalanceSettlement `json:"settlements"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
