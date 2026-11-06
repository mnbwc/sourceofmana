extends Node
## M3 阀门探针：无头最小客户端（第二实例）
## 用法: godot471 --headless --path . -- --agent-client <name>
## 不进 GUI/FSM，直接 multiplayer 连接 → Network RPC 登录 → ConnectCharacter → 打印收到的事件

var accountName : String = ""
var stage : String = "init"
var results : Array = []
var _elapsed : float = 0.0
var _lastFsm : int = -1
const STAGE_TIMEOUT_SEC : float = 120.0
## 掉线检测：IN_GAME 之后回到 LOGIN_SCREEN（DisconnectServer 路径）视为 server 掉线，记 DISCONNECTED 退出，
## 消除「server 死亡后探针卡在 CHAR_SCREEN 挂死」的问题（tick5 定案）。

func _enter_tree() -> void:
	# _init/_enter_tree 早于 Launcher._ready：在自起 Server 判定前改写全局
	if "--agent-client" in OS.get_cmdline_user_args():
		NetworkCommons.IsLocal = true

func _ready() -> void:
	var args : PackedStringArray = OS.get_cmdline_user_args()
	var idx : int = args.find("--agent-client")
	if idx < 0 or idx + 1 >= args.size():
		set_process(false)
		return
	accountName = "ai_" + args[idx + 1]
	Util.PrintLog("AgentClient", "starting for %s" % accountName)
	# 连接本地已起的服务器（WebSocket Testing 端口），并禁止本实例自起 Server
	NetworkCommons.IsLocal = true
	Launcher.Client()  # 初始化 Client 服务（同人类路径）
	var timer : Timer = Timer.new()
	timer.wait_time = 1.0
	timer.timeout.connect(_tick)
	add_child(timer)
	timer.start()

func _tick() -> void:
	_elapsed += 1.0
	# --- 掉线检测：进过 IN_GAME 后 FSM 回 LOGIN_SCREEN = server 掉线（DisconnectServer）---
	if _lastFsm == FSM.States.IN_GAME and FSM.currentState == FSM.States.LOGIN_SCREEN:
		results.append({"event": "DISCONNECTED", "frame": NetworkCommons.FrameID()})
		_print_results()
		get_tree().quit(3)
		return
	_lastFsm = FSM.currentState
	# --- 各阶段超时保护：任何阶段卡死 120s 强制退出（防 CHAR_SCREEN 挂死）---
	if stage != "done" and _elapsed > STAGE_TIMEOUT_SEC:
		results.append({"event": "TIMEOUT", "stage": stage, "fsm": FSM.currentState})
		_print_results()
		get_tree().quit(4)
		return
	match stage:
		"init":
			if Network.clientConnected:
				Util.PrintLog("AgentClient", "connected, registering")
				stage = "register"
		"register":
			var ok : bool = Network.CreateAccount(accountName, "TestPass123!", accountName + "@local.test", false)
			Util.PrintLog("AgentClient", "CreateAccount -> %s" % ok)
			stage = "login"
		"login":
			var ok : bool = Network.LoginWithPassword(accountName, "TestPass123!", false)
			Util.PrintLog("AgentClient", "Login -> %s" % ok)
			stage = "create_char"
		"create_char":
			# 角色/账号已存在（testing.db 复用）时 Create 静默失败——发送后无论结果都尝试连接
			if true:
				# traits 结构按 GUI Traits.GetValues() 同源：hairstyle/haircolor/race/skintone/gender
				var hairstyles : PackedInt64Array = DB.HairstylesDB.keys()
				var haircolors : PackedInt64Array = DB.PalettesDB[DB.Palette.HAIR].keys()
				var races : PackedInt64Array = DB.RacesDB.keys()
				var race : RaceData = DB.GetRace(races[0])
				var skinKeys : Array = race.skins.keys() if race else []
				var traits : Dictionary = {
					"hairstyle": hairstyles[0],
					"haircolor": haircolors[0],
					"race": races[0],
					"skintone": skinKeys[0].hash() if skinKeys.size() > 0 else 0,
					"gender": 0,
				}
				# 属性：力量拉满（上限 1级=4+1=5 点：攻 10+5*3=25 vs Maggot def1 → 单发 ~24，
				# 50hp 三发内；总和 ≤ GetMaxAttributePoints(1)）
				var attrs : Dictionary = {
					"strength": 5, "vitality": 0, "agility": 0,
					"endurance": 0, "concentration": 0,
				}
				Network.CreateCharacter(accountName, traits, attrs)
				Util.PrintLog("AgentClient", "CreateCharacter sent")
				stage = "connect_char"
		"connect_char":
			Network.ConnectCharacter(accountName)
			Util.PrintLog("AgentClient", "ConnectCharacter sent")
			stage = "in_game_wait"
		"in_game_wait":
			if FSM.currentState == FSM.States.IN_GAME:
				# 复用账号兜底：旧号属性可能是 0 力（Create 静默失败时）——进入游戏后
				# 走服务端 SetAttributes（只升不降，预算 ≤ GetMaxAttributePoints(level)）
				var stat : ActorStats = Launcher.Player.stat if Launcher.Player else null
				if stat and stat.strength < 5:
					Network.SetAttributes(5, 0, 0, 0, 0)
					Util.PrintLog("AgentClient", "SetAttributes strength=5 sent (was %d)" % stat.strength)
				results.append({"event": "ENTERED_GAME", "frame": NetworkCommons.FrameID()})
				_print_results()
				stage = "done"
		"done":
			set_process(false)
			# --agent-keep: 保持在线供 bridge E2E 使用；默认行为=退出
			if OS.get_cmdline_user_args().find("--agent-keep") >= 0:
				Util.PrintLog("AgentClient", "keeping alive for bridge E2E")
			else:
				get_tree().quit(0)

func _my_peer_id() -> int:
	return multiplayer.get_unique_id()

func OnCharacterError(err : int) -> void:
	if stage != "done":
		results.append({"event": "CHARACTER_ERROR", "err": err, "fsm": FSM.currentState})
		Util.PrintLog("AgentClient", "CharacterError -> %d" % err)

func _print_results() -> void:
	for r in results:
		print("AGENT-RESULT: ", JSON.stringify(r))
	print("AGENT-DONE")
