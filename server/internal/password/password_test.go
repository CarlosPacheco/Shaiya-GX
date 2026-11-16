package password

import "testing"

func TestHashAndUpgrade(t *testing.T) {
	a, e := Hash("local")
	if e != nil {
		t.Fatal(e)
	}
	b, e := Hash("local")
	if e != nil || a == b {
		t.Fatal("salt must vary", e)
	}
	for _, tc := range []struct {
		stored, value string
		valid, legacy bool
	}{
		{a, "local", true, false}, {a, "wrong", false, false}, {"local", "local", true, true},
		{"$argon2id$v=19$m=999999999,t=2,p=1$x$x", "local", false, false}, {a + "$extra", "local", false, false},
	} {
		ok, legacy := Verify(tc.stored, tc.value)
		if ok != tc.valid || legacy != tc.legacy {
			t.Fatal(tc, ok, legacy)
		}
	}
}
