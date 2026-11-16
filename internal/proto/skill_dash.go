package proto

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/skilltiming"
	"math"
)

// GXSkillDash is a server-authored Stinger trajectory. Native skill and move
// packets keep their layouts. Clients never submit dash destinations or speed.
const GXSkillDash Opcode = 0x7F11

type SkillDash struct {
	Stopped  bool
	Source   uint32
	From, To [3]float32
	Started  float64 // synchronized map physics seconds
	Contact  float32 // native ALT contact frame
}

func SkillDashPacket(d SkillDash) []byte {
	w := NewWriter(GXSkillDash)
	w.U32(d.Source)
	for _, p := range [][3]float32{d.From, d.To} {
		for _, v := range p {
			w.F32(v)
		}
	}
	bits := math.Float64bits(d.Started)
	w.U32(uint32(bits))
	w.U32(uint32(bits >> 32))
	w.F32(d.Contact)
	if d.Stopped {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}
func DecodeSkillDash(p []byte) (d SkillDash, err error) {
	if len(p) != 43 || p[42] > 1 || Op(p) != GXSkillDash {
		return d, fmt.Errorf("invalid skill dash size")
	}
	r := NewReader(p)
	d.Source = r.U32()
	for i := range d.From {
		d.From[i] = r.F32()
	}
	for i := range d.To {
		d.To[i] = r.F32()
	}
	d.Started = math.Float64frombits(r.U64())
	d.Contact = r.F32()
	d.Stopped = r.Bool()
	if d.Source == 0 || math.IsNaN(d.Started) || math.IsInf(d.Started, 0) || d.Started < 0 || d.Contact < 10 || d.Contact > 30 {
		return SkillDash{}, fmt.Errorf("invalid skill dash metadata")
	}
	for _, v := range append(d.From[:], d.To[:]...) {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return SkillDash{}, fmt.Errorf("non-finite skill dash")
		}
	}
	if math.IsNaN(float64(d.Contact)) || d.Distance() > 32 {
		return SkillDash{}, fmt.Errorf("invalid skill dash range")
	}
	return d, r.Err()
}
func (d SkillDash) Distance() float64 {
	var n float64
	for i := range d.From {
		x := float64(d.To[i] - d.From[i])
		n += x * x
	}
	return math.Sqrt(n)
}
func (d SkillDash) TravelEnd() float64 {
	return skilltiming.StingerWindup + d.Distance()/skilltiming.StingerSpeed
}
func (d SkillDash) Duration() float64 { return d.TravelEnd() + float64(d.Contact-10)/30 }
func (d SkillDash) Position(elapsed float64) [3]float32 {
	f := float32(1)
	if n := d.Distance(); n > 0 {
		f = float32(min(1, max(0, (elapsed-skilltiming.StingerWindup)*skilltiming.StingerSpeed/n)))
	}
	p := d.From
	for i := range p {
		p[i] += (d.To[i] - d.From[i]) * f
	}
	return p
}
func (d SkillDash) Frame(elapsed float64) float32 {
	if elapsed <= skilltiming.StingerWindup {
		return float32(max(elapsed, 0) * 30)
	}
	return 10 + float32(max(0, elapsed-d.TravelEnd())*30)
}
