package main

// AgentBridge — TCP NDJSON interface on port 9101.
//
// Mirrors SoM's sources/network/AgentBridge.gd: an AI-arena probe that speaks
// newline-delimited JSON (one request per line, one JSON line per reply) over
// a raw TCP socket, separate from the game's WebSocket SceneMultiplayer
// channel. The bridge is read-only with respect to the game world except for
// the explicit apply_action ops, which dispatch to the same Server methods a
// human client would use (SetClickPos, TriggerSkill, TriggerChat, ...).
//
// Design notes:
//   - The bridge owns its own listener goroutine; it never touches the
//     WebSocket server's goroutines or locks beyond the Server's own mutexes.
//   - Each TCP client is one goroutine reading NDJSON lines and one shared
//     write lock per connection, so replies are never interleaved.
//   - Event buffers are per in-game peerID (not per TCP client), matching
//     SoM's _events dictionary. drain_events clears the buffer for that peer.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Bridge protocol constants (must match SoM's NetworkCommons).
const (
	bridgeProtocolVersion uint64 = 4156244460
	bridgeMaxClients             = 16
	bridgeEventBufferMax         = 512
	// nearbyRadius is the AOI radius for snapshot's nearby list. SoM's
	// _nearby uses 480 by default, but the migration spec asks for ~250px.
	bridgeNearbyRadius float32 = 250.0
	// yWeight mirrors SoM's visual distance reading (y-axis weighted 1.42).
	bridgeYWeight float64 = 1.42
	// bridgeDefaultPort is the SoM AgentBridge listen port.
	bridgeDefaultPort = 9101
)

// bridgeEvent is one entry in a peer's event buffer.
type bridgeEvent struct {
	Seq   int64       `json:"seq"`
	Kind  string      `json:"kind"` // arena, kill, damaged, died, looted, chat, party, trade, npc
	Data  interface{} `json:"data,omitempty"`
	Frame int64       `json:"frame"`
}

// bridgeVerdict is the latest action verdict for a peer.
type bridgeVerdict struct {
	peerID      int
	action      string
	verdict     string // APPLIED, PENDING, REJECTED, NO_EFFECT
	reason      string
	provisional bool
	basedOn     int64
	// snapshots for state-delta verdicts
	beforePos     Vec2
	beforeHealth  int64
	targetRID     int64
	startedAt     time.Time
	pendingExists bool
}

// Bridge wraps a Server with the AgentBridge TCP interface.
type Bridge struct {
	srv      *Server
	mu       sync.Mutex
	listener net.Listener
	clients  map[net.Conn]struct{}
	// events maps in-game peerID -> buffered events (cleared by drain_events).
	events map[int][]bridgeEvent
	// eventSeq is the global monotonically increasing event sequence.
	eventSeq int64
	// verdicts maps peerID -> latest verdict.
	verdicts map[int]*bridgeVerdict
	// spectators maps target peerID -> spectator count.
	spectators map[int]int
	// frame is the bridge's own tick counter (incremented every 100ms).
	frame int64
	// arena is the arena episode service (episode lifecycle + scoring).
	arena *arenaService
	// done is closed on Stop.
	done chan struct{}
	once sync.Once
}

// StartBridge starts the AgentBridge TCP listener on addr (e.g. ":9101").
// The listener runs in its own goroutine; StartBridge returns once the
// socket is bound (or fails). Use Bridge.Stop to shut it down.
func StartBridge(s *Server, addr string) (*Bridge, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("bridge listen %s: %w", addr, err)
	}
	b := &Bridge{
		srv:        s,
		listener:   ln,
		clients:    make(map[net.Conn]struct{}),
		events:     make(map[int][]bridgeEvent),
		verdicts:   make(map[int]*bridgeVerdict),
		spectators: make(map[int]int),
		done:       make(chan struct{}),
	}
	b.arena = newArenaService(b)
	s.bridge = b
	go b.frameLoop()
	go b.arena.runTicker(b.done)
	go b.acceptLoop()
	log.Printf("AgentBridge: listening on %s", ln.Addr())
	return b, nil
}

// Stop shuts the bridge listener down and closes all client connections.
func (b *Bridge) Stop() {
	b.once.Do(func() {
		close(b.done)
		if b.listener != nil {
			b.listener.Close()
		}
		b.arena.shutdown()
		b.mu.Lock()
		for c := range b.clients {
			c.Close()
		}
		b.clients = make(map[net.Conn]struct{})
		b.mu.Unlock()
	})
}

