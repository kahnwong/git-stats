package main

import (
	"encoding/json/v2"
	"os"
	"testing"
	"time"
)

func TestGitHubCacheFresh(t *testing.T) {
	now := time.Date(2026, time.June, 1, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		age  time.Duration
		want bool
	}{
		{"fresh", time.Hour, true},
		{"exactly one day", 24 * time.Hour, true},
		{"expired", 24*time.Hour + time.Nanosecond, false},
		{"future", -time.Hour, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(githubStatsCache{FetchedAt: now.Add(-tc.age), Languages: []languageStat{}})
			if err != nil {
				t.Fatal(err)
			}
			if got := githubCacheFresh(data, now); got != tc.want {
				t.Fatalf("fresh = %v, want %v", got, tc.want)
			}
		})
	}
	for _, data := range []string{
		`[]`, `{`, `{}`, `{"languages":[]}`,
		`{"fetched_at":"invalid","languages":[]}`,
		`{"fetched_at":"2026-06-01T12:00:00Z"}`,
		`{"fetched_at":"2026-06-01T12:00:00Z","languages":null}`,
	} {
		if githubCacheFresh([]byte(data), now) {
			t.Errorf("invalid cache considered fresh: %s", data)
		}
	}
}

func TestFetchGitHubStatsSkipsFreshCache(t *testing.T) {
	t.Chdir(t.TempDir())
	// A fresh cache must work even without valid GitHub credentials.
	t.Setenv("GH_TOKEN", "invalid-token")
	data, err := json.Marshal(githubStatsCache{
		FetchedAt: time.Now().UTC(),
		Languages: []languageStat{{"Go", 100, "100%"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll("data", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("data/github.json", data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fetchGitHubStats(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile("data/github.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Fatal("fresh cache was rewritten")
	}
}
