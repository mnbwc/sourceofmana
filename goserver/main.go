package main

// Phase-0 feasibility server: a Go WebSocket endpoint that speaks enough of
// Godot's SceneMultiplayer protocol for a real Godot client to connect, send
// an RPC, and receive a reply. This validates the protocol bridge before any
// game logic is ported.
//
// Run with:  go run .            (listens on :27012)
// Test with: go test -run TestEndToEnd -v

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	// Godot's WebSocketMultiplayerPeer sends no Origin header, so the default
	// same-origin check must be disabled.
	CheckOrigin: func(r *http.Request) bool { return true },
	// Godot advertises the "multiplayer" subprotocol in some builds; accept
	// any subprotocol so the handshake never fails.
	Subprotocols: []string{"multiplayer"},
}

// Client represents one connected Godot peer.
type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	mu     sync.Mutex
	peerID int
	// Session is the per-peer login state machine (see login.go).
	session *Session
	// nodeCache maps cache IDs to node paths for nodes the peer advertised to
	// us (client -> server direction). Used when decoding incoming RPCs.
	nodeCache map[uint32]string
	// serverCacheID is the cache ID we assigned to our own node and announced
	// to the client (server -> client direction). Used when we push RPCs.
	serverCacheID uint32
	// serverCacheAnnounced tracks whether the client confirmed our node.
	serverCacheAnnounced bool
}

func (c *Client) writePacket(buf []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.conn.WriteMessage(websocket.BinaryMessage, buf); err != nil {
		log.Printf("peer %d write: %v", c.peerID, err)
	}
}

func (c *Client) enqueue(buf []byte) {
	select {
	case c.send <- buf:
	default:
		log.Printf("peer %d send channel full, dropping packet", c.peerID)
	}
}

// announceNode sends a SIMPLIFY_PATH packet for the node the client addressed,
// so the client learns which cache ID to use when sending us RPCs. The MD5 is
// a placeholder — Godot only logs a warning on mismatch and continues.
func (c *Client) announceNode(path string) {
	if c.serverCacheAnnounced {
		return
	}
	c.serverCacheID = 1
	md5 := "00000000000000000000000000000000"
	c.enqueue(BuildSimplifyPathPacket(c.serverCacheID, md5, path))
	c.serverCacheAnnounced = true
	log.Printf("peer %d announced node cacheID=%d path=%q", c.peerID, c.serverCacheID, path)
}

// Server holds all connected clients.
type Server struct {
	mu      sync.RWMutex
	clients map[int]*Client
	nextID  int
	// store is the account database, mirrored to disk (see store.go).
	store *AccountStore
	// world is the phase-2 in-game entity state (AOI/movement).
	world *world
	// npcs/quest/dialogue state (phase-3).
	npcs      *npcWorld
	progress  *progressStore
	dialogues *dialogueStore
	// party state (phase-3).
	party *partyService
	// trade state (phase-3).
	trade *tradeService
	// phase-6: buffs, ground drops, private-stats sync.
	buffMu     sync.Mutex
	buffs      map[int64][]*Buff
	dropMu     sync.Mutex
	drops      map[int64]*Drop
	nextDropID int64
	// phase-8: GM commands, bestiary, skills.
	randSrc rand.Source
	// password reset codes (progression.go).
	resetCodes map[string]resetCode
	// bridge is the AgentBridge TCP NDJSON interface (port 9101). It is
	// started in main after the WebSocket server is listening.
	bridge *Bridge
}

// resetCode is a server-side password reset verification code.
type resetCode struct {
	Code    string
	Expires time.Time
}

func NewServer() *Server {
	s := &Server{
		clients:    make(map[int]*Client),
		nextID:     2,           // peer 1 is the server itself in Godot's numbering
		store:      loadStore(), // phase-3: restore accounts from disk if present
		world:      newWorld(),
		npcs:       newNpcWorld(),
		progress:   newProgressStore(),
		dialogues:  newDialogueStore(),
		party:      newPartyService(),
		trade:      newTradeService(),
		buffs:      make(map[int64][]*Buff),
		drops:      make(map[int64]*Drop),
		randSrc:    rand.NewSource(time.Now().UnixNano()),
		resetCodes: make(map[string]resetCode),
	}
	go s.worldTick()     // phase-2 movement broadcaster
	s.spawnNpcEntities() // phase-8: add NPCs to the world as targetable entities
	return s
}

