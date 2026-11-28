package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// GX widens the original 580 px sheet by mallExtra for the preview.
const MallWidth, MallHeight = 580 + mallExtra, 446
const GiftBoxWidth, GiftBoxHeight = 333, 315

type MallProduct struct {
	Badge                   ItemBadge
	Name, Code, Description string
	Cost                    uint32
	ItemIDs                 []uint32
	Icon                    *render.Texture
	UV                      UV
}

type MallAction uint8

const (
	MallNone MallAction = iota
	MallClose
	MallBuy
	MallGift
	MallShowGifts
	MallCloseGifts
)

// Mall uses the original 1024x512 and 512x512 EP5 interface sheets.
type Mall struct {
	Position, GiftPosition                       WindowPosition
	frame, giftFrame, blank, scrollbar           *render.Texture
	tabs                                         [4]Button
	buy, gift, giftBox, close, cancel, giftClose Button
	arrows                                       [2]Button
	scrolling, turning                           bool
	turnX                                        int
	// PreviewYaw turns the previewed character or mount (drag to rotate).
	PreviewYaw float32
	Category, Scroll, Selected, GiftPage         int
	GiftOpen                                     bool
	heldProduct, heldGift, heldTab               int
	selectedGift                                 int
	lastGift                                     int
	lastGiftClick                                float64
}

func NewMall(t Textures, blank *render.Texture) (*Mall, error) {
	ld := &loader{t: t}
	m := &Mall{frame: ld.get("ins_market.png"), giftFrame: ld.get("ins_market_store.png"), scrollbar: ld.get("scrollbar.png"), blank: blank, Selected: -1, heldProduct: -1, heldGift: -1, heldTab: -1, selectedGift: -1, lastGift: -1}
	for i := range m.arrows {
		b := &m.arrows[i]
		*b = Button{Tex: m.scrollbar, OffX: 565 + mallListShift, OffY: mallListTop + i*(340-mallListTop), DrawW: 16, DrawH: 32, HitW: 13, HitH: 19}
		for state, col := range []int{0, 1, 1, 1, 2} {
			b.UV[state] = UV{float32(col) / 8, float32(i) / 2, float32(col+1) / 8, float32(i+1) / 2}
		}
	}
	tabs := ld.get("ins_market_menu_button.png")
	for i := range m.tabs {
		b := &m.tabs[i]
		// GX: the four tabs are compacted over the product list.
		*b = Button{Tex: tabs, OffX: 16 + i*mallTabStep, OffY: 46, DrawW: mallTabStep * 4 / 3, DrawH: 32, HitW: mallTabStep, HitH: 26, Toggle: true}
		for state, row := range []int{0, 1, 2, 1, 3} {
			b.UV[state] = UV{float32(i) / 4, float32(row) / 4, float32(i+1) / 4, float32(row+1) / 4}
		}
	}
	tex := ld.get("ins_market_button.png")
	for i, b := range []*Button{&m.buy, &m.gift} {
		*b = Button{Tex: tex, OffX: []int{MallBuyX, MallGiftX}[i], OffY: 401, DrawW: 128, DrawH: 32, HitW: 86, HitH: 26}
		for state, row := range []int{0, 1, 2, 2, 3} {
			b.UV[state] = UV{float32(i) / 2, float32(row) / 8, float32(i+1) / 2, float32(row+1) / 8}
		}
	}
	m.giftBox = Button{Tex: ld.get("ins_market_store_button.png"), OffX: MallGiftBoxX, OffY: 401, DrawW: 128, DrawH: 32, HitW: 107, HitH: 26, UV: FourRowUV}
	m.cancel = Button{Tex: ld.get("cancel_button.png"), OffX: MallBuyX + mallButtonStep, OffY: 401, DrawW: 128, DrawH: 32, HitW: 65, HitH: 25, UV: FourRowUV}
	closeTex := ld.get("talk1_close_button.png")
	m.close = nativeClose(closeTex, 554+mallExtra, 3)
	m.giftClose = nativeClose(closeTex, 307, 4)
	return m, ld.err
}
func (m *Mall) Origin(w, h int) (int, int) {
	return m.Position.Origin(w, h, MallWidth, MallHeight, (w-MallWidth)/2, (h-MallHeight)/2)
}
func (m *Mall) GiftOrigin(w, h int) (int, int) {
	return m.GiftPosition.Origin(w, h, GiftBoxWidth, GiftBoxHeight, (w-GiftBoxWidth)/2, (h-GiftBoxHeight)/2)
}
func (m *Mall) Reset() {
	m.Category, m.Scroll, m.Selected, m.GiftPage = 0, 0, -1, 0
	m.GiftOpen = false
	m.scrolling, m.turning, m.PreviewYaw = false, false, 0
	m.heldProduct, m.heldGift, m.heldTab, m.selectedGift, m.lastGift = -1, -1, -1, -1, -1
}

