package authwire

import (
	"strings"
	"testing"
)

func TestBoundedCredentialRoundTrip(t *testing.T) {
	value := strings.Repeat("long passphrase ", 7)
	packet, err := Encode("dev_account", value)
	if err != nil {
		t.Fatal(err)
	}
	user, got, err := Decode(packet)
	if err != nil || user != "dev_account" || got != value {
		t.Fatal(user, got, err)
	}
	for _, bad := range [][]byte{nil, packet[:len(packet)-1], append(packet, 0)} {
		if _, _, err := Decode(bad); err == nil {
			t.Fatal("invalid packet accepted")
		}
	}
	if _, err := Encode("account", strings.Repeat("x", 129)); err == nil {
		t.Fatal("unbounded credential")
	}
}
func FuzzCredentialDecode(f *testing.F) {
	p, _ := Encode("dev", "password")
	f.Add(p)
	f.Fuzz(func(t *testing.T, b []byte) { Decode(b) })
}