func (s *Server) add(c *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c.peerID] = c
}

func (s *Server) remove(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, id)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade: %v", err)
		return
	}

	s.mu.Lock()
	peerID := s.nextID
	s.nextID++
	s.mu.Unlock()

	c := &Client{
		conn:      conn,
		send:      make(chan []byte, 256),
		peerID:    peerID,
		nodeCache: make(map[uint32]string),
		session:   &Session{charID: peerUnknownID},
	}
	s.add(c)

	log.Printf("peer %d connected from %s", peerID, conn.RemoteAddr())

	// Godot's WebSocketMultiplayerPeer client stays in CONNECTION_CONNECTING
	// until the server pushes a 4-byte little-endian unique_id >= 2. Send it
	// immediately after the handshake.
	idBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(idBytes, uint32(peerID))
	if err := conn.WriteMessage(websocket.BinaryMessage, idBytes); err != nil {
		log.Printf("peer %d failed to send unique_id: %v", peerID, err)
		s.remove(peerID)
		conn.Close()
		return
	}
	log.Printf("peer %d assigned unique_id=%d", peerID, peerID)

	// Writer goroutine: drains the send channel.
	go func() {
		for buf := range c.send {
			if err := conn.WriteMessage(websocket.BinaryMessage, buf); err != nil {
				log.Printf("peer %d write: %v", peerID, err)
				return
			}
		}
	}()

	// Reader: Godot expects the server to push packets; the client sends
	// sys packets for peer admission and RPC packets for remote calls.
	// Limit message size to 64 KB to prevent memory exhaustion attacks.
	conn.SetReadLimit(64 * 1024)
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("peer %d read: %v", peerID, err)
			break
		}
		if len(data) == 0 {
			continue
		}
		s.handlePacket(c, data)
	}

	s.remove(peerID)
	s.leaveWorld(c) // phase 2: notify AOI peers about the departure
	close(c.send)
	conn.Close()
	log.Printf("peer %d disconnected", peerID)
}

// handlePacket routes one SceneMultiplayer packet.
func (s *Server) handlePacket(c *Client, data []byte) {
	cmd, rpc, cacheID, path, err := ParsePacket(data)
	if err != nil {
		log.Printf("peer %d parse: %v (hex %x)", c.peerID, err, data)
		return
	}
	log.Printf("peer %d packet cmd=%d hex=%x", c.peerID, cmd, data)
	switch cmd {
	case CmdRemoteCall:
		if rpc == nil {
			return
		}
		log.Printf("peer %d RPC nameID=%d args=%s", c.peerID, rpc.NameID, formatArgs(rpc.Args))
		s.onRemoteCall(c, rpc)
	case CmdSimplifyPath:
		log.Printf("peer %d SIMPLIFY_PATH cacheID=%d path=%q", c.peerID, cacheID, path)
		c.nodeCache[cacheID] = path
		// Acknowledge so the client marks the cache entry confirmed.
		c.enqueue(BuildConfirmPathPacket(cacheID, true))
		// Teach the client about our own node so it can address replies to us.
		// The client's cache of the server (peer 1) starts empty; without this
		// it cannot route anything back.
		c.announceNode(path)
	case CmdConfirmPath:
		log.Printf("peer %d CONFIRM_PATH cacheID=%d", c.peerID, cacheID)
	case CmdSys:
		log.Printf("peer %d SYS packet: %x", c.peerID, data)
		s.onSysPacket(c, data)
	default:
		log.Printf("peer %d cmd=%d (unhandled)", c.peerID, cmd)
	}
}