// frameLoop increments the bridge frame counter every 100ms, matching SoM's
// NetworkCommons.FrameID() cadence (the game server ticks at 10 Hz).
func (b *Bridge) frameLoop() {
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			b.mu.Lock()
			b.frame++
			b.mu.Unlock()
		case <-b.done:
			return
		}
	}
}

// currentFrame returns the bridge frame count.
func (b *Bridge) currentFrame() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.frame
}

// acceptLoop accepts TCP connections until the listener is closed.
func (b *Bridge) acceptLoop() {
	for {
		conn, err := b.listener.Accept()
		if err != nil {
			select {
			case <-b.done:
				return
			default:
			}
			// Transient accept errors: back off briefly and retry.
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			log.Printf("AgentBridge: accept: %v", err)
			return
		}
		b.mu.Lock()
		if len(b.clients) >= bridgeMaxClients {
			b.mu.Unlock()
			log.Printf("AgentBridge: max clients reached, rejecting %s", conn.RemoteAddr())
			conn.Close()
			continue
		}
		b.clients[conn] = struct{}{}
		b.mu.Unlock()
		log.Printf("AgentBridge: client connected: %s", conn.RemoteAddr())
		go b.serveConn(conn)
	}
}

// serveConn reads NDJSON lines from one TCP client until disconnect.
func (b *Bridge) serveConn(conn net.Conn) {
	defer func() {
		b.mu.Lock()
		delete(b.clients, conn)
		b.mu.Unlock()
		conn.Close()
		log.Printf("AgentBridge: client disconnected: %s", conn.RemoteAddr())
	}()

	// Buffered reader so we can read line-by-line without a scanner limit.
	rd := bufio.NewReaderSize(conn, 64*1024)
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		reply := b.handleLine(line)
		// Replies must be a single line: json.Marshal never emits newlines
		// inside string values (it escapes them), so this is safe.
		if _, err := conn.Write([]byte(reply + "\n")); err != nil {
			return
		}
	}
}

// handleLine parses one NDJSON request and returns one JSON reply line.
func (b *Bridge) handleLine(line string) string {
	var req map[string]interface{}
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "INVALID_PARAMS", "detail": "invalid JSON",
		})
	}
	op, _ := req["op"].(string)
	if op == "" {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "INVALID_PARAMS", "detail": "missing op",
		})
	}
	switch op {
	case "ping":
		return b.opPing(req)
	case "list_players":
		return b.opListPlayers(req)
	case "snapshot":
		return b.opSnapshot(req)
	case "drain_events":
		return b.opDrainEvents(req)
	case "apply_action":
		return b.opApplyAction(req)
	case "latest_verdict":
		return b.opLatestVerdict(req)
	case "spectate":
		return b.opSpectate(req)
	case "spectate_feed":
		return b.opSpectateFeed(req)
	case "spectate_stop":
		return b.opSpectateStop(req)
	case "close_session":
		return b.opCloseSession(req)
	case "arena_start":
		return b.opArenaStart(req)
	case "arena_status":
		return b.opArenaStatus(req)
	case "arena_stop":
		return b.opArenaStop(req)
	default:
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "INVALID_PARAMS", "detail": "unknown op " + op,
		})
	}
}

// mustJSON marshals v, guaranteeing a single line (json.Marshal escapes
// newlines inside strings).
func mustJSON(v interface{}) string {
	buf, err := json.Marshal(v)
	if err != nil {
		// Should not happen for the types we send; fall back to a fixed error.
		return `{"ok":false,"code":"INTERNAL","detail":"marshal failed"}`
	}
	return string(buf)
}

// ---- helpers ----

// peerIDFrom extracts the peerID field, returning -1 when missing/invalid.
func peerIDFrom(req map[string]interface{}) int {
	v, ok := req["peerID"]
	if !ok {
		return -1
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	case string:
		i, _ := strconv.Atoi(n)
		return i
	}
	return -1
}

// opPing returns the current frame and protocol version.
func (b *Bridge) opPing(req map[string]interface{}) string {
	return mustJSON(map[string]interface{}{
		"ok":              true,
		"frame":           b.currentFrame(),
		"protocolVersion": bridgeProtocolVersion,
	})
}

// bridgePlayer is one entry of list_players.
type bridgePlayer struct {
	PeerID  int    `json:"peerID"`
	Nick    string `json:"nick"`
	IsAgent bool   `json:"isAgent"`
}

