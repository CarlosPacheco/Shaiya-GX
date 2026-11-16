package ui

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
)

const QuickPotionWidth, QuickPotionHeight = 94, 54

func QuickPotionOrigin(p WindowPosition, w, h int) (int, int) {
	return p.Origin(w, h, 94, 54, (w-448)/2-8-94, h-64-8-54)
}
func QuickPotionCell(mx, my, x, y int) int {
	for i := 0; i < 2; i++ {
		if mx >= x+11+i*40 && mx < x+43+i*40 && my >= y+13 && my < y+45 {
			return i
		}
	}
	return -1
}
func DrawQuickPotions(s *render.Sprites, f *Font, frame, blank *render.Texture, x, y int, slots [2]ActionSlot) {
	if frame != nil {
		s.Draw(frame, float32(x), float32(y), 94, 54, 0, 0, 94.0/float32(frame.W), 54.0/float32(frame.H), white)
	}
	for i, slot := range slots {
		cx, cy := x+11+i*40, y+13
		// Native 0x4ddc30 draws shortcuts before item icons (filled cells cover them).
		f.DrawOutlined(s, []string{"Z", "X"}[i], x+30+i*40, y+32, white)
		if slot.Icon != nil {
			u := slot.UV
			tint := white
			if slot.Count == 0 {
				tint = [4]float32{.45, .45, .45, .65}
			}
			s.Draw(slot.Icon, float32(cx), float32(cy), 32, 32, u.U0, u.V0, u.U1, u.V1, tint)
		}
		if slot.Cooldown > 0 {
			ratio := float32(1)
			if slot.CooldownDuration > 0 {
				ratio = float32(slot.Cooldown / slot.CooldownDuration)
			}
			s.DrawCooldown(blank, float32(cx), float32(cy), 32, ratio, [4]float32{1, 0, 0, 80.0 / 255})
			label := fmt.Sprintf("%.0f", math.Ceil(slot.Cooldown))
			f.DrawOutlined(s, label, cx+16-f.Width(label)/2, cy+9, [4]float32{1, 1, 0, 1})
		}
		if slot.Count > 0 {
			// Native 0x4a3400 item-count anchor, called by the potion bar.
			f.DrawOutlined(s, fmt.Sprint(slot.Count), cx+12, cy+16, white)
		}
	}
}
