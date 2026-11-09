extends Node

# Phase-1/2 e2e client. One simulated SoM client per Godot process.
# NAME selects this client's role (default "solo"):
#   solo : run the login chain only, expect "E2E PASS: solo login chain".
#   peerA: client A - login, wait for peerB visibility, then move and
#         expect to see peerB's movement; prints E2E PASS when both
#         visibility and movement sync are confirmed.
#   peerB: client B - login, then report visibility of peerA and wait
#         for peerA's movement; prints its own E2E PASS line.
# Run via run_e2e.sh which launches the two peers concurrently.
#
# @rpc config is ignored by Godot (nameIDs are sorted by method name),
# so the 4th argument is kept as the Network.gd sorted index for
# documentation only.

const SERVER_URL := "ws://127.0.0.1:27012/ws"

var role := "solo"
var my_nick := "Wone"
var peer_nick := "Wtwo"
var peer: WebSocketMultiplayerPeer
var mp: SceneMultiplayer
var frames := 0
var connected := false
var connected_frame := 0
var chain: Array = []
var step := 0
var got_replies := 0
var sent_for_step := false
var fail := false
var seen_preloads: Array = []
var seen_online := {}
var seen_move_x := {}
var seen_chats: Array = []
var seen_emotes: Array = []
var seen_casts: Array = []
var update_count := 0
var full_count := 0
var prev_move_x := {}
var moved_ids := {}
var moved_other := false
var sent_gameplay := false
var gameplay_sent_frame := 0
# The peer's agentRID, learned from PreloadPlayer (keyed by nick). Movement
# is detected by comparing the peer entity's x against its FIRST observed
# x, so a single UpdateEntity is enough — we do not need two observations.
var peer_rid := 0
var peer_first_x := 0.0
var peer_spawn_seen := false
# NPC/quest tracking (phase 3). NPCs arrive via PreloadEntity during enterWorld.
var seen_npcs := {}            # nick -> RID
var npc_interact_sent := false
var npc_next_sent := false
var npc_choice_sent := false
var npc_select_sent := false
var npc_stage := 0             # 0=none 1=interact 2=next 3=choice 4=select
var npc_rid := 0
var seen_context_texts: Array = []
var seen_context_choices: Array = []
var seen_context_continues := 0
var seen_context_closes := 0
var seen_quest_updates: Array = []
var seen_public_stats: Array = []
# Party tracking (phase 3).
var seen_party_invite := ""
var seen_party_updates: Array = []
var seen_party_leader := 0
var party_stage := 0
var party_sent := false
# Trade tracking (phase 3).
var seen_trade_requested := ""
var seen_trade_opened := ""
var seen_trade_updates: Array = []
var seen_trade_closed := ""
var trade_stage := 0
# Combat tracking (phase 3).
var seen_alterations: Array = []   # [agentRID, targetRID, value, alteration, skillID]
var seen_deaths: Array = []        # agentRIDs that entered DEATH state
var seen_revives: Array = []       # agentRIDs that returned to IDLE
var combat_stage := 0
var combat_skill_id := 2089110122  # Flar (high damage, range 120)
var combat_casts := 0
var combat_walk_sent := false
var combat_arrived := false
var combat_start_frame := 0
var peer_pos := Vector2.ZERO        # peer's last known position
var my_rid := 0                    # our own agentRID (from UpdatePublicStats)
var last_pos_by_rid := {}          # RID -> latest position (any entity)
# Phase-4 items/economy tracking.
var seen_item_added: Array = []      # [itemID, customfield, count]
var seen_item_removed: Array = []   # [itemID, customfield, count, itemIndex]
var seen_item_equiped: Array = []   # [agentRID, itemID, customfield, state, itemIndex]
var seen_attributes: Array = []     # [str, vit, agi, end, con]
var seen_trade_closed_count := 0
# Phase-6 tracking.
var seen_entity_buffs: Array = []     # [agentRID, effect, skillID, enabled]
var seen_update_buffs: Array = []     # [effect, value, duration, skillID]
var seen_drop_added: Array = []       # [dropID, itemID, pos]
var my_rid_from_preload := 0          # our own agentRID (from PreloadPlayer)
var my_pos := Vector2.ZERO            # our own position (from FullUpdateEntity)
var seen_drop_removed: Array = []     # [dropID]
var pickup_sent_drops: Array = []     # dropIDs we already sent PickupDrop for
var seen_removes: Array = []         # [agentRID]
var seen_private_stats: Array = []   # [experience, gp]

func _mk_chain(acc_name: String, char_name: String) -> Array:
	return [
		{"send": "CreateAccount", "args": [acc_name, "secret1", "e2e@t.c", true, 0],
			"expect": [["AuthTokenResult", null], ["AuthError", 0]]},
		{"send": "CharacterListing", "args": [],
			"expect": [["CharacterError", 11]]},
		{"send": "CreateCharacter", "args": [char_name, {"race": 1, "gender": 0}, {}],
			"expect": [["CharacterInfo", null], ["CharacterError", 0]]},
		{"send": "CharacterListing", "args": [],
			"expect": [["CharacterInfo", null], ["CharacterError", 0]]},
		{"send": "ConnectCharacter", "args": [char_name],
			"expect": [["CharacterError", 0]]},
	]

@rpc("any_peer", "call_remote", "reliable", 25)
func CreateAccount(accountName, password, email, rememberMe, platform = 0, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 45)
func LoginWithPassword(accountName, password, rememberMe, platform = 0, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 1)
func AuthError(err, peerID = 0) -> void:
	_on_reply("AuthError", [err, peerID])

@rpc("any_peer", "call_remote", "reliable", 46)
func LoginWithToken(accountName, token, platform = 0, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 2)
func AuthTokenResult(accountName, token, peerID = 0) -> void:
	_on_reply("AuthTokenResult", [accountName, token, peerID])

@rpc("any_peer", "call_remote", "reliable", 64)
func RequestPasswordReset(accountName, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 18)
func ConfirmPasswordReset(accountName, code, newPassword, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 7)
func ChangePassword(currentPassword, newPassword, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 28)
func DisconnectAccount(peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 9)
func CharacterInfo(info, equipment, peerID = 0) -> void:
	_on_reply("CharacterInfo", [info, equipment, peerID])

@rpc("any_peer", "call_remote", "reliable", 26)
func CreateCharacter(charName, traits, attributes, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 27)
func DeleteCharacter(charName, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 19)
func ConnectCharacter(nickname, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 8)
func CharacterError(err, peerID = 0) -> void:
	_on_reply("CharacterError", [err, peerID])

@rpc("any_peer", "call_remote", "reliable", 29)
func DisconnectCharacter(peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 10)
func CharacterListing(peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 63)
func RequestOnlineList(peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 59)
func RefreshOnlineList(players, peerID = 0) -> void:
	for n in players:
		var s := str(n)
		if not s.is_empty():
			seen_online[s] = true