// opListPlayers lists all in-game entities (players + NPCs).
func (b *Bridge) opListPlayers(req map[string]interface{}) string {
	out := make([]bridgePlayer, 0)
	for _, e := range b.srv.world.snapshot() {
		isAgent := e.NpcRID != 0 || e.CharID < 0
		out = append(out, bridgePlayer{PeerID: e.PeerID, Nick: e.Name, IsAgent: isAgent})
	}
	return mustJSON(map[string]interface{}{"ok": true, "players": out})
}

// bridgeVec2 is a JSON-friendly vector.
type bridgeVec2 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

// bridgeHealth is a JSON-friendly health pair.
type bridgeHealth struct {
	Current int64 `json:"current"`
	Max     int64 `json:"max"`
}

// bridgeTarget returns the entity's click target as a vec2 or null.
func bridgeTarget(e *Entity) interface{} {
	if e.Target == nil {
		return nil
	}
	return bridgeVec2{X: e.Target.X, Y: e.Target.Y}
}

// bridgePlayerSnapshot is the "player" object of a snapshot.
type bridgePlayerSnapshot struct {
	Position    bridgeVec2   `json:"position"`
	Alive       bool         `json:"alive"`
	Health      bridgeHealth `json:"health"`
	Level       int64        `json:"level"`
	PeerID      int          `json:"peerID"`
	Nick        string       `json:"nick"`
	IsAgent     bool         `json:"isAgent"`
	MapID       int          `json:"mapID"`
	Orientation bridgeVec2   `json:"orientation"`
	Target      interface{}  `json:"target"`
}

// bridgeNearbyEntry is one entry of a snapshot's nearby list.
type bridgeNearbyEntry struct {
	RID         int64        `json:"rid"`
	Nick        string       `json:"nick"`
	Type        string       `json:"type"`
	Alive       bool         `json:"alive"`
	Health      bridgeHealth `json:"health"`
	State       int64        `json:"state"`
	Position    bridgeVec2   `json:"position"`
	Distance    float64      `json:"distance"`
	PeerID      int          `json:"peerID"`
	Level       int64        `json:"level"`
	Orientation bridgeVec2   `json:"orientation"`
	Target      interface{}  `json:"target"`
}

// entityAlive mirrors ActorCommons.IsAlive: state != DEATH and health > 0.
func entityAlive(e *Entity) bool {
	return e.State != stateDeath && e.Health > 0
}

// visualDistance mirrors SoM's visual reading: Euclidean with y weighted 1.42.
func visualDistance(a, b Vec2) float64 {
	dx := float64(a.X - b.X)
	dy := float64(a.Y-b.Y) * bridgeYWeight
	return math.Sqrt(dx*dx + dy*dy)
}

// buildSnapshot produces the snapshot object for the given peer's entity.
// Returns (snapshot, nil) on success or (nil, errorReason) when the peer has
// no in-game entity.
func (b *Bridge) buildSnapshot(peerID int) (map[string]interface{}, string) {
	e := b.entityByPeer(peerID)
	if e == nil {
		return nil, "PEER_NOT_FOUND"
	}
	player := bridgePlayerSnapshot{
		Position:    bridgeVec2{X: e.Pos.X, Y: e.Pos.Y},
		Alive:       entityAlive(e),
		Health:      bridgeHealth{Current: e.Health, Max: e.MaxHealth},
		Level:       e.Level,
		PeerID:      e.PeerID,
		Nick:        e.Name,
		IsAgent:     e.NpcRID != 0 || e.CharID < 0,
		MapID:       e.MapID,
		Orientation: bridgeVec2{X: e.Orient.X, Y: e.Orient.Y},
		Target:      bridgeTarget(e),
	}
	nearby := make([]bridgeNearbyEntry, 0)
	for _, other := range b.srv.world.snapshot() {
		if other.CharID == e.CharID {
			continue
		}
		if other.MapID != e.MapID {
			continue
		}
		dist := visualDistance(e.Pos, other.Pos)
		if dist > float64(bridgeNearbyRadius) {
			continue
		}
		typ := "player"
		if other.NpcRID != 0 || other.CharID < 0 {
			typ = "monster"
		}
		nearby = append(nearby, bridgeNearbyEntry{
			RID:         other.CharID,
			Nick:        other.Name,
			Type:        typ,
			Alive:       entityAlive(other),
			Health:      bridgeHealth{Current: other.Health, Max: other.MaxHealth},
			State:       other.State,
			Position:    bridgeVec2{X: other.Pos.X, Y: other.Pos.Y},
			Distance:    dist,
			PeerID:      other.PeerID,
			Level:       other.Level,
			Orientation: bridgeVec2{X: other.Orient.X, Y: other.Orient.Y},
			Target:      bridgeTarget(other),
		})
	}
	snap := map[string]interface{}{
		"frame":  b.currentFrame(),
		"player": player,
		"nearby": nearby,
	}
	return snap, ""
}

