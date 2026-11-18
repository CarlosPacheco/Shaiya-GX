package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const MessageNone, MessageOK, MessageCancel = 0, 1, 2

// Message uses the native type 0/1 dialogs: 0x4c4d10, 0x4c49d0, 0x4c4ae0.
// The caller owns modality and any action associated with confirmation.
type Message struct {
	extra            int
	detail           string
	detailColor      [4]float32
	frame            *render.Texture
	ok, cancel       Button
	Text             string
	Visible, Confirm bool
	// Movable lets the header drag the dialog. The position lasts for the
	// session and survives reopening; the native dialog is centred.
	Movable      bool
	Position     WindowPosition
	dragging     bool
	grabX, grabY int
}

const messageWidth, messageHeaderHeight = 256, 53

// Origin is the dialog's top-left corner.
func (m *Message) Origin(w, h int) (int, int) {
	x, y := w/2-128, h/2-128-m.extra/2
	if !m.Movable || !m.Position.Moved {
		return x, y
	}
	return m.Position.Origin(w, h, messageWidth, 256+m.extra, x, y)
}

// drag handles the header drag; it reports whether the pointer is owned by it.
func (m *Message) drag(in *platform.Input, w, h int) bool {
	if !m.Movable {
		return false
	}
	x, y := m.Origin(w, h)
	if m.dragging {
		m.Position.Move(in.MouseX-m.grabX, in.MouseY-m.grabY, w, h, messageWidth, 256+m.extra)
		if in.Clicked || !in.MouseDown {
			m.dragging = false
		}
		return true
	}
	if in.Pressed && in.MouseX >= x && in.MouseX < x+messageWidth && in.MouseY >= y && in.MouseY < y+messageHeaderHeight {
		m.dragging, m.grabX, m.grabY = true, in.MouseX-x, in.MouseY-y
		return true
	}
	return false
}

func NewMessage(t Textures) (*Message, error) {
	ld := &loader{t: t}
	m := &Message{frame: ld.get("message.png")}
	m.ok = Button{Tex: ld.get("ok_button.png"), DrawW: 128, DrawH: 32, HitW: 65, HitH: 25, UV: FourRowUV}
	m.cancel = Button{Tex: ld.get("cancel_button.png"), DrawW: 128, DrawH: 32, HitW: 65, HitH: 25, UV: FourRowUV}
	return m, ld.err
}

func (m *Message) Open(text string, confirm bool) {
	m.Text, m.Confirm, m.Visible = text, confirm, true
	m.dragging = false
	m.extra = 0
	m.detail = ""
	for _, b := range []*Button{&m.ok, &m.cancel} {
		b.held, b.state = false, StateNormal
	}
}

func (m *Message) controls(w, h int) {
	x, y := m.Origin(w, h)
	m.ok.OriginX, m.ok.OriginY, m.ok.OffX, m.ok.OffY = x, y, 95, 114+m.extra
	if m.Confirm {
		m.ok.OffX = 48
	}
	m.cancel.OriginX, m.cancel.OriginY, m.cancel.OffX, m.cancel.OffY = x, y, 143, 114+m.extra
}

func (m *Message) Update(in *platform.Input, w, h int) int {
	if !m.Visible {
		return MessageNone
	}
	if m.drag(in, w, h) {
		return MessageNone
	}
	m.controls(w, h)
	result := MessageNone
	if in.HasKey(platform.KeyEscape) || m.Confirm && m.cancel.Update(in) {
		result = MessageCancel
	} else if m.ok.Update(in) || in.HasKey(platform.KeyEnter) {
		result = MessageOK
	}
	if result != MessageNone {
		m.Visible = false
	}
	return result
}

func (m *Message) Draw(s *render.Sprites, f *Font, w, h int) {
	if !m.Visible {
		return
	}
	m.controls(w, h)
	x, y := m.Origin(w, h)
	if m.extra == 0 {
		s.DrawImage(m.frame, float32(x), float32(y))
	} else {
		s.Draw(m.frame, float32(x), float32(y), 256, 53, 0, 0, 1, 53.0/256, white)
		s.Draw(m.frame, float32(x), float32(y+53), 256, float32(58+m.extra), 0, 53.0/256, 1, 111.0/256, white)
		s.Draw(m.frame, float32(x), float32(y+111+m.extra), 256, 145, 0, 111.0/256, 1, 1, white)
	}
	lines := wrapLines(f, m.Text, 216)
	lines = lines[:min(4+m.extra/14, len(lines))]
	for i, line := range lines {
		f.Draw(s, line, x+25, y+53+i*14, white, 216)
	}
	if m.detail != "" {
		for i, line := range wrapLines(f, m.detail, 216) {
			f.Draw(s, line, x+25, y+53+(len(lines)+i)*14, m.detailColor, 216)
		}
	}
	m.ok.Draw(s)
	if m.Confirm {
		m.cancel.Draw(s)
	}
}

// OpenFull preserves all lines of native SysMsg confirmations.
func (m *Message) OpenFull(text string, confirm bool, f *Font) {
	m.Open(text, confirm)
	if f != nil {
		m.extra = max(0, len(wrapLines(f, text, 216))-4) * 14
	}
}

// OpenWithDetail appends a colored line to the classic confirmation dialog.
func (m *Message) OpenWithDetail(text, detail string, color [4]float32, f *Font) {
	m.Open(text, true)
	m.detail, m.detailColor = detail, color
	if f != nil {
		n := len(wrapLines(f, text, 216)) + len(wrapLines(f, detail, 216))
		m.extra = max(0, n-4) * 14
	}
}
