package main

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"
)

type forgejoStatsCache struct {
	FetchedAt time.Time      `json:"fetched_at"`
	Languages []languageStat `json:"languages"`
}

// fetchForgejoStats fetches the authenticated user's repositories, or an organization's.
func fetchForgejoStats(organization ...string) error {
	if len(organization) > 1 {
		return fmt.Errorf("expected at most one Forgejo organization")
	}
	org := ""
	path := "data/forgejo.json"
	if len(organization) == 1 {
		org = organization[0]
		if org == "" || strings.ContainsAny(org, "/\\") || org == "." || org == ".." {
			return fmt.Errorf("invalid Forgejo organization %q", org)
		}
		path = "data/forgejo-" + org + ".json"
	}
	cached, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read Forgejo cache: %w", err)
	}
	if err == nil {
		var cache forgejoStatsCache
		if json.Unmarshal(cached, &cache) == nil {
			age := time.Since(cache.FetchedAt)
			if !cache.FetchedAt.IsZero() && cache.Languages != nil && age >= 0 && age <= 24*time.Hour {
				return nil
			}
		}
	}

	url := os.Getenv("FORGEJO_URL")
	if url == "" {
		url = "https://codeberg.org"
	}
	token := os.Getenv("FORGEJO_TOKEN")
	if token == "" {
		return fmt.Errorf("FORGEJO_TOKEN is required")
	}
	client, err := gitea.NewClient(url, gitea.SetToken(token))
	if err != nil {
		return fmt.Errorf("create Forgejo client: %w", err)
	}
	stats, err := forgejoLanguageStats(client, org)
	if err != nil {
		return err
	}
	data, err := json.Marshal(forgejoStatsCache{
		FetchedAt: time.Now().UTC(), Languages: stats,
	}, jsontext.WithIndent("  "))
	if err != nil {
		return fmt.Errorf("encode Forgejo stats: %w", err)
	}
	if err := os.MkdirAll("data", 0o755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write Forgejo stats: %w", err)
	}
	return nil
}

func forgejoLanguageStats(client *gitea.Client, org string) ([]languageStat, error) {
	owner := org
	if owner == "" {
		user, _, err := client.GetMyUserInfo()
		if err != nil {
			return nil, fmt.Errorf("fetch Forgejo user: %w", err)
		}
		owner = user.UserName
	}
	totals := make(map[string]int64)
	var total int64
	for page := 1; ; page++ {
		options := gitea.ListOptions{Page: page, PageSize: 100}
		var repos []*gitea.Repository
		var err error
		if org == "" {
			repos, _, err = client.ListUserRepos(owner, gitea.ListReposOptions{ListOptions: options})
		} else {
			repos, _, err = client.ListOrgRepos(org, gitea.ListOrgReposOptions{ListOptions: options})
		}
		if err != nil {
			return nil, fmt.Errorf("list Forgejo repositories for %s: %w", owner, err)
		}
		// Continue until an empty page: servers may cap the requested page size.
		if len(repos) == 0 {
			break
		}
		for _, repo := range repos {
			languages, _, err := client.GetRepoLanguages(owner, repo.Name)
			if err != nil {
				return nil, fmt.Errorf("fetch Forgejo languages for %s/%s: %w", owner, repo.Name, err)
			}
			for language, bytes := range languages {
				totals[language] += bytes
				total += bytes
			}
		}
	}
	return aggregateLanguageStats(totals, total), nil
}
