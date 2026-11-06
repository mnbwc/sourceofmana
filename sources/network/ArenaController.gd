# Arena Controller (M2) — episode 生命周期 + 难度模板 + 权威终局分。
# 计分全部来自 AgentBridge 已有钩子（_score 字典），本控制器只做：
#   start（warp 到模板 spawn + 快照 xp）→ 镜像增量 → 结束判定 → episode_end。
# 零上游改动；红线同 Kaetram arena.ts：服务端钩子计分、不自报、isAgent 只用于审计。
extends Node

const TEMPLATES := {
	"tutorial_maggot": {
		"label": "新手：蛆虫×3",
		"spawn": Vector2(2230, 1250),  # Maggot 纯净区（world spawn id17）
		"targetNick": "Maggot",
		"killsRequired": 3,
		"maxDurationMSec": 300000,
		"maxDeaths": 3,
		"brief": "击杀 3 只蛆虫。安全区，适合热身。"
	},
	"grind_maggot": {
		"label": "进阶：蛆虫×10",
		"spawn": Vector2(2230, 1250),
		"targetNick": "Maggot",
		"killsRequired": 10,
		"maxDurationMSec": 600000,
		"maxDeaths": 2,
		"brief": "击杀 10 只蛆虫，注意血量与 respawn。"
	}
}

var _episodes : Dictionary = {}  # peerID -> episode dict
var _counter : int = 0

func _ready() -> void:
	if not "--server" in OS.get_cmdline_args():
		set_process(false)
		return
	# MVP-10 ARENA_MODE（玩法模块开关，默认 off）：off 时模块零感知——不处理 episode、
	# 拒绝 arena_start（UNKNOWN_OP 语义由 op 分支的门控实现）
	if not ArenaModeEnabled():
		arenaEnabled = false
		set_process(false)

static func ArenaModeEnabled() -> bool:
	if OS.get_environment("ARENA_MODE").to_lower() == "on":
		return true
	return "--arena-mode=on" in OS.get_cmdline_args()

var arenaEnabled : bool = true

func _process(_delta : float) -> void:
	for peerID in _episodes.keys().duplicate():
		var ep : Dictionary = _episodes[peerID]
		if ep.get("status") != "active":
			continue
		_mirror_score(peerID, ep)
		var elapsed : int = Time.get_ticks_msec() - int(ep.get("startedAtMsec", 0))
		if elapsed >= int(ep["template"].get("maxDurationMSec", 300000)):
			_finish(peerID, ep, "expired")
			continue
		if int(ep["score"].get("kills", 0)) >= int(ep["template"].get("killsRequired", 1)):
			_finish(peerID, ep, "kills_required_met")
			continue
		if int(ep["score"].get("deaths", 0)) > int(ep["template"].get("maxDeaths", 1)):
			_finish(peerID, ep, "max_deaths")

# ---- 生命周期（由 AgentBridge op 调用） --------------------------------

func Start(peerID : int, nick : String, templateKey : String) -> Dictionary:
	if _episodes.has(peerID) and _episodes[peerID].get("status") == "active":
		return {"ok": false, "code": "EPISODE_ACTIVE"}
	if not TEMPLATES.has(templateKey):
		return {"ok": false, "code": "UNKNOWN_TEMPLATE", "known": TEMPLATES.keys()}
	var agent = Peers.GetAgent(peerID)
	if agent == null:
		return {"ok": false, "code": "SESSION_CONFLICT", "detail": "peer not found"}
	var template : Dictionary = TEMPLATES[templateKey]
	_counter += 1
	var ep : Dictionary = {
		"id": "ep-%d-%d" % [_counter, Time.get_ticks_msec()],
		"peerID": peerID, "nick": nick, "templateKey": templateKey,
		"template": template,
		"score": {"kills": 0, "deaths": 0, "itemsLooted": 0, "xpGained": 0,
			"xpBaseline": agent.stat.experience if agent.stat else 0,
			"killsBaseline": int(bridge_score(peerID).get("kills", 0)),
			"deathsBaseline": int(bridge_score(peerID).get("deaths", 0)),
			"lootBaseline": int(bridge_score(peerID).get("itemsLooted", 0)),
			"levelBaseline": agent.stat.level if agent.stat else 1},
		"startedAtMsec": Time.get_ticks_msec(),
		"status": "active"
	}
	# 权威放置：alive 时 TriggerRespawn 对活体无效（pos 不变，tick14 挂账）——
	# 改走 NpcCommons.Warp（NPC 传送同源通路 → World.Warp → Spawn，人类玩家同 packet 路径），
	# 并设置 respawnDestination 供死后 respawn 回 spawn 区
	agent.respawnDestination = Destination.new(LauncherCommons.DefaultStartMapID, template["spawn"])
	NpcCommons.Warp(agent, LauncherCommons.DefaultStartMapID, template["spawn"])
	_episodes[peerID] = ep
	_push_arena_event(peerID, ep, "episode_start", {"episodeId": ep["id"], "template": templateKey,
		"brief": template["brief"], "killsRequired": template["killsRequired"]})
	return {"ok": true, "episodeId": ep["id"], "template": templateKey, "brief": template["brief"]}

