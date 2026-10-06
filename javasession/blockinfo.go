package javasession

import (
	"sync"

	"github.com/df-mc/dragonfly/server/world"
)

// buildBlockInfo builds the per-runtime-id tables chunk encoding uses, from the default block
// registry (the one Dragonfly worlds use unless configured otherwise).
func buildBlockInfo() blockInfo {
	reg := world.DefaultBlockRegistry
	n := reg.BlockCount()
	bi := blockInfo{java: make([]uint32, n), air: make([]bool, n), fluid: make([]bool, n)}
	java := javaStates(reg)
	for rid := 0; rid < n; rid++ {
		name, _, _ := reg.RuntimeIDToState(uint32(rid))
		bi.air[rid] = name == "minecraft:air" || name == "minecraft:cave_air" || name == "minecraft:void_air" ||
			name == "minecraft:structure_void" || name == "minecraft:light_block"
		bi.fluid[rid] = reg.LiquidBlock(uint32(rid))
		bi.java[rid] = java[rid]
	}
	return bi
}

func airRID() uint32 { return world.DefaultBlockRegistry.AirRuntimeID() }

// javaStates is the Java block state id for every runtime id.
// TEMPORARY until javamap lands: air is air (0), everything else stone (1).
func javaStates(reg world.BlockRegistry) []uint32 {
	n := reg.BlockCount()
	out := make([]uint32, n)
	for rid := range out {
		if name, _, _ := reg.RuntimeIDToState(uint32(rid)); name != "minecraft:air" {
			out[rid] = 1
		}
	}
	return out
}

var (
	biomesOnce sync.Once
	biomes     map[uint32]uint32
)

// biomeTable maps Dragonfly (Bedrock) biome ids to Java biome registry ids.
// TEMPORARY until javamap lands: empty, so everything is plains.
func biomeTable() map[uint32]uint32 {
	biomesOnce.Do(func() { biomes = map[uint32]uint32{} })
	return biomes
}
