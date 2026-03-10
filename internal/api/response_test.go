package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		data           interface{}
		expectSuccess  bool
		expectDataNil  bool
		expectDataJSON string
	}{
		{
			name:           "200 with string data",
			status:         http.StatusOK,
			data:           "hello",
			expectSuccess:  true,
			expectDataJSON: `"hello"`,
		},
		{
			name:          "200 with nil data",
			status:        http.StatusOK,
			data:          nil,
			expectSuccess: true,
			expectDataNil: true,
		},
		{
			name:           "201 with map data",
			status:         http.StatusCreated,
			data:           map[string]string{"key": "value"},
			expectSuccess:  true,
			expectDataJSON: `{"key":"value"}`,
		},
		{
			name:           "200 with slice data",
			status:         http.StatusOK,
			data:           []int{1, 2, 3},
			expectSuccess:  true,
			expectDataJSON: `[1,2,3]`,
		},
		{
			name:           "400 sets success false",
			status:         http.StatusBadRequest,
			data:           "bad request",
			expectSuccess:  false,
			expectDataJSON: `"bad request"`,
		},
		{
			name:           "500 sets success false",
			status:         http.StatusInternalServerError,
			data:           "server error",
			expectSuccess:  false,
			expectDataJSON: `"server error"`,
		},
		{
			name:          "204 with nil data is success",
			status:        http.StatusNoContent,
			data:          nil,
			expectSuccess: true,
			expectDataNil: true,
		},
		{
			name:           "299 is still success",
			status:         299,
			data:           "edge case",
			expectSuccess:  true,
			expectDataJSON: `"edge case"`,
		},
		{
			name:           "300 is not success",
			status:         300,
			data:           "redirect",
			expectSuccess:  false,
			expectDataJSON: `"redirect"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeJSON(rec, tt.status, tt.data)

			res := rec.Result()
			defer func() { _ = res.Body.Close() }()

			// Check status code
			if res.StatusCode != tt.status {
				t.Errorf("status = %d, want %d", res.StatusCode, tt.status)
			}

			// Check content type
			ct := res.Header.Get("Content-Type")
			if ct != "application/json" {
				t.Errorf("Content-Type = %q, want %q", ct, "application/json")
			}

			// Decode response
			var resp APIResponse
			if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Success != tt.expectSuccess {
				t.Errorf("success = %v, want %v", resp.Success, tt.expectSuccess)
			}

			if resp.Error != "" {
				t.Errorf("error should be empty in writeJSON, got %q", resp.Error)
			}

			if tt.expectDataNil {
				if resp.Data != nil {
					t.Errorf("expected data to be nil, got %v", resp.Data)
				}
			} else {
				if resp.Data == nil {
					t.Fatal("expected data to be non-nil")
				}
				// Marshal the data field and compare
				gotData, err := json.Marshal(resp.Data)
				if err != nil {
					t.Fatalf("failed to marshal data: %v", err)
				}
				if string(gotData) != tt.expectDataJSON {
					t.Errorf("data = %s, want %s", string(gotData), tt.expectDataJSON)
				}
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		message string
	}{
		{
			name:    "400 bad request",
			status:  http.StatusBadRequest,
			message: "invalid input",
		},
		{
			name:    "404 not found",
			status:  http.StatusNotFound,
			message: "resource not found",
		},
		{
			name:    "500 internal error",
			status:  http.StatusInternalServerError,
			message: "something went wrong",
		},
		{
			name:    "403 forbidden",
			status:  http.StatusForbidden,
			message: "access denied",
		},
		{
			name:    "empty error message",
			status:  http.StatusBadRequest,
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeError(rec, tt.status, tt.message)

			res := rec.Result()
			defer func() { _ = res.Body.Close() }()

			// Check status code
			if res.StatusCode != tt.status {
				t.Errorf("status = %d, want %d", res.StatusCode, tt.status)
			}

			// Check content type
			ct := res.Header.Get("Content-Type")
			if ct != "application/json" {
				t.Errorf("Content-Type = %q, want %q", ct, "application/json")
			}

			// Decode response
			var resp APIResponse
			if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			// Success must always be false for errors
			if resp.Success {
				t.Error("success should be false for error responses")
			}

			if resp.Error != tt.message {
				t.Errorf("error = %q, want %q", resp.Error, tt.message)
			}

			// Data should be nil/omitted for error responses
			if resp.Data != nil {
				t.Errorf("data should be nil for error responses, got %v", resp.Data)
			}
		})
	}
}
