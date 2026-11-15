//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"math"
	"os"
	"strings"
	"testing"
)

func TestGMMapTeleportAgainstIsolatedGX(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_GM_MAP") != "1" {
		t.Skip("requires disposable GM fixture")
	}
	st, _, send, wait := testworld.Enter(t)
	wait(func() bool { return st.Self.IsGM })
	if v, ok := st.OJLimits[[2]uint8{16, 55}]; !ok || v.Stats != 3 || v.Value != 13 {
		t.Fatal("missing authoritative OJ catalog", v, ok)
	}
	for _, tc := range []struct {
		mapID      uint16
		x, z, size float32
	}{{1, 580, 1762, 2048}, {3, 230, 60, 1024}, {1, 578, 1762, 2048}} {
		n := len(st.Chat)
		send(proto.GMMapTeleportPacket(tc.mapID, tc.x/tc.size, 1-tc.z/tc.size))
		wait(func() bool { return len(st.Chat) > n })
		message := st.Chat[len(st.Chat)-1].Text
		if !strings.Contains(message, "GM: map ") {
			t.Fatal("teleport refused", message)
		}
		if math.Abs(float64(st.Self.X-tc.x)) > .002 || math.Abs(float64(st.Self.Z-tc.z)) > .002 {
			t.Fatal("destination shifted", st.Self)
		}
		if tc.mapID == 3 && (st.Self.Y < 10 || st.Self.Y > 16) {
			t.Fatal("dungeon height missing", st.Self.Y)
		}
		t.Logf("Map %d exact X/Z %.3f/%.3f; authoritative Y %.3f", tc.mapID, st.Self.X, st.Self.Z, st.Self.Y)
	}
	oldX, oldY, oldZ := st.Self.X, st.Self.Y, st.Self.Z
	for _, p := range [][]byte{proto.GMMapTeleportPacket(1, float32(math.NaN()), 0), proto.GMMapTeleportPacket(65535, .5, .5)} {
		n := len(st.Chat)
		send(p)
		wait(func() bool { return len(st.Chat) > n })
		if st.Self.X != oldX || st.Self.Y != oldY || st.Self.Z != oldZ {
			t.Fatal("invalid request moved player")
		}
	}
}