@rpc("any_peer", "call_remote", "reliable", 0)
func AddOnlinePlayer(playerName, peerID = 0) -> void:
	seen_online[playerName] = true

@rpc("any_peer", "call_remote", "reliable", 62)
func RemoveOnlinePlayer(playerName, peerID = 0) -> void:
	seen_online.erase(playerName)

@rpc("any_peer", "call_remote", "reliable", 65)
func RequestRtcUpgrade(peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 71)
func RtcConfig(iceServers, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 72)
func RtcOffer(sdp, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 68)
func RtcAnswer(sdp, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 69)
func RtcCandidateToClient(media, index, candidateName, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 70)
func RtcCandidateToServer(media, index, candidateName, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 73)
func RtcReady(peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 104)
func TriggerRespawn(peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 118)
func WarpPlayer(mapID, playerPos, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 54)
func PreloadEntity(agentRID, actorType, currentShape, nick, defaultState, peerID = 0) -> void:
	# NPCs are preloaded during enterWorld; record them for the npc role.
	if actorType == 1 and nick != my_nick:
		seen_npcs[nick] = agentRID

@rpc("any_peer", "call_remote", "reliable", 55)
func PreloadPlayer(agentRID, spirit, currentShape, nick, level, health, hairstyle, haircolor, gender, race, skintone, equipment, peerID = 0) -> void:
	seen_preloads.append(nick)
	if nick == my_nick:
		my_rid_from_preload = agentRID
	if nick == peer_nick:
		peer_rid = agentRID

@rpc("any_peer", "call_remote", "reliable", 61)
func RemoveEntity(agentRID, peerID = 0) -> void:
	seen_removes.append(agentRID)

@rpc("any_peer", "call_remote", "reliable", 31)
func DisplayProgressionTracker(label, value, maxValue, unit, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 15)
func ClearProgressionTracker(peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 30)
func DisplayActions(actions, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 56)
func PushNotification(notif, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 75)
func SetClickPos(pos, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 76)
func SetMovePos(pos, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 77)
func SetOrientation(direction, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 112)
func UpdateEntity(agentRID, velocity, position, frameID, peerID = 0) -> void:
	update_count += 1
	var x: float = position[0]
	if prev_move_x.has(agentRID) and absf(x - prev_move_x[agentRID]) > 0.5:
		moved_ids[agentRID] = true
	prev_move_x[agentRID] = seen_move_x.get(agentRID, x)
	seen_move_x[agentRID] = x
	# Always remember the latest position per RID so a position that arrives
	# before PreloadPlayer (which sets peer_rid) is not lost.
	last_pos_by_rid[agentRID] = position
	if agentRID == peer_rid:
		peer_pos = position
		_track_peer_move(agentRID, x)

@rpc("any_peer", "call_remote", "reliable", 39)
func FullUpdateEntity(agentRID, velocity, position, orientation, agentState, skillCastID, isRunning, frameID, peerID = 0) -> void:
	full_count += 1
	if agentRID == my_rid_from_preload:
		my_pos = position
	var x: float = position[0]
	if prev_move_x.has(agentRID) and absf(x - prev_move_x[agentRID]) > 0.5:
		moved_ids[agentRID] = true
	prev_move_x[agentRID] = seen_move_x.get(agentRID, x)
	seen_move_x[agentRID] = x
	last_pos_by_rid[agentRID] = position
	if agentRID == peer_rid:
		peer_pos = position
		_track_peer_move(agentRID, x)
	# Combat: track DEATH (state 4) and revive (state 0) transitions.
	if agentState == 4:
		seen_deaths.append(agentRID)
	elif agentState == 0 and seen_deaths.has(agentRID):
		seen_revives.append(agentRID)

# Record the peer entity's first position, then flag movement once its x has
# shifted by more than 0.5 px. Works with a single update (no second
# observation required), which matters because a peer may leave right after
# its first broadcast.
func _track_peer_move(agentRID: int, x: float) -> void:
	if agentRID != peer_rid:
		return
	if not peer_spawn_seen:
		peer_spawn_seen = true
		peer_first_x = x
		return
	if absf(x - peer_first_x) > 0.5:
		moved_ids[agentRID] = true

@rpc("any_peer", "call_remote", "reliable", 14)
func ClearNavigation(peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 78)
func SetViewportSize(halfWidth, halfHeight, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 94)
func TriggerEmote(emoteID, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 35)
func Emote(senderagentRID, emoteID, peerID = 0) -> void:
	seen_emotes.append([senderagentRID, emoteID])

@rpc("any_peer", "call_remote", "reliable", 106)
func TriggerSit(peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 38)
func Express(agentRID, text, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 90)
func TriggerChat(channelName, text, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 12)
func ChatQuery(channelName, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 100)
func TriggerPartyInvite(targetNick, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 98)
func TriggerPartyAccept(peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 99)
func TriggerPartyDecline(peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 102)
func TriggerPartyLeave(peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 101)
func TriggerPartyKick(targetNick, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 103)
func TriggerPartyPromote(targetNick, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 51)
func PartyInvite(inviterName, peerID = 0) -> void:
	seen_party_invite = inviterName

@rpc("any_peer", "call_remote", "reliable", 52)
func PartyUpdate(members, leaderRID, peerID = 0) -> void:
	seen_party_updates.append([members, leaderRID])
	seen_party_leader = leaderRID

@rpc("any_peer", "call_remote", "reliable", 11)
func ChatPlayer(channelName, callerName, text, agentRID, peerID = 0) -> void:
	seen_chats.append([channelName, callerName, text])

@rpc("any_peer", "call_remote", "reliable", 13)
func ChatSystem(channelName, text, peerID = 0) -> void:
	seen_chats.append([channelName, "SYSTEM", text])

@rpc("any_peer", "call_remote", "reliable", 81)
func ToggleContext(enable, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 23)
func ContextText(author, text, peerID = 0) -> void:
	seen_context_texts.append([author, text])

@rpc("any_peer", "call_remote", "reliable", 24)
func ContextThink(author, text, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 22)
func ContextContinue(peerID = 0) -> void:
	seen_context_continues += 1

@rpc("any_peer", "call_remote", "reliable", 21)
func ContextClose(peerID = 0) -> void:
	seen_context_closes += 1

@rpc("any_peer", "call_remote", "reliable", 20)
func ContextChoice(texts, peerID = 0) -> void:
	seen_context_choices.append(texts)

