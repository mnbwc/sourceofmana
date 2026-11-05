extends Node
## AgentBridge — AI 竞技场探针 (autoload)
## 方案 A：进程内只读探针 + WebSocket JSON 通道。
## 不新增 @rpc 到 Network.gd（S17：ProtocolVersion 哈希自 Network 节点 get_rpc_config()，
## 独立 autoload 的节点不在其中，本文件即使加 @rpc 也不影响版本）。
## 端口 9101（避开游戏 6108/6118），协议=每行一个 JSON（NDJSON）。

const BRIDGE_PORT : int = 9101  # 可被 cmdline --bridge-port 覆盖（同机多实例隔离）
var bridgePort : int = BRIDGE_PORT
const MAX_CLIENTS : int = 16

var tcp : TCPServer = TCPServer.new()
var clients : Dictionary = {}  # StreamPeerTCP -> recv buffer String
var snapshotCache : Dictionary = {}  # frame -> cached snapshot (多 agent 共享)
var cachedFrame : int = -1
var _events : Dictionary = {}      # peerID -> Array[event]（事件订阅缓冲）
var _pending : Dictionary = {}     # peerID -> {action, basedOnFrame, targetRID, beforePos, beforeHealth, seq}
var _eventSeq : int = 0
var _hooked : Dictionary = {}      # peerID -> true（信号钩子防重复）
var _pendingVerdicts : Array = []  # 每帧裁决结果（snapshot/drain_events 带回最近一条）
var _score : Dictionary = {}       # peerID -> {kills, deaths}（G2 v0.1：按 peerID 累计）

func _ready() -> void:
	if not "--server" in OS.get_cmdline_args():
		set_process(false)
		return
	# Launcher.Scene 是 call_deferred 挂载的，autoload _ready 时还没就绪——只判命令行即可
	var cargs : PackedStringArray = OS.get_cmdline_args()
	var bpIdx : int = cargs.find("--bridge-port")
	if bpIdx != -1 and bpIdx + 1 < cargs.size():
		bridgePort = int(cargs[bpIdx + 1])
	var err : Error = tcp.listen(bridgePort, "*")
	if err != OK:
		printerr("[AgentBridge] listen %d failed: %d" % [BRIDGE_PORT, err])
		set_process(false)
		return
	Util.PrintInfo("AgentBridge", "listening on %d" % bridgePort)

func _process(_delta : float) -> void:
	# 接受新连接
	while tcp.is_connection_available():
		var peer : StreamPeerTCP = tcp.take_connection()
		clients[peer] = ""
		Util.PrintInfo("AgentBridge", "client connected: %s" % peer.get_connected_host())
	# 处理既有连接
	for peer in clients.keys():
		_poll_client(peer)
	# 裁决轮询：未决动作等状态增量
	if not _pending.is_empty():
		for pid in _pending.keys():
			var pending : Dictionary = _pending[pid]
			if pending.get("action", "") in ["navigate", "attack"]:
				var v : Dictionary = _apply_verdict(pending.get("action", ""), pending)
				if v.get("verdict", "") != "PENDING":
					v["basedOnFrame"] = pending.get("basedOnFrame", NetworkCommons.FrameID())
					_pendingVerdicts.append({"peerID": pid, "verdict": v})
				else:
					v["basedOnFrame"] = pending.get("basedOnFrame", NetworkCommons.FrameID())
					_pendingVerdicts.append({"peerID": pid, "verdict": v, "provisional": true})

