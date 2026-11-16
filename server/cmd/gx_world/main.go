// Command gx_world is the Shaiya 5.4 game server on the original
// EP4.5 tables translated to SQLite: character select, world simulation (maps,
// NPCs, mobs, combat, skills), items, trade and the item mall.
package main

import (
	"context"
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/transport"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
	"github.com/Spelunkern/Shaiya-GX/server/internal/netx"
	"github.com/Spelunkern/Shaiya-GX/server/internal/world"
)

const maxConnections = 500

var nextConnection atomic.Int64

func main() {
	if err := config.LoadRuntimeEnv(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	logger, err := logx.New("gx_world")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	logger.Printf("Starting GX Server World")
	if err := db.Open(db.UserData, db.GameData, db.GameDefs); err != nil {
		logger.Printf("GX Server World database error: %v", err)
		os.Exit(1)
	}
	defs, err := db.Definitions()
	if err != nil {
		logger.Printf("GX Server World failed to load PS_GameDefs: %v", err)
		os.Exit(1)
	}
	logger.Printf("GX Server World loaded definitions items=%d skills=%d mobs=%d products=%d status=%d base_items=%d",
		len(defs.Items), len(defs.Skills), len(defs.Mobs), len(defs.Products), len(defs.Status), len(defs.BaseItems))
	if err := world.LoadGameConfig(logger); err != nil {
		logger.Printf("GX Server World failed to load data/game (copy the EP4.5 PS_Game/Data files there): %v", err)
		os.Exit(1)
	}
	world.InitWorldCalendar(logger)
	world.InitKillCount(logger)
	if err := world.LoadGuilds(logger); err != nil {
		logger.Printf("GX Server World failed to load guilds: %v", err)
	}
	if err := world.LoadSvmaps(logger); err != nil {
		logger.Printf("GX Server World failed to load maps: %v", err)
		os.Exit(1)
	}
	if err := world.LoadRelics(logger); err != nil {
		logger.Printf("GX Server World failed to load obelisks: %v", err)
		os.Exit(1)
	}
	world.StartSweeps(logger)

	listener, err := transport.Listen(config.WorldListen())
	if err != nil {
		logger.Printf("GX Server World failed to bind %s: %v", config.WorldListen(), err)
		os.Exit(1)
	}
	logger.Printf("GX Server World listening on %s", config.WorldListen())
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	defer logger.Close()
	defer db.Close()
	if err := netx.Serve(ctx, listener, maxConnections, 32, func(conn net.Conn) {
		log := logger.Tagged(fmt.Sprintf("conn%d", nextConnection.Add(1)))
		defer func() {
			if r := recover(); r != nil {
				log.Printf("world session panic: %v", r)
			}
		}()
		if err := world.Serve(conn, logger, log); err != nil {
			log.Printf("world client error: %v", err)
		}
	}); err != nil {
		logger.Printf("server accept error: %v", err)
	}

}
