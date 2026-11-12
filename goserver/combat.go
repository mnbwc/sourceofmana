package main

// Phase-3 combat: skill resolution, damage, death, and respawn.
//
// Mirrors SoM's combat flow (Skill.gd / Stats.gd / BaseAgent.gd):
//   - TriggerSkill(targetRID, skillID) -> Skill.Cast -> Process -> Handle
//   - Handle -> Damaged -> target.stat.SetHealth(-value)
//     -> Network.NotifyNeighbours(agent, "TargetAlteration",
//          [agentRID, targetRID, value, alteration, skillID, true], true, true)
//   - SetHealth clamps and calls Killed() when health drops to 0
//     (state = DEATH).
//   - TriggerRespawn -> player.Respawn() -> Revive() (half health, IDLE)
//     + WarpTo(respawnDestination).
//
// The server is authoritative for health: the client sends TriggerSkill, the
// server computes damage from a small skill table and broadcasts the result.

import (
	"log"
	"math"
	"math/rand"
	"time"
)

// defaultHealth is the starting max health for a new character.
const defaultHealth = 100

// ActorCommons.State values used by combat.
const (
	stateIdle  = 0
	stateDeath = 4
)

// ActorCommons.Alteration values (CellCommons/Network.gd).
const (
	altUnknown = -1
	altHit     = 0
	altCrit    = 1
	altMiss    = 2
	altDodge   = 3
	altHeal    = 4
)

// skillTable maps a skill ID to its damage/range/cooldown profile.
// Values mirror the preset .tres skills shipped with SoM:
//   - Melee (id 229218829): close-range physical attack, repeat.
//   - Archer (id 2814726266): ranged physical attack, repeat, range 100.
//   - Flar (id 2089110122): magic attack, range 120, cooldown 3s.
//   - Inma (id 2089220490): magic attack, range 128, cooldown 5s.
//   - ManaBurst (id 2393790290): close-range magic attack, range 40.
//   - SonicWave (id 123697108): magic attack, range 64.
//   - Spitfire (id 3219153995): close-range magic attack, range 32.
type skillProfile struct {
	ID           int64
	Attack       int64 // flat attack modifier (Modifier.Attack=20 / MAttack=7)
	Magic        bool  // true = magic damage (uses MAttack/MDefense)
	Range        int64 // skillRange (added to the actor's attackRange)
	Cooldown     float64
	Repeat       bool
	Stamina      int64   // stamina cost per cast
	HealthMod    int64   // positive = heal instead of damage (Modifier.Health=1)
	BuffEffect   int64   // CellCommons.Modifier applied on hit (0 = none)
	BuffValue    int64   // magnitude of the buff effect
	BuffDuration float64 // seconds (0 = instant, no buff)
}

var skillTable = map[int64]skillProfile{
	229218829:  {ID: 229218829, Attack: 1, Range: 0, Cooldown: 0.5, Repeat: true, Stamina: 10},                                                           // Melee
	2814726266: {ID: 2814726266, Attack: 1, Range: 100, Cooldown: 0.5, Repeat: true, Stamina: 10},                                                        // Archer
	2089110122: {ID: 2089110122, Attack: 50, Magic: true, Range: 120, Cooldown: 3, Stamina: 10},                                                          // Flar
	2089220490: {ID: 2089220490, Attack: 50, Magic: true, Range: 128, Cooldown: 5, Stamina: 15, BuffEffect: modDefense, BuffValue: -5, BuffDuration: 10}, // Inma: lowers target defense
	2393790290: {ID: 2393790290, Attack: 20, Magic: true, Range: 40, Cooldown: 1, Stamina: 10},                                                           // ManaBurst
	123697108:  {ID: 123697108, Attack: 50, Magic: true, Range: 64, Cooldown: 3, Stamina: 10, BuffEffect: modWalkSpeed, BuffValue: -20, BuffDuration: 5}, // SonicWave: slows target
	3219153995: {ID: 3219153995, Attack: 20, Magic: true, Range: 32, Cooldown: 1, Stamina: 5},                                                            // Spitfire
}