func _poll_client(peer : StreamPeerTCP) -> void:
	peer.poll()
	var status : int = peer.get_status()
	if status != StreamPeerTCP.STATUS_CONNECTED and status != StreamPeerTCP.STATUS_CONNECTING:
		clients.erase(peer)
		Util.PrintInfo("AgentBridge", "client disconnected")
		if clients.is_empty():
			_pending.clear()
			_pendingVerdicts.clear()
		return
	while peer.get_available_bytes() > 0:
		clients[peer] += peer.get_utf8_string(peer.get_available_bytes()).replace(String.chr(0), "")
	var buf : String = clients[peer]
	while buf.contains("\n"):
		var idx : int = buf.find("\n")
		var line : String = buf.substr(0, idx).strip_edges()
		buf = buf.substr(idx + 1)
		if not line.is_empty():
			var reply : String = _handle(line)
			peer.put_data((reply + "\n").to_utf8_buffer())
	clients[peer] = buf

func _handle(line : String) -> String:
	var req = JSON.parse_string(line)
	if req == null or not req.has("op"):
		return _json({"ok": false, "code": "INVALID_PARAMS", "detail": "missing op"})
	var op : String = req["op"]
	match op:
		"ping":
			return _json({"ok": true, "frame": NetworkCommons.FrameID(), "protocolVersion": NetworkCommons.ProtocolVersion})
		"list_players":
			return _json({"ok": true, "players": _collect_players()})
		"snapshot":
			if req.has("peerID"):
				return _json({"ok": true, "snapshot": _snapshot(int(req["peerID"]))})
			return _json({"ok": false, "code": "INVALID_PARAMS", "detail": "missing peerID"})
		"drain_events":
			return _drain_events(req)
		"debug_hooks":
			var dp : int = int(req.get("peerID", -1))
			var dagent = Peers.GetAgent(dp)
			var attackerList : Array = []
			var skillKeys : Array = []
			var coolingList : Array = []
			if dagent and dagent.progress:
				for k in dagent.progress.skills.keys():
					skillKeys.append(k)
			if dagent and "cooldownTimers" in dagent:
				for k in dagent.cooldownTimers:
					if dagent.cooldownTimers[k]:
						coolingList.append(k)
			if dagent and "attackers" in dagent:
				for e in dagent.attackers:
					attackerList.append(str(e.get("attacker").nick) + ":" + str(e.get("damage")))
			return _json({"ok": true, "hooked": _hooked.has(dp), "eventsBuf": (_events.get(dp, []).size()),
				"hasDamagedConn": dagent.agent_damaged.is_connected(_on_agent_damaged.bind(dp)) if dagent else null,
				"attackers": attackerList,
				"skills": skillKeys,
				"mana": ({"current": dagent.stat.mana, "max": dagent.stat.current.maxMana} if dagent and dagent.stat else {}),
				"cooling": coolingList})
		"latest_verdict":
			return _latest_verdict(req)
		"apply_action":
			return _apply_action(req)
		"close_session":
			return _close_session(req)
		_:
			return _json({"ok": false, "code": "INVALID_PARAMS", "detail": "unknown op %s" % op})

func _json(data) -> String:
	return JSON.stringify(data)

# ---- observe ----
func _collect_players() -> Array:
	var out : Array = []
	for peerID in Peers.peers.keys():
		var agent = Peers.GetAgent(peerID)
		if agent:
			out.append({"peerID": peerID, "nick": agent.nick, "isAgent": agent.nick.begins_with("ai_")})
	return out

