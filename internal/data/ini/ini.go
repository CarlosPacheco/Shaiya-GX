// Package ini reads the small INI files of the GX data folder: [sections] in
// file order, key=value lines, ; and # comments, UTF-8 or UTF-16LE.
package ini

import (
	"bytes"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

type Pair struct{ Key, Value string }

type Section struct {
	Name  string
	Pairs []Pair
}

// Get returns the first value of key (case-insensitive).
func (s Section) Get(key string) (string, bool) {
	for _, p := range s.Pairs {
		if strings.EqualFold(p.Key, key) {
			return p.Value, true
		}
	}
	return "", false
}

// All returns every value of key in order.
func (s Section) All(key string) []string {
	var values []string
	for _, p := range s.Pairs {
		if strings.EqualFold(p.Key, key) {
			values = append(values, p.Value)
		}
	}
	return values
}

// Decode turns UTF-16LE (BOM or NUL bytes) and UTF-8 (optional BOM) into text.
func Decode(raw []byte) string {
	if bytes.HasPrefix(raw, []byte{0xff, 0xfe}) || len(raw) > 1 && raw[1] == 0 && !utf8.Valid(raw) {
		raw = bytes.TrimPrefix(raw, []byte{0xff, 0xfe})
		units := make([]uint16, 0, len(raw)/2)
		for i := 0; i+1 < len(raw); i += 2 {
			units = append(units, uint16(raw[i])|uint16(raw[i+1])<<8)
		}
		return string(utf16.Decode(units))
	}
	return strings.TrimPrefix(string(raw), "\ufeff")
}

// Parse lists the sections in file order. Keys before the first section belong
// to a section with an empty name.
func Parse(raw []byte) []Section {
	var sections []Section
	current := -1
	for _, line := range strings.Split(Decode(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == ';' || line[0] == '#' {
			continue
		}
		if line[0] == '[' && strings.HasSuffix(line, "]") {
			sections = append(sections, Section{Name: strings.TrimSpace(line[1 : len(line)-1])})
			current = len(sections) - 1
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if current < 0 {
			sections, current = append(sections, Section{}), 0
		}
		sections[current].Pairs = append(sections[current].Pairs, Pair{strings.TrimSpace(key), strings.TrimSpace(value)})
	}
	return sections
}
