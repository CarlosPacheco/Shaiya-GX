package world

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/crypto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
)

func gmWireSession(t *testing.T, ch db.Character, admin bool) (*Session, <-chan []byte) {
	t.Helper()
	a, b := net.Pipe()
	t.Cleanup(func() { a.Close(); b.Close() })
	packets := make(chan []byte, 512)
	go func() {
		defer close(packets)
		for {
			var header [2]byte
			if _, err := io.ReadFull(b, header[:]); err != nil {
				return
			}
			body := make([]byte, int(binary.LittleEndian.Uint16(header[:]))-2)
			if _, err := io.ReadFull(b, body); err != nil {
				return
			}
			packets <- crypto.ExpandedXOR(body)
		}
	}()
	logger, err := logx.New("gx_gm_test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { logger.Close() })
	s := &Session{log: logger.Tagged("gm"), logger: logger, char: &ch, isAdmin: admin, gmTurbo: false, expanded: true, out: &connWriter{conn: a}, aesSend: crypto.NewAESCTR([16]byte{}, [16]byte{})}
	return s, packets
}

func waitWorldPacket(t *testing.T, packets <-chan []byte, opcode proto.Opcode) []byte {
	t.Helper()
	timeout := time.NewTimer(3 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case p, ok := <-packets:
			if !ok {
				t.Fatalf("connection closed before packet 0x%04x", opcode)
			}
			if proto.Op(p) == opcode {
				return p
			}
		case <-timeout.C:
			t.Fatalf("missing packet 0x%04x", opcode)
		}
	}
}

