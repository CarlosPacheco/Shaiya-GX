package ui

import (
	"strconv"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const RevolverPages, RevolverSlots = 4, 6

// RevolverArt lists the parts loaded by the EP5.4 constructor and orientation switch.
var RevolverArt = [...]string{
	"Revolver_View1.png", "Revolver_View2.png", "RevolverAdd1.png",
	"RevolverAdd2.png", "RevolverAdd3.png", "RevolverAdd4.png",
	"RevolverAdd5.png", "RevolverAdd6.png", "RevolverAdd7.png",
}

type RevolverBox struct{ X, Y, W, H int }

func (b RevolverBox) Contains(x, y int) bool {
	return x >= b.X && y >= b.Y && x < b.X+b.W && y < b.Y+b.H
}

type RevolverLayout struct {
	Panel, Main, Editor RevolverBox
	Vertical            bool
}

func NewRevolverLayout(pos WindowPosition, w, h, count int, expanded bool) RevolverLayout {
	width, height := 158, 58
	if pos.Vertical {
		width, height = 58, 158
	}
	if expanded {
		if pos.Vertical {
			width, height = 111, max(height, 40*count+40)
		} else {
			width, height = max(width, 40*count+40), 111
		}
	}
	x, y := pos.Origin(w, h, width, height, w-204, 220)
	l := RevolverLayout{Panel: RevolverBox{x, y, width, height}, Main: RevolverBox{x, y, 158, 58}, Vertical: pos.Vertical}
	if pos.Vertical {
		l.Main = RevolverBox{x, y, 58, 158}
	}
	if expanded {
		if pos.Vertical {
			l.Main.X += 53
			l.Editor = RevolverBox{x, y, 53, 40*count + 40}
		} else {
			l.Editor = RevolverBox{x, y + 58, 40*count + 40, 53}
		}
	}
	return l
}

func (l RevolverLayout) Cell(i int, editor bool) RevolverBox {
	if editor {
		if l.Vertical {
			return RevolverBox{l.Editor.X + 10, l.Editor.Y + 29 + 40*i, 32, 32}
		}
		return RevolverBox{l.Editor.X + 29 + 40*i, l.Editor.Y + 10, 32, 32}
	}
	if l.Vertical {
		return RevolverBox{l.Main.X + 7, l.Main.Y + 22 + 40*i, 32, 32}
	}
	return RevolverBox{l.Main.X + 22 + 40*i, l.Main.Y + 18, 32, 32}
}

// RevolverCompactSlot maps a visible cell to the previous, current or next
// entry of the current page.
func RevolverCompactSlot(cell, selected, count int) int {
	return (selected + cell + count - 1) % count
}

// Controls are ordered rotate, reset, editor, advance, go back,
// add cell, remove cell, and next page.
func (l RevolverLayout) Controls() [8]RevolverBox {
	x, y := l.Main.X, l.Main.Y
	var out [8]RevolverBox
	if l.Vertical {
		out = [8]RevolverBox{{x + 14, y + 141, 12, 12}, {x + 25, y + 141, 12, 12}, {x + 37, y + 141, 12, 12}, {x + 40, y + 17, 14, 14}, {x + 40, y + 124, 14, 14}}
	} else {
		out = [8]RevolverBox{{x + 141, y + 14, 12, 12}, {x + 141, y + 25, 12, 12}, {x + 141, y + 37, 12, 12}, {x + 19, y + 1, 14, 14}, {x + 124, y + 1, 14, 14}}
	}
	if l.Vertical {
		out[7] = RevolverBox{x + 21, y + 5, 14, 14}
	} else {
		out[7] = RevolverBox{x + 7, y + 27, 14, 14}
	}
	if l.Editor.W > 0 {
		x, y = l.Editor.X, l.Editor.Y
		if l.Vertical {
			out[5], out[6] = RevolverBox{x + 10, y + 7, 16, 16}, RevolverBox{x + 27, y + 7, 16, 16}
		} else {
			out[5], out[6] = RevolverBox{x + 4, y + 10, 16, 16}, RevolverBox{x + 4, y + 27, 16, 16}
		}
	}
	return out
}

func (l RevolverLayout) Draw(s *render.Sprites, f *Font, art [9]*render.Texture, blank *render.Texture, slots [RevolverSlots]ActionSlot, page, selected, count int) {
	draw := func(tex *render.Texture, b RevolverBox, rgba [4]float32) {
		if tex == nil {
			tex = blank
			rgba = [4]float32{.16, .12, .09, .94}
		}
		s.Draw(tex, float32(b.X), float32(b.Y), float32(b.W), float32(b.H), 0, 0, min(1, float32(b.W)/float32(tex.W)), min(1, float32(b.H)/float32(tex.H)), rgba)
	}
	orientation, parts := 0, 2
	if l.Vertical {
		orientation, parts = 1, 6
	}
	draw(art[orientation], l.Main, white)
	mainCell := l.Cell(1, false)
	if art[orientation] == nil {
		for i, box := range l.Controls() {
			if box.W > 0 && i < 5 {
				f.DrawOutlined(s, []string{"↻", "↶", "+", "‹", "›"}[i], box.X, box.Y, white)
			}
		}
	}
	f.DrawOutlined(s, "R", mainCell.X+22, mainCell.Y+18, white)
	pageControl := l.Controls()[7]
	f.DrawOutlined(s, strconv.Itoa(page+1), pageControl.X, pageControl.Y, white)
	icon := func(slot ActionSlot, b RevolverBox, tint [4]float32) {
		if slot.Icon == nil {
			return
		}
		u := slot.UV
		s.Draw(slot.Icon, float32(b.X), float32(b.Y), 32, 32, u.U0, u.V0, u.U1, u.V1, tint)
		drawShortcutCooldown(s, f, blank, b.X, b.Y, slot.Cooldown, slot.CooldownDuration)
	}
	for i := range 3 {
		tint := white
		if i != 1 {
			tint = [4]float32{.4, .4, .4, 1}
		}
		icon(slots[RevolverCompactSlot(i, selected, count)], l.Cell(i, false), tint)
	}
	if l.Editor.W == 0 {
		return
	}
	x, y := l.Editor.X, l.Editor.Y
	if l.Vertical {
		draw(art[parts], RevolverBox{x, y, 53, 25}, white)
		draw(art[parts+2], RevolverBox{x, y + 25 + 40*count, 53, 15}, white)
	} else {
		draw(art[parts], RevolverBox{x, y, 25, 53}, white)
		draw(art[parts+2], RevolverBox{x + 25 + 40*count, y, 15, 53}, white)
	}
	for i := range count {
		b := l.Cell(i, true)
		frame := RevolverBox{b.X - 4, l.Editor.Y, 40, 53}
		if l.Vertical {
			frame = RevolverBox{l.Editor.X, b.Y - 4, 53, 40}
		}
		draw(art[parts+1], frame, white)
		f.DrawOutlined(s, strconv.Itoa(i+1), b.X+22, b.Y+18, white)
		icon(slots[i], b, white)
		if i == selected && art[5] != nil {
			draw(art[5], RevolverBox{b.X - 4, b.Y - 4, 41, 41}, white)
		}
	}
	if art[parts] == nil {
		for i, label := range []string{"+", "−"} {
			b := l.Controls()[5+i]
			f.DrawOutlined(s, label, b.X, b.Y, white)
		}
	}
}
