package javasession

import (
	"time"

	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	v777 "github.com/ezchr/go-mc/java/v777"
	"github.com/ezchr/go-mc/java/wire"
	"github.com/go-gl/mathgl/mgl64"
)

// handle handles one packet from the client.
func (s *Session) handle(id int32, body []byte) error {
	r := wire.NewReader(body)
	switch id {
	case v777.ServerboundPlayKeepAlive:
		ka := r.Int64()
		if ka == s.keepAlive.Load() {
			s.latency.Store(int64(time.Duration(time.Now().UnixNano()-ka) / 2))
			s.keepAlive.Store(0)
		}
	case v777.ServerboundPlayAcceptTeleportation:
		tp := r.VarInt()
		s.pendingTeleport.CompareAndSwap(tp, 0)
	case v777.ServerboundPlayMovePlayerPos:
		x, y, z := r.Float64(), r.Float64(), r.Float64()
		flags := r.Byte()
		if r.Err == nil {
			s.move(&mgl64.Vec3{x, y, z}, nil, flags)
		}
	case v777.ServerboundPlayMovePlayerPosRot:
		x, y, z := r.Float64(), r.Float64(), r.Float64()
		yaw, pitch := r.Float32(), r.Float32()
		flags := r.Byte()
		if r.Err == nil {
			s.move(&mgl64.Vec3{x, y, z}, &[2]float32{yaw, pitch}, flags)
		}
	case v777.ServerboundPlayMovePlayerRot:
		yaw, pitch := r.Float32(), r.Float32()
		flags := r.Byte()
		if r.Err == nil {
			s.move(nil, &[2]float32{yaw, pitch}, flags)
		}
	case v777.ServerboundPlayChunkBatchReceived:
		rate := float64(r.Float32())
		if rate != rate || rate < 0.01 { // NaN or nonsense: vanilla clamps the same way
			rate = 0.01
		}
		s.chunkRate.Store(int64(min(rate, 64) * 1000))
		s.batchInFlight.Store(false)
	default:
		if ok, err := s.handleInput(id, body); ok {
			return err
		}
	}
	return r.Err
}

// move applies a client move. Positions are feet positions on both editions.
func (s *Session) move(pos *mgl64.Vec3, rot *[2]float32, flags byte) {
	if s.pendingTeleport.Load() != 0 {
		return // the client hasn't caught up with our last teleport yet
	}
	err := s.withPlayer(func(tx *world.Tx, c session.Controllable) {
		var delta mgl64.Vec3
		if pos != nil {
			delta = pos.Sub(c.Position())
		}
		var dyaw, dpitch float64
		if rot != nil {
			r := c.Rotation()
			dyaw, dpitch = float64(rot[0])-r.Yaw(), float64(rot[1])-r.Pitch()
		}
		c.Move(delta, dyaw, dpitch)
	})
	if err != nil && !stopped(err) {
		s.log.Debug("move", "err", err)
	}
}
