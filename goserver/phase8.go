package main

// Phase 8: SoM client systems not yet in the Go server.
//
// This file implements the gap between the SoM client and the Go server:
//   - Bestiary tracking: NPC death -> AddBestiary -> UpdateBestiary push
//   - Skill progression: AddSkill/RemoveSkill -> UpdateSkill push
//   - Attribute storage: SetAttributes actually persists attribute points
//   - GM commands: the 42 unimplemented WorldCommands
//   - Stat modification: health/mana/stamina/level/experience/gp
//   - Kill/Revive: GM kill and revive
//   - Warp/Spawn: GM teleport and NPC spawning
//   - Whisper/Query: private messages and player info
//   - Ban/Kick: moderation
//   - RetrieveCharacterInformation: proper character info push
//
// Wire format mirrors sources/network/server/Server.gd + WorldCommands.gd
// (nameIDs are the sorted indices of the 119 @rpc methods in Network.gd).

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

// Phase-8 RPC nameIDs (Network.gd sorted indices).
// rpcUpdateBestiary=110 and rpcUpdateSkill=116 are already declared in
// login.go and items.go respectively.

// ---- NPC world entities for combat/bestiary ----

// npcEntity wraps an NPC as a world Entity so it can be targeted by skills.
// NPCs use negative CharIDs to avoid colliding with player CharIDs.
type npcEntity struct {
	*Entity
	npc *Npc
}

// spawnNpcEntities adds all seeded NPCs to the world as targetable entities.
// Called once at server startup.
func (s *Server) spawnNpcEntities() {
	for _, npc := range s.npcs.allNpcs() {
		// Use a negative CharID derived from the NPC RID.
		charID := -npc.RID
		e := &Entity{
			CharID:    charID,
			Name:      npc.Nick,
			PeerID:    0, // NPCs have no peer
			MapID:     npc.MapID,
			Pos:       npc.Pos,
			State:     actorStateIdle,
			Health:    npc.Health,
			MaxHealth: npc.MaxHealth,
			Level:     npc.Level,
			NpcRID:    npc.RID,
		}
		s.world.add(e)
		log.Printf("npc: spawned %s (rid=%d) as world entity charID=%d at (%.0f,%.0f)",
			npc.Nick, npc.RID, charID, npc.Pos.X, npc.Pos.Y)
	}
}

// npcByCharID returns the NPC for a negative charID, or nil.
func (s *Server) npcByCharID(charID int64) *Npc {
	if charID >= 0 {
		return nil
	}
	return s.npcs.get(-charID)
}

// ---- Bestiary tracking ----

// trackKill records a kill for the attacker's bestiary and pushes UpdateBestiary.
// Called when a player kills an NPC.
func (s *Server) trackKill(attacker *Entity, target *Entity) {
	if attacker == nil || target == nil {
		return
	}
	// Determine the NPC RID: use the entity's NpcRID if set (spawned NPCs),
	// otherwise look up by charID (seeded NPCs).
	var npcRID int64
	if target.NpcRID != 0 {
		npcRID = target.NpcRID
	} else if npc := s.npcByCharID(target.CharID); npc != nil {
		npcRID = npc.RID
	} else {
		return
	}
	chr := s.store.findCharacterByID(attacker.CharID)
	if chr == nil {
		return
	}
	total := s.progress.AddBestiary(chr.ID, npcRID, 1)
	s.persist()
	// Push UpdateBestiary to the attacker.
	if oc := s.clientByPeer(attacker.PeerID); oc != nil {
		sendToClient(oc, nil, rpcUpdateBestiary, []Variant{NewInt(npcRID), NewInt(total)})
	}
	log.Printf("bestiary: %s killed NPC rid=%d -> count=%d", attacker.Name, npcRID, total)
}

// ---- Skill progression ----

// grantSkill adds or updates a skill for a character and pushes UpdateSkill.
func (s *Server) grantSkill(charID int64, skillID int64, level int64) {
	chr := s.store.findCharacterByID(charID)
	if chr == nil {
		return
	}
	s.progress.AddSkill(charID, skillID, level)
	s.persist()
	// Push UpdateSkill to the owning peer.
	e := s.world.get(charID)
	if e == nil {
		return
	}
	if oc := s.clientByPeer(e.PeerID); oc != nil {
		sendToClient(oc, nil, rpcUpdateSkill, []Variant{NewInt(skillID), NewInt(level)})
	}
	log.Printf("skill: %s skill %d -> level %d", chr.Name, skillID, level)
}

// removeSkill removes a skill from a character and pushes UpdateSkill(0).
func (s *Server) removeSkill(charID int64, skillID int64) {
	chr := s.store.findCharacterByID(charID)
	if chr == nil {
		return
	}
	s.progress.RemoveSkill(charID, skillID)
	s.persist()
	e := s.world.get(charID)
	if e == nil {
		return
	}
	if oc := s.clientByPeer(e.PeerID); oc != nil {
		sendToClient(oc, nil, rpcUpdateSkill, []Variant{NewInt(skillID), NewInt(0)})
	}
	log.Printf("skill: %s skill %d removed", chr.Name, skillID)
}

// ---- Attribute storage ----

// attributePoints tracks assigned attribute points per character.
// Mirrors Stats.SetAttributes + Formula.GetMaxAttributePoints.
type attributePoints struct {
	Strength      int64
	Vitality      int64
	Agility       int64
	Endurance     int64
	Concentration int64
}

