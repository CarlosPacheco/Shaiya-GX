package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// The blacksmith is the repair window of Game.exe (0x4e4d30, 255x417) turned
// into the forge menu, with the Link/Extract (0x494d00), Enchant (0x4e0910)
// and Recreation (0x4ad1d0) pages of 512x446 behind it. Repair itself is
// not offered: its first button opens the recreation page.
const (
	BlacksmithWidth, BlacksmithHeight = 512, 446
	ForgeMenuWidth, ForgeMenuHeight   = 255, 417
)
const (
	ForgeLapis = iota
	ForgeEnchant
	ForgeRemake
	ForgeMenu
)

type Blacksmith struct {
	Position                               WindowPosition
	Mode, GemSlot                          int
	frame, lapis, enchantFrame, menuFrame  *render.Texture
	close, link, extract, enchant, compose Button
	menu                                   [4]Button // recreation, cancel, link/extract, enchant
	back, menuClose                        Button    // cancel: return to the menu; menu close
	welcome                                panelScroll
	menuWelcome                            panelScroll
	textOffset, textLines                  int
}

// Size is the window size of the current page.
func (v *Blacksmith) Size() (int, int) {
	if v.Mode == ForgeMenu {
		return ForgeMenuWidth, ForgeMenuHeight
	}
	return BlacksmithWidth, BlacksmithHeight
}

// Native NPC sheets store disabled before pressed, unlike generic menu sheets.
var npcVerticalUV = [5]UV{{0, 0, 1, .25}, {0, .25, 1, .5}, {0, .75, 1, 1}, {0, .75, 1, 1}, {0, .5, 1, .75}}

func npcHorizontalUV(row int) (uv [5]UV) {
	for i, col := range []int{0, 1, 3, 3, 2} {
		uv[i] = UV{float32(col) / 4, float32(row) / 2, float32(col+1) / 4, float32(row+1) / 2}
	}
	return
}
func NewBlacksmith(t Textures) (*Blacksmith, error) {
	ld := &loader{t: t}
	v := &Blacksmith{Mode: ForgeMenu, frame: ld.get("Remake3.png"), lapis: ld.get("rapis.png"), enchantFrame: ld.get("rapisian.png"), menuFrame: ld.get("repair.png")}
	v.welcome = newPanelScroll(ld.get("scrollbar.png"), 228, 82, 167)
	v.menuWelcome = newPanelScroll(ld.get("scrollbar.png"), 228, 112, 167)
	v.close = nativeClose(ld.get("talk1_close_button.png"), 483, 3)
	sheet := ld.get("rapis_button2.png")
	v.link = Button{Tex: sheet, OffX: 292, OffY: 374, DrawW: 128, DrawH: 32, HitW: 124, HitH: 27}
	v.extract = Button{Tex: sheet, OffX: 427, OffY: 374, DrawW: 64, DrawH: 32, HitW: 64, HitH: 27}
	for i, row := range []int{0, 1, 3, 3, 2} {
		v.link.UV[i] = UV{0, float32(row) / 4, .5, float32(row+1) / 4}
		v.extract.UV[i] = UV{.5, float32(row) / 4, .75, float32(row+1) / 4}
	}
	enchantSheet := ld.get("repair_button4.png")
	v.enchant = Button{Tex: enchantSheet, OffX: 348, OffY: 374, DrawW: 128, DrawH: 32, HitW: 65, HitH: 27, UV: npcVerticalUV}
	remodel := ld.get("Btn_Remodeling.png")
	v.compose = Button{Tex: remodel, OffX: 351, OffY: 390, DrawW: 128, DrawH: 32, HitW: 65, HitH: 27, UV: npcVerticalUV}
	// repair_button1 holds Repair All and Cancel side by side; only Cancel is used.
	cancel := ld.get("repair_button1.png")
	cancelButton := func(x, y int) Button {
		b := Button{Tex: cancel, OffX: x, OffY: y, DrawW: 128, DrawH: 32, HitW: 66, HitH: 26}
		for i, row := range []int{0, 1, 3, 3, 2} {
			b.UV[i] = UV{.5, float32(row) / 4, 1, float32(row+1) / 4}
		}
		return b
	}
	v.back = cancelButton(97, 302)
	v.menu[0] = Button{Tex: remodel, OffX: 32, OffY: 295, DrawW: 128, DrawH: 32, HitW: 96, HitH: 26, UV: npcVerticalUV}
	v.menu[1] = cancelButton(148, 295)
	v.menu[1].HitW = 66
	v.menu[2] = Button{Tex: ld.get("repair_button3.png"), OffX: 31, OffY: 357, DrawW: 128, DrawH: 32, HitW: 96, HitH: 27, UV: npcVerticalUV}
	v.menu[3] = Button{Tex: enchantSheet, OffX: 147, OffY: 357, DrawW: 128, DrawH: 32, HitW: 65, HitH: 27, UV: npcVerticalUV}
	v.menuClose = nativeClose(ld.get("talk1_close_button.png"), 228, 5)
	return v, ld.err
}
func (v *Blacksmith) Origin(w, h int) (int, int) {
	width, height := v.Size()
	return v.Position.Origin(w, h, width, height, min(32, max(0, w-width)), min(82, max(0, h-height)))
}
func BlacksmithSlot(slot int) (int, int) { return (&Blacksmith{}).Slot(slot) }
func (v *Blacksmith) Slot(slot int) (int, int) {
	if v.Mode == ForgeMenu {
		return -100, -100
	}
	if v.Mode == ForgeLapis {
		switch slot {
		case 0:
			return 367, 93
		case 1:
			return 270 + min(5, max(0, v.GemSlot))*39, 206
		case 3:
			return 367, 288
		}
		return -100, -100
	}
	if slot == 0 {
		if v.Mode == ForgeRemake {
			return 370, 276
		}
		return 367, 288
	}
	if v.Mode == ForgeRemake {
		if slot == 2 {
			return 371, 105
		}
		return -100, -100
	}
	if slot == 1 {
		return 309, 131
	}
	if slot == 2 {
		return 427, 131
	}
	return -100, -100
}
func BlacksmithHit(x, y int) int { return (&Blacksmith{}).Hit(x, y) }
func (v *Blacksmith) Hit(x, y int) int {
	for slot := 0; slot < 4; slot++ {
		sx, sy := v.Slot(slot)
		if sx >= 0 && x >= sx && x < sx+32 && y >= sy && y < sy+32 {
			return slot
		}
	}
	return -1
}
func (v *Blacksmith) place(x, y int, busy, ready, extractReady, enchantReady, composeReady bool) {
	for _, b := range []*Button{&v.close, &v.link, &v.extract, &v.enchant, &v.compose, &v.back, &v.menuClose} {
		b.OriginX, b.OriginY = x, y
	}
	for i := range v.menu {
		v.menu[i].OriginX, v.menu[i].OriginY, v.menu[i].Disabled = x, y, busy && i != 1
	}
	v.link.Disabled = busy || !ready
	v.extract.Disabled = busy || !extractReady
	v.enchant.Disabled = busy || !enchantReady
	v.compose.Disabled = busy || !composeReady
	v.back.Disabled = busy
}

