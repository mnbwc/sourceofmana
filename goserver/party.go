package main

// Phase-3 party system.
//
// Mirrors SoM's server-side party flow (sources/network/server/PartyService.gd,
// sources/network/server/Server.gd PartyInvite/PartyAccept/PartyDecline/
// PartyLeave/PartyKick/PartyPromote):
//
//   TriggerPartyInvite(nick)  ->  PartyInvite(inviterName) to the target
//   TriggerPartyAccept        ->  join the inviter's party, PartyUpdate to all
//   TriggerPartyDecline       ->  drop the pending invite
//   TriggerPartyLeave         ->  leave, PartyUpdate to remaining members
//   TriggerPartyKick(nick)    ->  leader kicks a member, PartyUpdate to all
//   TriggerPartyPromote(nick) ->  leader promotes a member, PartyUpdate to all
//
// PartyUpdate carries the member RIDs as a JSON array string (the client does
// JSON.parse_string(members)), plus the leader RID.

import (
	"encoding/json"
	"log"
	"sync"
)

// maxPartyMembers mirrors ActorCommons.MaxCharacterCount.
const maxPartyMembers = 4

// party is one party: a leader RID plus a member set.
type party struct {
	Leader  int64
	Members map[int64]bool
}

// partyService holds all parties and pending invites.
type partyService struct {
	mu       sync.Mutex
	parties  map[int]*party  // pid -> party
	ridToPID map[int64]int   // charID (agentRID) -> pid
	invites  map[int64]int64 // target charID -> inviter charID
	nextID   int
}

func newPartyService() *partyService {
	return &partyService{
		parties:  make(map[int]*party),
		ridToPID: make(map[int64]int),
		invites:  make(map[int64]int64),
		nextID:   1,
	}
}

// createParty mirrors PartyService.CreateParty.
func (ps *partyService) createParty(leaderRID int64) int {
	if _, ok := ps.ridToPID[leaderRID]; ok {
		return 0
	}
	pid := ps.nextID
	ps.nextID++
	ps.parties[pid] = &party{Leader: leaderRID, Members: map[int64]bool{leaderRID: true}}
	ps.ridToPID[leaderRID] = pid
	return pid
}

// disband mirrors PartyService.Disband.
func (ps *partyService) disband(pid int) {
	p, ok := ps.parties[pid]
	if !ok {
		return
	}
	for rid := range p.Members {
		delete(ps.ridToPID, rid)
	}
	delete(ps.parties, pid)
}

// addInvite mirrors PartyService.AddInvite.
func (ps *partyService) addInvite(targetRID, inviterRID int64) bool {
	if _, ok := ps.ridToPID[targetRID]; ok {
		return false // already in a party
	}
	ps.invites[targetRID] = inviterRID
	return true
}

// popInvite mirrors PartyService.PopInvite.
func (ps *partyService) popInvite(targetRID int64) int64 {
	inviter := ps.invites[targetRID]
	delete(ps.invites, targetRID)
	return inviter
}

// join mirrors PartyService.Join.
func (ps *partyService) join(pid int, rid int64) bool {
	p, ok := ps.parties[pid]
	if !ok {
		return false
	}
	if _, ok := ps.ridToPID[rid]; ok {
		return false
	}
	if len(p.Members) >= maxPartyMembers {
		return false
	}
	p.Members[rid] = true
	ps.ridToPID[rid] = pid
	return true
}

// leave mirrors PartyService.Leave. Returns the pid (0 if the rid was not in a
// party). The party is disbanded if empty; leadership transfers if the leader
// left.
func (ps *partyService) leave(rid int64) int {
	pid, ok := ps.ridToPID[rid]
	if !ok {
		return 0
	}
	p := ps.parties[pid]
	delete(p.Members, rid)
	delete(ps.ridToPID, rid)
	if len(p.Members) == 0 {
		ps.disband(pid)
	} else if p.Leader == rid {
		for m := range p.Members {
			p.Leader = m
			break
		}
	}
	return pid
}

// kick mirrors PartyService.Kick.
func (ps *partyService) kick(pid int, leaderRID, targetRID int64) bool {
	p, ok := ps.parties[pid]
	if !ok || p.Leader != leaderRID {
		return false
	}
	if !p.Members[targetRID] || targetRID == leaderRID {
		return false
	}
	return ps.leave(targetRID) == pid
}

// promote mirrors PartyService.Promote.
func (ps *partyService) promote(pid int, leaderRID, targetRID int64) bool {
	p, ok := ps.parties[pid]
	if !ok || p.Leader != leaderRID {
		return false
	}
	if !p.Members[targetRID] {
		return false
	}
	p.Leader = targetRID
	return true
}

// partyID mirrors PartyService.GetPartyID.
func (ps *partyService) partyID(rid int64) int {
	return ps.ridToPID[rid]
}

// members mirrors PartyService.GetMembers (returns charIDs in stable order).
func (ps *partyService) members(pid int) []int64 {
	p, ok := ps.parties[pid]
	if !ok {
		return nil
	}
	out := make([]int64, 0, len(p.Members))
	for rid := range p.Members {
		out = append(out, rid)
	}
	return out
}

