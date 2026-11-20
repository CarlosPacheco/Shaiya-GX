package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const PersonalMarketWidth, PersonalMarketHeight = 192, 391

const PersonalMarketNone = -1
const (
	PersonalMarketClose = proto.MyShopSlots + iota
	PersonalMarketMessage
	PersonalMarketToggle
	PersonalMarketPrice
	PersonalMarketRemove
	PersonalMarketBuy
)

type PersonalMarket struct {
	Position    WindowPosition
	frame, open *render.Texture
	close       Button
	held        int
}

func NewPersonalMarket(t Textures) (*PersonalMarket, error) {
	ld := &loader{t: t}
	p := &PersonalMarket{held: PersonalMarketNone, close: nativeClose(ld.get("talk1_close_button.png"), 163, 5)}
	if optional, ok := t.(interface{ OptionalInterface(string) *render.Texture }); ok {
		p.frame = optional.OptionalInterface("p_market.png")
		p.open = optional.OptionalInterface("p_market_button.png")
	}
	return p, ld.err
}

func (p *PersonalMarket) Origin(w, h int) (int, int) {
	return p.Position.Origin(w, h, PersonalMarketWidth, PersonalMarketHeight, min(46, max(0, w-PersonalMarketWidth)), min(86, max(0, h-PersonalMarketHeight)))
}

func PersonalMarketCell(index int) (x, y int) {
	return 20 + index%4*39, 148 + index/4*39
}

func PersonalMarketCellAt(x, y int) int {
	for i := range proto.MyShopSlots {
		cx, cy := PersonalMarketCell(i)
		if x >= cx && x < cx+34 && y >= cy && y < cy+34 {
			return i
		}
	}
	return PersonalMarketNone
}

func (p *PersonalMarket) CancelHeld() {
	p.held = PersonalMarketNone
	p.close.held = false
}

func (p *PersonalMarket) Update(in *platform.Input, w, h int, buyer bool) int {
	x, y := p.Origin(w, h)
	p.close.OriginX, p.close.OriginY = x, y
	if p.close.Update(in) || in.HasKey(platform.KeyEscape) {
		return PersonalMarketClose
	}
	hit := PersonalMarketNone
	mx, my := in.MouseX-x, in.MouseY-y
	switch {
	case mx >= 16 && mx < 174 && my >= 62 && my < 119 && !buyer:
		hit = PersonalMarketMessage
	case my >= 360 && my < 383:
		switch {
		case buyer && mx >= 58 && mx < 132:
			hit = PersonalMarketBuy
		case !buyer && mx >= 9 && mx < 81:
			hit = PersonalMarketToggle
		case !buyer && mx >= 83 && mx < 124:
			hit = PersonalMarketPrice
		case !buyer && mx >= 126 && mx < 180:
			hit = PersonalMarketRemove
		}
	default:
		hit = PersonalMarketCellAt(mx, my)
	}
	if in.Pressed {
		p.held = hit
	}
	if in.Clicked {
		held := p.held
		p.held = PersonalMarketNone
		if held == hit {
			return hit
		}
	}
	return PersonalMarketNone
}

// Draw controls and the native frame. Item icons come from instance data owned by the app.
func (p *PersonalMarket) Draw(s *render.Sprites, f *Font, blank *render.Texture, w, h int, title, detail string, selected int, buyer, selling, busy bool) {
	x, y := p.Origin(w, h)
	rect := func(left, top, width, height int, color [4]float32) {
		s.Draw(blank, float32(x+left), float32(y+top), float32(width), float32(height), 0, 0, 1, 1, color)
	}
	if p.frame != nil {
		s.Draw(p.frame, float32(x), float32(y), PersonalMarketWidth, PersonalMarketHeight, 0, 0, float32(PersonalMarketWidth)/256, float32(PersonalMarketHeight)/512, white)
	} else {
		rect(0, 0, PersonalMarketWidth, PersonalMarketHeight, RGBColor(0x584933))
		rect(2, 2, PersonalMarketWidth-4, PersonalMarketHeight-4, RGBColor(0x1c1a18))
		rect(16, 62, 158, 57, RGBColor(0x0d0d0e))
		f.Draw(s, "Player Market", x+17, y+8, white, 144)
		f.Draw(s, "Market Message", x+17, y+45, white, 144)
		f.Draw(s, "Products", x+17, y+127, white, 144)
		for i := range proto.MyShopSlots {
			cx, cy := PersonalMarketCell(i)
			rect(cx-1, cy-1, 35, 35, RGBColor(0x665745))
			rect(cx, cy, 33, 33, RGBColor(0x292723))
		}
	}
	if selected >= 0 && selected < proto.MyShopSlots {
		cx, cy := PersonalMarketCell(selected)
		rect(cx-1, cy-1, 35, 35, RGBColor(0xa48352))
	}
	_, lines := wrapMarkupWidth(f, title, white, false, 151)
	for i, line := range lines[:min(len(lines), max(1, 50/f.Height))] {
		f.Draw(s, line, x+20, y+65+i*f.Height, white, 151)
	}
	f.Draw(s, detail, x+13, y+344, RGBColor(0xe6cc92), 166)
	button := func(label string, left, width int, enabled bool) {
		tint := RGBColor(0xe6cc92)
		if !enabled {
			tint = RGBColor(0x888178)
		}
		rect(left, 360, width, 21, RGBColor(0x665745))
		rect(left+1, 361, width-2, 19, RGBColor(0x28231d))
		f.Draw(s, label, x+left+(width-f.Width(label))/2, y+362, tint, width)
	}
	if buyer {
		button("Buy", 58, 74, !busy && selected >= 0)
	} else {
		if selling {
			button("Stop selling", 9, 72, !busy)
		} else if p.open != nil {
			row := float32(0)
			if busy {
				row = .75
			} else if p.held == PersonalMarketToggle {
				row = .5
			}
			s.Draw(p.open, float32(x+13), float32(y+359), 64, 23, 0, row, 1, row+23.0/128, white)
		} else {
			button("Open", 9, 72, !busy)
		}
		button("Price", 83, 41, !selling && !busy && selected >= 0)
		button("Remove", 126, 54, !selling && !busy && selected >= 0)
	}
	p.close.OriginX, p.close.OriginY = x, y
	p.close.Draw(s)
}
