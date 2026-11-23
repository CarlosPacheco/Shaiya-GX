package app

import "testing"

func TestGMTargetArgumentBecomesTheSelectedID(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 5
	for _, text := range []string{"/mera 100", "/kill", "/set 1 2", "hello t"} {
		if got, ok := a.gmTargetText(text); !ok || got != text {
			t.Fatal("a command without the t argument was rewritten", text, got)
		}
	}
	a.state.Target = 0
	if _, ok := a.gmTargetText("/kill t"); ok {
		t.Fatal("/kill t with no target went through")
	}
	a.state.Target = 5
	if _, ok := a.gmTargetText("/mera t"); ok {
		t.Fatal("/mera t aimed at yourself")
	}
	a.state.Target = 77
	for text, want := range map[string]string{"/mera t": "/mera #77", "/NERA T": "/NERA #77", "/kill t": "/kill #77"} {
		if got, ok := a.gmTargetText(text); !ok || got != want {
			t.Fatalf("%q became %q", text, got)
		}
	}
}
