package ui

import (
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type GuildCreateDialog struct {
	frame        *render.Texture
	noticeFrame  *render.Texture
	ok, cancel   Button
	Name, Remark TextField
	Visible      bool
	NoticeMode   bool
	focus        int
	// Anchor places the founding dialog at (+0xa0,+0x8c) of the guild window, as
	// 0x49a560 does; unset it is centred.
	Anchor func(w, h int) (int, int)
}

func NewGuildCreateDialog(t Textures) (*GuildCreateDialog, error) {
	ld := &loader{t: t}
	d := &GuildCreateDialog{frame: ld.get("guild_create_message.png"), noticeFrame: ld.get("guild_message_input.png"), Name: TextField{Max: 24}, Remark: TextField{Max: 64}}
	d.ok = Button{Tex: ld.get("ok_button.png"), OffX: 48, OffY: 210, DrawW: 128, DrawH: 32, HitW: 65, HitH: 25, UV: FourRowUV}
	d.cancel = Button{Tex: ld.get("cancel_button.png"), OffX: 143, OffY: 210, DrawW: 128, DrawH: 32, HitW: 65, HitH: 25, UV: FourRowUV}
	return d, ld.err
}
func (d *GuildCreateDialog) Open() {
	d.Visible = true
	d.NoticeMode = false
	d.focus = 0
	d.Name.Text = ""
	d.Remark.Text = ""
	d.ok.held = false
	d.cancel.held = false
}

// OpenNotice reuses the original guild text dialog; editing the notice does
// not permit changing the guild name.
func (d *GuildCreateDialog) OpenNotice(name, remark string) {
	d.Open()
	d.NoticeMode, d.focus = true, 1
	d.Name.Text, d.Remark.Text = name, remark
}
func (d *GuildCreateDialog) origin(w, h int) (int, int) {
	if d.NoticeMode {
		d.ok.OffX, d.ok.OffY = 45, 115
		d.cancel.OffX, d.cancel.OffY = 145, 115
		return w/2 - 128, h/2 - 78
	}
	d.ok.OffX, d.ok.OffY = 48, 210
	d.cancel.OffX, d.cancel.OffY = 143, 210
	if d.Anchor != nil {
		return d.Anchor(w, h)
	}
	return w/2 - 128, h/2 - 128
}
func (d *GuildCreateDialog) Update(in *platform.Input, w, h int) int {
	if !d.Visible {
		return MessageNone
	}
	x, y := d.origin(w, h)
	d.ok.OriginX, d.ok.OriginY = x, y
	d.cancel.OriginX, d.cancel.OriginY = x, y
	if in.Pressed && in.MouseX >= x+24 && in.MouseX < x+236 {
		if !d.NoticeMode && in.MouseY >= y+60 && in.MouseY < y+92 {
			d.focus = 0
		}
		if in.MouseY >= y+128 && in.MouseY < y+200 {
			d.focus = 1
		}
	}
	if !d.NoticeMode && in.HasKey(platform.KeyTab) {
		d.focus = 1 - d.focus
	}
	if d.focus == 0 {
		d.Name.Edit(in)
	} else {
		d.Remark.Edit(in)
	}
	d.ok.Disabled = !d.NoticeMode && len(strings.TrimSpace(d.Name.Text)) < 2
	if in.HasKey(platform.KeyEscape) || d.cancel.Update(in) {
		d.Visible = false
		return MessageCancel
	}
	if d.ok.Update(in) || in.HasKey(platform.KeyEnter) && !d.ok.Disabled {
		d.Visible = false
		return MessageOK
	}
	return MessageNone
}
func (d *GuildCreateDialog) Draw(s *render.Sprites, f *Font, w, h int) {
	if !d.Visible {
		return
	}
	x, y := d.origin(w, h)
	frame := d.frame
	if d.NoticeMode {
		frame = d.noticeFrame
	}
	s.DrawImage(frame, float32(x), float32(y))
	name, remark := d.Name.Text, d.Remark.Text
	if d.focus == 0 {
		name += "_"
	} else {
		remark += "_"
	}
	if !d.NoticeMode {
		f.Draw(s, name, x+0x28, y+0x46, white, 0) // 0x49a560: name at (+0x28,+0x46)
	}
	if d.NoticeMode {
		for i, line := range wrapLines(f, remark, 200) {
			if i >= 4 {
				break
			}
			f.Draw(s, line, x+28, y+46+i*14, white, 200)
		}
	} else {
		// The message is split once at 200 px: lines at (+0x23,+0x8a) and (+0x23,+0x99).
		for i, line := range wrapLines(f, remark, 200) {
			if i >= 2 {
				break
			}
			f.Draw(s, line, x+0x23, y+0x8a+i*0xf, white, 0)
		}
	}
	d.ok.OriginX, d.ok.OriginY = x, y
	d.cancel.OriginX, d.cancel.OriginY = x, y
	d.ok.Disabled = !d.NoticeMode && len(strings.TrimSpace(d.Name.Text)) < 2
	d.ok.Draw(s)
	d.cancel.Draw(s)
}
