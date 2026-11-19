// Package gm defines GX's small, shared GM command vocabulary.
package gm

import "strings"

const TurboMultiplier = 12

// Parse recognizes commands independently of chat channel or account privileges.
// Authority is checked exclusively by the world server.
func Parse(text string) (string, []string, bool) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", nil, false
	}
	command := strings.ToLower(fields[0])
	switch command {
	case "/set", "/iclear", "/turbo", "/getitem", "/mmake", "/nmake", "/cmove", "/cure", "/amove", "/asummon", "/weather", "/notice":
		return command, fields[1:], true
	}
	return "", nil, false
}

// Retired reserves removed commands so old clients cannot broadcast them as chat.
func Retired(text string) bool {
	fields := strings.Fields(text)
	return len(fields) > 0 && (strings.EqualFold(fields[0], "/char") || strings.EqualFold(fields[0], "/attack"))
}