// onSysPacket handles Godot's system commands (peer admission, relay, etc.).
func (s *Server) onSysPacket(c *Client, data []byte) {
	if len(data) < 2 {
		return
	}
	switch data[1] {
	case 0: // SYS_COMMAND_ADD_PEER
		log.Printf("peer %d ADD_PEER sys packet", c.peerID)
	case 1: // SYS_COMMAND_DEL_PEER
		log.Printf("peer %d DEL_PEER sys packet", c.peerID)
	default:
		log.Printf("peer %d unknown sys cmd %d", c.peerID, data[1])
	}
}

func formatArgs(args []Variant) string {
	out := "["
	for i, a := range args {
		if i > 0 {
			out += " "
		}
		switch a.Type {
		case VariantString, VariantStringName:
			out += a.Str
		case VariantInt:
			out += itoa(a.Int)
		case VariantBool:
			if a.Bool {
				out += "true"
			} else {
				out += "false"
			}
		case VariantVector2:
			out += "(" + ftoa(float64(a.Vec2[0])) + "," + ftoa(float64(a.Vec2[1])) + ")"
		default:
			out += "?"
		}
	}
	return out + "]"
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func ftoa(v float64) string {
	// Minimal float formatting for logs; avoid importing strconv for now.
	if v == float64(int64(v)) {
		return itoa(int64(v))
	}
	return itoa(int64(v)) + ".0"
}

// routes wires the HTTP endpoints for the server.
func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	// Accept WebSocket connections at any path.  Godot's WebSocketMultiplayerPeer
	// connects to the bare host:port without a path, so a strict /ws-only route
	// would reject browser clients.  The original SoM server (Godot WebSocketServer)
	// also accepts any path, so this matches its behaviour.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") == "websocket" {
			s.handleWS(w, r)
			return
		}
		// CORS preflight (used by browser web clients)
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"ok":true}`)
	})
	// GM web API (browser-based admin panel).
	s.gmRoutes(mux)
	return mux
}

// bindAddr returns "IP:PORT" from the per-endpoint bind env vars.
// Resolution order (per endpoint):
//   1. <PREFIX>_BIND_IP  (e.g. WS_BIND_IP, WSS_BIND_IP, BRIDGE_BIND_IP)
//   2. BIND_IP           (global fallback)
//   3. ""               (all interfaces)
// This lets a VPS expose the game port publicly while keeping the admin
// endpoints (AgentBridge) loopback-only, without running two server
// instances.
func bindAddr(portEnv string, bindPrefix string, defaultPort int) string {
	ip := os.Getenv(bindPrefix + "_BIND_IP")
	if ip == "" {
		ip = os.Getenv("BIND_IP")
	}
	port := envInt(portEnv, defaultPort)
	if ip == "" {
		return fmt.Sprintf(":%d", port)
	}
	return fmt.Sprintf("%s:%d", ip, port)
}

func main() {
	s := NewServer()

	// WebSocket endpoint (ws://) — native clients, AI headless, nginx proxy.
	wsAddr := bindAddr("WS_PORT", "WS", 27012)
	log.Printf("Go Game Server (ws://) running on %s", wsAddr)
	go func() {
		log.Fatal(http.ListenAndServe(wsAddr, s.routes()))
	}()

	// Secure WebSocket endpoint (wss://) — browser clients.
	if os.Getenv("WSS_PORT") != "" {
		wssAddr := bindAddr("WSS_PORT", "WSS", 27013)
		tlsCfg := loadTLSConfig()
		log.Printf("Go Game Server (wss://) running on %s", wssAddr)
		go func() {
			server := &http.Server{
				Addr:      wssAddr,
				Handler:   s.routes(),
				TLSConfig: tlsCfg,
			}
			log.Fatal(server.ListenAndServeTLS("", ""))
		}()
	}

	// AgentBridge TCP NDJSON interface — AI agent exploration.
	bridgeAddr := bindAddr("BRIDGE_PORT", "BRIDGE", bridgeDefaultPort)
	log.Printf("AgentBridge running on %s", bridgeAddr)
	if _, err := StartBridge(s, bridgeAddr); err != nil {
		log.Printf("AgentBridge: start failed: %v", err)
	}
	select {}
}
