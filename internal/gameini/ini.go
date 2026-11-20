// Package gameini reads and writes the server's game configuration dialect.
package gameini

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

type File map[string]map[string]string

var blockComment = regexp.MustCompile(`(?s)/\*.*?\*/|"(?:[^"\\]|\\.)*"`)

func Read(path string) (File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(raw)
}

func Parse(raw []byte) (File, error) {
	text := blockComment.ReplaceAllStringFunc(strings.TrimPrefix(string(raw), "\ufeff"), func(s string) string {
		if s[0] == '"' {
			return s
		}
		return strings.Repeat("\n", strings.Count(s, "\n"))
	})
	out := File{}
	var section map[string]string
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			name, _, found := strings.Cut(line[1:], "]")
			if !found || strings.TrimSpace(name) == "" {
				return nil, fmt.Errorf("line %d has an invalid section", i+1)
			}
			name = strings.ToUpper(strings.TrimSpace(name))
			if out[name] == nil {
				out[name] = map[string]string{}
			}
			section = out[name]
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		k = strings.TrimSpace(k)
		if !ok || section == nil || k == "" || strings.HasPrefix(k, ";") || strings.HasPrefix(k, "//") || strings.HasPrefix(k, "#") {
			continue
		}
		v = strings.TrimSpace(v)
		if strings.HasPrefix(v, "\"") {
			q, err := strconv.QuotedPrefix(v)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", i+1, err)
			}
			v, err = strconv.Unquote(q)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", i+1, err)
			}
		} else {
			v, _, _ = strings.Cut(v, "//")
			v, _, _ = strings.Cut(v, ";")
			v = strings.TrimSpace(v)
		}
		section[strings.ToUpper(k)] = v
	}
	return out, nil
}

// Entries returns numeric sections sorted by index.
func (f File) Entries(prefix string, bits int) ([]Entry, error) {
	var entries []Entry
	prefix = strings.ToUpper(prefix)
	for name, fields := range f {
		value, ok := strings.CutPrefix(name, prefix)
		if !ok {
			continue
		}
		id, err := strconv.ParseUint(value, 10, bits)
		if err != nil {
			return nil, fmt.Errorf("invalid section [%s]", name)
		}
		entries = append(entries, Entry{id, fields})
	}
	slices.SortFunc(entries, func(a, b Entry) int {
		if a.Index < b.Index {
			return -1
		}
		if a.Index > b.Index {
			return 1
		}
		return 0
	})
	for i := 1; i < len(entries); i++ {
		if entries[i-1].Index == entries[i].Index {
			return nil, fmt.Errorf("duplicate %s index %d", prefix, entries[i].Index)
		}
	}
	return entries, nil
}

type Entry struct {
	Index  uint64
	Fields map[string]string
}

// Fields splits the original coordinate and schedule list syntax.
func Fields(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == '(' || r == ')' || r == ',' || unicode.IsSpace(r) })
}

type Record struct {
	Fields map[string]string
	Err    error
}

func (r *Record) Text(key string) string { return r.Fields[strings.ToUpper(key)] }

func (r *Record) Uint(key string, bits int) uint64 {
	v, err := strconv.ParseUint(r.Text(key), 10, bits)
	if err != nil && r.Err == nil {
		r.Err = fmt.Errorf("%s: %w", key, err)
	}
	return v
}

func (r *Record) Percent(key string, parse func(string) (uint64, error)) uint64 {
	v, err := parse(r.Text(key))
	if err != nil && r.Err == nil {
		r.Err = fmt.Errorf("%s: %w", key, err)
	}
	return v
}
