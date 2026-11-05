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
static func ResolvePort(useWebSocket : bool) -> int:
	if useWebSocket:
		return PortOverrideWS if PortOverrideWS > 0 else NetworkCommons.WebSocketPortTesting
	return PortOverrideENet if PortOverrideENet > 0 else NetworkCommons.ENetPortTesting
static var isMobile : bool				= OS.has_feature("android") or OS.has_feature("ios") or Util.IsMobile()
static var isWeb : bool					= OS.has_feature("web")
