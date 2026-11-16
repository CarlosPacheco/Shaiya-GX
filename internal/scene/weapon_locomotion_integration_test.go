//go:build integration

package scene

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestOriginalWeaponLocomotionGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "weapon clips", Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := NewAssets(arc)

	for _, race := range []uint8{0, 3} {
		for gender := uint8(0); gender < 2; gender++ {
			c, err := BuildCharacter(a, Appearance{Race: race, Gender: gender, Job: 0, Helmet: -1, WeaponType: 51}, "000_normal")
			if err != nil {
				t.Fatal(err)
			}
			for _, weapon := range []uint8{45, 46, 49, 51} {
				c.weaponType = weapon
				c.ReleaseAction()
				c.SetClip("002_run")
				if !strings.Contains(strings.ToLower(c.clipFile), "run") || strings.Contains(c.clipFile, "002_run") {
					t.Fatalf("generic run for weapon %d: %s", weapon, c.clipFile)
				}
				c.combatPoseLeft = 10
				c.SetClip("000_normal")
				if !strings.Contains(strings.ToLower(c.clipFile), "ready") {
					t.Fatalf("missing ready %d: %s", weapon, c.clipFile)
				}
				c.combatPoseLeft = 0
				c.SetClip("000_normal")
				if !strings.Contains(c.clipFile, "000_normal") {
					t.Fatalf("stance did not expire: %s", c.clipFile)
				}
				c.SetClip("007_swim")
				if !strings.Contains(c.clipFile, "007_swim") {
					t.Fatal("weapon overrode swimming", c.clipFile)
				}
			}

		}
	}
}
