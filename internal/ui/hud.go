package ui

import (
	"fmt"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// HUD uses the native status art/layout (0x509f10, 0x5097b0), target
// art/layout (0x50a2e0, 0x50b6a0).
type HUD struct {
	NumberFont                                                       *Font // 12px resource labels, separate from names
	Position, TargetPosition                                         WindowPosition
	status, bars, circle, portraitMask, target, targetBar, elemental *render.Texture
}

type Resource struct{ Current, Max uint32 }

type HUDState struct {
	ViewportW, ViewportH  int
	Name                  string
	Level                 uint16
	HP, MP, SP            Resource
	Exp, PrevExp, NextExp uint32
	BlessCount, BlessMax  uint32
	BlessKnown            bool
	BlessCountry          uint8
	BlessRemainingMS      uint32
	Clock                 float64
	TargetName            string
	TargetHP              Resource
	TargetKnown           bool
	TargetNoValue         bool // NPC targets: a full bar without numbers
	TargetColor           [4]float32
	TargetElement         uint8
	Portrait              *render.Texture
}

func NewHUD(t Textures) (*HUD, error) {
	h := &HUD{}
	for _, asset := range []struct {
		name string
		dst  **render.Texture
	}{
		{"main_stats_bar_bg.png", &h.status}, {"main_stats_bar.png", &h.bars},
		{"main_stats_circle.png", &h.circle}, {"enemy_bar_bg.png", &h.target},
		{"main_stats_circle2.png", &h.portraitMask},
		{"enemy_bar.png", &h.targetBar},
		{"enemy_elemental.png", &h.elemental},
	} {
		tex, err := t.Interface(asset.name)
		if err != nil {
			return nil, err
		}
		*asset.dst = tex
	}
	return h, nil
}

// progress clamps server values and avoids unsigned subtraction underflow.
func progress(value, start, end uint32) float32 {
	if end <= start || value <= start {
		return 0
	}
	if value >= end {
		return 1
	}
	return float32(float64(value-start) / float64(end-start))
}

func (h *HUD) Draw(s *render.Sprites, f *Font, state HUDState) {
	// Phoenix ep4_ui.gd::player: root (8,8), fixed label rectangles.
	x, y := h.Origin(state.ViewportW, state.ViewportH)
	tx, ty := h.TargetOrigin(state.ViewportW, state.ViewportH)
	s.DrawImage(h.status, float32(x), float32(y))
	for i, pool := range []Resource{state.HP, state.MP, state.SP} {
		n := progress(pool.Current, 0, pool.Max)
		// Native 0x5097b0: the 128px atlas includes its own leading padding.
		s.Draw(h.bars, float32(x+70), float32(y+40+16*i), 128*n, 16, 0, float32(i)*.25, n, float32(i+1)*.25, white)
	}
	s.DrawImage(h.circle, float32(x), float32(y))
	if state.Portrait != nil {
		s.DrawMasked(state.Portrait, h.portraitMask, float32(x+16), float32(y+26))
	}
	label := func(text string, left, top, width, height int, center bool) {
		if center {
			left += (width - f.Width(text)) / 2
		}
		top += (height - f.Height) / 2
		drawStatusText(s, f, text, float32(left), float32(top), 1, width)
	}
	// Centered over the bars (x+82, 114 wide), not left-aligned in the plate.
	label(state.Name, x+82, y+18, 114, 16, true) // +4: middle of the dark name plate
	label(fmt.Sprint(state.Level), x+62, y+15, 22, 20, true)
	for i, pool := range []Resource{state.HP, state.MP, state.SP} {
		text := fmt.Sprintf("%d/%d", min(pool.Current, pool.Max), pool.Max)
		numbers := h.NumberFont
		if numbers == nil {
			numbers = f
		}
		// 0x5097b0 uses five pixels per ASCII character to center the
		// rectangle; 0x509590 draws black then white one pixel up/left.
		left, top := x+135-len(text)*5/2, y+[]int{41, 58, 73}[i]
		drawResourceText(s, numbers, text, float32(left-1), float32(top-1), 1, 128)
	}

	if state.TargetName != "" {
		s.DrawImage(h.target, float32(tx), float32(ty+8))
		tint := state.TargetColor
		if tint[3] == 0 {
			tint = white
		}
		f.Draw(s, state.TargetName, tx+102-f.Width(state.TargetName)/2, ty+26, tint, 170)
		if uv, ok := ElementUV(state.TargetElement); ok {
			s.Draw(h.elemental, float32(tx), float32(ty+14), 32, 32, uv.U0, uv.V0, uv.U1, uv.V1, white)
		}
		if state.TargetKnown {
			n := progress(state.TargetHP.Current, 0, state.TargetHP.Max)
			s.Draw(h.targetBar, float32(tx+41), float32(ty+45), 140*n, 16, 0, 0, 140*n/256, 1, white)
			text := fmt.Sprintf("%d/%d", state.TargetHP.Current, state.TargetHP.Max)
			if state.TargetHP.Max == 0 {
				text = Trf(12370, state.TargetHP.Current)
			}
			// Centered on the 140px bar drawn at tx+41.
			if !state.TargetNoValue {
				f.Draw(s, text, tx+111-f.Width(text)/2, ty+43, white, 140)
			}
		}
	}
}

func (h *HUD) Origin(w, height int) (int, int) {
	if w <= 0 || height <= 0 {
		return 8, 8
	}
	return h.Position.Origin(w, height, 213, 114, 8, 8)
}
func (h *HUD) TargetOrigin(w, height int) (int, int) {
	if w <= 0 || height <= 0 {
		return 256, 0
	}
	return h.TargetPosition.Origin(w, height, 256, 64, 256, 0)
}

// DrawCast follows Game.exe 0x004a3020: a 5x64 black rectangle and a
// 3x62 blue elapsed-progress column, filling upwards per the requested UI.
// No texture, spell name or timer text.
func (h *HUD) DrawCast(s *render.Sprites, solid *render.Texture, x, y int, left, total float32) {
	fill, ok := castFillHeight(left, total)
	if !ok {
		return
	}
	drawProgressColumn(s, solid, x, y, fill, [4]float32{0, 0, 1, 1})
}

// DrawOxygen reuses the cast column with the remaining fraction filled.
func (h *HUD) DrawOxygen(s *render.Sprites, solid *render.Texture, x, y int, fraction float64) {
	fill := int(math.RoundToEven(min(1, max(0, fraction)) * 62))
	drawProgressColumn(s, solid, x, y, fill, oxygenColor)
}

// oxygenColor is the breath column: light cyan, so it never reads as the blue cast bar.
var oxygenColor = [4]float32{.35, .85, 1, 1}

func drawProgressColumn(s *render.Sprites, solid *render.Texture, x, y, fill int, color [4]float32) {
	s.Draw(solid, float32(x), float32(y), 5, 64, 0, 0, 1, 1, [4]float32{0, 0, 0, 1})
	s.Draw(solid, float32(x+1), float32(y+63-fill), 3, float32(fill), 0, 0, 1, 1, color)
}

func castFillHeight(left, total float32) (int, bool) {
	if left <= 0 || total <= 0 || math.IsNaN(float64(left)) || math.IsNaN(float64(total)) || math.IsInf(float64(left), 0) || math.IsInf(float64(total), 0) {
		return 0, false
	}
	return int(math.RoundToEven((1 - min(float64(left)/float64(total), 1)) * 62)), true
}
