package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestChecker_CheckLinks(t *testing.T) {
	// Мок-сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewChecker(5 * time.Second)

	tests := []struct {
		name     string
		links    []string
		wantLen  int
	}{
		{
			name:    "single link",
			links:   []string{server.URL},
			wantLen: 1,
		},
		{
			name:    "multiple links",
			links:   []string{server.URL, server.URL + "/page"},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := checker.CheckLinks(context.Background(), tt.links)
			if len(results) != tt.wantLen {
				t.Errorf("Expected %d results, got %d", tt.wantLen, len(results))
			}
		})
	}
}

func TestChecker_UnavailableLink(t *testing.T) {
	checker := NewChecker(2 * time.Second)

	results := checker.CheckLinks(context.Background(), []string{"http://this-domain-does-not-exist-12345.com"})

	if results["http://this-domain-does-not-exist-12345.com"] != StatusNotAvailable {
		t.Errorf("Expected 'not available' for invalid domain")
	}
}

func TestChecker_ContextCancellation(t *testing.T) {
	checker := NewChecker(30 * time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отмена сразу

	results := checker.CheckLinks(ctx, []string{"google.com"})

	if results["google.com"] != StatusNotAvailable {
		t.Errorf("Expected 'not available' for cancelled context")
	}
}
