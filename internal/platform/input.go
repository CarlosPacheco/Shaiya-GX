package platform

import (
	"github.com/go-gl/glfw/v3.4/glfw"
	"strings"
)

// Key is a subset of keys the UI uses.
type Key int

const (
	KeyNone Key = iota
	KeyEnter
	KeyTab
	KeyBackspace
	KeyEscape
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyDelete
	KeyNumLock
	KeyHome
	KeyPageUp
	KeyPageDown
	KeySelfTarget // F2 selects the local player.
	KeyF9
	KeyF10
	KeyF11 // Native English client: DirectInput 0x57 toggles the whole UI.
	KeySpace
	KeyTarget         // Native English client: DirectInput 0x29 (grave accent).
	KeyPreviousTarget // Press edge only; Backspace still repeats inside text fields.
	KeyRevolver
	KeyRevolverReset
	KeyRevolverPage1
	KeyRevolverPage2
	KeyRevolverPage3
	KeyRevolverPage4
	KeyChatWhisper
	KeyChatParty
	KeyChatGuild
	KeyChatTrade
	KeyChatArea
	KeyEnd
	KeySelectAll
	KeyCopy
	KeyCut
	KeyPaste
)

// Input is one frame of input: the state plus the events since last frame.
type Input struct {
	KeypadChars      string // character events that belong to keypad action edges
	ActionHeld       uint32 // held number-row/keypad slots, independent of repeat events
	Actions          []int  // secondary/third-bar press edges, flattened slot indices
	MouseX, MouseY   int
	MouseLook        bool    // captured pointer; UI coordinates stay at the drag anchor
	MouseDX, MouseDY float64 // framebuffer displacement during captured look
	MouseDown        bool    // left button held
	Clicked          bool    // left button released this frame
	Pressed          bool    // left button pressed this frame
	Chars            []rune
	Keys             []Key
	ReadClipboard    func() string
	WriteClipboard   func(string)
	Shift            bool
	MiddlePressed    bool    // middle button pressed this frame
	Released         string  // released A..Z, for native double-tap controls
	RightDown        bool    // right button held
	RightPressed     bool    // right button pressed this frame
	Space            bool    // space held
	Wheel            float64 // wheel delta in Win32 units (120 per notch)
	Down             string  // snapshot: held A..Z and arrow markers < > ^ v
	SuppressAutoRun  bool    // modal/chat updates pause automatic travel
}

// Held reports whether a letter key or an arrow ('<' '>' '^' 'v') is down.
func (in *Input) Held(k rune) bool { return strings.ContainsRune(in.Down, k) }

func (in *Input) HasKey(k Key) bool {
	for _, x := range in.Keys {
		if x == k {
			return true
		}
	}
	return false
}

var keyMap = map[glfw.Key]Key{
	glfw.KeyGraveAccent: KeyTarget,
	glfw.KeyF2:          KeySelfTarget,
	glfw.KeyEnter:       KeyEnter, glfw.KeyKPEnter: KeyEnter, glfw.KeyTab: KeyTab,
	glfw.KeyBackspace: KeyBackspace, glfw.KeyEscape: KeyEscape, glfw.KeyUp: KeyUp,
	glfw.KeyHome: KeyHome, glfw.KeyEnd: KeyEnd, glfw.KeyPageUp: KeyPageUp, glfw.KeyPageDown: KeyPageDown,
	glfw.KeyF9: KeyF9, glfw.KeyF10: KeyF10, glfw.KeyF11: KeyF11, glfw.KeySpace: KeySpace,
	glfw.KeyDown: KeyDown, glfw.KeyLeft: KeyLeft, glfw.KeyRight: KeyRight, glfw.KeyDelete: KeyDelete,
}

func thirdBarModifier(mods glfw.ModifierKey) bool {
	return mods&(glfw.ModControl|glfw.ModAlt|glfw.ModShift|glfw.ModSuper) == glfw.ModControl
}

