package text

import (
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Spelunkern/Shaiya-GX/internal/data/ini"
)

// WordFilter masks literal rules from Filter.INI.
type WordFilter struct {
	pattern *regexp.Regexp
	cache   map[string]string
}

// ParseWordFilter reads Filter.INI: every Word= line is a literal, case-
// insensitive rule (UTF-8 or UTF-16LE). No rules returns nil so callers can skip
// filtering entirely.
//
//	[Filter]
//	Word=example
//	Word=another phrase
func ParseWordFilter(data []byte) *WordFilter {
	var terms []string
	for _, section := range ini.Parse(data) {
		for _, value := range section.All("word") {
			if value != "" {
				terms = append(terms, regexp.QuoteMeta(value))
			}
		}
	}
	if len(terms) == 0 {
		return nil
	}
	// Longest consumes the entire phrase when rules share a prefix.
	slices.Sort(terms)
	pattern, err := regexp.Compile("(?i:" + strings.Join(slices.Compact(terms), "|") + ")")
	if err != nil {
		return nil
	}
	pattern.Longest()
	return &WordFilter{pattern: pattern, cache: make(map[string]string)}
}

// Censor caches rendered bodies so history refreshes and speech bubbles reuse
// their results. Call from the UI thread.
func (f *WordFilter) Censor(body string) string {
	if masked, ok := f.cache[body]; ok {
		return masked
	}
	masked := f.pattern.ReplaceAllStringFunc(body, func(match string) string {
		return strings.Repeat("*", utf8.RuneCountInString(match))
	})
	if len(f.cache) >= 512 {
		clear(f.cache)
	}
	f.cache[body] = masked
	return masked
}