func _snapshot(peerID : int) -> Dictionary:
	var frame : int = NetworkCommons.FrameID()
	var agent = Peers.GetAgent(peerID)
	if agent == null:
		return {"error": "PEER_NOT_FOUND"}
	# 按 FrameID 缓存共享快照（L2：多 agent 一帧一份基础快照）
	var snap : Dictionary = {
		"frame": frame,
		"player": {
			"peerID": peerID,
			"nick": agent.nick,
			"position": {"x": agent.position.x, "y": agent.position.y},
			"mapID": agent.mapID if "mapID" in agent else -1,
			"alive": ActorCommons.IsAlive(agent),
			"level": agent.stat.level if agent.stat else 0,
			"health": {"current": agent.stat.health, "max": agent.stat.current.maxHealth} if agent.stat and "current" in agent.stat else {"current": agent.stat.health, "max": 0} if agent.stat else {},
			"actionInProgress": SkillCommons.HasAnyActionInProgress(agent),
			"nav": {"hasAgent": agent.agent != null, "finished": (agent.agent.is_navigation_finished() if agent.agent else null),
				"target": ({"x": agent.agent.target_position.x, "y": agent.agent.target_position.y} if agent.agent else null),
				"next": ({"x": agent.agent.get_next_path_position().x, "y": agent.agent.get_next_path_position().y} if agent.agent else null),
				"physicsOn": agent.is_physics_processing() if agent is Node else null,
				"goal": agent.hasCurrentGoal, "input": ({"x": agent.currentInput.x, "y": agent.currentInput.y} if "currentInput" in agent else null)},
		},
		"nearby": _nearby(agent),
	}
	if not snap.has("error"):
		snap["score"] = _score.get(peerID, {"kills": 0, "deaths": 0})
	return snap

# 同地图 ±radius 范围实体（observe 范围），供 attack 选择目标
func _nearby(agent, radius : float = 480.0) -> Array:
	var out : Array = []
	for rid in WorldAgent.agents.keys():
		var other : BaseAgent = WorldAgent.agents[rid]
		if other == null or other == agent or not is_instance_valid(other):
			continue
		var mapA : WorldMap = WorldAgent.GetMapFromAgent(agent)
		var mapB : WorldMap = WorldAgent.GetMapFromAgent(other)
		if mapA == null or mapB == null or mapA != mapB:
			continue
		var dist : float = agent.position.distance_to(other.position)
		if dist > radius:
			continue
		var isPlayer : bool = other is PlayerAgent
		var isMonster : bool = other is AIAgent and not isPlayer and not (other is NpcAgent)
		out.append({
			"rid": rid,
			"nick": other.nick,
			"type": "player" if isPlayer else ("monster" if isMonster else "npc"),
			"position": {"x": other.position.x, "y": other.position.y},
			"distance": dist,
			"alive": ActorCommons.IsAlive(other),
			"health": {"current": other.stat.health, "max": other.stat.current.maxHealth} if other.stat and "current" in other.stat else {},
		})
	out.sort_custom(func(a, b): return a["distance"] < b["distance"])
	return out

# 同源派发：与 Network.CallServer 一样走 callv.call_deferred（进程内不绕过 Footprint/FairUse 节流）
func _server_peer() -> NetServer:
	var server : NetServer = Peers.GetAssociatedNetServer(_dispatchPeerID)
	return server if server else Network.ENetServer

var _dispatchPeerID : int = -1

