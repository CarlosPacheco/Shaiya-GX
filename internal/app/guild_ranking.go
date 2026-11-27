package app

import (
	"fmt"
	"strconv"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func (a *App) confirmedGuildRanking(p []byte) {
	if a.state == nil {
		return
	}
	switch proto.Op(p) {
	case proto.GuildRankingNotice:
		n := a.state.Guild.RankingNotice
		var message string
		switch n.Kind {
		case 100:
			message = a.msg.Format(12302, map[string]string{"v": fmt.Sprint(n.Map)})
		case 101:
			message = a.tr(12249)
		case 102:
			message = a.tr(12250)
		default:
			if n.Kind < 24 {
				message = fmt.Sprintf("Guild ranking battle on map %d opens at %02d:00 server time.", n.Map, n.Kind)
			}
		}
		if message != "" {
			a.questMessage(message)
		}
	case proto.GuildRankingList:
		a.guild.directoryDirty = true
		a.questMessageID(12127)
	}
}

// guildPointsVisible: the window exists on the ranking battle map only
// (0x49ee40: FUN_004cbca0 == 2, map 50) and N toggles it (SysMsg 8008).
func (a *App) guildPointsVisible() bool {
	return a.state != nil && a.guildPointUI != nil && a.state.MapID == 50 && !a.guild.pointsHidden
}

func (a *App) updateGuildPoints(in *platform.Input) bool {
	if a.state == nil || a.state.MapID != 50 {
		return false
	}
	for _, r := range in.Chars {
		if r == 'n' || r == 'N' {
			a.guild.pointsHidden = !a.guild.pointsHidden
			return true
		}
	}
	if a.guildPointsVisible() && a.guildPointUI.Update(in) {
		a.guild.pointsHidden = true
		return true
	}
	return false
}

// drawGuildRanking draws the ranking battle point window (0x49ee40): the
// guild's points, the top guild of the country and its points.
func (a *App) drawGuildRanking(w, h int) {
	if !a.guildPointsVisible() {
		return
	}
	v := a.state.Guild.RankingPoints
	top := ""
	if info, ok := a.state.Guild.Directory[v.Leader]; ok {
		top = info.Name
	}
	a.guildPointUI.Draw(a.sprites, a.font,
		[3]string{a.msg.Get(8036), a.msg.Get(8056), a.msg.Get(8055)},
		[3]string{strconv.FormatUint(uint64(v.Points), 10), top, strconv.FormatUint(uint64(v.Maximum), 10)})
}
