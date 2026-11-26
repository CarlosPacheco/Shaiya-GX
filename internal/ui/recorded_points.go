package ui

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// RecordedPoints uses the original waypoint panel at 0x5dca21, its two
// controls per location, and the text positions from 0x5dcf36.
const RecordedPointsWidth, RecordedPointsHeight = 256, 420

type RecordedPoints struct {
	Position     WindowPosition
	frame        *render.Texture
	close        Button
	save, travel [proto.TeleportPointCount]Button
}

func NewRecordedPoints(t Textures) (*RecordedPoints, error) {
	var frame, close, save, travel *render.Texture
	var err error
	if t != nil {
		ld := &loader{t: t}
		frame = ld.get("waypoint.png")
		close = ld.get("talk1_close_button.png")
		save = ld.get("way_button.png")
		travel = ld.get("gatekeeper_button.png")
		err = ld.err
	}
	v := &RecordedPoints{frame: frame, close: nativeClose(close, 228, 1)}
	for i := range v.save {
		y := i * 92
		v.save[i] = Button{Tex: save, OffX: 120, OffY: 50 + y, DrawW: 128, DrawH: 32, HitW: 78, HitH: 21, UV: FourRowUV}
		v.travel[i] = Button{Tex: travel, OffX: 179, OffY: 76 + y, DrawW: 64, DrawH: 64, HitW: 58, HitH: 40,
			UV: [5]UV{{0, 0, .25, 1}, {.25, 0, .5, 1}, {.5, 0, .75, 1}, {.5, 0, .75, 1}, {.75, 0, 1, 1}}}
	}
	return v, err
}

func (v *RecordedPoints) Origin(w, h int) (int, int) {
	return v.Position.Origin(w, h, RecordedPointsWidth, RecordedPointsHeight, w/2-127, h/2-180)
}

func (v *RecordedPoints) controls(w, h int, busy bool, points [proto.TeleportPointCount]proto.TeleportPoint) {
	x, y := v.Origin(w, h)
	v.close.OriginX, v.close.OriginY, v.close.Disabled = x, y, busy
	for i := range v.save {
		v.save[i].OriginX, v.save[i].OriginY, v.save[i].Disabled = x, y, busy
		v.travel[i].OriginX, v.travel[i].OriginY, v.travel[i].Disabled = x, y, busy || !points[i].Saved
	}
}

func (v *RecordedPoints) Update(in *platform.Input, w, h int, busy bool, points [proto.TeleportPointCount]proto.TeleportPoint) (action, slot uint8, closed bool) {
	v.controls(w, h, busy, points)
	for i := range v.save {
		if v.save[i].Update(in) {
			return proto.TeleportSave, uint8(i), false
		}
		if v.travel[i].Update(in) {
			return proto.TeleportTravel, uint8(i), false
		}
	}
	return 0, 0, v.close.Update(in) || !busy && in.CloseOnEscape()
}

func (v *RecordedPoints) Draw(s *render.Sprites, f *Font, messages text.SysMsg, w, h int, busy bool, points [proto.TeleportPointCount]proto.TeleportPoint) {
	x, y := v.Origin(w, h)
	v.controls(w, h, busy, points)
	s.DrawImage(v.frame, float32(x), float32(y))
	v.close.Draw(s)
	for i, point := range points {
		v.save[i].Draw(s)
		// The original hides travel until there is a recorded destination.
		if point.Saved {
			v.travel[i].Draw(s)
			f.Draw(s, fmt.Sprintf("%d %d %d", int(point.X), int(point.Y), int(point.Z)), x+39, y+83+i*92, white, 135)
			f.Draw(s, MapName(messages, point.Map), x+39, y+102+i*92, white, 135)
		}
	}
}
