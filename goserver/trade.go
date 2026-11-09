package main

// Phase-3 trade system.
//
// Mirrors SoM's server-side trade flow (sources/world/WorldTrade.gd,
// sources/network/server/Server.gd TradeRespond/TradeSetOffer/TradeLock/
// TradeCancel):
//
//   /trade <nick>        ->  TradeRequested(requesterName) to the target
//   TradeRespond(accept) ->  accept -> TradeOpened to both, then TradeUpdate
//                            decline -> PushNotification (system message)
//   TradeSetOffer(entries, gp) -> update own offer, unlock both, TradeUpdate
//   TradeLock(revision, state) -> lock own side; both locked -> commit
//   TradeCancel           ->  TradeClosed(reason) to both
//
// TradeUpdate carries the offers as opaque arrays (itemID/count/customfield
// triples). Full item validation is deferred to the inventory system; the
// server tracks the session state machine and performs the commit on
// both-locked.

import (
	"log"
	"strings"
	"sync"
	"time"
)

// tradeRequestTimeout mirrors ActorCommons.TradeRequestTimeoutMs.
const tradeRequestTimeout = 30 * time.Second

// tradeOffer is one side's offer: a list of entries plus a GP amount.
type tradeOffer struct {
	Entries []tradeEntry
	GP      int64
}

// tradeEntry mirrors WorldTrade's [cell.id, count, cell.customfield].
type tradeEntry struct {
	ItemID      int64
	Count       int64
	Customfield string
}

// tradeSession mirrors WorldTrade.Session.
type tradeSession struct {
	// parties[0] = requester, parties[1] = target.
	parties  [2]*Entity
	offers   [2]tradeOffer
	locked   [2]bool
	revision int64
}

// other returns the index of the other party.
func (ts *tradeSession) other(idx int) int {
	if idx == 0 {
		return 1
	}
	return 0
}

// indexOf returns 0 for the requester, 1 for the target.
func (ts *tradeSession) indexOf(p *Entity) int {
	if ts.parties[0] == p {
		return 0
	}
	return 1
}

// tradeService holds pending requests and live sessions.
type tradeService struct {
	mu       sync.Mutex
	requests map[int64]int64 // target charID -> requester charID
	reqTick  map[int64]time.Time
	// sessions keyed by charID (both parties map to the same session).
	sessions map[int64]*tradeSession
}

func newTradeService() *tradeService {
	return &tradeService{
		requests: make(map[int64]int64),
		reqTick:  make(map[int64]time.Time),
		sessions: make(map[int64]*tradeSession),
	}
}

// request mirrors WorldTrade.Request. It notifies via the server.
func (s *Server) tradeRequest(requester, target *Entity) {
	now := time.Now()
	s.trade.mu.Lock()
	// Drop an expired request for the target.
	if tick, ok := s.trade.reqTick[target.CharID]; ok && now.Sub(tick) > tradeRequestTimeout {
		delete(s.trade.requests, target.CharID)
		delete(s.trade.reqTick, target.CharID)
	}
	_, targetBusy := s.trade.sessions[target.CharID]
	_, requesterBusy := s.trade.sessions[requester.CharID]
	tooFar := requester.MapID != target.MapID || requester.Pos.dist(target.Pos) > aoiRadius
	if targetBusy {
		s.trade.mu.Unlock()
		s.notifyEntity(requester, target.Name+" is busy.")
		return
	}
	if requesterBusy {
		s.trade.mu.Unlock()
		s.notifyEntity(requester, "You can't trade right now.")
		return
	}
	if tooFar {
		s.trade.mu.Unlock()
		s.notifyEntity(requester, target.Name+" is too far away to trade.")
		return
	}
	s.trade.requests[target.CharID] = requester.CharID
	s.trade.reqTick[target.CharID] = now
	s.trade.mu.Unlock()
}

