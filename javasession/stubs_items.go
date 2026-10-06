package javasession

// Not implemented yet (items area): these do nothing. Generated with tools/stubgen from
// player.Session; delete a method here when it gets a real implementation.

import (
	"time"

	"github.com/df-mc/dragonfly/server/session"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// Methods the Java session does not implement yet: they do nothing. Generated with
// tools/stubgen from player.Session; delete a line when the method gets a real implementation.

func (*Session) OpenBlockContainer(_ cube.Pos, _ *world.Tx) {}
func (*Session) OpenTrade(_ *world.Tx, _ world.Entity, _ string, _ []session.TradeOffer, _ func(index int, times int) bool) {
}
func (*Session) SendChargeItemComplete()                  {}
func (*Session) UpdateTradeOffers(_ []session.TradeOffer) {}
func (*Session) ViewBrewingUpdate(_ time.Duration, _ time.Duration, _ int32, _ int32, _ int32, _ int32) {
}
func (*Session) ViewFurnaceUpdate(_ time.Duration, _ time.Duration, _ time.Duration, _ time.Duration, _ time.Duration, _ time.Duration) {
}
