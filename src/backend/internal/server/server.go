package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	roomiesclock "github.com/roomies/backend/internal/clock"
	"github.com/roomies/backend/internal/config"
	"github.com/roomies/backend/internal/handlers"
	"github.com/roomies/backend/internal/middleware"
	"github.com/roomies/backend/internal/repository"
)

type Dependencies struct {
	Config           *config.Config
	Logger           *slog.Logger
	Clock            roomiesclock.Clock
	UserRepo         *repository.UserRepository
	HouseRepo        *repository.HouseRepository
	ExpenseRepo      *repository.ExpenseRepository
	NoteRepo         *repository.NoteRepository
	ReliabilityRepo  *repository.ReliabilityRepository
	NotificationRepo *repository.NotificationRepository
	HouseholdRepo    *repository.HouseholdRepository
	Ready            func() bool
}

func NewHandler(deps Dependencies) http.Handler {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	clk := deps.Clock
	if clk == nil {
		clk = roomiesclock.RealClock{}
	}
	isReady := deps.Ready
	if isReady == nil {
		isReady = func() bool { return true }
	}

	authHandler := handlers.NewAuthHandler(deps.UserRepo, deps.Config.JWTSecret)
	houseHandler := handlers.NewHouseHandler(deps.HouseRepo, deps.UserRepo, deps.ReliabilityRepo)
	expenseHandler := handlers.NewExpenseHandler(deps.ExpenseRepo, deps.HouseRepo)
	noteHandler := handlers.NewNoteHandler(deps.NoteRepo, deps.HouseRepo)
	balanceHandler := handlers.NewBalanceHandler(deps.ExpenseRepo, deps.HouseRepo)
	invitationHandler := handlers.NewInvitationHandler(deps.HouseRepo, deps.ReliabilityRepo, clk, deps.Config.PublicBaseURL, time.Duration(deps.Config.InvitationTTL)*time.Hour)
	eventsHandler := handlers.NewHouseEventsHandler(deps.HouseRepo, deps.ReliabilityRepo, 250*time.Millisecond)
	notificationHandler := handlers.NewNotificationHandler(deps.NotificationRepo, deps.HouseRepo)
	householdHandler := handlers.NewHouseholdHandler(deps.HouseholdRepo, deps.HouseRepo)

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: deps.Config.CORSAllowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type", "Idempotency-Key", "Last-Event-ID"},
		MaxAge:         300,
	}))
	r.Use(chimiddleware.RequestID)
	r.Use(middleware.StructuredRequestLogger(logger))
	r.Use(chimiddleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	r.Get("/api/notifications/vapid-public-key", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// The public VAPID key is intentionally discoverable; the private key never leaves configuration.
		_ = json.NewEncoder(w).Encode(map[string]string{"public_key": deps.Config.WebPushPublicKey})
	})

	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !isReady() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"starting"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTAuth(deps.Config.JWTSecret))
		r.Use(middleware.Idempotency(deps.ReliabilityRepo))

		r.Get("/api/auth/me", authHandler.Me)
		r.Post("/api/invitations/accept", invitationHandler.Accept)

		r.Route("/api/houses", func(r chi.Router) {
			r.Get("/", houseHandler.List)
			r.Post("/", houseHandler.Create)

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", houseHandler.Get)
				r.Put("/", houseHandler.Update)
				r.Get("/events", eventsHandler.List)
				r.Get("/notification-preferences", notificationHandler.GetPreferences)
				r.Put("/notification-preferences", notificationHandler.PutPreferences)
				r.Route("/notification-subscriptions", func(r chi.Router) {
					r.Get("/", notificationHandler.ListSubscriptions)
					r.Post("/", notificationHandler.CreateSubscription)
					r.Delete("/{subscriptionId}", notificationHandler.DeleteSubscription)
				})
				r.Route("/scheduled-events", func(r chi.Router) {
					r.Get("/", notificationHandler.ListEvents)
					r.Post("/", notificationHandler.CreateEvent)
					r.Put("/{eventId}", notificationHandler.UpdateEvent)
					r.Delete("/{eventId}", notificationHandler.DeleteEvent)
				})
				r.Route("/groceries", func(r chi.Router) {
					r.Get("/", householdHandler.ListGroceries)
					r.Post("/", householdHandler.CreateGrocery)
					r.Get("/{groceryId}", householdHandler.GetGrocery)
					r.Put("/{groceryId}", householdHandler.UpdateGrocery)
					r.Post("/{groceryId}/toggle", householdHandler.ToggleGrocery)
					r.Delete("/{groceryId}", householdHandler.DeleteGrocery)
				})
				r.Route("/chores", func(r chi.Router) {
					r.Get("/", householdHandler.ListChores)
					r.Post("/", householdHandler.CreateChore)
					r.Get("/{choreId}", householdHandler.GetChore)
					r.Put("/{choreId}", householdHandler.UpdateChore)
					r.Delete("/{choreId}", householdHandler.DeleteChore)
					r.Get("/{choreId}/completions", householdHandler.ListChoreCompletions)
					r.Post("/{choreId}/completions", householdHandler.CompleteChore)
				})
				r.Route("/calendar", func(r chi.Router) {
					r.Get("/", householdHandler.ListCalendar)
					r.Post("/", householdHandler.CreateCalendar)
					r.Get("/{calendarId}", householdHandler.GetCalendar)
					r.Put("/{calendarId}", householdHandler.UpdateCalendar)
					r.Delete("/{calendarId}", householdHandler.DeleteCalendar)
				})
				r.Route("/chat", func(r chi.Router) {
					r.Get("/", householdHandler.ListChat)
					r.Post("/", householdHandler.CreateChat)
					r.Put("/{messageId}", householdHandler.EditChat)
					r.Delete("/{messageId}", householdHandler.DeleteChat)
				})

				r.Route("/members", func(r chi.Router) {
					r.Get("/", houseHandler.ListMembers)
					r.Post("/", houseHandler.AddMember)
					r.Put("/{userId}", houseHandler.UpdateMemberRole)
					r.Delete("/{userId}", houseHandler.RemoveMember)
				})

				r.Route("/invites", func(r chi.Router) {
					r.Get("/", invitationHandler.List)
					r.Post("/", invitationHandler.Create)
					r.Delete("/{inviteId}", invitationHandler.Revoke)
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

	return r
}
