package text

import (
	_ "embed"
)

// gxRows are the GX-only SysMsg rows (ids from 12001) and the default text for
// native ids that older sysmsg-uni.txt files may lack. sysmsg-uni.txt always
// wins; a translation replaces these by providing the same ids. The same rows
// are written into the data folder by tools/patch_sysmsg.py so they can be
// edited per language.
//
//go:embed sysmsg-gx.txt
var gxRows []byte

// Defaults holds the embedded rows, by id.
var Defaults = parseMessageTable(string(gxRows))

// Lookup is Get with the embedded defaults as fallback.
func (m SysMsg) Lookup(id int) string {
	if s := m[id]; s != "" {
		return s
	}
	return Defaults[id]
}
