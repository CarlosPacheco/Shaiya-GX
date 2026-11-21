package text

import (
	"bytes"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// WordFilter masks literal rules from a local SysMsg-style table.
type WordFilter struct {
	pattern *regexp.Regexp
	cache   map[string]string
}

// ParseWordFilter accepts the native UTF-16LE table and UTF-8 user edits.
// An empty table returns nil so callers can skip filtering entirely.
func ParseWordFilter(data []byte) *WordFilter {
	var table SysMsg
	if bytes.Contains(data, []byte{0}) || !utf8.Valid(data) {
		table = ParseSysMsg(data)
	} else {
		table = parseMessageTable(strings.TrimPrefix(string(data), "\ufeff"))
	}
	var terms []string
	for _, value := range table {
		if value != "" {
			terms = append(terms, regexp.QuoteMeta(value))
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