// GX layout: the product list is compacted to the left half and the right
// half shows the character (or mount) wearing the selected product.
const (
	mallTabStep     = 92
	mallColumn      = 178
	mallListRight   = 386
	mallListShift   = mallListRight - 12 - 556 // scrollbar strip moved left
	mallListLeft    = 16
	mallListTop     = 82
	mallListRows    = 4
	mallListBottom  = mallListTop + mallListRows*65
	mallExtra       = 120
	mallButtonStep  = 92
	mallTrack       = 320 - mallListTop - 20
	mallProductWide = 174
	mallPointsY     = 337
	// Bottom buttons: gifts on the left, buy and cancel on the right.
	MallGiftBoxX = 24
	MallGiftX    = MallGiftBoxX + 113
	MallBuyX     = MallWidth - 91 - mallButtonStep
)

func MallProductRect(index int) (int, int) { return 19 + (index%2)*mallColumn, mallListTop + (index/2)*65 }

// MallPreviewRect is the character viewport, relative to the window.
func MallPreviewRect() (x, y, w, h int) { return mallListRight + 14, 47, 566 + mallExtra - mallListRight - 14, 282 }

func (m *Mall) listCell(x, y, mouseX, mouseY, count int) int {
	if mouseX < x+mallListLeft || mouseX >= x+mallListLeft+2*mallColumn || mouseY < y+mallListTop || mouseY >= y+mallListBottom {
		return -1
	}
	col, row := (mouseX-x-mallListLeft)/mallColumn, (mouseY-y-mallListTop)/65
	index := (m.Scroll+row)*2 + col
	if index >= count {
		return -1
	}
	return index
}
func MallGiftCell(slot int) (int, int)     { return 10 + (slot%40)%8*39, 78 + (slot%40)/8*39 }

func (m *Mall) ProductAt(w, h, mouseX, mouseY, count int) int {
	x, y := m.Origin(w, h)
	return m.listCell(x, y, mouseX, mouseY, count)
}

func (m *Mall) Update(in *platform.Input, w, h, count int, busy bool) (MallAction, bool) {
	x, y := m.Origin(w, h)
	for i := range m.tabs {
		b := &m.tabs[i]
		b.OriginX, b.OriginY, b.Checked = x, y, i == m.Category
		if b.Update(in) {
			m.Category, m.Scroll, m.Selected = i, 0, -1
		}
	}
	m.close.OriginX, m.close.OriginY = x, y
	m.cancel.OriginX, m.cancel.OriginY = x, y
	if m.close.Update(in) || in.CloseOnEscape() || m.cancel.Update(in) {
		return MallClose, true
	}
	m.giftBox.OriginX, m.giftBox.OriginY = x, y
	if m.giftBox.Update(in) {
		m.GiftOpen = true
		return MallShowGifts, true
	}
	for i, b := range []*Button{&m.buy, &m.gift} {
		b.OriginX, b.OriginY, b.Disabled = x, y, busy || m.Selected < 0 || m.Selected >= count
		if b.Update(in) {
			if i == 0 {
				return MallBuy, true
			}
			return MallGift, true
		}
	}
	insideList := in.MouseX >= x+mallListLeft && in.MouseX < x+mallListRight && in.MouseY >= y+mallListTop && in.MouseY < y+mallListBottom
	maxScroll := max(0, (count+1)/2-mallListRows)
	for i := range m.arrows {
		b := &m.arrows[i]
		b.OriginX, b.OriginY, b.Disabled = x, y, maxScroll == 0 || i == 0 && m.Scroll == 0 || i == 1 && m.Scroll >= maxScroll
		if b.Update(in) {
			m.Scroll = max(0, min(maxScroll, m.Scroll+i*2-1))
		}
	}
	if maxScroll > 0 && in.Pressed && in.MouseX >= x+565+mallListShift && in.MouseX < x+580+mallListShift && in.MouseY >= y+mallListTop+20 && in.MouseY < y+340 {
		m.scrolling = true
	}
	if m.scrolling {
		if in.MouseDown || in.Clicked {
			m.Scroll = max(0, min(maxScroll, int(math.Round(float64(in.MouseY-y-mallListTop-20)*float64(maxScroll)/mallTrack))))
		}
		if in.Clicked || !in.MouseDown {
			m.scrolling = false
		}
	}
	if insideList && in.Wheel != 0 {
		if in.Wheel > 0 {
			m.Scroll--
		} else {
			m.Scroll++
		}
		m.Scroll = max(0, min(m.Scroll, maxScroll))
	}
	px, py, pw, ph := MallPreviewRect()
	if in.Pressed && in.MouseX >= x+px && in.MouseX < x+px+pw && in.MouseY >= y+py && in.MouseY < y+py+ph {
		m.turning, m.turnX = true, in.MouseX
	}
	if m.turning {
		m.PreviewYaw += float32(in.MouseX-m.turnX) * .012
		m.turnX = in.MouseX
		if in.Clicked || !in.MouseDown {
			m.turning = false
		}
	}
	hit := m.listCell(x, y, in.MouseX, in.MouseY, count)
	if in.Pressed {
		m.heldProduct = hit
	}
	if in.Clicked {
		if hit >= 0 && hit == m.heldProduct {
			m.Selected = hit
		}
		m.heldProduct = -1
	}
	inside := in.MouseX >= x && in.MouseX < x+MallWidth && in.MouseY >= y && in.MouseY < y+MallHeight
	return MallNone, m.scrolling || m.turning || inside && (in.Pressed || in.Clicked || in.MouseDown || in.RightPressed || in.RightDown || in.Wheel != 0)
}