func gmResponse(t *testing.T, packets <-chan []byte) proto.GMState {
	t.Helper()
	g, err := proto.DecodeGMState(waitWorldPacket(t, packets, proto.GXGMState))
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestGMCommandsRequireRoleAndToggleAuthoritativeState(t *testing.T) {
	s, packets := gmWireSession(t, db.Character{ID: 990020}, false)
	for _, command := range []string{"/turbo", "/iclear", "/getitem 25 1 200", "/mmake 1 3", "/nmake 1 1", "/cmove 1", "/cure"} {
		if !s.handleGMCommand(command) {
			t.Fatal("GM command escaped to public chat")
		}
		g := gmResponse(t, packets)
		if g.GM || g.Turbo || s.gmTurbo {
			t.Fatal("non-GM acquired privileges")
		}
	}
	s.isAdmin, s.gmTurbo = true, true
	state := game.New()
	state.SelfID = s.char.ID
	for _, want := range []bool{false, true} {
		s.handleGMCommand("/turbo")
		g := gmResponse(t, packets)
		if !g.GM || g.Turbo != want || s.gmTurbo != want {
			t.Fatal("bad toggle", g)
		}
		message := "GM: turbo disabled. Fall damage follows server settings. Server speed limits restored."
		if want {
			message = "GM: turbo enabled. Fall damage disabled. Server speed limits disabled."
		}
		if g.Message != message || s.movement.credit != 0 || s.movement.speedCorrectionPending {
			t.Fatal("toggle retained movement credit or failed to report the rules", g)
		}
		// Exercise the exact server codec against the client state machine.
		s.gmReply("")
		p := waitWorldPacket(t, packets, proto.GXGMState)
		if _, e := state.Apply(p); e != nil || state.GMTurbo != want || !state.Self.IsGM {
			t.Fatal("client GM state", e)
		}
	}
	for _, bad := range []string{"/turbo 1", "/getitem 25 1 -1", "/mmake 1 101", "/nmake 1", "/nmake 14 1", "/nmake 1001", "/nmake 1 1 2", "/cmove 65536"} {
		s.handleGMCommand(bad)
		gmResponse(t, packets)
	}
	if !s.gmTurbo {
		t.Fatal("malformed command changed state")
	}
	if s.handleGMCommand("ordinary chat") {
		t.Fatal("normal chat consumed")
	}
}

func TestGMGrantAndClearPersistOnlyOwnInventory(t *testing.T) {
	account := testAccount(t)
	id, err := db.CreateCharacter(account, db.NewCharacter{Slot: 4, Name: "GMCommandProbe", HP: 100, MP: 100, SP: 100, MapID: 60070})
	if err != nil {
		t.Fatal(err)
	}
	s, packets := gmWireSession(t, db.Character{ID: id, MapID: 60070, Level: 1, CurrentHP: 100}, true)
	defs, err := db.ReloadDefsForTest()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.SetDefsForTest(defs) })
	pool, _ := db.Pool(db.GameData)
	t.Cleanup(func() {
		pool.Exec(`DELETE FROM PS_GameData_CharItems WHERE CharID=? OR CharID=?`, id, id+1000000)
		pool.Exec(`DELETE FROM PS_GameData_Chars WHERE CharID=?`, id)
	})
	s.handleGMCommand("/getitem 25 1")
	gmResponse(t, packets)
	if len(s.items) != 1 || s.items[0].Count != 1 {
		t.Fatal("omitted quantity must grant one", s.items)
	}
	s.handleGMCommand("/getitem 25 1 199")
	reply := gmResponse(t, packets)
	if len(s.items) != 1 {
		t.Fatalf("200 apples should occupy one stack, got %d: %+v", len(s.items), reply)
	}
	stored, err := db.CharacterItems(id)
	if err != nil || len(stored) != 1 {
		t.Fatal("grant persistence", len(stored), err)
	}
	sum := 0
	for _, it := range stored {
		sum += int(it.Count)
		if it.Count > 255 {
			t.Fatal("stack overflow")
		}
	}
	if sum != 200 {
		t.Fatal("wrong quantity", sum)
	}
	// A request that cannot fit must leave the existing stacks and DB intact.
	s.handleGMCommand("/getitem 25 1 119880")
	gmResponse(t, packets)
	stored, err = db.CharacterItems(id)
	if err != nil || len(stored) != 1 {
		t.Fatal("partial failed grant")
	}
	other := db.Item{Bag: 1, Slot: 23, Type: 25, TypeID: 1, Count: 1}
	if err = db.AddCharacterItem(id+1000000, &other); err != nil {
		t.Fatal(err)
	}
	equipped := db.Item{Bag: 0, Slot: 0, Type: 1, TypeID: 1, Count: 1}
	if err = db.AddCharacterItem(id, &equipped); err != nil {
		t.Fatal(err)
	}
	s.items = append(s.items, equipped)
	s.mounted = true
	s.handleGMCommand("/iclear")
	gmResponse(t, packets)
	stored, err = db.CharacterItems(id)
	if err != nil || len(stored) != 1 || len(s.items) != 1 || stored[0].UID != equipped.UID || !s.mounted {
		t.Fatal("clear failed", err)
	}

	// A fresh carried weapon can still be equipped after clearing the bags.
	var weapon *db.ItemDef
	for _, def := range defs.Items {
		if canEquip(s.char, def, 5, s.items) {
			weapon = def
			break
		}
	}
	if weapon == nil {
		t.Fatal("missing equippable weapon fixture")
	}
	if err = s.gmGiveItem(weapon.Type, weapon.TypeID, 1); err != nil {
		t.Fatal(err)
	}
	gmResponse(t, packets)
	wi := findItem(s.items, 1, 0)
	if wi < 0 {
		t.Fatal("fresh weapon not granted")
	}
	uid := s.items[wi].UID
	s.handleItemBagToBag(proto.ItemMovePacket(1, 0, 0, 5))
	wi = findItem(s.items, 0, 5)
	if wi < 0 || s.items[wi].UID != uid {
		t.Fatal("cannot equip after iclear")
	}
	remaining, err := db.CharacterItems(id + 1000000)
	if err != nil || len(remaining) != 1 {
		t.Fatal("clear affected another character", err)
	}
}

