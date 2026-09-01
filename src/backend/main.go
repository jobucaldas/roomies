package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/roomies/backend/internal/config"
	"github.com/roomies/backend/internal/database"
	"github.com/roomies/backend/internal/handlers"
	"github.com/roomies/backend/internal/middleware"
	"github.com/roomies/backend/internal/repository"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	houseRepo := repository.NewHouseRepository(db)
	expenseRepo := repository.NewExpenseRepository(db)
	noteRepo := repository.NewNoteRepository(db)

	authHandler := handlers.NewAuthHandler(userRepo, cfg.JWTSecret)
	houseHandler := handlers.NewHouseHandler(houseRepo, userRepo)
	expenseHandler := handlers.NewExpenseHandler(expenseRepo, houseRepo)
	noteHandler := handlers.NewNoteHandler(noteRepo, houseRepo)
	balanceHandler := handlers.NewBalanceHandler(expenseRepo, houseRepo)

	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: cfg.CORSAllowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
		MaxAge:         300,
	}))
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// Health is intentionally unauthenticated so container orchestrators can probe it.
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTAuth(cfg.JWTSecret))

		r.Get("/api/auth/me", authHandler.Me)

		r.Route("/api/houses", func(r chi.Router) {
			r.Get("/", houseHandler.List)
			r.Post("/", houseHandler.Create)

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", houseHandler.Get)
				r.Put("/", houseHandler.Update)

				r.Route("/members", func(r chi.Router) {
					r.Get("/", houseHandler.ListMembers)
					r.Post("/", houseHandler.AddMember)
					r.Put("/{userId}", houseHandler.UpdateMemberRole)
					r.Delete("/{userId}", houseHandler.RemoveMember)
				})

				r.Route("/expenses", func(r chi.Router) {
					r.Get("/", expenseHandler.List)
					r.Post("/", expenseHandler.Create)
					r.Route("/{eid}", func(r chi.Router) {
						r.Get("/", expenseHandler.Get)
						r.Put("/", expenseHandler.Update)
						r.Delete("/", expenseHandler.Delete)
						r.Post("/visibility", expenseHandler.SetVisibility)
					})
				})

				r.Get("/balances", balanceHandler.GetBalances)

				r.Route("/notes", func(r chi.Router) {
					r.Get("/", noteHandler.List)
					r.Post("/", noteHandler.Create)
					r.Route("/{nid}", func(r chi.Router) {
						r.Get("/", noteHandler.Get)
						r.Put("/", noteHandler.Update)
						r.Delete("/", noteHandler.Delete)
					})
				})
			})
		})
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("Roomies backend starting on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