# ---- applyAction（映射 Server.gd 既有方法，与人同源）----
func _apply_action(req : Dictionary) -> String:
	var peerID : int = int(req.get("peerID", -1))
	var action : String = req.get("name", "")
	var args : Dictionary = req.get("args", {})
	var agent = Peers.GetAgent(peerID)
	if agent == null:
		return _json({"ok": false, "code": "SESSION_CONFLICT", "detail": "peer not found"})
	_dispatchPeerID = peerID
	_hook_agent(peerID, agent)
	var frame : int = NetworkCommons.FrameID()
	match action:
		"navigate":
			# 同源：人类客户端点击 → SetClickPos
			var pos : Vector2 = Vector2(float(args.get("x", 0)), float(args.get("y", 0)))
			var before : Vector2 = agent.position
			var speer : NetServer = _server_peer()
			# ownScript 挂起时 SetClickPos 静默 return（S16）——桥派发前解除（与人点地走同效）
			if agent.ownScript:
				agent.ClearScript()
			speer.SetClickPos(pos, peerID)
			_pending[peerID] = _verdict_req(agent, req)
			_pending[peerID]["peerID"] = peerID
			_pending[peerID]["basedOnFrame"] = frame
			# 立即给一条快照裁决（多为 PENDING），终局裁决随 snapshot/drain_events 轮询返回
			return _json({"ok": true, "code": "ACTION_OK", "verdict": "PENDING", "basedOnFrame": frame,
				"before": {"x": before.x, "y": before.y}})
		"inspect":
			var rid : int = int(args.get("rid", 0))
			_server_peer().TriggerSelect(rid, peerID)
			return _json({"ok": true, "code": "ACTION_OK", "verdict": "APPLIED",
				"reason": "select_stats_requested", "basedOnFrame": frame})
		"attack":
			var rid : int = int(args.get("rid", 0))
			var skillName : String = str(args.get("skill", "Melee"))
			var skillID : int = int(args.get("skillID", 0))
			if skillID == 0:
				skillID = DB.GetCellHash(skillName)
			var skill : SkillCell = DB.SkillsDB.get(skillID) if DB.SkillsDB.has(skillID) else null
			if skill == null:
				return _json({"ok": false, "code": "INVALID_PARAMS", "detail": "unknown skill %s" % skillName, "basedOnFrame": frame})
			var target : BaseAgent = WorldAgent.GetAgent(rid)
			if target == null:
				return _json({"ok": false, "code": "TARGET_NOT_FOUND", "detail": "no agent for rid %d" % rid, "basedOnFrame": frame})
			var beforeHealth : int = target.stat.health if target.stat else 0
			var beforeDist : float = agent.position.distance_to(target.position)
			_server_peer().TriggerSelect(rid, peerID)
			_server_peer().TriggerSkill(rid, skillID, peerID)
			_pending[peerID] = _verdict_req(agent, req)
			_pending[peerID]["peerID"] = peerID
			_pending[peerID]["basedOnFrame"] = frame
			return _json({"ok": true, "code": "ACTION_OK", "verdict": "PENDING", "basedOnFrame": frame,
				"before": {"targetRID": rid, "targetHealth": beforeHealth, "distance": beforeDist}})
		"loot":
			var dropID : int = int(args.get("dropID", 0))
			var ok : bool = WorldDrop.PickupDrop(dropID, agent)
			return _json({"ok": true, "code": "ACTION_OK" if ok else "REJECTED_RULE",
				"verdict": "APPLIED" if ok else "REJECTED", "basedOnFrame": frame})
		"say":
			var text : String = str(args.get("text", "")).substr(0, 200)
			# 红线：只用 LOCAL 频道，GLOBAL 会外发 Discord
			_server_peer().TriggerChat(str(GUICommons.ChatChannel.LOCAL), text, peerID)
			return _json({"ok": true, "code": "ACTION_OK", "verdict": "APPLIED",
				"reason": "chat_sent_local", "basedOnFrame": frame})
		_:
			return _json({"ok": false, "code": "INVALID_PARAMS", "detail": "unknown action %s" % action})

func _latest_verdict(req : Dictionary) -> String:
	var peerID : int = int(req.get("peerID", -1))
	var found : Dictionary = {}
	var i : int = _pendingVerdicts.size() - 1
	while i >= 0:
		if int(_pendingVerdicts[i].get("peerID", -1)) == peerID:
			found = _pendingVerdicts[i]
			break
		i -= 1
	if found.is_empty():
		return _json({"ok": true, "verdict": null})
	if not found.get("provisional", false):
		_pendingVerdicts.remove_at(i)
	return _json({"ok": true, "verdict": found.get("verdict", {}), "provisional": found.get("provisional", false)})

# ---- M3 四元组阀门：事件订阅 ----
func _hook_agent(peerID : int, agent : BaseAgent) -> void:
	if _hooked.has(peerID) or agent == null:
		return
	_hooked[peerID] = true
	_events[peerID] = []
	if not agent.agent_damaged.is_connected(_on_agent_damaged.bind(peerID)):
		agent.agent_damaged.connect(_on_agent_damaged.bind(peerID))
	if not agent.agent_killed.is_connected(_on_agent_killed.bind(peerID)):
		agent.agent_killed.connect(_on_agent_killed.bind(peerID))
	if not agent.enemy_killed.is_connected(_on_enemy_killed.bind(peerID)):
		agent.enemy_killed.connect(_on_enemy_killed.bind(peerID))

