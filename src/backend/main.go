package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	roomiesclock "github.com/roomies/backend/internal/clock"
	"github.com/roomies/backend/internal/config"
	"github.com/roomies/backend/internal/database"
	"github.com/roomies/backend/internal/providers"
	"github.com/roomies/backend/internal/repository"
	"github.com/roomies/backend/internal/server"
	"github.com/roomies/backend/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err := run(logger, os.Args[1:]); err != nil {
		logger.Error("roomies_backend_exit", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(logger *slog.Logger, args []string) error {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	command := "serve"
	if len(args) > 0 {
		command = args[0]
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer db.Close()
	if err := database.RunMigrations(db); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	clk := roomiesclock.RealClock{}
	userRepo := repository.NewUserRepository(db)
	houseRepo := repository.NewHouseRepository(db)
	expenseRepo := repository.NewExpenseRepository(db)
	noteRepo := repository.NewNoteRepository(db)
	reliabilityRepo := repository.NewReliabilityRepository(db, clk)

	switch command {
	case "serve":
		return runServer(ctx, logger, cfg, clk, userRepo, houseRepo, expenseRepo, noteRepo, reliabilityRepo)
	case "worker":
		return runWorker(ctx, logger, cfg, reliabilityRepo)
	default:
		return fmt.Errorf("unknown subcommand %q", command)
	}
}

func runServer(ctx context.Context, logger *slog.Logger, cfg *config.Config, clk roomiesclock.Clock,
	userRepo *repository.UserRepository, houseRepo *repository.HouseRepository,
	expenseRepo *repository.ExpenseRepository, noteRepo *repository.NoteRepository,
	reliabilityRepo *repository.ReliabilityRepository,
) error {
	handler := server.NewHandler(server.Dependencies{
		Config:          cfg,
		Logger:          logger,
		Clock:           clk,
		UserRepo:        userRepo,
		HouseRepo:       houseRepo,
		ExpenseRepo:     expenseRepo,
		NoteRepo:        noteRepo,
		ReliabilityRepo: reliabilityRepo,
	})
	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       60 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		logger.Info("serve_starting", slog.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()
	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil {
			return err
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logger.Info("serve_stopping")
	return httpServer.Shutdown(shutdownCtx)
}

func runWorker(ctx context.Context, logger *slog.Logger, cfg *config.Config, reliabilityRepo *repository.ReliabilityRepository) error {
	provider, err := providers.NewInvitationProvider(cfg)
	if err != nil {
		return fmt.Errorf("configure invitation provider: %w", err)
	}
	if _, fake := provider.(*providers.FakeInvitationProvider); fake {
		logger.Warn("invitation_provider_fake_only", slog.String("reason", "development/test environment or SMTP credentials absent"))
	} else {
		logger.Info("invitation_provider_smtp")
	}
	owner, err := os.Hostname()
	if err != nil {
		owner = "roomies-worker"
	}
	owner = fmt.Sprintf("%s-%d", owner, os.Getpid())
	processor := worker.New(
		logger,
		reliabilityRepo,
		provider,
		owner,
		time.Duration(cfg.JobPollInterval)*time.Second,
		time.Duration(cfg.JobLeaseSeconds)*time.Second,
	)
	healthServer := &http.Server{
		Addr:              ":" + cfg.WorkerHealthPort,
		Handler:           workerHealthHandler(processor),
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 2)
	go func() {
		logger.Info("worker_health_starting", slog.String("addr", healthServer.Addr))
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	go func() {
		logger.Info("worker_starting", slog.String("owner", owner))
		errCh <- processor.Run(ctx)
	}()
	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil {
			return err
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logger.Info("worker_stopping", slog.String("owner", owner))
	return healthServer.Shutdown(shutdownCtx)
}

func workerHealthHandler(processor interface{ Ready() bool }) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !processor.Ready() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"starting"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})
	return mux
}
