package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const (
	DeathSingle = iota
	DeathParty
	DeathWaiting
)

// DeathDialog uses the original message frame and the button layouts from
// 0x4c4d10 (types 2, 3/6 and 7). The caller owns the server-side choice.
type DeathDialog struct {
	frame, singleTex, okTex, partyTex *render.Texture
	left, right                       Button
	kind                              int
	Text                              string
}

func NewDeathDialog(t Textures) (*DeathDialog, error) {
	l := &loader{t: t}
	d := &DeathDialog{frame: l.get("message.png"), singleTex: l.get("message_button.png"), okTex: l.get("ok_button.png"), partyTex: l.get("message_button2.png"), kind: -1}
	return d, l.err
}
func (d *DeathDialog) Reset() { d.kind = -1 }
func (d *DeathDialog) Set(kind int, text string) {
	d.Text = text
	if d.kind == kind {
		return
	}
	d.kind = kind
	d.left = Button{DrawW: 128, DrawH: 32, HitW: 65, HitH: 25, OffY: 114, UV: FourRowUV}
	d.right = Button{DrawW: 128, DrawH: 32, HitW: 95, HitH: 25, OffY: 114, UV: FourRowUV}
	switch kind {
	case DeathSingle:
		d.left.Tex = d.singleTex
		d.left.OffX = 95
	case DeathParty:
		d.left.Tex = d.okTex
		d.left.OffX = 38
		d.right.Tex = d.partyTex
		d.right.OffX = 128

	}
}
func (d *DeathDialog) position(w, h int) {
	x, y := w/2-128, h/2-128
	d.left.OriginX, d.left.OriginY = x, y
	d.right.OriginX, d.right.OriginY = x, y
}
func (d *DeathDialog) Update(in *platform.Input, w, h int) int {
	d.position(w, h)
	if d.kind == DeathWaiting {
		return 0
	}
	if d.left.Update(in) {
		return 1
	}
	if d.kind == DeathParty && d.right.Update(in) {
		return 2
	}
	return 0
}
func (d *DeathDialog) Draw(s *render.Sprites, f *Font, w, h int) {
	d.position(w, h)
	x, y := w/2-128, h/2-128
	s.DrawImage(d.frame, float32(x), float32(y))
	lines := wrapLines(f, d.Text, 216)
	for i, line := range lines[:min(len(lines), 4)] {
		f.Draw(s, line, x+25, y+53+i*14, white, 216)
	}
	if d.kind != DeathWaiting {
		d.left.Draw(s)
	}
	if d.kind == DeathParty {
		d.right.Draw(s)
	}
}
