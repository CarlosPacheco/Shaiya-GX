package app

import (
	"bytes"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"testing"
)

func TestChatEntriesIdentifyChannelsAndWhisperDirection(t *testing.T) {
	for _, tc := range []struct {
		line   game.ChatLine
		want   string
		system bool
	}{
		{game.ChatLine{Op: proto.ChatParty, Sender: "Alice", Text: "Hi"}, "[Party]Alice : Hi", false},
		{game.ChatLine{Op: proto.ChatGuild, Sender: "Alice", Text: "Hi"}, "[Guild]Alice : Hi", false},
		{game.ChatLine{Op: proto.ChatWhisper, Sender: "Alice", Text: "Hi"}, "Whisper from Alice: Hi", false},
		{game.ChatLine{Op: proto.ChatWhisper, Sender: "Alice", Direction: 1, Text: "Hi"}, "Whisper to Alice: Hi", false},
		{game.ChatLine{Op: proto.ChatError, Text: "Missing"}, "Missing", true},
	} {
		got := chatEntry(chatTestMessages, tc.line)
		if got.Text != tc.want || got.System != tc.system {
			t.Fatalf("got %+v, want %q", got, tc.want)
		}
	}
}

func TestFormattedChatTracksChangesAndReusesUnchangedHistory(t *testing.T) {
	a := inputApp()
	a.msg = chatTestMessages
	a.state.Chat = []game.ChatLine{{Op: proto.ChatParty, Sender: "Alice", Text: "Hi"}}
	first := a.formattedChat()
	if len(first) != 1 || first[0].Text != "[Party]Alice : Hi" {
		t.Fatal(first)
	}
	if got := testing.AllocsPerRun(100, func() { a.formattedChat() }); got != 0 {
		t.Fatalf("unchanged history allocated: %v", got)
	}
	a.state.Chat[0].Text = "Changed"
	if got := a.formattedChat(); len(got) != 1 || got[0].Text != "[Party]Alice : Changed" {
		t.Fatal(got)
	}
	a.state.Chat = nil
	if len(a.formattedChat()) != 0 {
		t.Fatal("cleared history remained visible")
	}
}

func BenchmarkFormattedChat(b *testing.B) {
	a := inputApp()
	a.msg = chatTestMessages
	for range 200 {
		a.state.Chat = append(a.state.Chat, game.ChatLine{Op: proto.ChatParty, Sender: "Alice", Text: "An ordinary chat message"})
	}
	a.formattedChat()
	b.Run("reformat", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			entries := make([]ui.ChatEntry, 0, len(a.state.Chat))
			for _, line := range a.state.Chat {
				entries = append(entries, chatEntry(chatTestMessages, line))
			}
			if len(entries) != 200 {
				b.Fatal("bad fixture")
			}
		}
	})
	b.Run("unchanged", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			a.formattedChat()
		}
	})
}

func TestWhisperCommandUsesTargetAndText(t *testing.T) {
	packet, handled := whisperCommand("/W Alice hello there")
	w := proto.NewWriter(proto.ChatWhisper)
	w.Fixed("Alice", 21, true)
	w.Text("hello there")
	if !handled || !bytes.Equal(packet, w.Bytes()) {
		t.Fatalf("bad whisper: %x", packet)
	}
	for _, text := range []string{"/w", "/whisper Alice", "/w 123456789012345678901 hello"} {
		if p, ok := whisperCommand(text); !ok || p != nil {
			t.Fatalf("invalid command %q: %x", text, p)
		}
	}
	if _, ok := whisperCommand("ordinary text"); ok {
		t.Fatal("ordinary chat consumed")
	}
}

func TestChatCapturesTypingAndCancel(t *testing.T) {
	a := &App{}
	if a.updateChat(&platform.Input{}) {
		t.Fatal("idle chat captured movement")
	}
	if !a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}) || !a.chatOpen {
		t.Fatal("chat did not open")
	}
	if !a.updateChat(&platform.Input{Chars: []rune("wasd")}) || a.chatInput.Text != "wasd" {
		t.Fatal("chat lost typed movement keys")
	}
	if !a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}) || a.chatOpen {
		t.Fatal("escape did not cancel")
	}
	if a.updateChat(&platform.Input{}) {
		t.Fatal("cancel kept input captured")
	}
}

