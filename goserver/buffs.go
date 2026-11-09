package main

import (
	"log"
	"time"
)

// Phase-6: buffs, ground drops, private-stats sync, entity removal,
// notifications, expressions, projectiles, morphs, context toggles,
// navigation clearing, inventory retrieval, and tutorial/UI helpers.
//
// Wire format mirrors sources/network/server/Server.gd + World.gd
// (nameIDs are the sorted indices of the 119 @rpc methods in Network.gd):
//   UpdateBuff=111, EntityBuff=36, DropAdded=32, DropRemoved=34,
//   UpdatePrivateStats=113, RemoveEntity=61, RetrieveInventory=67,
//   PushNotification=56, Express=38, ThrowProjectile=80, Morphed=49,
//   ToggleContext=81, ClearNavigation=14.

const (
	rpcClearNavigation           = 14
	rpcClearProgressionTracker   = 15
	rpcCommandModifier           = 17
	rpcDisplayActions            = 30
	rpcDisplayProgressionTracker = 31
	rpcDropAdded                 = 32
	rpcDropRemoved               = 34
	rpcExpress                   = 38
	rpcHighlightUI               = 40
	rpcMorphed                   = 49
	rpcOpenUI                    = 50
	rpcRemoveEntity              = 61
	rpcRetrieveInventory         = 67
	rpcToggleContext             = 81
	rpcUpdatePrivateStats        = 113
)

// CellCommons.Modifier enum (0-based, mirrors sources/cell/CellCommons.gd).
const (
	modNone           = 0
	modHealth         = 1
	modMana           = 2
	modStamina        = 3
	modMaxMana        = 4
	modRegenMana      = 5
	modCritRate       = 6
	modMAttack        = 7
	modMDefense       = 8
	modMaxStamina     = 9
	modRegenStamina   = 10
	modCooldownDelay  = 11
	modMaxHealth      = 12
	modRegenHealth    = 13
	modDefense        = 14
	modCastDelay      = 15
	modDodgeRate      = 16
	modAttackRange    = 17
	modWalkSpeed      = 18
	modWeightCapacity = 19
	modAttack         = 20
	modHide           = 21
	modInvisible      = 22
	modAccuracy       = 23
	modCount          = 24
)

// Buff is a server-tracked temporary stat modifier on an entity.
type Buff struct {
	Effect  int64
	Value   Variant
	SkillID int64
	Expires time.Time
}

// buffsOn returns the active buff list for a charID, pruning expired ones.
func (s *Server) buffsOn(charID int64) []*Buff {
	s.buffMu.Lock()
	defer s.buffMu.Unlock()
	now := time.Now()
	list := s.buffs[charID]
	kept := list[:0]
	for _, b := range list {
		if now.Before(b.Expires) {
			kept = append(kept, b)
		}
	}
	s.buffs[charID] = kept
	return kept
}

// applyBuff adds or refreshes a buff on an entity, then broadcasts EntityBuff
// (AOI) and UpdateBuff (to the entity itself, mirroring Server.UpdateBuff push).
func (s *Server) applyBuff(charID int64, effect, skillID int64, value Variant, duration time.Duration) {
	s.buffMu.Lock()
	list := s.buffs[charID]
	now := time.Now()
	replaced := false
	for i, b := range list {
		if b.Effect == effect && b.SkillID == skillID {
			list[i].Value = value
			list[i].Expires = now.Add(duration)
			replaced = true
			break
		}
	}
	if !replaced {
		list = append(list, &Buff{Effect: effect, Value: value, SkillID: skillID, Expires: now.Add(duration)})
	}
	s.buffs[charID] = list
	s.buffMu.Unlock()

	e := s.world.get(charID)
	if e == nil {
		return
	}
	// EntityBuff: broadcast FX to AOI (mirrors Network.EntityBuff).
	s.notifyNeighbours(e, rpcEntityBuff, []Variant{
		NewInt(charID), NewInt(effect), NewInt(skillID), NewBool(true),
	}, true)
	// UpdateBuff: the entity's own stat panel (mirrors Network.UpdateBuff).
	if oc := s.clientByPeer(e.PeerID); oc != nil {
		sendToClient(oc, nil, rpcUpdateBuff, []Variant{
			NewInt(effect), value, NewFloat(duration.Seconds()), NewInt(skillID),
		})
	}
}

