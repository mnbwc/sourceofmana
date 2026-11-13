extends Node

# Remote AI client: connects to the HK VPS Go server via --server-address.
# Mirrors keepalive_client.gd's login chain but uses LauncherCommons to
# resolve the server address from the command line.

const DEFAULT_REMOTE := "ws://154.222.30.52:27012/ws"

var role := "solo"
var my_nick := "Wremote"
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
	pass

@rpc("any_peer", "call_remote", "reliable", 45)
func LoginWithPassword(accountName, password, rememberMe, platform = 0, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 1)
func AuthError(err, peerID = 0) -> void:
	_on_reply("AuthError", [err, peerID])

@rpc("any_peer", "call_remote", "reliable", 46)
func LoginWithToken(accountName, token, platform = 0, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 2)
func AuthTokenResult(accountName, token, peerID = 0) -> void:
	_on_reply("AuthTokenResult", [accountName, token, peerID])

@rpc("any_peer", "call_remote", "reliable", 64)
func RequestPasswordReset(accountName, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 18)
func ConfirmPasswordReset(accountName, code, newPassword, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 7)
func ChangePassword(currentPassword, newPassword, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 28)
func DisconnectAccount(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 9)
func CharacterInfo(info, equipment, peerID = 0) -> void:
	_on_reply("CharacterInfo", [info, equipment, peerID])

@rpc("any_peer", "call_remote", "reliable", 26)
func CreateCharacter(charName, traits, attributes, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 27)
func DeleteCharacter(charName, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 19)
func ConnectCharacter(nickname, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 8)
func CharacterError(err, peerID = 0) -> void:
	_on_reply("CharacterError", [err, peerID])

