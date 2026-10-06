// Package javasession is a Dragonfly session for Java Edition clients: it implements player.Session
// (and so world.Viewer) by writing Java packets, and turns the client's packets into calls on the
// player. Java players join through Run, which accepts clients from the java/server listener.
package javasession

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/df-mc/dragonfly/server/player/skin"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	jchunk "github.com/ezchr/go-mc/java/chunk"
	"github.com/ezchr/go-mc/java/server"
	v777 "github.com/ezchr/go-mc/java/v777"
	"github.com/ezchr/go-mc/java/wire"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
)

// Session is one Java Edition player's connection.
type Session struct {
	log  *slog.Logger
	jp   *server.Player
	id   uuid.UUID // the player's Dragonfly UUID (see Identity)
	xuid string
	skin skin.Skin
	peer *session.Peer // how Bedrock clients list this player
	conn *wire.Conn

	ent     *world.EntityHandle
	onClose func(*world.Tx, session.Controllable)

	// Java entity ids of the entities this client sees (it is selfEntityID itself).
	entMu        sync.Mutex
	entityIDs    map[*world.EntityHandle]int32
	tracks       map[int32]*track
	nextEntityID int32

	chunkRadius int32
	dim         string // the client's current Java dimension
	loader      *world.Loader

	vitalsMu sync.Mutex
	vitals   vitals

	input inputState

	timeMu      sync.Mutex
	time        int
	timeStopped bool
	lastCentre  world.ChunkPos
	centreSent  bool
	closeOnce   sync.Once

	// Chunk sending: the client says how many chunks per tick it can take (in thousandths);
	// one batch waits for its acknowledgement at a time.
	chunkRate     atomic.Int64
	batchInFlight atomic.Bool
	chunks        chunkState
	col           jchunk.Column
	hm            [1][]uint64

	// Teleports: moves from the client are ignored until it accepts the last one.
	teleportID      atomic.Int32
	pendingTeleport atomic.Int32

	latency   atomic.Int64 // nanoseconds
	keepAlive atomic.Int64 // id of the keep-alive we're waiting on, 0 if none

	outMu  sync.Mutex
	out    []outPacket
	wake   chan struct{}
	closed chan struct{}
	once   sync.Once

	writers sync.Pool
}

type outPacket struct {
	id int32
	w  *wire.Writer
}

func newSession(jp *server.Player, radius int32, log *slog.Logger) *Session {
	s := &Session{
		log:         log.With("player", jp.Profile.Name, "edition", "java"),
		jp:          jp,
		conn:        jp.Conn,
		chunkRadius: radius,
		wake:        make(chan struct{}, 1),
		closed:      make(chan struct{}),
		entityIDs:   map[*world.EntityHandle]int32{},
		tracks:      map[int32]*track{},
		vitals:      vitals{health: 20, food: 20, saturation: 5},
	}
	s.chunkRate.Store(9000) // vanilla's starting rate: 9 chunks per tick
	s.writers.New = func() any { return &wire.Writer{B: make([]byte, 0, 256)} }
	go s.writeLoop()
	return s
}

// packet returns a writer for a packet body; pass it to queue.
func (s *Session) packet() *wire.Writer {
	w := s.writers.Get().(*wire.Writer)
	w.Reset()
	return w
}

