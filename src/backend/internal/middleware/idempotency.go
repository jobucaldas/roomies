package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"

	"github.com/roomies/backend/internal/repository"
)

const maxIdempotencyBodyBytes = 1 << 20

func Idempotency(repo *repository.ReliabilityRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if repo == nil || !supportsIdempotency(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}
			userID := GetUserID(r.Context())
			if userID == "" {
				next.ServeHTTP(w, r)
				return
			}
			body, err := io.ReadAll(io.LimitReader(r.Body, maxIdempotencyBodyBytes+1))
			if err != nil {
				writeError(w, http.StatusBadRequest, "failed to read request body")
				return
			}
			if len(body) > maxIdempotencyBodyBytes {
				writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(body))
			fingerprint := bodySHA256(body)

			stored, err := repo.StartIdempotentRequest(r.Context(), userID, key, r.Method, r.URL.Path, fingerprint)
			if err == nil && stored != nil {
				if stored.ContentType != "" {
					w.Header().Set("Content-Type", stored.ContentType)
				}
				w.Header().Set("X-Idempotent-Replay", "true")
				w.WriteHeader(stored.StatusCode)
				_, _ = w.Write([]byte(stored.Body))
				return
			}
			if err == repository.ErrIdempotencyConflict {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			if err == repository.ErrIdempotencyInProgress {
				writeError(w, http.StatusConflict, "request already in progress")
				return
			}
			if err != nil {
				writeError(w, http.StatusInternalServerError, "idempotency unavailable")
				return
			}

			capture := &responseCapture{ResponseWriter: w}
			next.ServeHTTP(capture, r)
			status := capture.statusCode
			if status == 0 {
				status = http.StatusOK
			}
			if status >= http.StatusInternalServerError {
				_ = repo.AbortIdempotentRequest(r.Context(), userID, key)
				return
			}
			contentType := capture.Header().Get("Content-Type")
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			_ = repo.CompleteIdempotentRequest(r.Context(), userID, key, repository.StoredHTTPResponse{
				StatusCode:  status,
				Body:        capture.body.String(),
				ContentType: contentType,
			})
		})
	}
}

func supportsIdempotency(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}

func bodySHA256(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

type responseCapture struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (w *responseCapture) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseCapture) Write(body []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	w.body.Write(body)
	return w.ResponseWriter.Write(body)
}
