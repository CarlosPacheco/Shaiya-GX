package config

import (
	"os"
	"regexp"
	"strings"
)

type GameINI map[string]map[string]string

var blockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)

// ReadGameINI parses the original server's ini dialect: [SECTION], key = value,
// with ;, // and /* */ comments. Section and key names are upper-cased.
func ReadGameINI(path string) (GameINI, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := blockComment.ReplaceAllString(string(raw), "")
	out := GameINI{}
	section := ""
	for _, line := range strings.Split(text, "\n") {
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		if i := strings.Index(line, ";"); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		switch {
		case line == "":
		case strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]"):
			section = strings.ToUpper(strings.TrimSpace(line[1 : len(line)-1]))
			if out[section] == nil {
				out[section] = map[string]string{}
			}
		default:
			k, v, ok := strings.Cut(line, "=")
			if ok && section != "" {
				out[section][strings.ToUpper(strings.TrimSpace(k))] = strings.TrimSpace(v)
			}
		}
	}
	return out, nil
}
