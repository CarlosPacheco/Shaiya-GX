package platform

import (
	"fmt"
	"unicode"

	"github.com/go-gl/glfw/v3.4/glfw"
)

type PhysicalKey = glfw.Key

const PhysicalEscape = glfw.KeyEscape

// Slots follow the two columns of Hotkey Options, then Extra Hotkey's ten
// buttons. A zero entry uses the built-in default.
type KeyBindings [30]PhysicalKey

var defaultBindings = KeyBindings{
	glfw.KeyA, glfw.KeyD, glfw.KeyW, glfw.KeyS, glfw.KeyQ, glfw.KeyE,
	glfw.KeyTab, glfw.KeyNumLock, glfw.KeyC, glfw.KeySpace, glfw.KeyT, glfw.KeyI,
	glfw.KeyK, glfw.KeyB, glfw.KeyU, glfw.KeyG, glfw.KeyO, glfw.KeyM, glfw.KeyV, glfw.KeyEscape,
	glfw.KeyKP1, glfw.KeyKP2, glfw.KeyKP3, glfw.KeyKP4, glfw.KeyKP5, glfw.KeyKP6,
	glfw.KeyKP7, glfw.KeyKP8, glfw.KeyKP9, glfw.KeyKP0,
}

func (b KeyBindings) At(slot int) PhysicalKey {
	if b[slot] != 0 {
		return b[slot]
	}
	return defaultBindings[slot]
}

func bindingKeyAllowed(k PhysicalKey) bool {
	for _, key := range defaultBindings {
		if k == key {
			return true
		}
	}
	switch k {
	case glfw.KeyJ, glfw.KeyL, glfw.KeyY:
		return true
	}
	return false
}

func (b KeyBindings) Valid() bool {
	for i := range b {
		key := b.At(i)
		if !bindingKeyAllowed(key) {
			return false
		}
		for j := 0; j < i; j++ {
			if b.At(j) == key {
				return false
			}
		}
	}
	return true
}

// Bind exchanges the old assignments when the chosen key is already in use.
func (b *KeyBindings) Bind(slot int, key PhysicalKey) bool {
	if !bindingKeyAllowed(key) {
		return false
	}
	old := b.At(slot)
	if old == key {
		return true
	}
	for i := range b {
		if b.At(i) == key {
			b[i] = old
			if old == defaultBindings[i] {
				b[i] = 0
			}
			break
		}
	}
	b[slot] = key
	if key == defaultBindings[slot] {
		b[slot] = 0
	}
	return true
}

func BindingName(k PhysicalKey) string {
	switch {
	case k >= glfw.KeyA && k <= glfw.KeyZ:
		return string(rune(k))
	case k >= glfw.KeyKP0 && k <= glfw.KeyKP9:
		return fmt.Sprintf("Num %d", k-glfw.KeyKP0)
	}
	switch k {
	case glfw.KeyTab:
		return "Tab"
	case glfw.KeyEscape:
		return "Esc"
	case glfw.KeySpace:
		return "Space"
	case glfw.KeyNumLock:
		return "NumLock"
	}
	return ""
}

func bindingKeyBit(k PhysicalKey) uint64 {
	switch {
	case k >= glfw.KeyA && k <= glfw.KeyZ:
		return 1 << (k - glfw.KeyA)
	case k >= glfw.KeyKP0 && k <= glfw.KeyKP9:
		return 1 << (k - glfw.KeyKP0 + 26)
	case k == glfw.KeySpace:
		return 1 << 36
	case k == glfw.KeyTab:
		return 1 << 37
	case k == glfw.KeyEscape:
		return 1 << 38
	case k == glfw.KeyNumLock:
		return 1 << 39
	}
	return 0
}

func movementBinding(slot int) bool { return slot >= 0 && slot < 6 }

func (b KeyBindings) boundMask() uint64 {
	var mask uint64
	for i, key := range defaultBindings {
		mask |= bindingKeyBit(key) | bindingKeyBit(b.At(i))
	}
	return mask
}

func (b KeyBindings) slot(key PhysicalKey) int {
	for i := range b {
		if b.At(i) == key {
			return i
		}
	}
	return -1
}

// WithBindings translates hardware keys for gameplay. Text fields continue to
// receive the Unicode character events and editing keys from the raw input.
func (in Input) WithBindings(b KeyBindings) Input {
	if !in.physical {
		return in
	}
	mask := b.boundMask()
	chars := make([]rune, 0, len(in.Chars))
	repeats := make([]bool, 0, len(in.Chars))
	for i, char := range in.Chars {
		if i >= len(in.charKeys) || bindingKeyBit(in.charKeys[i])&mask == 0 {
			chars = append(chars, char)
			repeats = append(repeats, i < len(in.CharRepeats) && in.CharRepeats[i])
		}
	}
	in.Chars, in.CharRepeats, in.KeypadChars = chars, repeats, ""
	keys := make([]Key, 0, len(in.Keys))
	for _, key := range in.Keys {
		switch key {
		case KeyTab, KeyEscape, KeySpace, KeyNumLock:
		default:
			keys = append(keys, key)
		}
	}
	in.Keys = keys
	actions := make([]int, 0, len(in.Actions))
	for _, action := range in.Actions {
		if action >= 20 {
			actions = append(actions, action)
		}
	}
	in.Actions = actions
	in.ActionHeld &^= ((1 << 10) - 1) << 10
	var down [32]byte
	n := 0
	for _, key := range in.Down {
		if bindingKeyBit(PhysicalKey(key))&mask == 0 {
			down[n] = byte(key)
			n++
		}
	}
	for i := range b {
		if in.heldKeys&bindingKeyBit(b.At(i)) == 0 {
			continue
		}
		if movementBinding(i) {
			down[n] = byte(defaultBindings[i])
			n++
		} else if i >= 20 {
			in.ActionHeld |= 1 << (i - 10)
		}
	}
	in.Down = string(down[:n])
	in.Space = in.heldKeys&bindingKeyBit(b.At(9)) != 0
	in.Released = ""
	for _, k := range in.keyReleases {
		if slot := b.slot(k); movementBinding(slot) {
			in.Released += string(rune(defaultBindings[slot]))
		}
	}
	for _, k := range in.KeyPresses {
		slot := b.slot(k)
		if k == glfw.KeyL && slot < 0 {
			in.Keys = append(in.Keys, KeyPersonalLight)
		}
		if slot < 0 || movementBinding(slot) {
			continue
		}
		switch slot {
		case 6:
			in.Keys = append(in.Keys, KeyTab)
		case 7:
			in.Keys = append(in.Keys, KeyNumLock)
		case 9:
			in.Keys = append(in.Keys, KeySpace)
		case 19:
			in.Keys = append(in.Keys, KeyEscape)
		default:
			if slot >= 20 {
				in.Actions = append(in.Actions, slot-10)
			} else {
				in.Chars = append(in.Chars, unicode.ToLower(rune(defaultBindings[slot])))
				in.CharRepeats = append(in.CharRepeats, false)
			}
		}
	}
	return in
}
