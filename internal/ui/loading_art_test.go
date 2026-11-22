package ui

import (
	"reflect"
	"testing"
)

func TestMapScreensIgnoreInvalidEntries(t *testing.T) {
	got := ParseMapScreens([]byte("\xef\xbb\xbf; comment\n[MapScreens]\n64=loading64.png\n 7 = seven.png \nx=bad.png\n70000=big.png\n9=../escape.png\n10=\n11\n# 12=off.png\n"))
	want := map[uint16]string{64: "loading64.png", 7: "seven.png"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestRandomLoadingPoolFollowsTheFiles(t *testing.T) {
	files := []string{"loading.png", "random_loading01.png", "Random_Loading15.PNG", "random_loading02.jpg", "random_loadingx.png", "loadbar.png", "random_loading11.png"}
	want := []string{"random_loading01.png", "Random_Loading15.PNG", "random_loading11.png"}
	if got := RandomLoadingNames(files); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
