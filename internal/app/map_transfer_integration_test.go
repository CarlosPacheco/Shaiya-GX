//go:build integration

package app

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
)

// Uses only the isolated account/database provisioned by the map-transfer probe.
func TestRepeatedMapTransfersAgainstIsolatedGX(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_MAP_TRANSFERS") != "1" {
		t.Skip("requires isolated GM fixture")
	}
	st, w, send, _ := testworld.Enter(t)
	a := inputApp()
	a.state, a.world = st, w
	wait := func(cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for !cond() {
			a.pumpWorld()
			if a.world == nil || a.state != st {
				t.Fatal("disconnected during repeated map transfer")
			}
			if time.Now().After(deadline) {
				t.Fatalf("transfer barrier timed out: map=%d turbo=%v", st.MapID, st.GMTurbo)
			}
			time.Sleep(time.Millisecond)
		}
	}
	for _, mapID := range []uint16{18, 45, 1, 18, 18, 45, 1} {
		send(proto.ChatPacket(proto.ChannelNormal, false, fmt.Sprintf("/cmove %d", mapID)))
		wait(func() bool { return st.MapID == mapID })
		// This command is admitted only after the load acknowledgement. Ping alone
		// would hide the regression because loading sessions still accept it.
		want := !st.GMTurbo
		send(proto.ChatPacket(proto.ChannelNormal, false, "/turbo"))
		wait(func() bool { return st.GMTurbo == want })
		if st.Phase != game.PhaseInWorld {
			t.Fatal("transfer did not remain in world")
		}
	}
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
	t.Log("Seven consecutive map commands, including same-map teleport, stayed connected and accepted gameplay commands")
}
