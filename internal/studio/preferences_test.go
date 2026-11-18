package studio

import "testing"

func TestInstallationPathsRoundTrip(t *testing.T) {
	root := t.TempDir()
	got, err := LoadInstallationPaths(root)
	if err != nil || got != (InstallationPaths{}) {
		t.Fatalf("first launch: %+v, %v", got, err)
	}
	for _, want := range []InstallationPaths{
		{DataDir: `C:\GX Studio\Client Data`, Database: `D:\GX Server\runtime\rapture.sqlite`},
		{DataDir: `/home/test/Client "GX" /data`, Database: "/home/test/#archivo=1.sqlite"},
	} {
		if err := SaveInstallationPaths(root, want); err != nil {
			t.Fatal(err)
		}
		got, err = LoadInstallationPaths(root)
		if err != nil || got != want {
			t.Fatalf("got %+v, %v; want %+v", got, err, want)
		}
	}
}
