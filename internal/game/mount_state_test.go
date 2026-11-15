package game

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestCanonicalMountStateNeverTogglesOnAcknowledgement(t *testing.T) {
	s := New()
	s.SelfID = 42
	for _, state := range []uint8{proto.VehicleCasting, proto.VehicleDriver, proto.VehicleOff} {
		v := proto.VehicleState{ID: 42, State: state, Race: 0, Model: 0, Speed: 4, Seats: 1, Driver: 42}
		s.Apply(proto.VehicleStatePacket(v))
		// Native result arrives after dismount and before canonical mount completion.
		w := proto.NewWriter(proto.UseVehicle)
		w.U8(1)
		w.U8(0)
		s.Apply(w.Bytes())
		if s.Self.Mounted != (state == proto.VehicleDriver) {
			t.Fatal("native result changed canonical state", state)
		}
	}
}
