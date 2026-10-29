package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateStatsSVG(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	fetched := false
	fetch := func() error {
		fetched = true
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, "github.json"), []byte(`{"languages":[{"language":"Go","bytes":100},{"language":"Python","bytes":50}]}`), 0o644)
	}
	svg, err := generateStatsSVG(dir, 1, []string{"Go"}, nil, fetch)
	if err != nil {
		t.Fatal(err)
	}
	if !fetched || !strings.Contains(string(svg), "Python 100%") || strings.Contains(string(svg), "Go ") {
		t.Fatal("fetch or filters not applied")
	}
	written, err := os.ReadFile(filepath.Join(dir, "stats.svg"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(written, svg) {
		t.Fatal("saved SVG differs from rendered SVG")
	}
	assertValidSVG(t, string(written))
	want := errors.New("fetch failed")
	if _, err := generateStatsSVG(dir, 0, nil, nil, func() error { return want }); !errors.Is(err, want) {
		t.Fatalf("unexpected error: %v", err)
	}
	preserved, err := os.ReadFile(filepath.Join(dir, "stats.svg"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(preserved, written) {
		t.Fatal("failed generation changed SVG")
	}
	temps, err := filepath.Glob(filepath.Join(dir, ".stats-*.svg"))
	if err != nil || len(temps) != 0 {
		t.Fatal("temporary files left behind")
	}
}
