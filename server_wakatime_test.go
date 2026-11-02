package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWakaTimePreview(t *testing.T) {
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/users/current/stats/last_7_days" || r.Header.Get("Authorization") != "Basic c2VjcmV0" {
			t.Errorf("unexpected WakaTime request: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"data":{"languages":[{"name":"Go","total_seconds":6},{"name":"Python","total_seconds":3},{"name":"HTML","total_seconds":1}]}}`)
	}))
	defer upstream.Close()
	t.Setenv("WAKATIME_API_KEY", "secret")
	t.Setenv("WAKATIME_API_URL", upstream.URL+"/api/v1")
	directory := t.TempDir()
	handler, err := newStatsHandler(directory, func() error {
		t.Error("WakaTime preview must not fetch Git stats")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		query  string
		status int
		want   string
	}{
		{"", 200, "Go 60%"},
		{"?limit=1&exclude=go", 200, "Python 100%"},
		{"?exclude=Go&exclude=Python", 200, "HTML 100%"},
		{"?exclude=Go,Python,HTML", 200, "No language data available."},
		{"?limit=0", 400, "limit must be a positive integer"},
		{"?limit=-1", 400, "limit must be a positive integer"},
		{"?limit=abc", 400, "limit must be a positive integer"},
		{"?limit=1&limit=2", 400, "limit must be a positive integer"},
	} {
		t.Run(tc.query, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", "/wakatime"+tc.query, nil))
			if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.want) {
				t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
			}
			if tc.status == 200 {
				assertValidSVG(t, w.Body.String())
				if w.Header().Get("Content-Type") != "image/svg+xml; charset=utf-8" || w.Header().Get("Cache-Control") != "no-cache" {
					t.Error("missing SVG response headers")
				}
				data, err := os.ReadFile(filepath.Join(directory, "wakatime-stats.svg"))
				if err != nil || string(data) != w.Body.String() {
					t.Fatalf("output mismatch: %v", err)
				}
			}
		})
	}
	if calls != 4 {
		t.Errorf("upstream calls = %d, want 4", calls)
	}
}

func TestWakaTimePreviewErrors(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer upstream.Close()
	t.Setenv("WAKATIME_API_URL", upstream.URL)
	for _, tc := range []struct {
		key    string
		status int
	}{
		{"", http.StatusServiceUnavailable},
		{"secret", http.StatusInternalServerError},
	} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			t.Setenv("WAKATIME_API_KEY", tc.key)
			directory := t.TempDir()
			path := filepath.Join(directory, "wakatime-stats.svg")
			if err := os.WriteFile(path, []byte("previous"), 0o644); err != nil {
				t.Fatal(err)
			}
			handler, err := newStatsHandler(directory, func() error { return nil })
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", "/wakatime", nil))
			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d", w.Code, tc.status)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != "previous" {
				t.Fatalf("failed preview changed output: %v", err)
			}
			git := httptest.NewRecorder()
			handler.ServeHTTP(git, httptest.NewRequest("GET", "/", nil))
			if git.Code != http.StatusOK {
				t.Fatalf("Git preview affected: %d", git.Code)
			}
		})
	}
}