// piece copies a frame region (sx,sy,sw,sh) to (dx,dy) at size (dw,dh).
func (m *Mall) piece(s *render.Sprites, x, y, dx, dy, dw, dh, sx, sy, sw, sh int) {
	s.Draw(m.frame, float32(x+dx), float32(y+dy), float32(dw), float32(dh), float32(sx)/1024, float32(sy)/512, float32(sx+sw)/1024, float32(sy+sh)/512, white)
}

func (m *Mall) line(s *render.Sprites, x, y, w, h int, c [4]float32) {
	if m.blank != nil {
		s.Draw(m.blank, float32(x), float32(y), float32(w), float32(h), 0, 0, 1, 1, c)
	}
}

// drawLayout rebuilds the original sheet into the GX layout: a points panel
// in place of the identity panel, a narrower list and the preview box.
func (m *Mall) drawLayout(s *render.Sprites, x, y int) {
	s.Draw(m.frame, float32(x), float32(y), 580, MallHeight, 0, 0, 580.0/1024, float32(MallHeight)/512, white)
	// Widen: the right part of the sheet moves out, the title recentres.
	m.piece(s, x, y, 360+mallExtra, 0, 220, MallHeight, 360, 0, 220, MallHeight)
	m.piece(s, x, y, 360, 0, mallExtra, MallHeight, 360, 0, mallExtra, MallHeight)
	m.piece(s, x, y, 240, 0, 60, 40, 380, 0, 60, 40)
	m.piece(s, x, y, 240+mallExtra/2, 0, 120, 40, 240, 0, 120, 40)
	// Wall over the old panels.
	for dx := 14; dx < mallListRight+14; dx += 126 {
		w := min(126, mallListRight+14-dx)
		for _, r := range [][2]int{{44, 82}, {126, 82}, {208, 82}, {290, 74}} {
			m.piece(s, x, y, dx, r[0], w, r[1], 238, 44, w, r[1])
		}
	}
	// Tab band and the taller list box under it.
	m.piece(s, x, y, 6, 42, mallListRight-6, 36, 6, 127, mallListRight-6, 36)
	top := mallListTop - 2
	m.piece(s, x, y, 16, top, mallListRight-28, 6, 16, 162, mallListRight-28, 6)
	m.piece(s, x, y, 16, top+6, mallListRight-28, 357-top, 16, 168, mallListRight-28, 189)
	m.piece(s, x, y, 16, 357, mallListRight-28, 7, 16, 357, mallListRight-28, 7)
	m.piece(s, x, y, mallListRight-12, top, 12, 364-top, 556, 162, 12, 202)
	// Preview box and the points panel under it.
	px, py, pw, ph := MallPreviewRect()
	m.piece(s, x, y, px, py, pw, ph, 20, 166, 260, 192)
	edge := [4]float32{55.0 / 255, 55.0 / 255, 54.0 / 255, 1}
	m.line(s, x+px-1, y+py-1, pw+2, 1, edge)
	m.line(s, x+px-1, y+py-1, 1, ph+2, edge)
	m.line(s, x+px+pw, y+py-1, 1, ph+2, edge)
	m.line(s, x+px-1, y+py+ph, pw+2, 1, edge)
	// Wall under the preview, around the points panel.
	for dx := px - 2; dx < px+pw+2; dx += 126 {
		w := min(126, px+pw+2-dx)
		m.piece(s, x, y, dx, py+ph+1, w, 364-py-ph, 238, 44, w, 364-py-ph)
	}
	// Points strip: as wide as the preview, list-box finish and a gold rule.
	m.piece(s, x, y, px, mallPointsY, pw, 24, 20, 200, 260, 24)
	m.line(s, x+px-1, y+mallPointsY-1, pw+2, 1, edge)
	m.line(s, x+px-1, y+mallPointsY+24, pw+2, 1, edge)
	m.line(s, x+px-1, y+mallPointsY-1, 1, 26, edge)
	m.line(s, x+px+pw, y+mallPointsY-1, 1, 26, edge)
	m.line(s, x+px+8, y+mallPointsY+3, 2, 18, [4]float32{.95, .8, .45, .9})
}

