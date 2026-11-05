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

func _poll_client(peer : StreamPeerTCP) -> void:
	peer.poll()
	var status : int = peer.get_status()
	if status != StreamPeerTCP.STATUS_CONNECTED and status != StreamPeerTCP.STATUS_CONNECTING:
		clients.erase(peer)
		Util.PrintInfo("AgentBridge", "client disconnected")
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
			"actionInProgress": agent.IsActionInProgress() if agent.has_method("IsActionInProgress") else null,
		},
	}
	return snap

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
	var frame : int = NetworkCommons.FrameID()
	match action:
		"navigate":
			# 同源：人类客户端点击 → SetClickPos
			var pos : Vector2 = Vector2(float(args.get("x", 0)), float(args.get("y", 0)))
			var before : Vector2 = agent.position
			_server_peer().SetClickPos(pos, peerID)
			# 裁决回执：等待一帧看状态变化（这里先回 ACCEPTED+探针承诺，M5 补状态增量裁决）
			return _json({"ok": true, "code": "ACTION_OK", "verdict": "PENDING_PROBE",
				"basedOnFrame": frame, "before": {"x": before.x, "y": before.y}})
		"inspect":
			var rid : int = int(args.get("rid", 0))
			_server_peer().TriggerSelect(rid, peerID)
			return _json({"ok": true, "code": "ACTION_OK", "verdict": "PENDING_PROBE", "basedOnFrame": frame})
		"attack":
			var rid : int = int(args.get("rid", 0))
			var skillID : int = int(args.get("skillID", 0))
			_server_peer().TriggerSelect(rid, peerID)
			_server_peer().TriggerSkill(rid, skillID, peerID)
			return _json({"ok": true, "code": "ACTION_OK", "verdict": "PENDING_PROBE", "basedOnFrame": frame})
		"loot":
			var dropID : int = int(args.get("dropID", 0))
			var ok : bool = WorldDrop.PickupDrop(dropID, agent)
			return _json({"ok": true, "code": "ACTION_OK" if ok else "REJECTED_RULE",
				"verdict": "APPLIED" if ok else "REJECTED", "basedOnFrame": frame})
		"say":
			var text : String = str(args.get("text", "")).substr(0, 200)
			# 红线：只用 LOCAL 频道，GLOBAL 会外发 Discord
			_server_peer().TriggerChat(str(GUICommons.ChatChannel.LOCAL), text, peerID)
			return _json({"ok": true, "code": "ACTION_OK", "verdict": "PENDING_PROBE", "basedOnFrame": frame})
		_:
			return _json({"ok": false, "code": "INVALID_PARAMS", "detail": "unknown action %s" % action})
