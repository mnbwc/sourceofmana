package main

// Phase-2: AOI / movement state sync.
//
// Server-authoritative lightweight world: each in-game peer owns one Entity
// with position/velocity/orientation. A 10 Hz tick advances movement and
// broadcasts UpdateEntity / FullUpdateEntity to peers within AOI radius,
// mirroring SoM's World.gd flow (WarpPlayer -> PreloadPlayer exchange ->
// FullUpdateEntity self -> AddOnlinePlayer broadcast).

import (
	"log"
	"math"
	"sync"
	"time"
)

// nameIDs above index 35 (post-Emote numbering, see rpc_ids_119.txt).
const (
	rpcFullUpdateEntity = 39
	rpcUpdateEntity     = 112
)

const (
	walkSpeed = 120.0  // px/s, SoM walk speed ballpark
	aoiRadius = 1000.0 // px, interest radius
	tickEvery = 100 * time.Millisecond
	spawnMap  = 0
)

// Vec2 is a small server-side vector (kept separate from Variant helpers).
type Vec2 struct {
	X, Y float32
}

func (v Vec2) add(o Vec2) Vec2     { return Vec2{v.X + o.X, v.Y + o.Y} }
func (v Vec2) sub(o Vec2) Vec2     { return Vec2{v.X - o.X, v.Y - o.Y} }
func (v Vec2) len() float32        { return float32(math.Hypot(float64(v.X), float64(v.Y))) }
func (v Vec2) dist(o Vec2) float32 { return v.sub(o).len() }
func (v Vec2) norm() Vec2 {
	l := v.len()
	if l == 0 {
		return Vec2{}
	}
	return Vec2{v.X / l, v.Y / l}
}
func (v Vec2) scaled(f float32) Vec2 { return Vec2{v.X * f, v.Y * f} }

// Entity is the server-side view of one in-game character.
type Entity struct {
	CharID  int64
	Name    string
	PeerID  int
	MapID   int
	Pos     Vec2
	Vel     Vec2
	Orient  Vec2
	Target  *Vec2 // SetClickPos destination (nil = no active walk target)
	State   int64
	Running bool
	// Combat stats (phase-3). Health is authoritative on the server.
	Health    int64
	MaxHealth int64
	Level     int64
	// NpcRID is the RID of the NPC this entity represents (0 for players).
	// Used by Phase-8 bestiary tracking for spawned NPCs.
	NpcRID int64
	// DeadAt records when the entity entered the DEATH state, so field
	// monsters can be respawned after a delay. Zero while alive.
	DeadAt time.Time
}

// world holds all in-game entities keyed by charID.
type world struct {
	mu       sync.Mutex
	entities map[int64]*Entity
}

func newWorld() *world {
	return &world{entities: make(map[int64]*Entity)}
}

func (w *world) add(e *Entity) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.entities[e.CharID] = e
}

func (w *world) remove(charID int64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.entities, charID)
}

func (w *world) get(charID int64) *Entity {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.entities[charID]
}

// byName returns the first entity with the given name (players are unique by
// name in SoM). Used by party invite/kick/promote which resolve by nickname.
func (w *world) byName(name string) *Entity {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, e := range w.entities {
		if e.Name == name {
			return e
		}
	}
	return nil
}

func (w *world) snapshot() []*Entity {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]*Entity, 0, len(w.entities))
	for _, e := range w.entities {
		out = append(out, e)
	}
	return out
}

