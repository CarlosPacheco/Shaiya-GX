package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// The Guild Manager (Guild Master NPC inside a guild house map, Game.exe
// FUN_0049e4b0/0049de50/0049d050) and the Etin Manager (merchant type 100,
// FUN_00483210/004820e0/00482880) share one two-pane layout: the left pane
// holds the NPC portrait, name, a text box and the tab buttons, the right pane
// one of the pages below. Coordinates are from those functions.
const (
	GuildHouseWidth, GuildHouseHeight = 586, 489

	HouseModeList    = 0 // Etin Manager: guild list (guild_manager_list)
	HouseModeManager = 1 // Guild Manager: upgrade services (guild_manager_service)
	HouseModeInfo    = 3 // service information (guild_manager_service_info)
	HouseModeBuy     = 4 // Etin Manager: buy the guild house (guild_manager_buy_house)

	houseRowStep = 43
)

// HouseKind selects the NPC window.
type HouseKind int

const (
	HouseGuildManager HouseKind = iota
	HouseEtinManager
)

// HouseRow is one service line of the Manager and Info pages.
type HouseRow struct {
	Show                     bool
	ShowUpgrade              bool // Manager page: the service can still be upgraded (texts and button)
	UpgradeOn                bool // the click is accepted (rank, upkeep and level checks)
	Icon                     int  // icon_skill cell (32 px, 16 per row)
	IconDim                  bool
	Title, RankText, FeeText string
	RankOK, FeeOK            bool
}

// HouseListRow is one line of the Etin Manager's guild list.
type HouseListRow struct{ Rank, Name, Master, Points string }

// HouseView is what the window draws.
type HouseView struct {
	Portrait             *render.Texture
	NpcName, Title, Text string
	Etin, Available      string
	Total                string
	Rows                 []HouseRow
	Selected             int
	List                 []HouseListRow
	ListOffset           int
	CanManage            bool // Guild Manager: the Manager tab is enabled (guild rank 1-2)
	CanBuy               bool
	HouseOwned           bool // the guild already owns its house: the Buy tab and button are greyed
}

// HouseAction is what the player did this frame; -1 means nothing.
type HouseAction struct {
	Close, Buy, Donate bool
	Tab                int
	Select, Upgrade    int
	Scroll             int
}

// GuildHouseWindow draws and hit-tests both house NPC windows.
type GuildHouseWindow struct {
	Position    WindowPosition
	frames      map[int]*render.Texture
	labels      *render.Texture
	icons       *render.Texture
	skill       *render.Texture
	close       Button
	tabs        [4]Button
	buy         Button
	configured  bool
	kind        HouseKind
	heldUpgrade int
	hover       int  // upgrade button under the pointer
	down        bool // primary button held
}

// button atlas rows (guild_manager_button.png, 128x32 cells, 4 state columns).
const (
	houseLabelInfo = iota
	houseLabelManager
	houseLabelGuildInfo
	houseLabelServiceInfo
	houseLabelBuyHouse
	houseLabelEtin
)

func houseButton(tex *render.Texture, x, y, row int) Button {
	b := Button{Tex: tex, OffX: x, OffY: y, DrawW: 128, DrawH: 32, HitW: 90, HitH: 27}
	for state, col := range []int{0, 1, 2, 2, 3} {
		b.UV[state] = UV{float32(col) / 4, float32(row) / 8, float32(col+1) / 4, float32(row+1) / 8}
	}
	return b
}

func NewGuildHouseWindow(t Textures) (*GuildHouseWindow, error) {
	ld := &loader{t: t}
	w := &GuildHouseWindow{frames: map[int]*render.Texture{
		HouseModeList:    ld.get("guild_manager_list.png"),
		HouseModeManager: ld.get("guild_manager_service.png"),
		HouseModeInfo:    ld.get("guild_manager_service_info.png"),
		HouseModeBuy:     ld.get("guild_manager_buy_house.png"),
	}, heldUpgrade: -1}
	w.labels = ld.get("guild_manager_button.png")
	w.icons = ld.get("icon/icon_skill.png")
	w.skill = ld.get("skill_button.png")
	w.close = nativeClose(ld.get("talk1_close_button.png"), 557, 7)
	w.buy = houseButton(w.labels, 371, 387, houseLabelBuyHouse)
	return w, ld.err
}

// Origin centres the window like the originals (screen/2 - 0x125, screen/2 - 0xfa).
func (w *GuildHouseWindow) Origin(sw, sh int) (int, int) {
	return w.Position.Origin(sw, sh, GuildHouseWidth, GuildHouseHeight, max(0, sw/2-0x125), max(0, sh/2-0xfa))
}

