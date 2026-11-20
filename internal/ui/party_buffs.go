package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// PartyBuff is one effect icon on a party/raid row. Skip keeps the gap of an
// entry the original could not resolve (0x50abd0 advances its slot anyway).
type PartyBuff struct {
	Tex          *render.Texture
	UV           UV
	Skill        uint16
	Level        uint8
	Debuff, Skip bool
}

// PartyBuffs is the effect strip of 0x50abd0: 16x16 icons 19px apart from
// (row x+146, y+7); the sixth and later wrap to y+26 starting at x+146 again.
// A debuff (level 100 or TypeEffect 3) is covered by the flashing overlay.
type PartyBuffs struct {
	Of      func(m proto.PartyMember) []PartyBuff
	MouseX  int
	MouseY  int
	Flash   float32
	Hover   PartyBuff
	Hovered bool
}

const (
	partyBuffX, partyBuffY, partyBuffY2 = 146, 7, 26
	partyBuffSize, partyBuffStep        = 16, 19
)

func partyBuffPos(x, rowY, i int) (int, int) {
	if i < 5 {
		return x + partyBuffX + partyBuffStep*i, rowY + partyBuffY
	}
	return x + partyBuffX + partyBuffStep*(i-5), rowY + partyBuffY2
}

func (b *PartyBuffs) draw(s *render.Sprites, white *render.Texture, x, rowY int, m proto.PartyMember, opacity float32) {
	if b == nil || b.Of == nil {
		return
	}
	for i, e := range b.Of(m) {
		if e.Skip {
			continue
		}
		bx, by := partyBuffPos(x, rowY, i)
		if e.Tex != nil {
			s.Draw(e.Tex, float32(bx), float32(by), partyBuffSize, partyBuffSize, e.UV.U0, e.UV.V0, e.UV.U1, e.UV.V1, [4]float32{1, 1, 1, opacity})
		}
		if e.Debuff && white != nil {
			s.Draw(white, float32(bx), float32(by), partyBuffSize, partyBuffSize, 0, 0, 1, 1, [4]float32{1, 1, 1, b.Flash * opacity})
		}
		if b.MouseX >= bx && b.MouseX <= bx+partyBuffSize && b.MouseY >= by && b.MouseY <= by+partyBuffSize {
			b.Hover, b.Hovered = e, true
		}
	}
}
