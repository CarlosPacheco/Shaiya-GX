// Package text reads the client's message tables.
package text

import (
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// SysMsg is sysmsg-uni.txt: UTF-16LE lines `id<TAB><TAB>"text"`, read by
// the client's message lookup (FUN_0041d4a0).
type SysMsg map[int]string

func decodeSysMsgUTF16(b []byte) string {
	if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE {
		b = b[2:]
	}
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = uint16(b[2*i]) | uint16(b[2*i+1])<<8
	}
	return string(utf16.Decode(u))
}

func ParseSysMsg(b []byte) SysMsg {
	m := parseMessageTable(decodeSysMsgUTF16(b), false)
	for id, v := range Defaults {
		if _, ok := m[id]; !ok {
			m[id] = v
		}
	}
	return m
}

func parseMessageTable(content string, requireQuotes bool) SysMsg {
	m := SysMsg{}
	for row := range strings.Lines(content) {
		line := strings.TrimSpace(row)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		i := strings.IndexAny(line, " \t")
		if i < 0 {
			continue
		}
		id, err := strconv.Atoi(line[:i])
		if err != nil {
			continue
		}
		v := strings.TrimSpace(line[i:])
		if requireQuotes && (len(v) < 2 || v[0] != '"' || v[len(v)-1] != '"') {
			continue
		}
		v = strings.TrimPrefix(v, "\"")
		v = strings.TrimSuffix(v, "\"")
		m[id] = strings.TrimSpace(strings.ReplaceAll(v, `\n`, "\n"))
	}
	return m
}

// Get returns the message or "" when missing.
func (m SysMsg) Get(id int) string { return m[id] }

// Preserve substitutions, line breaks, and formatting when replacing a message.
var messagePlaceholders = regexp.MustCompile(`<[A-Za-z][A-Za-z0-9]*>`)
var messageFormatting = regexp.MustCompile(`\n|\{/?[A-Za-z][A-Za-z0-9]*\}|%%`)

// WithTranslation accepts UTF-8 or BOM-marked UTF-16LE. Invalid, blank and
// absent rows leave the matching English message available.
func (m SysMsg) WithTranslation(data []byte) SysMsg {
	content := string(data)
	if strings.HasPrefix(content, "\xff\xfe") {
		content = decodeSysMsgUTF16(data)
	} else {
		content = strings.TrimPrefix(content, "\ufeff")
	}
	if !utf8.ValidString(content) {
		return m
	}
	out := make(SysMsg, len(m))
	maps.Copy(out, m)
	for id, translated := range parseMessageTable(content, true) {
		base := m.Lookup(id)
		if base == "" || translated == "" || strings.ContainsAny(translated, "\x00\ufffd") {
			continue
		}
		sourceTokens := messagePlaceholders.FindAllString(base, -1)
		targetTokens := messagePlaceholders.FindAllString(translated, -1)
		slices.Sort(sourceTokens)
		slices.Sort(targetTokens)
		if slices.Equal(sourceTokens, targetTokens) &&
			slices.Equal(messageFormatting.FindAllString(base, -1), messageFormatting.FindAllString(translated, -1)) {
			out[id] = translated
		}
	}
	return out
}
