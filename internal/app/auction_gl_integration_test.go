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
	a.auction.other = auctionList{selected: 0, rows: []proto.AuctionRow{
		{ID: 2, Name: "Gold settlement", Gold: 15380, Reason: "sold", Ends: time.Now().Add(24 * time.Hour).Unix()},
	}}
	for view := proto.AuctionBrowse; view <= proto.AuctionWatch; view++ {
		a.auction.view = view
		for _, modal := range []string{"", "setup", "bid", "buy", "cancel", "details"} {
			a.auction.modal = modal
			for _, width := range []int{404, 1024} {
				a.sprites.Begin(width, 768)
				a.drawAuction(width, 768)
				a.sprites.End()
				if code := gl.GetError(); code != gl.NO_ERROR {
					t.Fatalf("width %d view %d modal %q GL error %#x", width, view, modal, code)
				}
			}
		}
	}
}