// clearBuff removes a buff and notifies (duration=0 in SoM means clear).
func (s *Server) clearBuff(charID, effect, skillID int64) {
	s.buffMu.Lock()
	list := s.buffs[charID]
	kept := list[:0]
	for _, b := range list {
		if !(b.Effect == effect && b.SkillID == skillID) {
			kept = append(kept, b)
		}
	}
	s.buffs[charID] = kept
	s.buffMu.Unlock()

	e := s.world.get(charID)
	if e == nil {
		return
	}
	s.notifyNeighbours(e, rpcEntityBuff, []Variant{
		NewInt(charID), NewInt(effect), NewInt(skillID), NewBool(false),
	}, true)
	if oc := s.clientByPeer(e.PeerID); oc != nil {
		sendToClient(oc, nil, rpcUpdateBuff, []Variant{
			NewInt(effect), NewInt(0), NewFloat(0), NewInt(skillID),
		})
	}
}

// --- Ground drops ---

// Drop is a server-side ground drop (mirrors WorldDrop.gd).
type Drop struct {
	ID          int64
	ItemID      int64
	Customfield string
	Pos         Vec2
	MapID       int
	OwnerCharID int64
	Expires     time.Time
}

// spawnDrop creates a ground drop and broadcasts DropAdded to the map.
func (s *Server) spawnDrop(charID, itemID int64, customfield string, pos Vec2, mapID int) *Drop {
	s.dropMu.Lock()
	s.nextDropID++
	d := &Drop{
		ID:          s.nextDropID,
		ItemID:      itemID,
		Customfield: customfield,
		Pos:         pos,
		MapID:       mapID,
		OwnerCharID: charID,
		Expires:     time.Now().Add(dropLifetime),
	}
	s.drops[d.ID] = d
	s.dropMu.Unlock()
	s.broadcastToMap(d.MapID, rpcDropAdded, []Variant{
		NewInt(d.ID), NewInt(itemID), NewStringName(customfield), NewVec2(pos.X, pos.Y),
	})
	log.Printf("drop: spawn drop %d item=%d at (%.0f, %.0f)", d.ID, itemID, pos.X, pos.Y)
	return d
}

// removeDrop removes a ground drop and broadcasts DropRemoved.
func (s *Server) removeDrop(dropID int64) {
	s.dropMu.Lock()
	d := s.drops[dropID]
	delete(s.drops, dropID)
	s.dropMu.Unlock()
	if d != nil {
		s.broadcastToMap(d.MapID, rpcDropRemoved, []Variant{NewInt(dropID)})
		log.Printf("drop: remove drop %d", dropID)
	}
}

const dropLifetime = 60 * time.Second

// expireDrops removes ground drops whose lifetime has elapsed (WorldDrop.Timeout).
func (s *Server) expireDrops() {
	now := time.Now()
	var expired []int64
	s.dropMu.Lock()
	for id, d := range s.drops {
		if now.After(d.Expires) {
			expired = append(expired, id)
		}
	}
	s.dropMu.Unlock()
	for _, id := range expired {
		s.removeDrop(id)
	}
}

// --- Handlers ---

// handleRetrieveInventory implements Server.RetrieveInventory(peerID).
// SoM: client asks the server to resend its inventory/equipment.
func (s *Server) handleRetrieveInventory(c *Client, _ *ParsedRPC) {
	chr := s.charBySession(c)
	if chr == nil {
		return
	}
	sendToClient(c, nil, rpcRefreshInventory, []Variant{inventoryArray(chr)})
	sendToClient(c, nil, rpcRefreshEquipment, []Variant{equipmentDict(chr)})
}

// handlePushNotification implements Server.PushNotification(notif, peerID).
// Any_peer -> server; the server relays it to the target client.
func (s *Server) handlePushNotification(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 1 {
		return
	}
	notif := rpc.Args[0].Str
	// The second arg is the target peerID; if present, relay to that peer.
	targetPeerID := 0
	if len(rpc.Args) >= 2 {
		targetPeerID = int(rpc.Args[1].Int)
	}
	if targetPeerID > 0 {
		if tc := s.clientByPeer(targetPeerID); tc != nil {
			sendToClient(tc, nil, rpcPushNotification, []Variant{NewString(notif)})
		}
	} else {
		// No target: broadcast to AOI neighbours.
		e := s.world.get(c.session.charID)
		if e != nil {
			s.notifyNeighbours(e, rpcPushNotification, []Variant{NewString(notif)}, true)
		}
	}
	log.Printf("notif: peer %d -> %q (target=%d)", c.peerID, notif, targetPeerID)
}

// handleExpress implements Server.Express(agentRID, text, peerID).
// Broadcasts an expression bubble to AOI neighbours.
func (s *Server) handleExpress(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 2 {
		return
	}
	agentRID := rpc.Args[0].Int
	text := rpc.Args[1].Str
	e := s.world.get(agentRID)
	if e == nil {
		return
	}
	s.notifyNeighbours(e, rpcExpress, []Variant{NewInt(agentRID), NewString(text)}, true)
}

