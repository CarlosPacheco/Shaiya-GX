package scene

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/vehicle"
)

// Vehicle uses the native race-specific MON records and the same skinning
// pipeline as creatures. Seat bone indices are the original mount bindings
// recovered by Phoenix and loaded from the shared Vehicle/bones.ini.
type Vehicle struct {
	*CreatureActor
	seatBone      int
	passengerBone int
	board         bool
}

// These native MON records are the standing board variants.
func VehicleAlternate(shape uint8) bool { return shape == 6 || shape == 9 || shape == 11 }

func (s *Assets) BuildVehicle(race, shape uint8) (*Vehicle, error) {
	if race >= 4 {
		return nil, fmt.Errorf("invalid mount race %d", race)
	}
	if s.vehicles[race] == nil {
		folders := [4]string{"Hu", "El", "Vi", "De"}
		b, err := s.Arc.ReadFile("Vehicle/Vehicle_" + folders[race] + "_01.MON")
		if err != nil {
			return nil, err
		}
		s.vehicles[race], err = model.ParseMON(b)
		if err != nil {
			return nil, err
		}
	}
	rows := s.vehicles[race]
	if int(shape) >= len(rows) {
		return nil, fmt.Errorf("mount shape %d out of range", shape)
	}
	actor, err := s.buildCreature("Vehicle", rows[shape], rows[shape].Name)
	if err != nil {
		return nil, err
	}
	if err := actor.SetAction(8); err != nil {
		return nil, err
	}
	raw, err := s.Arc.ReadFile("Vehicle/bones.ini")
	if err != nil {
		return nil, err
	}
	bindings, err := vehicle.ParseBones(raw)
	if err != nil {
		return nil, err
	}
	bones, ok := bindings[race][shape]
	if !ok {
		return nil, fmt.Errorf("missing seat bindings for race %d MON %d", race, shape)
	}
	poses := actor.finals()
	if bones.Rider >= len(poses) || bones.Passenger >= len(poses) {
		return nil, fmt.Errorf("vehicle seat bone outside skeleton")
	}
	return &Vehicle{CreatureActor: actor, seatBone: bones.Rider, passengerBone: bones.Passenger, board: VehicleAlternate(shape)}, nil
}

func (v *Vehicle) SetMoving(moving bool) { v.SetLocomotion(moving, false) }
func (v *Vehicle) SetLocomotion(moving, walking bool) {
	if v == nil {
		return
	}
	action := 8
	if moving {
		action = 1
		if walking {
			action = 0
		}
	}
	if err := v.SetAction(action); err != nil {
		v.assets.Missing[err.Error()] = true
	}
}

// Seat uses the same cached pose as drawing, including before the first draw.
func (v *Vehicle) Seat() render.V3 { return v.RiderSeat(false) }
func (v *Vehicle) RiderSeat(passenger bool) render.V3 {
	if v == nil {
		return render.V3{}
	}
	bone := v.seatBone
	if passenger {
		bone = v.passengerBone
	}
	finals := v.finals()
	if bone >= 0 && bone < len(finals) {
		m := finals[bone].Mul(v.World)
		return render.V3{m[12], m[13], m[14]}
	}
	m := v.World
	if v.board {
		return render.V3{m[12], m[13] + .877, m[14]}
	}
	return render.V3{m[12], m[13] + max(.1, v.visual.Height*.7), m[14]}
}
