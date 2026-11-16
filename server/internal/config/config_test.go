package config

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestSQLiteFileURIsHaveNoAuthority(t *testing.T) {
	for _, path := range []string{"/tmp/Portable copy/área.sqlite", "C:/Portable copy/rapture.sqlite", "Z:/tmp/Portable copy/rapture.sqlite", "//server/share/rapture.sqlite"} {
		uri := url.URL{Scheme: "file", Path: sqliteURIPath(path)}
		parsed, err := url.Parse(uri.String())
		if err != nil || parsed.Host != "" || parsed.Opaque != "" {
			t.Fatalf("path %q became a non-file authority: %s, %v", path, uri.String(), err)
		}
		if len(path) > 1 && path[1] == ':' && parsed.Path != "/"+path {
			t.Fatalf("drive path was lost: %s", uri.String())
		}
	}
}

func TestSQLiteDSNRetainsPathsAndConnectionSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Portable copy", "área.sqlite")
	t.Setenv("RAPTURE_DB_PATH", path)
	value, err := DSN("PS_GameData")
	if err != nil {
		t.Fatal(err)
	}
	uri, err := url.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	if uri.Host != "" || uri.Path != sqliteURIPath(filepath.ToSlash(path)) {
		t.Fatal("database path changed", value)
	}
	if uri.Query().Get("_txlock") != "immediate" || len(uri.Query()["_pragma"]) != 4 {
		t.Fatal("connection settings changed", value)
	}
}

func TestFallDamageDefaultsEnabledAndAllowsExplicitOptOut(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
	}{{"", true}, {"true", true}, {"false", false}, {"invalid", false}} {
		t.Setenv("RAPTURE_FALL_DAMAGE", tc.value)
		if got := FallDamageEnabled(); got != tc.want {
			t.Fatal(tc.value, got)
		}
	}
}

func TestNetworkLimitsCannotBeDisabledAccidentally(t *testing.T) {
	for _, value := range []string{"0", "-1", "unlimited", "999999999999"} {
		t.Setenv("RAPTURE_CONNECTIONS_PER_IP", value)
		if _, err := Network("world"); err == nil {
			t.Fatal("unsafe policy accepted", value)
		}
	}
	t.Setenv("RAPTURE_CONNECTIONS_PER_IP", "64")
	policy, err := Network("world")
	if err != nil || policy.PerIP != 64 {
		t.Fatal(policy, err)
	}
}
func TestMalformedEnvDoesNotPartiallyApply(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".local"), 0700); err != nil {
		t.Fatal(err)
	}
	const key = "GX_TEST_CONFIG_VALIDATION_UNIQUE"
	if err := os.WriteFile(filepath.Join(root, ".local/runtime.env"), []byte(key+"=unexpected\ninvalid assignment\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := LoadRuntimeEnvFrom(root); err == nil {
		t.Fatal("malformed configuration accepted")
	}
	if _, ok := os.LookupEnv(key); ok {
		t.Fatal("partially applied configuration")
	}
}
