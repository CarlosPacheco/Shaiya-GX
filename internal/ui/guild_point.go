package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Guild ranking battle point window (Game.exe FUN_0049eae0 / 0049ee40): a
// 244x131 panel at (30,160), guild_point_alarm.tga drawn at 60% alpha, shown on
// the ranking map. The title is part of the texture.
const (
	GuildPointX, GuildPointY, GuildPointWidth, GuildPointHeight = 30, 160, 244, 131
)

type GuildPointWindow struct {
	frame *render.Texture
	close Button
}

func NewGuildPointWindow(t Textures) (*GuildPointWindow, error) {
	ld := &loader{t: t}
	w := &GuildPointWindow{frame: ld.get("guild_point_alarm.png")}
	w.close = nativeClose(ld.get("talk1_close_button.png"), 216, 5)
	w.close.OriginX, w.close.OriginY = GuildPointX, GuildPointY
	return w, ld.err
}

func (w *GuildPointWindow) Update(in *platform.Input) bool { return w.close.Update(in) }

// Draw shows the three lines: own points, the top guild and its points.
func (w *GuildPointWindow) Draw(s *render.Sprites, f *Font, labels [3]string, values [3]string) {
	x, y := GuildPointX, GuildPointY
	s.Draw(w.frame, float32(x), float32(y), GuildPointWidth, GuildPointHeight, 0, 0, float32(GuildPointWidth)/256, float32(GuildPointHeight)/256, [4]float32{1, 1, 1, 0x99 / 255.0})
	gold := [4]float32{0xc7 / 255.0, 0x91 / 255.0, 0x0a / 255.0, 1}
	for i, row := range [3]int{0x28, 0x46, 0x64} {
		f.Draw(s, labels[i], x+0xf, y+row, gold, 125)
		f.Draw(s, values[i], x+0x8c, y+row, white, 100)
	}
	w.close.Draw(s)
}
