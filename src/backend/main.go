package main

import (
	"context"
	"crypto/tls"
	"flag"
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
	"github.com/roomies/backend/internal/internaltls"
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
	if len(args) > 0 && args[0] == "tls-init" {
		return runTLSInit(logger, args[1:])
	}
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

	var deliveryCipher *providers.InvitationDeliveryCipher
	if cfg.SMTPHost != "" {
		deliveryCipher, err = providers.NewInvitationDeliveryCipher(cfg)
		if err != nil {
			return fmt.Errorf("configure invitation delivery encryption: %w", err)
		}
	}

	clk := roomiesclock.RealClock{}
	var notificationCipher *providers.NotificationDeliveryCipher
	if cfg.NotificationDeliveryKey != "" {
		notificationCipher, err = providers.NewNotificationDeliveryCipher(cfg.NotificationDeliveryKeyID, cfg.NotificationDeliveryKey, cfg.NotificationDeliveryOldKeys)
		if err != nil {
			return fmt.Errorf("configure notification delivery encryption: %w", err)
		}
	}
	userRepo := repository.NewUserRepository(db)
	houseRepo := repository.NewHouseRepository(db)
	notificationRepo := repository.NewNotificationRepository(db, clk, notificationCipher)
	expenseRepo := repository.NewExpenseRepository(db, notificationRepo)
	noteRepo := repository.NewNoteRepository(db)
	reliabilityRepo := repository.NewReliabilityRepository(db, clk, deliveryCipher)
	householdRepo := repository.NewHouseholdRepository(db, clk)

	switch command {
	case "serve":
		return runServer(ctx, logger, cfg, clk, userRepo, houseRepo, expenseRepo, noteRepo, reliabilityRepo, notificationRepo, householdRepo)
	case "worker":
		return runWorker(ctx, logger, cfg, userRepo, reliabilityRepo, notificationRepo, householdRepo, deliveryCipher)
	default:
		return fmt.Errorf("unknown subcommand %q", command)
	}
}

func runServer(ctx context.Context, logger *slog.Logger, cfg *config.Config, clk roomiesclock.Clock,
	userRepo *repository.UserRepository, houseRepo *repository.HouseRepository,
	expenseRepo *repository.ExpenseRepository, noteRepo *repository.NoteRepository,
	reliabilityRepo *repository.ReliabilityRepository, notificationRepo *repository.NotificationRepository,
	householdRepo *repository.HouseholdRepository,
) error {
	handler := server.NewHandler(server.Dependencies{
		Config:           cfg,
		Logger:           logger,
		Clock:            clk,
		UserRepo:         userRepo,
		HouseRepo:        houseRepo,
		ExpenseRepo:      expenseRepo,
		NoteRepo:         noteRepo,
		ReliabilityRepo:  reliabilityRepo,
		NotificationRepo: notificationRepo,
		HouseholdRepo:    householdRepo,
	})
	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       60 * time.Second,
	}
	useTLS := cfg.TLSCertFile != ""
	if useTLS {
		httpServer.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	} else if cfg.Environment == "production" {
		logger.Warn("serve_plaintext", slog.String("reason", "TLS_CERT_FILE is unset; traffic to the API is unencrypted until it reaches a TLS proxy"))
	}
	errCh := make(chan error, 1)
	go func() {
		logger.Info("serve_starting", slog.String("addr", httpServer.Addr), slog.Bool("tls", useTLS))
		var err error
		if useTLS {
			err = httpServer.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
		} else {
			err = httpServer.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
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

func runWorker(ctx context.Context, logger *slog.Logger, cfg *config.Config, userRepo *repository.UserRepository, reliabilityRepo *repository.ReliabilityRepository, notificationRepo *repository.NotificationRepository, householdRepo *repository.HouseholdRepository, deliveryCipher *providers.InvitationDeliveryCipher) error {
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
	notificationProvider, err := providers.NewNotificationProvider(ctx, cfg)
	if err != nil {
		return fmt.Errorf("configure notification provider: %w", err)
	}
	if notificationProvider.Fake() {
		logger.Warn("notification_provider_fake_only", slog.String("reason", "push credentials absent; no external delivery"))
	}
	processor := worker.New(
		logger,
		reliabilityRepo,
		provider,
		owner,
		time.Duration(cfg.JobPollInterval)*time.Second,
		time.Duration(cfg.JobLeaseSeconds)*time.Second,
		deliveryCipher,
	)
	processor.ConfigureNotifications(notificationRepo, notificationProvider)
	processor.ConfigureHousehold(householdRepo)
	processor.ConfigureSessions(userRepo)
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

// runTLSInit issues the internal CA and service certificates (see
// internaltls) into a shared volume before PostgreSQL and the API start.
func runTLSInit(logger *slog.Logger, args []string) error {
	flags := flag.NewFlagSet("tls-init", flag.ContinueOnError)
	dir := flags.String("dir", "/tls", "directory for the CA, certificates, keys, and pg_hba.conf")
	dbHosts := flags.String("db-hosts", "db", "comma-separated names the PostgreSQL certificate is valid for")
	backendHosts := flags.String("backend-hosts", "backend,localhost,127.0.0.1", "comma-separated names the API certificate is valid for")
	dbGroup := flags.Int("db-group", 70, "group id of the PostgreSQL server (70 in postgres:*-alpine, 999 in Debian images)")
	backendUser := flags.Int("backend-user", 10001, "uid that runs the API")
	if err := flags.Parse(args); err != nil {
		return err
	}
	issued, err := internaltls.Ensure(*dir, internaltls.Options{
		DBHosts:      internaltls.SplitHosts(*dbHosts),
		BackendHosts: internaltls.SplitHosts(*backendHosts),
		DBGroup:      *dbGroup,
		BackendUser:  *backendUser,
	})
	if err != nil {
		return fmt.Errorf("tls-init: %w", err)
	}
	logger.Info("tls_init_complete", slog.String("dir", *dir), slog.Bool("issued", issued))
	return nil
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
