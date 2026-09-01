package handlers_test

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	roomiesclock "github.com/roomies/backend/internal/clock"
	"github.com/roomies/backend/internal/config"
	"github.com/roomies/backend/internal/database"
	"github.com/roomies/backend/internal/repository"
	"github.com/roomies/backend/internal/server"
)

type testEnv struct {
	Router          http.Handler
	DB              *sqlx.DB
	Clock           *roomiesclock.FakeClock
	Config          *config.Config
	UserRepo        *repository.UserRepository
	HouseRepo       *repository.HouseRepository
	ExpenseRepo     *repository.ExpenseRepository
	NoteRepo        *repository.NoteRepository
	ReliabilityRepo *repository.ReliabilityRepository
	JWTSecret       string
	Cleanup         func()
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "sqlite://:memory:"
	}
	db, err := database.Connect(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	clk := roomiesclock.NewFake(time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC))
	cfg := &config.Config{
		Port:               "8080",
		WorkerHealthPort:   "8081",
		DatabaseURL:        databaseURL,
		JWTSecret:          "test-secret-used-for-jwt-signing",
		Environment:        "test",
		CORSAllowedOrigins: []string{"http://localhost:8081"},
		PublicBaseURL:      "https://roomies.test",
		InvitationTTL:      168,
		JobPollInterval:    1,
		JobLeaseSeconds:    30,
	}
	userRepo := repository.NewUserRepository(db)
	houseRepo := repository.NewHouseRepository(db)
	expenseRepo := repository.NewExpenseRepository(db)
	noteRepo := repository.NewNoteRepository(db)
	reliabilityRepo := repository.NewReliabilityRepository(db, clk)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	router := server.NewHandler(server.Dependencies{
		Config:          cfg,
		Logger:          logger,
		Clock:           clk,
		UserRepo:        userRepo,
		HouseRepo:       houseRepo,
		ExpenseRepo:     expenseRepo,
		NoteRepo:        noteRepo,
		ReliabilityRepo: reliabilityRepo,
	})
	return &testEnv{
		Router:          router,
		DB:              db,
		Clock:           clk,
		Config:          cfg,
		UserRepo:        userRepo,
		HouseRepo:       houseRepo,
		ExpenseRepo:     expenseRepo,
		NoteRepo:        noteRepo,
		ReliabilityRepo: reliabilityRepo,
		JWTSecret:       cfg.JWTSecret,
		Cleanup:         func() { _ = db.Close() },
	}
}