// respond mirrors WorldTrade.Respond. Returns the opened session, if any.
func (s *Server) tradeRespond(target *Entity, accept bool) *tradeSession {
	s.trade.mu.Lock()
	requesterRID, ok := s.trade.requests[target.CharID]
	delete(s.trade.requests, target.CharID)
	delete(s.trade.reqTick, target.CharID)
	if !ok {
		s.trade.mu.Unlock()
		return nil
	}
	if !accept {
		requester := s.world.get(requesterRID)
		s.trade.mu.Unlock()
		if requester != nil {
			s.notifyEntity(requester, target.Name+" declined your trade request.")
		}
		return nil
	}
	requester := s.world.get(requesterRID)
	if requester == nil {
		s.trade.mu.Unlock()
		return nil
	}
	_, targetBusy := s.trade.sessions[target.CharID]
	_, requesterBusy := s.trade.sessions[requester.CharID]
	tooFar := requester.MapID != target.MapID || requester.Pos.dist(target.Pos) > aoiRadius
	if targetBusy {
		s.trade.mu.Unlock()
		s.notifyEntity(target, "You can't trade right now.")
		return nil
	}
	if requesterBusy {
		s.trade.mu.Unlock()
		s.notifyEntity(target, requester.Name+" is busy.")
		return nil
	}
	if tooFar {
		s.trade.mu.Unlock()
		s.notifyEntity(target, requester.Name+" is too far away to trade.")
		return nil
	}
	session := &tradeSession{parties: [2]*Entity{requester, target}}
	s.trade.sessions[requester.CharID] = session
	s.trade.sessions[target.CharID] = session
	s.trade.mu.Unlock()
	return session
}

// setOffer mirrors WorldTrade.SetOffer (state-machine side; item validation
// is deferred to the inventory system).
func (s *Server) tradeSetOffer(player *Entity, entries []tradeEntry, gp int64) *tradeSession {
	s.trade.mu.Lock()
	defer s.trade.mu.Unlock()
	session := s.trade.sessions[player.CharID]
	if session == nil {
		return nil
	}
	if gp < 0 {
		s.notifyEntity(player, "Your offer is no longer valid.")
		return session
	}
	idx := session.indexOf(player)
	session.offers[idx] = tradeOffer{Entries: entries, GP: gp}
	session.revision++
	session.locked = [2]bool{}
	return session
}

// setLock mirrors WorldTrade.SetLock. Returns (session, committed).
func (s *Server) tradeSetLock(player *Entity, revision int64, state bool) (*tradeSession, bool) {
	s.trade.mu.Lock()
	defer s.trade.mu.Unlock()
	session := s.trade.sessions[player.CharID]
	if session == nil {
		return nil, false
	}
	if revision != session.revision {
		return session, false
	}
	idx := session.indexOf(player)
	session.locked[idx] = state
	if session.locked[0] && session.locked[1] {
		return session, true
	}
	return session, false
}

// cancel mirrors WorldTrade.Cancel.
func (s *Server) tradeCancel(player *Entity) *tradeSession {
	s.trade.mu.Lock()
	defer s.trade.mu.Unlock()
	session := s.trade.sessions[player.CharID]
	if session == nil {
		return nil
	}
	s.detachSessionLocked(session)
	return session
}

// commit mirrors WorldTrade.Commit: validate both offers, then atomically
// swap items and GP between the two characters.
func (s *Server) tradeCommit(session *tradeSession) {
	s.trade.mu.Lock()
	defer s.trade.mu.Unlock()

	// Validate both offers before committing.
	for idx, party := range session.parties {
		if party == nil {
			return
		}
		chr := s.store.findCharacterByID(party.CharID)
		if chr == nil {
			return
		}
		offer := session.offers[idx]
		// Verify GP.
		if offer.GP < 0 || offer.GP > chr.GP {
			log.Printf("trade: %s offered %d GP but only has %d", chr.Name, offer.GP, chr.GP)
			return
		}
		// Verify each item entry exists in inventory with sufficient count.
		for _, entry := range offer.Entries {
			if entry.Count <= 0 {
				log.Printf("trade: %s offered invalid count %d for item %d", chr.Name, entry.Count, entry.ItemID)
				return
			}
			if !s.hasItem(chr, entry.ItemID, entry.Customfield, entry.Count) {
				log.Printf("trade: %s offered %dx%d but doesn't own it", chr.Name, entry.Count, entry.ItemID)
				return
			}
		}
	}

	// Apply the swap: remove offered items/GP from each party, add the other's.
	for idx, party := range session.parties {
		chr := s.store.findCharacterByID(party.CharID)
		if chr == nil {
			return
		}
		offer := session.offers[idx]
		// Deduct offered GP.
		chr.GP -= offer.GP
		// Remove offered items.
		for _, entry := range offer.Entries {
			removeItem(chr, entry.ItemID, entry.Customfield, entry.Count)
		}
		// Add the other party's offered items.
		otherOffer := session.offers[session.other(idx)]
		chr.GP += otherOffer.GP
		for _, entry := range otherOffer.Entries {
			s.addItem(chr, entry.ItemID, entry.Customfield, entry.Count)
		}
		s.persist()
	}

	s.detachSessionLocked(session)
	log.Printf("trade: committed between %s and %s",
		session.parties[0].Name, session.parties[1].Name)
}

