package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

func chatTestFont() *Font {
	f := &Font{}
	for i := range f.glyphs {
		f.glyphs[i].adv = 6
	}
	return f
}

func TestOriginalChatModeButtonSelectsChannel(t *testing.T) {
	c, err := NewChatPanel(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	c.Typing = true
	y := chatInputY(768)
	click := func(x, y int) {
		c.Update(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 768)
		c.Update(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 768)
	}
	click(20, y+8)
	if !c.ModeOpen {
		t.Fatal("mode button did not open channels")
	}
	click(20, y-len(chatChannelOrder)*chatModePitch+2*chatModePitch+8)
	if c.Channel != 3 || c.ModeOpen {
		t.Fatal("party selection failed", c.Channel, c.ModeOpen)
	}
}

func TestChatWrapIncludesUnbrokenMessages(t *testing.T) {
	f := chatTestFont()
	text := strings.Repeat("x", 121)
	lines := wrapLines(f, text, 60)
	if strings.Join(lines, "") != text || len(lines) != 13 {
		t.Fatal("long message lost characters", lines)
	}
	for _, line := range lines {
		if f.Width(line) > 60 {
			t.Fatal("line exceeds chat width", line)
		}
	}
	if got := wrapLines(f, "one two\n\nthree", 42); strings.Join(got, "|") != "one two||three" {
		t.Fatal("word/newline wrapping changed", got)
	}
}

func TestChatPanesScrollIndependentlyAndKeepNewMessages(t *testing.T) {
	c, f := &ChatPanel{Rows: [2]int{5, 5}}, chatTestFont()
	entries := []ChatEntry{{Text: "System", System: true}}
	for i := range 20 {
		entries = append(entries, ChatEntry{Text: fmt.Sprint(i)})
	}
	c.SetEntries(f, entries)
	if len(c.panes[0].lines) != 1 || len(c.panes[1].lines) != 20 {
		t.Fatal("system and player histories were mixed")
	}
	in := platform.Input{MouseX: 40, MouseY: ChatOrigin(734) + 130, Wheel: 120}
	if !c.Update(&in, 734) || c.panes[1].offset != 3 || c.panes[0].offset != 0 {
		t.Fatal("wheel did not scroll only the pointed pane")
	}
	entries = append(entries, ChatEntry{Text: "new"})
	c.SetEntries(f, entries)
	if c.panes[1].offset != 4 {
		t.Fatal("new message moved an older view")
	}
	c.SetEntries(f, entries)
	if c.panes[1].offset != 4 {
		t.Fatal("unchanged history moved the view")
	}
	y, height, capacity := c.paneGeometry(734, 1)
	limit := len(c.panes[1].lines) - capacity
	top, travel, thumb := c.scrollGeometry(1, y, height, limit, c.panes[1].offset)
	in = platform.Input{MouseX: 8, MouseY: thumb + 5, Pressed: true, MouseDown: true}
	c.Update(&in, 734)
	if !c.panes[1].dragging || c.panes[1].offset != 4 {
		t.Fatal("grabbing thumb jumped history")
	}
	in.Pressed = false
	in.MouseY = top + 5
	c.Update(&in, 734)
	if c.panes[1].offset != limit {
		t.Fatal("thumb did not reach oldest messages")
	}
	in.MouseY = top + travel + 5
	c.Update(&in, 734)
	if c.panes[1].offset != 0 {
		t.Fatal("thumb did not return to newest messages")
	}
	in.MouseDown, in.Pressed, in.Clicked = false, false, true
	in.MouseX = 500
	if !c.Update(&in, 734) {
		t.Fatal("scroll release escaped to world")
	}
	// Another panel may consume a release. A subsequent world press must not
	// continue that old scrollbar drag.
	in = platform.Input{MouseX: 8, MouseY: ChatOrigin(734) + 150, Pressed: true, MouseDown: true}
	c.Update(&in, 734)
	in.MouseX = 500
	if c.Update(&in, 734) || c.panes[1].dragging {
		t.Fatal("new press continued an interrupted scrollbar drag")
	}
}

func TestPhoenixChatRetainsSystemMarkupAndLiteralPlayerText(t *testing.T) {
	c, f := &ChatPanel{Rows: [2]int{5, 5}}, chatTestFont()
	entries := []ChatEntry{{Text: "damage", System: true, Style: 0}, {Text: "acquired {c3}item{/c}", System: true, Style: 3, Markup: true}, {Text: "literal {c3}message", Style: 9}}
	c.SetEntries(f, entries)
	if c.panes[0].lines[1] != "acquired item" || len(c.panes[0].runs[1]) != 2 || c.panes[1].lines[0] != "literal {c3}message" {
		t.Fatal("system markup or literal chat changed", c.panes)
	}
}

func TestPhoenixChatGeometryAndZeroRows(t *testing.T) {
	c := &ChatPanel{Rows: [2]int{5, 5}}
	for i, want := range [][3]int{{426, 112, 7}, {540, 144, 9}} {
		y, height, capacity := c.paneGeometry(768, i)
		if [3]int{y, height, capacity} != want {
			t.Fatal(i, y, height, capacity)
		}
	}
	c.Rows = [2]int{0, 0}
	for i, want := range [][3]int{{586, 32, 2}, {620, 64, 4}} {
		y, height, capacity := c.paneGeometry(768, i)
		if [3]int{y, height, capacity} != want {
			t.Fatal("collapsed history", i, y, height, capacity)
		}
	}
}

func TestNativeChatScrollAnchorSurvivesHistoryLimit(t *testing.T) {
	c, f := &ChatPanel{Rows: [2]int{5, 5}}, chatTestFont()
	var entries []ChatEntry
	for i := range 100 {
		entries = append(entries, ChatEntry{Text: fmt.Sprint(i), Style: 9})
	}
	c.SetEntries(f, entries)
	c.panes[1].offset = 10
	entries = append(entries[1:], ChatEntry{Text: "100", Style: 9})
	c.SetEntries(f, entries)
	if c.panes[1].offset != 11 {
		t.Fatal("new message moved historical view", c.panes[1].offset)
	}
}

func TestChatCombatNumberColorResetsBeforeTheRestOfTheMessage(t *testing.T) {
	c, f := &ChatPanel{Rows: [2]int{5, 5}}, chatTestFont()
	base := RGBColor(ChatRGB(0))
	c.SetEntries(f, []ChatEntry{{Text: "Hit {c3}73{/c} damage.", System: true, Color: base, Markup: true}})
	runs := c.panes[0].runs[0]
	if c.panes[0].lines[0] != "Hit 73 damage." || len(runs) != 3 || runs[0].color != base || runs[1].text != "73" || runs[1].color != RGBColor(0xffff00) || runs[2].color != base {
		t.Fatal("numeric emphasis leaked into surrounding message", runs)
	}
}

func TestChatNativeFiltersAndResponsiveLayout(t *testing.T) {
	c, err := NewChatPanel(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	c.Rows, c.ViewportWidth = [2]int{20, 20}, 500
	const h = 420
	y, paneHeight, _ := c.paneGeometry(h, 0)
	lower, lowerHeight, _ := c.paneGeometry(h, 1)
	if y < 0 || lower-y-paneHeight != 2 || chatInputY(h)-lower-lowerHeight != 2 {
		t.Fatal("resized histories lost their gaps")
	}
	click := func(x, y int) {
		c.Update(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, h)
		c.Update(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, h)
	}
	click(8, c.optionY(h, 0)+8)
	x, y, width, _ := c.optionBounds(h)
	if c.option != chatFilters || x+width > c.ViewportWidth || y != h-296 {
		t.Fatal("filter panel did not fit the viewport")
	}
	click(x+30, y+56)
	f := chatTestFont()
	entries := []ChatEntry{
		{Text: "damage", System: true, Style: 0},
		{Text: "loot", System: true, Style: 3},
		{Text: "whisper", Style: 11},
	}
	c.SetEntries(f, entries)
	if c.Filters != 1 || len(c.panes[0].lines) != 1 || len(c.panes[1].lines) != 1 {
		t.Fatal("filter hid the wrong categories")
	}
	click(x+30, y+56)
	c.SetEntries(f, entries)
	if len(c.panes[0].lines) != 2 {
		t.Fatal("re-enabling damage lost stored entries")
	}
	click(8, c.optionY(h, 1)+8)
	x, y, _, _ = c.optionBounds(h)
	click(x+130, y+229)
	if !c.Transparent || c.backgroundHovered(h) || c.paneAlpha(0, chatNow()) != 1 {
		t.Fatal("Alpha should affect the background only")
	}
}

func TestChatComposerAndIndependentSidebarButtons(t *testing.T) {
	c, err := NewChatPanel(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	const h = 768
	y := chatInputY(h) + 4
	if c.Contains(100, y, h) {
		t.Fatal("idle composer captured world input")
	}
	c.Typing = true
	if !c.Contains(chatWidth-1, y, h) || c.Contains(chatWidth, y, h) {
		t.Fatal("composer and history do not share the same right edge")
	}
	c.Typing = false
	for _, rows := range []int{0, 5, 20} {
		c.Rows[1] = rows
		lower, height, _ := c.paneGeometry(h, 1)
		if c.optionY(h, 0) < lower || c.optionY(h, 1)+chatOptionSize != lower+height {
			t.Fatal("sidebar buttons escaped the lower history", rows)
		}
	}
	press := func(i int) {
		c.Update(&platform.Input{MouseX: 8, MouseY: c.optionY(h, i) + 8, Pressed: true, MouseDown: true}, h)
	}
	release := func(i int) {
		c.Update(&platform.Input{MouseX: 8, MouseY: c.optionY(h, i) + 8, Clicked: true}, h)
	}
	press(0)
	release(1)
	if c.OptionsOpen() {
		t.Fatal("dragging across buttons opened a panel")
	}
	press(0)
	release(0)
	if c.option != chatFilters || c.inputShown() {
		t.Fatal("filter button did not open filters alone")
	}
	press(1)
	release(1)
	if c.option != chatHelp || c.inputShown() {
		t.Fatal("help button did not replace filters")
	}
}

func TestChatSidebarUsesUnscaledNativeTiles(t *testing.T) {
	c := &ChatPanel{Rows: [2]int{5, 5}}
	c.controls(768)
	for _, p := range c.panes {
		for i, b := range p.arrows {
			if b.DrawW != 16 || b.DrawH != 32 || b.HitW != 13 || b.HitH != 19 || b.X() != 4 {
				t.Fatalf("arrow %d resized native tile or hitbox: %+v", i, b)
			}
			if uv := b.UV[StateNormal]; uv != (UV{0, float32(i) / 2, .125, float32(i+1) / 2}) {
				t.Fatalf("arrow %d uses wrong atlas cell: %+v", i, uv)
			}
		}
	}
	y, height, _ := c.paneGeometry(768, 1)
	top, travel, thumb := c.scrollGeometry(1, y, height, 40, 0)
	if thumb != top+travel || thumb+chatThumbSize != c.panes[1].arrows[1].Y() {
		t.Fatal("thumb track did not end at the lower arrow")
	}
	c.Rows = [2]int{}
	c.controls(768)
	c.Update(&platform.Input{MouseX: 8, MouseY: 600, MouseDown: true}, 768)
}