func _push_event(peerID : int, event : Dictionary) -> void:
	_eventSeq += 1
	event["seq"] = _eventSeq
	event["frame"] = NetworkCommons.FrameID()
	if not _events.has(peerID):
		_events[peerID] = []
	var buf : Array = _events[peerID]
	buf.append(event)
	while buf.size() > 512:
		buf.pop_front()

func _on_agent_damaged(target, value : int, peerID : int) -> void:
	# Skill.gd: agent_damaged.emit(target, info.value)（信号声明 1 参但实发 2 参）
	_push_event(peerID, {"kind": "damaged", "value": value,
		"health": target.stat.health if target and target.stat else 0})

func _on_agent_killed(killed, peerID : int) -> void:
	# BaseAgent.gd: agent_killed.emit(self)
	var sc : Dictionary = _score.get_or_add(peerID, {"kills": 0, "deaths": 0})
	sc["deaths"] = int(sc.get("deaths", 0)) + 1
	_score[peerID] = sc
	_push_event(peerID, {"kind": "died", "nick": killed.nick if killed else ""})

func _on_enemy_killed(attacker, monsterID : int, peerID : int) -> void:
	# 信号声明 1 参但 Formula.gd 实发 3 参（attacker, monsterID, bind的peerID）
	# → 第 1 位=attacker，第 2 位=怪物 data._id（DB hash）
	var sc : Dictionary = _score.get_or_add(peerID, {"kills": 0, "deaths": 0})
	sc["kills"] = int(sc.get("kills", 0)) + 1
	_score[peerID] = sc
	var monsterNick : String = ""
	var cell = DB.EntitiesDB.get(monsterID, null)
	if cell:
		monsterNick = cell._name
	_push_event(peerID, {"kind": "kill", "monsterID": monsterID, "monsterNick": monsterNick,
		"attackerNick": attacker.nick if attacker else ""})

# ---- G1 session 管理：close_session 断开游戏 peer 并清理桥侧状态 ----
func _close_session(req : Dictionary) -> String:
	var peerID : int = int(req.get("peerID", -1))
	if peerID < 0:
		return _json({"ok": false, "code": "INVALID_PARAMS", "detail": "missing peerID"})
	var agent = Peers.GetAgent(peerID)
	if agent == null:
		return _json({"ok": false, "code": "SESSION_CONFLICT", "detail": "peer not found"})
	# 与人类掉线同源：走 Server.FullyDisconnect（含 DisconnectAccount + RemovePeer）
	_server_peer_for(peerID).FullyDisconnect.call_deferred(peerID)
	_events.erase(peerID)
	_pending.erase(peerID)
	_hooked.erase(peerID)
	_pendingVerdicts = _pendingVerdicts.filter(func(v): return int(v.get("peerID", -1)) != peerID)
	return _json({"ok": true, "closed": peerID})

# 指定 peer 的所属 NetServer（dispatch 用，避免依赖 _dispatchPeerID 全局）
func _server_peer_for(peerID : int) -> NetServer:
	var server : NetServer = Peers.GetAssociatedNetServer(peerID)
	return server if server else Network.ENetServer

func _drain_events(req : Dictionary) -> String:
	var peerID : int = int(req.get("peerID", -1))
	var agent = Peers.GetAgent(peerID)
	if agent == null:
		return _json({"ok": false, "code": "SESSION_CONFLICT", "detail": "peer not found"})
	_hook_agent(peerID, agent)
	var since : int = int(req.get("since", 0))
	var out : Array = []
	for e in _events.get(peerID, []):
		if int(e.get("seq", 0)) > since:
			out.append(e)
	return _json({"ok": true, "events": out, "cursor": _eventSeq})

