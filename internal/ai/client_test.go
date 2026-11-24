package ai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestGenerateStreamCancelStopsUpstream(t *testing.T) {
	var cancelOnce sync.Once
	upstreamCancelled := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		for {
			select {
			case <-r.Context().Done():
				cancelOnce.Do(func() { close(upstreamCancelled) })
				return
			case <-time.After(10 * time.Millisecond):
			}
			_, _ = fmt.Fprintf(w, "{\"response\":\"tok\",\"done\":false}\n")
			flusher.Flush()
		}
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "test-model")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader, err := client.GenerateStream(ctx, "", "diagnose this")
	if err != nil {
		t.Fatalf("GenerateStream() error = %v", err)
	}

	streamErr := make(chan error, 1)
	go func() {
		streamErr <- StreamTokens(reader, func(token string, _ bool) error {
			// Simulate the browser disconnecting mid-stream.
			cancel()
			return nil
		})
	}()

	select {
	case <-upstreamCancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("upstream request was not cancelled when the context was cancelled")
	}

	select {
	case err := <-streamErr:
		if err == nil {
			t.Fatal("StreamTokens() error = nil, want read error after cancellation")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("StreamTokens did not return after cancellation")
	}
}

func TestGenerateStreamDeliversTokens(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = fmt.Fprint(w, "{\"response\":\"hello\",\"done\":false}\n{\"response\":\" world\",\"done\":true}\n")
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "test-model")
	reader, err := client.GenerateStream(context.Background(), "test-model", "hi")
	if err != nil {
		t.Fatalf("GenerateStream() error = %v", err)
	}

	var got string
	var sawDone bool
	if err := StreamTokens(reader, func(token string, done bool) error {
		got += token
		sawDone = sawDone || done
		return nil
	}); err != nil {
		t.Fatalf("StreamTokens() error = %v", err)
	}
	if got != "hello world" {
		t.Errorf("tokens = %q, want %q", got, "hello world")
	}
	if !sawDone {
		t.Error("StreamTokens never reported done")
	}
}

func TestGetStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	connected, err := NewClient(srv.URL, "test-model").GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if !connected {
		t.Error("GetStatus() connected = false, want true")
	}
}

func TestListModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"models":[{"name":"llama3.1"}]}`)
	}))
	t.Cleanup(srv.Close)

	models, err := NewClient(srv.URL, "test-model").ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if len(models) != 1 || models[0].Name != "llama3.1" {
		t.Errorf("models = %+v, want one model named llama3.1", models)
	}
}
