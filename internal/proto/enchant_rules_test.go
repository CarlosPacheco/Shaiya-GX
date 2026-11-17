package proto

import "testing"

func TestEnchantRulesRoundTripAndMalformed(t *testing.T) {
	r := EnchantRules{MaxLevel: 49}
	r.ArmorBonus[49] = 888
	r.WeaponBonus[20] = 311
	p := EnchantRulesPacket(r)
	got, err := DecodeEnchantRules(p)
	if err != nil || got != r {
		t.Fatal(got, err)
	}
	invalid := r
	invalid.MaxLevel = 50
	overflow := r
	overflow.ArmorBonus[1] = 2147483648
	for _, bad := range [][]byte{nil, p[:len(p)-1], append(append([]byte{}, p...), 0), EnchantRulesPacket(invalid), EnchantRulesPacket(overflow)} {
		if _, err := DecodeEnchantRules(bad); err == nil {
			t.Fatal("invalid rules accepted")
		}
	}
}
