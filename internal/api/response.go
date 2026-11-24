package api

import (
	"encoding/json"
	"errors"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/sebrandon1/ztp-dashboard/internal/hub"
	"github.com/sebrandon1/ztp-dashboard/internal/spoke"
)

// statusForError maps an error to the HTTP status code that best describes it.
// It unwraps sentinel errors (via errors.Is) and Kubernetes API errors, so
// handlers do not need to inspect error strings.
func statusForError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, hub.ErrNotConnected), errors.Is(err, spoke.ErrNotConnected):
		return http.StatusServiceUnavailable
	case errors.Is(err, hub.ErrResourceNotAllowed):
		return http.StatusForbidden
	case apierrors.IsNotFound(err):
		return http.StatusNotFound
	case apierrors.IsForbidden(err):
		return http.StatusForbidden
	case apierrors.IsConflict(err), apierrors.IsAlreadyExists(err):
		return http.StatusConflict
	case apierrors.IsBadRequest(err), apierrors.IsInvalid(err):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Success: status >= 200 && status < 300,
		Data:    data,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   message,
	})
}
