package scene

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/vehicle"
	"math"
)

// Vehicle uses the native race-specific MON records and the same skinning
// pipeline as creatures. Seat bone indices are the original mount bindings
// recovered by Phoenix and loaded from the shared Vehicle/bones.ini.
type Vehicle struct {
	*CreatureActor
	seatBone                          int
	passengerBone                     int
	board                             bool
	moving, walking, reverse, jumping bool
	jumpPending, jumpRecovery         bool
	idleTime                          float32
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
	if err := actor.SetAction(6); err != nil {
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

func (v *Vehicle) SetMoving(moving bool)              { v.SetLocomotion(moving, false) }
func (v *Vehicle) SetLocomotion(moving, walking bool) { v.SetMotion(moving, walking, false, false) }
func (v *Vehicle) PlayingJump() bool                  { return v != nil && v.action == 2 }

func (v *Vehicle) SetMotion(moving, walking, reverse, jumping bool) {
	if v != nil {
		v.jumpPending = v.jumpPending || jumping && !v.jumping
		v.moving, v.walking, v.reverse, v.jumping = moving, walking, reverse, jumping
	}
}

// MON 6 is breathing; 8 is the occasional gesture, not the resting loop.
// Phoenix's native vehicle audit uses an 11-second wait and 3-second window.
func (v *Vehicle) Update(dt float32) {
	if v == nil {
		return
	}
	if v.jumpPending {
		v.jumpPending, v.jumpRecovery, v.idleTime = false, false, 0
		if err := v.setAction(2, true, true); err != nil {
			v.assets.Missing[err.Error()] = true
		}
	}
	if v.action == 2 {
		v.CreatureActor.Update(dt)
		if end := v.clip.Duration(); v.Time >= end {
			v.Time = end
			// EP5.4 starts MON 2 immediately and blends to MON 6 at its end.
			if err := v.SetAction(6); err != nil {
				v.assets.Missing[err.Error()] = true
			} else {
				v.jumpRecovery = true
			}
		}
		return
	}
	if v.jumpRecovery && v.blendLeft > 0 {
		v.CreatureActor.Update(dt)
		return
	}
	v.jumpRecovery = false
	action := 6
	if v.moving {
		action = 1
		if v.walking || v.reverse {
			action = 0
		}
		v.idleTime = 0
	} else {
		v.idleTime += max(0, dt)
		if v.idleTime >= 14 {
			v.idleTime = 0
		} else if v.idleTime >= 11 {
			action = 8
		}
	}
	if err := v.SetAction(action); err != nil {
		v.assets.Missing[err.Error()] = true
		return
	}
	previous := v.Time
	v.CreatureActor.Update(dt)
	if v.clip != nil {
		if duration := v.clip.Duration(); duration > 0 {
			if v.reverse && action == 0 {
				v.Time = float32(math.Mod(float64(2*previous-v.Time), float64(duration)))
				if v.Time < 0 {
					v.Time += duration
				}
			} else if action == 8 {
				v.Time = min(v.Time, max(0, duration-.001))
			}
		}
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

// mountStepFrames are the frames at which a mount's hooves land, from the
// tables at Game.exe 0x4192e4 (walk) and 0x4195b1 (run): rows are the mount
// action (0 walk, 1 run), columns the rider's race (human horse, elf and vail
// puma, death eater basilisk). A threshold of frame 0 can never be crossed.
var mountStepFrames = [2][4][]float32{
	{{7, 17, 2, 13}, {14, 4}, {15, 5, 20, 10}, {2, 12, 9, 19}},
	{{15, 18, 7, 11}, {15, 11, 2, 18}, {9, 6, 17, 14}, {15, 0, 12, 17}},
}

// StepCursor is the playing action and clip time, for MountStepCrossed.
func (v *Vehicle) StepCursor() (action int, seconds float32) {
	if v == nil || v.CreatureActor == nil {
		return -1, 0
	}
	return v.action, v.Time
}

// MountStepCrossed reports that the cursor moved over a hoof frame between two
// StepCursor samples. A new action restarts at its first frame.
func MountStepCrossed(prevAction int, prevTime float32, action int, time float32, race uint8) bool {
	if action < 0 || action > 1 || race > 3 {
		return false
	}
	if prevAction != action {
		prevTime = -1
	}
	for _, frame := range mountStepFrames[action][race] {
		t := frame / 30
		if frame == 0 {
			continue
		}
		if time < prevTime {
			if t > prevTime || t <= time {
				return true
			}
		} else if prevTime < t && t <= time {
			return true
		}
	}
	return false
}
