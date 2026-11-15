package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
	"time"
)

func TestOJLimitsAreFramedOnTheServerWire(t *testing.T) {
	_, _ = forgeRuleSession(t)
	s, packets := gmWireSession(t, db.Character{ID: 990046}, true)
	expected := equipmentOJLimits(db.MustDefs())
	s.sendOJLimits()
	var got []proto.OJLimit
	for len(got) < len(expected) {
		select {
		case p := <-packets:
			rows, err := proto.DecodeOJLimits(p)
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, rows...)
		case <-time.After(time.Second):
			t.Fatal("missing framed OJ limits")
		}
	}
	for i, v := range got {
		if v != expected[i] {
			t.Fatal(got, expected)
		}
	}
}
