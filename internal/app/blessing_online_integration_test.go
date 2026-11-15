//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"os"
	"strconv"
	"testing"
)

func TestBlessingSnapshotAgainstIsolatedGX(t *testing.T) {
	expected := os.Getenv("SHAIYA_TEST_BLESSING_COUNT")
	if expected == "" {
		t.Skip("requires disposable blessing world fixture")
	}
	count, err := strconv.ParseUint(expected, 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	st, _, send, wait := testworld.Enter(t)
	send(proto.ChatPacket(proto.ChannelNormal, false, "/turbo"))
	wait(func() bool { return st.GMTurbo }) // barrier after every initial state packet
	if !st.Blessing.Known || st.Blessing.Count != uint32(count) {
		t.Fatalf("DB blessing overwritten during entry: %+v", st.Blessing)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := &App{state: st, arc: arc}
	a.loadBlessing()
	if a.blessingMax[0] != 12288 || a.blessingMax[1] != 12288 {
		t.Fatal("original client thresholds")
	}
	if count == 12288 {
		remaining := st.Blessing.RemainingMS
		if remaining < 590000 || remaining > 600000 {
			t.Fatal("missing maximum remainder", remaining)
		}
		a.confirmedBlessing(proto.BlessInit)
		if a.blessingBannerUntil != 0 || st.Blessing.RemainingMS != remaining {
			t.Fatal("reentry replayed blessing activation")
		}
		st.Blessing.Advance(1.5)
		if st.Blessing.RemainingMS != remaining-1500 {
			t.Fatal("maximum UI countdown")
		}
	} else if count == 1200 && len(a.blessingLines()) != 4 {
		t.Fatal("current UI benefits differ from saved tier")
	}
	t.Logf("encrypted DB blessing snapshot count=%d remaining_ms=%d; no placeholder overwrite or activation replay", count, st.Blessing.RemainingMS)
}