// queue sends a packet built with s.packet(). It never blocks on the network: viewer methods run
// on the world goroutine.
func (s *Session) queue(id int32, w *wire.Writer) {
	s.outMu.Lock()
	s.out = append(s.out, outPacket{id, w})
	s.outMu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// flushDelay is how long the writer waits after the first queued packet before writing: the world
// queues packets in bursts (a tick's movement of every visible entity), and one write per burst
// instead of one per packet saves most of the CPU (it was 70% syscalls at 30 players).
const flushDelay = 2 * time.Millisecond

func (s *Session) writeLoop() {
	var batch []outPacket
	for {
		select {
		case <-s.wake:
		case <-s.closed:
			return
		}
		time.Sleep(flushDelay)
		s.outMu.Lock()
		batch, s.out = s.out, batch[:0]
		s.outMu.Unlock()
		for _, p := range batch {
			err := s.conn.WritePacket(p.id, p.w.B)
			if cap(p.w.B) <= 1<<16 { // don't keep huge chunk buffers around
				s.writers.Put(p.w)
			}
			if err != nil {
				s.CloseConnection()
				return
			}
		}
		if err := s.conn.Flush(); err != nil {
			s.CloseConnection()
			return
		}
	}
}

// Addr ...
func (s *Session) Addr() net.Addr { return s.conn.NetConn().RemoteAddr() }

// Latency ...
func (s *Session) Latency() time.Duration { return time.Duration(s.latency.Load()) }

// ChunkRadius ...
func (s *Session) ChunkRadius() int32 { return s.chunkRadius }

// ClientData describes the Java client in the Bedrock form Dragonfly keeps.
func (s *Session) ClientData() login.ClientData {
	return login.ClientData{
		GameVersion:    server.GameVersion + " (Java)",
		LanguageCode:   s.jp.Info.Locale,
		DeviceModel:    "Java Edition",
		ThirdPartyName: s.jp.Profile.Name,
	}
}

// Disconnect sends the reason and closes the connection.
func (s *Session) Disconnect(message string) {
	w := s.packet()
	server.TextComponent(w, message)
	s.queue(v777.ClientboundPlayDisconnect, w)
	// Give the writer a moment to send it before the socket closes.
	time.AfterFunc(200*time.Millisecond, s.CloseConnection)
}

// CloseConnection closes the network connection; the read loop then removes the player.
func (s *Session) CloseConnection() {
	s.once.Do(func() {
		close(s.closed)
		s.conn.Close()
	})
}

// Close is called by the player when it is closed (in its world transaction, or with a nil tx if
// its world is gone). It hands the player to the server's close handler, which saves their data.
//
// Same order as the Bedrock session: save the player, close the chunk loader, and only then
// remove the player entity from the world (a player with a session is removed by the session).
func (s *Session) Close(tx *world.Tx, c session.Controllable) {
	s.closeOnce.Do(func() {
		if s.onClose != nil {
			s.onClose(tx, c)
		}
		if tx != nil {
			if s.loader != nil {
				s.loader.Close(tx)
			}
			tx.RemoveEntity(c)
			if s.ent != nil {
				_ = s.ent.Close()
			}
		}
		s.CloseConnection()
		forgetProfile(s.id)
		if s.peer != nil {
			session.RemovePeer(s.peer)
		}
		s.entMu.Lock()
		clear(s.entityIDs)
		s.entMu.Unlock()
	})
}

// Spawn starts the session once the player entity is in the world.
func (s *Session) Spawn(c session.Controllable, tx *world.Tx) {
	s.ent = c.H()
	s.SendHealth(c.Health(), c.MaxHealth(), c.Absorption())
	s.SendFood(c.Food(), 0, 0)
	s.SendExperience(c.ExperienceLevel(), c.ExperienceProgress())
	s.SendAbilities(c)
	pos := c.Position()
	s.loader = world.NewLoader(int(s.chunkRadius), tx.World(), s)
	s.loader.Move(tx, pos)
	s.sendCentre(pos)
	s.SpawnText(c)
	go s.tickLoop()
	go s.readLoop()
}

// withPlayer runs f in the player's world transaction.
func (s *Session) withPlayer(f func(tx *world.Tx, c session.Controllable)) error {
	_, err := world.CallRef(context.Background(), world.NewEntityRef[session.Controllable](s.ent), func(tx *world.Tx, c session.Controllable) (struct{}, error) {
		f(tx, c)
		return struct{}{}, nil
	})
	return err
}

func stopped(err error) bool {
	return errors.Is(err, world.ErrEntityClosed) || errors.Is(err, world.ErrWorldClosed) || errors.Is(err, world.ErrTaskCancelled)
}

// tickLoop moves the chunk loader with the player and sends chunks, 20 times a second.
func (s *Session) tickLoop() {
	t := time.NewTicker(time.Second / 20)
	defer t.Stop()
	ka := time.NewTicker(10 * time.Second)
	defer ka.Stop()
	for {
		select {
		case <-s.closed:
			return
		case <-ka.C:
			if s.keepAlive.Load() != 0 {
				s.log.Info("keep-alive timed out")
				s.Disconnect("Timed out")
				return
			}
			id := time.Now().UnixNano()
			s.keepAlive.Store(id)
			w := s.packet()
			w.Int64(id)
			s.queue(v777.ClientboundPlayKeepAlive, w)
		case <-t.C:
			err := s.withPlayer(func(tx *world.Tx, c session.Controllable) {
				pos := c.Position()
				s.loader.Move(tx, pos)
				s.sendCentre(pos)
				s.sendChunkBatch(tx)
				s.continueBreaking(c)
			})
			if err != nil {
				if !stopped(err) {
					s.log.Debug("tick", "err", err)
				}
				return
			}
		}
	}
}

// sendCentre tells the client which chunk it is in when that changes (it unloads chunks around it).
func (s *Session) sendCentre(pos mgl64.Vec3) {
	cp := world.ChunkPos{int32(math.Floor(pos[0])) >> 4, int32(math.Floor(pos[2])) >> 4}
	if s.centreSent && cp == s.lastCentre {
		return
	}
	s.lastCentre, s.centreSent = cp, true
	w := s.packet()
	w.VarInt(cp[0])
	w.VarInt(cp[1])
	s.queue(v777.ClientboundPlaySetChunkCacheCenter, w)
}

// readLoop handles the client's packets until the connection closes, then removes the player.
func (s *Session) readLoop() {
	defer func() {
		s.CloseConnection()
		// Closing the player calls s.Close, which saves the player through the server.
		err := s.withPlayer(func(tx *world.Tx, c session.Controllable) {
			if err := c.Close(); err != nil {
				s.log.Debug("close player", "err", err)
			}
		})
		if err != nil && !stopped(err) {
			s.log.Error("player not closed: data not saved", "err", err)
		}
		s.log.Info("left")
	}()
	for {
		id, body, err := s.conn.ReadPacket()
		if err != nil {
			s.log.Debug("read", "err", err)
			return
		}
		if err := s.handle(id, body); err != nil {
			s.log.Info("bad packet", "id", id, "err", err)
			s.Disconnect("Bad packet")
			return
		}
	}
}
