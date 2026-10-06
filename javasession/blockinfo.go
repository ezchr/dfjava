package javasession

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/ezchr/dfjava/javamap"
)

// buildBlockInfo builds the per-runtime-id tables chunk encoding uses, from the default block
// registry (the one Dragonfly worlds use unless configured otherwise).
func buildBlockInfo() blockInfo {
	reg := world.DefaultBlockRegistry
	java := javamap.DefaultBlockStates()
	n := len(java)
	bi := blockInfo{java: make([]uint32, n), air: make([]bool, n), fluid: make([]bool, n), waterlogged: make([]uint32, n)}
	for rid := 0; rid < n; rid++ {
		name, _, _ := reg.RuntimeIDToState(uint32(rid))
		bi.air[rid] = name == "minecraft:air" || name == "minecraft:cave_air" || name == "minecraft:void_air" ||
			name == "minecraft:structure_void" || name == "minecraft:light_block"
		bi.fluid[rid] = reg.LiquidBlock(uint32(rid))
		bi.java[rid] = uint32(java[rid])
		// The state to use when Bedrock keeps water in the block's second layer.
		bi.waterlogged[rid] = bi.java[rid]
		if javamap.Waterloggable(java[rid]) {
			bi.waterlogged[rid] = uint32(javamap.Waterlogged(java[rid]))
		}
	}
	return bi
}

func airRID() uint32 { return world.DefaultBlockRegistry.AirRuntimeID() }

// biomeID is the Java registry id of a Dragonfly (Bedrock) biome id.
func biomeID(b uint32) uint32 { return uint32(javamap.BiomeID(int(b))) }