// leader mirrors PartyService.GetLeader.
func (ps *partyService) leader(pid int) int64 {
	if p, ok := ps.parties[pid]; ok {
		return p.Leader
	}
	return 0
}

// ---- server handlers ----

// handleTriggerPartyInvite mirrors Server.gd PartyInvite.
func (s *Server) handleTriggerPartyInvite(c *Client, rpc *ParsedRPC) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	targetNick := argString(rpc.Args, 0)
	target := s.world.byName(targetNick)
	if target == nil || target.CharID == e.CharID {
		s.sendChatSystem(c, chatChannelLocal, "Player '"+targetNick+"' is not online")
		return
	}
	s.party.mu.Lock()
	ok := s.party.addInvite(target.CharID, e.CharID)
	s.party.mu.Unlock()
	if !ok {
		s.sendChatSystem(c, chatChannelLocal, targetNick+" is already in a party")
		return
	}
	if tc := s.clientByPeer(target.PeerID); tc != nil {
		sendToClient(tc, nil, rpcPartyInvite, []Variant{NewString(e.Name), NewInt(int64(target.PeerID))})
	}
	log.Printf("party: %s invited %s", e.Name, targetNick)
}

// handleTriggerPartyAccept mirrors Server.gd PartyAccept.
func (s *Server) handleTriggerPartyAccept(c *Client, rpc *ParsedRPC) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	s.party.mu.Lock()
	inviterRID := s.party.popInvite(e.CharID)
	var inviter *Entity
	if inviterRID != 0 {
		inviter = s.world.get(inviterRID)
	}
	var pid int
	if inviter != nil {
		pid = s.party.partyID(inviterRID)
		if pid == 0 {
			pid = s.party.createParty(inviterRID)
		}
		joined := s.party.join(pid, e.CharID)
		if !joined {
			s.party.mu.Unlock()
			return
		}
	} else {
		s.party.mu.Unlock()
		return
	}
	s.party.mu.Unlock()
	s.notifyPartyMembers(pid)
	log.Printf("party: %s accepted invite from char %d (pid=%d)", e.Name, inviterRID, pid)
}

// handleTriggerPartyDecline mirrors Server.gd PartyDecline.
func (s *Server) handleTriggerPartyDecline(c *Client, rpc *ParsedRPC) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	s.party.mu.Lock()
	s.party.popInvite(e.CharID)
	s.party.mu.Unlock()
	log.Printf("party: %s declined invite", e.Name)
}

// handleTriggerPartyLeave mirrors Server.gd PartyLeave.
func (s *Server) handleTriggerPartyLeave(c *Client, rpc *ParsedRPC) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	s.party.mu.Lock()
	pid := s.party.leave(e.CharID)
	s.party.mu.Unlock()
	if pid != 0 {
		s.notifyPartyMembers(pid)
	}
	log.Printf("party: %s left (pid=%d)", e.Name, pid)
}

// handleTriggerPartyKick mirrors Server.gd PartyKick.
func (s *Server) handleTriggerPartyKick(c *Client, rpc *ParsedRPC) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	targetNick := argString(rpc.Args, 0)
	target := s.world.byName(targetNick)
	if target == nil {
		return
	}
	s.party.mu.Lock()
	pid := s.party.partyID(e.CharID)
	ok := s.party.kick(pid, e.CharID, target.CharID)
	s.party.mu.Unlock()
	if ok {
		s.notifyPartyMembers(pid)
	}
	log.Printf("party: %s kicked %s (pid=%d ok=%v)", e.Name, targetNick, pid, ok)
}

// handleTriggerPartyPromote mirrors Server.gd PartyPromote.
func (s *Server) handleTriggerPartyPromote(c *Client, rpc *ParsedRPC) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	targetNick := argString(rpc.Args, 0)
	target := s.world.byName(targetNick)
	if target == nil {
		return
	}
	s.party.mu.Lock()
	pid := s.party.partyID(e.CharID)
	ok := s.party.promote(pid, e.CharID, target.CharID)
	s.party.mu.Unlock()
	if ok {
		s.notifyPartyMembers(pid)
	}
	log.Printf("party: %s promoted %s (pid=%d ok=%v)", e.Name, targetNick, pid, ok)
}

// notifyPartyMembers mirrors PartyNotifyMembers: send PartyUpdate to every
// member (members is a JSON array string of RIDs; the client parses it).
func (s *Server) notifyPartyMembers(pid int) {
	s.party.mu.Lock()
	members := s.party.members(pid)
	leader := s.party.leader(pid)
	s.party.mu.Unlock()
	if len(members) == 0 {
		return
	}
	raw, err := json.Marshal(members)
	if err != nil {
		return
	}
	for _, rid := range members {
		te := s.world.get(rid)
		if te == nil {
			continue
		}
		if mc := s.clientByPeer(te.PeerID); mc != nil {
			sendToClient(mc, nil, rpcPartyUpdate, []Variant{
				NewString(string(raw)), NewInt(leader), NewInt(int64(te.PeerID)),
			})
		}
	}
}

// sendChatSystem mirrors Network.ChatSystem(channel, text, peerID).
func (s *Server) sendChatSystem(c *Client, channel, text string) {
	sendToClient(c, nil, rpcChatSystem, []Variant{
		NewString(channel), NewString(text), NewInt(int64(c.peerID)),
	})
}
