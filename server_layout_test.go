package main

import (
	"strings"
	"testing"
)

func TestSVGReferenceLayout(t *testing.T) {
	stats := []languageStat{
		{"Go", 5722, "57.22%"}, {"Python", 1585, "15.85%"},
		{"HCL", 1193, "11.93%"}, {"Shell", 1077, "10.77%"},
		{"Nix", 242, "2.42%"}, {"Rust", 181, "1.81%"},
	}
	svg := string(renderLanguagesSVG(stats, nil))
	for _, want := range []string{
		`width="300" height="165" viewBox="0 0 300 165"`,
		`font: 600 18px 'Segoe UI', Ubuntu, Sans-Serif`,
		`@supports(-moz-appearance: auto) { .header { font-size: 15.5px; } }`,
		`font: 400 11px "Segoe UI", Ubuntu, Sans-Serif`,
		`<text x="25" y="35" class="header">`,
		`<rect x="25" y="55" width="250" height="8" rx="5"/>`,
		`<circle cx="30" cy="86" r="5"`,
		`<circle cx="180" cy="86" r="5"`,
		`<text x="40" y="90" class="lang-name">Go 57.22%</text>`,
		`<text x="40" y="115" class="lang-name">Python 15.85%</text>`,
		`<text x="40" y="140" class="lang-name">HCL 11.93%</text>`,
		`<text x="190" y="90" class="lang-name">Shell 10.77%</text>`,
		`<text x="190" y="115" class="lang-name">Nix 2.42%</text>`,
		`<text x="190" y="140" class="lang-name">Rust 1.81%</text>`,
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("missing reference layout: %s", want)
		}
	}
	long := string(renderLanguagesSVG([]languageStat{{"A long language name", 1, "100%"}}, nil))
	if strings.Contains(long, "textLength") || strings.Contains(long, "lengthAdjust") {
		t.Fatal("language labels should retain natural font spacing")
	}
	assertValidSVG(t, svg)
}
