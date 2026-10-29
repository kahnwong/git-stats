package main

import (
	"log/slog"
)

func main() {
	if err := serveStats(); err != nil {
		slog.Error("web server failed", "err", err)
	}
}