// entityByPeer finds the in-game entity owned by a peerID.
func (b *Bridge) entityByPeer(peerID int) *Entity {
	for _, e := range b.srv.world.snapshot() {
		if e.PeerID == peerID && e.CharID >= 0 {
			return e
		}
	}
	return nil
}

// opSnapshot returns the snapshot for a peer.
func (b *Bridge) opSnapshot(req map[string]interface{}) string {
	peerID := peerIDFrom(req)
	if peerID < 0 {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "INVALID_PARAMS", "detail": "missing peerID",
		})
	}
	snap, reason := b.buildSnapshot(peerID)
	if snap == nil {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "SESSION_CONFLICT", "detail": reason,
		})
	}
	b.hookAgent(peerID)
	return mustJSON(map[string]interface{}{"ok": true, "snapshot": snap})
}

// hookAgent ensures the peer has an event buffer (SoM's _hook_agent).
func (b *Bridge) hookAgent(peerID int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.events[peerID]; !ok {
		b.events[peerID] = []bridgeEvent{}
	}
}

// pushEvent appends an event to a peer's buffer with a fresh seq number.
func (b *Bridge) pushEvent(peerID int, kind string, data interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.eventSeq++
	ev := bridgeEvent{
		Seq:   b.eventSeq,
		Kind:  kind,
		Data:  data,
		Frame: b.frame,
	}
	buf := b.events[peerID]
	if buf == nil {
		buf = []bridgeEvent{}
	}
	buf = append(buf, ev)
	for len(buf) > bridgeEventBufferMax {
		buf = buf[1:]
	}
	b.events[peerID] = buf
}

// opDrainEvents returns and clears the event buffer for a peer.
func (b *Bridge) opDrainEvents(req map[string]interface{}) string {
	peerID := peerIDFrom(req)
	if peerID < 0 {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "INVALID_PARAMS", "detail": "missing peerID",
		})
	}
	if b.entityByPeer(peerID) == nil {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "SESSION_CONFLICT", "detail": "peer not found",
		})
	}
	b.hookAgent(peerID)
	b.mu.Lock()
	buf := b.events[peerID]
	// Clear the buffer but keep the key so the peer stays hooked.
	b.events[peerID] = []bridgeEvent{}
	cursor := b.eventSeq
	b.mu.Unlock()
	if buf == nil {
		buf = []bridgeEvent{}
	}
	return mustJSON(map[string]interface{}{
		"ok": true, "events": buf, "cursor": cursor,
	})
}

// ---- apply_action ----