@rpc("any_peer", "call_remote", "reliable", 29)
func DisconnectCharacter(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 10)
func CharacterListing(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 63)
func RequestOnlineList(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 59)
func RefreshOnlineList(players, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 0)
func AddOnlinePlayer(playerName, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 62)
func RemoveOnlinePlayer(playerName, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 65)
func RequestRtcUpgrade(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 71)
func RtcConfig(iceServers, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 72)
func RtcOffer(sdp, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 68)
func RtcAnswer(sdp, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 69)
func RtcCandidateToClient(media, index, candidateName, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 70)
func RtcCandidateToServer(media, index, candidateName, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 73)
func RtcReady(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 104)
func TriggerRespawn(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 118)
func WarpPlayer(mapID, playerPos, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 54)
func PreloadEntity(agentRID, actorType, currentShape, nick, defaultState, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 55)
func PreloadPlayer(agentRID, spirit, currentShape, nick, level, health, hairstyle, haircolor, gender, race, skintone, equipment, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 61)
func RemoveEntity(agentRID, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 31)
func DisplayProgressionTracker(label, value, maxValue, unit, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 15)
func ClearProgressionTracker(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 30)
func DisplayActions(actions, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 56)
func PushNotification(notif, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 75)
func SetClickPos(pos, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 76)
func SetMovePos(pos, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 77)
func SetOrientation(direction, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 112)
func UpdateEntity(agentRID, velocity, position, frameID, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 39)
func FullUpdateEntity(agentRID, velocity, position, orientation, agentState, skillCastID, isRunning, frameID, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 14)
func ClearNavigation(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 78)
func SetViewportSize(halfWidth, halfHeight, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 94)
func TriggerEmote(emoteID, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 35)
func Emote(senderagentRID, emoteID, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 106)
func TriggerSit(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 38)
func Express(agentRID, text, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 90)
func TriggerChat(channelName, text, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 12)
func ChatQuery(channelName, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 100)
func TriggerPartyInvite(targetNick, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 98)
func TriggerPartyAccept(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 99)
func TriggerPartyDecline(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 102)
func TriggerPartyLeave(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 101)
func TriggerPartyKick(targetNick, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 103)
func TriggerPartyPromote(targetNick, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 51)
func PartyInvite(inviterName, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 52)
func PartyUpdate(members, leaderRID, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 11)
func ChatPlayer(channelName, callerName, text, agentRID, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 13)
func ChatSystem(channelName, text, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 81)
func ToggleContext(enable, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 23)
func ContextText(author, text, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 24)
func ContextThink(author, text, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 22)
func ContextContinue(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 21)
func ContextClose(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 20)
func ContextChoice(texts, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 91)
func TriggerChoice(choiceID, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 92)
func TriggerCloseContext(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 97)
func TriggerNextContext(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 40)
func HighlightUI(target, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 50)
func OpenUI(target, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 4)
func CameraLookAt(pos, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 5)
func CameraReset(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 96)
func TriggerInteract(targetRID, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 95)
func TriggerExplore(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 107)
func TriggerSkill(targetRID, skillID, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 79)
func TargetAlteration(agentRID, targetRID, value, alteration, skillID, hasFeedback, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 6)
func Casted(agentRID, skillID, cooldown, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 80)
func ThrowProjectile(agentRID, targetPos, skillID, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 36)
func EntityBuff(agentRID, effect, skillID, enabled, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 49)
func Morphed(agentRID, morphID, notifyMorphing, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 114)
func UpdatePublicStats(agentRID, level, health, maxHealth, hairstyle, haircolor, gender, race, skintone, currentShape, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 113)
func UpdatePrivateStats(experience, gp, mana, stamina, karma, weight, shape, spirit, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 109)
func UpdateAttributes(strength, vitality, agility, endurance, concentration, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 105)
func TriggerSelect(agentRID, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 74)
func SetAttributes(strength, vitality, agility, endurance, concentration, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 44)
func LevelUp(agentRID, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 41)
func ItemAdded(itemID, customfield, count, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 43)
func ItemRemoved(itemID, customfield, count, itemIndex, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 42)
func ItemEquiped(agentRID, itemID, customfield, state, itemIndex, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 117)
func UseItem(itemID, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 33)
func DropItem(itemID, customfield, itemCount, itemIndex, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 37)
func EquipItem(itemID, customfield, itemIndex, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 108)
func UnequipItem(itemID, customfield, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 67)
func RetrieveInventory(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 58)
func RefreshInventory(cells, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 57)
func RefreshEquipment(agentRID, equipment, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 32)
func DropAdded(dropID, itemID, customfield, pos, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 34)
func DropRemoved(dropID, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 53)
func PickupDrop(dropID, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 87)
func TradeRespond(accept, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 88)
func TradeSetOffer(entries, gp, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 84)
func TradeLock(revision, state, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 82)
func TradeCancel(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 86)
func TradeRequested(requesterName, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 85)
func TradeOpened(otherName, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 89)
func TradeUpdate(ownOffer, ownGP, otherOffer, otherGP, ownLocked, otherLocked, revision, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 83)
func TradeClosed(reason, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 48)
func MerchantTrade(sellEntries, buyEntries, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 47)
func MerchantOpened(merchantName, stockPrices, sellPrices, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 116)
func UpdateSkill(skillID, level, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 111)
func UpdateBuff(effect, value, duration, skillID, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 110)
func UpdateBestiary(mobID, count, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 115)
func UpdateQuest(questID, state, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 60)
func RefreshProgress(skills, quests, bestiary, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 66)
func RetrieveCharacterInformation(peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 16)
func CommandFeedback(feedback, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 17)
func CommandModifier(effect, value, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 93)
func TriggerCommand(command, peerID = 0) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 3)
func BulkCall(methodName, bulkedArgs, peerID = 0) -> void:
	pass


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
	set_process(true)
	role = OS.get_environment("E2E_ROLE")
	if role.is_empty():
		role = "solo"
	# Build a unique account name to avoid collisions
	var suffix := str(randi() % 100000)
	chain = _mk_chain("remote_%s" % suffix, "R%s" % suffix.substr(0, 4))
	peer = WebSocketMultiplayerPeer.new()
	peer.set_outbound_buffer_size(131071)
	peer.set_inbound_buffer_size(131071)

	# Resolve server address: --server-address overrides, else default remote.
	# Parse from OS command-line args (LauncherCommons is not available in the
	# goserver testdata project).
	var server_host := "154.222.30.52"
	var server_port := 27012
	var cargs : PackedStringArray = OS.get_cmdline_args()
	for ua in OS.get_cmdline_user_args():
		cargs.append(ua)
	var i : int = cargs.find("--server-address")
	if i != -1 and i + 1 < cargs.size():
		server_host = cargs[i + 1]
	i = cargs.find("--ws-port")
	if i != -1 and i + 1 < cargs.size():
		server_port = int(cargs[i + 1])
	var server_url := "ws://%s:%d/ws" % [server_host, server_port]
	print("E2E role=", role, " connecting to ", server_url)
	peer.create_client(server_url)
	mp = MultiplayerAPI.create_default_interface()
	mp.set_root_path(get_tree().root.get_path())
	mp.multiplayer_peer = peer

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
			print("E2E FAIL: role=", role, " never connected to ", peer.get_peer_host())
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

	# ---- in game: keepalive ----
	print("E2E PASS: role=", role, " remote login chain completed after ", frames, " frames")
	_keepalive()

func _keepalive() -> void:
	while true:
		await get_tree().process_frame
		frames += 1
		peer.poll()
		mp.poll()
		if frames % 300 == 0:
			print("KEEPALIVE: still connected at frame ", frames)
