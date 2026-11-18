package proto

import "testing"

func TestSkillCooldownWire(t *testing.T) {
	w := NewWriter(GXSkillCooldown)
	w.U16(24)
	w.U16(180)
	c, err := DecodeSkillCooldown(w.Bytes())
	if err != nil || c.Skill != 24 || c.Seconds != 180 {
		t.Fatal(c, err)
	}
	if _, err := DecodeSkillCooldown([]byte{0x16, 0x7f, 1}); err == nil {
		t.Fatal("short cooldown accepted")
	}
}