func (m *Mall) Draw(s *render.Sprites, f *Font, w, h int, products []MallProduct, pointsLabel string, points uint32, pointsReady, busy bool) {
	x, y := m.Origin(w, h)
	m.drawLayout(s, x, y)
	px, _, pw, _ := MallPreviewRect()
	f.Draw(s, pointsLabel, x+px+16, y+mallPointsY+5, white, 80)
	pointText := "..."
	if pointsReady {
		pointText = fmt.Sprint(points)
	}
	f.Draw(s, pointText, x+px+pw-10-f.Width(pointText), y+mallPointsY+5, [4]float32{.95, .8, .45, 1}, 120)
	for i := range m.tabs {
		b := &m.tabs[i]
		b.OriginX, b.OriginY, b.Checked = x, y, i == m.Category
		b.Draw(s)
	}
	for i := 0; i < 2*mallListRows; i++ {
		index := m.Scroll*2 + i
		if index >= len(products) {
			break
		}
		p := products[index]
		dx, dy := MallProductRect(i)
		if index == m.Selected && m.blank != nil {
			s.Draw(m.blank, float32(x+dx), float32(y+dy), mallProductWide, 60, 0, 0, 1, 1, [4]float32{.65, .53, .3, .28})
		}
		if p.Icon != nil {
			s.Draw(p.Icon, float32(x+dx+4), float32(y+dy+6), 32, 32, p.UV.U0, p.UV.V0, p.UV.U1, p.UV.V1, white)
			p.Badge.Draw(s, x+dx+4, y+dy+6, 32, white)
		}
		lines := wrapWords(f, p.Name, mallProductWide-42, 2)
		for i, line := range lines {
			f.Draw(s, line, x+dx+40, y+dy+4+i*15, white, mallProductWide-42)
		}
		f.Draw(s, fmt.Sprintf("%d points", p.Cost), x+dx+40, y+dy+8+len(lines)*15, [4]float32{.95, .8, .45, 1}, mallProductWide-42)
	}
	for _, b := range []*Button{&m.buy, &m.gift} {
		b.OriginX, b.OriginY, b.Disabled = x, y, busy || m.Selected < 0 || m.Selected >= len(products)
		b.Draw(s)
	}
	for _, b := range []*Button{&m.giftBox, &m.cancel, &m.close} {
		b.OriginX, b.OriginY = x, y
		b.Draw(s)
	}
	last := max(0, (len(products)+1)/2-mallListRows)
	for i := range m.arrows {
		b := &m.arrows[i]
		b.OriginX, b.OriginY, b.Disabled = x, y, last == 0 || i == 0 && m.Scroll == 0 || i == 1 && m.Scroll >= last
		b.Draw(s)
	}
	thumb := mallListTop + 20
	if last > 0 {
		thumb += m.Scroll * mallTrack / last
	}
	s.Draw(m.scrollbar, float32(x+565+mallListShift), float32(y+thumb), 13, 8, .5, 0, .5+13.0/128, 8.0/64, white)
}