// handleThrowProjectile implements Server.ThrowProjectile(agentRID, targetPos,
// skillID, peerID). Broadcasts the projectile FX to AOI neighbours.
func (s *Server) handleThrowProjectile(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 3 {
		return
	}
	agentRID := rpc.Args[0].Int
	targetPos := rpc.Args[1].Vec2
	skillID := rpc.Args[2].Int
	e := s.world.get(agentRID)
	if e == nil {
		return
	}
	s.notifyNeighbours(e, rpcThrowProjectile, []Variant{
		NewInt(agentRID), NewVec2(targetPos[0], targetPos[1]), NewInt(skillID),
	}, true)
}

// handleMorphed implements Server.Morphed(agentRID, morphID, notifyMorphing,
// peerID). Broadcasts a morph state change to AOI neighbours.
func (s *Server) handleMorphed(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 3 {
		return
	}
	agentRID := rpc.Args[0].Int
	morphID := rpc.Args[1].Int
	notifyMorphing := rpc.Args[2].Bool
	e := s.world.get(agentRID)
	if e == nil {
		return
	}
	s.notifyNeighbours(e, rpcMorphed, []Variant{
		NewInt(agentRID), NewInt(morphID), NewBool(notifyMorphing),
	}, true)
}

// handleToggleContext implements Server.ToggleContext(enable, peerID).
func (s *Server) handleToggleContext(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 1 {
		return
	}
	log.Printf("context: peer %d toggle=%v", c.peerID, rpc.Args[0].Bool)
}

// handleClearNavigation implements Server.ClearNavigation(peerID).
// SoM: clears the player's current movement target.
func (s *Server) handleClearNavigation(c *Client, _ *ParsedRPC) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	e.Target = nil
	e.Vel = Vec2{}
	log.Printf("nav: peer %d cleared", c.peerID)
}

// --- Tutorial / UI handlers ---

// handleDisplayActions implements Server.DisplayActions(actions, peerID).
func (s *Server) handleDisplayActions(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 1 {
		return
	}
	log.Printf("ui: peer %d display actions (%d)", c.peerID, len(rpc.Args[0].Arr))
}

// handleDisplayProgressionTracker implements Server.DisplayProgressionTracker
// (label, value, maxValue, unit, peerID).
func (s *Server) handleDisplayProgressionTracker(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 4 {
		return
	}
	log.Printf("ui: peer %d tracker %q %d/%d %s", c.peerID,
		rpc.Args[0].Str, rpc.Args[1].Int, rpc.Args[2].Int, rpc.Args[3].Str)
}

// handleClearProgressionTracker implements Server.ClearProgressionTracker(peerID).
func (s *Server) handleClearProgressionTracker(c *Client, _ *ParsedRPC) {
	log.Printf("ui: peer %d clear tracker", c.peerID)
}

// handleHighlightUI implements Server.HighlightUI(target, peerID).
func (s *Server) handleHighlightUI(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 1 {
		return
	}
	log.Printf("ui: peer %d highlight %d", c.peerID, rpc.Args[0].Int)
}

// handleOpenUI implements Server.OpenUI(target, peerID).
func (s *Server) handleOpenUI(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 1 {
		return
	}
	log.Printf("ui: peer %d open %d", c.peerID, rpc.Args[0].Int)
}

// handleCommandModifier implements Server.CommandModifier(peerID).
// SoM sends this after TriggerCommand to modify command feedback.
func (s *Server) handleCommandModifier(c *Client, rpc *ParsedRPC) {
	log.Printf("cmd: peer %d modifier", c.peerID)
}

// --- Private stats sync ---

// pushPrivateStats sends UpdatePrivateStats to the character's own client.
// Called on connect and after XP/GP/mana/stamina changes.
func (s *Server) pushPrivateStats(charID int64) {
	chr := s.store.findCharacterByID(charID)
	if chr == nil {
		return
	}
	e := s.world.get(charID)
	if e == nil {
		return
	}
	oc := s.clientByPeer(e.PeerID)
	if oc == nil {
		return
	}
	sendToClient(oc, nil, rpcUpdatePrivateStats, []Variant{
		NewInt(chr.Experience), NewInt(chr.GP), NewInt(chr.Mana),
		NewInt(chr.Stamina), NewInt(chr.Karma), NewFloat(float64(chr.Weight)),
		NewInt(0), NewInt(0), // shape, spirit (unused for now)
	})
}