// configure positions the tab buttons of a kind.
func (w *GuildHouseWindow) configure(kind HouseKind) {
	// Rebuilding the buttons every frame would drop their pressed state, so a
	// click (press on one frame, release on another) could never complete.
	if w.configured && w.kind == kind {
		return
	}
	w.configured, w.kind = true, kind
	if kind == HouseEtinManager {
		w.tabs[0] = houseButton(w.labels, 26, 289, houseLabelBuyHouse)
		w.tabs[1] = houseButton(w.labels, 140, 289, houseLabelServiceInfo)
		w.tabs[2] = houseButton(w.labels, 26, 327, houseLabelInfo)
		w.tabs[3] = houseButton(w.labels, 140, 327, houseLabelEtin)
		return
	}
	w.tabs[0] = houseButton(w.labels, 26, 308, houseLabelServiceInfo)
	w.tabs[1] = houseButton(w.labels, 140, 308, houseLabelManager)
}

func (w *GuildHouseWindow) tabCount(kind HouseKind) int {
	if kind == HouseEtinManager {
		return 4
	}
	return 2
}

// tabMode is the page a tab opens (-1 for the Etin donation button).
func tabMode(kind HouseKind, i int) int {
	if kind == HouseEtinManager {
		return [4]int{HouseModeBuy, HouseModeInfo, HouseModeList, -1}[i]
	}
	return [2]int{HouseModeInfo, HouseModeManager}[i]
}

func houseTabDisabled(kind HouseKind, i int, v *HouseView) bool {
	if kind == HouseGuildManager {
		return tabMode(kind, i) == HouseModeManager && !v.CanManage
	}
	return tabMode(kind, i) == HouseModeBuy && v.HouseOwned
}

// rowY is the top of service row k: Info rows start at 0x5b, Manager rows at 0x56.
func rowY(mode, k int) int {
	if mode == HouseModeManager {
		return 0x56 + k*houseRowStep
	}
	return 0x5b + k*houseRowStep
}

// Update handles input for the current page.
func (w *GuildHouseWindow) Update(in *platform.Input, sw, sh int, kind HouseKind, mode int, v *HouseView) HouseAction {
	act := HouseAction{Tab: -1, Select: -1, Upgrade: -1}
	w.hover, w.down = -1, in.MouseDown
	x, y := w.Origin(sw, sh)
	w.configure(kind)
	w.close.OriginX, w.close.OriginY = x, y
	act.Close = w.close.Update(in)
	for i := 0; i < w.tabCount(kind); i++ {
		b := &w.tabs[i]
		b.OriginX, b.OriginY = x, y
		b.Disabled = houseTabDisabled(kind, i, v)
		if b.Update(in) {
			if m := tabMode(kind, i); m < 0 {
				act.Donate = true
			} else {
				act.Tab = m
			}
		}
	}
	if mode == HouseModeBuy {
		w.buy.OriginX, w.buy.OriginY, w.buy.Disabled = x, y, !v.CanBuy
		act.Buy = w.buy.Update(in)
	}
	switch mode {
	case HouseModeInfo, HouseModeManager:
		k := 0
		for i, r := range v.Rows {
			if !r.Show {
				continue
			}
			top := y + rowY(mode, k)
			if mode == HouseModeManager && r.ShowUpgrade && in.MouseX >= x+0x201 && in.MouseX <= x+0x217 && in.MouseY >= top && in.MouseY <= top+0x16 {
				w.hover = i
				if in.Pressed {
					w.heldUpgrade = i
				}
				if in.Clicked && w.heldUpgrade == i && r.UpgradeOn {
					act.Upgrade = i
				}
			} else if mode == HouseModeInfo && in.Clicked && in.MouseX >= x+0x10e && in.MouseX <= x+0x235 && in.MouseY >= top-6 && in.MouseY <= top+0x25 {
				act.Select = i
			}
			k++
		}
		if in.Clicked {
			w.heldUpgrade = -1
		}
	case HouseModeList:
		if in.Wheel != 0 && in.MouseX >= x+270 && in.MouseX < x+575 && in.MouseY >= y+91 && in.MouseY < y+440 {
			act.Scroll = 1
			if in.Wheel > 0 {
				act.Scroll = -1
			}
		}
	}
	return act
}

var houseRed = [4]float32{1, 50.0 / 255, 50.0 / 255, 1}

func houseTint(ok bool) [4]float32 {
	if ok {
		return white
	}
	return houseRed
}