func TestChatHistoryRestoresUnsentDraft(t *testing.T) {
	a := &App{chatHistory: []string{"first", "second"}}
	a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyEnter}})
	a.updateChat(&platform.Input{Chars: []rune("draft")})
	a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyUp}})
	if a.chatInput.Text != "second" {
		t.Fatal(a.chatInput.Text)
	}
	a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyUp}})
	if a.chatInput.Text != "first" {
		t.Fatal(a.chatInput.Text)
	}
	a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyDown, platform.KeyDown}})
	a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyDown}})
	if a.chatInput.Text != "draft" {
		t.Fatal("unsent draft lost", a.chatInput.Text)
	}
}

func TestChatHistoryOwnsPointerWithoutBlockingHeldMovement(t *testing.T) {
	a := inputApp()
	a.msg = chatTestMessages
	in := platform.Input{MouseX: 40, MouseY: ui.ChatOrigin(734) + 130, Wheel: 120, Down: "W"}
	controls := a.worldControls(&in, 1024, 734)
	if controls.Wheel != 0 || controls.Down != "W" {
		t.Fatal("chat history zoomed camera or stopped movement", controls)
	}
	in.Wheel, in.Pressed, in.MouseDown = 0, true, true
	a.worldControls(&in, 1024, 734)
	controls = a.worldControls(&platform.Input{MouseX: 512, MouseY: 384, Clicked: true}, 1024, 734)
	if controls.Clicked {
		t.Fatal("chat press released over world selected a target")
	}
}

var chatTestMessages = text.SysMsg{353: "[Party]<p> : <b>", 354: "[Whisper]<p> : <b>", 370: "[Guild]<p> : <b>"}

func TestPhoenixChatPickerOwnsClicksAboveActionBars(t *testing.T) {
	a := inputApp()
	a.chatOpen = true
	a.chatUI.ModeOpen = true
	in := platform.Input{MouseX: 40, MouseY: 734 - 82 - 6*15 + 15 + 8, Pressed: true, MouseDown: true, Down: "W"}
	controls := a.worldControls(&in, 1024, 734)
	if controls.Pressed || controls.Down != "W" {
		t.Fatal("channel picker lost click or blocked held movement", controls)
	}
	controls = a.worldControls(&platform.Input{MouseX: 40, MouseY: in.MouseY, Clicked: true}, 1024, 734)
	if controls.Clicked || a.chatUI.Channel != 5 {
		t.Fatal("picker release reached world or failed to select whisper", controls, a.chatUI.Channel)
	}
}

func TestPhoenixChatClickFocusAndChannelAliases(t *testing.T) {
	a := inputApp()
	// With the composer closed, the idle input row belongs to the world.
	if a.updateChatPointer(&platform.Input{MouseX: 100, MouseY: 734 - 81, Pressed: true, MouseDown: true}, 1024, 734) || a.chatOpen {
		t.Fatal("hidden message input captured a click")
	}
	for _, tc := range []struct {
		value   string
		channel uint8
		body    string
	}{{"/p hello", 3, "hello"}, {"/g hello", 2, "hello"}, {"/1 hello", 4, "hello"}, {"/trade hello", 1, "hello"}, {"/say hello", 0, "hello"}} {
		channel, body, ok := phoenixChannelCommand(tc.value)
		if !ok || channel != tc.channel || body != tc.body {
			t.Fatal(tc, channel, body, ok)
		}
	}
	// /party <name> is an invitation; other forms stay party chat.
	if _, _, ok := phoenixChannelCommand("/party Alice"); ok {
		t.Fatal("/party <name> must not be party chat")
	}
	for _, value := range []string{"/party", "/party hello team", "/PARTY  see you soon"} {
		if channel, _, ok := phoenixChannelCommand(value); !ok || channel != 3 {
			t.Fatal(value, channel, ok)
		}
	}
}

func TestPhoenixChannelColoursKeepSysMsgText(t *testing.T) {
	for _, tc := range []struct {
		op  proto.Opcode
		rgb uint32
	}{{proto.ChatNormal, 0xd5d4d3}, {proto.ChatParty, 0xa8f3a0}, {proto.ChatGuild, 0xeea4ed}, {proto.ChatWhisper, 0xf3a0a0}, {proto.ChatMap, 0xff3000}, {proto.ChatWorld, 0xf3dea0}} {
		line := game.ChatLine{Op: tc.op, Sender: "Alice", Text: "Message"}
		entry := chatEntry(chatTestMessages, line)
		if entry.Color != ui.RGBColor(tc.rgb) {
			t.Fatal(tc, entry)
		}
		if tc.op == proto.ChatParty && entry.Text != chatTestMessages.Format(353, map[string]string{"p": "Alice", "b": "Message"}) {
			t.Fatal("SysMsg template changed", entry)
		}
	}
}