// defaultAttackRange is the actor's base attackRange (BaseStats.gd default).
const defaultAttackRange = 32

// handleTriggerSkill mirrors Server.gd TriggerSkill -> Skill.Cast.
//
// SoM's full flow has cast timers, mana/stamina consumption, projectiles, and
// AI reactions. The Go server resolves instantaneously: validate the caster
// and target, compute damage from the skill table, apply it, and broadcast
// TargetAlteration to AOI neighbours (inclusive of the caster, matching
// Skill.Damaged's NotifyNeighbours(..., true, true)).
func (s *Server) handleTriggerSkill(c *Client, rpc *ParsedRPC) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	targetRID := argInt(rpc.Args, 0)
	skillID := argInt(rpc.Args, 1)
	if targetRID == e.CharID {
		// Self-targeted skills (heals) are not in the MVP table; ignore.
		return
	}
	target := s.world.get(targetRID)
	if target == nil {
		s.pushNotification(e, "Target is not online.")
		return
	}
	profile, ok := skillTable[skillID]
	if !ok {
		s.pushNotification(e, "Unknown skill.")
		return
	}
	if e.State == stateDeath {
		return
	}
	if target.State == stateDeath {
		s.pushNotification(e, target.Name+" is already dead.")
		return
	}
	// Range check: SkillCommons.IsNear uses attackRange + skillRange + both
	// actors' collision radii (16 px each in SoM).
	reach := float32(defaultAttackRange + profile.Range + 16 + 16)
	if e.Pos.dist(target.Pos) > reach {
		s.pushNotification(e, target.Name+" is too far away.")
		return
	}
	// Cast broadcast: Skill.Casted -> NotifyNeighbours(agent, "Casted", ...).
	s.notifyNeighbours(e, rpcCasted, []Variant{
		NewInt(e.CharID), NewInt(skillID), NewFloat(profile.Cooldown),
	}, true)

	// Damage resolution (SkillCommons.GetDamage + Skill.Damaged).
	dmg, alt := s.computeDamage(e, target, profile)
	if alt == altDodge || alt == altMiss {
		// Skill.Damaged broadcasts the alteration type from GetDamage (DODGE).
		s.notifyNeighbours(e, rpcTargetAlteration, []Variant{
			NewInt(e.CharID), NewInt(target.CharID), NewInt(0),
			NewInt(int64(alt)), NewInt(skillID), NewBool(true),
		}, true)
		return
	}
	// Apply damage (Stats.SetHealth(-value)) and broadcast.
	actual := dmg
	if actual > target.Health {
		actual = target.Health
	}
	s.applyDamage(e, target, actual, alt, skillID)
	// Phase-6: apply any on-hit buff/debuff to the target.
	if profile.BuffEffect != modNone && profile.BuffDuration > 0 {
		s.applyBuff(target.CharID, profile.BuffEffect, skillID,
			NewInt(profile.BuffValue), time.Duration(profile.BuffDuration*float64(time.Second)))
	}
	log.Printf("combat: %s -> %s skill=%d dmg=%d alt=%d health=%d/%d",
		e.Name, target.Name, skillID, actual, alt, target.Health, target.MaxHealth)
}

// computeDamage mirrors SkillCommons.GetDamage: base attack/mattack + skill
// modifier - target defense/mdefense, then evade/crit rolls.
//
// The actor's level is only a proxy: the real game derives attack and defense
// from attributes plus equipped items, so the equipped weapon's Attack and the
// armor's Defense modifiers are added here. Without them the starting Short
// Sword (+20 Attack) would make no difference and the "equip the knife, then
// fight" loop would be pointless.
func (s *Server) computeDamage(agent, target *Entity, profile skillProfile) (int64, int) {
	attack := profile.Attack + agent.Level + s.equipmentAttackBonus(agent)
	defense := target.Level/2 + s.equipmentDefenseBonus(target)
	dmg := attack - defense
	if dmg < 1 {
		dmg = 1
	}
	// Evade roll (SkillCommons.GetEvadeRate): keep it simple, ~5%.
	if rand.Float64() < 0.05 {
		return 0, altDodge
	}
	// Crit roll: ~10%, double damage.
	if rand.Float64() < 0.10 {
		return dmg * 2, altCrit
	}
	// Damage roll: SkillCommons.GetRNG -> randf_range(0.9, 1.0), then ceili
	// (SoM rounds partial damage up).
	dmg = int64(math.Ceil(float64(dmg) * (0.9 + rand.Float64()*0.1)))
	if dmg < 1 {
		dmg = 1
	}
	return dmg, altHit
}

