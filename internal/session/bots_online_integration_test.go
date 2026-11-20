//go:build integration

package session_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
)

func TestWorldBotsWithConnectedClients(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_BOTS") != "1" {
		t.Skip("requires a disposable GM and observer in open terrain")
	}
	gm, gmWire, send, wait := testworld.Enter(t)
	if !gm.Self.IsGM {
		wait(func() bool { return gm.Self.IsGM })
	}
	viewer, observer, _, _ := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	for _, requested := range []int{600, 1000} {
		t.Run(fmt.Sprint(requested), func(t *testing.T) {
			count := func(st *game.State) int {
				n := 0
				for id, e := range st.Entities {
					if id >= 0x60000000 && e.Kind == game.KindUser && e.Shape != nil {
						n++
					}
				}
				return n
			}
			var burstStart time.Time
			pump := func(done func() bool) {
				t.Helper()
				timer := time.NewTimer(12 * time.Second)
				defer timer.Stop()
				for !done() {
					var st *game.State
					var packet []byte
					var ok bool
					select {
					case packet, ok = <-gmWire.Packets:
						st = gm
					case packet, ok = <-observer.Packets:
						st = viewer
					case <-timer.C:
						t.Fatalf("waiting for %d bots: GM=%d observer=%d", requested, count(gm), count(viewer))
					}
					if !ok {
						t.Fatalf("connection closed with %d requested bots", requested)
					}
					if _, err := st.Apply(packet); err != nil {
						t.Fatalf("packet %#04x: %v", proto.Op(packet), err)
					}
				}
			}
			burstStart = time.Now()
			send(proto.ChatPacket(proto.ChannelNormal, false, fmt.Sprintf("/bots %d", requested)))
			pump(func() bool { return count(gm) == requested && count(viewer) == requested })
			t.Logf("both clients received %d complete appearances in %s", requested, time.Since(burstStart))
			send(proto.ChatPacket(proto.ChannelNormal, false, "/bots off"))
			pump(func() bool { return count(gm) == 0 && count(viewer) == 0 })
		})
	}
}