func TestWhisperSelectionSurvivesOptionsPreview(t *testing.T) {
	a := inputApp()
	a.chatUI.Channel = 5
	a.applyClientSettings()
	if a.chatUI.Channel != 5 {
		t.Fatal("applying audio/options preferences reset the active whisper channel")
	}
}

func TestNativeChatShortcutsRecipientCyclingAndEditing(t *testing.T) {
	a := inputApp()
	for _, line := range []game.ChatLine{
		{Op: proto.ChatWhisper, Sender: "Alice", Text: "a"},
		{Op: proto.ChatWhisper, Sender: "Bob", Text: "b"},
		{Op: proto.ChatWhisper, Sender: "alice", Text: "c"},
	} {
		a.state.AddChat(line)
	}
	a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyChatWhisper}})
	if !a.chatOpen || a.chatUI.Channel != 5 || !a.chatUI.RecipientFocus {
		t.Fatal("whisper shortcut did not focus its own field")
	}
	a.updateChat(&platform.Input{Chars: []rune("Claire")})
	if a.chatUI.Recipient != "Claire" || a.chatInput.Text != "" {
		t.Fatal("recipient typing leaked into body")
	}
	for _, want := range []string{"alice", "Bob", "alice"} {
		a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyTab}})
		if a.chatUI.Recipient != want || a.chatUI.RecipientFocus {
			t.Fatalf("recent recipient = %q, want %q", a.chatUI.Recipient, want)
		}
	}
	a.updateChat(&platform.Input{Chars: []rune("Hi 😀!")})
	a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyLeft}})
	a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyLeft}, Shift: true})
	var copied string
	a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyCopy}, WriteClipboard: func(s string) { copied = s }})
	a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyPaste}, ReadClipboard: func() string { return "Alice\n" }})
	if copied != "😀" || a.chatInput.Text != "Hi Alice!" {
		t.Fatalf("copy/paste at caret produced %q and %q", copied, a.chatInput.Text)
	}
	a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyChatGuild}})
	if a.chatUI.Channel != proto.ChannelGuild || a.chatInput.Text != "Hi Alice!" {
		t.Fatal("guild shortcut lost the composer draft")
	}
}

func TestBannedWordsRefreshReceivedChatWithoutChangingOriginal(t *testing.T) {
	a := inputApp()
	a.chatFilter = text.ParseWordFilter([]byte("[Filter]\nWord=bad\nWord=bad phrase\nWord=Árbol\n"))
	a.state.Chat = []game.ChatLine{
		{Op: proto.ChatWhisper, Sender: "bad", Text: "BAD PHRASE + árbol"},
		{Op: proto.ChatWhisper, Sender: "bad", Direction: 1, Text: "bad"},
		{Text: "bad"},
	}
	if got := a.formattedChat()[0].Text; got != "Whisper from bad: BAD PHRASE + árbol" {
		t.Fatal(got)
	}
	a.gameMenu.Settings.BannedWords = true
	got := a.formattedChat()
	if got[0].Text != "Whisper from bad: ********** + *****" || got[1].Text != "Whisper to bad: bad" || got[2].Text != "bad" {
		t.Fatalf("unexpected filtered history: %+v", got)
	}
	if a.state.Chat[0].Text != "BAD PHRASE + árbol" {
		t.Fatal("filter changed the original chat line")
	}
	a.gameMenu.Settings.BannedWords = false
	if got := a.formattedChat()[0].Text; got != "Whisper from bad: BAD PHRASE + árbol" {
		t.Fatal(got)
	}
}

func TestRaidNoticesAppearOnThePacketThatDeliversThem(t *testing.T) {
	a := inputApp()
	a.msg = text.SysMsg{6606: "[Union] <b>"}
	a.chatFilter = text.ParseWordFilter([]byte("[Filter]\nWord=gate\n"))
	a.gameMenu.Settings.BannedWords = true
	for i, body := range []string{"Gather at the gate", "Move now"} {
		w := proto.NewWriter(proto.ChatRaid)
		w.U32(7)
		w.Text(body)
		a.applyWorldPacket(w.Bytes())
		if len(a.screenNotices) != i+1 || len(a.state.Chat) != i+1 {
			t.Fatal("raid notice waited for another packet or duplicated", a.screenNotices, a.state.Chat)
		}
	}
	if a.screenNotices[0].text != "[Union] Gather at the ****" || a.screenNotices[1].text != "[Union] Move now" {
		t.Fatal(a.screenNotices)
	}
}