// enterWorld runs the SoM connect sequence: WarpPlayer to the spawn, spawn
// self via FullUpdateEntity, exchange PreloadPlayer with everyone in AOI,
// and broadcast the online-list update.
func (s *Server) enterWorld(c *Client, chr *Character) {
	pos := Vec2{100 + float32(chr.ID%8)*60, 100 + float32(chr.ID%5)*60}
	e := &Entity{
		CharID:    chr.ID,
		Name:      chr.Name,
		PeerID:    c.peerID,
		MapID:     spawnMap,
		Pos:       pos,
		State:     0, // ActorCommons.State.IDLE
		Health:    defaultHealth,
		MaxHealth: defaultHealth,
		Level:     0, // SoM Stats.level starts at 0 (0-based)
	}
	s.world.add(e)

	// 1. WarpPlayer(mapID, playerPos) -> the entering player.
	sendToClient(c, nil, rpcWarpPlayer, []Variant{NewInt(int64(e.MapID)), NewVec2(e.Pos.X, e.Pos.Y)})

	// 2. Spawn self: PreloadPlayer(agentRID, name, ...) so the client's
	//    entityCache has an entry, then FullUpdateEntity to position it.
	//    (SoM World.gd AgentWarped pushes PreloadPlayer before FullUpdateEntity;
	//    Map.gd FullUpdateEntity only spawns when entityCache has the RID.)
	sendToClient(c, nil, rpcPreloadPlayer, s.preloadPlayerArgs(e))
	sendToClient(c, nil, rpcFullUpdateEntity, fullUpdateArgs(e))

	// 3. PreloadPlayer exchange + known positions with everyone else.
	for _, other := range s.world.snapshot() {
		if other.CharID == e.CharID {
			continue
		}
		if oc := s.clientByPeer(other.PeerID); oc != nil {
			// Tell existing clients about the newcomer: PreloadPlayer + its
			// current position (UpdateEntity). Without UpdateEntity the peer
			// knows the entity exists but has no position to walk to.
			sendToClient(oc, nil, rpcPreloadPlayer, s.preloadPlayerArgs(e)) // tell them about us
			sendToClient(oc, nil, rpcUpdateEntity, updateEntityArgs(e))
		}
		sendToClient(c, nil, rpcPreloadPlayer, s.preloadPlayerArgs(other)) // tell us about them
		sendToClient(c, nil, rpcUpdateEntity, updateEntityArgs(other))
	}

	// 3b. Preload NPCs in AOI (World.gd AgentWarped -> BulkPreload(npcs)).
	for _, npc := range s.npcs.allNpcs() {
		if npc.MapID != e.MapID || e.Pos.dist(npc.Pos) > aoiRadius {
			continue
		}
		sendToClient(c, nil, rpcPreloadEntity, npcPublicStatsArgs(npc))
	}

	// 3c. Push quest/bestiary/skill progress (Progress.ImportProgress + RefreshProgress).
	s.pushRefreshProgress(c, chr.ID)

	// 3c2. Push inventory + equipment (SQL.GetStorage + SQL.GetEquipment).
	sendToClient(c, nil, rpcRefreshInventory, []Variant{inventoryArray(chr)})
	sendToClient(c, nil, rpcRefreshEquipment, []Variant{NewInt(chr.ID), equipmentDict(chr)})

	// 3d. Push the player's public stats (health/level etc.) to self and peers.
	s.sendPublicStats(e)

	// 3d2. Push private stats (experience/gp/mana/stamina/karma/weight) to self.
	s.pushPrivateStats(chr.ID)

	// 3d3. Starter drops: 3 Potions (1001) near the spawn so the item flow
	// has ground items to pick up (mirrors SoM's starter potions from the
	// Elder quest, which arrive as DropAdded pushes).
	for i := 0; i < 3; i++ {
		dp := Vec2{e.Pos.X + float32(16+i*12), e.Pos.Y + float32(8)}
		s.spawnDrop(chr.ID, 1001, "", dp, e.MapID)
	}

	// 4. Online list: RefreshOnlineList(names) to the newcomer, AddOnlinePlayer broadcast.
	names := make([]Variant, 0, len(s.world.snapshot()))
	for _, other := range s.world.snapshot() {
		names = append(names, NewString(other.Name))
	}
	sendToClient(c, nil, rpcRefreshOnlineList, []Variant{{Type: VariantPackedStringArray, Arr: names}})
	for _, other := range s.world.snapshot() {
		if other.CharID == e.CharID {
			continue
		}
		if oc := s.clientByPeer(other.PeerID); oc != nil {
			sendToClient(oc, nil, rpcAddOnlinePlayer, []Variant{NewString(e.Name)})
		}
	}
	log.Printf("world: %s (char %d, peer %d) entered at (%.0f,%.0f), %d players online",
		e.Name, e.CharID, e.PeerID, e.Pos.X, e.Pos.Y, len(s.world.snapshot()))
}