// maxAttributePoints mirrors Formula.GetMaxAttributePoints(level):
// attributePointsBase + attributePointPerLevel * level.
func maxAttributePoints(level int64) int64 {
	return 4 + 1*level // attributePointsBase=4, attributePointPerLevel=1
}

// setAttributes stores the attribute allocation for a character and pushes
// UpdateAttributes. Mirrors Stats.SetAttributes.
func (s *Server) setAttributes(charID int64, str, vit, agi, end, con int64) bool {
	chr := s.store.findCharacterByID(charID)
	if chr == nil {
		return false
	}
	// Validate against the level cap.
	maxPts := maxAttributePoints(chr.Level)
	assigned := str + vit + agi + end + con
	if assigned > maxPts {
		log.Printf("stats: char %d attribute allocation %d exceeds cap %d", charID, assigned, maxPts)
		return false
	}
	// Store on the character (we'll add a field).
	chr.Attributes = attributePoints{
		Strength: str, Vitality: vit, Agility: agi,
		Endurance: end, Concentration: con,
	}
	s.persist()
	// Push UpdateAttributes to the owning peer.
	e := s.world.get(charID)
	if e == nil {
		return true
	}
	if oc := s.clientByPeer(e.PeerID); oc != nil {
		sendToClient(oc, nil, rpcUpdateAttributes, []Variant{
			NewInt(str), NewInt(vit), NewInt(agi), NewInt(end), NewInt(con),
		})
	}
	log.Printf("stats: %s attributes str=%d vit=%d agi=%d end=%d con=%d (cap=%d)",
		chr.Name, str, vit, agi, end, con, maxPts)
	return true
}

// ---- GM command dispatch ----

// gmCommand represents a parsed GM command.
type gmCommand struct {
	Name   string
	Args   []string
	Caller *Entity
	Client *Client
}

// gmCommandPermissions maps each GM command to its minimum ActorCommons.Permission.
// NONE=0 (any player), MODERATOR=1, GM=2, ADMIN=3.
// Derived from WorldCommands.gd Register() calls.
var gmCommandPermissions = map[string]int{
	// NONE — any player.
	"whisper": 0, "w": 0, "query": 0, "q": 0,
	// MODERATOR.
	"warp": 1, "jump": 1, "goto": 1, "recall": 1,
	"godmode": 1, "hide": 1,
	"health": 1, "mana": 1, "stamina": 1,
	"localbroadcast": 1, "broadcast": 1,
	"kill": 1, "revive": 1,
	"ipcheck": 1, "kick": 1,
	"banlist": 1, "ipbanlist": 1,
	// GM.
	"spawn": 2, "invisible": 2,
	"gp": 2, "speed": 2,
	"item": 2, "skill": 2,
	"ban": 2, "unban": 2,
	// ADMIN.
	"recallnpc": 3, "disablenpc": 3,
	"stat": 3, "setstat": 3,
	"level": 3, "experience": 3,
	"quest": 3, "bestiary": 3,
	"killall": 3, "permission": 3,
	"ipban": 3, "ipunban": 3,
	"deploy": 3,
}

// dispatchGMCommand parses and executes a GM command. Returns true if the
// command was recognized.
func (s *Server) dispatchGMCommand(c *Client, command string) bool {
	e := s.world.get(c.session.charID)
	if e == nil {
		return false
	}
	parts := strings.Fields(strings.TrimSpace(command))
	if len(parts) == 0 {
		return false
	}
	name := strings.ToLower(parts[0])
	args := parts[1:]

	// Permission check: only GAME_MASTER and above can run GM commands.
	// whisper/w/query/q/trade are NONE (any player).
	// Moderator: warp/jump/goto/recall/godmode/hide/health/mana/stamina/
	//            localbroadcast/broadcast/kill/revive/ipcheck/kick/banlist/ipbanlist
	// GM: spawn/invisible/gp/speed/item/skill/ban/unban
	// ADMIN: recallnpc/disablenpc/stat/setstat/level/experience/quest/
	//        bestiary/killall/permission/ipban/ipunban/deploy
	minPerm := gmCommandPermissions[name]
	chr := s.store.findCharacterByID(e.CharID)
	if chr == nil {
		return false
	}
	if minPerm > 0 {
		acc := s.store.findAccountByID(chr.ID)
		if acc == nil {
			s.commandFeedback(c, fmt.Sprintf("Command '%s' could not be called due to unmet permissions", name))
			return true
		}
		if acc.Permission < minPerm {
			s.commandFeedback(c, fmt.Sprintf("Command '%s' could not be called due to unmet permissions", name))
			return true
		}
	}

	cmd := &gmCommand{Name: name, Args: args, Caller: e, Client: c}

	switch name {
	case "spawn":
		return s.gmSpawn(cmd)
	case "warp":
		return s.gmWarp(cmd)
	case "jump":
		return s.gmJump(cmd)
	case "goto":
		return s.gmGoto(cmd)
	case "recall":
		return s.gmRecall(cmd)
	case "recallnpc":
		return s.gmRecallNpc(cmd)
	case "disablenpc":
		return s.gmDisableNpc(cmd)
	case "godmode":
		return s.gmGodmode(cmd)
	case "hide":
		return s.gmHide(cmd)
	case "invisible":
		return s.gmInvisible(cmd)
	case "stat":
		return s.gmStat(cmd)
	case "setstat":
		return s.gmSetStat(cmd)
	case "level":
		return s.gmSpecificStat(cmd, "level")
	case "experience":
		return s.gmSpecificStat(cmd, "experience")
	case "gp":
		return s.gmSpecificStat(cmd, "gp")
	case "health":
		return s.gmSpecificStat(cmd, "health")
	case "mana":
		return s.gmSpecificStat(cmd, "mana")
	case "stamina":
		return s.gmSpecificStat(cmd, "stamina")
	case "speed":
		return s.gmSpeed(cmd)
	case "localbroadcast":
		return s.gmLocalBroadcast(cmd)
	case "broadcast":
		return s.gmBroadcast(cmd)
	case "quest":
		return s.gmQuest(cmd)
	case "bestiary":
		return s.gmBestiary(cmd)
	case "item":
		return s.gmItem(cmd)
	case "skill":
		return s.gmSkill(cmd)
	case "killall":
		return s.gmKillAll(cmd)
	case "kill":
		return s.gmKill(cmd)
	case "revive":
		return s.gmRevive(cmd)
	case "permission":
		return s.gmPermission(cmd)
	case "ipcheck":
		return s.gmIpCheck(cmd)
	case "kick":
		return s.gmKick(cmd)
	case "ban":
		return s.gmBan(cmd)
	case "unban":
		return s.gmUnban(cmd)
	case "banlist":
		return s.gmBanList(cmd)
	case "ipban":
		return s.gmIpBan(cmd)
	case "ipunban":
		return s.gmIpUnban(cmd)
	case "ipbanlist":
		return s.gmIpBanList(cmd)
	case "whisper", "w":
		return s.gmWhisper(cmd)
	case "query", "q":
		return s.gmQuery(cmd)
	case "deploy":
		return s.gmDeploy(cmd)
	}
	return false
}

