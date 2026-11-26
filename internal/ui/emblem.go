package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/emblem"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// The cape customization window of the emblem NPC (Game.exe 0x480a00 builds
// it, 0x47f400 handles input and 0x480620 draws it). Every coordinate below is
// the original one relative to the window, whose art is emblem.png (652x512).
const (
	EmblemWidth, EmblemHeight = 652, 512
	// EmblemShapes is the number of selectors: front, back and emblem.
	EmblemShapes = 3
)

var (
	// Shape frames: preview square origin and the clickable area that makes the
	// selector current (0x47f400 tests x 0x118..0x16f and these y ranges).
	emblemPreviewY = [EmblemShapes]int{0x57, 0xe4, 0x171}
	emblemFrameY   = [EmblemShapes][2]int{{0x4c, 0xb2}, {0xd9, 0x13f}, {0x165, 0x1cc}}
	emblemMarkY    = [EmblemShapes]int{0x52, 0xdf, 0x16c}
	emblemArrowY   = [EmblemShapes]int{0x9b, 0x128, 0x1b5}
)

// EmblemView is what the window shows.
type EmblemView struct {
	Selection [EmblemShapes]int   // -1 none, else zero-based table index
	Colors    [EmblemShapes]uint8 // palette index per shape
	Textures  [EmblemShapes]*render.Texture
	Counts    [EmblemShapes]int // table sizes
	Active    int               // 0 none, else the shape whose colour the grid sets
	HasItem   bool
	Busy      bool
	Cost      uint32
	Gold      string // formatted price with its unit
}

// EmblemAction reports one frame of input.
type EmblemAction struct {
	Close, Apply, Remove bool
	Arrow                int // -1, else 2*shape (previous) or 2*shape+1 (next)
	Frame                int // 0 none, else 1..3: the shape frame clicked
	Color                int // -1, else the palette index clicked
}

type EmblemWindow struct {
	Position            WindowPosition
	frame, mark         *render.Texture
	close, apply, erase Button
	arrows              [EmblemShapes * 2]Button
}

func NewEmblemWindow(t Textures) (*EmblemWindow, error) {
	ld := &loader{t: t}
	v := &EmblemWindow{frame: ld.get("emblem.png"), mark: ld.get("emblem_select_mark.png")}
	v.close = nativeClose(ld.get("talk1_close_button.png"), EmblemWidth-33, 3)
	sheet := ld.get("emblem_button.png")
	button := func(x, hitW int, col float32) Button {
		b := Button{Tex: sheet, OffX: x, OffY: 0x1c7, DrawW: 128, DrawH: 32, HitW: hitW, HitH: 27}
		for i, row := range []int{0, 1, 2, 2, 3} {
			b.UV[i] = UV{col, float32(row) / 4, col + .5, float32(row+1) / 4}
		}
		return b
	}
	v.apply, v.erase = button(0x1a, 0x54, 0), button(0x94, 0x49, .5)
	arrow := ld.get("emblem_slect_arrow.png")
	for s := range EmblemShapes {
		for side := range 2 {
			b := Button{Tex: arrow, OffX: 0x123 + side*0x27, OffY: emblemArrowY[s], DrawW: 32, DrawH: 32, HitW: 0x17, HitH: 0x18}
			for i, col := range []int{0, 1, 2, 2, 3} {
				b.UV[i] = UV{float32(col) / 4, float32(side) / 2, float32(col+1) / 4, float32(side+1) / 2}
			}
			v.arrows[s*2+side] = b
		}
	}
	return v, ld.err
}

func (v *EmblemWindow) Origin(w, h int) (int, int) {
	return v.Position.Origin(w, h, EmblemWidth, EmblemHeight, min(32, max(0, w-EmblemWidth)), min(82, max(0, h-EmblemHeight)))
}

// ItemSlot is the 80x80 drop area for the cape (0x47f400: 0x218..0x267, 0x3a..0x87).
func EmblemItemSlot() (x, y, w, h int) { return 0x218, 0x3a, 0x50, 0x4e }

