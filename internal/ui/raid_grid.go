package ui

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// The raid window: 5 columns x 6 rows of party2 cells, one per union slot.
// Constructor 0x5140d0 (cell 98x35), update 0x515f40, draw 0x5160f0; its
// default origin is (520, 425) and every sprite uses the UNIONPARTY
// ALPAH_VALUE alpha.
const (
	RaidGridWidth, RaidGridHeight = 490, 210
	RaidCellWidth, RaidCellHeight = 98, 35
	// Above the window: the page tabs flush with its left edge and the class
	// counter flush with its right edge.
	RaidTabX, RaidTabY, RaidTabStep    = 0, -26, 24
	RaidCounterCell, RaidCounterHeight = 32, 18
	RaidCounterWidth                   = 6 * RaidCounterCell
	RaidWindowWidth                    = RaidGridWidth
)

func (p *PartyPanel) RaidGridOrigin(w, h int) (int, int) {
	return p.RaidGridPosition.Origin(w, h, RaidWindowWidth, RaidGridHeight, 520, 425)
}

// RaidGridVisible is the "Raid Window On/Off" setting (DAT_021fd8c4).
func (p *PartyPanel) RaidGridVisible() bool { return p.raidCompact }

// RaidOpacity is the window transparency (ALPAH_VALUE).
func (p *PartyPanel) RaidOpacity() float32 { return p.raidOpacity }

// RaidTabAt is the page tab (0-4) under the pointer, or -1: the tabs are the
// same buttons as the party groups, drawn flush above the raid's top-left.
func (p *PartyPanel) RaidTabAt(w, h, mouseX, mouseY int) int {
	x, y := p.RaidGridOrigin(w, h)
	tx, ty := x+RaidTabX, y+RaidTabY
	if mouseX < tx || mouseX >= tx+proto.RaidPages*RaidTabStep || mouseY < ty+7 || mouseY >= y {
		return -1
	}
	return (mouseX - tx) / RaidTabStep
}

// RaidGridSlotAt is the absolute slot under the pointer (page*30 + column*6 +
// row), or -1.
func (p *PartyPanel) RaidGridSlotAt(w, h, mouseX, mouseY int) int {
	x, y := p.RaidGridOrigin(w, h)
	if mouseX < x || mouseX >= x+RaidGridWidth || mouseY < y || mouseY >= y+RaidGridHeight {
		return -1
	}
	return p.RaidPage*proto.RaidPageSlots + (mouseX-x)/RaidCellWidth*6 + (mouseY-y)/RaidCellHeight
}

// RaidGridInside covers the cells and the page tabs.
func (p *PartyPanel) RaidGridInside(w, h, mouseX, mouseY int) bool {
	x, y := p.RaidGridOrigin(w, h)
	if mouseX >= x && mouseX < x+RaidWindowWidth && mouseY >= y && mouseY < y+RaidGridHeight {
		return true
	}
	return p.RaidTabAt(w, h, mouseX, mouseY) >= 0
}

// UpdateRaidGrid selects the member whose cell is pressed and released and
// switches the viewed raid with the tabs.
func (p *PartyPanel) UpdateRaidGrid(in *platform.Input, w, h int, slots map[uint16]uint32) (uint32, bool) {
	id := uint32(0)
	if slot := p.RaidGridSlotAt(w, h, in.MouseX, in.MouseY); slot >= 0 {
		id = slots[uint16(slot)]
	}
	tab := p.RaidTabAt(w, h, in.MouseX, in.MouseY)
	if in.Pressed {
		p.raidGridHeld, p.raidGridTabHeld = id, tab+1
	}
	selected := uint32(0)
	if in.Clicked {
		if tab >= 0 && p.raidGridTabHeld == tab+1 {
			p.RaidPage = tab
		} else if p.raidGridHeld == id {
			selected = id
		}
		p.raidGridHeld, p.raidGridTabHeld = 0, 0
	}
	return selected, p.RaidGridInside(w, h, in.MouseX, in.MouseY) && (in.Pressed || in.Clicked || in.MouseDown || in.RightPressed || in.RightDown || in.Wheel != 0)
}

// RaidClassCounts counts the raid's members by class (0-5).
func RaidClassCounts(members []proto.PartyMember, slots map[uint16]uint32) (counts [6]int) {
	for _, id := range slots {
		for i := range members {
			if members[i].ID == id && id != 0 && members[i].Class < 6 {
				counts[members[i].Class]++
				break
			}
		}
	}
	return counts
}

