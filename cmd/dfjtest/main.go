// dfjtest is a plain Dragonfly server with a superflat world that Java Edition clients join
// natively through javasession. For testing the Java session against real clients.
package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/biome"
	"github.com/df-mc/dragonfly/server/world/generator"
	"github.com/ezchr/dfjava/javasession"
	jserver "github.com/ezchr/go-mc/java/server"
)

func main() {
	javaAddr := flag.String("java", "127.0.0.1:25620", "Java listen address")
	bedrockAddr := flag.String("bedrock", "127.0.0.1:19160", "Bedrock listen address")
	folder := flag.String("world", "dfjtest-world", "world folder")
	radius := flag.Int("radius", 6, "chunk radius")
	survival := flag.Bool("survival", true, "new players start in survival (Dragonfly defaults to creative)")
	flag.Parse()
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	uc := server.DefaultConfig()
	uc.Network.Address = *bedrockAddr
	uc.World.Folder = *folder
	uc.Players.SaveData = false
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
	}
}
