package app

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
		a.sprites.Draw(a.white, float32(x-6), float32(y-6), float32(width/2*2+8), 12, 0, 0, 1, 1, [4]float32{29.0 / 255, 31.0 / 255, 36.0 / 255, 205.0 / 255})
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