func TestGMSpawnsAreSeparatedAndVisibleImmediately(t *testing.T) {
	const mapID = 60071
	RegisterMapGrid(mapID, 200, nil)
	t.Cleanup(func() {
		realm.shardsMu.Lock()
		delete(realm.shards, mapID)
		realm.shardsMu.Unlock()
		realm.gridsMu.Lock()
		delete(realm.grids, mapID)
		realm.gridsMu.Unlock()
	})
	defs, err := db.ReloadDefsForTest()
	if err != nil {
		t.Fatal(err)
	}
	var id uint16
	for key := range defs.Mobs {
		if id == 0 || key < id {
			id = key
		}
	}
	if id == 0 {
		t.Fatal("missing mob defs")
	}
	logger, err := logx.New("gx_gm_spawn_test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { logger.Close() })
	if err = loadNpcQuest(logger); err != nil {
		t.Fatal(err)
	}
	s, packets := gmWireSession(t, db.Character{ID: 990021, MapID: mapID, PosX: 100, PosY: 77, PosZ: 100}, true)
	s.player = RegisterPlayer(*s.char, nil, true, 0, s.out, nil)
	t.Cleanup(s.player.Unregister)
	s.player.moveViewCell(100, 100)
	if err = s.gmSpawn(id, 12, 0); err != nil {
		t.Fatal(err)
	}
	gmResponse(t, packets)
	if err = s.gmSpawn(id, 12, 0); err != nil {
		t.Fatal(err)
	}
	gmResponse(t, packets)
	if !s.handleGMCommand(fmt.Sprintf("/mmake %d", id)) {
		t.Fatal("mob command not handled")
	}
	gmResponse(t, packets)
	if !s.handleGMCommand("/nmake 1 1") {
		t.Fatal("NPC command not handled")
	}
	gmResponse(t, packets)
	entities := EntitiesOnMap(mapID)
	if len(entities.Mobs) != 25 || len(entities.NPCs) != 1 {
		t.Fatal("spawn count")
	}
	for i, m := range entities.Mobs {
		if m.CurrentHP == 0 || m.respawnDelay != 0 || !s.player.knows(viewMob, m.ID) {
			t.Fatal("spawn not live/visible")
		}
		for _, n := range entities.Mobs[:i] {
			if DistanceXZ(m.PosX, m.PosZ, n.PosX, n.PosZ) < 2 {
				t.Fatal("stacked mobs")
			}
		}
		if DistanceXZ(m.PosX, m.PosZ, entities.NPCs[0].PosX, entities.NPCs[0].PosZ) < 2 {
			t.Fatal("stacked NPC")
		}
	}
	n := entities.NPCs[0]
	if n.Type != 1 || n.TypeID != 1 || n.PosY != 77 || !n.Summoned || !s.player.knows(viewNpc, n.ID) {
		t.Fatal("NPC not at server height or missing view")
	}
}

func TestGMMoveUsesRelativeSafeDestinationAndNormalTeleport(t *testing.T) {
	const source, target = 972, 973
	for _, id := range []uint16{source, target} {
		RegisterMapGrid(id, 200, nil)
		t.Cleanup(func() {
			realm.shardsMu.Lock()
			delete(realm.shards, id)
			realm.shardsMu.Unlock()
			realm.gridsMu.Lock()
			delete(realm.grids, id)
			realm.gridsMu.Unlock()
		})
	}
	oldZone, hadZone := mapConfig.Zones[target]
	mapConfig.Zones[target] = ZoneConfig{CreateType: partyCreateType}
	t.Cleanup(func() {
		if hadZone {
			mapConfig.Zones[target] = oldZone
		} else {
			delete(mapConfig.Zones, target)
		}
	})
	sh := shard(target)
	sh.entities.SpawnAreas = []MapSpawnArea{{Boxes: []SpawnBox{
		{LowerX: 10, UpperX: 20, LowerY: 30, LowerZ: 10, UpperZ: 20},
		{LowerX: 170, UpperX: 180, LowerY: 75, LowerZ: 170, UpperZ: 180},
	}}}
	s, packets := gmWireSession(t, db.Character{ID: 990030, MapID: source, PosX: 180, PosY: 8, PosZ: 180}, true)
	g := &mapgeometry.Geometry{}
	g.Reindex()
	height := float64(float32(75.9) - mapgeometry.FootSkin)
	g.Add(collision.Triangle{{0, height, 0}, {0, height, 200}, {200, height, 0}})
	g.Add(collision.Triangle{{200, height, 200}, {200, height, 0}, {0, height, 200}})
	lower := float64(float32(40) - mapgeometry.FootSkin)
	g.Add(collision.Triangle{{20, lower, 40}, {20, lower, 70}, {50, lower, 40}})
	g.Add(collision.Triangle{{50, lower, 70}, {50, lower, 40}, {20, lower, 70}})
	s.movement = playerMovement{geometry: g, mapID: target}
	s.player = &ConnectedPlayer{CharacterID: s.char.ID, character: *s.char}
	s.player.SetMap(source)
	s.player.SetSitting(true)

	if err := s.gmMove(target); err != nil {
		t.Fatal(err)
	}
	if s.char.MapID != target || s.char.PosX != 175 || s.char.PosZ != 175 || s.char.PosY != 75.9 {
		t.Fatalf("unsafe or unrelated destination: %+v", s.char)
	}
	state := game.New()
	state.SelfID = s.char.ID
	state.MapID = source
	state.Self.Sitting = true
	if s.player.IsSitting() {
		t.Fatal("/cmove carried the sitting posture to the destination")
	}
	for {
		select {
		case p := <-packets:
			if _, err := state.Apply(p); err != nil {
				t.Fatal(err)
			}
			if proto.Op(p) == proto.GXGMState {
				if state.MapID != target || state.Self.X != 175 || state.Self.Sitting || !state.Self.IsGM || state.GMTurbo {
					t.Fatal("teleport did not reach client state")
				}
				if err := s.gmMove(974); err == nil || s.char.MapID != target {
					t.Fatal("unknown map accepted")
				}
				sh.entities.SpawnAreas = nil
				shard(source).entities.Portals = []MapPortal{{TargetMapID: target, TargetX: 35, TargetY: 40, TargetZ: 55}}
				if err := s.gmMove(target); err != nil {
					t.Fatal("incoming portal fallback", err)
				}
				gmResponse(t, packets)
				if s.char.PosX != 35 || s.char.PosY != 40 || s.char.PosZ != 55 {
					t.Fatal("wrong fallback position")
				}
				shard(source).entities.Portals = nil
				mapConfig.Zones[target] = ZoneConfig{
					RebirthLight: &MapPos{target, 30, 40, 55},
					RebirthFury:  &MapPos{target, 45, 40, 55},
				}
				for _, tc := range []struct {
					faction uint8
					x       float32
				}{{FactionLight, 30}, {FactionDark, 45}} {
					s.faction = tc.faction
					if err := s.gmMove(target); err != nil {
						t.Fatalf("local faction bind: %v", err)
					}
					gmResponse(t, packets)
					if s.char.PosX != tc.x || s.char.PosY != 40 || s.char.PosZ != 55 {
						t.Fatalf("wrong faction bind: %+v", s.char)
					}
				}
				mapConfig.Zones[target] = ZoneConfig{RebirthFury: &MapPos{source, 45, 40, 55}}
				if err := s.gmMove(target); err == nil {
					t.Fatal("accepted a bind that points to another map")
				}
				return
			}
		case <-time.After(3 * time.Second):
			t.Fatal("teleport packets missing")
		}
	}
}

func TestGMSetPersistsAndUpdatesClientWithoutReopeningRewards(t *testing.T) {
	account := testAccount(t)
	id, err := db.CreateCharacter(account, db.NewCharacter{Slot: 3, Name: "GMSetProbe", HP: 100, MP: 100, SP: 100, MapID: 60070})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.Pool(db.GameData)
	t.Cleanup(func() { pool.Exec(`DELETE FROM PS_GameData_Chars WHERE CharID=?`, id) })
	ch, err := db.CharacterByID(id)
	if err != nil {
		t.Fatal(err)
	}
	ch.KillLevel, ch.DeadLevel = 20, 12
	ch.Kills, ch.Money = 0, 0
	s, packets := gmWireSession(t, ch, true)
	state := game.New()
	state.SelfID = id
	for _, tc := range []struct {
		command      string
		money, kills uint32
	}{
		{"/set money 10000000", 10000000, 0},
		{"/set kill 99999999999999999999999999999999999999", 10000000, 1000000},
		{"/SET MONEY 4294967296", db.MoneyCap, 1000000},
		{"/set kill 50", db.MoneyCap, 50},
		{"/set money 0", 0, 50},
		{"/set kill 0", 0, 0},
	} {
		if !s.handleGMCommand(tc.command) {
			t.Fatal("command escaped to chat")
		}
		for {
			select {
			case p := <-packets:
				if _, err := state.Apply(p); err != nil {
					t.Fatal(err)
				}
				if proto.Op(p) != proto.GXGMState {
					continue
				}
			case <-time.After(3 * time.Second):
				t.Fatal("missing set confirmation")
			}
			break
		}
		saved, err := db.CharacterByID(id)
		if err != nil {
			t.Fatal(err)
		}
		if saved.Money != tc.money || saved.Kills != tc.kills || state.Details.Money != tc.money || state.Details.Kills != tc.kills || saved.KillLevel != 20 || saved.DeadLevel != 12 {
			t.Fatalf("%s: persistence/client mismatch saved=%+v details=%+v", tc.command, saved, state.Details)
		}
	}
	before := *s.char
	for _, bad := range []string{"/set", "/set kill -1", "/set money 1.5", "/set money +1", "/set death 1", "/set kill 10 other", "/set kill 999999x"} {
		s.handleGMCommand(bad)
		gmResponse(t, packets)
		if *s.char != before {
			t.Fatalf("invalid command mutated character: %s", bad)
		}
	}
	s.isAdmin = false
	s.handleGMCommand("/set money 500")
	gmResponse(t, packets)
	if *s.char != before {
		t.Fatal("non-GM changed own money")
	}
}

// /cure restores HP, MP and SP and clears every spell and item cooldown.
func TestGMCureRestoresPoolsAndCooldowns(t *testing.T) {
	s, packets := potentialSession(t)
	s.isAdmin = true
	s.buffs = []ActiveBuff{
		{ID: 9870, Effect: typeEffectBuff},
		{ID: 9871, Effect: typeEffectDebuff},
		{ID: 9872, Effect: typeEffectDebuff, MapCurse: true},
	}
	st := s.stats()
	s.char.CurrentHP, s.char.CurrentMP, s.char.CurrentSP = 1, 1, 1
	s.skillCooldowns = map[uint16]time.Time{30: time.Now().Add(time.Minute), 31: time.Now().Add(time.Hour)}
	s.itemCooldowns = map[uint8]time.Time{1: time.Now().Add(time.Minute)}
	vehicleMu.Lock()
	vehicleCooldowns[s.char.ID] = time.Now().Add(time.Minute)
	vehicleMu.Unlock()
	t.Cleanup(func() {
		vehicleMu.Lock()
		delete(vehicleCooldowns, s.char.ID)
		vehicleMu.Unlock()
	})
	if !s.handleGMCommand("/cure") {
		t.Fatal("/cure escaped to public chat")
	}
	after := s.stats()
	if after.CurrentHP != st.MaxHP || after.CurrentMP != st.MaxMP || after.CurrentSP != st.MaxSP {
		t.Fatalf("pools %d/%d/%d, want %d/%d/%d", after.CurrentHP, after.CurrentMP, after.CurrentSP, st.MaxHP, st.MaxMP, st.MaxSP)
	}
	if len(s.skillCooldowns) != 0 || len(s.itemCooldowns) != 0 {
		t.Fatal("a cooldown survived")
	}
	if len(s.buffs) != 1 || s.buffs[0].ID != 9870 {
		t.Fatal("GM cure did not remove all debuffs while preserving the buff", s.buffs)
	}
	if cooldown := s.player.vehicleState().CooldownMS; cooldown != 0 {
		t.Fatalf("mount cooldown survived: %d ms", cooldown)
	}
	reset := false
	removed := map[uint32]bool{}
	gmReply := false
	for !gmReply {
		select {
		case p := <-packets:
			switch proto.Op(p) {
			case proto.BuffRemove:
				removed[proto.NewReader(p).U32()] = true
			case proto.GXItemCooldownReset:
				_, err := game.New().Apply(p)
				if err != nil {
					t.Fatal(err)
				}
				reset = true
			case proto.GXGMState:
				gmReply = true
			}
		case <-time.After(3 * time.Second):
			t.Fatal("missing cure response")
		}
	}
	if !removed[9871] || !removed[9872] || removed[9870] {
		t.Fatal("missing or incorrect debuff removal notifications", removed)
	}
	if !reset {
		t.Fatal("missing item cooldown reset")
	}
	s.dead = true
	s.char.CurrentHP = 0
	s.handleGMCommand("/cure")
	if s.char.CurrentHP != 0 {
		t.Fatal("/cure must not revive")
	}
}

// /amove goes to another character and /asummon brings one to the GM at once,
// with no item, cast or confirmation. Both are GM only.
func TestGMAmoveAndAsummonMoveAtOnce(t *testing.T) {
	gm, _ := potentialSession(t)
	gm.isAdmin = true
	floor := func() *mapgeometry.Geometry {
		g := &mapgeometry.Geometry{}
		g.Reindex()
		h := float64(float32(10) - mapgeometry.FootSkin)
		g.Add(collision.Triangle{{0, h, 0}, {0, h, 1000}, {1000, h, 0}})
		g.Add(collision.Triangle{{1000, h, 1000}, {1000, h, 0}, {0, h, 1000}})
		return g
	}
	mapID := gm.char.MapID
	gm.char.PosX, gm.char.PosY, gm.char.PosZ = 100, 10, 100
	gm.movement = playerMovement{geometry: floor(), mapID: mapID}
	other, _ := gmWireSession(t, *gm.char, false)
	other.char.ID, other.char.Name = gm.char.ID+700, "AmoveTarget"
	other.char.PosX, other.char.PosY, other.char.PosZ = 520, 10, 380
	other.movement = playerMovement{geometry: floor(), mapID: mapID}
	other.skillCooldowns, other.itemCooldowns = map[uint16]time.Time{}, map[uint8]time.Time{}
	other.player = RegisterPlayer(*other.char, nil, false, other.faction, other.out, other.logger)
	t.Cleanup(other.player.Unregister)

	if !gm.handleGMCommand("/amove AmoveTarget") {
		t.Fatal("/amove escaped to public chat")
	}
	if gm.char.PosX < 519 || gm.char.PosX > 521 || gm.char.PosZ < 379 || gm.char.PosZ > 381 {
		t.Fatalf("/amove left the GM at %v,%v, want about 520,380", gm.char.PosX, gm.char.PosZ)
	}

	gm.char.PosX, gm.char.PosZ = 800, 700
	gm.player.UpdateCharacter(*gm.char)
	gm.handleGMCommand("/asummon AmoveTarget")
	other.tickForcedTeleport()
	if other.char.PosX < 799 || other.char.PosX > 801 || other.char.PosZ < 699 || other.char.PosZ > 701 {
		t.Fatalf("/asummon left the character at %v,%v, want about 800,700", other.char.PosX, other.char.PosZ)
	}

	// A player cannot use them.
	gm.isAdmin = false
	x := gm.char.PosX
	gm.char.PosX += 50
	gm.handleGMCommand("/amove AmoveTarget")
	if gm.char.PosX != x+50 {
		t.Fatal("a non-GM moved with /amove")
	}
}

func TestGMSetStatAndSkillPoints(t *testing.T) {
	account := testAccount(t)
	id, err := db.CreateCharacter(account, db.NewCharacter{Slot: 4, Name: "GMPointsProbe", HP: 100, MP: 100, SP: 100, MapID: 60070})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.Pool(db.GameData)
	t.Cleanup(func() { pool.Exec(`DELETE FROM PS_GameData_Chars WHERE CharID=?`, id) })
	ch, err := db.CharacterByID(id)
	if err != nil {
		t.Fatal(err)
	}
	s, packets := gmWireSession(t, ch, true)
	state := game.New()
	state.SelfID = id
	for _, tc := range []struct {
		command     string
		stat, skill uint16
	}{
		{"/set statpoint 25", 25, ch.SkillPoint},
		{"/set skillpoint 7", 25, 7},
		{"/set statpoint 99999999", db.BaseStatCap, 7},
	} {
		if !s.handleGMCommand(tc.command) {
			t.Fatal("command escaped to chat")
		}
		for done := false; !done; {
			select {
			case p := <-packets:
				if _, err := state.Apply(p); err != nil {
					t.Fatal(err)
				}
				done = proto.Op(p) == proto.GXGMState
			case <-time.After(3 * time.Second):
				t.Fatal("missing set confirmation")
			}
		}
		saved, err := db.CharacterByID(id)
		if err != nil {
			t.Fatal(err)
		}
		if saved.StatPoint != tc.stat || saved.SkillPoint != tc.skill || state.Details.StatPoint != tc.stat || state.Details.SkillPoint != tc.skill {
			t.Fatalf("%s: saved %d/%d client %d/%d", tc.command, saved.StatPoint, saved.SkillPoint, state.Details.StatPoint, state.Details.SkillPoint)
		}
	}
}

// /bless raises the GM's country to the top tier unless a blessing is active.
func TestGMBlessActivatesOnlyWithoutBlessing(t *testing.T) {
	withTiers(t)
	s, _ := potentialSession(t)
	s.isAdmin = true
	c := int(s.faction)
	AddKillCount(uint8(c), 1, nopLog{})
	if !s.handleGMCommand("/bless") {
		t.Fatal("/bless escaped to public chat")
	}
	kcMu.Lock()
	count, hold := kcWorld[c].count, kcWorld[c].resetAt
	kcMu.Unlock()
	if count != 4 || hold.IsZero() {
		t.Fatalf("after /bless count=%d hold=%v", count, hold)
	}
	// An active blessing on either country leaves everything unchanged.
	s.faction = uint8(1 - c)
	s.handleGMCommand("/bless")
	kcMu.Lock()
	other, otherHold := kcWorld[1-c].count, kcWorld[1-c].resetAt
	kcMu.Unlock()
	if other != 0 || !otherHold.IsZero() {
		t.Fatalf("/bless during a blessing changed country %d: count=%d", 1-c, other)
	}
}