// opApplyAction dispatches an action to the same handlers a human client uses.
func (b *Bridge) opApplyAction(req map[string]interface{}) string {
	peerID := peerIDFrom(req)
	if peerID < 0 {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "INVALID_PARAMS", "detail": "missing peerID",
		})
	}
	name, _ := req["name"].(string)
	args, _ := req["args"].(map[string]interface{})
	if args == nil {
		args = map[string]interface{}{}
	}
	c := b.srv.clientByPeer(peerID)
	if c == nil {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "SESSION_CONFLICT", "detail": "peer not found",
		})
	}
	e := b.entityByPeer(peerID)
	if e == nil {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "SESSION_CONFLICT", "detail": "peer not in world",
		})
	}
	frame := b.currentFrame()
	b.hookAgent(peerID)

	switch name {
	case "navigate":
		x, y := argsFloat(args, "x"), argsFloat(args, "y")
		before := e.Pos
		t := Vec2{X: float32(x), Y: float32(y)}
		e.Target = &t
		b.setVerdict(peerID, &bridgeVerdict{
			peerID: peerID, action: "navigate", verdict: "APPLIED",
			provisional: true, basedOn: frame,
			beforePos: before, startedAt: time.Now(),
		})
		return mustJSON(map[string]interface{}{
			"ok": true, "code": "APPLIED", "verdict": "APPLIED", "provisional": true,
			"basedOnFrame": frame,
		})

	case "attack":
		rid := argsInt(args, "rid")
		skill := argsInt(args, "skill")
		target := b.srv.world.get(rid)
		if target == nil {
			return mustJSON(map[string]interface{}{
				"ok": false, "code": "TARGET_NOT_FOUND",
				"detail": fmt.Sprintf("no agent for rid %d", rid), "basedOnFrame": frame,
			})
		}
		beforeHealth := target.Health
		// Reuse the real combat path (applyDamage) so damage/death/loot/
		// bestiary all behave exactly like a human client's TriggerSkill.
		s := b.srv
		reach := float32(defaultAttackRange + skillRangeFor(skill) + 32)
		if e.Pos.dist(target.Pos) > reach {
			b.setVerdict(peerID, &bridgeVerdict{
				peerID: peerID, action: "attack", verdict: "REJECTED",
				reason: "target_too_far", basedOn: frame, targetRID: rid,
			})
			return mustJSON(map[string]interface{}{
				"ok": true, "code": "APPLIED", "verdict": "PENDING", "provisional": true,
				"basedOnFrame": frame,
			})
		}
		// Record the verdict with the pre-attack state so latest_verdict can
		// resolve it against the world (target health / death).
		b.setVerdict(peerID, &bridgeVerdict{
			peerID: peerID, action: "attack", verdict: "APPLIED",
			provisional: true, basedOn: frame,
			beforeHealth: beforeHealth, targetRID: rid,
			startedAt: time.Now(), pendingExists: true,
		})
		s.applyDamage(e, target, s.attackDamageFor(skill, e, target), altHit, skill)
		// Emit bridge events mirroring SoM's signal hooks.
		if target.Health < beforeHealth {
			b.pushEvent(peerID, "damaged", map[string]interface{}{
				"rid": rid, "nick": target.Name, "value": beforeHealth - target.Health,
				"health": target.Health,
			})
		}
		if target.Health <= 0 {
			// Arena monsters are scored by the arena service (kill + arena
			// events, episode bookkeeping); everything else keeps the generic
			// kill/died events.
			if !b.arena.handleMonsterDeath(e, target) {
				b.pushEvent(peerID, "kill", map[string]interface{}{
					"rid": rid, "nick": target.Name, "attackerNick": e.Name,
				})
				b.pushEvent(peerID, "died", map[string]interface{}{"nick": target.Name})
			}
		}
		return mustJSON(map[string]interface{}{
			"ok": true, "code": "APPLIED", "verdict": "APPLIED", "provisional": true,
			"basedOnFrame": frame,
		})

	case "say":
		text, _ := args["text"].(string)
		if len(text) > 200 {
			text = text[:200]
		}
		// Use the real chat handler so LOCAL delivery matches human clients.
		b.srv.handleTriggerChat(c, &ParsedRPC{NameID: rpcTriggerChat, Args: []Variant{
			NewString(chatChannelLocal), NewString(text),
		}})
		b.pushEvent(peerID, "chat", map[string]interface{}{
			"channel": chatChannelLocal, "from": e.Name, "text": text,
		})
		return mustJSON(map[string]interface{}{
			"ok": true, "code": "APPLIED", "verdict": "APPLIED", "provisional": true,
			"basedOnFrame": frame,
		})

	case "inspect":
		rid := argsInt(args, "rid")
		b.srv.handleTriggerSelect(c, &ParsedRPC{NameID: rpcTriggerSelect, Args: []Variant{NewInt(rid)}})
		b.pushEvent(peerID, "npc", map[string]interface{}{"rid": rid, "action": "inspect"})
		return mustJSON(map[string]interface{}{
			"ok": true, "code": "APPLIED", "verdict": "APPLIED", "provisional": true,
			"reason": "select_stats_requested", "basedOnFrame": frame,
		})

	case "respawn":
		if entityAlive(e) {
			return mustJSON(map[string]interface{}{
				"ok": true, "code": "APPLIED", "verdict": "APPLIED", "provisional": true,
				"reason": "already_alive", "basedOnFrame": frame,
			})
		}
		b.setVerdict(peerID, &bridgeVerdict{
			peerID: peerID, action: "respawn", verdict: "APPLIED",
			provisional: true, basedOn: frame,
			startedAt: time.Now(), pendingExists: true,
		})
		b.srv.handleTriggerRespawn(c, &ParsedRPC{NameID: rpcTriggerRespawn})
		return mustJSON(map[string]interface{}{
			"ok": true, "code": "APPLIED", "verdict": "APPLIED", "provisional": true,
			"basedOnFrame": frame,
		})

	case "party":
		op, _ := args["op"].(string)
		target, _ := args["target"].(string)
		switch op {
		case "invite":
			b.srv.handleTriggerPartyInvite(c, &ParsedRPC{NameID: rpcTriggerPartyInvite, Args: []Variant{NewString(target)}})
			b.pushEvent(peerID, "party", map[string]interface{}{"op": "invite", "target": target})
			return mustJSON(map[string]interface{}{
				"ok": true, "code": "APPLIED", "verdict": "APPLIED", "provisional": true,
				"reason": "party_invite_sent", "basedOnFrame": frame,
			})
		case "accept":
			b.srv.handleTriggerPartyAccept(c, &ParsedRPC{NameID: rpcTriggerPartyAccept})
			b.pushEvent(peerID, "party", map[string]interface{}{"op": "accept"})
			return mustJSON(map[string]interface{}{
				"ok": true, "code": "APPLIED", "verdict": "APPLIED", "provisional": true,
				"reason": "party_accept_sent", "basedOnFrame": frame,
			})
		case "decline":
			b.srv.handleTriggerPartyDecline(c, &ParsedRPC{NameID: rpcTriggerPartyDecline})
			b.pushEvent(peerID, "party", map[string]interface{}{"op": "decline"})
			return mustJSON(map[string]interface{}{
				"ok": true, "code": "APPLIED", "verdict": "APPLIED", "provisional": true,
				"reason": "party_declined", "basedOnFrame": frame,
			})
		case "leave":
			b.srv.handleTriggerPartyLeave(c, &ParsedRPC{NameID: rpcTriggerPartyLeave})
			b.pushEvent(peerID, "party", map[string]interface{}{"op": "leave"})
			return mustJSON(map[string]interface{}{
				"ok": true, "code": "APPLIED", "verdict": "APPLIED", "provisional": true,
				"reason": "party_leave_sent", "basedOnFrame": frame,
			})
		case "kick":
			b.srv.handleTriggerPartyKick(c, &ParsedRPC{NameID: rpcTriggerPartyKick, Args: []Variant{NewString(target)}})
			b.pushEvent(peerID, "party", map[string]interface{}{"op": "kick", "target": target})
			return mustJSON(map[string]interface{}{
				"ok": true, "code": "APPLIED", "verdict": "APPLIED", "provisional": true,
				"reason": "party_kick_sent", "basedOnFrame": frame,
			})
		default:
			return mustJSON(map[string]interface{}{
				"ok": false, "code": "INVALID_PARAMS", "detail": "unknown party op: " + op,
			})
		}

	default:
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "INVALID_PARAMS", "detail": "unknown action: " + name,
		})
	}
}

