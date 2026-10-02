package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/roomies/backend/internal/models"
)

type contextKey string

const userIDKey contextKey = "user_id"
const userEmailKey contextKey = "user_email"

const (
	// SessionCookieName is the HttpOnly session JWT. JavaScript cannot read it.
	SessionCookieName = "roomies_session"
	// SessionHintCookieName is a non-secret flag so the web app can decide
	// whether to call /api/auth/me without probing on every anonymous visit.
	SessionHintCookieName = "roomies_session_hint"
)

func GetUserID(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

func GetUserEmail(ctx context.Context) string {
	v, _ := ctx.Value(userEmailKey).(string)
	return v
}

// SessionToken returns the raw session JWT from the request, if any.
func SessionToken(r *http.Request) (string, bool) {
	return sessionToken(r)
}

// sessionToken prefers an Authorization bearer so API clients and tests keep
// working, and otherwise reads the HttpOnly session cookie used by Flutter web.
func sessionToken(r *http.Request) (string, bool) {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			return "", false
		}
		raw := strings.TrimSpace(parts[1])
		return raw, raw != ""
	}
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return "", false
	}
	raw := strings.TrimSpace(cookie.Value)
	return raw, raw != ""
}

func JWTAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, ok := sessionToken(r)
			if !ok {
				writeError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			token, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
				if t.Method != jwt.SigningMethodHS256 {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

			if err != nil || !token.Valid {
				writeError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeError(w, http.StatusUnauthorized, "invalid token claims")
				return
			}

			userID, _ := claims["user_id"].(string)
			email, _ := claims["email"].(string)

			if userID == "" {
				writeError(w, http.StatusUnauthorized, "invalid token payload")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			ctx = context.WithValue(ctx, userEmailKey, email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(houseRepo interface {
	GetMember(ctx context.Context, houseID, userID string) (*models.HouseMember, error)
}, allowedRoles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserID(r.Context())
			if userID == "" {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			houseID := chi.URLParam(r, "id")
			if houseID == "" {
				writeError(w, http.StatusBadRequest, "house id required")
				return
			}

			member, err := houseRepo.GetMember(r.Context(), houseID, userID)
			if err != nil {
				writeError(w, http.StatusForbidden, "not a member of this house")
				return
			}

			if !allowed[member.Role] {
				writeError(w, http.StatusForbidden, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"error":"` + msg + `"}`))
}
