package main

import (
	"bytes"
	_ "embed"
	"encoding/json/v2"
	"fmt"
	"html"
	"log/slog"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

// GitHub Linguist colors: https://github.com/github-linguist/linguist
// License is included in assets/linguist-LICENSE.
//
//go:embed assets/languages.yml
var linguistLanguages []byte

func languageColors() (map[string]string, error) {
	var languages map[string]struct {
		Color string `yaml:"color"`
	}
	if err := yaml.Unmarshal(linguistLanguages, &languages); err != nil {
		return nil, fmt.Errorf("parse Linguist colors: %w", err)
	}
	colors := make(map[string]string, len(languages))
	for name, language := range languages {
		if language.Color != "" {
			colors[name] = language.Color
		}
	}
	return colors, nil
}

func combinedLanguageStats(directory string) ([]languageStat, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read stats directory: %w", err)
	}
	totals := make(map[string]int64)
	var total int64
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !(name == "github.json" || name == "forgejo.json" || (strings.HasPrefix(name, "forgejo-") && strings.HasSuffix(name, ".json"))) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		var cache githubStatsCache
		if err := json.Unmarshal(data, &cache); err != nil {
			return nil, fmt.Errorf("decode %s: %w", name, err)
		}
		if cache.Languages == nil {
			return nil, fmt.Errorf("missing languages in %s", name)
		}
		for _, stat := range cache.Languages {
			if stat.Bytes < 0 || stat.Bytes > math.MaxInt64-total {
				return nil, fmt.Errorf("invalid byte count in %s", name)
			}
			totals[stat.Language] += stat.Bytes
			total += stat.Bytes
		}
	}
	return aggregateLanguageStats(totals, total), nil
}

//go:embed assets/languages.svg.tmpl
var languagesSVGSource string

var languagesSVGTemplate = template.Must(template.New("languages").Parse(languagesSVGSource))

type svgLanguage struct {
	X, Width         float64
	Color, Label     string
	CircleX, CircleY int
	LabelX, LabelY   int
}

func renderLanguagesSVG(stats []languageStat, colors map[string]string) []byte {
	rows := (len(stats) + 1) / 2
	height := max(115, 90+rows*25)
	data := struct {
		Height, BorderHeight int
		Languages            []svgLanguage
	}{
		Height: height, BorderHeight: height - 1,
		Languages: make([]svgLanguage, 0, len(stats)),
	}
	var total float64
	for _, stat := range stats {
		total += float64(stat.Bytes)
	}
	x := 25.0
	for i, stat := range stats {
		color := colors[stat.Language]
		if color == "" {
			color = "#858585"
		}
		width := 0.0
		if total > 0 {
			width = float64(stat.Bytes) / total * 250
		}
		column, row := i/rows, i%rows
		labelX, labelY := 25+column*150, 90+row*25
		data.Languages = append(data.Languages, svgLanguage{
			X: x, Width: width,
			Color:   html.EscapeString(color),
			Label:   html.EscapeString(stat.Language + " " + stat.Percentage),
			CircleX: labelX + 5, CircleY: labelY - 4,
			LabelX: labelX + 15, LabelY: labelY,
		})
		x += width
	}
	var svg bytes.Buffer
	if err := languagesSVGTemplate.Execute(&svg, data); err != nil {
		panic(fmt.Errorf("render languages SVG: %w", err))
	}
	return svg.Bytes()
}

func displayedLanguageStats(stats []languageStat, limit int, exclusions []string) []languageStat {
	excluded := make(map[string]bool)
	for _, value := range exclusions {
		for name := range strings.SplitSeq(value, ",") {
			excluded[strings.ToLower(strings.TrimSpace(name))] = true
		}
	}
	totals := make(map[string]int64)
	var total int64
	// Combined statistics are already sorted by descending byte count.
	for _, stat := range stats {
		if excluded[strings.ToLower(stat.Language)] {
			continue
		}
		if limit > 0 && len(totals) >= limit {
			break
		}
		totals[stat.Language] = stat.Bytes
		total += stat.Bytes
	}
	return aggregateLanguageStats(totals, total)
}

func newStatsHandler(directory string, fetchStats func() error) (http.Handler, error) {
	colors, err := languageColors()
	if err != nil {
		return nil, err
	}
	var refreshMu sync.Mutex
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		limit := 0
		if query.Has("limit") {
			values := query["limit"]
			var err error
			limit, err = strconv.Atoi(values[0])
			if err != nil || limit < 1 || len(values) != 1 {
				http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
				return
			}
		}
		// Keep cache checks, writes, and reads serialized across requests.
		refreshMu.Lock()
		defer refreshMu.Unlock()
		svg, err := generateStatsSVG(directory, limit, query["exclude"], colors, fetchStats)
		if err != nil {
			slog.Error("generate language statistics", "err", err)
			http.Error(w, "Unable to generate language statistics", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(svg)
	})
	return mux, nil
}

func serveStats() error {
	handler, err := newStatsHandler("data", fetchAllStats)
	if err != nil {
		return err
	}
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	server := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	slog.Info("serving language statistics", "address", addr, "path", "/")
	return server.ListenAndServe()
}

func fetchAllStats() error {
	if err := fetchGitHubStats(); err != nil {
		return fmt.Errorf("fetch GitHub stats: %w", err)
	}
	if err := fetchForgejoStats(); err != nil {
		return fmt.Errorf("fetch personal Forgejo stats: %w", err)
	}
	for org := range strings.SplitSeq(os.Getenv("FORGEJO_ORGS"), ",") {
		org = strings.TrimSpace(org)
		if org == "" {
			continue
		}
		if err := fetchForgejoStats(org); err != nil {
			return fmt.Errorf("fetch Forgejo stats for %s: %w", org, err)
		}
	}
	return nil
}
