package game

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func blessingPacket(op proto.Opcode, country uint8, count, remaining uint32) []byte {
	w := proto.NewWriter(op)
	w.U8(country)
	w.U32(count)
	if op == proto.BlessInit {
		w.U32(remaining)
	}
	return w.Bytes()
}

func TestBlessingSnapshotAndBroadcast(t *testing.T) {
	s := New()
	s.Phase = PhaseLoading
	apply := func(p []byte) {
		t.Helper()
		if _, err := s.Apply(p); err != nil {
			t.Fatal(err)
		}
	}
	apply(blessingPacket(proto.BlessInit, 0, 1200, 500))
	apply(blessingPacket(proto.BlessInit, 0, 0, 0)) // every authoritative snapshot applies during loading
	if s.Blessing.Count != 0 || s.Blessing.RemainingMS != 0 {
		t.Fatal("authoritative zero snapshot ignored", s.Blessing)
	}
	apply(blessingPacket(proto.CountryKillCount, 0, 0, 0))
	if s.Blessing.Count != 0 || s.Blessing.RemainingMS != 0 {
		t.Fatal("confirmed reset ignored")
	}
	apply(blessingPacket(proto.BlessInit, 0, 0, 0))
	s.Phase = PhaseInWorld
	apply(blessingPacket(proto.BlessInit, 0, 250, 0))
	apply(blessingPacket(proto.BlessInit, 0, 0, 0))
	if s.Blessing.Count != 0 {
		t.Fatal("later authoritative snapshot ignored")
	}
	before := s.Blessing
	if _, err := s.Apply([]byte{0x11, 0x02, 0}); err == nil {
		t.Fatal("truncated packet accepted")
	}
	if s.Blessing != before {
		t.Fatal("truncated packet changed blessing")
	}
}

func TestBlessingCountdownUsesElapsedTimeAndKeepsAuthority(t *testing.T) {
	b := BlessingState{Known: true, Count: 12288, RemainingMS: 600000}
	for range 600 {
		b.Advance(1.0 / 600)
	}
	if b.RemainingMS < 598999 || b.RemainingMS > 599001 {
		t.Fatal("fractional frame time drift", b)
	}
	b.Advance(1000)
	if b.RemainingMS != 0 || b.Count != 12288 {
		t.Fatal("client invented count reset", b)
	}
	b.apply(proto.BlessInit, proto.BlessingUpdate{Country: 0, Count: 12288, RemainingMS: 12345})
	if b.RemainingMS != 12345 {
		t.Fatal("reentry remaining ignored")
	}
	b.apply(proto.CountryKillCount, proto.BlessingUpdate{Country: 0, Count: 0})
	if b.RemainingMS != 0 {
		t.Fatal("authoritative reset kept timer")
	}
}
