package main

import (
	"context"
	"encoding/base64"
	"encoding/json/v2"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultWakaTimeAPIURL = "https://wakatime.com/api/v1"

func wakaTimeLanguageStats(ctx context.Context, apiKey, apiURL string) ([]languageStat, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("WakaTime API key cannot be empty")
	}
	if apiURL == "" {
		apiURL = defaultWakaTimeAPIURL
	}
	base, err := url.Parse(apiURL)
	if err != nil || base.Host == "" || (base.Scheme != "https" && base.Scheme != "http") || base.RawQuery != "" || base.Fragment != "" || base.User != nil {
		return nil, fmt.Errorf("invalid WakaTime API URL")
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/users/current/stats/last_7_days"
	base.RawPath = ""
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create WakaTime request: %w", err)
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(apiKey)))
	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch WakaTime stats: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch WakaTime stats: HTTP %d", response.StatusCode)
	}
	var payload struct {
		Data *struct {
			Languages []struct {
				Name         string  `json:"name"`
				TotalSeconds float64 `json:"total_seconds"`
			} `json:"languages"`
		} `json:"data"`
	}
	if err := json.UnmarshalRead(response.Body, &payload); err != nil {
		return nil, fmt.Errorf("decode WakaTime stats: %w", err)
	}
	if payload.Data == nil || payload.Data.Languages == nil {
		return nil, fmt.Errorf("missing WakaTime language statistics")
	}
	totals := make(map[string]int64)
	var total int64
	for _, language := range payload.Data.Languages {
		// Reuse the language chart's integer weights, in microseconds rather than bytes.
		weight := math.Round(language.TotalSeconds * 1e6)
		if language.Name == "" || math.IsNaN(weight) || math.IsInf(weight, 0) || weight < 0 || weight >= float64(math.MaxInt64) {
			return nil, fmt.Errorf("invalid WakaTime language statistics")
		}
		value := int64(weight)
		if value > math.MaxInt64-total {
			return nil, fmt.Errorf("WakaTime duration overflow")
		}
		totals[language.Name] += value
		total += value
	}
	return aggregateLanguageStats(totals, total), nil
}

func generateWakaTimeSVG(ctx context.Context, directory, apiKey, apiURL string, limit int, exclusions []string, colors map[string]string) ([]byte, error) {
	stats, err := wakaTimeLanguageStats(ctx, apiKey, apiURL)
	if err != nil {
		return nil, err
	}
	stats = displayedLanguageStats(stats, limit, exclusions)
	svg := renderLanguageUsageSVG(stats, colors, "WakaTime Stats (last 7 days)", "WakaTime language usage by coding time over the last 7 days.")
	return svg, writeStatsSVG(directory, "wakatime-stats.svg", svg)
}