// Gift box uses six pages of 40 slots; only a double click or right click claims.
func (m *Mall) UpdateGifts(in *platform.Input, w, h int, stored map[uint8]proto.MallStoredItem, now float64, busy bool) (int, MallAction, bool) {
	x, y := m.GiftOrigin(w, h)
	m.giftClose.OriginX, m.giftClose.OriginY = x, y
	if m.giftClose.Update(in) || in.CloseOnEscape() {
		m.GiftOpen = false
		return -1, MallCloseGifts, true
	}
	tab := -1
	if in.MouseX >= x+10 && in.MouseX < x+322 && in.MouseY >= y+54 && in.MouseY < y+77 {
		tab = (in.MouseX - x - 10) / 52
	}
	hit := -1
	for i := 0; i < 40; i++ {
		dx, dy := MallGiftCell(i)
		if in.MouseX >= x+dx && in.MouseX < x+dx+32 && in.MouseY >= y+dy && in.MouseY < y+dy+32 {
			slot := m.GiftPage*40 + i
			if _, ok := stored[uint8(slot)]; ok {
				hit = slot
			}
			break
		}
	}
	if in.Pressed {
		m.heldGift, m.heldTab = hit, tab
	}
	claim := -1
	if in.RightPressed && hit >= 0 && !busy {
		claim = hit
	}
	if in.Clicked {
		if tab >= 0 && m.heldTab == tab {
			m.GiftPage, m.selectedGift = tab, -1
		}
		if hit >= 0 && m.heldGift == hit {
			if hit == m.lastGift && now >= m.lastGiftClick && now-m.lastGiftClick <= .35 && !busy {
				claim = hit
				m.lastGift = -1
			} else {
				m.selectedGift, m.lastGift, m.lastGiftClick = hit, hit, now
			}
		}
		m.heldGift, m.heldTab = -1, -1
	}
	inside := in.MouseX >= x && in.MouseX < x+GiftBoxWidth && in.MouseY >= y && in.MouseY < y+GiftBoxHeight
	return claim, MallNone, inside && (in.Pressed || in.Clicked || in.MouseDown || in.RightPressed || in.RightDown || in.Wheel != 0)
}
func (m *Mall) DrawGifts(s *render.Sprites, f *Font, w, h int, stored map[uint8]proto.MallStoredItem, icon func(proto.MallStoredItem) (*render.Texture, UV, bool), count func(proto.MallStoredItem) uint16, badges ...func(proto.MallStoredItem) ItemBadge) {
	x, y := m.GiftOrigin(w, h)
	s.Draw(m.giftFrame, float32(x), float32(y), GiftBoxWidth, GiftBoxHeight, 0, 0, float32(GiftBoxWidth)/512, float32(GiftBoxHeight)/512, white)
	if m.blank != nil {
		s.Draw(m.blank, float32(x+10+m.GiftPage*52), float32(y+54), 50, 22, 0, 0, 1, 1, [4]float32{.8, .65, .35, .25})
	}
	for i := 0; i < 40; i++ {
		slot := m.GiftPage*40 + i
		it, ok := stored[uint8(slot)]
		if !ok {
			continue
		}
		dx, dy := MallGiftCell(i)
		if slot == m.selectedGift && m.blank != nil {
			s.Draw(m.blank, float32(x+dx), float32(y+dy), 32, 32, 0, 0, 1, 1, [4]float32{.8, .65, .35, .4})
		}
		if tex, uv, ok := icon(it); ok && tex != nil {
			s.Draw(tex, float32(x+dx), float32(y+dy), 32, 32, uv.U0, uv.V0, uv.U1, uv.V1, white)
			if len(badges) > 0 {
				badges[0](it).Draw(s, x+dx, y+dy, 32, white)
			}
		}
		if n := count(it); n > 0 {
			f.Draw(s, fmt.Sprint(n), x+dx+12, y+dy+19, white, 20)
		}
	}
	f.Draw(s, "Right-click to claim", x+14, y+282, [4]float32{.8, .8, .8, 1}, 240)
	m.giftClose.OriginX, m.giftClose.OriginY = x, y
	m.giftClose.Draw(s)
}

// wrapWords splits text into at most lines rows of width pixels.
func wrapWords(f *Font, text string, width, lines int) []string {
	var out []string
	line := ""
	for _, word := range strings.Fields(text) {
		next := word
		if line != "" {
			next = line + " " + word
		}
		if line != "" && f.Width(next) > width && len(out) < lines-1 {
			out = append(out, line)
			line = word
		} else {
			line = next
		}
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}

// MallControlRects are the window areas that keep presses from dragging it:
// the product list, its scrollbar and the rotating preview.
func MallControlRects() [][4]int {
	px, py, pw, ph := MallPreviewRect()
	return [][4]int{{mallListLeft, mallListTop, 2 * mallColumn, mallListBottom - mallListTop}, {565 + mallListShift, mallListTop, 15, 340 - mallListTop + 19}, {px, py, pw, ph}}
}
