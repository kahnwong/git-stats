package main

import (
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatsHandlerRefreshesBeforeRendering(t *testing.T) {
	dir := t.TempDir()
	calls := 0
	handler, err := newStatsHandler(dir, func() error {
		calls++
		return os.WriteFile(filepath.Join(dir, "github.json"), []byte(`{"languages":[{"language":"Go","bytes":100}]}`), 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("fetched during handler initialization")
	}
	for i := range 2 {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), "Go 100%") {
			t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
		}
		saved, err := os.ReadFile(filepath.Join(dir, "stats.svg"))
		if err != nil || string(saved) != w.Body.String() {
			t.Fatalf("saved SVG differs from response: %v", err)
		}
		if calls != i+1 {
			t.Fatalf("refresh calls = %d, want %d", calls, i+1)
		}
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/?limit=invalid", nil))
	if w.Code != 400 || calls != 2 {
		t.Fatal("invalid query should not refresh")
	}
}

func TestStatsHandlerRefreshFailure(t *testing.T) {
	handler, err := newStatsHandler(t.TempDir(), func() error { return errors.New("upstream failure") })
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 500 || strings.Contains(w.Body.String(), "<svg") {
		t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
	}
}
