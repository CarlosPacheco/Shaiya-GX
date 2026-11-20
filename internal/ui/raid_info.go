package ui

import "github.com/Spelunkern/Shaiya-GX/internal/render"

// The raid properties panel (0x513bc0, built by 0x514900): 125x45 at (90, 100)
// with the message2 fill and lines of text from (+10, +10), 15px apart.
const RaidInfoWidth, RaidInfoHeight = 125, 58

func (p *PartyPanel) RaidInfoOrigin(w, h int) (int, int) {
	return p.RaidInfoPosition.Origin(w, h, RaidInfoWidth, RaidInfoHeight, 90, 100)
}

func (p *PartyPanel) RaidInfoInside(w, h, mouseX, mouseY int) bool {
	x, y := p.RaidInfoOrigin(w, h)
	return mouseX >= x && mouseX < x+RaidInfoWidth && mouseY >= y && mouseY < y+RaidInfoHeight
}

// DrawRaidInfo draws the panel; dt moves the hover fade (0 to .5 in half a second).
func (p *PartyPanel) DrawRaidInfo(s *render.Sprites, f *Font, w, h int, lines [3]string, mouseX, mouseY int, dt float32) {
	x, y := p.RaidInfoOrigin(w, h)
	if p.RaidInfoInside(w, h, mouseX, mouseY) {
		p.raidInfoFade = min(p.raidInfoFade+dt, .5)
	} else {
		p.raidInfoFade = max(p.raidInfoFade-dt, 0)
	}
	// GX makes the panel easier to read: the fill is always there (55%) and
	// grows to 85% under the pointer; the text is fully opaque.
	if p.infoFill != nil {
		s.Draw(p.infoFill, float32(x), float32(y), RaidInfoWidth, RaidInfoHeight, 0, .3, .5, .3+float32(RaidInfoHeight)/256, [4]float32{1, 1, 1, .55 + p.raidInfoFade*.6})
	}
	for i, line := range lines {
		f.Draw(s, line, x+10, y+10+i*15, [4]float32{1, 1, 1, 1}, RaidInfoWidth-14)
	}
}

// RaidViewSettings are the window options kept between sessions.
func (p *PartyPanel) RaidViewSettings() (opacity float32, compact, groups bool) {
	return p.raidOpacity, p.raidCompact, p.raidGroups
}

func (p *PartyPanel) SetRaidViewSettings(opacity float32, compact, groups bool) {
	p.raidOpacity, p.raidCompact, p.raidGroups = min(max(opacity, 0), 1), compact, groups
}
