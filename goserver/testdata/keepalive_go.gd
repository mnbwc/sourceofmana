extends Node

const SERVER_URL := "ws://127.0.0.1:27012/ws"

var peer: WebSocketMultiplayerPeer
var mp: SceneMultiplayer
var frames := 0
var connected := false
var connected_frame := 0
var step := 0
var got_replies := 0
var sent_for_step := false
var account_name := "keep_go_%d" % [randi() % 100000]
var char_name := "KeepGo"

func _ready() -> void:
	randomize()
	peer = WebSocketMultiplayerPeer.new()
	peer.set_outbound_buffer_size(131071)
	peer.set_inbound_buffer_size(131071)
	peer.create_client(SERVER_URL)
	mp = MultiplayerAPI.create_default_interface()
	mp.set_root_path(get_tree().root.get_path())
	mp.multiplayer_peer = peer
	print("KEEPALIVE connecting to ", SERVER_URL)

func _process(_delta: float) -> void:
	frames += 1
	peer.poll()
	mp.poll()

	if not connected:
		if peer.get_connection_status() == MultiplayerPeer.CONNECTION_CONNECTED:
			connected = true
			connected_frame = frames
			print("KEEPALIVE CONNECTED after ", frames, " frames")
		elif frames > 300:
			print("KEEPALIVE FAIL: timeout connecting")
			get_tree().quit(1)
			return

	if connected and step < 5:
		if not sent_for_step and frames > connected_frame + step * 5 + 5:
			match step:
				0:
					print("KEEPALIVE step 0: CreateAccount ", account_name)
					mp.rpc(1, self, "CreateAccount", [account_name, "secret1", "keep@t.c", true, 0])
				1:
					print("KEEPALIVE step 1: CharacterListing")
					mp.rpc(1, self, "CharacterListing", [])
				2:
					print("KEEPALIVE step 2: CreateCharacter ", char_name)
					mp.rpc(1, self, "CreateCharacter", [char_name, {"race": 1, "gender": 0}, {}])
				3:
					print("KEEPALIVE step 3: CharacterListing")
					mp.rpc(1, self, "CharacterListing", [])
				4:
					print("KEEPALIVE step 4: ConnectCharacter ", char_name)
					mp.rpc(1, self, "ConnectCharacter", [char_name])
			sent_for_step = true
		return

	if connected and step >= 5:
		if frames % 300 == 0:
			print("KEEPALIVE still connected at frame ", frames)

@rpc("any_peer", "call_remote", "reliable", 25)
func CreateAccount(accountName, password, email, rememberMe, platform = 0, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 45)
func LoginWithPassword(accountName, password, rememberMe, platform = 0, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 1)
func AuthError(err, peerID = 0) -> void:
	print("KEEPALIVE AuthError err=", err)
@rpc("any_peer", "call_remote", "reliable", 46)
func AuthTokenResult(accountName, token, peerID = 0) -> void:
	print("KEEPALIVE AuthTokenResult")
@rpc("any_peer", "call_remote", "reliable", 2)
func CharacterInfo(info, equipment, peerID = 0) -> void:
	print("KEEPALIVE CharacterInfo")
@rpc("any_peer", "call_remote", "reliable", 64)
func CharacterListing(peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 18)
func CharacterError(err, peerID = 0) -> void:
	print("KEEPALIVE CharacterError err=", err)
	if err == 0:
		got_replies += 1
		if got_replies >= 1:
			got_replies = 0
			step += 1
			sent_for_step = false
@rpc("any_peer", "call_remote", "reliable", 7)
func ConnectCharacter(nickname, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 28)
func CreateCharacter(charName, traits, attributes, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 9)
func DeleteCharacter(charName, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 26)
func DisconnectAccount(peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 27)
func DisconnectCharacter(peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 19)
func RequestOnlineList(peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 8)
func RefreshOnlineList(players, peerID = 0) -> void:
	print("KEEPALIVE RefreshOnlineList: ", players)
@rpc("any_peer", "call_remote", "reliable", 29)
func AddOnlinePlayer(playerName, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 10)
func RemoveOnlinePlayer(playerName, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 63)
func RequestRtcUpgrade(peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 59)
func RtcConfig(iceServers, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 0)
func RtcOffer(sdp, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 62)
func RtcAnswer(sdp, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 65)
func RtcCandidateToClient(media, index, candidateName, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 71)
func RtcCandidateToServer(media, index, candidateName, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 72)
func RtcReady(peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 68)
func TriggerRespawn(peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 69)
func WarpPlayer(mapID, playerPos, peerID = 0) -> void:
	print("KEEPALIVE WarpPlayer mapID=", mapID, " pos=", playerPos)
