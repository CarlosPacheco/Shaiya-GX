package ui

import (
	"fmt"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
)

// lookup resolves SysMsg ids; the app points it at the loaded sysmsg-uni.txt.
// Without it (offline tools, tests) the embedded GX rows answer.
var lookup func(id int) string

// SetText installs the SysMsg lookup used by every label in this package.
func SetText(f func(id int) string) { lookup = f }

// Tr returns the SysMsg text for id. UI text lives in the tables, not in code.
func Tr(id int) string {
	if lookup != nil {
		if s := lookup(id); s != "" {
			return s
		}
	}
	return text.Defaults[id]
}

// Trf is Tr with the <v>, <v2>... placeholders replaced by vals in order.
func Trf(id int, vals ...any) string {
	out := Tr(id)
	for i, v := range vals {
		token := "<v>"
		if i > 0 {
			token = fmt.Sprintf("<v%d>", i+1)
		}
		out = strings.ReplaceAll(out, token, fmt.Sprint(v))
	}
	return out
}
