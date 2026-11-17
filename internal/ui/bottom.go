package ui

import (
	"fmt"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

type BottomAction uint8

const (
	BottomNone BottomAction = iota
	BottomInventory
	BottomSkills
	BottomStatus
	BottomGuild
	BottomSubActions
	BottomMall
	BottomOptions
	BottomQuests
)

// BottomBar uses the 800/1024 art, button layout and EXP pass from
// 0x483610/0x485517. Other resolutions keep this art unscaled and centered.
type BottomBar struct {
	frames, bars, sheets [2]*render.Texture
	buttons              [8]Button
	hover                int
	expHover             bool
	blessActive          bool
	blessStartedAt       float64
	QuestOpen, GuildOpen bool
}

func NewBottomBar(t Textures) (*BottomBar, error) {
	b := &BottomBar{hover: -1}
	for i, suffix := range []string{"8", ""} {
		for _, asset := range []struct {
			name string
			dst  **render.Texture
		}{{"main_bottom", &b.frames[i]}, {"main_bottom_bar", &b.bars[i]}, {"main_bottom_button", &b.sheets[i]}} {
			tex, err := t.Interface(asset.name + suffix + ".png")
			if err != nil {
				return nil, err
			}
			*asset.dst = tex
		}
	}
	return b, nil
}

type bottomGeometry struct {
	variant, x, y, width                 int
	buttonX, buttonY, pitch, hitH, drawW int
}

func bottomLayout(w, h int) bottomGeometry {
	g := bottomGeometry{1, (w - 1024) / 2, h - 64, 1024, 646, 10, 35, 32, 64}
	if w < 1024 {
		g = bottomGeometry{0, (w - 800) / 2, h - 64, 800, 505, 22, 27, 23, 32}
	}
	return g
}

func (b *BottomBar) controls(w, h int, inventory, skills, status, subActions bool) {
	g := bottomLayout(w, h)
	for i := range b.buttons {
		button := &b.buttons[i]
		button.Tex, button.OriginX, button.OriginY = b.sheets[g.variant], g.x, g.y
		button.OffX, button.OffY = g.buttonX+i*g.pitch, g.buttonY
		button.HitW, button.HitH = g.pitch, g.hitH
		if g.variant == 0 {
			button.HitW = 25
		}
		button.DrawW, button.DrawH = g.drawW, 32
		button.Toggle = true
		button.Checked = i == 1 && status || i == 2 && inventory || i == 3 && skills || i == 4 && subActions || i == 5 && b.QuestOpen || i == 6 && b.GuildOpen
		button.Disabled = false
		for state, row := range []int{0, 1, 2, 2, 3} {
			button.UV[state] = UV{float32(i) / 8, float32(row) / 4, float32(i+1) / 8, float32(row+1) / 4}
		}
	}
}

// UpdateInventoryButton lets the chat composer keep inventory accessible.
func (b *BottomBar) UpdateInventoryButton(in *platform.Input, w, h int, open bool) bool {
	b.controls(w, h, open, false, false, false)
	return b.buttons[2].Update(in)
}

func (b *BottomBar) Update(in *platform.Input, w, h int, inventory, skills, status, subActions bool) (BottomAction, bool) {
	b.controls(w, h, inventory, skills, status, subActions)
	b.hover = -1
	action, held := BottomNone, false
	for i := range b.buttons {
		button := &b.buttons[i]
		if in.Pressed {
			button.held = false
		}
		held = held || button.held
		if button.hit(in) {
			b.hover = i
		}
		if button.Update(in) {
			if i == 0 {
				action = BottomMall
			} else if i == 1 {
				action = BottomStatus
			} else if i == 2 {
				action = BottomInventory
			} else if i == 3 {
				action = BottomSkills
			} else if i == 4 {
				action = BottomSubActions
			} else if i == 5 {
				action = BottomQuests
			} else if i == 6 {
				action = BottomGuild
			} else if i == 7 {
				action = BottomOptions
			}
		}
	}
	g := bottomLayout(w, h)
	x0, y0, x1, y1 := 21, 47, 453, 59
	if g.variant == 0 {
		x0, y0, x1, y1 = 16, 50, 353, 60
	}
	b.expHover = in.MouseX >= g.x+x0 && in.MouseX < g.x+x1 && in.MouseY >= g.y+y0 && in.MouseY < g.y+y1
	over := in.MouseX >= g.x && in.MouseX < g.x+g.width && in.MouseY >= g.y && in.MouseY < h
	captured := over && (in.Pressed || in.Clicked || in.MouseDown || in.RightPressed || in.RightDown || in.Wheel != 0)
	return action, captured || held && in.Clicked
}

func blessingLayout(variant int) (x, y, width int) {
	if variant == 0 {
		return 446, 53, 336
	}
	return 572, 49, 428
}

func (b *BottomBar) Draw(s *render.Sprites, f *Font, blank *render.Texture, w, h int, state HUDState, inventory, skills, status, subActions bool) {
	b.controls(w, h, inventory, skills, status, subActions)
	g := bottomLayout(w, h)
	x, y := g.x, g.y
	s.DrawImage(b.frames[g.variant], float32(x), float32(y))
	ex, ey, textX, uvEnd := 23, 49, 222, float32(.3)
	if g.variant == 0 {
		ex, ey, textX, uvEnd = 19, 53, 170, .4
	}
	n := progress(state.Exp, state.PrevExp, state.NextExp)
	s.Draw(b.bars[g.variant], float32(x+ex), float32(y+ey), 512*n, 7, 0, 0, 1, uvEnd, white)
	f.Draw(s, fmt.Sprintf("%.1f%%", n*100), x+textX, y+49, white, 70)
	active := state.BlessKnown && state.BlessRemainingMS > 0
	if active && !b.blessActive {
		b.blessStartedAt = state.Clock
	}
	b.blessActive = active
	if state.BlessKnown && state.BlessMax > 0 {
		bx, by, width := blessingLayout(g.variant)
		value := progress(state.BlessCount, 0, state.BlessMax)
		if state.BlessRemainingMS > 0 {
			value = progress(state.BlessRemainingMS, 0, 600000)
		}
		filled := int(float32(width) * value)
		// Original 00485680: crop the same right-anchored atlas region,
		// not a stretched bar. No percentage label is drawn over blessing.
		draw := func(alpha float32, v1 float32) {
			s.Draw(b.bars[g.variant], float32(x+bx+width-filled), float32(y+by), float32(filled), 7,
				float32(width-filled)/512, .5, float32(width)/512, v1, [4]float32{1, 1, 1, alpha})
		}
		draw(1, .9)
		if state.BlessRemainingMS > 0 {
			// Native 00864e2c cycles 2 -> 0 at two units per second;
			// alpha is triangular, with an additive second atlas pass.
			phase := math.Mod(max(0, state.Clock-b.blessStartedAt)*2, 2)
			alpha := float32(1 - math.Abs(1-phase))
			v1 := float32(.9)
			if g.variant == 0 {
				v1 = 1
			}
			s.Flush()
			gl.BlendFunc(gl.SRC_ALPHA, gl.ONE)
			draw(alpha, v1)
			s.Flush()
			gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		}
	}
	for i := range b.buttons {
		b.buttons[i].Draw(s)
	}
}

// BlessingAt uses the native footer hit box, including its transparent margin.
func (b *BottomBar) BlessingAt(mx, my, w, h int) bool {
	g := bottomLayout(w, h)
	x0, y0, x1, y1 := 570, 47, 1002, 59
	if g.variant == 0 {
		x0, y0, x1, y1 = 445, 50, 783, 60
	}
	return mx >= g.x+x0 && mx < g.x+x1 && my >= g.y+y0 && my < g.y+y1
}