// setVerdict stores the latest verdict for a peer.
func (b *Bridge) setVerdict(peerID int, v *bridgeVerdict) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.verdicts[peerID] = v
}

// opLatestVerdict returns the latest action verdict for a peer. Fire-and-forget
// actions (say/inspect/party) are APPLIED immediately; state-delta actions
// (navigate/attack/respawn) are resolved against the current world state so
// the verdict can be APPLIED, PENDING, or REJECTED.
func (b *Bridge) opLatestVerdict(req map[string]interface{}) string {
	peerID := peerIDFrom(req)
	if peerID < 0 {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "INVALID_PARAMS", "detail": "missing peerID",
		})
	}
	b.mu.Lock()
	v := b.verdicts[peerID]
	b.mu.Unlock()
	if v == nil {
		return mustJSON(map[string]interface{}{"ok": true, "verdict": nil})
	}
	// Resolve state-delta verdicts against the current world state. The
	// action may still be in flight (PENDING) or may have had no effect
	// (REJECTED) — both are reported honestly rather than assumed APPLIED.
	// Work on a copy so we never mutate the shared verdict outside the lock;
	// a concurrent apply_action simply replaces the verdict afterwards.
	out := *v
	if out.pendingExists {
		out.verdict, out.reason = b.resolveVerdict(peerID, &out)
		out.provisional = false
		if out.verdict != "PENDING" {
			out.pendingExists = false
		}
		// Publish the resolved verdict back under the lock.
		b.mu.Lock()
		if cur := b.verdicts[peerID]; cur == v {
			resolved := out
			b.verdicts[peerID] = &resolved
		}
		b.mu.Unlock()
	}
	return mustJSON(map[string]interface{}{
		"ok": true, "verdict": out.verdict, "provisional": out.provisional,
		"reason": out.reason,
	})
}

