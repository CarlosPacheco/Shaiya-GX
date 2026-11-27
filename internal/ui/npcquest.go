package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const NPCQuestWidth = 256
const NPCQuestRows = 6

type NPCQuest struct {
	hoverRow                int
	storyScroll, listScroll panelScroll
	textLines               int
	// Rewards are the quest's reward rows (Shaiya Core style, not in EP5.4):
	// up to three, each with an item icon drawn by DrawRewardIcon. When
	// RewardSelectable is set a click picks RewardSelected.
	Rewards                     []NPCQuestReward
	RewardSelectable            bool
	RewardSelected              int
	DrawRewardIcon              func(index, x, y int)
	rewardHover                 int
	Detail                      bool
	Position                    WindowPosition
	HasService, ActionDisabled  bool
	list, take, complete        *render.Texture
	close, action, finish, back Button
	serviceQuest, serviceReturn Button
	held                        int
	textOffset                  int
}

// shortText reports whether the description shares the page with rewards.
func (v *NPCQuest) shortText(ending bool) bool { return ending || len(v.Rewards) > 0 }

func (v *NPCQuest) ResetDescriptionScroll() { v.textOffset = 0 }

type NPCQuestAction struct {
	Row, Offset         int
	Close, Back, Accept bool
	// Reward is the reward row clicked this frame, or -1.
	Reward int
}

// NPCQuestReward is one reward row: the item name (empty for no item) and
// the experience/gold line.
type NPCQuestReward struct {
	Item, Detail string
	Icon         bool
}

// Reward rows below the shortened description.
const (
	npcQuestRewardTop    = 306
	npcQuestRewardHeight = 40
)

func (v *NPCQuest) rewardAt(in *platform.Input, x, y int) int {
	top := y + npcQuestRewardTop
	if in.MouseX < x+20 || in.MouseX >= x+236 || in.MouseY < top || in.MouseY >= top+len(v.Rewards)*npcQuestRewardHeight {
		return -1
	}
	return (in.MouseY - top) / npcQuestRewardHeight
}

func NewNPCQuest(t Textures) (*NPCQuest, error) {
	ld := &loader{t: t}
	v := &NPCQuest{RewardSelected: -1, rewardHover: -1, list: ld.get("talk1_bg.png"), take: ld.get("quest_take.png"), complete: ld.get("quest_complete.png"), held: -1, hoverRow: -1}
	v.storyScroll = newPanelScroll(ld.get("scrollbar.png"), 227, 114, 329)
	v.listScroll = newPanelScroll(ld.get("scrollbar.png"), 227, 142, 166)
	v.close = nativeClose(ld.get("talk1_close_button.png"), 228, 5)
	v.action = Button{Tex: ld.get("quest_take_button.png"), OffX: 53, DrawW: 128, DrawH: 32, HitW: 67, HitH: 27, UV: npcVerticalUV}
	v.finish = Button{Tex: ld.get("talk_button.png"), OffX: 53, DrawW: 128, DrawH: 32, HitW: 65, HitH: 26, UV: npcHorizontalUV(0)}
	v.serviceQuest = Button{Tex: ld.get("talk_button.png"), DrawW: 128, DrawH: 32, HitW: 65, HitH: 26, UV: npcHorizontalUV(1)}
	v.serviceReturn = Button{Tex: ld.get("talk_blank_button.png"), DrawW: 128, DrawH: 32, HitW: 65, HitH: 26, UV: v.finish.UV}
	v.back = Button{Tex: ld.get("cancel_button.png"), OffX: 145, DrawW: 128, DrawH: 32, HitW: 67, HitH: 27, UV: npcVerticalUV}
	return v, ld.err
}