@rpc("any_peer", "call_remote", "reliable", 70)
func PreloadEntity(agentRID, actorType, currentShape, nick, defaultState, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 73)
func PreloadPlayer(agentRID, spirit, currentShape, nick, level, health, hairstyle, haircolor, gender, race, skintone, equipment, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 104)
func RemoveEntity(agentRID, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 118)
func DisplayProgressionTracker(label, value, maxValue, unit, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 54)
func ClearProgressionTracker(peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 55)
func DisplayActions(actions, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 61)
func PushNotification(notif, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 15)
func SetClickPos(pos, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 30)
func SetMovePos(pos, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 56)
func SetOrientation(direction, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 75)
func UpdateEntity(agentRID, velocity, position, frameID, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 76)
func FullUpdateEntity(agentRID, velocity, position, orientation, agentState, skillCastID, isRunning, frameID, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 77)
func ClearNavigation(peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 112)
func SetViewportSize(halfWidth, halfHeight, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 39)
func TriggerEmote(emoteID, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 14)
func Emote(senderagentRID, emoteID, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 78)
func TriggerSit(peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 94)
func Express(agentRID, text, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 35)
func TriggerChat(channelName, text, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 106)
func ChatPlayer(channelName, callerName, text, agentRID, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 38)
func ChatSystem(channelName, text, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 90)
func ChatQuery(channelName, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 12)
func TriggerPartyInvite(targetNick, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 100)
func TriggerPartyAccept(peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 98)
func TriggerPartyDecline(peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 99)
func TriggerPartyLeave(peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 102)
func TriggerPartyKick(targetNick, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 101)
func TriggerPartyPromote(targetNick, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 103)
func PartyInvite(inviterName, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 51)
func PartyUpdate(members, leaderRID, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 52)
func TriggerInteract(targetRID, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 11)
func TriggerExplore(peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 13)
func TriggerSkill(targetRID, skillID, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 81)
func TriggerChoice(choiceID, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 23)
func TriggerCloseContext(peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 24)
func TriggerNextContext(peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 22)
func HighlightUI(target, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 21)
func OpenUI(target, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 20)
func CameraLookAt(pos, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 91)
func CameraReset(peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 92)
func TargetAlteration(agentRID, targetRID, value, alteration, skillID, hasFeedback, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 97)
func Casted(agentRID, skillID, cooldown, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 40)
func ThrowProjectile(agentRID, targetPos, skillID, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 50)
func EntityBuff(agentRID, effect, skillID, enabled, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 4)
func Morphed(agentRID, morphID, notifyMorphing, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 5)
func UpdatePublicStats(agentRID, level, health, maxHealth, hairstyle, haircolor, gender, race, skintone, currentShape, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 96)
func UpdatePrivateStats(experience, gp, mana, stamina, karma, weight, shape, spirit, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 95)
func SetAttributes(strength, vitality, agility, endurance, concentration, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 107)
func UpdateAttributes(strength, vitality, agility, endurance, concentration, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 79)
func TriggerSelect(agentRID, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 6)
func LevelUp(agentRID, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 80)
func ItemAdded(itemID, customfield, count, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 36)
func ItemRemoved(itemID, customfield, count, itemIndex, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 33)
func ItemEquiped(agentRID, itemID, customfield, state, itemIndex, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 117)
func UseItem(itemID, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 37)
func EquipItem(itemID, customfield, itemIndex, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 108)
func UnequipItem(itemID, customfield, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 67)
func RetrieveInventory(peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 58)
func RefreshInventory(cells, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 57)
func RefreshEquipment(agentRID, equipment, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 32)
func DropAdded(dropID, itemID, customfield, pos, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 34)
func DropRemoved(dropID, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 53)
func PickupDrop(dropID, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 87)
func TradeRespond(accept, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 88)
func TradeSetOffer(entries, gp, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 84)
func TradeLock(revision, state, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 82)
func TradeCancel(peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 86)
func TradeRequested(requesterName, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 85)
func TradeOpened(otherName, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 89)
func TradeUpdate(ownOffer, ownGP, otherOffer, otherGP, ownLocked, otherLocked, revision, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 83)
func TradeClosed(reason, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 48)
func MerchantTrade(sellEntries, buyEntries, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 47)
func MerchantOpened(merchantName, stockPrices, sellPrices, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 116)
func UpdateSkill(skillID, level, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 111)
func UpdateBuff(effect, value, duration, skillID, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 110)
func UpdateBestiary(mobID, count, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 115)
func UpdateQuest(questID, state, peerID = 0) -> void: pass
@rpc("any_peer", "call_remote", "reliable", 60)
func RefreshProgress(skills, quests, bestiary, peerID = 0) -> void: pass