// Draw renders the window for a kind and page.
func (w *GuildHouseWindow) Draw(s *render.Sprites, f *Font, blank *render.Texture, sw, sh int, kind HouseKind, mode int, v *HouseView) {
	x, y := w.Origin(sw, sh)
	w.configure(kind)
	if frame := w.frames[mode]; frame != nil {
		s.Draw(frame, float32(x), float32(y), GuildHouseWidth, GuildHouseHeight, 0, 0, float32(GuildHouseWidth)/1024, float32(GuildHouseHeight)/512, white)
	}
	if v.Portrait != nil {
		s.Draw(v.Portrait, float32(x+7), float32(y+7), 64, 64, 0, 0, 1, 1, white)
	}
	// Left pane: the NPC name and the window's title are centred at x+0x86.
	if v.NpcName != "" {
		f.Draw(s, v.NpcName, x+0x86-f.Width(v.NpcName)/2, y+0x20, white, 0)
	}
	if v.Title != "" {
		f.Draw(s, v.Title, x+0x86-f.Width(v.Title)/2, y+0x36, white, 0)
	}
	// 0x52cf20: the text box starts at (+0x1a,+0x66), 200 px wide, 10 lines.
	for i, line := range wrapLines(f, v.Text, 200) {
		if i >= 10 {
			break
		}
		f.Draw(s, line, x+0x1a, y+0x66+i*15, white, 200)
	}
	switch mode {
	case HouseModeInfo, HouseModeManager:
		w.drawServices(s, f, blank, x, y, mode, v)
	case HouseModeList:
		for i := 0; i < 12 && v.ListOffset+i < len(v.List); i++ {
			r := v.List[v.ListOffset+i]
			ry := y + 0x6e + i*0x1b
			f.Draw(s, r.Rank, x+0x112, ry, white, 30)
			f.Draw(s, r.Name, x+0x131, ry, white, 110)
			f.Draw(s, r.Points, x+0x19f, ry, white, 60)
			f.Draw(s, r.Master, x+0x1e0, ry, white, 90)
		}
	case HouseModeBuy:
		w.buy.OriginX, w.buy.OriginY, w.buy.Disabled = x, y, !v.CanBuy
		w.buy.Draw(s)
	}
	for i := 0; i < w.tabCount(kind); i++ {
		b := &w.tabs[i]
		b.OriginX, b.OriginY = x, y
		b.Disabled = houseTabDisabled(kind, i, v)
		b.Draw(s)
	}
	w.close.OriginX, w.close.OriginY = x, y
	w.close.Draw(s)
}

func (w *GuildHouseWindow) drawServices(s *render.Sprites, f *Font, blank *render.Texture, x, y, mode int, v *HouseView) {
	f.Draw(s, v.Etin, x+0x15c, y+0x36, white, 70)
	f.Draw(s, v.Available, x+499, y+0x36, white, 70)
	f.Draw(s, v.Total, x+0x200, y+0x1b5, white, 70)
	k := 0
	for i, r := range v.Rows {
		if !r.Show {
			continue
		}
		top := y + rowY(mode, k)
		if mode == HouseModeInfo && i == v.Selected {
			s.Draw(blank, float32(x+0x10e), float32(y+0x55+k*houseRowStep), 0x127, houseRowStep, 0, 0, 1, 1, [4]float32{160.0 / 255, 160.0 / 255, 160.0 / 255, 0x7a / 255.0})
		}
		tint := white
		if r.IconDim {
			tint = [4]float32{64.0 / 255, 64.0 / 255, 64.0 / 255, 1}
		}
		iy, ty := top, top
		if mode == HouseModeManager {
			iy, ty = top+5, top+5
		}
		col, row := r.Icon%16, r.Icon/16
		s.Draw(w.icons, float32(x+0x117), float32(iy), 32, 32, float32(col)/16, float32(row)/16, float32(col+1)/16, float32(row+1)/16, tint)
		f.Draw(s, r.Title, x+0x141, ty, white, 190)
		sub := ty + 0x13
		if mode == HouseModeManager {
			sub = top + 0x18
		}
		if mode == HouseModeInfo || r.ShowUpgrade {
			f.Draw(s, r.RankText, x+0x141, sub, houseTint(r.RankOK), 100)
			f.Draw(s, r.FeeText, x+0x196, sub, houseTint(r.FeeOK), 175)
		}
		if mode == HouseModeManager {
			cell := func(c int) (float32, float32, float32, float32) { return float32(c) / 4, 0, float32(c+1) / 4, 1 }
			bx, by := float32(x+0x201), float32(top)
			if r.UpgradeOn {
				u0, v0, u1, v1 := cell(0)
				s.Draw(w.skill, bx, by, 32, 32, u0, v0, u1, v1, white)
				if w.hover == i {
					c := 1
					if w.down {
						c = 2
					}
					u0, v0, u1, v1 = cell(c)
					s.Draw(w.skill, bx, by, 32, 32, u0, v0, u1, v1, white)
				}
			} else {
				u0, v0, u1, v1 := cell(3)
				s.Draw(w.skill, bx, by, 32, 32, u0, v0, u1, v1, white)
			}
		}
		k++
	}
}
