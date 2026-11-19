package world

import (
	"slices"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
)

type nopLog struct{}

func (nopLog) Printf(string, ...any) {}

// withTiers installs a two-tier table per country for a test.
func withTiers(t *testing.T) {
	oldStore := kcStore
	kcStore = func(uint32, uint32) error { return nil }
	kcMu.Lock()
	oldTiers, oldWorld := kcTiers, kcWorld
	var a, b kcTier
	a.Kill, b.Kill = 2, 4
	a.Status[1], a.Status[4] = 10, 5                   // +10% HP, +5 STR
	b.Status[1], b.Status[4], b.Status[10] = 10, 5, 20 // tiers keep the earlier values
	kcTiers = [2][]kcTier{{a, b}, {a, b}}
	kcMu.Unlock()
	now := time.Now()
	kcMu.Lock()
	setKillCountLocked(0, 0, now)
	setKillCountLocked(1, 0, now)
	kcMu.Unlock()
	t.Cleanup(func() {
		kcStore = oldStore
		kcMu.Lock()
		kcTiers, kcWorld = oldTiers, oldWorld
		kcMu.Unlock()
	})
}

// CWorld::AddKillCount: tiers by count, the top tier holds and resets the
// other country; UpdateKillCount resets it after 10 minutes.
func TestKillCountTiers(t *testing.T) {
	withTiers(t)
	AddKillCount(0, 1, nopLog{})
	if st := kcStatus(0); st[1] != 0 {
		t.Fatal("a tier applies below the first count")
	}
	AddKillCount(1, 3, nopLog{})
	AddKillCount(0, 1, nopLog{})
	if st := kcStatus(0); st[1] != 10 || st[4] != 5 || st[10] != 0 {
		t.Fatalf("tier 0 status %v", st[:11])
	}
	AddKillCount(0, 2, nopLog{}) // 4: top tier
	kcMu.Lock()
	light, dark, hold := kcWorld[0].count, kcWorld[1].count, kcWorld[0].resetAt
	kcMu.Unlock()
	if light != 4 || dark != 0 || hold.IsZero() {
		t.Fatalf("after the top tier light=%d dark=%d hold=%v", light, dark, hold)
	}
	AddKillCount(1, 5, nopLog{})
	if kcWorld[1].count != 0 {
		t.Fatal("kills counted during the hold")
	}
	TickKillCount(hold.Add(time.Second), nopLog{})
	if kcWorld[0].count != 0 || !kcWorld[0].resetAt.IsZero() {
		t.Fatalf("hold did not reset: %d", kcWorld[0].count)
	}
}

func TestLoadKillStatus(t *testing.T) {
	log, err := logx.New("gx_world_test")
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	kcMu.Lock()
	old := kcTiers
	kcMu.Unlock()
	defer func() { kcMu.Lock(); kcTiers = old; kcMu.Unlock() }()
	if err := loadKillStatus(log); err != nil {
		t.Fatal(err)
	}
	if len(kcTiers[0]) != 30 || !slices.Equal(kcTiers[0], kcTiers[1]) || kcTiers[0][0].Kill != 150 || kcMaxKill(0) != 12288 {
		t.Fatalf("unexpected migrated tiers: %d/%d", len(kcTiers[0]), len(kcTiers[1]))
	}
	last := kcTiers[0][29].Status
	if last[11] != 10 || last[10] != 10 || last[25] != 100 || last[24] != 5 || last[1] != 9 {
		t.Fatalf("top tier status %v", last)
	}
}

// The tier's percents land on the pools and the recovery.
func TestKillCountStats(t *testing.T) {
	withTiers(t)
	AddKillCount(0, 4, nopLog{})
	base := StatsFor(nil, EquipmentStats{})
	var e EquipmentStats
	applyKCStatus(&e, 0)
	st := StatsFor(nil, e)
	if st.MaxHP != base.MaxHP+kcPercent(base.MaxHP, 10) || st.Str != base.Str+5 {
		t.Fatalf("pools/stats %d->%d str %d->%d", base.MaxHP, st.MaxHP, base.Str, st.Str)
	}
	if want := recoverBaseStanding + kcPercent(st.MaxHP, 20); st.Recover[1][0] != want {
		t.Fatalf("standing HP recovery %d, want %d", st.Recover[1][0], want)
	}
}

