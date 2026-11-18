package world

import (
	"io"
	"net"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

// TestGlobalPartyInvite covers /party <name> (0x7F15): no same-map limit,
// but GMs, the other faction and offline names are refused without telling
// them apart; 0x0B09 keeps its same-map rule.
func TestGlobalPartyInvite(t *testing.T) {
	logger, err := logx.New("gx_world_test")
	if err != nil {
		t.Fatal(err)
	}
	player := func(id uint32, name string, mapID uint16, faction uint8, admin bool) (*ConnectedPlayer, *Session) {
		a, b := net.Pipe()
		go io.Copy(io.Discard, b)
		p := RegisterPlayer(db.Character{ID: id, Name: name, MapID: mapID}, nil, admin, faction, &connWriter{conn: a}, logger)
		t.Cleanup(p.Unregister)
		return p, &Session{char: &db.Character{ID: id, Name: name}, player: p, faction: faction, log: logger.Tagged("unit"), logger: logger}
	}
	leader, ls := player(992001, "PartyLeadr", 1, FactionLight, false)
	far, fs := player(992002, "PartyFarOne", 40, FactionLight, false)
	gm, _ := player(992003, "PartyGMUser", 1, FactionLight, true)
	gmFar, _ := player(992005, "PartyGMFar", 40, FactionLight, true)
	gmFoe, _ := player(992006, "PartyGMFoe", 40, FactionDark, true)
	foe, _ := player(992004, "PartyFoeOne", 1, FactionDark, false)

	request := func(op packet.Opcode, name string) []byte {
		p := make([]byte, 23)
		p[0], p[1] = byte(op), byte(op>>8)
		copy(p[2:], name)
		return p
	}
	invited := func(p *ConnectedPlayer) (uint32, bool) {
		partyMu.Lock()
		defer partyMu.Unlock()
		return p.inviterID, p.inviteAnyMap
	}

	ls.handlePartyInvite(packet.PartyInviteName, request(packet.PartyInviteName, "PartyFarOne"))
	if id, _ := invited(far); id != 0 {
		t.Fatal("the native name invitation must still need the same map")
	}
	for _, name := range []string{"PartyGMUser", "PartyFoeOne", "NobodyOnline", "PartyLeadr"} {
		ls.handlePartyInviteGlobal(request(packet.GXPartyInviteName, name))
	}
	for _, p := range []*ConnectedPlayer{gm, foe, leader} {
		if id, _ := invited(p); id != 0 {
			t.Fatalf("%s must not be invitable", p.CharacterName)
		}
	}
	ls.handlePartyInviteGlobal(request(packet.GXPartyInviteName, "partyfarone"))
	if id, anyMap := invited(far); id != leader.CharacterID || !anyMap {
		t.Fatalf("global invitation not delivered: %d %v", id, anyMap)
	}
	fs.handlePartyAnswer([]byte{3, 0x0b, 0, 0, 0, 0, 0})
	partyMu.Lock()
	pt := partyOf(leader)
	joined := pt != nil && partyOf(far) == pt
	partyMu.Unlock()
	if !joined {
		t.Fatal("the invited player on another map could not join")
	}

	// A GM may invite another GM (same faction, any map); a GM still cannot
	// invite the other faction.
	_, gms := player(992007, "PartyGMBoss", 1, FactionLight, true)
	gms.handlePartyInviteGlobal(request(packet.GXPartyInviteName, "PartyGMFar"))
	gms.handlePartyInviteGlobal(request(packet.GXPartyInviteName, "PartyGMFoe"))
	if id, anyMap := invited(gmFar); id != 992007 || !anyMap {
		t.Fatalf("GM could not invite another GM: %d %v", id, anyMap)
	}
	if id, _ := invited(gmFoe); id != 0 {
		t.Fatal("a GM invited the other faction")
	}
}