// DrawRaidGrid draws the viewed raid's 30 cells with the tabs and the class
// counter. The selected member's cell (DAT_021fd808) is the red party2 row,
// the Move Group source (DAT_021fd80c) gets the orange outline and the last
// row has both. moveSlot is -1 when no move is armed.
func (p *PartyPanel) DrawRaidGrid(s *render.Sprites, f *Font, w, h int, members []proto.PartyMember, slots map[uint16]uint32, leader, sub, target uint32, moveSlot int) {
	x, y := p.RaidGridOrigin(w, h)
	alpha := p.raidOpacity
	if p.raidDraftReady {
		alpha = p.raidDraftOpacity // the slider previews live while the dialog is open
	}
	tint := [4]float32{1, 1, 1, alpha}
	for tab := 0; tab < proto.RaidPages; tab++ {
		v := float32(0)
		if tab == p.RaidPage {
			v = .25
		}
		s.Draw(p.raidSelector, float32(x+RaidTabX+tab*RaidTabStep), float32(y+RaidTabY), 32, 32, float32(tab)/8, v, float32(tab+1)/8, v+.25, tint) // follows the raid's transparency
	}
	base := p.RaidPage * proto.RaidPageSlots
	for i := 0; i < proto.RaidPageSlots; i++ {
		slot := base + i
		cx, cy := x+i/6*RaidCellWidth, y+i%6*RaidCellHeight

		id := slots[uint16(slot)]
		// party2 rows: normal, selected (red), Move Group source, both.
		row := 0
		if id != 0 && id == target {
			row = 1
		}
		if slot == moveSlot {
			row += 2
		}
		s.Draw(p.raidRows, float32(cx), float32(cy), 128, 64, 0, float32(row)*.25, 1, float32(row+1)*.25, tint)
		var m *proto.PartyMember
		for i := range members {
			if id != 0 && members[i].ID == id {
				m = &members[i]
				break
			}
		}
		if m == nil {
			continue
		}
		color := [4]float32{237. / 255, 237. / 255, 227. / 255, alpha}
		if m.MaxHP == 0 {
			color = [4]float32{164. / 255, 157. / 255, 157. / 255, alpha}
		}
		if m.MaxHP > 0 {
			s.Draw(p.graph, float32(cx+4), float32(cy+24), 90*float32(min(m.HP, m.MaxHP))/float32(m.MaxHP), 8, 0, 0, 0, .25, tint)
		}
		name := m.Name
		f.Draw(s, name, cx+max(0, (120-f.Width(name))/2), cy+8, color, 96)
		col := 0
		if m.Class < 6 {
			col = [...]int{2, 5, 1, 0, 3, 4}[m.Class]
		}
		s.Draw(p.icons, float32(cx+3), float32(cy+3), 22, 22, float32(col)/8, 0, float32(col+1)/8, 1, tint)
		if p.mark != nil && (id == leader || id == sub) {
			u := float32(0)
			if id != leader {
				u = .5
			}
			s.Draw(p.mark, float32(cx), float32(cy), 32, 32, u, 0, u+.5, 1, tint)
		}
	}

	// Class counter above the right edge, the mirror of the tabs: a small
	// icon and its number with a thin black outline over a semi-opaque
	// rectangle. It ignores the window transparency (a GX addition).
	counts := RaidClassCounts(members, slots)
	cx, cy := x+RaidGridWidth-RaidCounterWidth, y+RaidTabY+RaidCounterHeight-RaidCounterHeight+4
	if p.White != nil {
		s.Draw(p.White, float32(cx), float32(cy), RaidCounterWidth, RaidCounterHeight, 0, 0, 1, 1, [4]float32{0, 0, 0, .5})
	}
	for class := 0; class < 6; class++ {
		cellX := cx + class*RaidCounterCell
		col := [...]int{2, 5, 1, 0, 3, 4}[class]
		s.Draw(p.icons, float32(cellX+2), float32(cy+1), 16, 16, float32(col)/8, 0, float32(col+1)/8, 1, white)
		shade := [4]float32{.96, .96, .92, 1}
		if counts[class] == 0 {
			shade = [4]float32{.7, .68, .66, 1}
		}
		f.DrawOutlined(s, fmt.Sprint(counts[class]), cellX+20, cy+2, shade)
	}
}

// RaidViewSettings are the window options kept between sessions.
func (p *PartyPanel) RaidViewSettings() (opacity float32, compact, groups bool) {
	return p.raidOpacity, p.raidCompact, p.raidGroups
}

func (p *PartyPanel) SetRaidViewSettings(opacity float32, compact, groups bool) {
	p.raidOpacity, p.raidCompact, p.raidGroups = min(max(opacity, 0), 1), compact, groups
}
