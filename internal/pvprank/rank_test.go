package pvprank

import (
	"fmt"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
)

func TestEveryNativeRankBoundaryAndTitle(t *testing.T) {
	msg := text.SysMsg{5450: "first", 5451: "second", 5404: "final"}
	for i := 0; i < 15; i++ {
		msg[5389+i] = fmt.Sprintf("name%d", i)
	}
	if _, ok := Index(0); ok || Title(0, msg) != "" {
		t.Fatal("zero kills must not display an icon or title")
	}
	for i, minimum := range thresholds {
		for _, kills := range []uint32{minimum, minimum + 1} {
			if got, ok := Index(kills); !ok || got != i {
				t.Fatalf("kills %d: rank %d/%v, want %d", kills, got, ok, i)
			}
		}
		if i > 0 {
			if got, ok := Index(minimum - 1); !ok || got != i-1 {
				t.Fatalf("below threshold %d: rank %d/%v", minimum, got, ok)
			}
		}
		want := fmt.Sprintf("first name%d", i)
		if i >= 15 {
			want = fmt.Sprintf("second name%d", i-15)
		}
		if i == 30 {
			want = "final"
		}
		if got := Title(minimum, msg); got != want {
			t.Fatalf("rank %d title %q, want localized %q", i, got, want)
		}
	}
	if i, ok := Index(^uint32(0)); !ok || i != 30 {
		t.Fatal("maximum counter lost the final display rank")
	}
	if Title(50, nil) != "" {
		t.Fatal("missing SysMsg invented a title")
	}
}