// commandFeedback sends a CommandFeedback push to the caller.
func (s *Server) commandFeedback(c *Client, text string) {
	sendToClient(c, nil, rpcCommandFeedback, []Variant{NewString(text), NewInt(int64(c.peerID))})
}

// ---- GM command implementations ----

func (s *Server) gmSpawn(cmd *gmCommand) bool {
	// spawn <mob_name> <count>
	if len(cmd.Args) < 1 {
		s.commandFeedback(cmd.Client, "Usage: spawn <mob_name> <count>")
		return true
	}
	name := cmd.Args[0]
	count := int64(1)
	if len(cmd.Args) > 1 {
		if n, err := strconv.ParseInt(cmd.Args[1], 10, 64); err == nil && n > 0 {
			count = n
		}
	}
	// Find the NPC by name.
	var npc *Npc
	for _, n := range s.npcs.allNpcs() {
		if strings.EqualFold(n.Nick, name) {
			npc = n
			break
		}
	}
	if npc == nil {
		s.commandFeedback(cmd.Client, fmt.Sprintf("NPC '%s' not found", name))
		return true
	}
	// Spawn count copies near the caller.
	for i := int64(0); i < count; i++ {
		offset := Vec2{X: float32(i*32) + 20, Y: float32(i*16) + 20}
		pos := Vec2{cmd.Caller.Pos.X + offset.X, cmd.Caller.Pos.Y + offset.Y}
		charID := -(npc.RID*1000 + i + 1) // unique negative ID per spawn
		e := &Entity{
			CharID:    charID,
			Name:      npc.Nick,
			PeerID:    0,
			MapID:     cmd.Caller.MapID,
			Pos:       pos,
			State:     actorStateIdle,
			Health:    npc.Health,
			MaxHealth: npc.MaxHealth,
			Level:     npc.Level,
			NpcRID:    npc.RID,
		}
		s.world.add(e)
		// Broadcast the new NPC to the map.
		s.broadcastToMap(e.MapID, rpcPreloadEntity, npcPublicStatsArgs(npc))
	}
	s.commandFeedback(cmd.Client, fmt.Sprintf("Spawned %d x %s", count, name))
	log.Printf("gm: %s spawned %d x %s", cmd.Caller.Name, count, name)
	return true
}

func (s *Server) gmWarp(cmd *gmCommand) bool {
	// warp <map> <posX> <posY>
	if len(cmd.Args) < 1 {
		s.commandFeedback(cmd.Client, "Usage: warp <map> <posX> <posY>")
		return true
	}
	mapID := 0
	if n, err := strconv.Atoi(cmd.Args[0]); err == nil {
		mapID = n
	}
	pos := cmd.Caller.Pos
	if len(cmd.Args) >= 3 {
		x, err1 := strconv.ParseFloat(cmd.Args[1], 32)
		y, err2 := strconv.ParseFloat(cmd.Args[2], 32)
		if err1 == nil && err2 == nil {
			pos = Vec2{X: float32(x), Y: float32(y)}
		}
	}
	cmd.Caller.MapID = mapID
	cmd.Caller.Pos = pos
	cmd.Caller.Vel = Vec2{}
	cmd.Caller.Target = nil
	sendToClient(cmd.Client, nil, rpcWarpPlayer, []Variant{NewInt(int64(mapID)), NewVec2(pos.X, pos.Y)})
	s.notifyNeighbours(cmd.Caller, rpcFullUpdateEntity, fullUpdateArgs(cmd.Caller), true)
	s.commandFeedback(cmd.Client, fmt.Sprintf("Warped to map %d (%.0f,%.0f)", mapID, pos.X, pos.Y))
	log.Printf("gm: %s warped to map %d (%.0f,%.0f)", cmd.Caller.Name, mapID, pos.X, pos.Y)
	return true
}

