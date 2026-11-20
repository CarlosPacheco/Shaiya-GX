package chatguard

import (
	"strings"
	"testing"
)

func TestFilterRedactsUnsafeDestinations(t *testing.T) {
	for _, link := range []string{
		`file:///tmp/guide.txt`, `\\lan\shared\guide.txt`, `C:\Users\Player\guide.txt`,
		`/home/player/guide.txt`, `//localhost/guide`, `game.localhost/guide`,
		`http://127.0.0.1/guide`, `http://[::ffff:192.168.1.1]/guide`, `http://2130706433/guide`,
		`javascript:void(0)`, `steam://open/main`, `ms-settings:display`,
		`https://example.org/client%2Eexe?source=chat`, `hxxps[:]//example[.]org/setup.sh`,
		`https://gіthub.com/guide`, `https://github.com.example.org/guide`, `https://discord-login.example.org/`,
		`https://player@example.org/guide`,
	} {
		t.Run(link, func(t *testing.T) {
			message := "Guía («" + link + "»), y https://example.org/manual."
			got := Filter(message, false)
			if !got.Quiet || got.Impersonation || got.Text != "Guía («[link blocked]»), y https://example.org/manual." {
				t.Fatalf("unexpected result: %+v", got)
			}
		})
	}
}

func TestFilterPreservesOrdinaryChat(t *testing.T) {
	for _, message := range []string{
		"Venta 1.5 M! https://example.org/guide, github.com/Spelunkern/Shaiya-GX",
		"https://例え.jp/ https://mydiscordbot.example.org/ correo player@example.org",
		"Hola 👩‍💻 مرحبا Живой чат",
		"[GM] El evento empieza a las 20:00",
		"Nunca compartas tu contraseña con un GM",
	} {
		if got := Filter(message, false); got.Text != message || got.Quiet || got.Impersonation {
			t.Errorf("unexpected result for %q: %+v", message, got)
		}
	}
}

func TestFilterStaffImpersonationAndUnicode(t *testing.T) {
	message := "［ＧＭ］ Mandame tu contraseña"
	if got := Filter(message, false); !got.Impersonation || !got.Quiet || got.Text != "" {
		t.Fatalf("unverified staff lure was visible: %+v", got)
	}
	if got := Filter(message, true); got.Text != message || got.Quiet || got.Impersonation {
		t.Fatalf("verified staff text was hidden: %+v", got)
	}
	if got := Filter("Guía: https://example.org/client.\u202eexe", true); got.Text != "Guía: [link blocked]" || !got.Quiet {
		t.Fatalf("verified staff path bypassed link checks: %+v", got)
	}
	if got := Filter("A\u2066B\u200bC\nD 👩‍💻 "+strings.Repeat("!", 140), false); got.Text != "ABC D 👩‍💻 "+strings.Repeat("!", 12) || !got.Quiet {
		t.Fatalf("unsafe formatting or repeated glyphs survived: %+v", got)
	}
}

func TestSimilarSpamKeepsTradePricesDistinct(t *testing.T) {
	for _, test := range []struct {
		a, b string
		same bool
	}{
		{"VENDO espada legendaria!", "vendo espada le\u200bgendaria", true},
		{"vendo espada legendaria", "vendo espada legenaria", true},
		{"vendo espada legendaria 1.5 M", "vendo espada legendaria 15 M", false},
		{"vendo espada legendaria 32 M", "vendo espada legendaria 33 M", false},
		{"inv", "int", false},
	} {
		if got := Similar(SpamKey(test.a), SpamKey(test.b)); got != test.same {
			t.Errorf("Similar(%q, %q) = %t, want %t", test.a, test.b, got, test.same)
		}
	}
}
