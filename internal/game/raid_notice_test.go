package game

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestRaidNoticeIsStoredNotChatted(t *testing.T) {
	s := New()
	w := proto.NewWriter(proto.ChatRaid)
	w.U32(7)
	w.Text("Gather at the gate")
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.RaidNotice != "Gather at the gate" || s.RaidNoticeSerial != 1 || len(s.Chat) != 0 {
		t.Fatalf("notice %q serial %d chat %d", s.RaidNotice, s.RaidNoticeSerial, len(s.Chat))
	}
}
