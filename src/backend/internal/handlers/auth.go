package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/roomies/backend/internal/middleware"
	"github.com/roomies/backend/internal/models"
	"github.com/roomies/backend/internal/repository"
	"github.com/roomies/backend/internal/workosauth"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultSessionTTL    = 8 * time.Hour
	oauthTransactionTTL  = 10 * time.Minute
	oauthStateCookie     = "roomies_oauth_state"
	oauthVerifierCookie  = "roomies_oauth_verifier"
	defaultLoginLimit    = 40
	defaultRegisterLimit = 90
	defaultCallbackLimit = 15
)

type AuthHandler struct {
	userRepo      *repository.UserRepository
	jwtSecret     string
	workos        *workosauth.Client
	publicBaseURL string
	limiter       *middleware.WindowLimiter
	loginLimit    int
	registerLimit int
	callbackLimit int
	sessionTTL    time.Duration
}

func NewAuthHandler(userRepo *repository.UserRepository, jwtSecret string, workos *workosauth.Client) *AuthHandler {
	return &AuthHandler{
		userRepo:      userRepo,
		jwtSecret:     jwtSecret,
		workos:        workos,
		limiter:       middleware.NewWindowLimiter(time.Minute),
		loginLimit:    defaultLoginLimit,
		registerLimit: defaultRegisterLimit,
		callbackLimit: defaultCallbackLimit,
		sessionTTL:    defaultSessionTTL,
	}
}

// SetSessionTTL sets the access-token and session-cookie lifetime.
func (h *AuthHandler) SetSessionTTL(ttl time.Duration) {
	if ttl > 0 {
		h.sessionTTL = ttl
	}
}

// SetPublicBaseURL controls the Secure flag on session cookies. HTTPS origins
// always mark cookies Secure, including when TLS terminates at the proxy.
func (h *AuthHandler) SetPublicBaseURL(base string) {
	h.publicBaseURL = strings.TrimRight(strings.TrimSpace(base), "/")
}

// SetRateLimit overrides the per-IP budget for login, register, and the AuthKit
// code exchange. Tests use a tiny limit; production keeps the defaults.
func (h *AuthHandler) SetRateLimit(limit int) {
	h.loginLimit = limit
	h.registerLimit = limit
	h.callbackLimit = limit
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
	state, verifier, challenge, err := newOAuthTransaction()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to start sign-in"})
		return
	}
	authorizationURL, err := h.workos.AuthorizationURL(screenHint, state, challenge)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to start sign-in"})
		return
	}
	secure := h.secureCookie(r)
	setCookie(w, oauthStateCookie, state, int(oauthTransactionTTL.Seconds()), true, secure)
	setCookie(w, oauthVerifierCookie, verifier, int(oauthTransactionTTL.Seconds()), true, secure)
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
	if !h.allow(w, r, "workos_callback", h.callbackLimit) {
		return
	}
	var req workOSCallbackRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	req.State = strings.TrimSpace(req.State)
	stateCookie, stateErr := r.Cookie(oauthStateCookie)
	verifierCookie, verifierErr := r.Cookie(oauthVerifierCookie)
	secure := h.secureCookie(r)
	clearCookie(w, oauthStateCookie, true, secure)
	clearCookie(w, oauthVerifierCookie, true, secure)
	if req.Code == "" || req.State == "" || stateErr != nil || verifierErr != nil ||
		!fixedEqual(req.State, stateCookie.Value) || strings.TrimSpace(verifierCookie.Value) == "" {
		writeJSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "authentication failed"})
		return
	}
	result, err := h.workos.AuthenticateWithCode(r.Context(), req.Code, verifierCookie.Value, clientIP(r), r.UserAgent())
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
	h.finishAuth(w, r, user, http.StatusOK)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if !h.allow(w, r, "register", h.registerLimit) {
		return
	}
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

	h.finishAuth(w, r, user, http.StatusCreated)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if !h.allow(w, r, "login", h.loginLimit) {
		return
	}
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

	h.finishAuth(w, r, user, http.StatusOK)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	secure := h.secureCookie(r)
	clearCookie(w, middleware.SessionCookieName, true, secure)
	clearCookie(w, middleware.SessionHintCookieName, false, secure)
	w.WriteHeader(http.StatusNoContent)
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
		"exp":     time.Now().Add(h.sessionTTL).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}

func (h *AuthHandler) finishAuth(w http.ResponseWriter, r *http.Request, user *models.User, status int) {
	token, err := h.generateToken(user)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to generate token"})
		return
	}
	secure := h.secureCookie(r)
	maxAge := int(h.sessionTTL.Seconds())
	setCookie(w, middleware.SessionCookieName, token, maxAge, true, secure)
	setCookie(w, middleware.SessionHintCookieName, "1", maxAge, false, secure)
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Roomies-Client")), "web") {
		writeJSON(w, status, struct {
			User models.User `json:"user"`
		}{User: *user})
		return
	}
	writeJSON(w, status, models.AuthResponse{Token: token, User: *user})
}

func (h *AuthHandler) allow(w http.ResponseWriter, r *http.Request, route string, limit int) bool {
	if h.limiter == nil || h.limiter.Allow(route+"|"+rateLimitIP(r), limit) {
		return true
	}
	w.Header().Set("Retry-After", "60")
	writeJSON(w, http.StatusTooManyRequests, models.ErrorResponse{Error: "too many requests"})
	return false
}

func (h *AuthHandler) secureCookie(r *http.Request) bool {
	if r != nil && r.TLS != nil {
		return true
	}
	if r != nil && strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https") {
		return true
	}
	parsed, err := url.Parse(h.publicBaseURL)
	return err == nil && parsed.Scheme == "https"
}

func newOAuthTransaction() (state, verifier, challenge string, err error) {
	state, err = randomURLToken()
	if err != nil {
		return "", "", "", err
	}
	verifier, err = randomURLToken()
	if err != nil {
		return "", "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return state, verifier, challenge, nil
}

func randomURLToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func fixedEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func setCookie(w http.ResponseWriter, name, value string, maxAge int, httpOnly, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: httpOnly,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearCookie(w http.ResponseWriter, name string, httpOnly, secure bool) {
	setCookie(w, name, "", -1, httpOnly, secure)
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		return forwarded
	}
	return remoteIP(r)
}

// rateLimitIP uses the last X-Forwarded-For hop. Caddy replaces that header
// with the connecting client, so a browser cannot pick its own bucket.
func rateLimitIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if ip := strings.TrimSpace(parts[len(parts)-1]); ip != "" {
			return ip
		}
	}
	return remoteIP(r)
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || host == "" {
		return r.RemoteAddr
	}
	return host
}