// InstallInput registers callbacks; call Frame once per frame.
func (w *Window) InstallInput() *InputQueue {
	q := &InputQueue{}
	w.SetCharCallback(func(_ *glfw.Window, r rune) {
		if q.suppressChordChar {
			return
		}
		if q.keypadChar && r >= '0' && r <= '9' {
			q.keypadChar = false
			q.cur.KeypadChars += string(r)
		}
		q.cur.Chars = append(q.cur.Chars, r)
	})
	w.SetKeyCallback(func(win *glfw.Window, k glfw.Key, _ int, a glfw.Action, mods glfw.ModifierKey) {
		q.cur.Shift = mods&glfw.ModShift != 0
		q.keypadChar = false
		q.suppressChordChar = false
		if a == glfw.Press {
			var key Key
			switch {
			case mods&(glfw.ModControl|glfw.ModAlt|glfw.ModSuper) == glfw.ModControl && k == glfw.KeyA:
				key = KeySelectAll
			case mods&(glfw.ModControl|glfw.ModAlt|glfw.ModSuper) == glfw.ModControl && k == glfw.KeyC:
				key = KeyCopy
			case mods&(glfw.ModControl|glfw.ModAlt|glfw.ModSuper) == glfw.ModControl && k == glfw.KeyX:
				key = KeyCut
			case mods&(glfw.ModControl|glfw.ModAlt|glfw.ModSuper) == glfw.ModControl && k == glfw.KeyV:
				key = KeyPaste
			case k == glfw.KeyZ && win.GetKey(glfw.KeyLeftShift) == glfw.Press:
				key = KeyChatWhisper
			case k == glfw.KeyEnter || k == glfw.KeyKPEnter:
				switch {
				case win.GetKey(glfw.KeyLeftControl) == glfw.Press:
					key = KeyChatParty
				case win.GetKey(glfw.KeyLeftAlt) == glfw.Press:
					key = KeyChatGuild
				case win.GetKey(glfw.KeyLeftShift) == glfw.Press:
					key = KeyChatTrade
				case win.GetKey(glfw.KeyRightShift) == glfw.Press:
					key = KeyChatArea
				}
			}
			if key != KeyNone {
				q.cur.Keys = append(q.cur.Keys, key)
				q.suppressChordChar = true
				return
			}
		}
		if mods&(glfw.ModAlt|glfw.ModControl|glfw.ModSuper) == glfw.ModAlt {
			var key Key
			switch {
			case k == glfw.KeyR:
				key = KeyRevolverReset
			case k >= glfw.Key1 && k <= glfw.Key4:
				key = KeyRevolverPage1 + Key(k-glfw.Key1)
			}
			if key != KeyNone {
				q.suppressChordChar = a != glfw.Release
				if a == glfw.Press {
					q.cur.Keys = append(q.cur.Keys, key)
				}
				return
			}
		}
		if k == glfw.KeyR && mods&(glfw.ModAlt|glfw.ModControl|glfw.ModSuper) == 0 && a == glfw.Press {
			q.cur.Keys = append(q.cur.Keys, KeyRevolver)
		}
		if k >= glfw.Key0 && k <= glfw.Key9 && thirdBarModifier(mods) {
			q.suppressChordChar = a != glfw.Release
			if a == glfw.Press {
				digit := int(k - glfw.Key0)
				q.cur.Actions = append(q.cur.Actions, 20+(digit+9)%10)
			}
			return
		}
		if k >= glfw.KeyKP0 && k <= glfw.KeyKP9 {
			if a == glfw.Press {
				digit := int(k - glfw.KeyKP0)
				q.cur.Actions = append(q.cur.Actions, 10+(digit+9)%10)
				q.keypadChar = true
			}
			return
		}
		if a == glfw.Release {
			if k >= glfw.KeyA && k <= glfw.KeyZ {
				q.cur.Released += string(rune('A' + k - glfw.KeyA))
			}
			return
		}
		if k == glfw.KeyNumLock {
			if a == glfw.Press {
				q.cur.Keys = append(q.cur.Keys, KeyNumLock)
			}
			return
		}
		if k == glfw.KeyBackspace && a == glfw.Press {
			q.cur.Keys = append(q.cur.Keys, KeyPreviousTarget)
		}
		if mk, ok := keyMap[k]; ok && (a == glfw.Press || mk <= KeyDelete && mk != KeyTab) {
			q.cur.Keys = append(q.cur.Keys, mk)
		}
	})
	w.SetScrollCallback(func(_ *glfw.Window, _, dy float64) { q.cur.Wheel += dy * 120 })
	w.SetMouseButtonCallback(func(_ *glfw.Window, b glfw.MouseButton, a glfw.Action, _ glfw.ModifierKey) {
		if b == glfw.MouseButtonMiddle {
			q.cur.MiddlePressed = a == glfw.Press
			return
		}
		if b == glfw.MouseButtonRight {
			q.cur.RightDown = a == glfw.Press
			if a == glfw.Press {
				q.cur.RightPressed = true
			}
			return
		}
		if b != glfw.MouseButtonLeft {
			return
		}
		if a == glfw.Press {
			q.cur.Pressed, q.cur.MouseDown = true, true
		} else if a == glfw.Release {
			q.cur.Clicked, q.cur.MouseDown = true, false
		}
	})
	q.win = w
	return q
}

