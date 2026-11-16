package effect

// Meteor describes the original CMeteor controller, not another effect asset.
// Game.exe 0x450780/0x430530/0x430a80 selects it before ordinary EFT playback.
// Sequences and meshes remain shared with the existing renderer.
type Meteor struct {
	Duration, Rate, Height, Lateral, Drain, ImpactHeight float32
	FallingSequence, ImpactSequence                      int
	MaxLive                                              int
}

func SkillMeteor(skill uint16) (Meteor, bool) {
	switch skill {
	case 53, 68, 71, 162, 163:
		p := Meteor{Duration: 5, Rate: 5, Height: 15, Lateral: 7.5, Drain: 1, ImpactHeight: 1, FallingSequence: 0, ImpactSequence: 1, MaxLive: 50}
		if skill == 53 {
			p.Duration, p.Rate, p.Height, p.Lateral = 2, 10, 20, 10
		}
		return p, true
	}
	return Meteor{}, false
}