// equipmentAttackBonus returns the Attack modifier of the weapon equipped by
// e's character (ActorCommons.Slot.WEAPON = 6). NPCs and unequipped actors
// contribute nothing.
func (s *Server) equipmentAttackBonus(e *Entity) int64 {
	return s.equipmentModifier(e, 6, "Attack")
}

// equipmentDefenseBonus returns the summed Defense modifier of everything
// e's character has equipped. NPCs and unequipped actors contribute nothing.
func (s *Server) equipmentDefenseBonus(e *Entity) int64 {
	if e == nil || e.CharID <= 0 {
		return 0
	}
	chr := s.store.findCharacterByID(e.CharID)
	if chr == nil {
		return 0
	}
	var total int64
	for _, rec := range chr.Equipment {
		if cell := itemCatalog[rec.ItemID]; cell != nil {
			total += cell.Defense
		}
	}
	return total
}

// equipmentModifier reads one modifier value off a single equipped slot.
func (s *Server) equipmentModifier(e *Entity, slot int64, kind string) int64 {
	if e == nil || e.CharID <= 0 {
		return 0
	}
	chr := s.store.findCharacterByID(e.CharID)
	if chr == nil {
		return 0
	}
	rec, ok := chr.Equipment[slot]
	if !ok {
		return 0
	}
	cell := itemCatalog[rec.ItemID]
	if cell == nil {
		return 0
	}
	switch kind {
	case "Attack":
		return cell.Attack
	case "Defense":
		return cell.Defense
	}
	return 0
}

// applyDamage mirrors Skill.Damaged + Stats.SetHealth(-value):
// clamp the damage to the target's current health, subtract it, broadcast
// TargetAlteration to AOI neighbours (inclusive), and trigger death when
// health reaches zero (BaseAgent.Killed -> state = DEATH).
func (s *Server) applyDamage(agent, target *Entity, value int64, alt int, skillID int64) {
	if value > target.Health {
		value = target.Health
	}
	previousHealth := target.Health
	target.Health -= value
	// Mirror PlayerAgent.UpdateLastStats -> Network.NotifyInstance(...,
	// "UpdatePublicStats", ...): keep AOI clients' HP bars in sync.
	s.sendPublicStats(target)
	// NotifyNeighbours(agent, "TargetAlteration", [agentRID, targetRID, value,
	// alteration, skillID, true], inclusive, reliable).
	s.notifyNeighbours(agent, rpcTargetAlteration, []Variant{
		NewInt(agent.CharID), NewInt(target.CharID), NewInt(value),
		NewInt(int64(alt)), NewInt(skillID), NewBool(true),
	}, true)

	// Stats.SetHealth: health <= 0 and previousHealth > 0 -> Killed().
	if target.Health <= 0 && previousHealth > 0 {
		s.killEntity(target)
		// Phase-8: track bestiary for NPC kills.
		s.trackKill(agent, target)
		// Grant experience to the killer (phase-5 progression).
		s.grantKillExperience(agent, target)
	}
}

