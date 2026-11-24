package world

import (
	"sync"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/movespeed"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

const vehicleBoardRange float32 = 10
const vehicleCastDuration = 2 * time.Second

var vehicleMu sync.Mutex
var vehicleCooldowns = map[uint32]time.Time{}

type vehiclePresence struct {
	state                              proto.VehicleState
	castEnd                            time.Time
	inviteFrom, inviteTo, inviteDriver uint32
	inviteEnd                          time.Time
}

func remainingVehicleMS(end, now time.Time) uint32 {
	if end.IsZero() || !end.After(now) {
		return 0
	}
	return uint32(end.Sub(now) / time.Millisecond)
}
func (p *ConnectedPlayer) vehicleState() proto.VehicleState {
	vehicleMu.Lock()
	defer vehicleMu.Unlock()
	v := p.ride.state
	v.ID = p.CharacterID
	v.CastMS = remainingVehicleMS(p.ride.castEnd, time.Now())
	v.CooldownMS = remainingVehicleMS(vehicleCooldowns[p.CharacterID], time.Now())
	return v
}
func publishVehicle(p *ConnectedPlayer) {
	if p == nil || p.out == nil {
		return
	}
	v := p.vehicleState()
	frame := buildVehicleState(v)
	p.Send(frame, "GX vehicle state")
	for _, o := range PlayersOnMap(p.MapID()) {
		if o != p && o.canSee(p) && o.knows(viewUser, p.CharacterID) {
			o.Send(frame, "GX vehicle state")
		}
	}
}

// Only shared presence changes here; each session applies its own position.
func clearVehicle(p *ConnectedPlayer) {
	if p == nil {
		return
	}
	vehicleMu.Lock()
	ids := []uint32{p.ride.state.Driver, p.ride.state.Passenger, p.ride.inviteFrom, p.ride.inviteTo}
	p.ride.state = proto.VehicleState{ID: p.CharacterID}
	p.ride.castEnd = time.Time{}
	p.ride.inviteFrom = 0
	p.ride.inviteTo = 0
	p.ride.inviteDriver = 0
	var peers []*ConnectedPlayer
	for _, id := range ids {
		if id == 0 || id == p.CharacterID {
			continue
		}
		q := FindPlayerByID(id)
		if q == nil {
			continue
		}
		if q.ride.state.Driver == p.CharacterID && q.ride.state.State == proto.VehiclePassenger {
			q.ride.state = proto.VehicleState{ID: q.CharacterID}
		}
		if q.ride.state.Passenger == p.CharacterID {
			q.ride.state.Passenger = 0
		}
		if q.ride.inviteFrom == p.CharacterID {
			q.ride.inviteFrom = 0
			q.ride.inviteDriver = 0
		}
		if q.ride.inviteTo == p.CharacterID {
			q.ride.inviteTo = 0
		}
		peers = append(peers, q)
	}
	vehicleMu.Unlock()
	p.setMountShape(0)
	publishVehicle(p)
	for _, q := range peers {
		publishVehicle(q)
	}
}
func (s *Session) mountDefinition() (*db.Item, *db.ItemDef) {
	it := equippedItem(s.items, mountSlot)
	if it == nil || it.Type != 42 {
		return nil, nil
	}
	return it, db.MustDefs().Item(it.Type, it.TypeID)
}
func validMount(d *db.ItemDef) bool {
	return d != nil && d.Type == 42 && d.Speed >= 4 && d.Speed <= 5 && d.AttackTime >= 1 && d.AttackTime <= 2 && d.Range >= 0 && d.Range < 256
}
func (s *Session) vehicleResult(ok, on bool) {
	w := packet.New(packet.UseVehicle)
	w.Bool(ok)
	w.Bool(on)
	s.send(w.MustFinish(), "0x0216 vehicle result")
}

// mountForbiddenMap reports maps that do not allow mounts: dungeons (Map.ini
// MapType D), including their party and guild instances.
func mountForbiddenMap(mapID uint16) bool {
	z, ok := mapConfig.Zones[realMap(mapID)]
	return ok && z.Dungeon
}

// keepsMountAcrossTeleport lets a lone rider keep the mount through every kind
// of travel (portals, gatekeepers, GM moves, town return, queued moves) when the
// destination allows mounts. Dungeons, a driver with a passenger and rebirth
// leave the character on foot.
func (s *Session) keepsMountAcrossTeleport(mapID uint16, label string) bool {
	if s.player == nil || !s.mounted || s.dead || mountForbiddenMap(mapID) || label == "rebirth" {
		return false
	}
	v := s.player.vehicleState()
	return v.State == proto.VehicleDriver && v.Passenger == 0
}

func (s *Session) handleUseVehicle() {
	if s.char == nil || s.player == nil {
		return
	}
	v := s.player.vehicleState()
	if v.State != proto.VehicleOff {
		s.dismount()
		s.vehicleResult(true, false)
		return
	}
	it, d := s.mountDefinition()
	vehicleMu.Lock()
	cooldown := vehicleCooldowns[s.char.ID]
	vehicleMu.Unlock()
	if !validMount(d) || mountForbiddenMap(s.char.MapID) || s.dead || s.cast != nil || s.player.IsSitting() || s.movement.ladder.Active || s.vehicleBlocked() || time.Now().Before(cooldown) || !canEquip(s.char, d, mountSlot, s.items) {
		s.vehicleResult(false, false)
		return
	}
	s.cancelMove()
	s.autoAttack = false
	s.mountItemUID = it.UID
	s.movement.motion = 0
	s.movement.credit = 0
	vehicleMu.Lock()
	s.player.ride.state = proto.VehicleState{ID: s.char.ID, State: proto.VehicleCasting, Race: s.char.Race, Model: uint8(d.Range), Speed: d.Speed, Seats: d.AttackTime}
	s.player.ride.castEnd = time.Now().Add(vehicleCastDuration)
	vehicleMu.Unlock()
	w := packet.New(packet.Opcode(proto.VehicleCast))
	w.U32(s.char.ID)
	s.send(w.MustFinish(), "0x0217 mount cast")
	s.sendToViewers(w.MustFinish(), "0x0217 mount cast")
	publishVehicle(s.player)
}
func (s *Session) dismount() {
	if s.player != nil {
		clearVehicle(s.player)
	}
	s.mounted = false
	s.mountItemUID = 0
}
func (s *Session) cancelVehicleCast() {
	if s.player != nil && s.player.vehicleState().State == proto.VehicleCasting {
		s.dismount()
		s.vehicleResult(false, false)
	}
}
func (s *Session) tickVehicle(now time.Time) {
	if s.char == nil || s.player == nil {
		return
	}
	v := s.player.vehicleState()
	if v.State != proto.VehicleOff && (s.dead || s.vehicleBlocked()) {
		if v.State == proto.VehicleCasting {
			s.cancelVehicleCast()
		} else {
			s.dismount()
		}
		return
	}
	if v.State == proto.VehicleCasting {
		it, d := s.mountDefinition()
		if it == nil || it.UID != s.mountItemUID || !validMount(d) || !canEquip(s.char, d, mountSlot, s.items) {
			s.cancelVehicleCast()
			return
		}
		vehicleMu.Lock()
		ready := !now.Before(s.player.ride.castEnd)
		vehicleMu.Unlock()
		if ready {
			cooldown := now.Add(time.Duration(d.ReqIg) * time.Second)
			vehicleMu.Lock()
			s.player.ride.state.Model = uint8(d.Range)
			s.player.ride.state.Speed = d.Speed
			s.player.ride.state.Seats = d.AttackTime
			s.player.ride.state.State = proto.VehicleDriver
			s.player.ride.state.Driver = s.char.ID
			s.player.ride.castEnd = time.Time{}
			for id, end := range vehicleCooldowns {
				if !end.After(now) {
					delete(vehicleCooldowns, id)
				}
			}
			vehicleCooldowns[s.char.ID] = cooldown
			vehicleMu.Unlock()
			s.mounted = true
			s.vehicleResult(true, true)
			publishVehicle(s.player)
		}
		return
	}
	s.mounted = v.State >= proto.VehicleDriver
	if v.State == proto.VehicleDriver {
		it, d := s.mountDefinition()
		if it == nil || it.UID != s.mountItemUID || !validMount(d) {
			s.dismount()
		}
		return
	}
	if v.State == proto.VehiclePassenger {
		driver := FindPlayerByID(v.Driver)
		if driver == nil || driver.MapID() != s.char.MapID || !driver.IsAlive() {
			s.dismount()
			return
		}
		dv := driver.vehicleState()
		if dv.State != proto.VehicleDriver || dv.Passenger != s.char.ID {
			s.dismount()
			return
		}
		ch := driver.Snapshot()
		s.char.PosX, s.char.PosY, s.char.PosZ, s.char.Dir = ch.PosX, ch.PosY, ch.PosZ, ch.Dir
		s.movement.geometry = nil
		s.syncPlayer()
		s.positionDirty = true
		s.send(buildCharacterMove(s.char, 0, uint16(ch.Dir)), "passenger follows vehicle")
		s.player.moveViewCell(ch.PosX, ch.PosZ)
		s.refreshView()
		s.player.UpdateMotion(0, uint16(ch.Dir))
		s.sendToViewers(buildCharacterMove(s.char, 0, uint16(ch.Dir)), "passenger follows vehicle")
	}
}
func (s *Session) mountSpeed() float32 {
	if s.player != nil {
		v := s.player.vehicleState()
		if v.State >= proto.VehicleDriver {
			return movespeed.MountedUnits(v.Speed)
		}
	}
	return 7
}
func vehicleNear(a, b *ConnectedPlayer) bool {
	if a == nil || b == nil || a == b || a.MapID() != b.MapID() || !a.IsAlive() || !b.IsAlive() || a.IsSitting() || b.IsSitting() {
		return false
	}
	x, y := a.Snapshot(), b.Snapshot()
	dx, dy, dz := x.PosX-y.PosX, x.PosY-y.PosY, x.PosZ-y.PosZ
	return x.Race/2 == y.Race/2 && dx*dx+dy*dy+dz*dz <= vehicleBoardRange*vehicleBoardRange
}
func vehicleReply(p *ConnectedPlayer, code uint8) {
	if p != nil {
		w := packet.New(packet.Opcode(proto.VehicleAnswer))
		w.U8(code)
		p.Send(w.MustFinish(), "0x021E vehicle answer")
	}
}
func (s *Session) handleVehicleInvite(packetData []byte) {
	if s.player == nil || len(packetData) != 6 {
		return
	}
	id, _ := packet.U32(packetData[2:])
	other := FindPlayerByID(id)
	if s.cast != nil || s.dead || s.vehicleBlocked() || !vehicleNear(s.player, other) || !s.player.canSee(other) {
		vehicleReply(s.player, 2)
		return
	}
	vehicleMu.Lock()
	a, b := s.player, other
	driver := a
	if a.ride.state.State == proto.VehicleOff && b.ride.state.State == proto.VehicleDriver {
		driver = b
	}
	passenger := b
	if driver == b {
		passenger = a
	}
	now := time.Now()
	valid := driver.ride.state.State == proto.VehicleDriver && driver.ride.state.Seats == 2 && driver.ride.state.Passenger == 0 && passenger.ride.state.State == proto.VehicleOff && passenger.formLook.Load() == nil && !(a.ride.inviteTo != 0 && now.Before(a.ride.inviteEnd)) && !(b.ride.inviteFrom != 0 && now.Before(b.ride.inviteEnd))
	if valid {
		a.ride.inviteTo = b.CharacterID
		a.ride.inviteEnd = now.Add(30 * time.Second)
		b.ride.inviteFrom = a.CharacterID
		b.ride.inviteDriver = driver.CharacterID
		b.ride.inviteEnd = a.ride.inviteEnd
	}
	vehicleMu.Unlock()
	if !valid {
		vehicleReply(a, 2)
		return
	}
	w := packet.New(packet.Opcode(proto.VehicleInvite))
	w.U32(a.CharacterID)
	b.Send(w.MustFinish(), "0x021D go together invitation")
}
func (s *Session) handleVehicleAnswer(data []byte) {
	if s.player == nil || len(data) != 3 {
		return
	}
	vehicleMu.Lock()
	id, driverID, deadline := s.player.ride.inviteFrom, s.player.ride.inviteDriver, s.player.ride.inviteEnd
	s.player.ride.inviteFrom = 0
	s.player.ride.inviteDriver = 0
	from := FindPlayerByID(id)
	if from != nil {
		from.ride.inviteTo = 0
	}
	vehicleMu.Unlock()
	if from == nil || id == 0 || time.Now().After(deadline) {
		vehicleReply(s.player, 2)
		return
	}
	if data[2] != 0 {
		vehicleReply(from, 1)
		vehicleReply(s.player, 1)
		return
	}
	driver, passenger := from, s.player
	if driverID == s.player.CharacterID {
		driver, passenger = s.player, from
	}
	if s.cast != nil || s.dead || s.vehicleBlocked() || !vehicleNear(driver, passenger) || !driver.canSee(passenger) || !passenger.canSee(driver) {
		vehicleReply(from, 2)
		vehicleReply(s.player, 2)
		return
	}
	vehicleMu.Lock()
	dv := driver.ride.state
	valid := dv.State == proto.VehicleDriver && dv.Seats == 2 && dv.Passenger == 0 && passenger.ride.state.State == proto.VehicleOff
	if valid {
		driver.ride.state.Passenger = passenger.CharacterID
		dv.ID = passenger.CharacterID
		dv.State = proto.VehiclePassenger
		dv.Passenger = passenger.CharacterID
		passenger.ride.state = dv
	}
	vehicleMu.Unlock()
	if !valid {
		vehicleReply(from, 2)
		vehicleReply(s.player, 2)
		return
	}
	vehicleReply(from, 0)
	vehicleReply(s.player, 0)
	publishVehicle(driver)
	publishVehicle(passenger)
}

func (s *Session) vehicleBlocked() bool {
	a, move, action := s.blocks()
	return a != 0 || move != 0 || action != 0 || s.activeForm().kind != proto.FormNone
}

func (s *Session) ridingVehicle() bool {
	return s.mounted || s.player != nil && s.player.vehicleState().State >= proto.VehicleDriver
}

func buildVehicleState(v proto.VehicleState) []byte {
	raw := proto.VehicleStatePacket(v)
	w := packet.New(packet.Opcode(proto.GXVehicleState))
	w.Bytes(raw[2:])
	return w.MustFinish()
}
