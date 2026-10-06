// dfjtest is a plain Dragonfly server with a superflat world that Java Edition clients join
// natively through javasession. For testing the Java session against real clients.
package main

import (
	"flag"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/biome"
	"github.com/df-mc/dragonfly/server/world/generator"
	"github.com/ezchr/dfjava/javasession"
	jserver "github.com/ezchr/go-mc/java/server"
	"github.com/go-gl/mathgl/mgl64"
)

func main() {
	javaAddr := flag.String("java", "127.0.0.1:25620", "Java listen address")
	bedrockAddr := flag.String("bedrock", "127.0.0.1:19160", "Bedrock listen address")
	folder := flag.String("world", "dfjtest-world", "world folder")
	radius := flag.Int("radius", 6, "chunk radius")
	pprofAddr := flag.String("pprof", "", "serve net/http/pprof on this address (for profiling)")
	spawnTest := flag.Bool("spawntest", false, "spawn test entities (TNT, falling sand, xp orbs, an item) near each joining player")
	netherTest := flag.Bool("nethertest", false, "move each joining player to the nether after 10 s")
	noAuth := flag.Bool("noauth", false, "let Bedrock clients join without Xbox authentication (for test bots)")
	survival := flag.Bool("survival", true, "new players start in survival (Dragonfly defaults to creative)")
	flag.Parse()
	if *pprofAddr != "" {
		go http.ListenAndServe(*pprofAddr, nil)
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	uc := server.DefaultConfig()
	uc.Network.Address = *bedrockAddr
	uc.World.Folder = *folder
	uc.Players.SaveData = false
	uc.Server.AuthEnabled = !*noAuth
	conf, err := uc.Config(log)
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}
	conf.Generator = func(world.Dimension) world.Generator {
		return generator.NewFlat(biome.Plains{}, []world.Block{block.Grass{}, block.Dirt{}, block.Dirt{}, block.Bedrock{}})
	}
	srv := conf.New()
	if *survival {
		srv.World().SetDefaultGameMode(world.GameModeSurvival)
	}
	srv.CloseOnProgramEnd()
	srv.Listen()

	jl, err := jserver.Listen(*javaAddr, jserver.Config{
		CompressionThreshold: 256,
		Brand:                "dragonfly",
		Log:                  log,
		Status: func() jserver.Status {
			return jserver.Status{MOTD: "Dragonfly (native Java)", MaxPlayers: 100, Online: srv.PlayerCount()}
		},
	})
	if err != nil {
		log.Error("java listen", "err", err)
		os.Exit(1)
	}
	log.Info("java listening", "addr", jl.Addr())
	go javasession.Run(javasession.Config{Server: srv, Listener: jl, ChunkRadius: *radius, Log: log})

	for p := range srv.Accept() {
		log.Info("player in world", "name", p.Name(), "pos", p.Position())
		if *netherTest {
			h := p.H()
			go func() {
				time.Sleep(10 * time.Second)
				srv.World().Do(func(tx *world.Tx) {
					e, ok := h.Entity(tx)
					if !ok {
						return
					}
					handle := tx.RemoveEntity(e)
					srv.Nether().Do(func(tx *world.Tx) {
						tx.AddEntityAt(handle, mgl64.Vec3{0.5, 10, 0.5})
						log.Info("moved to the nether", "name", p.Name())
					})
				})
			}()
		}
		if *spawnTest {
			tx := p.Tx()
			at := p.Position().Add(mgl64.Vec3{3, 2, 0})
			tx.AddEntity(entity.NewText("§l§6Top Kills§r\n§a1. Steve §f- §e42", at.Add(mgl64.Vec3{0, 1, -6})))
			tx.AddEntity(entity.NewFallingBlock(world.EntitySpawnOpts{Position: at}, block.Sand{}))
			tx.AddEntity(entity.NewTNT(world.EntitySpawnOpts{Position: at.Add(mgl64.Vec3{0, 0, 3})}, 8*time.Second))
			tx.AddEntity(entity.NewExperienceOrb(world.EntitySpawnOpts{Position: at.Add(mgl64.Vec3{0, 0, -3})}, 5))
			tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: at.Add(mgl64.Vec3{-6, 0, 0})}, item.NewStack(item.Diamond{}, 3)))
		}
	}
}
