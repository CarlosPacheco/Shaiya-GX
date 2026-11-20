package ui

import (
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
)

func (p *PartyPanel) RaidGridOrigin(w, h int) (int, int) {
	return p.RaidGridPosition.Origin(w, h, RaidGridWidth, RaidGridHeight, 520, 425)
}

// RaidGridVisible is the "Raid Window On/Off" setting (DAT_021fd8c4).
func (p *PartyPanel) RaidGridVisible() bool { return p.raidCompact }

// RaidOpacity is the window transparency (ALPAH_VALUE).
func (p *PartyPanel) RaidOpacity() float32 { return p.raidOpacity }

// RaidGridSlotAt is the slot (column*6 + row) under the pointer, or -1.
func (p *PartyPanel) RaidGridSlotAt(w, h, mouseX, mouseY int) int {
	x, y := p.RaidGridOrigin(w, h)
	if mouseX < x || mouseX >= x+RaidGridWidth || mouseY < y || mouseY >= y+RaidGridHeight {
		return -1
	}
	return (mouseX-x)/RaidCellWidth*6 + (mouseY-y)/RaidCellHeight
}

// UpdateRaidGrid selects the member whose cell is pressed and released.
func (p *PartyPanel) UpdateRaidGrid(in *platform.Input, w, h int, slots map[uint16]uint32) (uint32, bool) {
	id := uint32(0)
	if slot := p.RaidGridSlotAt(w, h, in.MouseX, in.MouseY); slot >= 0 {
		id = slots[uint16(slot)]
	}
	if in.Pressed {
		p.raidGridHeld = id
	}
	selected := uint32(0)
	if in.Clicked {
		if p.raidGridHeld == id {
			selected = id
		}
		p.raidGridHeld = 0
	}
	x, y := p.RaidGridOrigin(w, h)
	inside := in.MouseX >= x && in.MouseX < x+RaidGridWidth && in.MouseY >= y && in.MouseY < y+RaidGridHeight
	return selected, inside && (in.Pressed || in.Clicked || in.MouseDown || in.RightPressed || in.RightDown || in.Wheel != 0)
}

// DrawRaidGrid draws all 30 cells. The selected member's cell (DAT_021fd808)
// is the red party2 row, the Move Group source (DAT_021fd80c) gets the orange
// outline and the last row has both. moveSlot is -1 when no move is armed.
func (p *PartyPanel) DrawRaidGrid(s *render.Sprites, f *Font, w, h int, members []proto.PartyMember, slots map[uint16]uint32, leader, sub, target uint32, moveSlot int) {
	x, y := p.RaidGridOrigin(w, h)
	alpha := p.raidOpacity
	tint := [4]float32{1, 1, 1, alpha}
	for slot := 0; slot < 30; slot++ {
		cx, cy := x+slot/6*RaidCellWidth, y+slot%6*RaidCellHeight
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
}
