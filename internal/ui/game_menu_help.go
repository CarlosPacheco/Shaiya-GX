package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// The Help page of the options window (Game.exe FUN_004fa800 / 004fbbb0 /
// 004f7680 / 004f9cd0): six tabs, a row of section entries at y=134 and the
// picture of the chosen entry in a 200-row window, scrolled for the tall ones.
const (
	helpNavY          = 0x86
	helpPageX         = 0x13
	helpPageY         = 0x9c
	helpPageRows      = 200
	helpScrollStep    = 20
	helpScrollX       = 0x1ab
	helpScrollY       = 0x83
	helpScrollHeight  = 0xe8
	helpTabHitW       = 87
	helpTabHitH       = 22
	helpNavHitH       = 20
	helpNavCellH      = 32
	helpEtinNavCellH  = 30
	helpTabs          = 6
	helpPagePixelSpan = 512
)

var helpTabPos = [helpTabs][2]int{{81, 63}, {184, 63}, {288, 63}, {81, 91}, {184, 91}, {288, 91}}

type helpEntry struct {
	x, w  int
	pages []string // picture files, stacked
}

// helpSections lists every section's entries in the original order.
var helpSections = [helpTabs][]helpEntry{
	{{54, 30, []string{"1_1_move"}}, {95, 27, []string{"1_2_combat"}}, {134, 60, []string{"1_3_pickup"}}, {206, 27, []string{"1_4_chat_01", "1_4_chat_02"}},
		{248, 60, []string{"1_5_skill"}}, {323, 38, []string{"1_6_quest"}}, {373, 27, []string{"1_7_party"}}},
	{{95, 39, []string{"2_1_levelup"}}, {147, 63, []string{"2_2_useitem"}}, {222, 65, []string{"2_3_rapis"}}, {298, 52, []string{"2_4_trade"}}},
	{{27, 36, []string{"3_1_mouse"}}, {69, 36, []string{"3_2_keyboard"}}, {112, 37, []string{"3_3_hotkey"}}, {156, 35, []string{"3_4_minimap"}}, {200, 35, []string{"3_5_rader"}},
		{243, 38, []string{"3_6_stats"}}, {287, 37, []string{"3_7_inventory"}}, {329, 36, []string{"3_8_skill"}}, {373, 47, []string{"3_9_quest"}}},
	{{192, 62, []string{"4_1_bless"}}},
	{{192, 62, []string{"5_1_subskill"}}},
	{{192, 62, []string{"6_1_playmode"}}},
}

var helpNavSheets = [helpTabs]string{"1_basic_nav", "2_play_nav", "3_interface_nav", "4_etc_nav", "4_etc_nav", "4_etc_nav"}

type menuHelp struct {
	tab, entry int
	offset     int // scroll units of helpScrollStep rows
	tabs       [helpTabs]Button
	nav        map[string]*render.Texture
	pages      map[string]*render.Texture
	scroll     panelScroll
}

func newMenuHelp(ld *loader) *menuHelp {
	h := &menuHelp{nav: map[string]*render.Texture{}, pages: map[string]*render.Texture{}}
	sheet := ld.get("option_help_button.png")
	for i := range h.tabs {
		col, group := i%3, i/3
		b := Button{Tex: sheet, OffX: helpTabPos[i][0], OffY: helpTabPos[i][1], DrawW: 128, DrawH: 32, HitW: helpTabHitW, HitH: helpTabHitH}
		for state, row := range []int{0, 1, 2, 2, 3} {
			x, y := col*128, group*128+row*32
			b.UV[state] = UV{float32(x) / 512, float32(y) / 256, float32(x+128) / 512, float32(y+32) / 256}
		}
		h.tabs[i] = b
	}
	for _, name := range helpNavSheets {
		if h.nav[name] == nil {
			h.nav[name] = ld.get(name + ".png")
		}
	}
	for _, section := range helpSections {
		for _, e := range section {
			for _, p := range e.pages {
				h.pages[p] = ld.get(p + ".png")
			}
		}
	}
	h.scroll = newPanelScroll(ld.get("scrollbar.png"), helpScrollX, helpScrollY, helpScrollHeight)
	return h
}