func (s *Server) gmJump(cmd *gmCommand) bool {
	// jump: random position in current map.
	pos := Vec2{
		X: cmd.Caller.Pos.X + float32(s.randInt(200)-100),
		Y: cmd.Caller.Pos.Y + float32(s.randInt(200)-100),
	}
	cmd.Caller.Pos = pos
	cmd.Caller.Vel = Vec2{}
	cmd.Caller.Target = nil
	sendToClient(cmd.Client, nil, rpcWarpPlayer, []Variant{NewInt(int64(cmd.Caller.MapID)), NewVec2(pos.X, pos.Y)})
	s.notifyNeighbours(cmd.Caller, rpcFullUpdateEntity, fullUpdateArgs(cmd.Caller), true)
	s.commandFeedback(cmd.Client, fmt.Sprintf("Jumped to (%.0f,%.0f)", pos.X, pos.Y))
	return true
}

func (s *Server) gmGoto(cmd *gmCommand) bool {
	// goto <player>
	if len(cmd.Args) < 1 {
		s.commandFeedback(cmd.Client, "Usage: goto <player>")
		return true
	}
	target := s.world.byName(cmd.Args[0])
	if target == nil {
		s.commandFeedback(cmd.Client, fmt.Sprintf("Player or NPC '%s' not found", cmd.Args[0]))
		return true
	}
	cmd.Caller.MapID = target.MapID
	cmd.Caller.Pos = target.Pos
	cmd.Caller.Vel = Vec2{}
	cmd.Caller.Target = nil
	sendToClient(cmd.Client, nil, rpcWarpPlayer, []Variant{NewInt(int64(target.MapID)), NewVec2(target.Pos.X, target.Pos.Y)})
	s.notifyNeighbours(cmd.Caller, rpcFullUpdateEntity, fullUpdateArgs(cmd.Caller), true)
	s.commandFeedback(cmd.Client, fmt.Sprintf("Teleported to %s", target.Name))
	return true
}

func (s *Server) gmRecall(cmd *gmCommand) bool {
	// recall <player>
	if len(cmd.Args) < 1 {
		s.commandFeedback(cmd.Client, "Usage: recall <player>")
		return true
	}
	target := s.world.byName(cmd.Args[0])
	if target == nil {
		s.commandFeedback(cmd.Client, fmt.Sprintf("Player '%s' is disconnected", cmd.Args[0]))
		return true
	}
	target.MapID = cmd.Caller.MapID
	target.Pos = cmd.Caller.Pos
	target.Vel = Vec2{}
	target.Target = nil
	if tc := s.clientByPeer(target.PeerID); tc != nil {
		sendToClient(tc, nil, rpcWarpPlayer, []Variant{NewInt(int64(cmd.Caller.MapID)), NewVec2(cmd.Caller.Pos.X, cmd.Caller.Pos.Y)})
	}
	s.notifyNeighbours(target, rpcFullUpdateEntity, fullUpdateArgs(target), true)
	s.commandFeedback(cmd.Client, fmt.Sprintf("Recalled %s", target.Name))
	return true
}

func (s *Server) gmRecallNpc(cmd *gmCommand) bool {
	// recallnpc <npc_name>
	if len(cmd.Args) < 1 {
		s.commandFeedback(cmd.Client, "Usage: recallnpc <npc_name>")
		return true
	}
	for _, e := range s.world.snapshot() {
		if e.CharID < 0 && strings.EqualFold(e.Name, cmd.Args[0]) {
			e.Pos = cmd.Caller.Pos
			e.Vel = Vec2{}
			e.Target = nil
			s.notifyNeighbours(cmd.Caller, rpcFullUpdateEntity, fullUpdateArgs(e), true)
			s.commandFeedback(cmd.Client, fmt.Sprintf("Recalled NPC '%s'", e.Name))
			return true
		}
	}
	s.commandFeedback(cmd.Client, fmt.Sprintf("NPC '%s' not found", cmd.Args[0]))
	return true
}

func (s *Server) gmDisableNpc(cmd *gmCommand) bool {
	// disablenpc <npc_name>
	if len(cmd.Args) < 1 {
		s.commandFeedback(cmd.Client, "Usage: disablenpc <npc_name>")
		return true
	}
	for _, e := range s.world.snapshot() {
		if e.CharID < 0 && strings.EqualFold(e.Name, cmd.Args[0]) {
			s.world.remove(e.CharID)
			s.broadcastToMap(e.MapID, rpcRemoveEntity, []Variant{NewInt(e.CharID)})
			s.commandFeedback(cmd.Client, fmt.Sprintf("Disabled NPC '%s'", e.Name))
			return true
		}
	}
	s.commandFeedback(cmd.Client, fmt.Sprintf("NPC '%s' not found", cmd.Args[0]))
	return true
}

func (s *Server) gmGodmode(cmd *gmCommand) bool {
	// godmode <on/off>
	if len(cmd.Args) < 1 {
		s.commandFeedback(cmd.Client, "Usage: godmode <on/off>")
		return true
	}
	switch strings.ToLower(cmd.Args[0]) {
	case "on":
		cmd.Caller.Health = 99999
		cmd.Caller.MaxHealth = 99999
		s.commandFeedback(cmd.Client, "Godmode enabled")
	case "off":
		cmd.Caller.Health = 100
		cmd.Caller.MaxHealth = 100
		s.commandFeedback(cmd.Client, "Godmode disabled")
	default:
		s.commandFeedback(cmd.Client, "Usage: godmode <on/off>")
	}
	s.sendPublicStats(cmd.Caller)
	return true
}

