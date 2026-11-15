package proto

import "fmt"

// Native requests and invitation messages. 0x021C is also the server's
// passenger attachment notification; direction determines its layout.
const (
	VehicleCast    Opcode = 0x0217
	VehicleLeave   Opcode = 0x021C
	VehicleInvite  Opcode = 0x021D
	VehicleAnswer  Opcode = 0x021E
	GXVehicleState Opcode = 0x7F10
)

const (
	VehicleOff uint8 = iota
	VehicleCasting
	VehicleDriver
	VehiclePassenger
)

// VehicleState carries independent mount metadata without overloading the
// native appearance byte or changing any existing packet layout.
type VehicleState struct {
	ID                               uint32
	State, Race, Model, Speed, Seats uint8
	Driver, Passenger                uint32
	CastMS, CooldownMS               uint32
}

func VehicleStatePacket(v VehicleState) []byte {
	w := NewWriter(GXVehicleState)
	w.U32(v.ID)
	w.U8(v.State)
	w.U8(v.Race)
	w.U8(v.Model)
	w.U8(v.Speed)
	w.U8(v.Seats)
	w.U32(v.Driver)
	w.U32(v.Passenger)
	w.U32(v.CastMS)
	w.U32(v.CooldownMS)
	return w.Bytes()
}
func DecodeVehicleState(p []byte) (VehicleState, error) {
	if len(p) != 27 || Op(p) != GXVehicleState {
		return VehicleState{}, fmt.Errorf("invalid vehicle state size")
	}
	r := NewReader(p)
	v := VehicleState{ID: r.U32(), State: r.U8(), Race: r.U8(), Model: r.U8(), Speed: r.U8(), Seats: r.U8(), Driver: r.U32(), Passenger: r.U32(), CastMS: r.U32(), CooldownMS: r.U32()}
	if v.State > VehiclePassenger || v.Race > 3 || (v.State >= VehicleDriver && (v.Speed < 4 || v.Speed > 5 || v.Seats < 1 || v.Seats > 2 || v.Driver == 0)) {
		return v, fmt.Errorf("invalid vehicle state")
	}
	return v, r.Err()
}
func VehicleInvitePacket(id uint32) []byte {
	w := NewWriter(VehicleInvite)
	w.U32(id)
	return w.Bytes()
}

// The original answer byte is zero for acceptance.
func VehicleAnswerPacket(accept bool) []byte {
	w := NewWriter(VehicleAnswer)
	if accept {
		w.U8(0)
	} else {
		w.U8(1)
	}
	return w.Bytes()
}
func VehicleLeavePacket() []byte { return NewWriter(VehicleLeave).Bytes() }
