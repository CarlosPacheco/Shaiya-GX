//go:build integration

package app

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestAuctionPanelsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires a current OpenGL runtime and installed data")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "GX auction test", Width: 1024, Height: 768, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state = game.New()
	a.auction = auctionView{npc: 1, selected: 0, focus: -1, rows: []proto.AuctionRow{{ID: 1, Name: "Healing Potion", Seller: "Merchant", Type: 25, TypeID: 1, Count: 4, Price: 550, Buyout: 1200, Ends: time.Now().Add(time.Hour).Unix(), Active: true}}}
	for view := proto.AuctionBrowse; view <= proto.AuctionWatch; view++ {
		a.auction.view = view
		for _, modal := range []string{"", "setup", "bid", "buy", "cancel", "details"} {
			a.auction.modal = modal
			a.sprites.Begin(1024, 768)
			a.drawAuction(1024, 768)
			a.sprites.End()
			if code := gl.GetError(); code != gl.NO_ERROR {
				t.Fatalf("view %d modal %q GL error %#x", view, modal, code)
			}
		}
	}
}
