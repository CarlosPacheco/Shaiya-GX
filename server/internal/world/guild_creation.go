package world

import (
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"strings"
)

// guildCreateRequest (0x0D21) starts a solo foundation or a group vote.
func (s *Session) guildCreateRequest(p []byte) {
	ch, me := s.char, s.player
	name, remark := strings.TrimSpace(fixedText(p, 2, 24)), fixedText(p, 0x1b, 64)
	fail := func(result uint8) {
		s.send(buildGuildCreateResult(result, 0, 0, name, remark), "0x0D21 guild create refused")
	}
	if name == "" {
		fail(6)
		return
	}
	partyMu.Lock()
	group := partyOf(me)
	players := []*ConnectedPlayer{me}
	if group != nil {
		players = append([]*ConnectedPlayer(nil), group.members...)
	}
	partyMu.Unlock()
	if group != nil && len(players) < 2 {
		fail(3)
		return
	}
	guildMu.Lock()
	defer guildMu.Unlock()
	if me.guild.guild != nil || me.guild.create != nil {
		fail(8)
		return
	}
	for _, m := range players {
		switch {
		case m.guild.guild != nil:
			fail(8)
			return
		case m.guild.create != nil || m.MapID() != ch.MapID || m.Faction() != me.Faction():
			fail(3)
			return
		}
	}
	for _, g := range guildsByID {
		if strings.EqualFold(g.Name, name) {
			fail(6)
			return
		}
	}
	c := &guildCreate{name: name, remark: remark, masterID: ch.ID, master: me, pending: map[uint32]bool{}}
	for _, m := range players {
		if g := m.guild.applied; g != nil {
			g.cancelApplication(m)
			if m.guild.applied != nil {
				fail(0)
				return
			}
		}
	}
	c.players = players
	if group == nil {
		c.complete(s.logger, s)
		return
	}
	for _, m := range players {
		c.pending[m.CharacterID] = true
		m.guild.create = c
	}
	// SendGuildCreateAgree (0x0D22): master id, name, remark to all.
	w := packet.New(packet.GuildCreateAgree)
	w.U32(ch.ID)
	w.FixedString(name, 25)
	w.FixedString(remark, 65)
	c.sendAll(w.MustFinish(), "0x0D22 guild create agree")
}

func (c *guildCreate) sendAll(frame []byte, label string) {
	for _, m := range c.players {
		m.Send(frame, label)
	}
}

// fail is SendGuildCreateFailed / SendGuildFireFailed then ResetCreate.
func (c *guildCreate) fail(result uint8) {
	if c.disband {
		c.sendAll(guildResult(packet.GuildDisband, result), "0x0D03 guild disband failed")
	} else {
		c.sendAll(buildGuildCreateFailed(result, c.name), "0x0D01 guild create failed")
	}
	c.reset()
}

func (c *guildCreate) reset() {
	for _, m := range c.players {
		if m.guild.create == c {
			m.guild.create = nil
		}
	}
}

// guildCreateAnswer (0x0D22): the group must accept unanimously.
func (s *Session) guildCreateAnswer(accept bool) {
	guildMu.Lock()
	defer guildMu.Unlock()
	me := s.player
	c := me.guild.create
	if me.guild.guild != nil || c == nil || c.disband {
		return
	}
	if !accept {
		c.fail(7)
		return
	}
	if !c.pending[s.char.ID] {
		return
	}
	delete(c.pending, s.char.ID)
	if len(c.pending) == 0 {
		if c.complete(s.logger, s) {
			c.reset()
		}
	}
}

// complete is CGuildMgr::CreateGuild 0x004321e0.
func (c *guildCreate) complete(log *logx.Logger, caller *Session) bool {
	for _, g := range guildsByID {
		if strings.EqualFold(g.Name, c.name) {
			c.sendAll(buildGuildCreateFailed(6, c.name), "0x0D01 guild create failed")
			return true
		}
	}
	owner := c.master.owner
	if owner == nil {
		c.sendAll(buildGuildCreateFailed(6, c.name), "guild master unavailable")
		return true
	}
	if owner != caller {
		if !owner.stateMu.TryLock() {
			return false
		}
		defer owner.stateMu.Unlock()
	}
	if owner.char == nil || owner.player != c.master {
		return true
	}
	master := *owner.char
	fee := guildHouse.CreateMoney
	if master.Money < fee {
		c.sendAll(buildGuildCreateFailed(2, c.name), "0x0D01 guild create failed")
		return true
	}
	g := &guild{ID: nextGuildID, Name: c.name, Master: master.Name, Remark: c.remark, Country: c.master.Faction(),
		Rank: 31, gmHouse: owner.isAdmin, members: map[uint32]*guildMember{}, applicants: map[uint32]*guildMember{}}
	g.forceGMHouse()
	founders := make([]db.GuildCharRow, 0, len(c.players))
	for _, p := range c.players {
		rank := uint8(guildRankMember)
		if p == c.master {
			rank = guildRankMaster
		}
		founders = append(founders, db.GuildCharRow{CharID: p.CharacterID, Rank: rank})
	}
	if err := db.CreateGuildGroup(g.ID, g.Name, c.master.guild.userID, g.Remark, g.Country, master, fee, founders); err != nil {
		log.Printf("GX Server World guild create failed: %v", err)
		c.sendAll(buildGuildCreateFailed(6, c.name), "0x0D01 guild create failed")
		return true
	}
	nextGuildID++
	guildsByID[g.ID] = g
	owner.char.Money -= fee
	owner.syncPlayer()
	owner.send(buildSetMoney(owner.char.Money), "0x0213 guild fee")
	for _, p := range c.players {
		snap := p.Snapshot()
		rank := uint8(guildRankMember)
		if p == c.master {
			rank = guildRankMaster
		}
		g.members[p.CharacterID] = &guildMember{CharID: p.CharacterID, Name: snap.Name, Job: snap.Class, Level: snap.Level,
			Rank: rank, player: p}
		guildOfChar[p.CharacterID] = g
		p.guild.guild, p.guild.applied = g, nil
	}
	for _, p := range c.players {
		p.Send(buildGuildCreateResult(1, g.ID, g.members[p.CharacterID].Rank, g.Name, g.Remark), "0x0D21 guild created")
		for _, f := range g.memberListPackets() {
			p.Send(f, "0x0D12 guild members")
		}
	}
	w := packet.New(packet.GuildNew)
	writeGuildInfo(w, g)
	guildSendCountry(g.Country, w.MustFinish(), "0x0D30 new guild")
	log.Printf("GX Server World guild created id=%d name=%q master=%q", g.ID, g.Name, g.Master)
	return true
}

// completePendingGuild retries a fully accepted group on the master's own
// session loop when its state lock was busy during another member's reply.
func (s *Session) completePendingGuild() {
	if s.player == nil {
		return
	}
	guildMu.Lock()
	defer guildMu.Unlock()
	c := s.player.guild.create
	if c != nil && !c.disband && c.master == s.player && len(c.pending) == 0 {
		if c.complete(s.logger, s) {
			c.reset()
		}
	}
}

// guildDisbandRequest (0x0D03): a master with at most 2 members disbands
// at once; otherwise two more party admins of the guild must agree.