func Status(peerID : int) -> Dictionary:
	var ep : Dictionary = _episodes.get(peerID, {})
	if ep.is_empty() or ep.get("status") != "active":
		return {"ok": true, "episode": null}
	_mirror_score(peerID, ep)
	return {"ok": true, "episode": _export(ep)}

func Stop(peerID : int) -> Dictionary:
	var ep : Dictionary = _episodes.get(peerID, {})
	if ep.is_empty() or ep.get("status") != "active":
		return {"ok": false, "code": "NO_ACTIVE_EPISODE"}
	return _finish(peerID, ep, "explicit_stop")

func Teardown(peerID : int) -> void:
	# close_session 防泄漏：活跃 episode 按 explicit_stop 收尾
	var ep : Dictionary = _episodes.get(peerID, {})
	if not ep.is_empty() and ep.get("status") == "active":
		_finish(peerID, ep, "explicit_stop")

func GetResult(episodeId : String) -> Dictionary:
	for peerID in _episodes.keys():
		var ep : Dictionary = _episodes[peerID]
		if str(ep.get("id")) == episodeId:
			return {"ok": true, "episode": _export(ep)}
	return {"ok": false, "code": "NOT_FOUND"}

# ---- 内部 ---------------------------------------------------------------

func _mirror_score(peerID : int, ep : Dictionary) -> void:
	# 计分镜像：以桥 _score 为权威源，episode 用 baseline 差值（episode 独立计分，不继承历史）
	var sc : Dictionary = bridge_score(peerID)
	if sc.is_empty():
		return
	ep["score"]["kills"] = max(0, int(sc.get("kills", 0)) - int(ep["score"].get("killsBaseline", 0)))
	ep["score"]["deaths"] = max(0, int(sc.get("deaths", 0)) - int(ep["score"].get("deathsBaseline", 0)))
	ep["score"]["itemsLooted"] = max(0, int(sc.get("itemsLooted", 0)) - int(ep["score"].get("lootBaseline", 0)))
	var agent = Peers.GetAgent(peerID)
	if agent and agent.stat:
		var totalXp : int = int(agent.stat.experience) + max(0, int(agent.stat.level) - int(ep["score"].get("levelBaseline", 1))) * 1000
		ep["score"]["xpGained"] = max(0, totalXp - int(ep["score"].get("xpBaseline", totalXp)))

func bridge_score(peerID : int) -> Dictionary:
	var bridge = get_node_or_null("/root/AgentBridge")
	if bridge == null:
		return {}
	return bridge._score.get(peerID, {})

func _finish(peerID : int, ep : Dictionary, reason : String) -> Dictionary:
	_mirror_score(peerID, ep)
	var status : String = "completed"
	if reason == "max_deaths":
		status = "failed"
	elif reason == "expired":
		status = "expired"
	ep["status"] = status
	ep["endReason"] = reason
	_push_arena_event(peerID, ep, "episode_end", {"episodeId": ep["id"], "template": ep["templateKey"],
		"status": status, "endReason": reason,
		"score": {"kills": ep["score"]["kills"], "deaths": ep["score"]["deaths"],
			"itemsLooted": ep["score"]["itemsLooted"], "xpGained": ep["score"]["xpGained"]}})
	# 排行榜落盘（tick18）：终局 episode 追加 JSONL
	var lb = get_node_or_null("/root/ArenaLeaderboard")
	if lb:
		ep["nick"] = ep.get("nick", "")
		lb.Record(ep)
	return {"ok": true, "status": status, "endReason": reason, "score": ep["score"].duplicate()}

func _push_arena_event(peerID : int, _ep : Dictionary, kind : String, payload : Dictionary) -> void:
	var bridge = get_node_or_null("/root/AgentBridge")
	if bridge:
		payload["kind"] = "arena"
		payload["arenaKind"] = kind
		bridge._push_event(peerID, payload)

func _export(ep : Dictionary) -> Dictionary:
	return {"id": ep["id"], "template": ep["templateKey"], "status": ep["status"],
		"score": ep["score"].duplicate(), "startedAtMsec": ep["startedAtMsec"]}