# ---- M3 四元组阀门：裁决回执（替换 PENDING_PROBE）----
func _verdict_req(agent, req : Dictionary) -> Dictionary:
	# beforeHealth 必须取目标（attack 裁决对象）的血，agent.stat.health 是攻击者自己的
	var targetRID : int = int(req.get("args", {}).get("rid", 0))
	var targetHealth : int = 0
	var target : BaseAgent = WorldAgent.GetAgent(targetRID)
	if target and target.stat:
		targetHealth = target.stat.health
	return {
		"action": str(req.get("name", "")),
		"basedOnFrame": NetworkCommons.FrameID(),
		"beforePos": {"x": agent.position.x, "y": agent.position.y},
		"beforeHealth": targetHealth,
		"targetRID": int(req.get("args", {}).get("rid", 0)),
		"startedAtMsec": Time.get_ticks_msec(),
	}

func _apply_verdict(kind : String, pending : Dictionary) -> Dictionary:
	var peerID : int = int(pending.get("peerID", -1))
	var agent = Peers.GetAgent(peerID)
	if agent == null:
		return {"verdict": "UNVERIFIABLE", "reason": "peer gone"}
	var nowMsec : int = Time.get_ticks_msec()
	var pos : Dictionary = {"x": agent.position.x, "y": agent.position.y}
	var beforePos : Dictionary = pending.get("beforePos", {})
	var moved : bool = Vector2(pos.x - beforePos.get("x", pos.x), pos.y - beforePos.get("y", pos.y)).length_squared() > 1.0
	var elapsed : int = nowMsec - int(pending.get("startedAtMsec", nowMsec))
	match pending.get("action", ""):
		"navigate":
			# 状态增量裁决：等最多 ~1.2s（30fps 服务端 30+ 帧）看位置变化
			if moved:
				_pending.erase(peerID)
				return {"verdict": "APPLIED", "reason": "position_changed", "before": beforePos, "after": pos, "elapsedMs": elapsed}
			if elapsed < 1200:
				return {"verdict": "PENDING"}
			_pending.erase(peerID)
			return {"verdict": "REJECTED" if SkillCommons.HasAnyActionInProgress(agent) else "NO_EFFECT", "reason": "no_position_change", "after": pos, "elapsedMs": elapsed}
		"attack":
			var target : BaseAgent = WorldAgent.GetAgent(int(pending.get("targetRID", 0)))
			var beforeHealth : int = int(pending.get("beforeHealth", 0))
			var targetAlive : bool = target != null and ActorCommons.IsAlive(target)
			# 裁决：目标死亡 / 目标掉血 / 攻击者受伤（反击）均视为生效
			if target != null:
				var nowHealth : int = target.stat.health if target.stat else 0
				if not targetAlive:
					_pending.erase(peerID)
					return {"verdict": "APPLIED", "reason": "target_died", "elapsedMs": elapsed}
				if nowHealth < beforeHealth:
					_pending.erase(peerID)
					return {"verdict": "APPLIED", "reason": "target_health_decreased", "delta": beforeHealth - nowHealth, "elapsedMs": elapsed}
			for e in (agent.attackers if "attackers" in agent else []):
				pass  # attackers 非空说明目标已反击（agent 被打），留作 M5 归因
			if elapsed >= 1200:
				_pending.erase(peerID)
				var cast : bool = SkillCommons.HasAnyActionInProgress(agent)
				return {"verdict": "APPLIED" if cast else "NO_EFFECT", "reason": "casting_in_progress" if cast else "no_target_change", "elapsedMs": elapsed}
			return {"verdict": "PENDING"}
		"inspect", "say", "loot":
			_pending.erase(peerID)
			return {"verdict": "APPLIED", "reason": "fire_and_forget", "after": pos}
		_:
			_pending.erase(peerID)
			return {"verdict": "UNVERIFIABLE", "reason": "unknown action"}

