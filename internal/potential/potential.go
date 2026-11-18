// Package potential holds the rules shared by the Potential skills
// (TypeEffect 0, TypeDetail 6: Power, Avoid, Nimble, Viable and Motive).
//
// A Potential is a learned, non-activatable skill. When the character's HP
// falls below the skill's LimitHP percent the server applies it as an ordinary
// buff for KeepSeconds and starts its reuse timer. The original data leaves
// KeepTime and ResetTime at zero (the numbers lived in server code), so both
// client and server fill them in when the definitions are loaded.
package potential

const (
	Detail       = 6   // TypeDetail of every Potential
	KeepSeconds  = 30  // buff duration, from the skill descriptions
	ReuseSeconds = 180 // three minutes, the generic spell reuse time
)

// Is reports whether a skill with this TypeEffect and TypeDetail is a Potential.
func Is(effect uint8, detail uint16) bool { return effect == 0 && detail == Detail }
