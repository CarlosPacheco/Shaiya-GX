package ui

import (
	"fmt"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const ActionSlots = 10
const ActionBars = 3
const ActionPages = 5
const TotalActionSlots = ActionSlots * ActionPages

type ActionSlot struct {
	Badge            ItemBadge
	Icon             *render.Texture
	UV               UV
	Label            string
	Count            uint16
	Cooldown         float64
	Pressed          bool
	CooldownDuration float64
}

// Phoenix ep4_action_bars.gd and main_slot_hud.gd: two independently
// movable bars, background drag handles and centre-preserving orientation.
type ActionBar struct {
	frames                [2]*render.Texture
	change, plus          *render.Texture
	pages                 [ActionBars]int // offset from each bar’s initial page
	HideSecond, ShowThird bool
	Position              [ActionBars]WindowPosition
	Hover, held           int
	dragging, controlHeld int
	control               int // one-based; zero means idle
	offsetX, offsetY      int
}

func NewActionBar(t Textures, plus *render.Texture) (*ActionBar, error) {
	ld := &loader{t: t}
	return &ActionBar{plus: plus, frames: [2]*render.Texture{ld.get("main_slot_1.png"), ld.get("main_slot_2.png")}, change: ld.get("main_slot_change.png"), Hover: -1, held: -1}, ld.err
}
func (b *ActionBar) Size(bar int) (int, int) {
	if b.Position[bar].Vertical {
		return 56, 449
	}
	return 448, 54
}
func ActionBarOrigin(w, h int) (int, int) { return max(0, (w-448)/2), max(0, h-64-8-54*2-6) }
func (b *ActionBar) Origin(bar, w, h int) (int, int) {
	x, y := ActionBarOrigin(w, h)
	if bar == 2 {
		y -= 60
	} else {
		y += bar * 60
	}
	width, height := b.Size(bar)
	return b.Position[bar].Origin(w, h, width, height, x, y)
}

// VisibleIndex converts a physical key or cell to its current logical page slot.
func (b *ActionBar) VisibleIndex(index int) int {
	if index < 0 || index >= ActionBars*ActionSlots {
		return -1
	}
	bar, cell := index/ActionSlots, index%ActionSlots
	if (bar > 0 && b.HideSecond) || (bar == 2 && !b.ShowThird) {
		return -1
	}
	return (bar+b.pages[bar])*ActionSlots + cell
}

func (b *ActionBar) cellOrigin(bar, cell, w, h int) (int, int) {
	x, y := b.Origin(bar, w, h)
	if b.Position[bar].Vertical {
		return x + 13, y + 30 + 40*cell
	}
	return x + 30 + 40*cell, y + 10
}

func (b *ActionBar) SlotOrigin(index, w, h int) (int, int) {
	for bar := 0; bar < ActionBars; bar++ {
		if first := b.VisibleIndex(bar * ActionSlots); first >= 0 && index >= first && index < first+ActionSlots {
			return b.cellOrigin(bar, index-first, w, h)
		}
	}
	return -10000, -10000
}

func (b *ActionBar) SlotAt(mx, my, w, h int) int {
	bar := b.barAt(mx, my, w, h)
	if bar < 0 {
		return -1
	}
	first := b.VisibleIndex(bar * ActionSlots)
	for cell := 0; cell < ActionSlots; cell++ {
		x, y := b.cellOrigin(bar, cell, w, h)
		if mx >= x && mx < x+32 && my >= y && my < y+32 {
			return first + cell
		}
	}
	return -1
}

// ActionSlotAt retains the default-layout helper for synthetic fixtures.
func ActionSlotAt(mx, my, w, h int) int { return (&ActionBar{}).SlotAt(mx, my, w, h) }

const (
	barBackground = iota
	barPrevious
	barNext
	barRotate
	barToggle
)

func (b *ActionBar) controlAt(bar, mx, my, w, h int) int {
	x, y := b.Origin(bar, w, h)
	mx, my = mx-x, my-y
	prevX, prevY, nextX, nextY := 6, 4, 6, 32
	rotateX, rotateY, toggleX, toggleY := 424, 30, 426, 8
	if b.Position[bar].Vertical {
		prevX, prevY, nextX, nextY = 4, 6, 32, 6
		rotateX, rotateY, toggleX, toggleY = 4, 425, 32, 426
	}
	switch {
	case mx >= prevX && mx < prevX+12 && my >= prevY && my < prevY+12:
		return barPrevious
	case mx >= nextX && mx < nextX+12 && my >= nextY && my < nextY+12:
		return barNext
	case bar < ActionBars-1 && mx >= toggleX && mx < toggleX+16 && my >= toggleY && my < toggleY+16:
		return barToggle
	case mx >= rotateX && mx < rotateX+22 && my >= rotateY && my < rotateY+24:
		return barRotate
	}
	return barBackground
}

func (b *ActionBar) barAt(mx, my, w, h int) int {
	for bar := ActionBars - 1; bar >= 0; bar-- {
		if b.VisibleIndex(bar*ActionSlots) < 0 {
			continue
		}
		x, y := b.Origin(bar, w, h)
		bw, bh := b.Size(bar)
		if mx >= x && mx < x+bw && my >= y && my < y+bh {
			return bar
		}
	}
	return -1
}

func (b *ActionBar) Contains(mx, my, w, h int) bool {
	return b.barAt(mx, my, w, h) >= 0
}
func (b *ActionBar) LayoutActive() bool { return b.dragging > 0 || b.controlHeld > 0 }
func (b *ActionBar) CancelHeld()        { b.held = -1 }

// Layout captures the complete drag/release; changed requests persistence.
func (b *ActionBar) Layout(in *platform.Input, w, h int) (captured, changed bool) {
	if b.dragging > 0 {
		bar := b.dragging - 1
		bw, bh := b.Size(bar)
		b.Position[bar].Move(in.MouseX-b.offsetX, in.MouseY-b.offsetY, w, h, bw, bh)
		if in.Clicked || !in.MouseDown {
			b.dragging = 0
			changed = true
		}
		return true, changed
	}
	if b.controlHeld > 0 {
		bar := b.controlHeld - 1
		if in.Clicked {
			if b.controlAt(bar, in.MouseX, in.MouseY, w, h) == b.control {
				switch b.control {
				case barPrevious:
					b.pages[bar] = max(0, bar+b.pages[bar]-1) - bar
				case barNext:
					b.pages[bar] = min(ActionPages-1, bar+b.pages[bar]+1) - bar
				case barToggle:
					if bar == 0 {
						b.HideSecond = !b.HideSecond
						if b.HideSecond {
							b.ShowThird = false
						}
					} else {
						b.ShowThird = !b.ShowThird
					}
					changed = true
				case barRotate:
					x, y := b.Origin(bar, w, h)
					bw, bh := b.Size(bar)
					cx, cy := x+bw/2, y+bh/2
					b.Position[bar].Vertical = !b.Position[bar].Vertical
					bw, bh = b.Size(bar)
					b.Position[bar].Move(cx-bw/2, cy-bh/2, w, h, bw, bh)
					changed = true
				}
				b.Hover = -1
			}
			b.controlHeld = 0
		} else if !in.MouseDown {
			b.controlHeld = 0
		}
		return true, changed
	}
	if !in.Pressed || b.SlotAt(in.MouseX, in.MouseY, w, h) >= 0 {
		return false, false
	}
	bar := b.barAt(in.MouseX, in.MouseY, w, h)
	if bar < 0 {
		return false, false
	}
	b.CancelHeld()
	if control := b.controlAt(bar, in.MouseX, in.MouseY, w, h); control != barBackground {
		b.controlHeld, b.control = bar+1, control
	} else {
		x, y := b.Origin(bar, w, h)
		b.dragging = bar + 1
		b.offsetX, b.offsetY = in.MouseX-x, in.MouseY-y
	}
	return true, false
}
func (b *ActionBar) Update(in *platform.Input, w, h int) (int, bool) {
	b.Hover = b.SlotAt(in.MouseX, in.MouseY, w, h)
	if in.Pressed {
		b.held = b.Hover
	}
	click := -1
	wasHeld := b.held >= 0
	if in.Clicked {
		if b.Hover >= 0 && b.held == b.Hover {
			click = b.Hover
		}
		b.held = -1
	}
	return click, b.Contains(in.MouseX, in.MouseY, w, h) && (in.Pressed || in.Clicked || in.MouseDown || in.RightPressed || in.RightDown || in.Wheel != 0) || wasHeld && in.Clicked
}
func (b *ActionBar) Draw(s *render.Sprites, f *Font, blank *render.Texture, w, h int, slots [TotalActionSlots]ActionSlot, heldKeys uint32) {
	for bar := 0; bar < ActionBars; bar++ {
		first := b.VisibleIndex(bar * ActionSlots)
		if first < 0 {
			continue
		}
		x, y := b.Origin(bar, w, h)
		bw, bh := b.Size(bar)
		orientation := 0
		if b.Position[bar].Vertical {
			orientation = 1
		}
		tex := b.frames[orientation]
		if tex != nil {
			s.Draw(tex, float32(x), float32(y), float32(bw), float32(bh), 0, 0, float32(bw)/float32(tex.W), float32(bh)/float32(tex.H), white)
		}
		nx, ny := x+10, y+21
		cx, cy, tx, ty := x+426, y+27, x+426, y+8
		if orientation == 1 {
			nx, ny = x+22, y+9
			cx, cy, tx, ty = x+10, y+426, x+32, y+426
		}
		f.Draw(s, fmt.Sprint(first/ActionSlots+1), nx, ny, white, 18)
		if b.change != nil {
			v := float32(1-orientation) * .5
			s.Draw(b.change, float32(cx), float32(cy), 16, 16, 0, v, 16.0/float32(b.change.W), v+16.0/float32(b.change.H), white)
		}
		if bar < ActionBars-1 {
			showingNext := b.VisibleIndex((bar+1)*ActionSlots) >= 0
			if b.plus != nil {
				v := float32(0)
				if showingNext {
					v = .5
				}
				s.Draw(b.plus, float32(tx), float32(ty), 32, 32, 0, v, 1, v+.5, white)
			} else {
				label := "+"
				if showingNext {
					label = "−"
				}
				f.DrawOutlined(s, label, tx+2, ty, white)
			}
		}
		for cell := 0; cell < ActionSlots; cell++ {
			index := first + cell
			slot := slots[index]
			if slot.Icon == nil {
				continue
			}
			cx, cy := b.cellOrigin(bar, cell, w, h)
			u := slot.UV
			s.Draw(slot.Icon, float32(cx), float32(cy), 32, 32, u.U0, u.V0, u.U1, u.V1, white)
			slot.Badge.Draw(s, cx, cy, 32, white)
			if slot.Count > 0 {
				f.DrawOutlined(s, fmt.Sprint(slot.Count), cx+12, cy+15, white)
			}
			drawShortcutCooldown(s, f, blank, cx, cy, slot.Cooldown, slot.CooldownDuration)
			if slot.Pressed || b.held == index || heldKeys&(1<<(bar*ActionSlots+cell)) != 0 {
				s.Draw(blank, float32(cx), float32(cy), 32, 32, 0, 0, 1, 1, [4]float32{1, 1, 1, 80.0 / 255})
			}
		}
	}
}

func drawShortcutCooldown(s *render.Sprites, f *Font, blank *render.Texture, x, y int, remaining, duration float64) {
	if remaining <= 0 {
		return
	}
	ratio := float32(1)
	if duration > 0 {
		ratio = float32(remaining / duration)
	}
	s.DrawCooldown(blank, float32(x), float32(y), 32, ratio, [4]float32{1, 0, 0, 80.0 / 255})
	label := fmt.Sprint(int(math.Ceil(remaining)))
	f.DrawOutlined(s, label, x+16-f.Width(label)/2, y+9, [4]float32{1, 1, 0, 1})
}

// SkillIcon matches 0x4a1810: thousands select the sheet, remainder is 1-based.
func SkillIcon(icon uint16) (string, UV, bool) {
	index := int(icon%1000) - 1
	if index < 0 || index >= 256 {
		return "", UV{}, false
	}
	name := "icon/icon_skill.png"
	if icon >= 1000 {
		name = fmt.Sprintf("icon/icon_skill%d_dds.png", int(icon)/1000+1)
	}
	u, v := float32(index%16)/16, float32(index/16)/16
	return name, UV{u, v, u + 1.0/16, v + 1.0/16}, true
}