// Reset restores the main menu page and clears scroll and socket state.
func (v *Blacksmith) Reset() {
	v.Mode, v.GemSlot, v.textOffset = ForgeMenu, 0, 0
}

// Update returns the close, link, extract, enchant and recreate actions.
// Menu buttons and Cancel switch pages by themselves.
func (v *Blacksmith) Update(in *platform.Input, w, h int, busy, ready, extractReady, enchantReady, composeReady bool) (bool, bool, bool, bool, bool) {
	x, y := v.Origin(w, h)
	v.place(x, y, busy, ready, extractReady, enchantReady, composeReady)
	v.close.Disabled, v.menuClose.Disabled = busy, busy
	if v.Mode == ForgeMenu {
		for i := range v.menu {
			if !v.menu[i].Update(in) {
				continue
			}
			switch i {
			case 0:
				v.Mode = ForgeRemake
			case 1:
				return true, false, false, false, false
			case 2:
				v.Mode = ForgeLapis
			case 3:
				v.Mode = ForgeEnchant
			}
			v.textOffset = 0
			return false, false, false, false, false
		}
		v.textOffset = v.menuWelcome.Update(in, x, y, v.textLines, 11, v.textOffset)
		return v.menuClose.Update(in), false, false, false, false
	}
	if v.back.Update(in) {
		v.Reset()
		return false, false, false, false, false
	}
	v.textOffset = v.welcome.Update(in, x, y, v.textLines, 10, v.textOffset)
	return v.close.Update(in), v.Mode == ForgeLapis && v.link.Update(in), v.Mode == ForgeLapis && v.extract.Update(in), v.Mode == ForgeEnchant && v.enchant.Update(in), v.Mode == ForgeRemake && v.compose.Update(in)
}

func (v *Blacksmith) Draw(s *render.Sprites, f *Font, w, h int, name, welcome string, busy, ready, extractReady, enchantReady, composeReady bool) {
	x, y := v.Origin(w, h)
	v.place(x, y, busy, ready, extractReady, enchantReady, composeReady)
	if v.Mode == ForgeMenu {
		s.Draw(v.menuFrame, float32(x), float32(y), ForgeMenuWidth, ForgeMenuHeight, 0, 0, ForgeMenuWidth/256.0, ForgeMenuHeight/512.0, white)
		f.Draw(s, name, x+0x41+(0xff-0x41-f.Width(name))/2, y+0x16, white, 0)
		lines := wrapLines(f, welcome, 205)
		v.textLines = len(lines)
		v.textOffset = min(v.textOffset, max(0, len(lines)-11))
		for i, line := range lines[v.textOffset:min(len(lines), v.textOffset+11)] {
			f.Draw(s, line, x+20, y+118+i*15, white, 205)
		}
		v.menuWelcome.Draw(s, x, y, len(lines), 11, v.textOffset)
		for i := range v.menu {
			v.menu[i].Draw(s)
		}
		if busy {
			f.Draw(s, Tr(12259), x+20, y+270, white, 208)
		}
		v.menuClose.Draw(s)
		return
	}
	frame := v.frame
	text := Tr(12364)
	switch v.Mode {
	case ForgeLapis:
		frame = v.lapis
		text = Tr(12365)
	case ForgeEnchant:
		frame = v.enchantFrame
		text = Tr(12366)
	}
	s.Draw(frame, float32(x), float32(y), 512, 446, 0, 0, 1, 446.0/512, white)
	lines := wrapLines(f, text, 193)
	v.textLines = len(lines)
	v.textOffset = min(v.textOffset, max(0, len(lines)-10))
	for i, line := range lines[v.textOffset:min(len(lines), v.textOffset+10)] {
		f.Draw(s, line, x+26, y+92+i*15, RGBColor(0xc8c8c8), 193)
	}
	v.welcome.Draw(s, x, y, len(lines), 10, v.textOffset)
	v.back.Draw(s)
	switch v.Mode {
	case ForgeLapis:
		v.link.Draw(s)
		v.extract.Draw(s)
	case ForgeEnchant:
		v.enchant.Draw(s)
	case ForgeRemake:
		v.compose.Draw(s)
	}
	if busy {
		f.Draw(s, Tr(12259), x+26, y+265, white, 208)
	}
	v.close.Draw(s)
}