// hasItem checks if the character owns at least count of the item.
func (s *Server) hasItem(chr *Character, itemID int64, customfield string, count int64) bool {
	var owned int64
	for _, item := range chr.Inventory {
		if item.ItemID == itemID && item.Customfield == customfield {
			owned += item.Count
		}
	}
	return owned >= count
}

// addItem adds count of an item to the character's inventory (stacking).
func (s *Server) addItem(chr *Character, itemID int64, customfield string, count int64) {
	for i := range chr.Inventory {
		if chr.Inventory[i].ItemID == itemID && chr.Inventory[i].Customfield == customfield {
			chr.Inventory[i].Count += count
			return
		}
	}
	chr.Inventory = append(chr.Inventory, ItemRecord{ItemID: itemID, Customfield: customfield, Count: count})
}

// detachSessionLocked removes the session from both parties. Callers hold mu.
func (s *Server) detachSessionLocked(session *tradeSession) {
	for _, party := range session.parties {
		delete(s.trade.sessions, party.CharID)
	}
}

// notifyEntity sends a PushNotification to an entity's peer.
func (s *Server) notifyEntity(e *Entity, text string) {
	if e == nil {
		return
	}
	if c := s.clientByPeer(e.PeerID); c != nil {
		sendToClient(c, nil, rpcPushNotification, []Variant{NewString(text), NewInt(int64(e.PeerID))})
	}
}

// sendTradeUpdate mirrors WorldTrade.SendUpdate: TradeUpdate to both parties.
func (s *Server) sendTradeUpdate(session *tradeSession) {
	for idx, p := range session.parties {
		if p == nil {
			continue
		}
		other := session.other(idx)
		if c := s.clientByPeer(p.PeerID); c != nil {
			sendToClient(c, nil, rpcTradeUpdate, []Variant{
				offerArgs(session.offers[idx]),
				NewInt(session.offers[idx].GP),
				offerArgs(session.offers[other]),
				NewInt(session.offers[other].GP),
				NewBool(session.locked[idx]),
				NewBool(session.locked[other]),
				NewInt(session.revision),
				NewInt(int64(p.PeerID)),
			})
		}
	}
}

// sendTradeClosed mirrors WorldTrade.Close: TradeClosed(reason) to both.
func (s *Server) sendTradeClosed(session *tradeSession, reason string) {
	for _, p := range session.parties {
		if p == nil {
			continue
		}
		if c := s.clientByPeer(p.PeerID); c != nil {
			sendToClient(c, nil, rpcTradeClosed, []Variant{NewString(reason), NewInt(int64(p.PeerID))})
		}
	}
}

// ---- wire helpers ----

// offerArgs converts a tradeOffer to the wire format (array of
// [itemID, count, customfield] arrays).
func offerArgs(o tradeOffer) Variant {
	entries := make([]Variant, 0, len(o.Entries))
	for _, e := range o.Entries {
		entries = append(entries, Variant{Type: VariantArray, Arr: []Variant{
			NewInt(e.ItemID), NewInt(e.Count), NewString(e.Customfield),
		}})
	}
	return Variant{Type: VariantArray, Arr: entries}
}

