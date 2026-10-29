package main

import (
	"flag"
	"log/slog"
	"os"
)

func main() {
	dev := flag.Bool("dev", false, "start the development web server")
	limit := flag.Int("limit", 0, "maximum languages in the generated SVG (0 means all)")
	exclude := flag.String("exclude", "", "comma-separated languages to exclude from the generated SVG")
	flag.Parse()
	if *limit < 0 || flag.NArg() != 0 {
		slog.Error("limit must be nonnegative and positional arguments are not supported")
		os.Exit(1)
	}
	if *dev {
		if err := serveStats(); err != nil {
			slog.Error("web server failed", "err", err)
			os.Exit(1)
		}
		return
	}
	colors, err := languageColors()
	if err == nil {
		_, err = generateStatsSVG("data", *limit, []string{*exclude}, colors, fetchAllStats)
	}
	if err != nil {
		slog.Error("SVG generation failed", "err", err)
		os.Exit(1)
	}
	slog.Info("SVG generated", "path", "data/stats.svg")
}
