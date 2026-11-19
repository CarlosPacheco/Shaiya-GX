package app

import (
	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type screenNotice struct {
	text  string
	style uint8
	at    float64
}

func nativeNoticeStyle(style uint8) bool {
	switch style {
	case 8, 24, 27, 28, 29, 30, 32, 33:
		return true
	}
	return false
}

// Native 0041f6b0 keeps at most five entries independently of chat history.
// 004bfb90 expires them after ten seconds; 004bf670 draws centered 17px rows.
func (a *App) addScreenNotice(style uint8, text string) {
	a.screenNotices = append(a.screenNotices, screenNotice{text, style, a.clock})
	if len(a.screenNotices) > 5 {
		a.screenNotices = append(a.screenNotices[:0], a.screenNotices[len(a.screenNotices)-5:]...)
	}
}
func (a *App) drawScreenNotices(w, h int) {
	rows := a.screenNotices[:0]
	for _, n := range a.screenNotices {
		if a.clock-n.at <= 10 {
			rows = append(rows, n)
		}
	}
	a.screenNotices = rows
	// Native English locale 3 uses regular 15px Arial at actor +0x3a8,
	// the same font as chat (00544620 / 00541aa0), not the 13px tooltip.
	f := a.chatTextFont
	if f == nil {
		f = a.font
	}
	if f == nil {
		return
	}
	for i, n := range rows {
		width := f.Width(n.text)
		x, y := w/2-width/2, h/2-150+i*17
		// The native box (x-6, y-6, width+8, 12) is two pixels larger here so the
		// 15px text sits inside it.
		a.drawNewsPanel(x-8, y-8, width/2*2+12, 16)
		color := white
		if n.style == 30 || n.style == 32 {
			color = [4]float32{1, 128.0 / 255, 1, 1}
		}
		f.DrawOutlined(a.sprites, n.text, x, y, color)
	}
}

func (a *App) playNewsSound() {
	if a.sound != nil && a.arc != nil {
		name := a.audioPath("ui_news001.wav", false)
		a.audioError(name, a.sound.Play(name, 1))
	}
}

// Native 0050ff10: width/height describe the interior, with 4px atlas
// borders. The center selects the diffuse color instead of the texture.
func (a *App) drawNewsPanel(x, y, width, height int) {
	if a.assets == nil {
		return
	}
	frame := a.tipFrame()
	if frame == nil {
		return
	}
	draw := func(px, py, w, h int, u0, v0, u1, v1 float32) {
		a.sprites.Draw(frame, float32(px), float32(py), float32(w), float32(h), u0, v0, u1, v1, white)
	}
	draw(x, y, 4, 4, 0, 0, .25, .25)
	draw(x+4, y, width, 4, .25, 0, .5, .25)
	draw(x+width+4, y, 4, 4, .5, 0, .75, .25)
	draw(x, y+4, 4, height+4, .25, .25, .5, .5)
	draw(x+width+4, y+4, 4, height+4, .75, .25, 1, .5)
	a.sprites.Draw(a.white, float32(x+4), float32(y+4), float32(width), float32(height+4), 0, 0, 1, 1, [4]float32{29.0 / 255, 31.0 / 255, 36.0 / 255, 205.0 / 255})
	draw(x, y+height+8, 4, 4, 0, .25, .25, .5)
	draw(x+4, y+height+8, width, 4, .75, 0, 1, .25)
	draw(x+width+4, y+height+8, 4, 4, .5, .25, .75, .5)
}

// tipFrame is TipBox.TGA with the colour of its transparent texels set to the
// border colour: the art stores white there, which bilinear sampling would blend
// into the panel edges as a white fringe.
func (a *App) tipFrame() *render.Texture {
	if a.tipTexture != nil || a.arc == nil {
		return a.tipTexture
	}
	b, err := a.arc.ReadFile("interface/tipbox.png")
	if err != nil {
		return nil
	}
	t, err := img.DecodeResource("interface/tipbox.png", b)
	if err != nil || t.RGBA == nil {
		return nil
	}
	pix := t.RGBA.Pix
	for i := 0; i+3 < len(pix); i += 4 {
		if pix[i+3] == 0 {
			pix[i], pix[i+1], pix[i+2] = 26, 26, 26
		}
	}
	a.tipTexture = render.Upload(t)
	return a.tipTexture
}
