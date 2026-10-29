package main

import (
	"cmp"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

const githubLanguagesQuery = `
query($endCursor: String) {
  viewer {
    repositories(first: 100, after: $endCursor, ownerAffiliations: OWNER, isFork: false) {
      pageInfo { hasNextPage endCursor }
      nodes {
        languages(first: 20) {
          edges {
            size
            node { name }
          }
        }
      }
    }
  }
}`

type languageStat struct {
	Language   string `json:"language"`
	Bytes      int64  `json:"bytes"`
	Percentage string `json:"percentage"`
}

type githubStatsCache struct {
	FetchedAt time.Time      `json:"fetched_at"`
	Languages []languageStat `json:"languages"`
}

func githubCacheFresh(data []byte, now time.Time) bool {
	var cache githubStatsCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return false
	}
	age := now.Sub(cache.FetchedAt)
	return !cache.FetchedAt.IsZero() && cache.Languages != nil && age >= 0 && age <= 24*time.Hour
}

type graphqlClient interface {
	Do(string, map[string]any, any) error
}

func fetchGitHubStats() error {
	cached, err := os.ReadFile("data/github.json")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read GitHub stats cache: %w", err)
	}
	if err == nil && githubCacheFresh(cached, time.Now()) {
		return nil
	}

	client, err := api.DefaultGraphQLClient()
	if err != nil {
		return fmt.Errorf("create GitHub client: %w", err)
	}
	stats, err := githubLanguageStats(client)
	if err != nil {
		return err
	}
	cache := githubStatsCache{FetchedAt: time.Now().UTC(), Languages: stats}
	data, err := json.Marshal(cache, jsontext.WithIndent("  "))
	if err != nil {
		return fmt.Errorf("encode GitHub stats: %w", err)
	}
	if err := os.MkdirAll("data", 0o755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	if err := os.WriteFile("data/github.json", append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write GitHub stats: %w", err)
	}
	return nil
}

func githubLanguageStats(client graphqlClient) ([]languageStat, error) {
	variables := map[string]any{"endCursor": nil}
	totals := make(map[string]int64)
	var total int64
	for {
		var response struct {
			Viewer struct {
				Repositories struct {
					PageInfo struct {
						HasNextPage bool
						EndCursor   string
					}
					Nodes []struct {
						Languages struct {
							Edges []struct {
								Size int64
								Node struct{ Name string }
							}
						}
					}
				}
			}
		}
		if err := client.Do(githubLanguagesQuery, variables, &response); err != nil {
			return nil, fmt.Errorf("fetch GitHub languages: %w", err)
		}
		repositories := response.Viewer.Repositories
		for _, repository := range repositories.Nodes {
			for _, edge := range repository.Languages.Edges {
				totals[edge.Node.Name] += edge.Size
				total += edge.Size
			}
		}
		if !repositories.PageInfo.HasNextPage {
			break
		}
		cursor := repositories.PageInfo.EndCursor
		if cursor == "" || variables["endCursor"] == cursor {
			return nil, fmt.Errorf("GitHub pagination did not advance")
		}
		variables["endCursor"] = cursor
	}

	stats := make([]languageStat, 0, len(totals))
	for language, bytes := range totals {
		var percentage float64
		if total > 0 {
			percentage = math.Round(float64(bytes)/float64(total)*10000) / 100
		}
		stats = append(stats, languageStat{
			Language:   language,
			Bytes:      bytes,
			Percentage: strconv.FormatFloat(percentage, 'f', -1, 64) + "%",
		})
	}
	slices.SortFunc(stats, func(a, b languageStat) int {
		return cmp.Or(cmp.Compare(b.Bytes, a.Bytes), cmp.Compare(a.Language, b.Language))
	})
	return stats, nil
}