// resolveVerdict applies SoM's state-delta verdict logic for a pending action.
func (b *Bridge) resolveVerdict(peerID int, v *bridgeVerdict) (string, string) {
	e := b.entityByPeer(peerID)
	if e == nil {
		return "REJECTED", "peer gone"
	}
	elapsed := time.Since(v.startedAt)
	switch v.action {
	case "navigate":
		moved := e.Pos.dist(v.beforePos) > 1.0
		if moved {
			return "APPLIED", "position_changed"
		}
		if elapsed < 1200*time.Millisecond {
			return "PENDING", ""
		}
		return "REJECTED", "no_position_change"
	case "attack":
		target := b.srv.world.get(v.targetRID)
		if target != nil {
			if !entityAlive(target) {
				return "APPLIED", "target_died"
			}
			if target.Health < v.beforeHealth {
				return "APPLIED", "target_health_decreased"
			}
		}
		if elapsed >= 1200*time.Millisecond {
			return "REJECTED", "no_target_change"
		}
		return "PENDING", ""
	case "respawn":
		if entityAlive(e) {
			return "APPLIED", "revived"
		}
		return "PENDING", ""
	default:
		return "APPLIED", "fire_and_forget"
	}
}

// ---- spectate ----

// opSpectate joins spectate mode for a target peer (read-only).
func (b *Bridge) opSpectate(req map[string]interface{}) string {
	peerID := peerIDFrom(req)
	if peerID < 0 {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "INVALID_PARAMS", "detail": "missing peerID",
		})
	}
	e := b.entityByPeer(peerID)
	if e == nil {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "SESSION_CONFLICT", "detail": "peer not found",
		})
	}
	b.hookAgent(peerID)
	b.mu.Lock()
	b.spectators[peerID]++
	n := b.spectators[peerID]
	cursor := b.eventSeq
	b.mu.Unlock()
	snap, _ := b.buildSnapshot(peerID)
	return mustJSON(map[string]interface{}{
		"ok": true, "code": "SPECTATE_JOINED", "readOnly": true,
		"target":     map[string]interface{}{"peerID": peerID, "nick": e.Name},
		"spectators": n,
		"snapshot":   snap,
		"cursor":     cursor,
		"frame":      b.currentFrame(),
	})
}

// opSpectateFeed returns events since the cursor (and optionally a snapshot).
func (b *Bridge) opSpectateFeed(req map[string]interface{}) string {
	peerID := peerIDFrom(req)
	if peerID < 0 {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "INVALID_PARAMS", "detail": "missing peerID",
		})
	}
	e := b.entityByPeer(peerID)
	if e == nil {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "SESSION_CONFLICT", "detail": "peer not found",
		})
	}
	b.hookAgent(peerID)
	since := int64(argsInt(argsToMap(req), "since"))
	includeSnapshot := true
	if v, ok := req["includeSnapshot"]; ok {
		if bv, ok := v.(bool); ok {
			includeSnapshot = bv
		}
	}
	b.mu.Lock()
	all := b.events[peerID]
	out := make([]bridgeEvent, 0, len(all))
	for _, ev := range all {
		if ev.Seq > since {
			out = append(out, ev)
		}
	}
	cursor := b.eventSeq
	b.mu.Unlock()
	payload := map[string]interface{}{
		"ok": true, "readOnly": true,
		"target": map[string]interface{}{"peerID": peerID, "nick": e.Name},
		"events": out,
		"cursor": cursor,
		"frame":  b.currentFrame(),
	}
	if includeSnapshot {
		snap, _ := b.buildSnapshot(peerID)
		payload["snapshot"] = snap
	}
	return mustJSON(payload)
}

// opSpectateStop leaves spectate mode for a target peer.
func (b *Bridge) opSpectateStop(req map[string]interface{}) string {
	peerID := peerIDFrom(req)
	if peerID < 0 {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "INVALID_PARAMS", "detail": "missing peerID",
		})
	}
	b.mu.Lock()
	n := b.spectators[peerID]
	if n <= 1 {
		delete(b.spectators, peerID)
		n = 0
	} else {
		n--
		b.spectators[peerID] = n
	}
	b.mu.Unlock()
	return mustJSON(map[string]interface{}{
		"ok": true, "code": "SPECTATE_LEFT", "peerID": peerID,
		"spectators": n, "readOnly": true,
	})
}

// ---- arena ----

// arena_start starts an arena episode for a peer using the given template.
// It mirrors ArenaController.Start: EPISODE_ACTIVE when a run is already in
// progress, UNKNOWN_TEMPLATE for an unknown preset, SESSION_CONFLICT when the
// peer has no in-game entity.
func (b *Bridge) arena_start(peerID int, nick, template string) (map[string]interface{}, error) {
	return b.arena.start(peerID, nick, template), nil
}