// leaveWorld removes the peer's entity and notifies the others.
func (s *Server) leaveWorld(c *Client) {
	if c.session == nil {
		return
	}
	c.session.mu.Lock()
	inGame := c.session.inGame
	charID := c.session.charID
	c.session.mu.Unlock()
	if !inGame || charID == peerUnknownID {
		return
	}
	e := s.world.get(charID)
	s.world.remove(charID)
	c.session.mu.Lock()
	c.session.inGame = false
	c.session.mu.Unlock()
	if e == nil {
		return
	}
	for _, other := range s.world.snapshot() {
		if oc := s.clientByPeer(other.PeerID); oc != nil {
			sendToClient(oc, nil, rpcRemoveOnlinePlayer, []Variant{NewString(e.Name)})
			// Phase-6: RemoveEntity tells the client to despawn the actor node
			// (mirrors Network.RemoveEntity -> Map.RemoveEntity).
			sendToClient(oc, nil, rpcRemoveEntity, []Variant{NewInt(e.CharID)})
		}
	}
	// Party cleanup: drop any pending invite to this player and remove them
	// from their party (so a reconnecting player does not keep a stale slot).
	s.party.mu.Lock()
	delete(s.party.invites, e.CharID)
	pid := s.party.leave(e.CharID)
	s.party.mu.Unlock()
	if pid != 0 {
		s.notifyPartyMembers(pid)
	}
	// Trade cleanup: drop any pending request to this player and close any
	// live session they were in.
	s.trade.mu.Lock()
	delete(s.trade.requests, e.CharID)
	delete(s.trade.reqTick, e.CharID)
	tradeSession := s.trade.sessions[e.CharID]
	if tradeSession != nil {
		s.detachSessionLocked(tradeSession)
	}
	s.trade.mu.Unlock()
	if tradeSession != nil {
		s.sendTradeClosed(tradeSession, e.Name+" left the world.")
	}
	log.Printf("world: %s (char %d) left, %d players online", e.Name, e.CharID, len(s.world.snapshot()))
}

// clientByPeer returns the live Client for a peer ID, if connected.
func (s *Server) clientByPeer(peerID int) *Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clients[peerID]
}

func fullUpdateArgs(e *Entity) []Variant {
	return []Variant{
		NewInt(e.CharID),                // agentRID
		NewVec2(e.Vel.X, e.Vel.Y),       // velocity
		NewVec2(e.Pos.X, e.Pos.Y),       // position
		NewVec2(e.Orient.X, e.Orient.Y), // orientation
		NewInt(e.State),                 // agentState
		NewInt(0),                       // skillCastID
		NewBool(e.Running),              // isRunning
		NewInt(time.Now().UnixMilli()),  // frameID
	}
}

func updateEntityArgs(e *Entity) []Variant {
	return []Variant{
		NewInt(e.CharID),
		NewVec2(e.Vel.X, e.Vel.Y),
		NewVec2(e.Pos.X, e.Pos.Y),
		NewInt(time.Now().UnixMilli()),
	}
}

// preloadPlayerArgs mirrors World.gd BulkPreload for a PlayerAgent.
// Sources level/health/race/equipment from the Character record when available.
func (s *Server) preloadPlayerArgs(e *Entity) []Variant {
	level := e.Level
	health := e.Health
	race := int64(1)
	hairstyle := int64(0)
	haircolor := int64(0)
	gender := int64(0)
	skintone := int64(0)
	equipVariant := NewDict(map[string]Variant{})
	// Look up the character record for richer data.
	if e.CharID > 0 {
		if chr := s.store.findCharacterByID(e.CharID); chr != nil {
			level = chr.Level
			health = chr.Health
			race = chr.Race
			hairstyle = chr.Hairstyle
			haircolor = chr.Haircolor
			gender = chr.Gender
			skintone = chr.Skintone
			equipVariant = equipmentDict(chr)
		}
	}
	return []Variant{
		NewInt(e.CharID),  // agentRID
		NewInt(0),         // spirit
		NewInt(0),         // currentShape
		NewString(e.Name), // nick
		NewInt(level),     // level
		NewInt(health),    // health
		NewInt(hairstyle), // hairstyle
		NewInt(haircolor), // haircolor
		NewInt(gender),    // gender
		NewInt(race),      // race
		NewInt(skintone),  // skintone
		equipVariant,      // equipment
	}
}

