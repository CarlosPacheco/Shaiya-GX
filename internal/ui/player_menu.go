package ui

import "github.com/Spelunkern/Shaiya-GX/internal/render"

// 0x47de10: text starts at (10,10), rows advance 15px. 0x47e1e0
// uses a 90px content width. Longer translated labels may widen it.
const PlayerMenuRowHeight = 15
const PlayerMenuTextInset = 10

type PlayerMenu struct{ frame *render.Texture }

func NewPlayerMenu(t Textures) (*PlayerMenu, error) {
	frame, err := t.Interface("TipBox.png")
	return &PlayerMenu{frame: frame}, err
}

func PlayerMenuSize(f *Font, labels []string) (int, int) {
	width := 98 // 90px native content plus the 4px borders.
	if f != nil {
		for _, label := range labels {
			width = max(width, f.Width(label)+20)
		}
	}
	return width, len(labels)*PlayerMenuRowHeight + 32
}

func PlayerMenuRowAt(mx, my, x, y, width, count int) int {
	if mx < x+4 || mx >= x+width-4 || my < y+PlayerMenuTextInset || my >= y+PlayerMenuTextInset+count*PlayerMenuRowHeight {
		return -1
	}
	return (my - y - PlayerMenuTextInset) / PlayerMenuRowHeight
}

// Classic branch of 0x50ff10. The optional RightMenu_* skin is absent
// from this archive; the original falls back to the TipBox atlas.
func (p *PlayerMenu) Draw(s *render.Sprites, f *Font, solid *render.Texture, x, y, width int, labels []string, hover int) {
	_, height := PlayerMenuSize(f, labels)
	DrawTipFrame(s, solid, float32(x), float32(y), float32(width), float32(height), 1, TipFill, TipBorder)
	for i, label := range labels {
		tint := white
		if i == hover {
			tint = RGBColor(0x0000ff)
		} // native packed 0xff0000ff
		f.Draw(s, label, x+PlayerMenuTextInset, y+PlayerMenuTextInset+i*PlayerMenuRowHeight, tint, 0)
	}
}