// PreviewArea is the dark panel that holds the character preview.
func EmblemPreviewArea() (x, y, w, h int) { return 13, 45, 229, 345 }

func (v *EmblemWindow) place(x, y int, view *EmblemView) {
	for _, b := range []*Button{&v.close, &v.apply, &v.erase} {
		b.OriginX, b.OriginY = x, y
	}
	v.apply.Disabled = view.Busy || !view.HasItem
	v.erase.Disabled = view.Busy || !view.HasItem
	for i := range v.arrows {
		b := &v.arrows[i]
		b.OriginX, b.OriginY = x, y
		shape := i / 2
		b.Disabled = view.Busy || !view.HasItem || view.Counts[shape] == 0
	}
}

// PaletteCell is the window-relative corner of palette entry i: a 6x6 grid of
// 32x32 swatches every 34 pixels from (0x199, 0xdd).
func PaletteCell(i int) (int, int) { return 0x199 + (i%6)*0x22, 0xdd + (i/6)*0x22 }

// EmblemPaletteHit returns the clickable swatch at window-relative coordinates.
func EmblemPaletteHit(x, y int) int {
	for i := range emblem.PaletteSize {
		cx, cy := PaletteCell(i)
		if x >= cx && x <= cx+0x1e && y >= cy && y <= cy+0x1e {
			return i
		}
	}
	return -1
}

func (v *EmblemWindow) Update(in *platform.Input, w, h int, view *EmblemView) EmblemAction {
	x, y := v.Origin(w, h)
	v.place(x, y, view)
	act := EmblemAction{Arrow: -1, Color: -1}
	act.Close = v.close.Update(in)
	act.Apply = v.apply.Update(in)
	act.Remove = v.erase.Update(in)
	for i := range v.arrows {
		if v.arrows[i].Update(in) && act.Arrow < 0 {
			act.Arrow = i
		}
	}
	if !in.Clicked || view.Busy || !view.HasItem || act.Arrow >= 0 {
		return act
	}
	rx, ry := in.MouseX-x, in.MouseY-y
	if rx >= 0x118 && rx <= 0x16f {
		for s, r := range emblemFrameY {
			if ry >= r[0] && ry <= r[1] {
				act.Frame = s + 1
			}
		}
	}
	if view.Active != 0 {
		act.Color = EmblemPaletteHit(rx, ry)
	}
	return act
}

func (v *EmblemWindow) Draw(s *render.Sprites, f *Font, w, h int, view *EmblemView) {
	x, y := v.Origin(w, h)
	v.place(x, y, view)
	s.Draw(v.frame, float32(x), float32(y), EmblemWidth, EmblemHeight, 0, 0, float32(EmblemWidth)/1024, 1, white)
	text := view.Gold
	c := GoldRGB(view.Cost)
	f.Draw(s, text, x+0xf3-f.Width(text), y+0x198, [4]float32{float32(c[0]) / 255, float32(c[1]) / 255, float32(c[2]) / 255, 1}, 0)
	v.apply.Draw(s)
	v.erase.Draw(s)
	if view.HasItem {
		for i := range v.arrows {
			v.arrows[i].Draw(s)
		}
		for shape, tex := range view.Textures {
			if tex == nil || view.Selection[shape] < 0 {
				continue
			}
			c := emblem.Color(view.Colors[shape])
			s.Draw(tex, float32(x+0x123), float32(y+emblemPreviewY[shape]), 64, 64, 0, 0, 1, 1,
				[4]float32{float32(c[0]) / 255, float32(c[1]) / 255, float32(c[2]) / 255, 1})
		}
		if view.Active > 0 {
			s.Draw(v.mark, float32(x+0x11e), float32(y+emblemMarkY[view.Active-1]), 73, 73, 0, 0, 73.0/128, 73.0/128, white)
		}
	}
	v.close.Draw(s)
}
