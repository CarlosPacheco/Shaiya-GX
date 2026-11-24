package render

import (
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

const (
	hoverDepthRadius = 4
	hoverDepthSide   = 2*hoverDepthRadius + 1
	hoverDepthPixels = hoverDepthSide * hoverDepthSide
	hoverDepthBytes  = hoverDepthPixels * 4
	attackHoverGrace = 120 * time.Millisecond
)

type hoverDepthSample struct {
	frame               uint64
	x, y, width, height int
	reversed            bool
	mouseX, mouseY      int
	capturedAt          time.Time
}
type hoverDepthSlot struct {
	sample hoverDepthSample
	buffer uint32
	fence  uintptr
}
type hoverDepthReadback struct {
	slots  [3]hoverDepthSlot
	ready  hoverDepthSample
	values *[hoverDepthPixels]float32
	issued uint64
}

// NameHoverVisibleAt reads completed depth transfers without waiting on the GPU.
// Copy a small region so the next frame can test the exact pointer pixel even
// after a short move. Clicks use NameVisibleAt for the current frame's depth.
func (s *Sprites) NameHoverVisibleAt(x, y, width, height int, depth float32) bool {
	return s.nameHoverVisibleAt(x, y, width, height, depth, false)
}

// NameAttackHoverVisibleAt bridges a short transfer delay at the original sample point.
func (s *Sprites) NameAttackHoverVisibleAt(x, y, width, height int, depth float32) bool {
	return s.nameHoverVisibleAt(x, y, width, height, depth, true)
}

func (s *Sprites) nameHoverVisibleAt(x, y, width, height int, depth float32, attackHover bool) bool {
	if s == nil || !s.depthValid || s.depth == nil || s.depth.W != width || s.depth.H != height || x < 0 || y < 0 || x >= width || y >= height {
		return false
	}
	r := &s.hoverDepth
	if r.issued != s.depthFrame {
		var available *hoverDepthSlot
		for n := range r.slots {
			slot := &r.slots[n]
			if slot.fence != 0 {
				status := gl.ClientWaitSync(slot.fence, 0, 0)
				if status == gl.ALREADY_SIGNALED || status == gl.CONDITION_SATISFIED {
					if slot.sample.frame > r.ready.frame {
						if r.values == nil {
							// Keep the GL copy destination in a standalone pointer-free allocation.
							r.values = new([hoverDepthPixels]float32)
						}
						bytes := slot.sample.width * slot.sample.height * 4
						gl.GetNamedBufferSubData(slot.buffer, 0, bytes, gl.Ptr(&r.values[0]))
						r.ready = slot.sample
					}
					gl.DeleteSync(slot.fence)
					slot.fence = 0
				} else if status == gl.WAIT_FAILED {
					gl.DeleteSync(slot.fence)
					slot.fence = 0
				}
			}
			if slot.fence == 0 && available == nil {
				available = slot
			}
		}
		if available != nil {
			if available.buffer == 0 {
				gl.CreateBuffers(1, &available.buffer)
				gl.NamedBufferData(available.buffer, hoverDepthBytes, nil, gl.STREAM_READ)
				addGPUBytes(hoverDepthBytes)
			}
			left, top := max(0, x-hoverDepthRadius), max(0, y-hoverDepthRadius)
			right, bottom := min(width, x+hoverDepthRadius+1), min(height, y+hoverDepthRadius+1)
			regionWidth, regionHeight := right-left, bottom-top
			available.sample = hoverDepthSample{
				frame: s.depthFrame, x: left, y: top, width: regionWidth, height: regionHeight,
				reversed: s.depthReversed, mouseX: x, mouseY: y, capturedAt: time.Now(),
			}
			gl.BindBuffer(gl.PIXEL_PACK_BUFFER, available.buffer)
			gl.GetTextureSubImage(s.depth.ID, 0, int32(left), int32(height-bottom), 0, int32(regionWidth), int32(regionHeight), 1, gl.DEPTH_COMPONENT, gl.FLOAT, hoverDepthBytes, nil)
			gl.BindBuffer(gl.PIXEL_PACK_BUFFER, 0)
			available.fence = gl.FenceSync(gl.SYNC_GPU_COMMANDS_COMPLETE, 0)
		}
		r.issued = s.depthFrame
	}
	sample := &r.ready
	column, row := x-sample.x, y-sample.y
	if sample.frame == 0 || sample.reversed != s.depthReversed ||
		column < 0 || column >= sample.width || row < 0 || row >= sample.height {
		return false
	}
	if s.depthFrame-sample.frame > 3 &&
		(!attackHover || x != sample.mouseX || y != sample.mouseY || time.Since(sample.capturedAt) >= attackHoverGrace) {
		return false
	}
	// The pointer uses top-down window coordinates; OpenGL packs bottom-up rows.
	value := r.values[(sample.height-1-row)*sample.width+column]
	if sample.reversed {
		value = 1 - value
	}
	return value+.000001 >= depth
}

// ResetHoverDepth drops samples across map changes, resizes and shutdown.
func (s *Sprites) ResetHoverDepth() {
	if s == nil {
		return
	}
	for _, slot := range s.hoverDepth.slots {
		if slot.fence != 0 {
			gl.DeleteSync(slot.fence)
		}
		if slot.buffer != 0 {
			gl.DeleteBuffers(1, &slot.buffer)
			removeGPUBytes(hoverDepthBytes)
		}
	}
	s.hoverDepth = hoverDepthReadback{}
}
