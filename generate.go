package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func generateStatsSVG(directory string, limit int, exclusions []string, colors map[string]string, fetchStats func() error) ([]byte, error) {
	if err := fetchStats(); err != nil {
		return nil, fmt.Errorf("refresh language statistics: %w", err)
	}
	stats, err := combinedLanguageStats(directory)
	if err != nil {
		return nil, err
	}
	stats = displayedLanguageStats(stats, limit, exclusions)
	svg := renderLanguagesSVG(stats, colors)
	return svg, writeStatsSVG(directory, "stats.svg", svg)
}

func writeStatsSVG(directory, name string, svg []byte) error {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create SVG directory: %w", err)
	}
	// Replace atomically so readers never see a partially written SVG.
	file, err := os.CreateTemp(directory, ".stats-*.svg")
	if err != nil {
		return fmt.Errorf("create SVG: %w", err)
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(svg); err != nil {
		file.Close()
		return fmt.Errorf("write SVG: %w", err)
	}
	if err := file.Chmod(0o644); err != nil {
		file.Close()
		return fmt.Errorf("set SVG permissions: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close SVG: %w", err)
	}
	if err := os.Rename(file.Name(), filepath.Join(directory, name)); err != nil {
		return fmt.Errorf("save SVG: %w", err)
	}
	return nil
}
