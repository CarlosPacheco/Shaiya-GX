package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"testing"
)

func TestScreenNewsPreservesChatCapsQueueAndExpires(t *testing.T) {
	a := inputApp()
	a.msg = text.SysMsg{463: "Destroyed <t>"}
	for i := range 7 {
		a.clock = float64(i)
		a.systemMessage(8, 463, map[string]string{"t": fmt.Sprint(i)})
	}
	if len(a.state.Chat) != 7 || len(a.screenNotices) != 5 || a.screenNotices[0].text != "Destroyed 2" {
		t.Fatal(a.state.Chat, a.screenNotices)
	}
	a.clock = 12
	// With no font this still ages the queue without a GL context.
	a.drawScreenNotices(1024, 768)
	if len(a.screenNotices) != 5 {
		t.Fatal("native expiry must be strictly after 10s")
	}
	a.clock = 12.01
	a.drawScreenNotices(1024, 768)
	if len(a.screenNotices) != 4 {
		t.Fatal("oldest notice did not expire")
	}
	a.clock = 17
	a.drawScreenNotices(1024, 768)
	if len(a.screenNotices) != 0 || len(a.state.Chat) != 7 {
		t.Fatal("notice expiry modified chat history")
	}
	a.systemMessage(7, 463, map[string]string{"t": "local"})
	if len(a.screenNotices) != 0 || len(a.state.Chat) != 8 {
		t.Fatal("local altar notice became news")
	}
}
