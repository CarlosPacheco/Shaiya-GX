package scene

import "testing"

func TestEveryCharacterUsesCorrectRangedAndSecondaryHand(t *testing.T) {
	for prefix, bones := range itemBones {
		for _, base := range []uint8{13, 14} {
			bone, m := itemAttachmentBinding(prefix, base, false, false)
			if bone != bones[1] || m[0] != 1 || m[12] != 0 {
				t.Fatal(prefix, base, "bow/crossbow hand", bone, m)
			}
		}
		bone, _ := itemAttachmentBinding(prefix, 11, false, false)
		if bone != bones[0] {
			t.Fatal(prefix, "javelin held in bow hand", bone)
		}
		bone, m := itemAttachmentBinding(prefix, 5, false, true)
		reversed := prefix[:2] == "hu" || prefix[:2] == "de"
		if bone != bones[1] || (m[0] < 0) != reversed {
			t.Fatal(prefix, "dual hand orientation", bone, m)
		}
	}
	bone, _ := itemAttachmentBinding("elmm", 19, true, false)
	if bone != 13 {
		t.Fatal("elmm shield must follow corrected Phoenix bone13", bone)
	}
}
