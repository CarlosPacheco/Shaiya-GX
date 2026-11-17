//go:build integration

package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/mapnav"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
	"github.com/Spelunkern/Shaiya-GX/server/internal/crypto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"github.com/Spelunkern/Shaiya-GX/server/internal/session"
	"github.com/Spelunkern/Shaiya-GX/server/internal/world"
)

// Opt-in: this walks in real time and starts/kills real world processes.
// It never grants levels, stats, items or progress to its two new characters.
func TestPartyProgressionJourney(t *testing.T) {
	if os.Getenv("GX_RUN_PARTY_JOURNEY") != "1" {
		t.Skip("opt-in real-time journey; set GX_RUN_PARTY_JOURNEY=1")
	}
	home := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(home); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(previous)
	// This opt-in case is run alone; TestMain already owns the private database.
	if err = db.Open(db.GameData, db.GameDefs); err != nil {
		t.Fatal(err)
	}
	logger, err := logx.New("journey_catalog")
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()
	if err = world.LoadGameConfig(logger); err != nil {
		t.Fatal(err)
	}
	if err = world.LoadSvmaps(logger); err != nil {
		t.Fatal(err)
	}
	j := &partyJourney{t: t, home: home, nav: map[uint16]*mapnav.Map{}}
	defer j.stopWorld()
	t.Cleanup(func() {
		if directory := os.Getenv("GX_JOURNEY_ARTIFACTS"); directory != "" {
			os.MkdirAll(directory, 0700)
			paths, _ := filepath.Glob(filepath.Join(home, "world-*.log"))
			for _, path := range paths {
				raw, _ := os.ReadFile(path)
				os.WriteFile(filepath.Join(directory, filepath.Base(path)), raw, 0600)
			}
		}
	})
	j.startWorld()
	pool, _ := db.Pool(db.GameData)
	for i := 0; i < 2; i++ {
		uid := uint32(996700 + i)
		if _, err = pool.Exec(`INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(?,?,'local',0,0,0,0,0,'N',0)`, uid, fmt.Sprintf("journey_%d", i)); err != nil {
			t.Fatal(err)
		}
		p := j.connect(uid)
		j.players = append(j.players, p)
		j.wait("initial faction", 15*time.Second, func() bool { return p.state.Phase == game.PhaseFaction })
		p.send(proto.FactionPacket(0))
		j.wait("faction selection", 10*time.Second, func() bool { return p.state.Phase == game.PhaseCharSelect })
		p.send(proto.CreateCharacterPacket(proto.NewCharacter{Slot: 0, Race: 0, Mode: 1, Class: 0, Name: fmt.Sprintf("GXJourney%d", i)}))
		j.wait("character creation", 10*time.Second, func() bool { return p.state.Slots[0].ID != 0 })
		p.id = p.state.Slots[0].ID
		p.send(proto.SelectCharacterPacket(p.id))
		j.wait("starter world", 15*time.Second, func() bool { return p.state.Phase == game.PhaseInWorld && p.state.InventoryReady })
		if p.state.Slots[0].Level != 1 {
			t.Fatal("journey did not start at level 1")
		}
	}
	t.Log("two normal-mode level-1 fighters created through native requests, with original starter inventory")
	j.formParty()
	start, end := j.npc(1, 8, 83), j.npc(1, 8, 87)
	j.walkBoth(mapnav.Point{start.PosX, start.PosY, start.PosZ}, 2)
	for _, p := range j.players {
		p.send(proto.QuestStartPacket(start.ID, 1))
	}
	j.wait("both accept quest 1", 10*time.Second, func() bool {
		for _, p := range j.players {
			if _, ok := p.state.Quests[1]; !ok {
				return false
			}
		}
		return true
	})
	t.Log("both players accepted quest 1 at the original NPC")
	// Choose a real quest spawn that is reachable on the baked map, rather
	// than assuming the debugging coordinates used by other independent probes.
	var area mapnav.Point
	best := float32(math.MaxFloat32)
	for _, mob := range world.EntitiesOnMap(1).Mobs {
		if mob.MobID != 2008 {
			continue
		}
		point := mapnav.Point{mob.PosX, mob.PosY, mob.PosZ}
		if _, ok := j.nav[1].Path(j.point(j.players[0]), point); !ok {
			continue
		}
		distance := xzDistance(j.point(j.players[0]), point)
		if distance < best {
			best = distance
			area = point
		}
	}
	if best == float32(math.MaxFloat32) {
		t.Fatal("no reachable original quest spawn")
	}
	t.Logf("walking to original quest spawn %v, %.1f units away", area, best)
	j.walkBoth(area, 3)
	kills, goldEvents, loot := 0, 0, 0
	initialMoney := [2]uint32{j.players[0].state.Details.Money, j.players[1].state.Details.Money}
	for kills < 30 {
		if kills >= 5 && goldEvents > 0 && loot > 0 {
			break
		}
		var target *game.Entity
		j.wait("living quest mob", 20*time.Second, func() bool {
			best := float32(math.MaxFloat32)
			for _, e := range j.players[0].state.Entities {
				if e.Kind != game.KindMob || e.TypeID != 2008 || e.Dead {
					continue
				}
				if _, ok := j.nav[1].Path(j.point(j.players[0]), mapnav.Point{e.X, e.Y, e.Z}); !ok {
					continue
				}
				d := xzDistance(j.point(j.players[0]), mapnav.Point{e.X, e.Y, e.Z})
				if d < best {
					best = d
					target = e
				}
			}
			return target != nil
		})
		targetID := target.ID
		j.walkBoth(mapnav.Point{target.X, target.Y, target.Z}, 1.3)
		before := [2]uint32{j.players[0].state.Details.Exp, j.players[1].state.Details.Exp}
		beforeCounters := [2]uint8{j.players[0].state.Quests[1].Counters[0], j.players[1].state.Quests[1].Counters[0]}
		beforeGold := [2]uint32{j.players[0].state.Details.Money, j.players[1].state.Details.Money}
		for _, p := range j.players {
			p.send(proto.TargetMobPacket(targetID))
			p.send(proto.MobAttackPacket(targetID))
		}
		j.wait("shared kill and quest progress", 65*time.Second, func() bool {
			for i, p := range j.players {
				if p.state.Self.Dead {
					t.Fatalf("level-1 party died fighting quest mob; player=%d hp=%d", i, p.state.Pools.HP)
				}
				if p.state.Quests[1].Counters[0] <= beforeCounters[i] || p.state.Details.Exp <= before[i] {
					return false
				}
			}
			return true
		})
		for _, p := range j.players {
			p.send(proto.StopAttackPacket())
		}
		kills++
		j.pause(400 * time.Millisecond)
		if j.players[0].state.Details.Money > beforeGold[0] || j.players[1].state.Details.Money > beforeGold[1] {
			da := j.players[0].state.Details.Money - beforeGold[0]
			db := j.players[1].state.Details.Money - beforeGold[1]
			if da == 0 || da != db {
				t.Fatalf("unequal same-level party gold shares: %d/%d", da, db)
			}
			goldEvents++
		}
		// Check combined delivery when player A picks up allowed party loot.
		for id, item := range j.players[0].state.GroundItems {
			if xzDistance(j.point(j.players[0]), mapnav.Point{item.X, j.players[0].state.Self.Y, item.Z}) > 9 {
				continue
			}
			old0, old1 := inventoryCount(j.players[0].state), inventoryCount(j.players[1].state)
			// Already inside the original ten-unit pickup range; no extra route is needed.
			j.players[0].send(proto.PickupPacket(id))
			j.wait("party loot delivery", 5*time.Second, func() bool { return inventoryCount(j.players[0].state)+inventoryCount(j.players[1].state) > old0+old1 })
			loot++
		}
		t.Logf("kill %d: quest=%d/%d exp=%d/%d gold=%d/%d hp=%d/%d", kills, j.players[0].state.Quests[1].Counters[0], j.players[1].state.Quests[1].Counters[0], j.players[0].state.Details.Exp, j.players[1].state.Details.Exp, j.players[0].state.Details.Money, j.players[1].state.Details.Money, j.players[0].state.Pools.HP, j.players[1].state.Pools.HP)
	}
	if loot == 0 || goldEvents == 0 {
		t.Fatalf("bounded 30-kill route did not observe random loot and shared gold: loot=%d gold_events=%d initial=%v", loot, goldEvents, initialMoney)
	}
	j.walkBoth(mapnav.Point{end.PosX, end.PosY, end.PosZ}, 2)
	for _, p := range j.players {
		p.send(proto.QuestEndPacket(end.ID, 1))
	}
	j.wait("both quest rewards", 10*time.Second, func() bool {
		for _, p := range j.players {
			if p.state.LastQuestEnd.ID != 1 || p.state.LastQuestEnd.Success != 1 {
				return false
			}
		}
		return true
	})
	for _, p := range j.players {
		if p.state.Slots[0].Level <= 1 {
			t.Fatal("quest route produced no level-up")
		}
		j.learnSkill(p)
	}
	t.Log("quest turn-in, earned level/stat/skill-point progression and skill learning confirmed for both players")
	gate := j.npc(1, 2, 32)
	j.walkBoth(mapnav.Point{gate.PosX, gate.PosY, gate.PosZ}, 2)
	for _, p := range j.players {
		p.send(proto.GatekeeperPacket(gate.ID, 1))
	}
	j.wait("native gatekeeper travel", 10*time.Second, func() bool {
		for _, p := range j.players {
			if p.state.LastGatekeeper.Result != 1 || p.state.Self.Z > 1000 {
				return false
			}
		}
		return true
	})
	var portal world.MapPortal
	for _, p := range world.EntitiesOnMap(1).Portals {
		if p.TargetMapID == 5 {
			portal = p
			break
		}
	}
	if portal.TargetMapID == 0 {
		t.Fatal("beginner dungeon portal missing")
	}
	// Mob navigation is not a player route planner and cannot connect this
	// exterior approach. Position-only staging is explicit, after normal logout;
	// levels, stats, money, quests, skills and inventory remain untouched.
	for _, p := range j.players {
		p.send(proto.LogoutPacket())
	}
	j.wait("portal staging logout", 10*time.Second, func() bool {
		for _, p := range j.players {
			if p.state.Phase != game.PhaseCharSelect {
				return false
			}
		}
		return true
	})
	staged := j.snapshot()
	for _, p := range j.players {
		p.close()
	}
	j.pause(250 * time.Millisecond)
	for _, saved := range staged {
		ch := saved.character
		ch.MapID, ch.PosX, ch.PosY, ch.PosZ = 1, portal.PosX+3, portal.PosY, portal.PosZ
		if err := db.SaveCharacter(&ch); err != nil {
			t.Fatal(err)
		}
	}
	j.players = nil
	for _, saved := range staged {
		p := j.connect(saved.uid)
		p.id = saved.character.ID
		j.players = append(j.players, p)
		j.wait("staged select", 15*time.Second, func() bool { return p.state.Phase == game.PhaseCharSelect })
		p.send(proto.SelectCharacterPacket(p.id))
		j.wait("staged entry", 15*time.Second, func() bool { return p.state.Phase == game.PhaseInWorld && p.state.InventoryReady })
	}
	j.formParty()
	t.Log("POSITION-ONLY FIXTURE CHECKPOINT: staged at original portal; overland route from gatekeeper remains unvalidated; earned progression preserved")
	j.walkBoth(mapnav.Point{portal.PosX, portal.PosY, portal.PosZ}, 0.5)
	for _, p := range j.players {
		if p.state.MapID == 1 {
			w := proto.NewWriter(proto.CharacterEnteredPortal)
			w.U8(portal.Index)
			p.send(w.Bytes())
		}
	}
	j.wait("both enter beginner dungeon", 12*time.Second, func() bool { return j.players[0].state.MapID == 5 && j.players[1].state.MapID == 5 })
	t.Logf("both players entered original dungeon map 5 at levels %d/%d", j.players[0].state.Slots[0].Level, j.players[1].state.Slots[0].Level)
	beforeRestart := j.snapshot()
	for _, p := range j.players {
		p.close()
	}
	j.stopWorld()
	j.startWorld()
	j.players = nil
	for _, saved := range beforeRestart {
		p := j.connect(saved.uid)
		p.id = saved.character.ID
		j.players = append(j.players, p)
		j.wait("restart character select", 15*time.Second, func() bool { return p.state.Phase == game.PhaseCharSelect && p.state.Slots[0].ID == p.id })
		p.send(proto.SelectCharacterPacket(p.id))
		j.wait("restart world", 15*time.Second, func() bool { return p.state.Phase == game.PhaseInWorld && p.state.InventoryReady })
	}
	j.compareSnapshot(beforeRestart)
	j.formParty()
	t.Log("real world process restarted; both characters recovered exact committed progression, items, learned skills, completed quest and dungeon position; party re-formed")
	// A real dungeon mob deals the damage; there are no HP/level edits.
	victim := j.players[1]
	var enemy *game.Entity
	j.wait("dungeon mob", 15*time.Second, func() bool {
		best := float32(math.MaxFloat32)
		for _, e := range victim.state.Entities {
			def := db.MustDefs().Mobs[e.TypeID]
			if e.Kind == game.KindMob && !e.Dead && def != nil && def.MaxHP >= 300 && def.Attacks[0].OK != 0 && def.Attacks[0].Attack+def.Attacks[0].Plus > 0 {
				d := xzDistance(j.point(victim), mapnav.Point{e.X, e.Y, e.Z})
				if d < best {
					best = d
					enemy = e
				}
			}
		}
		return enemy != nil
	})
	j.stopOnDeath = true
	j.walkOne(victim, mapnav.Point{enemy.X, enemy.Y, enemy.Z}, 1.1)
	j.stopOnDeath = false
	if !victim.state.Self.Dead {
		t.Logf("provoking real dungeon combat mob id=%d type=%d hp=%d", enemy.ID, enemy.TypeID, db.MustDefs().Mobs[enemy.TypeID].MaxHP)
		beforeAttack := victim.seen[proto.CharacterMobAutoAttack]
		victim.send(proto.MobAttackPacket(enemy.ID))
		j.wait("dungeon mob provoked", 10*time.Second, func() bool { return victim.seen[proto.CharacterMobAutoAttack] > beforeAttack })
		victim.send(proto.StopAttackPacket())

	}
	j.wait("dungeon death", 90*time.Second, func() bool { return victim.state.Self.Dead })
	if j.players[0].state.Self.Dead {
		t.Fatal("party resurrection requires the parked leader to remain alive")
	}
	victim.send(proto.ReviveChoicePacket(proto.ReviveParty))
	j.wait("original 30-second party resurrection", 40*time.Second, func() bool { return !victim.state.Self.Dead && victim.state.Pools.HP > 0 })
	if victim.state.FinishedQuests[1] != 1 {
		t.Fatal("death lost completed quest")
	}
	t.Logf("real dungeon mob death and native party resurrection confirmed: map=%d level=%d exp=%d hp=%d", victim.state.MapID, victim.state.Slots[0].Level, victim.state.Details.Exp, victim.state.Pools.HP)
	if victim.state.MapID != 5 || xzDistance(j.point(victim), j.point(j.players[0])) > 3 {
		t.Fatal("party resurrection did not return to the living leader")
	}
	t.Log("resurrected player returned to the living party leader in the dungeon after the native wait")
	for _, p := range j.players {
		p.send(proto.LogoutPacket())
	}
	j.wait("final logout", 10*time.Second, func() bool {
		for _, p := range j.players {
			if p.state.Phase != game.PhaseCharSelect {
				return false
			}
		}
		return true
	})
	var integrity string
	if err = pool.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatal("final integrity", integrity, err)
	}
	t.Logf("journey complete: %d shared kills, %d loot pickups, %d shared-gold events; SQLite integrity OK", kills, loot, goldEvents)
}

