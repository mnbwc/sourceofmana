package main

// Phase-3 gameplay: chat, emotes, and skill-cast notifications.
//
// Mirrors SoM's Network.gd notify model:
//   - LOCAL chat  -> NotifyNeighbours (AOI)
//   - GLOBAL chat -> NotifyGlobal (all in-game peers)
//   - PARTY chat  -> party members only (phase-3 later; falls back to system msg)
//   - TriggerEmote-> Emote to AOI neighbours, inclusive of the sender
//   - Casted      -> Casted to AOI neighbours (client displays the skill FX)

import (
	"log"
)

// chat channels (GUICommons.ChatChannel)
const (
	chatChannelLocal  = "LOCAL"
	chatChannelGlobal = "GLOBAL"
	chatChannelParty  = "PARTY"
)

// notifyNeighbours sends callback(nameID, args+peerID) to every in-game peer
// within AOI of the sender, optionally including the sender itself.
func (s *Server) notifyNeighbours(sender *Entity, nameID uint16, args []Variant, inclusive bool) {
	senderArgs := append(append([]Variant{}, args...), NewInt(int64(sender.PeerID)))
	if inclusive {
		if sc := s.clientByPeer(sender.PeerID); sc != nil {
			sendToClient(sc, nil, nameID, senderArgs)
		}
	}
	for _, other := range s.world.snapshot() {
		if other.CharID == sender.CharID {
			continue
		}
		if other.MapID != sender.MapID || sender.Pos.dist(other.Pos) > aoiRadius {
			continue
		}
		if oc := s.clientByPeer(other.PeerID); oc != nil {
			sendToClient(oc, nil, nameID, senderArgs)
		}
	}
}

// notifyGlobal sends to every in-game peer.
func (s *Server) notifyGlobal(nameID uint16, args []Variant) {
	for _, e := range s.world.snapshot() {
		if oc := s.clientByPeer(e.PeerID); oc != nil {
			peerArgs := append(append([]Variant{}, args...), NewInt(int64(e.PeerID)))
			sendToClient(oc, nil, nameID, peerArgs)
		}
	}
}

// broadcastToMap sends a push to every in-game peer on the given map.
func (s *Server) broadcastToMap(mapID int, nameID uint16, args []Variant) {
	for _, e := range s.world.snapshot() {
		if e.MapID != mapID {
			continue
		}
		if oc := s.clientByPeer(e.PeerID); oc != nil {
			peerArgs := append(append([]Variant{}, args...), NewInt(int64(e.PeerID)))
			sendToClient(oc, nil, nameID, peerArgs)
		}
	}
}

// ---- handlers ----

// handleTriggerChat mirrors Server.gd TriggerChat.
func (s *Server) handleTriggerChat(c *Client, rpc *ParsedRPC) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	channel := argString(rpc.Args, 0)
	text := argString(rpc.Args, 1)
	if text == "" {
		return
	}
	args := []Variant{NewString(channel), NewString(e.Name), NewString(text), NewInt(e.CharID)}
	switch channel {
	case chatChannelLocal:
		s.notifyNeighbours(e, rpcChatPlayer, args, true)
	case chatChannelGlobal:
		s.notifyGlobal(rpcChatPlayer, args)
	case chatChannelParty:
		// PARTY chat: deliver to every other member (mirrors Server.gd TriggerChat).
		s.party.mu.Lock()
		pid := s.party.partyID(e.CharID)
		var members []int64
		if pid != 0 {
			members = s.party.members(pid)
		}
		s.party.mu.Unlock()
		if pid == 0 {
			s.sendChatSystem(c, chatChannelLocal, "You are not in a party")
			break
		}
		for _, rid := range members {
			if rid == e.CharID {
				continue
			}
			me := s.world.get(rid)
			if me == nil {
				continue
			}
			if mc := s.clientByPeer(me.PeerID); mc != nil {
				sendToClient(mc, nil, rpcChatPlayer, []Variant{
					NewString(chatChannelParty), NewString(e.Name), NewString(text), NewInt(e.CharID),
					NewInt(int64(me.PeerID)),
				})
			}
		}
	default:
		log.Printf("peer %d chat: unknown channel %q", c.peerID, channel)
	}
	log.Printf("chat[%s] %s: %s", channel, e.Name, text)
}

// handleChatQuery mirrors Server.gd ChatQuery (requests the client's chat state).
func (s *Server) handleChatQuery(c *Client, rpc *ParsedRPC) {
	// The client sends ChatQuery to ask the server for pending system messages.
	// There are none in this stub; the client just refreshes its own state.
	channel := argString(rpc.Args, 0)
	log.Printf("peer %d ChatQuery channel=%s", c.peerID, channel)
}

// handleTriggerEmote mirrors Server.gd TriggerEmote: Emote to AOI neighbours
// inclusive of the sender.
func (s *Server) handleTriggerEmote(c *Client, rpc *ParsedRPC) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	emoteID := argInt(rpc.Args, 0)
	args := []Variant{NewInt(e.CharID), NewInt(emoteID)}
	s.notifyNeighbours(e, rpcEmote, args, true)
	log.Printf("emote: %s emoteID=%d", e.Name, emoteID)
}

// handleCasted mirrors Skill.gd Casted -> Network.NotifyNeighbours(agent, "Casted", ...).
// The server does not resolve damage here (no combat sim yet); it relays the
// cast so neighbours render the skill FX, exactly like SoM's notification layer.
func (s *Server) handleCasted(c *Client, rpc *ParsedRPC) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	skillID := argInt(rpc.Args, 1)
	cooldown := argFloat(rpc.Args, 2)
	args := []Variant{NewInt(e.CharID), NewInt(skillID), NewFloat(cooldown)}
	s.notifyNeighbours(e, rpcCasted, args, true)
	log.Printf("cast: %s skill=%d cooldown=%.2f", e.Name, skillID, cooldown)
}

func argInt(args []Variant, i int) int64 {
	if i < len(args) && args[i].Type == VariantInt {
		return args[i].Int
	}
	return 0
}

func argFloat(args []Variant, i int) float64 {
	if i < len(args) && args[i].Type == VariantFloat {
		return args[i].Float
	}
	return 0
}