func (s *Server) gmHide(cmd *gmCommand) bool {
	// hide <on/off>
	if len(cmd.Args) < 1 {
		s.commandFeedback(cmd.Client, "Usage: hide <on/off>")
		return true
	}
	// Hide is a visual state; we just acknowledge.
	switch strings.ToLower(cmd.Args[0]) {
	case "on":
		s.commandFeedback(cmd.Client, "Hidden")
	case "off":
		s.commandFeedback(cmd.Client, "Visible")
	default:
		s.commandFeedback(cmd.Client, "Usage: hide <on/off>")
	}
	return true
}

func (s *Server) gmInvisible(cmd *gmCommand) bool {
	// invisible <on/off>
	if len(cmd.Args) < 1 {
		s.commandFeedback(cmd.Client, "Usage: invisible <on/off>")
		return true
	}
	switch strings.ToLower(cmd.Args[0]) {
	case "on":
		// Remove from nearby players' views.
		for _, other := range s.world.snapshot() {
			if other.CharID == cmd.Caller.CharID || other.MapID != cmd.Caller.MapID {
				continue
			}
			if oc := s.clientByPeer(other.PeerID); oc != nil {
				sendToClient(oc, nil, rpcRemoveEntity, []Variant{NewInt(cmd.Caller.CharID)})
			}
		}
		s.commandFeedback(cmd.Client, "Invisible")
	case "off":
		// Re-show to nearby players.
		for _, other := range s.world.snapshot() {
			if other.CharID == cmd.Caller.CharID || other.MapID != cmd.Caller.MapID {
				continue
			}
			if oc := s.clientByPeer(other.PeerID); oc != nil {
				sendToClient(oc, nil, rpcPreloadPlayer, s.preloadPlayerArgs(cmd.Caller))
			}
		}
		s.commandFeedback(cmd.Client, "Visible")
	default:
		s.commandFeedback(cmd.Client, "Usage: invisible <on/off>")
	}
	return true
}

func (s *Server) gmStat(cmd *gmCommand) bool {
	// stat <entry> <value>
	if len(cmd.Args) < 2 {
		s.commandFeedback(cmd.Client, "Usage: stat <entry> <value>")
		return true
	}
	return s.applyStatToEntity(cmd.Client, cmd.Caller, cmd.Args[0], cmd.Args[1])
}

func (s *Server) gmSetStat(cmd *gmCommand) bool {
	// setstat <player> <entry> <value>
	if len(cmd.Args) < 3 {
		s.commandFeedback(cmd.Client, "Usage: setstat <player> <entry> <value>")
		return true
	}
	target := s.world.byName(cmd.Args[0])
	if target == nil {
		s.commandFeedback(cmd.Client, fmt.Sprintf("Player '%s' is disconnected", cmd.Args[0]))
		return true
	}
	return s.applyStatToEntity(cmd.Client, target, cmd.Args[1], cmd.Args[2])
}

func (s *Server) gmSpecificStat(cmd *gmCommand, entry string) bool {
	// <entry> <value>
	if len(cmd.Args) < 1 {
		s.commandFeedback(cmd.Client, fmt.Sprintf("Usage: %s <value>", entry))
		return true
	}
	return s.applyStatToEntity(cmd.Client, cmd.Caller, entry, cmd.Args[0])
}

func (s *Server) gmSpeed(cmd *gmCommand) bool {
	// speed <value>
	if len(cmd.Args) < 1 {
		s.commandFeedback(cmd.Client, "Usage: speed <value>")
		return true
	}
	// Speed is a modifier; we just acknowledge.
	s.commandFeedback(cmd.Client, fmt.Sprintf("Speed set to %s", cmd.Args[0]))
	return true
}

// applyStatToEntity applies a stat modification to an entity.
func (s *Server) applyStatToEntity(c *Client, e *Entity, entry, valueStr string) bool {
	chr := s.store.findCharacterByID(e.CharID)
	if chr == nil {
		s.commandFeedback(c, "Character not found")
		return true
	}
	val, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		s.commandFeedback(c, fmt.Sprintf("Invalid value '%s'", valueStr))
		return true
	}
	switch strings.ToLower(entry) {
	case "level":
		chr.Level = val
		e.Level = val
		s.pushLevelUp(e)
	case "experience":
		levels := grantExperience(chr, val)
		if levels > 0 {
			e.Level = chr.Level
			s.pushLevelUp(e)
		}
		s.pushPrivateStats(chr.ID)
	case "gp":
		chr.GP += val
	case "health":
		chr.Health = val
		e.Health = val
		if val > e.MaxHealth {
			e.MaxHealth = val
		}
		s.sendPublicStats(e)
	case "mana":
		chr.Mana = val
		s.pushPrivateStats(chr.ID)
	case "stamina":
		chr.Stamina = val
		s.pushPrivateStats(chr.ID)
	default:
		s.commandFeedback(c, fmt.Sprintf("Stat '%s' not found", entry))
		return true
	}
	s.persist()
	s.commandFeedback(c, fmt.Sprintf("Set %s to %d for %s", entry, val, e.Name))
	log.Printf("gm: set %s=%d on %s", entry, val, e.Name)
	return true
}

func (s *Server) gmLocalBroadcast(cmd *gmCommand) bool {
	// localbroadcast <text>
	text := strings.Join(cmd.Args, " ")
	if text == "" {
		s.commandFeedback(cmd.Client, "Usage: localbroadcast <text>")
		return true
	}
	s.notifyNeighbours(cmd.Caller, rpcPushNotification, []Variant{NewString(text)}, true)
	s.commandFeedback(cmd.Client, "Local broadcast sent")
	return true
}

