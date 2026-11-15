package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"math"
	"strings"
)

type vehicleView struct {
	effectBudget              int
	inviteSerial, replySerial uint64
	expires, cooldownEnd      float64
	castState                 uint8
	peerName                  string
	castEffect                *scene.Effect
	pending                   bool
	deadline                  float64
}

func (a *App) requestVehicle() {
	if a.state == nil || a.world == nil || a.state.Self.Dead || a.vehicle.pending || a.inventory.pending || a.shop.pending {
		return
	}
	if !a.state.Self.Mounted && a.state.Self.Vehicle.State != proto.VehicleCasting {
		if a.clock < a.vehicle.cooldownEnd {
			return
		}
		item, ok := a.state.Inventory[[2]uint8{0, 13}]
		if !ok || item.Type != 42 {
			a.questMessage(a.chatEntityName(a.state.SelfID) + " " + a.msg.Get(5245))
			return
		}
	}
	if err := a.world.Send(proto.UseVehiclePacket()); err != nil {
		a.backToLogin("Unable to use mount.")
		return
	}
	a.vehicle.pending, a.vehicle.deadline = true, a.clock+10
}
func (a *App) vehicleCommand(text string) bool {
	if !strings.EqualFold(strings.TrimSpace(text), "/mount") {
		return false
	}
	a.requestVehicle()
	return true
}
func (a *App) confirmedVehicle(p []byte) {
	if proto.Op(p) == proto.GXVehicleState {
		v, err := proto.DecodeVehicleState(p)
		if err != nil || v.ID != a.state.SelfID {
			return
		}
		a.vehicle.cooldownEnd = a.clock + float64(v.CooldownMS)/1000
		if v.State != proto.VehicleCasting {
			a.vehicle.pending = false
		}
		if v.State == proto.VehicleCasting && a.vehicle.castState != v.State && a.wv != nil && a.wv.self != nil {
			a.wv.self.PlayCast(85, 2)
			a.playSound("ef_in_atrbu01.wav", a.wv.pos[0], a.wv.pos[2])
			if a.clientSettings().SkillEffects {
				a.vehicle.castEffect, _ = scene.LoadEffect(a.assets, "Effect/IN_mc_yellow.EFT", 1)
			}
		}
		if v.State != proto.VehicleCasting && a.vehicle.castState == proto.VehicleCasting && a.wv != nil && a.wv.self != nil {
			a.wv.self.CancelCast()
			a.vehicle.castEffect = nil
		}
		if v.State >= proto.VehicleDriver && a.vehicle.castState < proto.VehicleDriver {
			a.questMessage(a.chatEntityName(a.state.SelfID) + " " + a.msg.Get(5243))
		}
		if v.State == proto.VehicleOff && a.vehicle.castState >= proto.VehicleDriver {
			a.questMessage(a.chatEntityName(a.state.SelfID) + " " + a.msg.Get(5244))
		}
		a.vehicle.castState = v.State
		return
	}
	if proto.Op(p) != proto.UseVehicle {
		return
	}
	r, err := proto.DecodeVehicleResult(p)
	if err != nil {
		return
	}
	a.vehicle.pending = false
	if r.Result == 0 {
		a.questMessage(a.chatEntityName(a.state.SelfID) + " " + a.msg.Get(5245))
	}
}

func (a *App) updateVehicleInvite(in *platform.Input, w, h int) bool {
	if a.vehicleMessage == nil || a.state == nil {
		return false
	}
	if a.vehicle.inviteSerial != a.state.VehicleInviteSerial {
		a.vehicle.inviteSerial = a.state.VehicleInviteSerial
		a.vehicle.expires = a.clock + 30
		a.vehicle.peerName = a.chatEntityName(a.state.VehicleInviteID)
		a.vehicleMessage.Open(a.chatEntityName(a.state.VehicleInviteID)+" "+a.msg.Get(5367), true)
	}
	if a.vehicle.replySerial != a.state.VehicleReplySerial {
		a.vehicle.replySerial = a.state.VehicleReplySerial
		id := 5370
		if a.state.VehicleReply == 0 {
			id = 5369
		} else if a.state.VehicleReply == 1 {
			id = 5368
		}
		a.questMessage(a.vehicle.peerName + " " + a.msg.Get(id))
		a.vehicleMessage.Visible = false
	}
	if !a.vehicleMessage.Visible {
		return false
	}
	if a.clock >= a.vehicle.expires {
		a.vehicleMessage.Visible = false
		return false
	}
	answer := a.vehicleMessage.Update(in, w, h)
	if answer != ui.MessageNone && a.world != nil {
		if err := a.world.Send(proto.VehicleAnswerPacket(answer == ui.MessageOK)); err != nil {
			a.backToLogin("Unable to answer mount invitation.")
		}
	}
	return true
}
func (a *App) driverMount(id uint32) *scene.Vehicle {
	if id == a.state.SelfID {
		return a.wv.selfMount
	}
	if c := a.wv.actors[id]; c != nil {
		return c.mount
	}
	return nil
}
func (a *App) attachVehiclePassengers() {
	attach := func(v proto.VehicleState, c *scene.Character) {
		if v.State != proto.VehiclePassenger || c == nil {
			return
		}
		if m := a.driverMount(v.Driver); m != nil {
			world := c.World
			height := float32(math.Sqrt(float64(world[4]*world[4] + world[5]*world[5] + world[6]*world[6])))
			pos := m.RiderSeat(true)
			for i := 0; i < 12; i++ {
				world[i] = m.World[i]
			}
			world[4] *= height
			world[5] *= height
			world[6] *= height
			world[12], world[13], world[14] = pos[0], pos[1], pos[2]
			c.World = world
		}
	}
	attach(a.state.Self.Vehicle, a.wv.self)
	for id, e := range a.state.Entities {
		if c := a.wv.actors[id]; c != nil {
			attach(e.Vehicle, c.character)
		}
	}
	if x := a.vehicle.castEffect; x != nil {
		x.SetTransform(render.Basis(a.wv.pos, a.wv.facing.Scale(-1), render.V3{0, 1, 0}))
		x.Update(float32(a.frameDT))
	}
}