// arena_status returns the peer's current episode (null when there is none).
func (b *Bridge) arena_status(peerID int) map[string]interface{} {
	return b.arena.status(peerID)
}

// arena_stop ends the peer's active episode, removing its monsters.
func (b *Bridge) arena_stop(peerID int) map[string]interface{} {
	return b.arena.stop(peerID)
}

// opArenaStart handles the arena_start op (requires peerID, nick, template).
func (b *Bridge) opArenaStart(req map[string]interface{}) string {
	peerID := peerIDFrom(req)
	if peerID < 0 {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "INVALID_PARAMS", "detail": "missing peerID",
		})
	}
	nick, _ := req["nick"].(string)
	if nick == "" {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "INVALID_PARAMS", "detail": "missing nick",
		})
	}
	template, _ := req["template"].(string)
	if template == "" {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "INVALID_PARAMS", "detail": "missing template",
		})
	}
	res, err := b.arena_start(peerID, nick, template)
	if err != nil {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "INTERNAL", "detail": err.Error(),
		})
	}
	return mustJSON(res)
}

// opArenaStatus handles the arena_status op (requires peerID).
func (b *Bridge) opArenaStatus(req map[string]interface{}) string {
	peerID := peerIDFrom(req)
	if peerID < 0 {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "INVALID_PARAMS", "detail": "missing peerID",
		})
	}
	return mustJSON(b.arena_status(peerID))
}

// opArenaStop handles the arena_stop op (requires peerID).
func (b *Bridge) opArenaStop(req map[string]interface{}) string {
	peerID := peerIDFrom(req)
	if peerID < 0 {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "INVALID_PARAMS", "detail": "missing peerID",
		})
	}
	return mustJSON(b.arena_stop(peerID))
}

// opCloseSession closes the game session for a peer (mirrors FullyDisconnect).
func (b *Bridge) opCloseSession(req map[string]interface{}) string {
	peerID := peerIDFrom(req)
	if peerID < 0 {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "INVALID_PARAMS", "detail": "missing peerID",
		})
	}
	if b.entityByPeer(peerID) == nil {
		return mustJSON(map[string]interface{}{
			"ok": false, "code": "SESSION_CONFLICT", "detail": "peer not found",
		})
	}
	// Clean up bridge-side state.
	b.mu.Lock()
	delete(b.events, peerID)
	delete(b.verdicts, peerID)
	delete(b.spectators, peerID)
	b.mu.Unlock()
	// Disconnect the game peer: close its WebSocket so the server's normal
	// disconnect path (leaveWorld, party/trade cleanup) runs.
	if c := b.srv.clientByPeer(peerID); c != nil {
		c.conn.Close()
	}
	return mustJSON(map[string]interface{}{"ok": true, "code": "CLOSED"})
}

// ---- arg helpers ----

// argsToMap normalises a request field to a map (nil when absent).
func argsToMap(req map[string]interface{}) map[string]interface{} {
	if m, ok := req["args"].(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

// argsInt reads an integer field from a JSON args map.
func argsInt(args map[string]interface{}, key string) int64 {
	v, ok := args[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int64(n)
	case json.Number:
		i, _ := n.Int64()
		return i
	case string:
		i, _ := strconv.ParseInt(n, 10, 64)
		return i
	}
	return 0
}

// argsFloat reads a float field from a JSON args map.
func argsFloat(args map[string]interface{}, key string) float64 {
	v, ok := args[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case json.Number:
		f, _ := n.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return f
	}
	return 0
}

// skillRangeFor returns the skill's range bonus (0 for unknown skills).
func skillRangeFor(skillID int64) int64 {
	if p, ok := skillTable[skillID]; ok {
		return p.Range
	}
	return 0
}

// attackDamageFor computes a deterministic damage value for a bridge attack.
// It mirrors computeDamage's shape (attack + level - defense, no RNG roll) but
// is deterministic so tests can rely on the result. Equipment modifiers are
// included so bridge attacks and RPC attacks hit just as hard.
func (s *Server) attackDamageFor(skillID int64, agent, target *Entity) int64 {
	profile, ok := skillTable[skillID]
	if !ok {
		// Unknown skill: fall back to a small flat hit.
		return 5
	}
	attack := profile.Attack + agent.Level + s.equipmentAttackBonus(agent)
	defense := target.Level/2 + s.equipmentDefenseBonus(target)
	dmg := attack - defense
	if dmg < 1 {
		dmg = 1
	}
	return dmg
}
