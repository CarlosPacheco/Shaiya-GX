package ui

import (
	"unicode"
	"unicode/utf8"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/textwire"
)

type ChatCaret struct {
	Position, Anchor int
}

func (c *ChatCaret) End(s string) {
	c.Position = utf8.RuneCountInString(s)
	c.Anchor = c.Position
}

func (c ChatCaret) span(n int) (int, int) {
	a, b := min(n, max(0, c.Position)), min(n, max(0, c.Anchor))
	return min(a, b), max(a, b)
}

// EditAt edits a chat field at its caret and preserves the UTF-8 wire limit.
func (t *TextField) EditAt(in *platform.Input, caret *ChatCaret) {
	text := []rune(t.Text)
	caret.Position = min(len(text), max(0, caret.Position))
	var pasted string
	for _, key := range in.Keys {
		left, right := caret.span(len(text))
		switch key {
		case platform.KeySelectAll:
			caret.Anchor, caret.Position = 0, len(text)
		case platform.KeyHome, platform.KeyEnd, platform.KeyLeft, platform.KeyRight:
			at := caret.Position
			switch key {
			case platform.KeyHome:
				at = 0
			case platform.KeyEnd:
				at = len(text)
			case platform.KeyLeft:
				if !in.Shift && left != right {
					at = left
				} else {
					at = utf8.RuneCountInString(textwire.DropLast(string(text[:at])))
				}
			case platform.KeyRight:
				if !in.Shift && left != right {
					at = right
				} else {
					at = len(text) - utf8.RuneCountInString(textwire.DropFirst(string(text[at:])))
				}
			}
			caret.Position = at
			if !in.Shift {
				caret.Anchor = at
			}
		case platform.KeyCopy, platform.KeyCut:
			if left != right && in.WriteClipboard != nil {
				in.WriteClipboard(string(text[left:right]))
			}
			if key == platform.KeyCut && left != right {
				text = append(text[:left], text[right:]...)
				caret.Position, caret.Anchor = left, left
			}
		case platform.KeyDelete, platform.KeyBackspace:
			if left == right {
				if key == platform.KeyBackspace {
					left = utf8.RuneCountInString(textwire.DropLast(string(text[:left])))
				} else {
					right = len(text) - utf8.RuneCountInString(textwire.DropFirst(string(text[right:])))
				}
			}
			text = append(text[:left], text[right:]...)
			caret.Position, caret.Anchor = left, left
		case platform.KeyPaste:
			if in.ReadClipboard != nil {
				pasted = in.ReadClipboard()
			}
		}
	}
	if len(in.Chars) == 0 && pasted == "" {
		t.Text = string(text)
		return
	}
	var inserted []rune
	for _, r := range pasted + string(in.Chars) {
		if !unicode.IsControl(r) && utf8.ValidRune(r) {
			inserted = append(inserted, r)
		}
	}
	if len(inserted) == 0 {
		t.Text = string(text)
		return
	}
	left, right := caret.span(len(text))
	prefix, suffix := string(text[:left]), string(text[right:])
	insert := textwire.Prefix(string(inserted), max(0, t.Max-len(prefix)-len(suffix)))
	t.Text = prefix + insert + suffix
	caret.End(prefix + insert)
}

func drawChatSelection(s *render.Sprites, f *Font, blank *render.Texture, runes []rune, first int, caret ChatCaret, x, y, width int) {
	left, right := caret.span(len(runes))
	if right <= first || left == right {
		return
	}
	start := min(width, f.Width(string(runes[first:max(first, left)])))
	end := min(width, f.Width(string(runes[first:right])))
	if end > start {
		s.Draw(blank, float32(x+start), float32(y), float32(end-start), float32(f.Height), 0, 0, 1, 1, [4]float32{.25, .41, .68, .8})
	}
}
