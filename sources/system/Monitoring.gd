extends Node
class_name Monitoring

#
static func IsEnabled() -> bool:
	return Conf.GetUserValue("Privacy-BugReports", false)

# Sentry 类型注解被刻意省略：Web 导出不带 sentry 扩展时这些全局类不存在，
# 带类型注解会在编译期解析失败。运行期用 class_exists() 守卫。
static func Configure(options):
	if options == null:
		return
	if ClassDB.class_exists("SentryOptions"):
		options.godot_logger.event_mask = ClassDB.class_get_integer_constant("SentryOptions", "MASK_ERROR") \
			| ClassDB.class_get_integer_constant("SentryOptions", "MASK_WARNING") \
			| ClassDB.class_get_integer_constant("SentryOptions", "MASK_SCRIPT") \
			| ClassDB.class_get_integer_constant("SentryOptions", "MASK_SHADER")
	options.debug = false
	options.attach_log = false
	options.before_send = BeforeSend

static func BeforeSend(event):
	return event if IsEnabled() else null

static func SetPlayer(playerName : String):
	if not _sentry_ready():
		return
	if not playerName.is_empty():
		var user = ClassDB.instantiate("SentryUser")
		user.username = playerName
		_sentry_sdk().set_user(user)
		_sentry_sdk().set_tag("player", playerName)

static func SetTransport(transport : Peers.TransportType):
	if not _sentry_ready():
		return
	_sentry_sdk().set_tag("transport", Peers.TransportType.keys()[transport].to_lower())

#
static func Init():
	if not OS.has_feature("sentry"):
		return
	if not IsEnabled():
		return
	if not ClassDB.class_exists("SentrySDK"):
		return

	_sentry_sdk().init(Configure)
	if _sentry_sdk().is_enabled():
		_sentry_sdk().set_tag("platform", OS.get_name())
		_sentry_sdk().set_tag("version", str(ProjectSettings.get_setting("application/config/version", "")))
		_sentry_sdk().set_tag("role", "server" if "--server" in OS.get_cmdline_args() else "client")
		_sentry_sdk().set_tag("headless", "true" if DisplayServer.get_name() == "headless" else "false")

# --- helpers -------------------------------------------------------------
# The sentry GDExtension may be absent in Web exports. Resolve its entry point
# lazily so a missing extension degrades to a no-op instead of a script error.
static func _sentry_sdk():
	return Engine.get_singleton("SentrySDK")

static func _sentry_ready() -> bool:
	if not ClassDB.class_exists("SentrySDK"):
		return false
	var sdk = Engine.get_singleton("SentrySDK")
	return sdk != null and sdk.is_enabled()
