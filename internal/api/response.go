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

// maxJSONBodyBytes caps the request body size accepted by JSON handlers.
const maxJSONBodyBytes = 1 << 20 // 1 MiB

// decodeJSON decodes a JSON request body into v, rejecting bodies larger than
// maxJSONBodyBytes. Callers that allow an empty body can special-case io.EOF.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	return json.NewDecoder(http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)).Decode(v)
}

// writeDecodeError reports a request-body decode failure: 413 when the body
// exceeded maxJSONBodyBytes, 400 otherwise.
func writeDecodeError(w http.ResponseWriter, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
		return
	}
	writeError(w, http.StatusBadRequest, "invalid request body")
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   message,
	})
}
