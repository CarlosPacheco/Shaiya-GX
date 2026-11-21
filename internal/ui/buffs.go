package ui

import "fmt"

// BuffLayout keeps separate movable positive/negative effect strips. Native
// icons are 32px; the 6px gap follows Phoenix, duration bars follow 0x4a2fa0.
type BuffLayout struct{ Position [2]WindowPosition }

func (b *BuffLayout) Rect(w, h, group, count int) (x, y, width, height int) {
	columns := max(1, min(10, w/38))
	width = max(32, min(max(1, count), columns)*38-6)
	height = max(37, ((max(1, count)+columns-1)/columns)*44-7)
	dx := max(0, min(w/2+128, w-width))
	returnOrigin := b.Position[group]
	x, y = returnOrigin.Origin(w, h, width, height, dx, 60+group*48)
	return
}
func (b *BuffLayout) Slot(w, h, group, index, count int) (int, int) {
	x, y, _, _ := b.Rect(w, h, group, count)
	columns := max(1, min(10, w/38))
	return x + (index%columns)*38, y + (index/columns)*44
}
func BuffSlot(w, index int) (int, int) { b := BuffLayout{}; return b.Slot(w, 768, 0, index, 32) }
func BuffHit(w, count, mx, my int) int {
	for i := 0; i < count; i++ {
		x, y := BuffSlot(w, i)
		if mx >= x && mx < x+32 && my >= y && my < y+37 {
			return i
		}
	}
	return -1
}

// BuffDuration is presentation only; reaching zero never removes a buff.
func BuffDuration(seconds uint32, timed bool) string {
	if !timed {
		return "-"
	}
	if seconds >= 3600 {
		return fmt.Sprintf("%dh", (seconds-1)/3600+1)
	}
	if seconds >= 60 {
		return fmt.Sprintf("%dm", (seconds-1)/60+1)
	}
	return fmt.Sprintf("%ds", seconds)
}

// BuffRemainingText is the PS0107 hover footer (0x49eb50). The caller in
// 0x4b8e02 omits it when no whole seconds remain.
func BuffRemainingText(seconds uint32) string {
	if seconds == 0 {
		return ""
	}
	days := seconds / 86400
	clock := fmt.Sprintf("%02d:%02d:%02d", seconds/3600%24, seconds/60%60, seconds%60)
	if days != 0 {
		return fmt.Sprintf("%02d - %s", days, clock)
	}
	return clock
}