// parseOffer converts the wire format back to a tradeOffer.
func parseOffer(v Variant) []tradeEntry {
	if v.Type != VariantArray {
		return nil
	}
	out := make([]tradeEntry, 0, len(v.Arr))
	for _, e := range v.Arr {
		if e.Type != VariantArray && e.Type != VariantPackedInt64Array {
			continue
		}
		if len(e.Arr) < 2 {
			continue
		}
		entry := tradeEntry{ItemID: e.Arr[0].Int, Count: e.Arr[1].Int}
		if len(e.Arr) >= 3 {
			entry.Customfield = e.Arr[2].Str
		}
		out = append(out, entry)
	}
	return out
}

// ---- server handlers ----

// handleTriggerTradeRequest initiates a trade (mirrors WorldCommands.CommandTrade).
func (s *Server) handleTriggerTradeRequest(c *Client, targetNick string) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	target := s.world.byName(targetNick)
	if target == nil {
		sendToClient(c, nil, rpcCommandFeedback, []Variant{NewString("Player '" + targetNick + "' is not online"), NewInt(int64(c.peerID))})
		return
	}
	if target.CharID == e.CharID {
		sendToClient(c, nil, rpcCommandFeedback, []Variant{NewString("You cannot trade with yourself"), NewInt(int64(c.peerID))})
		return
	}
	s.tradeRequest(e, target)
	// TradeRequested -> the target; PushNotification -> the requester.
	if tc := s.clientByPeer(target.PeerID); tc != nil {
		sendToClient(tc, nil, rpcTradeRequested, []Variant{NewString(e.Name), NewInt(int64(target.PeerID))})
	}
	s.notifyEntity(e, "Trade request sent to "+target.Name)
	log.Printf("trade: %s requested %s", e.Name, targetNick)
}

// handleTradeRespond mirrors Server.gd TradeRespond.
func (s *Server) handleTradeRespond(c *Client, rpc *ParsedRPC) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	accept := argBool(rpc.Args, 0)
	session := s.tradeRespond(e, accept)
	if session == nil {
		return
	}
	// TradeOpened to both, then the initial TradeUpdate.
	for _, p := range session.parties {
		other := session.parties[session.other(session.indexOf(p))]
		if pc := s.clientByPeer(p.PeerID); pc != nil {
			sendToClient(pc, nil, rpcTradeOpened, []Variant{NewString(other.Name), NewInt(int64(p.PeerID))})
		}
	}
	s.sendTradeUpdate(session)
	log.Printf("trade: opened between %s and %s", session.parties[0].Name, session.parties[1].Name)
}

// handleTradeSetOffer mirrors Server.gd TradeSetOffer.
func (s *Server) handleTradeSetOffer(c *Client, rpc *ParsedRPC) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	entries := parseOffer(argVariant(rpc.Args, 0))
	gp := argInt(rpc.Args, 1)
	session := s.tradeSetOffer(e, entries, gp)
	if session != nil {
		s.sendTradeUpdate(session)
	}
}

// handleTradeLock mirrors Server.gd TradeLock.
func (s *Server) handleTradeLock(c *Client, rpc *ParsedRPC) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	revision := argInt(rpc.Args, 0)
	state := argBool(rpc.Args, 1)
	session, committed := s.tradeSetLock(e, revision, state)
	if session == nil {
		return
	}
	if committed {
		s.tradeCommit(session)
		s.sendTradeClosed(session, "Trade completed.")
		log.Printf("trade: completed between %s and %s", session.parties[0].Name, session.parties[1].Name)
		return
	}
	s.sendTradeUpdate(session)
}

// handleTradeCancel mirrors Server.gd TradeCancel.
func (s *Server) handleTradeCancel(c *Client, rpc *ParsedRPC) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	reason := e.Name + " cancelled the trade."
	session := s.tradeCancel(e)
	if session != nil {
		s.sendTradeClosed(session, reason)
	}
}

// handleTradeCommand parses the /trade slash command and dispatches it.
func (s *Server) handleTradeCommand(c *Client, command string) {
	cmd := strings.TrimSpace(command)
	if !strings.HasPrefix(cmd, "trade ") {
		return
	}
	rest := strings.TrimSpace(strings.TrimPrefix(cmd, "trade"))
	rest = strings.Trim(rest, "\"")
	if rest == "" {
		sendToClient(c, nil, rpcCommandFeedback, []Variant{NewString("Usage: trade <player>"), NewInt(int64(c.peerID))})
		return
	}
	s.handleTriggerTradeRequest(c, rest)
}
