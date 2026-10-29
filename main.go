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
}
