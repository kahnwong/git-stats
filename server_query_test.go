package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLanguageQueryParameters(t *testing.T) {
	dir := t.TempDir()
	data := `{"languages":[{"language":"Go","bytes":600},{"language":"Python","bytes":300},{"language":"Shell","bytes":100}]}`
	if err := os.WriteFile(filepath.Join(dir, "github.json"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	handler, err := newStatsHandler(dir, func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		query  string
		status int
		want   []string
		absent []string
	}{
		{"", 200, []string{"Go 60%", "Python 30%", "Shell 10%"}, nil},
		{"?limit=2", 200, []string{"Go 66.67%", "Python 33.33%"}, []string{"Shell "}},
		{"?limit=1&exclude=go", 200, []string{"Python 100%"}, []string{"Go ", "Shell "}},
		{"?exclude=%20GO%20,python", 200, []string{"Shell 100%"}, []string{"Go ", "Python "}},
		{"?exclude=Go&exclude=Python,Shell", 200, []string{"No language data available."}, []string{"Go ", "Python ", "Shell "}},
		{"?limit=99&exclude=unknown,,", 200, []string{"Go 60%", "Python 30%", "Shell 10%"}, nil},
		{"?limit=0", 400, nil, nil},
		{"?limit=-1", 400, nil, nil},
		{"?limit=abc", 400, nil, nil},
		{"?limit=1.5", 400, nil, nil},
		{"?limit=", 400, nil, nil},
		{"?limit=1&limit=2", 400, nil, nil},
	} {
		t.Run(tc.query, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", "/"+tc.query, nil))
			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d", w.Code, tc.status)
			}
			body := w.Body.String()
			for _, want := range tc.want {
				if !strings.Contains(body, want) {
					t.Errorf("missing %q", want)
				}
			}
			for _, absent := range tc.absent {
				if strings.Contains(body, absent) {
					t.Errorf("unexpected %q", absent)
				}
			}
			if w.Code == 200 {
				assertValidSVG(t, body)
			}
		})
	}
}