@rpc("any_peer", "call_remote", "reliable", 91)
func TriggerChoice(choiceID, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 92)
func TriggerCloseContext(peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 97)
func TriggerNextContext(peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 40)
func HighlightUI(target, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 50)
func OpenUI(target, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 4)
func CameraLookAt(pos, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 5)
func CameraReset(peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 96)
func TriggerInteract(targetRID, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 95)
func TriggerExplore(peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 107)
func TriggerSkill(targetRID, skillID, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 79)
func TargetAlteration(agentRID, targetRID, value, alteration, skillID, hasFeedback, peerID = 0) -> void:
	seen_alterations.append([agentRID, targetRID, value, alteration, skillID])

@rpc("any_peer", "call_remote", "reliable", 6)
func Casted(agentRID, skillID, cooldown, peerID = 0) -> void:
	seen_casts.append([agentRID, skillID, cooldown])

@rpc("any_peer", "call_remote", "reliable", 80)
func ThrowProjectile(agentRID, targetPos, skillID, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 36)
func EntityBuff(agentRID, effect, skillID, enabled, peerID = 0) -> void:
	seen_entity_buffs.append([agentRID, effect, skillID, enabled])

@rpc("any_peer", "call_remote", "reliable", 49)
func Morphed(agentRID, morphID, notifyMorphing, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 114)
func UpdatePublicStats(agentRID, level, health, maxHealth, hairstyle, haircolor, gender, race, skintone, currentShape, peerID = 0) -> void:
	seen_public_stats.append([agentRID, level, health])
	# Learn our own RID: the server pushes UpdatePublicStats for self during
	# enterWorld and after stat changes. The first push for our nick is us.
	if my_rid == 0 and agentRID != 0:
		# We can't be sure which push is ours, but the death/revive broadcasts
		# are inclusive and carry our RID. We'll infer it from context instead.
		pass

@rpc("any_peer", "call_remote", "reliable", 113)
func UpdatePrivateStats(experience, gp, mana, stamina, karma, weight, shape, spirit, peerID = 0) -> void:
	seen_private_stats.append([experience, gp])
@rpc("any_peer", "call_remote", "reliable", 109)
func UpdateAttributes(strength, vitality, agility, endurance, concentration, peerID = 0) -> void:
	seen_attributes = [strength, vitality, agility, endurance, concentration]

@rpc("any_peer", "call_remote", "reliable", 105)
func TriggerSelect(agentRID, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 74)
func SetAttributes(strength, vitality, agility, endurance, concentration, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 44)
func LevelUp(agentRID, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 41)
func ItemAdded(itemID, customfield, count, peerID = 0) -> void:
	seen_item_added.append([itemID, customfield, count])

@rpc("any_peer", "call_remote", "reliable", 43)
func ItemRemoved(itemID, customfield, count, itemIndex, peerID = 0) -> void:
	seen_item_removed.append([itemID, customfield, count, itemIndex])

@rpc("any_peer", "call_remote", "reliable", 42)
func ItemEquiped(agentRID, itemID, customfield, state, itemIndex, peerID = 0) -> void:
	seen_item_equiped.append([agentRID, itemID, customfield, state, itemIndex])

@rpc("any_peer", "call_remote", "reliable", 117)
func UseItem(itemID, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 33)
func DropItem(itemID, customfield, itemCount, itemIndex, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 37)
func EquipItem(itemID, customfield, itemIndex, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 108)
func UnequipItem(itemID, customfield, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 67)
func RetrieveInventory(peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 58)
func RefreshInventory(cells, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 57)
func RefreshEquipment(agentRID, equipment, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 32)
func DropAdded(dropID, itemID, customfield, pos, peerID = 0) -> void:
	seen_drop_added.append([dropID, itemID, pos])

@rpc("any_peer", "call_remote", "reliable", 34)
func DropRemoved(dropID, peerID = 0) -> void:
	seen_drop_removed.append(dropID)

@rpc("any_peer", "call_remote", "reliable", 53)
func PickupDrop(dropID, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 87)
func TradeRespond(accept, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 88)
func TradeSetOffer(entries, gp, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 84)
func TradeLock(revision, state, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 82)
func TradeCancel(peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 86)
func TradeRequested(requesterName, peerID = 0) -> void:
	seen_trade_requested = requesterName

@rpc("any_peer", "call_remote", "reliable", 85)
func TradeOpened(otherName, peerID = 0) -> void:
	seen_trade_opened = otherName

@rpc("any_peer", "call_remote", "reliable", 89)
func TradeUpdate(ownOffer, ownGP, otherOffer, otherGP, ownLocked, otherLocked, revision, peerID = 0) -> void:
	seen_trade_updates.append([ownGP, otherGP, ownLocked, otherLocked, revision])

@rpc("any_peer", "call_remote", "reliable", 83)
func TradeClosed(reason, peerID = 0) -> void:
	seen_trade_closed = reason
	seen_trade_closed_count += 1

@rpc("any_peer", "call_remote", "reliable", 48)
func MerchantTrade(sellEntries, buyEntries, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 47)
func MerchantOpened(merchantName, stockPrices, sellPrices, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 116)
func UpdateSkill(skillID, level, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 111)
func UpdateBuff(effect, value, duration, skillID, peerID = 0) -> void:
	seen_update_buffs.append([effect, value, duration, skillID])

@rpc("any_peer", "call_remote", "reliable", 110)
func UpdateBestiary(mobID, count, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 115)
func UpdateQuest(questID, state, peerID = 0) -> void:
	seen_quest_updates.append([questID, state])

