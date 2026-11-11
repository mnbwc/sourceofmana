extends RefCounted
class_name LauncherCommons

# Project
const ProjectName : String				= "Source of Mana"
const DiscordInviteLink : String		= "https://discord.com/invite/qR6ursz7xD"
const IRCLink : String					= "https://web.libera.chat/?channels=#sourceofmana"

# Map
static var DefaultStartMapID : int		= "Tulimshar".hash()
const DefaultStartPos : Vector2i		= Vector2i(2176, 2560) # Tile (68, 80)
const DefaultStartOffset : Vector2i		= Vector2i(64, 32)

static func GetRandomStartPos() -> Vector2i:
	return DefaultStartPos + Vector2i(randi_range(-DefaultStartOffset.x, DefaultStartOffset.x), randi_range(-DefaultStartOffset.y, DefaultStartOffset.y))

# MapPool
const EnableMapPool : bool				= false
const MapPoolMaxSize : int				= 10

const ServerMaxFPS : int				= 30

# Common accessors
static var IsTesting : bool				= not OS.has_feature("production")

# 端口覆盖：--port N（ENet）/ --ws-port N（WebSocket），供同机多实例隔离（AgentBridge --bridge-port 同理）
static var PortOverrideENet : int = -1
static var PortOverrideWS : int = -1
# 服务器地址覆盖：--server-address HOST，供 web 客户端指向 Go 服务器（wss://）
static var ServerAddressOverride : String = ""
static func ParsePortOverrides() -> void:
	# 兼容两种传参位置：--port 在 Godot 参数区（get_cmdline_args）或在 -- 后用户区
	# （get_cmdline_user_args，探针 `-- --agent-client X --port N` 场景——此前只读前者，
	# 客户端静默连 6119 旧实例，tick7 坑的真正根因）
	var cargs : PackedStringArray = OS.get_cmdline_args()
	for ua in OS.get_cmdline_user_args():
		cargs.append(ua)
	var i : int = cargs.find("--port")
	if i != -1 and i + 1 < cargs.size():
		PortOverrideENet = int(cargs[i + 1])
	i = cargs.find("--ws-port")
	if i != -1 and i + 1 < cargs.size():
		PortOverrideWS = int(cargs[i + 1])
	i = cargs.find("--server-address")
	if i != -1 and i + 1 < cargs.size():
		ServerAddressOverride = cargs[i + 1]
	# Web 导出：从 URL 参数读取 ?server=HOST 和 ?port=N，方便部署时
	# 不改代码即可指向不同服务器（如 https://som.org/?server=go.som.org&port=443）
	if isWeb:
		_ParseWebURLParams()
static func _ParseWebURLParams() -> void:
	var params : Dictionary = _GetURLParams()
	if params.is_empty():
		return
	if params.has("server") and not str(params["server"]).is_empty():
		ServerAddressOverride = str(params["server"])
	if params.has("port") and str(params["port"]).is_valid_int():
		var p : int = int(params["port"])
		if p > 0:
			PortOverrideWS = p
			PortOverrideENet = p
static func _GetURLParams() -> Dictionary:
	var result : Dictionary = {}
	if OS.has_feature("web"):
		# JavaScriptBridge is provided by the stock Godot web export template.
		if Engine.has_singleton("JavaScriptBridge"):
			var js = Engine.get_singleton("JavaScriptBridge")
			var search : String = js.eval("window.location.search")
			for pair in search.trim_prefix("?").split("&"):
				var kv : Array = pair.split("=", 1)
				if kv.size() == 2:
					result[kv[0]] = kv[1]
	return result
static func ResolvePort(useWebSocket : bool) -> int:
	if useWebSocket:
		return PortOverrideWS if PortOverrideWS > 0 else NetworkCommons.WebSocketPortTesting
	return PortOverrideENet if PortOverrideENet > 0 else NetworkCommons.ENetPortTesting
static func ResolveServerAddress(isLocal : bool) -> String:
	if isLocal:
		return NetworkCommons.LocalServerAddress
	if not ServerAddressOverride.is_empty():
		return ServerAddressOverride
	return NetworkCommons.ServerAddressTesting if IsTesting else NetworkCommons.ServerAddress
static var isMobile : bool				= OS.has_feature("android") or OS.has_feature("ios") or Util.IsMobile()
static var isWeb : bool					= OS.has_feature("web")
static var isFullPCK : bool				= false  # false = split PCKs (core + audio + maps); true = single full PCK
