package text

import (
	_ "embed"
)

// gxRows are the GX-only SysMsg rows (ids from 12001) and the default text for
// native ids that older sysmsg-uni.txt files may lack. These rows can also be
// installed in the English table by tools/patch_sysmsg.py.
//
//go:embed sysmsg-gx.txt
var gxRows []byte

// Defaults holds the embedded rows, by id.
var Defaults = parseMessageTable(string(gxRows), false)

// Lookup is Get with the embedded defaults as fallback.
func (m SysMsg) Lookup(id int) string {
	if s := m[id]; s != "" {
		return s
	}
	return Defaults[id]
}