@rpc("any_peer", "call_remote", "reliable", 60)
func RefreshProgress(skills, quests, bestiary, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 66)
func RetrieveCharacterInformation(peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 16)
func CommandFeedback(feedback, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 17)
func CommandModifier(effect, value, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 93)
func TriggerCommand(command, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment

@rpc("any_peer", "call_remote", "reliable", 3)
func BulkCall(methodName, bulkedArgs, peerID = 0) -> void:
	pass  # stub mirroring Network.gd for nameID alignment


func _on_reply(method: String, args: Array) -> void:
	if step >= chain.size():
		return
	var step_def: Dictionary = chain[step]
	var expect: Array = step_def["expect"]
	if got_replies >= expect.size():
		print("E2E FAIL: extra reply ", method)
		fail = true
		return
	var want_name: String = expect[got_replies][0]
	var want_err = expect[got_replies][1]
	got_replies += 1
	if method != want_name:
		print("E2E FAIL: step ", step, " (", step_def["send"], ") reply ", got_replies,
				": want ", want_name, " got ", method, " args=", args)
		fail = true
		return
	if want_err != null:
		if args.is_empty() or args[0] != want_err:
			print("E2E FAIL: step ", step, " (", step_def["send"], ") ", method,
					": want err=", want_err, " got args=", args)
			fail = true
			return
	print("E2E OK: role=", role, " step ", step, " (", step_def["send"], ") reply ",
			got_replies, "/", expect.size(), ": ", method)
	if got_replies == expect.size():
		step += 1
		got_replies = 0
		sent_for_step = false

func _ready() -> void:
	randomize()
	# Diagnostic: count incoming RPC packets.
	set_process(true)
	role = OS.get_environment("E2E_ROLE")
	if role.is_empty():
		role = "solo"
	if role == "peerA":
		my_nick = "Wone"
		peer_nick = "Wtwo"
	elif role == "peerB":
		my_nick = "Wtwo"
		peer_nick = "Wone"
	elif role == "npc":
		my_nick = "Wnpc"
		peer_nick = ""
	elif role == "partyA":
		my_nick = "Wpa"
		peer_nick = "Wpb"
	elif role == "partyB":
		my_nick = "Wpb"
		peer_nick = "Wpa"
	elif role == "tradeA":
		my_nick = "Wta"
		peer_nick = "Wtb"
	elif role == "tradeB":
		my_nick = "Wtb"
		peer_nick = "Wta"
	elif role == "combatA":
		my_nick = "Wca"
		peer_nick = "Wcb"
	elif role == "combatB":
		my_nick = "Wcb"
		peer_nick = "Wca"
	elif role == "itemA":
		my_nick = "Wia"
		peer_nick = ""
	elif role == "buffA":
		my_nick = "Wba"
		peer_nick = "Wbb"
	elif role == "buffB":
		my_nick = "Wbb"
		peer_nick = "Wba"
	chain = _mk_chain("w_%s_%d" % [role, randi() % 100000], my_nick)
	peer = WebSocketMultiplayerPeer.new()
	peer.set_outbound_buffer_size(131071)
	peer.set_inbound_buffer_size(131071)
	peer.create_client(SERVER_URL)
	mp = MultiplayerAPI.create_default_interface()
	mp.set_root_path(get_tree().root.get_path())
	mp.multiplayer_peer = peer
	print("E2E role=", role, " connecting to ", SERVER_URL)

	var recv_counts := {}

func _process(_delta: float) -> void:
	frames += 1
	peer.poll()
	mp.poll()

	if not connected:
		if peer.get_connection_status() == MultiplayerPeer.CONNECTION_CONNECTED:
			connected = true
			connected_frame = frames
			print("E2E CONNECTED role=", role, " after ", frames, " frames")
		elif frames > 300:
			print("E2E FAIL: role=", role, " never connected")
			get_tree().quit(1)
		return

	if fail:
		get_tree().quit(1)
		return

	if step < chain.size():
		if not sent_for_step and frames > connected_frame + step * 5 + 5:
			sent_for_step = true
			var step_def: Dictionary = chain[step]
			print("E2E role=", role, " sending step ", step, ": ", step_def["send"])
			mp.rpc(1, self, step_def["send"], step_def["args"])
			return
		if frames > connected_frame + 600:
			print("E2E FAIL: role=", role, " timeout at step ", step)
			get_tree().quit(1)
		return

	# ---- in game ----
	if role == "solo":
		print("E2E PASS: solo login chain completed after ", frames, " frames")
		get_tree().quit(0)
		return

	if role == "npc":
		_run_npc_role(frames, connected_frame)
		return

	if role == "partyA" or role == "partyB":
		_run_party_role(frames, connected_frame)
		return

	if role == "tradeA" or role == "tradeB":
		_run_trade_role(frames, connected_frame)
		return

	if role == "combatA" or role == "combatB":
		_run_combat_role(frames, connected_frame)
		return

	if role == "itemA":
		_run_item_a(frames, connected_frame)
		return

	if role == "buffA":
		_run_buff_a(frames, connected_frame)
		return

	if role == "buffB":
		_run_buff_b(frames, connected_frame)
		return

	if not (peer_nick in seen_preloads):
		if frames % 60 == 0:
			print("E2E role=", role, " waiting for peer visibility... preloads=", seen_preloads)
		if frames > connected_frame + 600:
			print("E2E FAIL: role=", role, " peer never visible (preloads=", seen_preloads, ")")
			get_tree().quit(1)
		return
	if not seen_online.has(peer_nick):
		print("E2E note: role=", role, " online list not yet showing peer: ", seen_online.keys())
	if not moved_other and seen_move_x.is_empty():
		print("E2E OK: role=", role, " peer visible after ", frames, " frames (preloads=", seen_preloads, ")")

	# Both peers walk right; each watches the *other* entity move. charIDs are
	# assigned in connect order, so the peer's entity is whichever tracked ID
	# is not ours. Our own ID is the one whose position never changes (the
	# server excludes the mover from its own broadcasts).
	# Keep walking until the peer has observed us (re-issue the move command
	# every 30 frames; the server's tick keeps broadcasting while we move).
	if frames % 30 == 0:
		mp.rpc(1, self, "SetMovePos", [Vector2(1, 0)])
		if not moved_other:
			moved_other = true
			print("E2E role=", role, " SetMovePos(1,0)")
	# The peer's entity is identified by PreloadPlayer (peer_rid). Movement is
	# detected by comparing its x against its FIRST observed x, so a single
	# UpdateEntity suffices — a peer may leave right after its first broadcast.
	if peer_rid != 0 and not sent_gameplay and moved_ids.has(peer_rid):
		print("E2E OK: role=", role, " observed peer entity ", peer_rid,
				" moving (x=", seen_move_x[peer_rid], " first=", peer_first_x, ")")
		sent_gameplay = true
		gameplay_sent_frame = frames
		# Exchange chat/emote/cast with the peer.
		mp.rpc(1, self, "TriggerChat", ["LOCAL", "hello from " + role])
		mp.rpc(1, self, "TriggerEmote", [42])
		mp.rpc(1, self, "Casted", [0, 7, 2.5])
		print("E2E role=", role, " sent chat/emote/cast")
		return
	# Verify the peer's gameplay arrived. Chat carries the sender's nick, so
	# filter by nickname (not role) to avoid counting our own LOCAL echo.
	# Emote/Casted carry the sender's agentRID; the peer's RID is peer_rid.
	var peer_chats := 0
	for c in seen_chats:
		if c[1] == peer_nick:
			peer_chats += 1
	var peer_emotes := 0
	for e in seen_emotes:
		if e[0] == peer_rid:
			peer_emotes += 1
	var peer_casts := 0
	for cst in seen_casts:
		if cst[0] == peer_rid:
			peer_casts += 1
	if sent_gameplay and peer_chats > 0 and peer_emotes > 0 and peer_casts > 0:
		# Wait a few frames before quitting so the peer has time to receive
		# our chat/emote/cast broadcasts (otherwise the peer sees us leave
		# before our packets arrive).
		if frames - gameplay_sent_frame < 30:
			return
		print("E2E PASS: role=", role, " world sync + gameplay (chats=", peer_chats,
				" emotes=", peer_emotes, " casts=", peer_casts, " updates=", update_count, ")")
		get_tree().quit(0)
		return
	if frames > connected_frame + 1200:
		if not sent_gameplay:
			print("E2E FAIL: role=", role, " no peer movement observed (seen=", seen_move_x,
					" peer_rid=", peer_rid, " updates=", update_count, " fulls=", full_count,
					" moved=", moved_ids, ")")
		else:
			print("E2E FAIL: role=", role, " gameplay not observed (chats=", peer_chats,
					" emotes=", peer_emotes, " casts=", peer_casts, ")")
		get_tree().quit(1)
	return


# _run_npc_role drives the phase-3 NPC/quest flow end to end:
#   wait for the Elder NPC to preload -> TriggerInteract -> ContextText +
#   ContextContinue -> TriggerNextContext -> ContextText + ContextChoice ->
#   TriggerChoice(0) -> ContextText echo + UpdateQuest(1, active) ->
#   TriggerSelect(1) -> UpdatePublicStats -> PASS.
func _run_npc_role(frames: int, connected_frame: int) -> void:
	if npc_stage == 0:
		# Wait for the Elder NPC to be preloaded into our AOI.
		if seen_npcs.has("Elder"):
			npc_rid = seen_npcs["Elder"]
			print("E2E role=npc Elder preloaded rid=", npc_rid)
			mp.rpc(1, self, "TriggerInteract", [npc_rid])
			npc_stage = 1
			return
		if frames % 60 == 0:
			print("E2E role=npc waiting for Elder (npcs=", seen_npcs, ")")
		if frames > connected_frame + 600:
			print("E2E FAIL: role=npc Elder never preloaded (npcs=", seen_npcs, ")")
			get_tree().quit(1)
		return
	if npc_stage == 1:
		# Expect ContextText (greeting) + ContextContinue (step has no choices).
		if seen_context_texts.size() >= 1 and seen_context_continues >= 1:
			print("E2E OK: role=npc interact -> text='", seen_context_texts[0][1],
					"' + continue")
			mp.rpc(1, self, "TriggerNextContext", [])
			npc_stage = 2
			return
		if frames > connected_frame + 600:
			print("E2E FAIL: role=npc no interact dialogue (texts=", seen_context_texts,
					" continues=", seen_context_continues, ")")
			get_tree().quit(1)
		return
	if npc_stage == 2:
		# Expect ContextText (step 1) + ContextChoice (2 options).
		var want_texts := 2
		if seen_context_texts.size() >= want_texts and seen_context_choices.size() >= 1:
			var choices = seen_context_choices[0]
			print("E2E OK: role=npc next -> text='", seen_context_texts[1][1],
					"' + ", choices.size(), " choices: ", str(choices))
			mp.rpc(1, self, "TriggerChoice", [0])  # "I will help."
			npc_stage = 3
			return
		if frames > connected_frame + 600:
			print("E2E FAIL: role=npc no choice step (texts=", seen_context_texts,
					" choices=", seen_context_choices, ")")
			get_tree().quit(1)
		return
	if npc_stage == 3:
		# Expect the chosen-text echo + UpdateQuest(1, active).
		var quest_active := false
		for q in seen_quest_updates:
			if q[0] == 1 and q[1] == 1:
				quest_active = true
		if seen_context_texts.size() >= 3 and quest_active:
			print("E2E OK: role=npc choice -> echo='", seen_context_texts[2][1],
					"' + UpdateQuest(1, active)")
			mp.rpc(1, self, "TriggerSelect", [npc_rid])
			npc_stage = 4
			return
		if frames > connected_frame + 600:
			print("E2E FAIL: role=npc choice not applied (texts=", seen_context_texts,
					" quests=", seen_quest_updates, ")")
			get_tree().quit(1)
		return
	if npc_stage == 4:
		# Expect UpdatePublicStats for the Elder (agentRID == npc_rid).
		for ps in seen_public_stats:
			if ps[0] == npc_rid:
				print("E2E PASS: role=npc quest flow + select (texts=", seen_context_texts.size(),
						" choices=", seen_context_choices.size(), " quests=", seen_quest_updates,
						" stats=", seen_public_stats, ")")
				get_tree().quit(0)
				return
		if frames > connected_frame + 600:
			print("E2E FAIL: role=npc no public stats (stats=", seen_public_stats, ")")
			get_tree().quit(1)
		return


# _run_party_role drives the phase-3 party flow between two clients:
#   partyA: invite partyB -> wait PartyUpdate -> PARTY chat -> promote B ->
#           wait PartyUpdate -> PASS.
#   partyB: wait PartyInvite -> accept -> wait PartyUpdate -> wait PARTY chat ->
#           wait PartyUpdate (leader changed) -> PASS.
# Both roles self-drive; each prints its own PASS line.
func _run_party_role(frames: int, connected_frame: int) -> void:
	if role == "partyA":
		_run_party_a(frames, connected_frame)
	else:
		_run_party_b(frames, connected_frame)


func _run_party_a(frames: int, connected_frame: int) -> void:
	match party_stage:
		0:
			# Wait for partyB to be visible (preloaded) before inviting, so the
			# server can resolve the nickname.
			if not (peer_nick in seen_preloads):
				if frames % 60 == 0:
					print("E2E role=partyA waiting for ", peer_nick, " (preloads=", seen_preloads, ")")
				if frames > connected_frame + 600:
					print("E2E FAIL: role=partyA peer never visible")
					get_tree().quit(1)
				return
			print("E2E role=partyA inviting ", peer_nick)
			mp.rpc(1, self, "TriggerPartyInvite", [peer_nick])
			party_stage = 1
		1:
			# Wait for the first PartyUpdate (both members).
			if seen_party_updates.size() >= 1:
				print("E2E OK: role=partyA PartyUpdate members=", seen_party_updates[0][0],
						" leader=", seen_party_updates[0][1])
				mp.rpc(1, self, "TriggerChat", ["PARTY", "hello party"])
				mp.rpc(1, self, "TriggerPartyPromote", [peer_nick])
				party_stage = 2
				return
			if frames > connected_frame + 600:
				print("E2E FAIL: role=partyA no PartyUpdate (", seen_party_updates, ")")
				get_tree().quit(1)
		2:
			# Wait for the promote to be reflected (2nd PartyUpdate).
			if seen_party_updates.size() >= 2:
				var leader = seen_party_updates[1][1]
				print("E2E PASS: role=partyA party flow (updates=", seen_party_updates.size(),
						" leader=", leader, ")")
				get_tree().quit(0)
				return
			if frames > connected_frame + 600:
				print("E2E FAIL: role=partyA promote not reflected (", seen_party_updates, ")")
				get_tree().quit(1)


func _run_party_b(frames: int, connected_frame: int) -> void:
	match party_stage:
		0:
			# Wait for the invite, then accept.
			if seen_party_invite != "":
				print("E2E OK: role=partyB invite from ", seen_party_invite)
				mp.rpc(1, self, "TriggerPartyAccept", [])
				party_stage = 1
				return
			if frames > connected_frame + 600:
				print("E2E FAIL: role=partyB no invite")
				get_tree().quit(1)
		1:
			# Wait for PartyUpdate (joined).
			if seen_party_updates.size() >= 1:
				print("E2E OK: role=partyB PartyUpdate members=", seen_party_updates[0][0],
						" leader=", seen_party_updates[0][1])
				party_stage = 2
				return
			if frames > connected_frame + 600:
				print("E2E FAIL: role=partyB no PartyUpdate")
				get_tree().quit(1)
		2:
			# Wait for the PARTY chat from partyA.
			var party_chats := 0
			for c in seen_chats:
				if c[0] == "PARTY" and c[1] == peer_nick:
					party_chats += 1
			if party_chats > 0:
				print("E2E OK: role=partyB PARTY chat from ", peer_nick)
				party_stage = 3
				return
			if frames > connected_frame + 600:
				print("E2E FAIL: role=partyB no PARTY chat (", seen_chats, ")")
				get_tree().quit(1)
		3:
			# Wait for the promote PartyUpdate (leader should now be us).
			if seen_party_updates.size() >= 2:
				var leader = seen_party_updates[1][1]
				print("E2E PASS: role=partyB party flow (updates=", seen_party_updates.size(),
						" leader=", leader, ")")
				get_tree().quit(0)
				return
			if frames > connected_frame + 600:
				print("E2E FAIL: role=partyB promote not reflected (", seen_party_updates, ")")
				get_tree().quit(1)


# _run_trade_role drives the phase-3 trade flow:
#   tradeA: /trade -> wait TradeOpened -> TradeUpdate(rev0) -> SetOffer ->
#           TradeUpdate(rev1) -> Lock(rev1) -> wait TradeClosed -> PASS.
#   tradeB: wait TradeRequested -> TradeRespond(accept) -> wait TradeOpened ->
#           TradeUpdate -> Lock(rev1) -> wait TradeClosed -> PASS.
func _run_trade_role(frames: int, connected_frame: int) -> void:
	if role == "tradeA":
		_run_trade_a(frames, connected_frame)
	else:
		_run_trade_b(frames, connected_frame)


func _run_trade_a(frames: int, connected_frame: int) -> void:
	match trade_stage:
		0:
			# Wait for tradeB to be visible before requesting the trade.
			if not (peer_nick in seen_preloads):
				if frames % 60 == 0:
					print("E2E role=tradeA waiting for ", peer_nick, " (preloads=", seen_preloads, ")")
				if frames > connected_frame + 600:
					print("E2E FAIL: role=tradeA peer never visible")
					get_tree().quit(1)
				return
			print("E2E role=tradeA /trade ", peer_nick)
			mp.rpc(1, self, "TriggerCommand", ["trade \"%s\"" % peer_nick])
			trade_stage = 1
		1:
			if seen_trade_opened != "":
				print("E2E OK: role=tradeA TradeOpened ", seen_trade_opened)
				trade_stage = 2
				return
			if frames > connected_frame + 600:
				print("E2E FAIL: role=tradeA no TradeOpened")
				get_tree().quit(1)
		2:
			# Wait for the initial TradeUpdate (revision 0).
			if seen_trade_updates.size() >= 1:
				print("E2E OK: role=tradeA TradeUpdate rev=", seen_trade_updates[0][4])
				# Offer 2x item 42 + 10 GP.
				var entries := [ [42, 2, ""] ]
				mp.rpc(1, self, "TradeSetOffer", [entries, 10])
				trade_stage = 3
				return
			if frames > connected_frame + 600:
				print("E2E FAIL: role=tradeA no TradeUpdate")
				get_tree().quit(1)
		3:
			# Wait for the offer TradeUpdate (revision 1).
			if seen_trade_updates.size() >= 2:
				print("E2E OK: role=tradeA offer accepted rev=", seen_trade_updates[1][4])
				mp.rpc(1, self, "TradeLock", [1, true])
				trade_stage = 4
				return
			if frames > connected_frame + 600:
				print("E2E FAIL: role=tradeA offer not reflected")
				get_tree().quit(1)
		4:
			if seen_trade_closed != "":
				print("E2E PASS: role=tradeA trade completed (", seen_trade_closed, ")")
				get_tree().quit(0)
				return
			if frames > connected_frame + 600:
				print("E2E FAIL: role=tradeA no TradeClosed (updates=", seen_trade_updates, ")")
				get_tree().quit(1)


func _run_trade_b(frames: int, connected_frame: int) -> void:
	match trade_stage:
		0:
			if seen_trade_requested != "":
				print("E2E OK: role=tradeB requested by ", seen_trade_requested)
				mp.rpc(1, self, "TradeRespond", [true])
				trade_stage = 1
				return
			if frames > connected_frame + 600:
				print("E2E FAIL: role=tradeB no TradeRequested")
				get_tree().quit(1)
		1:
			if seen_trade_opened != "":
				print("E2E OK: role=tradeB TradeOpened ", seen_trade_opened)
				trade_stage = 2
				return
			if frames > connected_frame + 600:
				print("E2E FAIL: role=tradeB no TradeOpened")
				get_tree().quit(1)
		2:
			# Wait for the initial TradeUpdate (revision 0), then accept A's offer.
			if seen_trade_updates.size() >= 1:
				trade_stage = 3
				return
			if frames > connected_frame + 600:
				print("E2E FAIL: role=tradeB no TradeUpdate")
				get_tree().quit(1)
		3:
			# Wait for A's offer (revision 1), then lock.
			if seen_trade_updates.size() >= 2:
				print("E2E OK: role=tradeB offer rev=", seen_trade_updates[1][4])
				mp.rpc(1, self, "TradeLock", [1, true])
				trade_stage = 4
				return
			if frames > connected_frame + 600:
				print("E2E FAIL: role=tradeB offer not reflected")
				get_tree().quit(1)
		4:
			if seen_trade_closed != "":
				print("E2E PASS: role=tradeB trade completed (", seen_trade_closed, ")")
				get_tree().quit(0)
				return
			if frames > connected_frame + 600:
				print("E2E FAIL: role=tradeB no TradeClosed (updates=", seen_trade_updates, ")")
				get_tree().quit(1)


# _run_combat_role drives the phase-3 combat flow:
#   combatA: wait for combatB visibility -> cast Flar repeatedly until B dies
#            -> wait for B's death broadcast -> wait for B's respawn -> PASS.
#   combatB: wait for A's death broadcast on itself -> send TriggerRespawn ->
#            wait for WarpPlayer + revive -> PASS.
# The server is authoritative: it computes damage, broadcasts TargetAlteration
# to AOI neighbours (inclusive), and pushes FullUpdateEntity on death/revive.
func _run_combat_role(frames: int, connected_frame: int) -> void:
	if role == "combatA":
		_run_combat_a(frames, connected_frame)
	else:
		_run_combat_b(frames, connected_frame)


func _run_combat_a(frames: int, connected_frame: int) -> void:
	match combat_stage:
		0:
			# Wait for combatB to be visible (preloaded) so we can target it.
			if not (peer_nick in seen_preloads):
				if frames % 60 == 0:
					print("E2E role=combatA waiting for ", peer_nick, " (preloads=", seen_preloads, ")")
				if frames > connected_frame + 600:
					print("E2E FAIL: role=combatA peer never visible")
					get_tree().quit(1)
				return
			# peer_rid is set by PreloadPlayer; peer_pos by UpdateEntity. A
			# position may have arrived before peer_rid was known, so resolve
			# it from the per-RID cache now.
			if peer_rid != 0 and last_pos_by_rid.has(peer_rid):
				peer_pos = last_pos_by_rid[peer_rid]
			if peer_pos == Vector2.ZERO:
				if frames % 60 == 0:
					print("E2E role=combatA waiting for ", peer_nick, " position (rid=", peer_rid, ")")
				if frames > connected_frame + 600:
					print("E2E FAIL: role=combatA no peer position (rid=", peer_rid, ")")
					get_tree().quit(1)
				return
			# Walk to B's last known position. The server's spawn grid can place
			# us up to ~350px apart, beyond Flar's 184px reach. SetClickPos moves
			# us toward B at 120 px/s.
			if not combat_walk_sent:
				combat_walk_sent = true
				combat_start_frame = frames
				print("E2E role=combatA walking to ", peer_nick, " at ", peer_pos, " (rid=", peer_rid, ")")
				mp.rpc(1, self, "SetClickPos", [peer_pos])
				return
			# Wait for arrival: the server's walk speed is 120 px/s, so a 350px
			# walk takes ~3s = 180 frames. We use a generous 240-frame wait.
			if frames - combat_start_frame > 240:
				print("E2E role=combatA arrived, casting Flar")
				combat_stage = 1
		1:
			# Cast Flar repeatedly until B dies.
			if seen_casts.size() >= combat_casts and seen_alterations.size() >= combat_casts:
				if seen_deaths.has(peer_rid):
					print("E2E OK: role=combatA B died after ", combat_casts, " casts")
					combat_stage = 2
					return
				if combat_casts < 10:
					mp.rpc(1, self, "TriggerSkill", [peer_rid, combat_skill_id])
					combat_casts += 1
					return
				if frames > connected_frame + 900:
					print("E2E FAIL: role=combatA B did not die (casts=", combat_casts, ")")
					get_tree().quit(1)
			if frames > connected_frame + 900:
				print("E2E FAIL: role=combatA no cast feedback (casts=", combat_casts,
					" seen_casts=", seen_casts.size(), " alts=", seen_alterations.size(), ")")
				get_tree().quit(1)
		2:
			# Wait for B to respawn (revive broadcast arrives as FullUpdateEntity).
			if seen_revives.has(peer_rid):
				print("E2E PASS: role=combatA combat flow (casts=", combat_casts,
						" alts=", seen_alterations.size(), " deaths=", seen_deaths.size(),
						" revives=", seen_revives.size(), ")")
				get_tree().quit(0)
				return
			if frames > connected_frame + 900:
				print("E2E FAIL: role=combatA B did not respawn (deaths=", seen_deaths,
					" revives=", seen_revives, ")")
				get_tree().quit(1)


func _run_combat_b(frames: int, connected_frame: int) -> void:
	match combat_stage:
		0:
			# Wait for A to kill us: the server pushes FullUpdateEntity(state=DEATH)
			# to us when our health reaches zero. The death broadcast is inclusive
			# (sent to the victim too), so any death we see is our own.
			if seen_deaths.size() > 0:
				print("E2E OK: role=combatB died (death broadcast received, rid=", seen_deaths[0], ")")
				combat_stage = 1
				return
			if frames % 60 == 0:
				print("E2E role=combatB waiting for death (deaths=", seen_deaths, ")")
			if frames > connected_frame + 900:
				print("E2E FAIL: role=combatB never died (deaths=", seen_deaths, ")")
				get_tree().quit(1)
		1:
			# Respawn: TriggerRespawn -> server pushes WarpPlayer + FullUpdateEntity.
			print("E2E role=combatB triggering respawn")
			mp.rpc(1, self, "TriggerRespawn", [])
			combat_stage = 2
		2:
			if seen_revives.size() > 0:
				print("E2E PASS: role=combatB respawned (revives=", seen_revives.size(), ")")
				get_tree().quit(0)
				return
			if frames > connected_frame + 900:
				print("E2E FAIL: role=combatB no respawn (revives=", seen_revives, ")")
				get_tree().quit(1)


# _run_item_a drives the phase-4 items/economy flow:
#   PickupDrop (gain potions) -> MerchantTrade sell (earn GP) ->
#   MerchantTrade buy (spend GP) -> UseItem (consume) ->
#   SetAttributes (echo) -> EquipItem (non-equipable should be rejected).
# Item IDs: Potion=1001 (usable, stackable), Antidote=1002 (usable),
# Iron Sword=1003 (WEAPON), Leather Armor=1004 (CHEST),
# Wooden Shield=1005 (SHIELD), Cloth Boots=1006 (FEET).
var item_stage := 0
var item_sub := 0
var buff_stage := 0
var buff_walk_sent := false
var buff_start_frame := 0
var buff_casts := 0

func _run_item_a(frames: int, connected_frame: int) -> void:
	match item_stage:
		0:
			# Pickup 3 potions. The server spawns 3 starter potions near our
			# spawn; their DropAdded pushes carry [dropID, itemID, customfield,
			# pos]. Filter by proximity (<=64 px from our position) because
			# broadcastToMap sends us every player's drops.
			if item_sub < 3:
				for d in seen_drop_added:
					if d[1] != 1001:
						continue
					if pickup_sent_drops.has(d[0]):
						continue
					var drop_pos: Vector2 = d[2]
					if my_pos.distance_to(drop_pos) <= 64.0:
						mp.rpc(1, self, "PickupDrop", [d[0]])
						pickup_sent_drops.append(d[0])
						item_sub += 1
						if item_sub >= 3:
							break
				return
			if seen_item_added.size() >= 3:
				print("E2E OK: role=itemA picked up 3 potions (added=", seen_item_added.size(), ")")
				item_stage = 1
				item_sub = 0
				return
			if frames % 60 == 0:
				print("E2E role=itemA waiting for pickups (added=", seen_item_added.size(),
						" drops=", seen_drop_added.size(), " my_pos=", my_pos, ")")
			if frames > connected_frame + 600:
				print("E2E FAIL: role=itemA pickups not registered")
				get_tree().quit(1)
		1:
			# Sell 3 potions -> +75 GP.
			mp.rpc(1, self, "MerchantTrade", [[[1001, 3, ""]], []])
			item_stage = 2
		2:
			if seen_trade_closed_count >= 1:
				var reason: String = seen_trade_closed
				if reason != "Trade completed.":
					print("E2E FAIL: role=itemA sell rejected: ", reason)
					get_tree().quit(1)
				print("E2E OK: role=itemA sold 3 potions (+75 GP)")
				item_stage = 3
				return
			if frames > connected_frame + 600:
				print("E2E FAIL: role=itemA sell no TradeClosed")
				get_tree().quit(1)
		3:
			# Buy 1 Antidote (30 GP) -> should succeed (75-30=45).
			mp.rpc(1, self, "MerchantTrade", [[], [[1002, 1, ""]]])
			item_stage = 4
		4:
			if seen_trade_closed_count >= 2:
				var reason: String = seen_trade_closed
				if reason != "Trade completed.":
					print("E2E FAIL: role=itemA buy rejected: ", reason)
					get_tree().quit(1)
				print("E2E OK: role=itemA bought Antidote (-30 GP)")
				item_stage = 5
				return
			if frames > connected_frame + 600:
				print("E2E FAIL: role=itemA buy no TradeClosed")
				get_tree().quit(1)
		5:
			# UseItem on the Antidote.
			mp.rpc(1, self, "UseItem", [1002])
			item_stage = 6
		6:
			if seen_item_removed.size() >= 1:
				var r: Array = seen_item_removed[0]
				if r[0] != 1002 or r[2] != 1:
					print("E2E FAIL: role=itemA ItemRemoved want 1002x1, got ", r)
					get_tree().quit(1)
				print("E2E OK: role=itemA used Antidote (removed)")
				item_stage = 7
				return
			if frames > connected_frame + 600:
				print("E2E FAIL: role=itemA use no ItemRemoved")
				get_tree().quit(1)
		7:
			# SetAttributes.
			mp.rpc(1, self, "SetAttributes", [2, 1, 1, 0, 0])
			item_stage = 8
		8:
			if seen_attributes.size() >= 5:
				if seen_attributes[0] != 2 or seen_attributes[1] != 1 or seen_attributes[2] != 1:
					print("E2E FAIL: role=itemA UpdateAttributes want 2/1/1, got ", seen_attributes)
					get_tree().quit(1)
				print("E2E OK: role=itemA set attributes ", seen_attributes)
				item_stage = 9
				return
			if frames > connected_frame + 600:
				print("E2E FAIL: role=itemA no UpdateAttributes")
				get_tree().quit(1)
		9:
			# EquipItem on a non-equipable item (Antidote, slot -1) must NOT push.
			mp.rpc(1, self, "EquipItem", [1002, "", 0])
			item_stage = 10
		10:
			if frames - connected_frame > 60:
				if seen_item_equiped.size() > 0:
					print("E2E FAIL: role=itemA equip non-equipable pushed ItemEquiped: ", seen_item_equiped)
					get_tree().quit(1)
			print("E2E PASS: role=itemA items/economy flow (added=", seen_item_added.size(),
				" removed=", seen_item_removed.size(), " attributes=", seen_attributes,
				" tradeClosed=", seen_trade_closed_count, ")")
			get_tree().quit(0)


func _run_buff_a(frames: int, connected_frame: int) -> void:
	match buff_stage:
		0:
			# Wait for the peer to be visible so we can target it.
			if not (peer_nick in seen_preloads):
				if frames % 60 == 0:
					print("E2E role=buffA waiting for ", peer_nick)
				if frames > connected_frame + 600:
					print("E2E FAIL: role=buffA peer never visible")
					get_tree().quit(1)
				return
			if peer_rid == 0:
				return
			# Walk to the peer (Inma range is 128+32+32=192px; spawn can be ~350px).
			if not buff_walk_sent:
				buff_walk_sent = true
				buff_start_frame = frames
				mp.rpc(1, self, "SetClickPos", [peer_pos])
				print("E2E role=buffA walking to ", peer_nick, " at ", peer_pos)
				return
			if frames - buff_start_frame > 240:
				print("E2E role=buffA arrived, casting Inma (defense debuff)")
				buff_stage = 1
		1:
			# Cast Inma repeatedly until we see an EntityBuff with enabled=true.
			if seen_entity_buffs.size() > 0:
				var found := false
				for b in seen_entity_buffs:
					if b[1] == 14 and b[3] == true:  # modDefense, enabled
						found = true
				if found:
					print("E2E OK: role=buffA EntityBuff received (defense debuff)")
					buff_stage = 2
					return
			if buff_casts < 5:
				mp.rpc(1, self, "TriggerSkill", [peer_rid, 2089220490])  # Inma
				buff_casts += 1
				return
			if frames > connected_frame + 900:
				print("E2E FAIL: role=buffA no EntityBuff (buffs=", seen_entity_buffs, ")")
				get_tree().quit(1)
		2:
			# EntityBuff is broadcast to the AOI (including the caster), so we
			# see the debuff on our target. UpdateBuff is sent to the target's
			# own client (buffB), not to us — buffA only verifies the broadcast.
			print("E2E PASS: role=buffA buff flow (entity_buffs=", seen_entity_buffs.size(), ")")
			get_tree().quit(0)


func _run_buff_b(frames: int, connected_frame: int) -> void:
	# Passive role: stay alive and wait for buffA to finish. The server's
	# EntityBuff broadcast is inclusive, so we see our own buffs too.
	# UpdateBuff is sent to the target (us), so we should see it as well.
	if frames > connected_frame + 900:
		if seen_entity_buffs.size() > 0 and seen_update_buffs.size() > 0:
			print("E2E PASS: role=buffB received EntityBuff (count=", seen_entity_buffs.size(),
				") + UpdateBuff (count=", seen_update_buffs.size(), ")")
			get_tree().quit(0)
		else:
			print("E2E FAIL: role=buffB no EntityBuff/UpdateBuff (entity_buffs=",
				seen_entity_buffs.size(), " update_buffs=", seen_update_buffs.size(), ")")
			get_tree().quit(1)

