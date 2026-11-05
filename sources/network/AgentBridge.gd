extends Node
## AgentBridge — AI 竞技场探针 (autoload)
## 方案 A：进程内只读探针 + WebSocket JSON 通道。
## 不新增 @rpc 到 Network.gd（S17：ProtocolVersion 哈希自 Network 节点 get_rpc_config()，
## 独立 autoload 的节点不在其中，本文件即使加 @rpc 也不影响版本）。
## 端口 9101（避开游戏 6108/6118），协议=每行一个 JSON（NDJSON）。

const BRIDGE_PORT : int = 9101
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

func _ready() -> void:
	if not "--server" in OS.get_cmdline_args():
		set_process(false)
		return
	# Launcher.Scene 是 call_deferred 挂载的，autoload _ready 时还没就绪——只判命令行即可
	var err : Error = tcp.listen(BRIDGE_PORT, "*")
	if err != OK:
		printerr("[AgentBridge] listen %d failed: %d" % [BRIDGE_PORT, err])
		set_process(false)
		return
	Util.PrintInfo("AgentBridge", "listening on %d" % BRIDGE_PORT)

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
		"latest_verdict":
			return _latest_verdict(req)
		"apply_action":
			return _apply_action(req)
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
		},
		"nearby": _nearby(agent),
	}
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
		var isMonster : bool = other is AIAgent and not (other is PlayerAgent)
		out.append({
			"rid": rid,
			"nick": other.nick,
			"type": "monster" if isMonster else ("player" if other is PlayerAgent else "npc"),
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
			_server_peer().SetClickPos(pos, peerID)
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
			var skillID : int = int(args.get("skillID", 0))
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

func _on_agent_damaged(value : int, peerID : int) -> void:
	_push_event(peerID, {"kind": "damaged", "value": value})

func _on_agent_killed(agent : BaseAgent, peerID : int) -> void:
	_push_event(peerID, {"kind": "died"})

func _on_enemy_killed(attacker : BaseAgent, monsterID : int, peerID : int) -> void:
	_push_event(peerID, {"kind": "kill", "monsterID": monsterID, "attackerNick": attacker.nick if attacker else ""})

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
	return {
		"action": str(req.get("name", "")),
		"basedOnFrame": NetworkCommons.FrameID(),
		"beforePos": {"x": agent.position.x, "y": agent.position.y},
		"beforeHealth": agent.stat.health if agent.stat else 0,
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

