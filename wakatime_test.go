package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateWakaTimeSVG(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/v1/users/current/stats/last_7_days" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Basic "+base64.StdEncoding.EncodeToString([]byte("secret")) {
			t.Errorf("incorrect authorization: %q", got)
		}
		fmt.Fprint(w, `{"data":{"languages":[{"name":"Go","total_seconds":1.5},{"name":"HTML","total_seconds":6},{"name":"A&B","total_seconds":0.5}]}}`)
	}))
	defer server.Close()
	directory := filepath.Join(t.TempDir(), "data")
	svg, err := generateWakaTimeSVG(t.Context(), directory, "secret", server.URL+"/api/v1/", 0, []string{"html"}, map[string]string{"Go": "#00add8"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"WakaTime Stats (last 7 days)", "coding time", "Go 75%", "A&amp;B 25%", "#00add8"} {
		if !strings.Contains(string(svg), want) {
			t.Errorf("SVG missing %q", want)
		}
	}
	if strings.Contains(string(svg), "HTML") || strings.Contains(string(svg), "by bytes") {
		t.Error("SVG contains excluded language or Git description")
	}
	written, err := os.ReadFile(filepath.Join(directory, "wakatime-stats.svg"))
	if err != nil || !bytes.Equal(written, svg) {
		t.Fatalf("output mismatch: %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, "stats.svg")); !os.IsNotExist(err) {
		t.Fatalf("Git output should not be written: %v", err)
	}
}

func TestWakaTimeErrorsAndEmptyStats(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		wantError  bool
	}{
		{"unauthorized", `{}`, 401, true},
		{"rate limited", `{}`, 429, true},
		{"malformed", `{`, 200, true},
		{"missing data", `{}`, 200, true},
		{"missing languages", `{"data":{}}`, 200, true},
		{"negative time", `{"data":{"languages":[{"name":"Go","total_seconds":-1}]}}`, 200, true},
		{"overflow", `{"data":{"languages":[{"name":"Go","total_seconds":1e30}]}}`, 200, true},
		{"empty", `{"data":{"languages":[]}}`, 200, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			directory := t.TempDir()
			path := filepath.Join(directory, "wakatime-stats.svg")
			if err := os.WriteFile(path, []byte("previous"), 0o644); err != nil {
				t.Fatal(err)
			}
			svg, err := generateWakaTimeSVG(t.Context(), directory, "secret", server.URL, 0, nil, nil)
			if (err != nil) != tc.wantError {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantError {
				data, err := os.ReadFile(path)
				if err != nil || string(data) != "previous" {
					t.Fatalf("failed request changed existing output: %v", err)
				}
			} else if !strings.Contains(string(svg), "No language data available") {
				t.Error("missing empty-state message")
			}
		})
	}
}

func TestWakaTimeInvalidConfiguration(t *testing.T) {
	for _, tc := range []struct{ key, url string }{
		{"", ""}, {"secret", "://bad"}, {"secret", "file:///tmp/stats"},
	} {
		if _, err := wakaTimeLanguageStats(t.Context(), tc.key, tc.url); err == nil {
			t.Error("expected configuration error")
		}
	}
}
