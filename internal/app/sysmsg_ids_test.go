package app

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
)

// Every GX id used by the code (12001+) must exist in the embedded rows, and
// every embedded GX row must be used, so the tables and the code never drift.
func TestGXSysMsgIDsMatchTheEmbeddedRows(t *testing.T) {
	use := regexp.MustCompile(`(?:\.tr|Tr|Trf|Format|systemMessage|questMessageID|backToLoginID|personalMarketNotice|localizedNotice)\((?:[^,()]*, )?(12[0-9]{3})\b|\b(12[0-9]{3})\b`)
	used := map[int]bool{}
	for _, dir := range []string{".", "../ui"} {
		files, _ := filepath.Glob(filepath.Join(dir, "*.go"))
		for _, f := range files {
			if filepath.Base(f) == "sysmsg_ids_test.go" || len(f) > 8 && f[len(f)-8:] == "_test.go" {
				continue
			}
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			for _, m := range use.FindAllStringSubmatch(string(b), -1) {
				for _, g := range m[1:] {
					if g == "" {
						continue
					}
					id, _ := strconv.Atoi(g)
					if id >= 12001 && id < 13000 {
						used[id] = true
					}
				}
			}
		}
	}
	for id := range used {
		if text.Defaults[id] == "" {
			t.Errorf("SysMsg id %d is used in code but has no embedded row", id)
		}
	}
	for id := range text.Defaults {
		if id >= 12001 && !used[id] {
			t.Errorf("embedded GX row %d (%q) is not used by any code", id, text.Defaults[id])
		}
	}
}

func TestSysMsgFileWinsOverEmbeddedDefaults(t *testing.T) {
	m := text.SysMsg{12001: "Translated"}
	if m.Lookup(12001) != "Translated" {
		t.Fatal("a table row must win over the embedded default")
	}
	if (text.SysMsg{}).Lookup(12001) == "" {
		t.Fatal("a missing id must fall back to the embedded default")
	}
}
