package main

import (
	"log/slog"
)

func main() {
	if err := fetchGitHubStats(); err != nil {
		slog.Error("fetchGitHubStats failed", "err", err)
		return
	}
	slog.Info("fetchGitHubStats succeeded")

	if err := fetchForgejoStats(); err != nil {
		slog.Error("fetchForgejoStats failed", "err", err)
		return
	}
	slog.Info("fetchForgejoStats succeeded")

	if err := fetchForgejoStats("archive"); err != nil {
		slog.Error("fetchForgejoStats failed", "org", "archive", "err", err)
		return
	}
	slog.Info("fetchForgejoStats succeeded", "org", "archive")

	if err := fetchForgejoStats("baania"); err != nil {
		slog.Error("fetchForgejoStats failed", "org", "baania", "err", err)
		return
	}
	slog.Info("fetchForgejoStats succeeded", "org", "baania")
}
