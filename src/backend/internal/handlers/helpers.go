package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/roomies/backend/internal/models"
	"github.com/roomies/backend/internal/repository"
)

const maxRequestBodyBytes = 1 << 20

func decodeJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain one JSON object")
		}
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeRepositoryError answers 400 with the message of a validation error and
// a generic 500 for anything else, so database and driver errors never reach
// clients.
func writeRepositoryError(w http.ResponseWriter, err error) {
	var input *repository.InputError
	if errors.As(err, &input) {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: input.Message})
		return
	}
	writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "internal error"})
}
