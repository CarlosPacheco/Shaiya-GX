package world

import (
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

func guildStatus(mode uint8, charID uint32) []byte {
	w := packet.New(packet.GuildStatus)
	w.U8(mode)
	w.U32(charID)
	return w.MustFinish()
}

func writeGuildMember(w *packet.Writer, m *guildMember) {
	w.U32(m.CharID)
	w.U8(m.Rank)
	w.U16(m.Level)
	w.U8(m.Job)
	w.FixedString(m.Name, 21)
}

// buildGuildMemberLists is SendGuildUserList: 0x0D12 online and 0x0D13
// offline members, 29 bytes each, 40 per packet, none sent when empty.
func (g *guild) memberListPackets() [][]byte {
	var out [][]byte
	for _, online := range []bool{true, false} {
		op := packet.GuildOnlineList
		if !online {
			op = packet.GuildOfflineList
		}
		var batch []*guildMember
		flush := func() {
			if len(batch) == 0 {
				return
			}
			w := packet.New(op)
			w.U8(uint8(len(batch)))
			for _, m := range batch {
				writeGuildMember(w, m)
			}
			out = append(out, w.MustFinish())
			batch = nil
		}
		for _, m := range g.members {
			if (m.player != nil) == online {
				batch = append(batch, m)
				if len(batch) == guildMemberBatch {
					flush()
				}
			}
		}
		flush()
	}
	return out
}

func writeGuildApplicant(w *packet.Writer, m *guildMember) {
	w.U32(m.CharID)
	w.U16(m.Level)
	w.U8(m.Job)
	w.FixedString(m.Name, 21)
}

// applicantListPackets is SendGuildJoinList (0x0D16, 28 bytes each).
func (g *guild) applicantListPackets() [][]byte {
	var out [][]byte
	var batch []*guildMember
	flush := func() {
		if len(batch) == 0 {
			return
		}
		w := packet.New(packet.GuildApplicantList)
		w.U8(uint8(len(batch)))
		for _, m := range batch {
			writeGuildApplicant(w, m)
		}
		out = append(out, w.MustFinish())
		batch = nil
	}
	for _, m := range g.applicants {
		batch = append(batch, m)
		if len(batch) == guildMemberBatch {
			flush()
		}
	}
	flush()
	return out
}

// writeGuildInfo is the 120-byte guild record of 0x0D2F and 0x0D30.
func writeGuildInfo(w *packet.Writer, g *guild) {
	w.U32(g.ID)
	w.FixedString(g.Name, 25)
	w.FixedString(g.Master, 21)
	w.FixedString(g.Remark, 65)
	w.U8(g.Rank)
	w.U32(g.Points)
}

// sendWorld is CWorld::SendAll restricted to a country.
func guildSendCountry(country uint8, frame []byte, label string) {
	for _, p := range AllPlayers() {
		if p.Faction() == country {
			p.Send(frame, label)
		}
	}
}

func buildGuildCreateFailed(result uint8, name string) []byte {
	w := packet.New(packet.GuildCreateFailed)
	w.U8(result)
	w.U32(0)
	w.U8(0)
	w.FixedString(name, 25)
	w.Bytes(make([]byte, 98-2-31)) // the original sends a 98-byte frame
	return w.MustFinish()
}

func buildGuildCreateResult(result uint8, id uint32, rank uint8, name, remark string) []byte {
	w := packet.New(packet.GuildCreate)
	w.U8(result)
	w.U32(id)
	w.U8(rank)
	w.FixedString(name, 25)
	w.FixedString(remark, 65)
	return w.MustFinish()
}

func buildGuildJoined(result uint8, g *guild, rank uint8) []byte {
	w := packet.New(packet.GuildJoinResult)
	w.U8(result)
	w.U32(g.ID)
	w.U8(rank)
	w.FixedString(g.Name, 25)
	return w.MustFinish()
}

func guildResult(op packet.Opcode, result uint8) []byte {
	w := packet.New(op)
	w.U8(result)
	return w.MustFinish()
}

func guildResultID(op packet.Opcode, result uint8, id uint32) []byte {
	w := packet.New(op)
	w.U8(result)
	w.U32(id)
	return w.MustFinish()
}

func guildID32(op packet.Opcode, id uint32) []byte {
	w := packet.New(op)
	w.U32(id)
	return w.MustFinish()
}

// guildNameFor / guildIDFor feed the character packets (0x0105, 0x0201,
// 0x0303). They must not be called with a player's mu held.
func buildGuildMemberAdded(m *guildMember) []byte {
	w := packet.New(packet.GuildMemberAdded)
	if m.player != nil {
		w.U8(1)
	} else {
		w.U8(0)
	}
	writeGuildMember(w, m)
	return w.MustFinish()
}

// guildApply (0x0D07): guild id; CGuild::JoinRequest.
func guildListPackets(country uint8) [][]byte {
	out := [][]byte{packet.New(packet.GuildListBegin).MustFinish()}
	var batch []*guild
	flush := func() {
		if len(batch) == 0 {
			return
		}
		w := packet.New(packet.GuildList)
		w.U8(uint8(len(batch)))
		for _, g := range batch {
			writeGuildInfo(w, g)
		}
		out = append(out, w.MustFinish())
		batch = nil
	}
	for _, g := range guildsByID {
		if g.Country == country {
			batch = append(batch, g)
			if len(batch) == guildListBatch {
				flush()
			}
		}
	}
	flush()
	return append(out, packet.New(packet.GuildListEnd).MustFinish())
}

// guildDisconnect is CUser::ReleaseGuild / CGuild::DisconnectUser at world
// exit; a pending creation or disband vote fails as a refusal.