// grantKillExperience awards experience to the attacking player when their
// target dies. The amount scales with the target's level (SoM MonsterAgent
// drops experience via Progress.AddExperience).
func (s *Server) grantKillExperience(agent, target *Entity) {
	if agent == nil || target == nil {
		return
	}
	chr := s.store.findCharacterByID(agent.CharID)
	if chr == nil {
		return
	}
	// Base XP = 10 + target level * 5. Player targets grant more than NPCs.
	xp := int64(10 + target.Level*5)
	if target.NpcRID != 0 || target.CharID < 0 { // NPC
		xp = int64(5 + target.Level*2)
	}
	levels := grantExperience(chr, xp)
	s.persist()
	// Push EXP alteration to the killer (TargetAlteration with EXP).
	s.notifyNeighbours(agent, rpcTargetAlteration, []Variant{
		NewInt(agent.CharID), NewInt(agent.CharID), NewInt(xp),
		NewInt(5), // ActorCommons.Alteration.EXP
		NewInt(0), NewBool(true),
	}, true)
	// Phase-6: keep the client's private stats (experience/gp) in sync.
	s.pushPrivateStats(agent.CharID)
	if levels > 0 {
		// Level up: update the world entity and broadcast.
		agent.Level = chr.Level
		s.pushLevelUp(agent)
		log.Printf("progression: %s reached level %d (+%d XP)", chr.Name, chr.Level, xp)
	} else {
		log.Printf("progression: %s +%d XP (level %d, %d/%d)",
			chr.Name, xp, chr.Level, chr.Experience, neededExperience(chr.Level))
	}
}

// killEntity mirrors BaseAgent.Killed: clear state, set DEATH, and notify
// AOI neighbours of the state change via FullUpdateEntity (requireFullUpdate).
func (s *Server) killEntity(target *Entity) {
	target.State = stateDeath
	target.Health = 0
	target.Vel = Vec2{}
	target.Target = nil
	target.DeadAt = time.Now()
	log.Printf("combat: %s died", target.Name)
	// Broadcast the death state to AOI neighbours and to the victim.
	s.notifyNeighbours(target, rpcFullUpdateEntity, fullUpdateArgs(target), true)
	// Phase-6: NPCs drop loot on death (mirrors MonsterAgent drop tables).
	if target.NpcRID != 0 || target.CharID < 0 {
		s.spawnNpcDrops(target)
	}
}

// npcDropTable resolves the drop table for a killed entity: the NPC's
// explicit table (tools/export/drops.json) when it has one, otherwise the
// generic Potion / GP roll.
func (s *Server) npcDropTable(target *Entity) []npcDrop {
	var npc *Npc
	if target.NpcRID != 0 {
		npc = s.npcs.get(target.NpcRID)
	} else if target.CharID < 0 {
		npc = s.npcByCharID(target.CharID)
	}
	if npc != nil && len(npc.Drops) > 0 {
		out := make([]npcDrop, 0, len(npc.Drops)+1)
		for _, d := range npc.Drops {
			if d.chance > 0 && rand.Float64() < d.chance {
				out = append(out, d)
			}
		}
		if target.Level >= 3 {
			out = append(out, npcDrop{itemID: 0, count: 5 + target.Level*2})
		}
		return out
	}
	drops := []npcDrop{{itemID: 1001, count: 1 + target.Level%3}} // Potion
	if target.Level >= 3 {
		drops = append(drops, npcDrop{itemID: 0, count: 5 + target.Level*2}) // GP bundle
	}
	return drops
}

// spawnNpcDrops mirrors MonsterAgent's drop roll: each NPC has a small chance
// to drop a consumable or GP. Drops are ground entities that expire after
// dropLifetime and can be picked up via PickupDrop.
//
// When the NPC has an explicit drop table (tools/export/drops.json: Maggot ->
// Maggot Slime, Skeleton -> Bone, ...) it is rolled; otherwise the monster
// falls back to the generic Potion / GP table.
func (s *Server) spawnNpcDrops(target *Entity) {
	table := s.npcDropTable(target)
	for _, d := range table {
		// Scatter the drop slightly around the corpse.
		offset := Vec2{
			X: float32(int(target.Pos.X)%7) * 8,
			Y: float32(int(target.Pos.Y)%5) * 8,
		}
		pos := Vec2{target.Pos.X + offset.X, target.Pos.Y + offset.Y}
		s.spawnDrop(target.CharID, d.itemID, "", pos, target.MapID)
	}
}

