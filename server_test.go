package main

import (
	"encoding/xml"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCombinedLanguageStats(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"github.json":          `{"languages":[{"language":"Go","bytes":100,"percentage":"99%"}]}`,
		"forgejo.json":         `{"languages":[{"language":"Python","bytes":100}]}`,
		"forgejo-archive.json": `{"languages":[{"language":"Go","bytes":100}]}`,
		"unrelated.json":       `invalid`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	stats, err := combinedLanguageStats(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != 2 || stats[0] != (languageStat{"Go", 200, "66.67%"}) || stats[1].Percentage != "33.33%" {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	handler, err := newStatsHandler(dir, func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), "image/svg+xml") {
		t.Fatalf("unexpected response: %v", w.Result())
	}
	if !strings.Contains(w.Body.String(), "Go 66.67%") || !strings.Contains(w.Body.String(), "#00ADD8") {
		t.Fatalf("missing language/color: %s", w.Body.String())
	}
	assertValidSVG(t, w.Body.String())
	if err := os.WriteFile(filepath.Join(dir, "forgejo.json"), []byte(`invalid`), 0o644); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 500 {
		t.Fatalf("got %d, want 500", w.Code)
	}
}

func assertValidSVG(t *testing.T, svg string) {
	t.Helper()
	decoder := xml.NewDecoder(strings.NewReader(svg))
	for {
		_, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("invalid SVG: %v", err)
		}
	}
}

func TestRenderLanguagesSVG(t *testing.T) {
	colors, err := languageColors()
	if err != nil {
		t.Fatal(err)
	}
	for _, stats := range [][]languageStat{
		{},
		{{"<script>&\"", 100, "100%"}},
		{{"Go", 0, "0%"}},
		{{"Go", 1, "50%"}, {"Python", 1, "50%"}, {"Unknown", 0, "0%"}},
	} {
		svg := string(renderLanguagesSVG(stats, colors))
		assertValidSVG(t, svg)
		if strings.Contains(svg, "<script>") || strings.Contains(svg, "NaN") {
			t.Fatalf("unsafe SVG: %s", svg)
		}
	}
}