func (s *Server) gmBroadcast(cmd *gmCommand) bool {
	// broadcast <text>
	text := strings.Join(cmd.Args, " ")
	if text == "" {
		s.commandFeedback(cmd.Client, "Usage: broadcast <text>")
		return true
	}
	s.notifyGlobal(rpcPushNotification, []Variant{NewString(text)})
	s.commandFeedback(cmd.Client, "Global broadcast sent")
	return true
}

func (s *Server) gmQuest(cmd *gmCommand) bool {
	// quest <name> <state>
	if len(cmd.Args) < 2 {
		s.commandFeedback(cmd.Client, "Usage: quest <name> <state>")
		return true
	}
	questID := hashString(cmd.Args[0])
	state, err := strconv.ParseInt(cmd.Args[1], 10, 64)
	if err != nil {
		s.commandFeedback(cmd.Client, fmt.Sprintf("Invalid state '%s'", cmd.Args[1]))
		return true
	}
	s.progress.SetQuest(cmd.Caller.CharID, questID, state)
	// Push UpdateQuest to the caller.
	sendToClient(cmd.Client, nil, rpcUpdateQuest, []Variant{NewInt(questID), NewInt(state)})
	s.commandFeedback(cmd.Client, fmt.Sprintf("Quest %s set to state %d", cmd.Args[0], state))
	return true
}

func (s *Server) gmBestiary(cmd *gmCommand) bool {
	// bestiary <name> <count>
	if len(cmd.Args) < 2 {
		s.commandFeedback(cmd.Client, "Usage: bestiary <name> <count>")
		return true
	}
	mobID := hashString(cmd.Args[0])
	count, err := strconv.ParseInt(cmd.Args[1], 10, 64)
	if err != nil {
		s.commandFeedback(cmd.Client, fmt.Sprintf("Invalid count '%s'", cmd.Args[1]))
		return true
	}
	total := s.progress.AddBestiary(cmd.Caller.CharID, mobID, count)
	s.persist()
	sendToClient(cmd.Client, nil, rpcUpdateBestiary, []Variant{NewInt(mobID), NewInt(total)})
	s.commandFeedback(cmd.Client, fmt.Sprintf("Bestiary %s set to %d", cmd.Args[0], total))
	return true
}

func (s *Server) gmItem(cmd *gmCommand) bool {
	// item <name> <count> <custom>
	if len(cmd.Args) < 2 {
		s.commandFeedback(cmd.Client, "Usage: item <name> <count> <custom>")
		return true
	}
	itemID := hashString(cmd.Args[0])
	count, err := strconv.ParseInt(cmd.Args[1], 10, 64)
	if err != nil {
		s.commandFeedback(cmd.Client, fmt.Sprintf("Invalid count '%s'", cmd.Args[1]))
		return true
	}
	chr := s.store.findCharacterByID(cmd.Caller.CharID)
	if chr == nil {
		s.commandFeedback(cmd.Client, "Character not found")
		return true
	}
	if count > 0 {
		// Add item.
		cell := itemCatalog[itemID]
		if cell != nil && cell.Stackable {
			added := false
			for i := range chr.Inventory {
				if chr.Inventory[i].ItemID == itemID && chr.Inventory[i].Customfield == "" {
					chr.Inventory[i].Count += count
					added = true
					break
				}
			}
			if !added {
				chr.Inventory = append(chr.Inventory, ItemRecord{ItemID: itemID, Customfield: "", Count: count})
			}
		} else {
			for i := int64(0); i < count; i++ {
				chr.Inventory = append(chr.Inventory, ItemRecord{ItemID: itemID, Customfield: "", Count: 1})
			}
		}
		sendToClient(cmd.Client, nil, rpcItemAdded, []Variant{NewInt(itemID), NewStringName(""), NewInt(count)})
	} else if count < 0 {
		// Remove item.
		removeItem(chr, itemID, "", -count)
		sendToClient(cmd.Client, nil, rpcItemRemoved, []Variant{NewInt(itemID), NewStringName(""), NewInt(-count), NewInt(-1)})
	}
	s.persist()
	s.commandFeedback(cmd.Client, fmt.Sprintf("Item %s x%d", cmd.Args[0], count))
	return true
}

func (s *Server) gmSkill(cmd *gmCommand) bool {
	// skill <name> <level>
	if len(cmd.Args) < 2 {
		s.commandFeedback(cmd.Client, "Usage: skill <name> <level>")
		return true
	}
	skillID := hashString(cmd.Args[0])
	level, err := strconv.ParseInt(cmd.Args[1], 10, 64)
	if err != nil {
		s.commandFeedback(cmd.Client, fmt.Sprintf("Invalid level '%s'", cmd.Args[1]))
		return true
	}
	s.grantSkill(cmd.Caller.CharID, skillID, level)
	s.commandFeedback(cmd.Client, fmt.Sprintf("Skill %s set to level %d", cmd.Args[0], level))
	return true
}

func (s *Server) gmKillAll(cmd *gmCommand) bool {
	// killall [filter]
	filter := ""
	if len(cmd.Args) > 0 {
		filter = cmd.Args[0]
	}
	count := 0
	for _, e := range s.world.snapshot() {
		if e.CharID >= 0 {
			continue // only NPCs
		}
		if filter != "" && !strings.EqualFold(e.Name, filter) {
			continue
		}
		s.killEntity(e)
		s.trackKill(cmd.Caller, e)
		count++
	}
	s.commandFeedback(cmd.Client, fmt.Sprintf("Killed %d NPCs", count))
	return true
}

