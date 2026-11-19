package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestNoticeSchedules(t *testing.T) {
	local := time.FixedZone("server", -3*60*60)
	start := time.Date(2026, 11, 16, 12, 0, 0, 0, local) // Monday
	notices := scheduleNotices([]db.ServerNotice{
		{Message: "interval", Frequency: 60, Fixed: [5]string{"1/27/10"}},
		{Message: "fixed", Frequency: 1, Fixed: [5]string{"1/12/10", "1/12/10", "1/13/10"}},
		{Message: "manual only"},
	}, start)
	if len(notices) != 2 {
		t.Fatal("manual notices must not be scheduled")
	}
	at := func(minutes int) time.Time { return start.Add(time.Duration(minutes) * time.Minute) }
	if notices[0].due(at(59)) || !notices[0].due(at(60)) || notices[0].due(at(60)) ||
		!notices[0].due(at(185)) || notices[0].due(at(186)) || !notices[0].due(at(240)) {
		t.Fatal("interval slipped from startup or repeated missed ticks")
	}
	if notices[1].due(at(1)) || notices[1].due(at(9)) || !notices[1].due(at(10)) ||
		notices[1].due(at(10).Add(time.Second)) || !notices[1].due(at(70)) ||
		notices[1].due(at(10).AddDate(0, 0, 1)) || !notices[1].due(at(10).AddDate(0, 0, 7)) {
		t.Fatal("fixed times must override frequency and run once per configured local minute")
	}
}

func TestNoticeCommandReachesBothFactions(t *testing.T) {
	old := noticeCatalog
	t.Cleanup(func() { noticeCatalog = old })
	var err error
	noticeCatalog, err = db.ServerNotices()
	if err != nil || len(noticeCatalog) != 1 || noticeCatalog[0].Frequency != 60 {
		t.Fatalf("missing hourly notice seed: %+v (%v)", noticeCatalog, err)
	}
	gm, gmWire := gmWireSession(t, db.Character{ID: 94001, MapID: 1}, true)
	player, playerWire := gmWireSession(t, db.Character{ID: 94002, MapID: 2}, false)
	for i, session := range []*Session{gm, player} {
		p := RegisterPlayer(*session.char, nil, session.isAdmin, uint8(i), session.out, session.logger)
		t.Cleanup(p.Unregister)
	}
	read := func(wire <-chan []byte) []byte {
		t.Helper()
		select {
		case data := <-wire:
			return data
		case <-time.After(time.Second):
			t.Fatal("missing notice packet")
			return nil
		}
	}
	for _, c := range []struct{ command, message string }{
		{"/notice index1", "GX Automatic Notice - Test"},
		{"/notice Keep  the spacing", "Keep  the spacing"},
	} {
		if !gm.handleGMCommand(c.command) {
			t.Fatal("notice command went through to normal chat")
		}
		for _, wire := range []<-chan []byte{gmWire, playerWire} {
			data := read(wire)
			msg, err := proto.DecodeChatByName(data)
			if err != nil || proto.Op(data) != proto.ChatWorldAdmin || msg.Sender != "Notice" || msg.Text != c.message {
				t.Fatalf("incorrect notice: %+v (%v)", msg, err)
			}
		}
	}
	if !player.handleGMCommand("/notice blocked") || proto.Op(read(playerWire)) != proto.GXGMState {
		t.Fatal("non-GM notice was not refused")
	}
	if !gm.handleGMCommand("/notice index404") || proto.Op(read(gmWire)) != proto.GXGMState {
		t.Fatal("unknown notice index was not refused")
	}
}