type InputQueue struct {
	win               *Window
	cur               Input
	mouselook         bool
	keypadChar        bool
	suppressChordChar bool
	anchorX, anchorY  float64
	lookX, lookY      float64
}

// Frame returns this frame's input and starts collecting the next one.
// Mouse coordinates are in framebuffer pixels.
func (q *InputQueue) Frame() Input {
	x, y := q.win.GetCursorPos()
	ww, _ := q.win.GetSize()
	fw, _ := q.win.GetFramebufferSize()
	scale := 1.0
	if ww > 0 {
		scale = float64(fw) / float64(ww)
	}
	in := q.cur
	in.ReadClipboard, in.WriteClipboard = glfw.GetClipboardString, glfw.SetClipboardString
	in.MouseX, in.MouseY = int(x*scale), int(y*scale)
	if q.mouselook {
		in.MouseLook = true
		in.MouseDX, in.MouseDY = (x-q.lookX)*scale, (y-q.lookY)*scale
		in.MouseX, in.MouseY = int(q.anchorX*scale), int(q.anchorY*scale)
		q.lookX, q.lookY = x, y
	}
	if q.win.GetAttrib(glfw.Focused) == 0 {
		q.cur = Input{}
		return Input{MouseX: in.MouseX, MouseY: in.MouseY}
	}
	var down [32]byte
	n := 0
	for key := glfw.KeyA; key <= glfw.KeyZ; key++ {
		if q.win.GetKey(key) == glfw.Press {
			down[n] = byte('A' + key - glfw.KeyA)
			n++
		}
	}
	for i, key := range []glfw.Key{glfw.KeyLeft, glfw.KeyRight, glfw.KeyUp, glfw.KeyDown, glfw.KeyPageUp, glfw.KeyPageDown} {
		if q.win.GetKey(key) == glfw.Press {
			down[n] = "<>^v+-"[i]
			n++
		}
	}
	in.Down = string(down[:n])
	in.Space = q.win.GetKey(glfw.KeySpace) == glfw.Press
	var mods glfw.ModifierKey
	for _, modifier := range []struct {
		left, right glfw.Key
		flag        glfw.ModifierKey
	}{
		{glfw.KeyLeftControl, glfw.KeyRightControl, glfw.ModControl},
		{glfw.KeyLeftShift, glfw.KeyRightShift, glfw.ModShift},
		{glfw.KeyLeftAlt, glfw.KeyRightAlt, glfw.ModAlt},
		{glfw.KeyLeftSuper, glfw.KeyRightSuper, glfw.ModSuper},
	} {
		if q.win.GetKey(modifier.left) == glfw.Press || q.win.GetKey(modifier.right) == glfw.Press {
			mods |= modifier.flag
		}
	}
	numberRowOffset := 0
	if thirdBarModifier(mods) {
		numberRowOffset = 20
	}
	for digit := 0; digit < 10; digit++ {
		slot := (digit + 9) % 10
		if q.win.GetKey(glfw.Key0+glfw.Key(digit)) == glfw.Press {
			in.ActionHeld |= 1 << (numberRowOffset + slot)
		}
		if q.win.GetKey(glfw.KeyKP0+glfw.Key(digit)) == glfw.Press {
			in.ActionHeld |= 1 << (10 + slot)
		}
	}
	q.cur = Input{MouseDown: in.MouseDown, RightDown: in.RightDown, Shift: in.Shift}
	return in
}

// SetMouseLook provides unbounded right-drag deltas and restores the original
// pointer location on release, matching Game.exe 0x5316e0.
func (q *InputQueue) SetMouseLook(active bool) {
	active = active && q.win.GetAttrib(glfw.Focused) != 0
	if active == q.mouselook {
		return
	}
	q.mouselook = active
	if active {
		q.anchorX, q.anchorY = q.win.GetCursorPos()
		q.win.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
		q.lookX, q.lookY = q.win.GetCursorPos()
	} else {
		q.win.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
		q.win.SetCursorPos(q.anchorX, q.anchorY)
	}
}