// handleTriggerRespawn mirrors Server.gd TriggerRespawn -> PlayerAgent.Respawn:
// Revive() (half max health, IDLE) then WarpTo(respawnDestination).
func (s *Server) handleTriggerRespawn(c *Client, rpc *ParsedRPC) {
	e := s.world.get(c.session.charID)
	if e == nil {
		log.Printf("combat: peer %d TriggerRespawn: no entity for char %d", c.peerID, c.session.charID)
		return
	}
	if e.State != stateDeath {
		// Revive() returns false for living actors; SoM ignores the call.
		log.Printf("combat: peer %d TriggerRespawn: %s not dead (state=%d)", c.peerID, e.Name, e.State)
		return
	}
	// Revive: half max health, back to IDLE.
	e.Health = e.MaxHealth / 2
	e.State = stateIdle
	// WarpTo(respawnDestination): the MVP respawns at the world spawn.
	e.Pos = spawnPos(e.CharID)
	e.Vel = Vec2{}
	e.Target = nil
	// WarpPlayer(mapID, pos) to the respawned player.
	sendToClient(c, nil, rpcWarpPlayer, []Variant{NewInt(int64(e.MapID)), NewVec2(e.Pos.X, e.Pos.Y)})
	// FullUpdateEntity so neighbours see the revived state/position.
	s.notifyNeighbours(e, rpcFullUpdateEntity, fullUpdateArgs(e), true)
	log.Printf("combat: %s respawned at (%.0f,%.0f) health=%d/%d",
		e.Name, e.Pos.X, e.Pos.Y, e.Health, e.MaxHealth)
}

// spawnPos mirrors the enterWorld spawn layout so respawns land on the same
// grid as fresh characters.
func spawnPos(charID int64) Vec2 {
	return Vec2{100 + float32(charID%8)*60, 100 + float32(charID%5)*60}
}

// pushNotification sends a PushNotification to a single player (NpcCommons).
func (s *Server) pushNotification(e *Entity, text string) {
	if sc := s.clientByPeer(e.PeerID); sc != nil {
		sendToClient(sc, nil, rpcPushNotification, []Variant{NewString(text), NewInt(int64(e.PeerID))})
	}
}

// sendPublicStats sends UpdatePublicStats for an entity to itself and its
// AOI neighbours. Mirrors PlayerAgent.UpdateLastStats ->
// Network.NotifyInstance(inst, "UpdatePublicStats", [...]).
func (s *Server) sendPublicStats(e *Entity) {
	race := int64(1)
	hairstyle := int64(0)
	haircolor := int64(0)
	gender := int64(0)
	skintone := int64(0)
	if e.CharID > 0 {
		if chr := s.store.findCharacterByID(e.CharID); chr != nil {
			race = chr.Race
			hairstyle = chr.Hairstyle
			haircolor = chr.Haircolor
			gender = chr.Gender
			skintone = chr.Skintone
		}
	}
	args := []Variant{
		NewInt(e.CharID),
		NewInt(e.Level),
		NewInt(e.Health),
		NewInt(e.MaxHealth),
		NewInt(hairstyle),
		NewInt(haircolor),
		NewInt(gender),
		NewInt(race),
		NewInt(skintone),
		NewInt(0), // currentShape
	}
	if sc := s.clientByPeer(e.PeerID); sc != nil {
		sendToClient(sc, nil, rpcUpdatePublicStats, args)
	}
	for _, other := range s.world.snapshot() {
		if other.CharID == e.CharID || other.MapID != e.MapID || e.Pos.dist(other.Pos) > aoiRadius {
			continue
		}
		if oc := s.clientByPeer(other.PeerID); oc != nil {
			sendToClient(oc, nil, rpcUpdatePublicStats, args)
		}
	}
}
