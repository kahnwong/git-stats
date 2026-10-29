package main

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestFetchForgejoStats(t *testing.T) {
	for _, org := range []string{"", "archive"} {
		t.Run("organization="+org, func(t *testing.T) {
			t.Chdir(t.TempDir())
			owner, endpoint, path := "alice", "/api/v1/users/alice/repos", "data/forgejo.json"
			if org != "" {
				owner, endpoint, path = org, "/api/v1/orgs/"+org+"/repos", "data/forgejo-"+org+".json"
			}
			requests := 0
			fail := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				w.Header().Set("Content-Type", "application/json")
				if fail {
					http.Error(w, "failed", http.StatusInternalServerError)
					return
				}
				switch r.URL.Path {
				case "/api/v1/version":
					fmt.Fprint(w, `{"version":"1.24.0"}`)
				case "/api/v1/user":
					if org != "" {
						t.Error("organization fetch requested current user")
					}
					fmt.Fprint(w, `{"login":"alice"}`)
				case endpoint:
					switch r.URL.Query().Get("page") {
					case "1":
						fmt.Fprint(w, `[{"name":"one"}]`)
					case "2":
						fmt.Fprint(w, `[{"name":"two"}]`)
					case "3":
						fmt.Fprint(w, `[]`)
					default:
						t.Errorf("unexpected page: %s", r.URL)
						fmt.Fprint(w, `[]`)
					}
				case "/api/v1/repos/" + owner + "/one/languages":
					fmt.Fprint(w, `{"Go":100,"Python":100}`)
				case "/api/v1/repos/" + owner + "/two/languages":
					fmt.Fprint(w, `{"Go":100}`)
				default:
					t.Errorf("unexpected request: %s", r.URL)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			t.Setenv("FORGEJO_URL", server.URL)
			t.Setenv("FORGEJO_TOKEN", "test-token")
			fetch := func() error {
				if org == "" {
					return fetchForgejoStats()
				}
				return fetchForgejoStats(org)
			}
			if err := fetch(); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var cache forgejoStatsCache
			if err := json.Unmarshal(data, &cache); err != nil {
				t.Fatal(err)
			}
			if cache.FetchedAt.IsZero() || len(cache.Languages) != 2 || cache.Languages[0] != (languageStat{"Go", 200, "66.67%"}) {
				t.Fatalf("unexpected cache: %+v", cache)
			}
			count := requests
			if err := fetch(); err != nil {
				t.Fatal(err)
			}
			if requests != count {
				t.Fatal("fresh cache made API requests")
			}
			for _, stale := range []string{`[]`, `invalid`, `{"languages":[]}`, `{"fetched_at":"2000-01-01T00:00:00Z","languages":[]}`} {
				if err := os.WriteFile(path, []byte(stale), 0o644); err != nil {
					t.Fatal(err)
				}
				count = requests
				if err := fetch(); err != nil {
					t.Fatal(err)
				}
				if requests == count {
					t.Fatal("stale cache did not fetch")
				}
			}
			cache.FetchedAt = time.Now().Add(-25 * time.Hour)
			stale, err := json.Marshal(cache)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, stale, 0o644); err != nil {
				t.Fatal(err)
			}
			fail = true
			if err := fetch(); err == nil {
				t.Fatal("expected API error")
			}
			preserved, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(preserved) != string(stale) {
				t.Fatal("failed fetch changed cache")
			}
		})
	}
}

func TestFetchForgejoStatsInvalidOrganization(t *testing.T) {
	for _, args := range [][]string{{""}, {"../archive"}, {"one", "two"}} {
		if err := fetchForgejoStats(args...); err == nil {
			t.Errorf("expected error for %v", args)
		}
	}
}
