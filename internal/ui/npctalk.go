package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// NPC conversation window (Game.exe FUN_0050fc50 / 0050f210 / 0050f4b0 /
// 0050faa0 / 0050f960): talk1_bg, the NPC portrait and name, the greeting in a
// scrolling text box, a service button on the left and the quest "talk" button
// on the right, both talk_blank_button with the label drawn over them.
const NPCTalkWidth, NPCTalkHeight = 250, 382

type NPCTalk struct {
	Position              WindowPosition
	frame                 *render.Texture
	close, service, quest Button
	scroll                panelScroll
	offset, lines         int
}

func NewNPCTalk(t Textures) (*NPCTalk, error) {
	ld := &loader{t: t}
	v := &NPCTalk{frame: ld.get("talk1_bg.png")}
	v.close = nativeClose(ld.get("talk1_close_button.png"), 228, 5)
	blank := ld.get("talk_blank_button.png")
	v.service = Button{Tex: blank, OffX: 53, OffY: 336, DrawW: 128, DrawH: 32, HitW: 65, HitH: 26, UV: npcHorizontalUV(0)}
	v.quest = Button{Tex: blank, OffX: 145, OffY: 336, DrawW: 128, DrawH: 32, HitW: 65, HitH: 26, UV: npcHorizontalUV(1)}
	v.scroll = newPanelScroll(ld.get("scrollbar.png"), 227, 82, 225)
	return v, ld.err
}

// Origin centres the window like the original (screen/2 - 0x7d, screen/2 - 0xbf).
func (v *NPCTalk) Origin(w, h int) (int, int) {
	return v.Position.Origin(w, h, NPCTalkWidth, NPCTalkHeight, max(0, w/2-0x7d), max(0, h/2-0xbf))
}

func (v *NPCTalk) Reset() { v.offset = 0 }

const npcTalkLines = 12

// Update returns the close, service and quest clicks.
func (v *NPCTalk) Update(in *platform.Input, w, h int, canQuest bool) (closed, service, quest bool) {
	x, y := v.Origin(w, h)
	for _, b := range []*Button{&v.close, &v.service, &v.quest} {
		b.OriginX, b.OriginY = x, y
	}
	v.quest.Disabled = !canQuest
	if in.MouseX >= x+16 && in.MouseX < x+243 && in.MouseY >= y+82 && in.MouseY < y+307 && in.Wheel != 0 {
		if in.Wheel < 0 {
			v.offset += 3
		} else {
			v.offset = max(0, v.offset-3)
		}
	}
	v.offset = v.scroll.Update(in, x, y, v.lines, npcTalkLines, v.offset)
	return v.close.Update(in), v.service.Update(in), v.quest.Update(in)
}

func (v *NPCTalk) Draw(s *render.Sprites, f *Font, w, h int, name, greeting, serviceLabel, questLabel string, canQuest bool) {
	x, y := v.Origin(w, h)
	s.Draw(v.frame, float32(x), float32(y), 256, 391, 0, 0, 1, 391.0/512, white)
	f.Draw(s, name, x+0x41+(0xf9-0x41-f.Width(name))/2, y+0x16, white, 0)
	lines := wrapLines(f, greeting, 200)
	v.lines = len(lines)
	v.offset = min(v.offset, max(0, len(lines)-npcTalkLines))
	for i, line := range lines[v.offset:min(len(lines), v.offset+npcTalkLines)] {
		f.Draw(s, line, x+23, y+89+i*15, white, 200)
	}
	if len(lines) > npcTalkLines {
		v.scroll.Draw(s, x, y, len(lines), npcTalkLines, v.offset)
	}
	for _, b := range []*Button{&v.close, &v.service, &v.quest} {
		b.OriginX, b.OriginY = x, y
	}
	v.quest.Disabled = !canQuest
	v.service.Draw(s)
	v.quest.Draw(s)
	label := func(b *Button, text string, enabled bool) {
		tint := white
		if !enabled {
			tint = [4]float32{.5, .5, .5, 1}
		}
		f.Draw(s, text, b.X()+(b.HitW-f.Width(text))/2, b.Y()+6, tint, b.HitW)
	}
	label(&v.service, serviceLabel, true)
	label(&v.quest, questLabel, canQuest)
	v.close.Draw(s)
}