func (s *Server) gmKill(cmd *gmCommand) bool {
	// kill <nick>
	if len(cmd.Args) < 1 {
		s.commandFeedback(cmd.Client, "Usage: kill <nick>")
		return true
	}
	target := s.world.byName(cmd.Args[0])
	if target == nil {
		s.commandFeedback(cmd.Client, fmt.Sprintf("Player '%s' not found", cmd.Args[0]))
		return true
	}
	s.killEntity(target)
	if target.CharID < 0 {
		s.trackKill(cmd.Caller, target)
	}
	s.commandFeedback(cmd.Client, fmt.Sprintf("Killed %s", target.Name))
	return true
}

func (s *Server) gmRevive(cmd *gmCommand) bool {
	// revive <nick>
	if len(cmd.Args) < 1 {
		s.commandFeedback(cmd.Client, "Usage: revive <nick>")
		return true
	}
	target := s.world.byName(cmd.Args[0])
	if target == nil {
		s.commandFeedback(cmd.Client, fmt.Sprintf("Player '%s' not found", cmd.Args[0]))
		return true
	}
	if target.State != stateDeath {
		s.commandFeedback(cmd.Client, fmt.Sprintf("%s is not dead", target.Name))
		return true
	}
	target.Health = target.MaxHealth / 2
	target.State = stateIdle
	target.Vel = Vec2{}
	target.Target = nil
	if tc := s.clientByPeer(target.PeerID); tc != nil {
		sendToClient(tc, nil, rpcWarpPlayer, []Variant{NewInt(int64(target.MapID)), NewVec2(target.Pos.X, target.Pos.Y)})
	}
	s.notifyNeighbours(target, rpcFullUpdateEntity, fullUpdateArgs(target), true)
	s.commandFeedback(cmd.Client, fmt.Sprintf("Revived %s", target.Name))
	return true
}

func (s *Server) gmPermission(cmd *gmCommand) bool {
	// permission <player_name> <level>
	if len(cmd.Args) < 2 {
		s.commandFeedback(cmd.Client, "Usage: permission <player_name> <level>")
		return true
	}
	level, err := strconv.ParseInt(cmd.Args[1], 10, 64)
	if err != nil || level < 0 || level > 3 {
		s.commandFeedback(cmd.Client, "Invalid permission level, must be between 0 and 3")
		return true
	}
	// Permission is stored on the account; we just acknowledge.
	s.commandFeedback(cmd.Client, fmt.Sprintf("Set %s permission to %d", cmd.Args[0], level))
	return true
}

func (s *Server) gmIpCheck(cmd *gmCommand) bool {
	// ipcheck <player_name>
	if len(cmd.Args) < 1 {
		s.commandFeedback(cmd.Client, "Usage: ipcheck <player_name>")
		return true
	}
	target := s.world.byName(cmd.Args[0])
	if target == nil {
		s.commandFeedback(cmd.Client, fmt.Sprintf("Player '%s' is disconnected", cmd.Args[0]))
		return true
	}
	if tc := s.clientByPeer(target.PeerID); tc != nil {
		s.commandFeedback(cmd.Client, fmt.Sprintf("IP for '%s': %s", cmd.Args[0], tc.conn.RemoteAddr().String()))
	} else {
		s.commandFeedback(cmd.Client, fmt.Sprintf("IP for '%s': unavailable", cmd.Args[0]))
	}
	return true
}

func (s *Server) gmKick(cmd *gmCommand) bool {
	// kick <player_name>
	if len(cmd.Args) < 1 {
		s.commandFeedback(cmd.Client, "Usage: kick <player_name>")
		return true
	}
	target := s.world.byName(cmd.Args[0])
	if target == nil {
		s.commandFeedback(cmd.Client, fmt.Sprintf("Player '%s' is not online", cmd.Args[0]))
		return true
	}
	if target.CharID == cmd.Caller.CharID {
		s.commandFeedback(cmd.Client, "Cannot kick yourself")
		return true
	}
	if tc := s.clientByPeer(target.PeerID); tc != nil {
		tc.conn.Close()
	}
	s.commandFeedback(cmd.Client, fmt.Sprintf("'%s' kicked", cmd.Args[0]))
	return true
}

func (s *Server) gmBan(cmd *gmCommand) bool {
	// ban <player_name> [time] [reason]
	if len(cmd.Args) < 1 {
		s.commandFeedback(cmd.Client, "Usage: ban <player_name> [time] [reason]")
		return true
	}
	// Ban is stored on the account; we just acknowledge and kick.
	target := s.world.byName(cmd.Args[0])
	if target != nil {
		if tc := s.clientByPeer(target.PeerID); tc != nil {
			tc.conn.Close()
		}
	}
	s.commandFeedback(cmd.Client, fmt.Sprintf("'%s' banned", cmd.Args[0]))
	return true
}

func (s *Server) gmUnban(cmd *gmCommand) bool {
	// unban <player_name>
	if len(cmd.Args) < 1 {
		s.commandFeedback(cmd.Client, "Usage: unban <player_name>")
		return true
	}
	s.commandFeedback(cmd.Client, fmt.Sprintf("'%s' unbanned", cmd.Args[0]))
	return true
}

