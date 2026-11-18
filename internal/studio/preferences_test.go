package studio

import "testing"

func TestPreferencesRoundTrip(t *testing.T) {
	root := t.TempDir()
	got, err := LoadPreferences(root)
	if err != nil || got != (Preferences{}) {
		t.Fatalf("first launch: %+v, %v", got, err)
	}
	for _, want := range []Preferences{
		{DataDir: `C:\GX Studio\Client Data`, Database: `D:\GX Server\runtime\rapture.sqlite`, Theme: "dark-blue"},
		{DataDir: `/home/test/Client "GX" /data`, Database: "/home/test/#archivo=1.sqlite"},
	} {
		if err := SavePreferences(root, want); err != nil {
			t.Fatal(err)
		}
		got, err = LoadPreferences(root)
		if err != nil || got != want {
			t.Fatalf("got %+v, %v; want %+v", got, err, want)
		}
	}
}