// worldTick advances movement and broadcasts state changes at 10 Hz.
func (s *Server) worldTick() {
	t := time.NewTicker(tickEvery)
	defer t.Stop()
	for range t.C {
		s.tickOnce()
	}
}

func (s *Server) tickOnce() {
	const dt = float32(tickEvery) / float32(time.Second)
	entities := s.world.snapshot()
	moved := make(map[*Entity]bool, len(entities))
	for _, e := range entities {
		if e.Target != nil {
			d := e.Target.sub(e.Pos)
			if d.len() <= walkSpeed*dt {
				e.Pos = *e.Target
				e.Target = nil
				e.Vel = Vec2{}
			} else {
				step := d.norm().scaled(walkSpeed * dt)
				e.Pos = e.Pos.add(step)
				e.Vel = d.norm().scaled(walkSpeed)
			}
			moved[e] = true
		} else if e.Vel.X != 0 || e.Vel.Y != 0 {
			e.Pos = e.Pos.add(e.Vel.scaled(dt))
			moved[e] = true
		}
	}
	if len(moved) == 0 {
		s.expireDrops()
		return
	}
	s.expireDrops()
	// Broadcast updates to AOI neighbours (not to the mover itself: it is
	// locally predicted, matching SoM's client-authoritative movement flow).
	for e := range moved {
		for _, other := range entities {
			if other == e {
				continue
			}
			if other.MapID != e.MapID || e.Pos.dist(other.Pos) > aoiRadius {
				continue
			}
			if oc := s.clientByPeer(other.PeerID); oc != nil {
				sendToClient(oc, nil, rpcUpdateEntity, updateEntityArgs(e))
			}
		}
	}
}

// ---- movement RPC handlers (Server.gd "Navigation" section) ----

func (s *Server) handleSetMovePos(c *Client, rpc *ParsedRPC) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	dir := argVec2(rpc.Args, 0)
	if dir.X != 0 || dir.Y != 0 {
		e.Vel = dir.norm().scaled(walkSpeed) // relative mode: keep walking this way
		e.Target = nil
	} else {
		e.Vel = Vec2{}
	}
	log.Printf("world: %s SetMovePos (%.1f,%.1f) -> vel=(%.1f,%.1f)", e.Name, dir.X, dir.Y, e.Vel.X, e.Vel.Y)
}

func (s *Server) handleSetClickPos(c *Client, rpc *ParsedRPC) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	pos := argVec2(rpc.Args, 0)
	t := pos
	e.Target = &t
	log.Printf("world: %s SetClickPos (%.1f,%.1f)", e.Name, pos.X, pos.Y)
}

func (s *Server) handleSetOrientation(c *Client, rpc *ParsedRPC) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	e.Orient = argVec2(rpc.Args, 0)
	// Orientation changes matter to AOI neighbours immediately.
	for _, other := range s.world.snapshot() {
		if other.CharID == e.CharID || other.MapID != e.MapID || e.Pos.dist(other.Pos) > aoiRadius {
			continue
		}
		if oc := s.clientByPeer(other.PeerID); oc != nil {
			sendToClient(oc, nil, rpcFullUpdateEntity, fullUpdateArgs(e))
		}
	}
}

func argVec2(args []Variant, i int) Vec2 {
	if i < len(args) && args[i].Type == VariantVector2 {
		return Vec2{args[i].Vec2[0], args[i].Vec2[1]}
	}
	return Vec2{}
}
