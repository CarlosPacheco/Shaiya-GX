package proto

import "testing"

func TestVehicleStateCodec(t *testing.T) {
	for _, state := range []uint8{VehicleOff, VehicleCasting, VehicleDriver, VehiclePassenger} {
		want := VehicleState{ID: 42, State: state, Race: 3, Model: 0, Speed: 5, Seats: 2, Driver: 42, Passenger: 43, CastMS: 2000, CooldownMS: 20000}
		p := VehicleStatePacket(want)
		got, err := DecodeVehicleState(p)
		if err != nil || got != want {
			t.Fatal(got, err)
		}
		if _, err = DecodeVehicleState(p[:len(p)-1]); err == nil {
			t.Fatal("truncated state accepted")
		}
	}
}
