package proto

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func partyRecordFixture(w *Writer) {
	w.U32(42)
	w.Fixed("Member", 21, true)
	w.U16(7)
	w.U8(2)
	for _, n := range []uint32{100, 200, 30, 50, 70, 90} {
		w.U32(n)
	}
	w.U16(3)
	w.F32(123.5)
	w.F32(7)
	w.F32(-5)
	w.U8(1)
	w.U16(31)
	w.U8(1)
	w.U32(12500)
}
func TestPartyNativeRecordAndTruncation(t *testing.T) {
	w := NewWriter(PartyRoster)
	w.U8(0)
	w.U8(1)
	partyRecordFixture(w)
	p := w.Bytes()
	list, err := DecodePartyList(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Members) != 1 || list.LeaderIndex != 0 {
		t.Fatal(list)
	}
	m := list.Members[0]
	if m.ID != 42 || m.Name != "Member" || m.Level != 7 || m.Class != 2 || m.MaxHP != 200 || m.MP != 30 || m.MaxMP != 50 || m.SP != 70 || m.MaxSP != 90 || m.Instance != 3 || m.X != 123.5 || m.Z != -5 || len(m.Effects) != 1 || m.Effects[0].Remaining != 12500 {
		t.Fatal(m)
	}
	for n := 0; n < len(p); n++ {
		if _, err := DecodePartyList(p[:n]); err == nil {
			t.Fatal("accepted truncated party", n)
		}
	}
	joined := NewWriter(PartyJoined)
	partyRecordFixture(joined)
	for n := 0; n < len(joined.Bytes()); n++ {
		if _, err := DecodePartyJoined(joined.Bytes()[:n]); err == nil {
			t.Fatal("accepted truncated join", n)
		}
	}
}
func TestPartyWritersMatchNativeBytes(t *testing.T) {
	for _, test := range []struct {
		got  []byte
		want string
	}{
		{PartyInvitePacket(0x01020304), "020b04030201"},
		{PartyAnswerPacket(true, 0x01020304), "030b0004030201"},
		{PartyAnswerPacket(false, 0x01020304), "030b0104030201"},
		{PartyLeavePacket(), "050b"},
		{PartyExpelPacket(0x01020304), "060b04030201"},
		{PartyLeaderPacket(0x01020304), "070b04030201"},
	} {
		want, _ := hex.DecodeString(test.want)
		if !bytes.Equal(test.got, want) {
			t.Fatalf("%x != %x", test.got, want)
		}
	}
	p := PartyInviteNamePacket("Member")
	if len(p) != 23 || !bytes.Equal(p[:9], []byte{9, 11, 'M', 'e', 'm', 'b', 'e', 'r', 0}) {
		t.Fatal(p)
	}
	p = PartyGlobalInvitePacket("Member")
	if len(p) != 23 || !bytes.Equal(p[:9], []byte{0x15, 0x7f, 'M', 'e', 'm', 'b', 'e', 'r', 0}) {
		t.Fatal(p)
	}
}
func FuzzPartyList(f *testing.F) {
	w := NewWriter(PartyRoster)
	w.U8(0)
	w.U8(1)
	partyRecordFixture(w)
	f.Add(w.Bytes())
	f.Fuzz(func(t *testing.T, p []byte) { _, _ = DecodePartyList(p); _, _ = DecodePartyJoined(p) })
}

func TestPartyMemberEffects(t *testing.T) {
	var m PartyMember
	m.AddEffect(10, 1)
	m.AddEffect(10, 1) // same level: untouched
	m.AddEffect(10, 2) // other level replaces in place
	m.AddEffect(10, 100)
	m.AddEffect(20, 3)
	if len(m.Effects) != 3 || m.Effects[0].Level != 2 || m.Effects[1].Level != 100 || m.Effects[2].Skill != 20 {
		t.Fatalf("effects=%+v", m.Effects)
	}
	m.RemoveEffect(10, 1) // no exact match
	m.RemoveEffect(10, 2)
	if len(m.Effects) != 2 || m.Effects[0].Level != 100 {
		t.Fatalf("after remove %+v", m.Effects)
	}
}
