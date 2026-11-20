package ui

import "github.com/Spelunkern/Shaiya-GX/internal/render"

// The raid properties panel (0x513bc0, built by 0x514900): 125x45 at (90, 100)
// with the message2 fill fading in to 50% while the pointer is over it and two
// lines of text at (+10, +10) and (+10, +25), white at 70%.
const RaidInfoWidth, RaidInfoHeight = 125, 45

func (p *PartyPanel) RaidInfoOrigin(w, h int) (int, int) {
	return p.RaidInfoPosition.Origin(w, h, RaidInfoWidth, RaidInfoHeight, 90, 100)
}

func (p *PartyPanel) RaidInfoInside(w, h, mouseX, mouseY int) bool {
	x, y := p.RaidInfoOrigin(w, h)
	return mouseX >= x && mouseX < x+RaidInfoWidth && mouseY >= y && mouseY < y+RaidInfoHeight
}

// DrawRaidInfo draws the panel; dt moves the hover fade (0 to .5 in half a second).
func (p *PartyPanel) DrawRaidInfo(s *render.Sprites, f *Font, w, h int, lines [2]string, mouseX, mouseY int, dt float32) {
	x, y := p.RaidInfoOrigin(w, h)
	if p.RaidInfoInside(w, h, mouseX, mouseY) {
		p.raidInfoFade = min(p.raidInfoFade+dt, .5)
	} else {
		p.raidInfoFade = max(p.raidInfoFade-dt, 0)
	}
	if p.infoFill != nil && p.raidInfoFade > 0 {
		s.Draw(p.infoFill, float32(x), float32(y), RaidInfoWidth, RaidInfoHeight, 0, .3, .5, .5, [4]float32{1, 1, 1, p.raidInfoFade})
	}
	for i, line := range lines {
		f.Draw(s, line, x+10, y+10+i*15, [4]float32{1, 1, 1, 178. / 255}, RaidInfoWidth-14)
	}
}

// RaidViewSettings are the window options kept between sessions.
func (p *PartyPanel) RaidViewSettings() (opacity float32, compact, groups bool) {
	return p.raidOpacity, p.raidCompact, p.raidGroups
}

func (p *PartyPanel) SetRaidViewSettings(opacity float32, compact, groups bool) {
	p.raidOpacity, p.raidCompact, p.raidGroups = min(max(opacity, 0), 1), compact, groups
}
