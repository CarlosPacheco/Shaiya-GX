package scene

import "testing"

func TestPlaybackRateFollowsClipKindAndIndices(t *testing.T) {
	var r playbackRate
	if r.factor() != 1 {
		t.Fatal("unknown indices must keep 30 fps")
	}
	r.known, r.move, r.attack, r.weaponBase, r.haveWeapon = true, 4, 6, 5, true
	r.kind = rateMove
	if r.factor() != 36.0/30 {
		t.Fatalf("move %g", r.factor())
	}
	r.kind = rateAttack
	if r.factor() != 34.0/30 {
		t.Fatalf("attack %g", r.factor())
	}
	r.kind = rateNone
	if r.factor() != 1 {
		t.Fatal("other actions keep 30 fps")
	}
	r.creature = 1.5
	if r.factor() != 1.5 {
		t.Fatal("creature override")
	}
}