type journeyPacket struct {
	payload []byte
	err     error
}
type journeyPlayer struct {
	uid, id uint32
	state   *game.State
	wire    *client
	packets chan journeyPacket
	done    chan struct{}
	closed  bool
	seen    map[proto.Opcode]uint64
}

func (p *journeyPlayer) send(b []byte) { p.wire.write(b) }
func (p *journeyPlayer) close() {
	if !p.closed {
		p.closed = true
		close(p.done)
		p.wire.conn.Close()
	}
}

type partyJourney struct {
	t             *testing.T
	home, address string
	players       []*journeyPlayer
	nav           map[uint16]*mapnav.Map
	process       *exec.Cmd
	input         io.WriteCloser
	exit          chan error
	stopOnDeath   bool
}

func (j *partyJourney) startWorld() {
	j.t.Helper()
	reservation, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		j.t.Fatal(err)
	}
	j.address = reservation.Addr().String()
	reservation.Close()
	cmd := exec.Command(os.Args[0], "-test.run=^TestJourneyWorldProcess$")
	temp := j.t.TempDir()
	cmd.Env = append(os.Environ(), "GX_JOURNEY_WORLD=1", "GX_JOURNEY_DATA="+config.DataDir(), "GX_JOURNEY_HOME="+j.home, "GX_JOURNEY_DATABASE="+config.DatabasePath(), "RAPTURE_WORLD_LISTEN="+j.address, "TMPDIR="+temp, "TMP="+temp, "TEMP="+temp)
	log, err := os.Create(filepath.Join(j.home, fmt.Sprintf("world-%d.log", time.Now().UnixNano())))
	if err != nil {
		j.t.Fatal(err)
	}
	cmd.Stdout, cmd.Stderr = log, log
	j.input, err = cmd.StdinPipe()
	if err != nil {
		j.t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		j.t.Fatal(err)
	}
	log.Close()
	j.process = cmd
	j.exit = make(chan error, 1)
	done := j.exit
	go func() { done <- cmd.Wait(); close(done) }()
	deadline := time.Now().Add(45 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", j.address, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		select {
		case err := <-j.exit:
			raw, _ := os.ReadFile(log.Name())
			j.t.Fatalf("world exited: %v\n%s", err, raw)
		default:
		}
		if time.Now().After(deadline) {
			j.t.Fatal("world startup timed out")
		}
		time.Sleep(50 * time.Millisecond)
	}
}
func (j *partyJourney) stopWorld() {
	if j.process == nil {
		return
	}
	j.input.Close()
	select {
	case <-j.exit:
	case <-time.After(5 * time.Second):
		j.process.Process.Kill()
		<-j.exit
	}
	j.process = nil
}
func TestJourneyWorldProcess(t *testing.T) {
	if os.Getenv("GX_JOURNEY_WORLD") != "1" {
		t.Skip("subprocess helper")
	}
	os.Setenv("RAPTURE_DB_PATH", os.Getenv("GX_JOURNEY_DATABASE"))
	os.Setenv("RAPTURE_DATA_DIR", os.Getenv("GX_JOURNEY_DATA"))
	os.Setenv("RAPTURE_DB_SEED", filepath.Join(os.Getenv("GX_JOURNEY_DATA"), "rapture.sqlite"))
	if err := os.Chdir(os.Getenv("GX_JOURNEY_HOME")); err != nil {
		t.Fatal(err)
	}
	go func() { io.Copy(io.Discard, os.Stdin); os.Exit(0) }()
	main()
}
func (j *partyJourney) connect(uid uint32) *journeyPlayer {
	j.t.Helper()
	var material [32]byte
	if _, err := rand.Read(material[:]); err != nil {
		j.t.Fatal(err)
	}
	identity := session.NewIdentity()
	if err := session.Save(session.Handoff{Identity: identity, AESMaterial: material, UserUID: uid, SessionID: session.NewSessionID()}); err != nil {
		j.t.Fatal(err)
	}
	conn, err := net.Dial("tcp", j.address)
	if err != nil {
		j.t.Fatal(err)
	}
	var key, iv [16]byte
	copy(key[:], material[:16])
	copy(iv[:], material[16:])
	iv = crypto.DeriveWorldAESIV(iv)
	p := &journeyPlayer{uid: uid, state: game.New(), packets: make(chan journeyPacket, 2048), done: make(chan struct{}), seen: map[proto.Opcode]uint64{}}
	p.wire = &client{t: j.t, conn: conn, send: crypto.NewAESCTR(key, iv), recv: crypto.NewAESCTR(key, iv)}
	j.t.Cleanup(p.close)
	go func() {
		expanded := false
		for {
			var header [2]byte
			_, err := io.ReadFull(conn, header[:])
			var payload []byte
			if err == nil {
				length := int(binary.LittleEndian.Uint16(header[:]))
				if length < 4 {
					err = fmt.Errorf("bad frame length %d", length)
				} else {
					payload = make([]byte, length-2)
					_, err = io.ReadFull(conn, payload)
					if err == nil {
						if expanded {
							payload = crypto.ExpandedXOR(payload)
						} else {
							payload = p.wire.recv.Transform(payload)
						}
						if proto.Op(payload) == proto.CharacterSkillBar {
							expanded = true
						}
						if proto.Op(payload) == proto.Logout {
							expanded = false
						}
					}
				}
			}
			select {
			case p.packets <- journeyPacket{payload, err}:
			case <-p.done:
				return
			}
			if err != nil {
				return
			}
		}
	}()
	handshake := packet.New(packet.GameHandshake)
	handshake.U32(uid)
	handshake.Bytes(identity[:])
	if _, err := conn.Write(handshake.MustFinish()); err != nil {
		j.t.Fatal(err)
	}
	return p
}
func (j *partyJourney) pump() {
	for _, p := range j.players {
		if p.closed {
			continue
		}
		for i := 0; i < 256; i++ {
			select {
			case next := <-p.packets:
				if next.err != nil {
					j.t.Fatal("world connection", p.uid, next.err)
				}
				p.seen[proto.Op(next.payload)]++
				ev, err := p.state.Apply(next.payload)
				if err != nil {
					j.t.Fatalf("client packet 0x%x: %v", proto.Op(next.payload), err)
				}
				if ev.MapLoadAck {
					p.send(proto.MapLoadedPacket())
				}
			default:
				i = 256
			}
		}
	}
}
func (j *partyJourney) wait(label string, d time.Duration, ready func() bool) {
	j.t.Helper()
	end := time.Now().Add(d)
	for !ready() {
		j.pump()
		if time.Now().After(end) {
			j.t.Fatalf("journey timeout: %s", label)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
func (j *partyJourney) pause(d time.Duration) {
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		j.pump()
		time.Sleep(5 * time.Millisecond)
	}
}
func (j *partyJourney) formParty() {
	a, b := j.players[0], j.players[1]
	a.send(proto.PartyInvitePacket(b.id))
	j.wait("party invitation", 10*time.Second, func() bool { return b.state.Party.InviteSerial > 0 })
	b.send(proto.PartyAnswerPacket(true, a.id))
	j.wait("party roster", 10*time.Second, func() bool { return len(a.state.Party.Members) == 2 && len(b.state.Party.Members) == 2 })
}
func (j *partyJourney) npc(mapID uint16, kind uint8, id uint16) world.MapNpc {
	for _, n := range world.EntitiesOnMap(mapID).NPCs {
		if n.Type == kind && n.TypeID == id {
			return n
		}
	}
	j.t.Fatalf("missing NPC %d/%d on map %d", kind, id, mapID)
	return world.MapNpc{}
}
func (j *partyJourney) point(p *journeyPlayer) mapnav.Point {
	return mapnav.Point{p.state.Self.X, p.state.Self.Y, p.state.Self.Z}
}
func xzDistance(a, b mapnav.Point) float32 {
	return float32(math.Hypot(float64(a[0]-b[0]), float64(a[2]-b[2])))
}
func (j *partyJourney) walkBoth(target mapnav.Point, radius float32) {
	j.walk(j.players, target, radius)
}
func (j *partyJourney) walkOne(p *journeyPlayer, target mapnav.Point, radius float32) {
	j.walk([]*journeyPlayer{p}, target, radius)
}
func (j *partyJourney) walk(players []*journeyPlayer, target mapnav.Point, radius float32) {
	j.t.Helper()
	mapID := players[0].state.MapID
	nav := j.nav[mapID]
	if nav == nil {
		var err error
		nav, err = mapnav.Load(filepath.Join(config.DataDir(), "geometry"), mapID)
		if err != nil {
			j.t.Fatal(err)
		}
		j.nav[mapID] = nav
	}
	end, ok := nav.Project(target, 4, 5)
	if !ok {
		j.t.Fatalf("no navigation floor near %v map=%d", target, mapID)
	}
	distance := xzDistance(j.point(players[0]), end)
	if distance > 30 {
		j.t.Logf("walking %.1f units on map %d to %v", distance, mapID, end)
	}
	deadline := time.Now().Add(time.Duration(distance/2+30) * time.Second)
	for {
		complete := true
		for _, p := range players {
			if p.state.MapID != mapID {
				continue
			}
			if p.state.Self.Dead {
				if j.stopOnDeath {
					return
				}
				j.t.Fatal("died while walking")
			}
			start := j.point(p)
			if xzDistance(start, end) <= radius {
				continue
			}
			complete = false
			path, ok := nav.Path(start, end)
			if !ok {
				j.t.Fatalf("no path map=%d %v -> %v", mapID, start, end)
			}
			next, travel, _, ok := nav.Step(start, path, .45, func(float32, float32) bool { return true })
			if !ok || travel < .001 {
				// A navmesh corner can project to a different foot height than an
				// actual player. Use the shared player collision controller for this
				// short step, never a forged jump/teleport or a collision bypass.
				g, available := nav.PlayerGeometry()
				if !available {
					j.t.Fatal("player physics unavailable")
				}
				corner := end
				for _, point := range path {
					if xzDistance(start, point) > .05 {
						corner = point
						break
					}
				}
				dx, dz := corner[0]-start[0], corner[2]-start[2]
				length := float32(math.Hypot(float64(dx), float64(dz)))
				if length < .001 {
					j.t.Fatal("empty navigation corner")
				}
				step := min(length, float32(.2))
				moved := g.GroundMove(mapgeometry.Position(start), mapgeometry.Position{dx / length * step, 0, dz / length * step}, false)
				if floor, supported := g.Below(moved[0], moved[1]+.05, moved[2], .65); supported {
					moved[1] = floor + mapgeometry.FootSkin
				}
				next = mapnav.Point(moved)
				if xzDistance(start, next) < .001 {
					j.t.Fatalf("player collision blocked route map=%d at %v corner=%v", mapID, start, corner)
				}
			}
			p.send(proto.MovePacket(1, 0, next[0], next[1], next[2]))
			// Same prediction as a client; authoritative correction packets may replace it.
			p.state.Self.X, p.state.Self.Y, p.state.Self.Z = next[0], next[1], next[2]
		}
		if complete {
			return
		}
		if time.Now().After(deadline) {
			j.t.Fatalf("walk timed out map=%d target=%v current=%v", mapID, end, j.point(players[0]))
		}
		j.pause(100 * time.Millisecond)
	}
}
func inventoryCount(st *game.State) uint32 {
	var count uint32
	for _, it := range st.Inventory {
		count += uint32(it.Count)
	}
	return count
}
func (j *partyJourney) learnSkill(p *journeyPlayer) {
	var skills []*db.SkillDef
	for key, def := range db.MustDefs().Skills {
		if key.Level == 1 && def.ReqLevel <= p.state.Slots[0].Level && def.ClassEligible[0] && def.TypeShow != 0 && def.Grow <= 1 && uint16(def.SkillPointCost) <= p.state.Details.SkillPoint && (def.Country == 0 || def.Country == 2 || def.Country >= 6) {
			skills = append(skills, def)
		}
	}
	sort.Slice(skills, func(i, k int) bool { return skills[i].SkillID < skills[k].SkillID })
	if len(skills) == 0 {
		j.t.Fatal("no eligible earned skill")
	}
	skill := skills[0]
	before := p.state.Details.SkillPoint
	p.send(proto.LearnSkillPacket(skill.SkillID, 1))
	j.wait("learned skill", 10*time.Second, func() bool {
		for _, s := range p.state.Skills {
			if s.ID == skill.SkillID {
				return true
			}
		}
		return false
	})
	saved, err := db.CharacterByID(p.id)
	if err != nil || saved.SkillPoint != before-uint16(skill.SkillPointCost) {
		j.t.Fatal("durable skill point debit mismatch", err)
	}
}

type journeySaved struct {
	uid       uint32
	character db.Character
	items     []db.Item
	skills    []db.CharacterSkill
	quests    []db.FinishedQuest
}

func (j *partyJourney) snapshot() []journeySaved {
	var result []journeySaved
	for _, p := range j.players {
		ch, err := db.CharacterByID(p.id)
		if err != nil {
			j.t.Fatal(err)
		}
		items, err := db.CharacterItems(p.id)
		if err != nil {
			j.t.Fatal(err)
		}
		skills, err := db.CharacterSkills(p.id)
		if err != nil {
			j.t.Fatal(err)
		}
		quests, err := db.FinishedQuests(p.id)
		if err != nil {
			j.t.Fatal(err)
		}
		result = append(result, journeySaved{p.uid, ch, items, skills, quests})
	}
	return result
}
func (j *partyJourney) compareSnapshot(saved []journeySaved) {
	current := j.snapshot()
	for i, before := range saved {
		after := current[i]
		if after.character.Level != before.character.Level || after.character.Exp != before.character.Exp || after.character.Money != before.character.Money || after.character.StatPoint != before.character.StatPoint || after.character.SkillPoint != before.character.SkillPoint || after.character.Str != before.character.Str || after.character.Dex != before.character.Dex || after.character.Rec != before.character.Rec || after.character.Int != before.character.Int || after.character.Wis != before.character.Wis || after.character.Luc != before.character.Luc || after.character.MapID != before.character.MapID || after.character.PosX != before.character.PosX || after.character.PosZ != before.character.PosZ || !reflect.DeepEqual(after.items, before.items) || !reflect.DeepEqual(after.skills, before.skills) || !reflect.DeepEqual(after.quests, before.quests) {
			j.t.Fatalf("restart changed committed state player=%d before=%+v after=%+v", i, before.character, after.character)
		}
		p := j.players[i]
		if p.state.Slots[0].Level != before.character.Level || p.state.Details.Exp != before.character.Exp || p.state.Details.Money != before.character.Money || p.state.FinishedQuests[1] != 1 {
			j.t.Fatal("reloaded client differs from committed state")
		}
	}
}