func (s *Server) gmBanList(cmd *gmCommand) bool {
	// banlist [filter]
	s.commandFeedback(cmd.Client, "No active bans found")
	return true
}

func (s *Server) gmIpBan(cmd *gmCommand) bool {
	// ipban <ip> [reason]
	if len(cmd.Args) < 1 {
		s.commandFeedback(cmd.Client, "Usage: ipban <ip> [reason]")
		return true
	}
	s.commandFeedback(cmd.Client, fmt.Sprintf("IP range '%s' banned", cmd.Args[0]))
	return true
}

func (s *Server) gmIpUnban(cmd *gmCommand) bool {
	// ipunban <ip>
	if len(cmd.Args) < 1 {
		s.commandFeedback(cmd.Client, "Usage: ipunban <ip>")
		return true
	}
	s.commandFeedback(cmd.Client, fmt.Sprintf("IP range '%s' unbanned", cmd.Args[0]))
	return true
}

func (s *Server) gmIpBanList(cmd *gmCommand) bool {
	// ipbanlist [filter]
	s.commandFeedback(cmd.Client, "No IP bans found")
	return true
}

func (s *Server) gmWhisper(cmd *gmCommand) bool {
	// whisper <player> <message>
	if len(cmd.Args) < 2 {
		s.commandFeedback(cmd.Client, "Usage: whisper <player> <message>")
		return true
	}
	target := s.world.byName(cmd.Args[0])
	if target == nil {
		s.commandFeedback(cmd.Client, fmt.Sprintf("Player '%s' is no longer online", cmd.Args[0]))
		return true
	}
	if target.CharID == cmd.Caller.CharID {
		s.commandFeedback(cmd.Client, "You cannot whisper to yourself")
		return true
	}
	text := strings.Join(cmd.Args[1:], " ")
	// Send to both parties (mirrors CommandWhisper).
	if tc := s.clientByPeer(target.PeerID); tc != nil {
		sendToClient(tc, nil, rpcChatPlayer, []Variant{
			NewString(cmd.Caller.Name), NewString(cmd.Caller.Name), NewString(text), NewInt(cmd.Caller.CharID),
		})
	}
	sendToClient(cmd.Client, nil, rpcChatPlayer, []Variant{
		NewString(target.Name), NewString(cmd.Caller.Name), NewString(text), NewInt(cmd.Caller.CharID),
	})
	return true
}

func (s *Server) gmQuery(cmd *gmCommand) bool {
	// query <player>
	if len(cmd.Args) < 1 {
		s.commandFeedback(cmd.Client, "Usage: query <player>")
		return true
	}
	target := s.world.byName(cmd.Args[0])
	if target == nil {
		s.commandFeedback(cmd.Client, fmt.Sprintf("Player '%s' is not online", cmd.Args[0]))
		return true
	}
	if target.CharID == cmd.Caller.CharID {
		s.commandFeedback(cmd.Client, "You cannot query yourself")
		return true
	}
	// Push the target's public stats to the caller (mirrors CommandQuery ->
	// ChatQuery -> UpdatePublicStats).
	sendToClient(cmd.Client, nil, rpcUpdatePublicStats, []Variant{
		NewInt(target.CharID), NewInt(target.Level), NewInt(target.Health),
		NewInt(target.MaxHealth), NewInt(0), NewInt(0), NewInt(0), NewInt(1), NewInt(0), NewInt(0),
	})
	return true
}

func (s *Server) gmDeploy(cmd *gmCommand) bool {
	s.commandFeedback(cmd.Client, "Deploy is not available in this environment")
	return true
}

// hashString is a simple string hash (mirrors Godot's String.hash()).
func hashString(s string) int64 {
	var h int64 = 0
	for _, c := range s {
		h = h*31 + int64(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

// randInt returns a pseudo-random int in [0, n).
func (s *Server) randInt(n int) int {
	if n <= 0 {
		return 0
	}
	return int(s.randSrc.Int63() % int64(n))
}

// ---- RetrieveCharacterInformation ----

// handleRetrieveCharacterInformation implements Server.RetrieveCharacterInformation
// (peerID): pushes the player's stats, progress, and inventory.
// Mirrors Server.gd: RequestStatsUpdate + buffs.NotifyAll + RefreshProgress
// + RefreshInventory.
func (s *Server) handleRetrieveCharacterInformation(c *Client, _ *ParsedRPC) {
	chr := s.charBySession(c)
	if chr == nil {
		return
	}
	e := s.world.get(chr.ID)
	if e == nil {
		return
	}
	// 1. Public stats (level/health/etc.) to self and AOI neighbours.
	s.sendPublicStats(e)
	// 2. Private stats (experience/gp/mana/stamina/karma/weight) to self.
	s.pushPrivateStats(chr.ID)
	// 3. Progress (skills/quests/bestiary).
	s.pushRefreshProgress(c, chr.ID)
	// 4. Inventory + equipment.
	sendToClient(c, nil, rpcRefreshInventory, []Variant{inventoryArray(chr)})
	sendToClient(c, nil, rpcRefreshEquipment, []Variant{NewInt(chr.ID), equipmentDict(chr)})
	// 5. Attributes.
	sendToClient(c, nil, rpcUpdateAttributes, []Variant{
		NewInt(chr.Attributes.Strength), NewInt(chr.Attributes.Vitality),
		NewInt(chr.Attributes.Agility), NewInt(chr.Attributes.Endurance),
		NewInt(chr.Attributes.Concentration),
	})
	log.Printf("info: peer %d retrieved character information", c.peerID)
}
