package handlers

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/roomies/backend/internal/middleware"
	"github.com/roomies/backend/internal/models"
	"github.com/roomies/backend/internal/repository"
	"github.com/roomies/backend/internal/workosauth"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	userRepo  *repository.UserRepository
	jwtSecret string
	workos    *workosauth.Client
}

func NewAuthHandler(userRepo *repository.UserRepository, jwtSecret string, workos *workosauth.Client) *AuthHandler {
	return &AuthHandler{userRepo: userRepo, jwtSecret: jwtSecret, workos: workos}
}

func (h *AuthHandler) Config(w http.ResponseWriter, r *http.Request) {
	enabled := h.workos != nil && h.workos.Enabled()
	redirectURI := ""
	if h.workos != nil {
		redirectURI = h.workos.RedirectURI
	}
	// Password endpoints stay available only when AuthKit is unset (CI clears WORKOS_*).
	// Enabling AuthKit rejects them so that path cannot bypass hosted sign-in.
	writeJSON(w, http.StatusOK, map[string]any{
		"authkit":      enabled,
		"password":     !enabled,
		"redirect_uri": redirectURI,
	})
}

func (h *AuthHandler) passwordEnabled() bool {
	return h.workos == nil || !h.workos.Enabled()
}

func (h *AuthHandler) rejectPasswordAuth(w http.ResponseWriter) bool {
	if h.passwordEnabled() {
		return false
	}
	writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "password authentication is disabled"})
	return true
}

func (h *AuthHandler) WorkOSAuthorize(w http.ResponseWriter, r *http.Request) {
	if h.workos == nil || !h.workos.Enabled() {
		writeJSON(w, http.StatusServiceUnavailable, models.ErrorResponse{Error: "WorkOS AuthKit is not configured"})
		return
	}
	screenHint := strings.TrimSpace(r.URL.Query().Get("screen_hint"))
	switch screenHint {
	case "", "sign-in", "sign-up":
	default:
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "screen_hint must be sign-in or sign-up"})
		return
	}
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	authorizationURL, err := h.workos.AuthorizationURL(screenHint, state)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": authorizationURL})
}

type workOSCallbackRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

func (h *AuthHandler) WorkOSCallback(w http.ResponseWriter, r *http.Request) {
	if h.workos == nil || !h.workos.Enabled() {
		writeJSON(w, http.StatusServiceUnavailable, models.ErrorResponse{Error: "WorkOS AuthKit is not configured"})
		return
	}
	var req workOSCallbackRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	if req.Code == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "code is required"})
		return
	}
	result, err := h.workos.AuthenticateWithCode(r.Context(), req.Code, clientIP(r), r.UserAgent())
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "authentication failed"})
		return
	}
	email := strings.ToLower(strings.TrimSpace(result.User.Email))
	name := workosauth.DisplayName(result.User)
	user, err := h.userRepo.UpsertFromWorkOS(r.Context(), result.User.ID, email, name)
	if errors.Is(err, repository.ErrLocalPasswordAccount) {
		writeJSON(w, http.StatusConflict, models.ErrorResponse{Error: "an account with this email already exists"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to persist authenticated user"})
		return
	}
	token, err := h.generateToken(user)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to generate token"})
		return
	}
	writeJSON(w, http.StatusOK, models.AuthResponse{Token: token, User: *user})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if h.rejectPasswordAuth(w) {
		return
	}
	var req models.RegisterRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Name == "" || req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "name, email, and password are required"})
		return
	}

	if len(req.Password) < 8 || len(req.Password) > 72 {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "password must be between 8 and 72 characters"})
		return
	}
	address, err := mail.ParseAddress(req.Email)
	if err != nil || address.Address != req.Email {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "email must be valid"})
		return
	}

	exists, err := h.userRepo.EmailExists(r.Context(), req.Email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "internal error"})
		return
	}
	if exists {
		writeJSON(w, http.StatusConflict, models.ErrorResponse{Error: "email already registered"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "internal error"})
		return
	}

	user := &models.User{
		ID:           models.NewID(),
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}

	if err := h.userRepo.Create(r.Context(), user); err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to create user"})
		return
	}

	token, err := h.generateToken(user)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to generate token"})
		return
	}

	writeJSON(w, http.StatusCreated, models.AuthResponse{Token: token, User: *user})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if h.rejectPasswordAuth(w) {
		return
	}
	var req models.LoginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "email and password are required"})
		return
	}

	user, err := h.userRepo.GetByEmail(r.Context(), req.Email)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "invalid email or password"})
		return
	}

	if user.PasswordHash == "" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		writeJSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "invalid email or password"})
		return
	}

	token, err := h.generateToken(user)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to generate token"})
		return
	}

	writeJSON(w, http.StatusOK, models.AuthResponse{Token: token, User: *user})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: "user not found"})
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) generateToken(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     time.Now().Add(72 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		return forwarded
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i >= 0 {
		return host[:i]
	}
	return host
}
