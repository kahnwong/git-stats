package main

import (
	"encoding/json/v2"
	"errors"
	"reflect"
	"testing"
)

type fakeGraphQLClient struct {
	pages   []string
	cursors []any
	err     error
}

func (c *fakeGraphQLClient) Do(_ string, variables map[string]any, response any) error {
	c.cursors = append(c.cursors, variables["endCursor"])
	if c.err != nil {
		return c.err
	}
	page := c.pages[0]
	c.pages = c.pages[1:]
	return json.Unmarshal([]byte(page), response, json.MatchCaseInsensitiveNames(true))
}

func TestGitHubLanguageStats(t *testing.T) {
	client := &fakeGraphQLClient{pages: []string{
		`{"viewer":{"repositories":{"pageInfo":{"hasNextPage":true,"endCursor":"next"},"nodes":[{"languages":{"edges":[{"size":100,"node":{"name":"Go"}},{"size":100,"node":{"name":"Python"}}]}}]}}}`,
		`{"viewer":{"repositories":{"pageInfo":{"hasNextPage":false},"nodes":[{"languages":{"edges":[{"size":100,"node":{"name":"Go"}}]}}]}}}`,
	}}
	got, err := githubLanguageStats(client)
	if err != nil {
		t.Fatal(err)
	}
	want := []languageStat{{"Go", 200, "66.67%"}, {"Python", 100, "33.33%"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if !reflect.DeepEqual(client.cursors, []any{nil, "next"}) {
		t.Fatalf("unexpected cursors: %v", client.cursors)
	}
}

func TestGitHubLanguageStatsEmpty(t *testing.T) {
	client := &fakeGraphQLClient{pages: []string{`{"viewer":{"repositories":{"nodes":[]}}}`}}
	got, err := githubLanguageStats(client)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "[]" {
		t.Fatalf("got %s, want []", data)
	}
}

func TestGitHubLanguageStatsError(t *testing.T) {
	want := errors.New("API failed")
	_, err := githubLanguageStats(&fakeGraphQLClient{err: want})
	if !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

func TestGitHubLanguageStatsZero(t *testing.T) {
	client := &fakeGraphQLClient{pages: []string{`{"viewer":{"repositories":{"nodes":[{"languages":{"edges":[{"size":0,"node":{"name":"Go"}}]}}]}}}`}}
	got, err := githubLanguageStats(client)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Percentage != "0%" {
		t.Fatalf("unexpected stats: %v", got)
	}
}
