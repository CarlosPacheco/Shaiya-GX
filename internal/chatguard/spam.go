package chatguard

import (
	"strings"
	"unicode"
)

func SpamKey(text string) string {
	letters := []rune(inspectionText(text))
	var out strings.Builder
	var previous rune
	run := 0
	for i, r := range letters {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsMark(r) {
			// Keep separators inside numbers so prices like 1.5 and 15 differ.
			if i > 0 && i+1 < len(letters) && unicode.IsDigit(letters[i-1]) && unicode.IsDigit(letters[i+1]) {
				out.WriteRune(r)
			}
			continue
		}
		if r == previous && !unicode.IsDigit(r) {
			run++
			if run > 1 {
				continue
			}
		} else {
			previous, run = r, 0
		}
		out.WriteRune(r)
	}
	return out.String()
}

// Similar accepts one textual edit on longer messages. An edited digit is
// significant even when the rest of a trade offer stays identical.
func Similar(a, b string) bool {
	if a == b {
		return true
	}
	x, y := []rune(a), []rune(b)
	if len(x) > len(y) {
		x, y = y, x
	}
	if len(x) < 16 || len(y)-len(x) > 1 {
		return false
	}
	i := 0
	for i < len(x) && x[i] == y[i] {
		i++
	}
	if i == len(y) {
		return true
	}
	if numericEdit(y, i) || len(x) == len(y) && numericEdit(x, i) {
		return false
	}
	for j := i + 1; j < len(y); j++ {
		if x[j-(len(y)-len(x))] != y[j] {
			return false
		}
	}
	return true
}

func numericEdit(text []rune, i int) bool {
	return unicode.IsDigit(text[i]) || i > 0 && i+1 < len(text) && !unicode.IsLetter(text[i]) && unicode.IsDigit(text[i-1]) && unicode.IsDigit(text[i+1])
}
