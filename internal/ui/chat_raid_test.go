package ui

import "testing"

func TestRaidChannelListsOnlyForRaidLeadership(t *testing.T) {
	c := &ChatPanel{}
	if got := c.channelList(); len(got) != 6 {
		t.Fatalf("a plain player sees %v", got)
	}
	c.RaidAllowed = true
	got := c.channelList()
	if len(got) != 7 || got[6] != 6 || chatChannelNames[6] != "Raid" {
		t.Fatalf("leader list %v", got)
	}
	if len(chatChannelOrder) != 6 {
		t.Fatal("the shared order was modified")
	}
}
