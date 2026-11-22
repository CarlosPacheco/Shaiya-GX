package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Appearance window constants, Game.exe 0x463680 (constructor) and 0x4625f0
// (layout): a 256x512 ch_face.tga frame at (w/12, h/2-300). The frame already
// carries every label; the window draws no text of its own.
const (
	AppearanceWidth, AppearanceHeight = 256, 512
	// Preview clip, 0x463480: x+15..x+240, down to y+227.
	AppearanceViewX, AppearanceViewTop, AppearanceViewRight, AppearanceViewBottom = 15, 30, 240, 227
)

// AppearanceEditor edits hair, face, height and (for the sex stone) sex.
type AppearanceEditor struct {
	frame      *render.Texture
	arrows     [6]Button // hair, face, height: previous/next
	sexes      [2]Button
	ok, cancel Button
	close      Button
	Values     [4]uint8
	// Hair and face style counts from the body's MLT files; the app refreshes
	// them whenever the sex changes. Height keeps the creation screen's five.
	Counts        [2]int
	Position      WindowPosition
	GenderEnabled bool
}

func AppearanceOrigin(w, h int) (int, int) { return w / 12, h/2 - 300 }

// Origin follows the native default until the player drags the window.
func (e *AppearanceEditor) Origin(w, h int) (int, int) {
	x, y := AppearanceOrigin(w, h)
	return e.Position.Origin(w, h, AppearanceWidth, AppearanceHeight, x, y)
}

func NewAppearanceEditor(t Textures) (*AppearanceEditor, error) {
	l := &loader{t: t}
	e := &AppearanceEditor{frame: l.get("ch_face.png")}
	tex := l.get("market_button.png")
	for i := range e.arrows {
		side, row := i%2, i/2
		b := &e.arrows[i]
		// 0x4625f0: 32px art, 23px hit, x 0x61/0xbd, y 0x13f/0x160/0x182.
		*b = Button{Tex: tex, OffX: 0x61 + side*(0xbd-0x61), OffY: []int{0x13f, 0x160, 0x182}[row], DrawW: 32, DrawH: 32, HitW: 23, HitH: 23}
		for state, v := range []int{0, 1, 2, 2, 3} {
			b.UV[state] = UV{float32(side) / 2, float32(v) / 4, float32(side+1) / 2, float32(v+1) / 4}
		}
	}
	sex := l.get("create_sex_button.png")
	for i := range e.sexes {
		e.sexes[i] = creationLabel(sex, 70+i*78, 273, i, 2)
	}
	e.ok = Button{Tex: l.get("ok_button.png"), OffX: 0x25, OffY: 0x1bf, DrawW: 128, DrawH: 32, HitW: 0x43, HitH: 0x1b, UV: FourRowUV}
	e.cancel = Button{Tex: l.get("cancel_button.png"), OffX: 0x96, OffY: 0x1bf, DrawW: 128, DrawH: 32, HitW: 0x43, HitH: 0x1b, UV: FourRowUV}
	e.close = nativeClose(l.get("talk1_close_button.png"), 229, 1)
	return e, l.err
}

func (e *AppearanceEditor) Open(values [4]uint8, genderEnabled bool) {
	e.Values, e.GenderEnabled = values, genderEnabled
	for _, b := range e.buttons() {
		b.held, b.state = false, StateNormal
	}
}

func (e *AppearanceEditor) buttons() []*Button {
	out := []*Button{&e.ok, &e.cancel, &e.close}
	for i := range e.arrows {
		out = append(out, &e.arrows[i])
	}
	for i := range e.sexes {
		out = append(out, &e.sexes[i])
	}
	return out
}

func (e *AppearanceEditor) place(w, h int) {
	x, y := e.Origin(w, h)
	for _, b := range e.buttons() {
		b.OriginX, b.OriginY = x, y
	}
	for i := range e.sexes {
		e.sexes[i].Disabled = !e.GenderEnabled
		e.sexes[i].Checked = e.Values[3] == uint8(i)
	}
}

// Contains reports whether the pointer is over the window, which owns it.
func (e *AppearanceEditor) Contains(mx, my, w, h int) bool {
	x, y := e.Origin(w, h)
	return mx >= x && mx < x+AppearanceWidth && my >= y && my < y+AppearanceHeight
}

// Update returns MessageOK when the confirm button is pressed (the app asks
// "Do you want to proceed?" before sending) and MessageCancel for cancel,
// the close button or Escape.
func (e *AppearanceEditor) Update(in *platform.Input, w, h int) int {
	e.place(w, h)
	for i := range e.arrows {
		if e.arrows[i].Update(in) {
			n := 5
			if i/2 < 2 && e.Counts[i/2] > 0 {
				n = e.Counts[i/2]
			}
			e.Values[i/2] = uint8((int(e.Values[i/2]) + n - 1 + i%2*2) % n)
		}
	}
	for i := range e.sexes {
		if e.sexes[i].Update(in) {
			e.Values[3] = uint8(i)
		}
	}
	if in.HasKey(platform.KeyEscape) || e.cancel.Update(in) || e.close.Update(in) {
		return MessageCancel
	}
	if e.ok.Update(in) {
		return MessageOK
	}
	return MessageNone
}

func (e *AppearanceEditor) Draw(s *render.Sprites, f *Font, w, h int) {
	e.place(w, h)
	x, y := e.Origin(w, h)
	s.Draw(e.frame, float32(x), float32(y), AppearanceWidth, AppearanceHeight, 0, 0, 1, 1, white)
	for _, b := range e.buttons() {
		b.Draw(s)
	}
}
