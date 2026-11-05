extends Node
## M3 阀门探针：无头最小客户端（第二实例）
## 用法: godot471 --headless --path . -- --agent-client <name>
## 不进 GUI/FSM，直接 multiplayer 连接 → Network RPC 登录 → ConnectCharacter → 打印收到的事件

var accountName : String = ""
var stage : String = "init"
var results : Array = []

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
				# 属性：全默认 0 也合法（CheckAttributes 只查总和上限）
				var attrs : Dictionary = {
					"strength": 0, "vitality": 0, "agility": 0,
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
				results.append({"event": "ENTERED_GAME", "frame": NetworkCommons.FrameID()})
				_print_results()
				stage = "done"
		"done":
			set_process(false)

func _my_peer_id() -> int:
	return multiplayer.get_unique_id()

func _print_results() -> void:
	for r in results:
		print("AGENT-RESULT: ", JSON.stringify(r))
	print("AGENT-DONE")
