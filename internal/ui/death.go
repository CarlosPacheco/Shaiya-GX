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
	// The header drags the dialog; the position lasts for the session.
	Position     WindowPosition
	dragging     bool
	grabX, grabY int
}

// Origin is the dialog's top-left corner (centred until it is moved).
func (d *DeathDialog) Origin(w, h int) (int, int) {
	x, y := w/2-128, h/2-128
	if !d.Position.Moved {
		return x, y
	}
	return d.Position.Origin(w, h, messageWidth, 256, x, y)
}

// drag handles the header drag; it reports whether the pointer is owned by it.
func (d *DeathDialog) drag(in *platform.Input, w, h int) bool {
	x, y := d.Origin(w, h)
	if d.dragging {
		d.Position.Move(in.MouseX-d.grabX, in.MouseY-d.grabY, w, h, messageWidth, 256)
		if in.Clicked || !in.MouseDown {
			d.dragging = false
		}
		return true
	}
	if in.Pressed && in.MouseX >= x && in.MouseX < x+messageWidth && in.MouseY >= y && in.MouseY < y+messageHeaderHeight {
		d.dragging, d.grabX, d.grabY = true, in.MouseX-x, in.MouseY-y
		return true
	}
	return false
}

func NewDeathDialog(t Textures) (*DeathDialog, error) {
	l := &loader{t: t}
	d := &DeathDialog{frame: l.get("message.png"), singleTex: l.get("message_button.png"), okTex: l.get("ok_button.png"), partyTex: l.get("message_button2.png"), kind: -1}
	return d, l.err
}
func (d *DeathDialog) Reset() { d.kind, d.dragging = -1, false }
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
	x, y := d.Origin(w, h)
	d.left.OriginX, d.left.OriginY = x, y
	d.right.OriginX, d.right.OriginY = x, y
}
func (d *DeathDialog) Update(in *platform.Input, w, h int) int {
	if d.drag(in, w, h) {
		return 0
	}
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
	x, y := d.Origin(w, h)
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