// CUser::UpdateRecover / InitEquipment: 28 flat sitting and 4 standing per
// pool, nothing in combat, the sitting amount truncated.
func TestRecoveryBase(t *testing.T) {
	st := StatsFor(nil, EquipmentStats{GemRecover: [3]uint32{3, 2, 1}})
	if st.Recover[0] != [3]uint32{31, 30, 29} || st.Recover[1] != [3]uint32{7, 6, 5} || st.Recover[2] != [3]uint32{} {
		t.Fatalf("recover %v", st.Recover)
	}
	// 5% of 1010 = 50.5 + 28 -> 78 (chopped, not rounded).
	if got := sittingRecovery(1010, 0, 28); got != 78 {
		t.Fatalf("sitting recovery %d, want 78", got)
	}
}

func TestObeliskDebitPreservesNativeTierHysteresisAndNeverUnderflows(t *testing.T) {
	withTiers(t)
	kcMu.Lock()
	third := kcTiers[0][1]
	third.Kill = 6
	kcTiers[0] = append(kcTiers[0], third)
	setKillCountLocked(0, 3, time.Now())
	kcMu.Unlock()
	AddKillCount(0, 2, nopLog{}) // Five counts, tier1; six would activate blessing.
	SubKillCount(0, 2, nopLog{})
	if kcWorld[0].count != 3 || kcWorld[0].index != 1 {
		t.Fatal("debit lost the original previous-tier hysteresis")
	}
	SubKillCount(0, 2, nopLog{})
	if kcWorld[0].count != 1 || kcWorld[0].index != -1 {
		t.Fatal("debit failed to reselect below previous tier")
	}
	SubKillCount(0, 100, nopLog{})
	if kcWorld[0].count != 0 {
		t.Fatal("country count underflow")
	}
	AddKillCount(1, 4, nopLog{})
	SubKillCount(1, 100, nopLog{})
	if kcWorld[1].count != 4 {
		t.Fatal("debit modified active blessing")
	}
}

func TestBlessingSnapshotUsesServerThresholdTimerAndBonuses(t *testing.T) {
	withTiers(t)
	AddKillCount(0, 1, nopLog{})
	snapshot, err := proto.DecodeBlessingSnapshot(buildBlessingSnapshot(0)[2:])
	if err != nil || !snapshot.Entry || snapshot.Count != 1 || snapshot.Maximum != 4 || snapshot.Bonuses != [27]uint16{} {
		t.Fatalf("entry before first tier: %+v, %v", snapshot, err)
	}
	AddKillCount(0, 3, nopLog{})
	kcMu.Lock()
	live := blessingSnapshotLocked(0, false)
	kcMu.Unlock()
	snapshot, err = proto.DecodeBlessingSnapshot(live[2:])
	if err != nil || snapshot.Entry || snapshot.Count != 4 || snapshot.RemainingMS < 590000 ||
		snapshot.Bonuses[0] != 10 || snapshot.Bonuses[3] != 5 || snapshot.Bonuses[9] != 20 {
		t.Fatalf("maximum transition: %+v, %v", snapshot, err)
	}
	TickKillCount(kcWorld[0].resetAt.Add(time.Second), nopLog{})
	snapshot, err = proto.DecodeBlessingSnapshot(buildBlessingSnapshot(0)[2:])
	if err != nil || snapshot.Count != 0 || snapshot.RemainingMS != 0 || snapshot.Bonuses != [27]uint16{} {
		t.Fatalf("expired blessing: %+v, %v", snapshot, err)
	}
}