func (h *menuHelp) pageRows() int {
	if h.tab == 0 {
		if h.entry == 3 {
			return 600
		}
		return 400
	}
	return 200
}

func (h *menuHelp) units() int { return (h.pageRows() + helpScrollStep - 1) / helpScrollStep }

func (h *menuHelp) reset() { h.tab, h.entry, h.offset = 0, 0, 0 }

func (h *menuHelp) select_(tab, entry int) {
	if tab == h.tab && entry == h.entry {
		return
	}
	h.tab, h.entry, h.offset = tab, entry, 0
}

func (h *menuHelp) update(in *platform.Input, x, y int) {
	for i := range h.tabs {
		b := &h.tabs[i]
		b.OriginX, b.OriginY = x, y
		if i == h.tab {
			continue
		}
		if b.Update(in) {
			h.select_(i, 0)
			return
		}
	}
	if in.Clicked {
		for i, e := range helpSections[h.tab] {
			if in.MouseX >= x+e.x && in.MouseX < x+e.x+e.w && in.MouseY >= y+helpNavY && in.MouseY < y+helpNavY+helpNavHitH {
				h.select_(h.tab, i)
				return
			}
		}
	}
	visible := helpPageRows / helpScrollStep
	if h.units() > visible {
		if in.MouseX >= x+helpPageX && in.MouseX < x+helpScrollX && in.MouseY >= y+helpNavY && in.MouseY < y+helpPageY+helpPageRows && in.Wheel != 0 {
			if in.Wheel < 0 {
				h.offset += 2
			} else {
				h.offset -= 2
			}
		}
		h.offset = h.scroll.Update(in, x, y, h.units(), visible, h.offset)
		h.offset = max(0, min(h.offset, h.units()-visible))
	} else {
		h.offset = 0
	}
}

func (h *menuHelp) draw(s *render.Sprites, x, y int) {
	for i := range h.tabs {
		b := &h.tabs[i]
		b.OriginX, b.OriginY = x, y
		if i == h.tab {
			// The current tab shows the highlighted cell (the grey row of the sheet
			// carries a wrong label in the third column).
			uv := b.UV[1]
			s.Draw(b.Tex, float32(b.X()), float32(b.Y()), float32(b.DrawW), float32(b.DrawH), uv.U0, uv.V0, uv.U1, uv.V1, white)
			continue
		}
		b.Draw(s)
	}
	sheet := h.nav[helpNavSheets[h.tab]]
	for i, e := range helpSections[h.tab] {
		cell, rows := helpNavCellH, 0
		if h.tab >= 3 {
			cell, rows = helpEtinNavCellH, h.tab-3
		} else {
			rows = i
		}
		v0, v1 := float32(rows*cell)/float32(sheet.H), float32((rows+1)*cell)/float32(sheet.H)
		s.Draw(sheet, float32(x+e.x), float32(y+helpNavY), 64, float32(cell), 0, v0, 1, v1, white)
	}
	pages := helpSections[h.tab][h.entry].pages
	rows := h.pageRows()
	if rows <= helpPageRows {
		if t := h.pages[pages[0]]; t != nil {
			s.Draw(t, float32(x+helpPageX), float32(y+helpPageY), float32(t.W), float32(t.H), 0, 0, 1, 1, white)
		}
		return
	}
	// The pictures are stacked in 512-row textures; 200 rows are shown.
	top, dy := h.offset*helpScrollStep, 0
	for dy < helpPageRows {
		index, row := top/helpPagePixelSpan, top%helpPagePixelSpan
		if index >= len(pages) {
			break
		}
		t := h.pages[pages[index]]
		if t == nil {
			break
		}
		span := min(helpPageRows-dy, helpPagePixelSpan-row)
		s.Draw(t, float32(x+helpPageX), float32(y+helpPageY+dy), float32(t.W), float32(span), 0, float32(row)/float32(t.H), 1, float32(row+span)/float32(t.H), white)
		top += span
		dy += span
	}
	h.scroll.Draw(s, x, y, h.units(), helpPageRows/helpScrollStep, h.offset)
}

// HelpSelect opens a tab and entry of the Help page (used by previews).
func (m *GameMenu) HelpSelect(tab, entry int) { m.help.select_(tab, entry) }