// ServiceButton switches between a service and the NPC's quest conversation.
func (v *NPCQuest) ServiceButton(in *platform.Input, x, y int, returning bool) bool {
	b := &v.serviceQuest
	if returning {
		b = &v.serviceReturn
	}
	b.OriginX, b.OriginY = x, y
	return b.Update(in)
}
func (v *NPCQuest) DrawServiceButton(s *render.Sprites, f *Font, x, y int, returning bool, label string) {
	b := &v.serviceQuest
	if returning {
		b = &v.serviceReturn
	}
	b.OriginX, b.OriginY = x, y
	b.Draw(s)
	if returning {
		f.Draw(s, label, x+8, y+6, white, 54)
	}
}
func NPCQuestOrigin(w, h int) (int, int) {
	return min(32, max(0, w-NPCQuestWidth)), min(82, max(0, h-512))
}
func (v *NPCQuest) Origin(w, h int) (int, int) {
	x, y := MarketOrigin(w, h)
	height := 391
	if v.Detail {
		height = 512
	}
	return v.Position.Origin(w, h, NPCQuestWidth, height, x, y)
}
func (v *NPCQuest) Update(in *platform.Input, w, h, count, offset int, detail, ending bool) NPCQuestAction {
	v.Detail = detail
	x, y := v.Origin(w, h)
	height := 391
	if detail {
		height = 512
	}
	v.close.OriginX, v.close.OriginY = x, y
	v.action.OriginX, v.action.OriginY = x, y
	v.finish.OriginX, v.finish.OriginY = x, y
	v.back.OriginX, v.back.OriginY = x, y
	v.action.OffY = height - 46
	v.finish.OffY = height - 46
	if !detail {

		v.finish.OffY = 336
	}
	v.action.Disabled, v.finish.Disabled = v.ActionDisabled, detail && v.ActionDisabled
	v.back.OffY = height - 46
	out := NPCQuestAction{Row: -1, Reward: -1, Offset: max(0, min(offset, max(0, count-NPCQuestRows)))}
	out.Close = v.close.Update(in)
	if detail {
		short := v.shortText(ending)
		bottom := y + 437
		if short {
			bottom = y + 297
		}
		if in.MouseX >= x+20 && in.MouseX < x+235 && in.MouseY >= y+110 && in.MouseY < bottom && in.Wheel != 0 {
			if in.Wheel < 0 {
				v.textOffset += 3
			} else {
				v.textOffset = max(0, v.textOffset-3)
			}
		}
		visible := 23
		scrollHeight := 329
		if short {
			visible = 13
			scrollHeight = 183
		}
		v.storyScroll.height = scrollHeight
		v.storyScroll.arrows[1].OffY = 114 + scrollHeight - 19
		if v.textLines > 0 {
			v.textOffset = v.storyScroll.Update(in, x, y, v.textLines, visible, v.textOffset)
		}
		v.rewardHover = v.rewardAt(in, x, y)
		if v.RewardSelectable {
			if in.Pressed {
				v.held = v.rewardHover
			}
			if in.Clicked {
				if v.rewardHover >= 0 && v.rewardHover == v.held {
					v.RewardSelected = v.rewardHover
					out.Reward = v.rewardHover
				}
				v.held = -1
			}
		}
		if ending {
			out.Accept = v.finish.Update(in)
		} else {
			out.Accept = v.action.Update(in)
		}
		if !v.HasService {
			out.Back = v.back.Update(in)
		}
		return out
	}
	if v.finish.Update(in) {
		out.Close = true
	}
	if count > 0 && !v.HasService {
		v.serviceQuest.OriginX, v.serviceQuest.OriginY = x+145, y+336
		if v.serviceQuest.Update(in) {
			out.Row = out.Offset
		}
	}
	if in.MouseX >= x+16 && in.MouseX < x+240 && in.MouseY >= y+142 && in.MouseY < y+310 && in.Wheel != 0 {
		d := 1
		if in.Wheel < 0 {
			d = -1
		}
		out.Offset = max(0, min(max(0, count-NPCQuestRows), out.Offset-d))
	}
	out.Offset = v.listScroll.Update(in, x, y, count, NPCQuestRows, out.Offset)
	row := -1
	if in.MouseX >= x+18 && in.MouseX < x+238 && in.MouseY >= y+142 && in.MouseY < y+142+NPCQuestRows*26 {
		row = (in.MouseY - y - 142) / 26
	}
	v.hoverRow = row
	if in.Pressed {
		v.held = row
	}
	if in.Clicked {
		if row >= 0 && row == v.held && out.Offset+row < count {
			out.Row = out.Offset + row
		}
		v.held = -1
	}
	return out
}
func (v *NPCQuest) Draw(s *render.Sprites, f *Font, w, h int, name, greeting string, portrait *render.Texture, entries []string, offset int, detail, ending bool, title, description string) {
	v.Detail = detail
	x, y := v.Origin(w, h)
	height := 391
	frame := v.list
	if detail {
		height = 512
		frame = v.take
		if ending {
			frame = v.complete
		}
	}
	s.Draw(frame, float32(x), float32(y), 256, float32(height), 0, 0, 1, float32(height)/512, white)
	if portrait != nil {
		s.Draw(portrait, float32(x+7), float32(y+7), 64, 64, 0, 0, 1, 1, white)
	}
	if !detail {
		f.Draw(s, Tr(12367), x+26, y+89, RGBColor(0xeee4ce), 196)
		v.finish.OriginX, v.finish.OriginY = x, y
		v.finish.OffY = 336
		v.finish.Disabled = false
		v.finish.Draw(s)
		if len(entries) > 0 && !v.HasService {
			v.serviceQuest.OriginX, v.serviceQuest.OriginY = x+145, y+336
			v.serviceQuest.Draw(s)
		}
		for i := 0; i < NPCQuestRows && offset+i < len(entries); i++ {
			tint := RGBColor(0xeee4ce)
			if i == v.hoverRow {
				tint = RGBColor(0xffd67f)
			}
			f.Draw(s, entries[offset+i], x+26, y+146+i*26, tint, 196)
		}
		if len(entries) > NPCQuestRows {
			v.listScroll.Draw(s, x, y, len(entries), NPCQuestRows, offset)
		}
	} else {
		f.Draw(s, title, x+25, y+86, white, 205)
		lines := wrapLines(f, description, 196)
		short := v.shortText(ending)
		limit := 23
		if short {
			limit = 13
		}
		v.textLines = len(lines)
		v.textOffset = min(v.textOffset, max(0, len(lines)-limit))
		for i, line := range lines[v.textOffset:min(len(lines), v.textOffset+limit)] {
			f.Draw(s, line, x+25, y+115+i*14, white, 205)
		}
		scrollHeight := 329
		if short {
			scrollHeight = 183
		}
		v.storyScroll.height = scrollHeight
		v.storyScroll.arrows[1].OffY = 114 + scrollHeight - 19
		if len(lines) > limit {
			v.storyScroll.Draw(s, x, y, len(lines), limit, v.textOffset)
		}
		for i, r := range v.Rewards {
			ry := y + npcQuestRewardTop + i*npcQuestRewardHeight
			tint := RGBColor(0xeee4ce)
			switch {
			case v.RewardSelectable && i == v.RewardSelected:
				tint = RGBColor(0xffd67f)
			case v.RewardSelectable && i == v.rewardHover:
				tint = white
			}
			tx := x + 26
			if r.Icon && v.DrawRewardIcon != nil {
				v.DrawRewardIcon(i, x+24, ry+2)
				tx = x + 62
			}
			if r.Item != "" {
				f.Draw(s, r.Item, tx, ry+4, tint, x+236-tx)
				f.Draw(s, r.Detail, tx, ry+19, tint, x+236-tx)
			} else {
				f.Draw(s, r.Detail, tx, ry+11, tint, x+236-tx)
			}
		}
		v.action.OriginX, v.action.OriginY = x, y
		v.finish.OriginX, v.finish.OriginY = x, y
		v.back.OriginX, v.back.OriginY = x, y
		v.action.OffY = height - 46
		v.finish.OffY = height - 46
		v.action.Disabled, v.finish.Disabled = v.ActionDisabled, v.ActionDisabled
		v.back.OffY = height - 46
		if !ending {
			v.action.Draw(s)
		} else {
			v.finish.Draw(s)
		}
		if !v.HasService {
			v.back.Draw(s)
		}
	}
	v.close.OriginX, v.close.OriginY = x, y
	v.close.Draw(s)
}

// RewardHover is the reward row under the pointer, or -1.
func (v *NPCQuest) RewardHover() int { return v.rewardHover }
