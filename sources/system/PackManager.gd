extends Node

# Runtime PCK pack manager.
# Downloads additional PCK packs from the same origin and mounts them
# via ProjectSettings.load_resource_pack(). Packs are stored in user://packs/
# (IndexedDB on Web, persistent on desktop).
#
# Usage:
#   PackManager.ensure("audio", "packs/audio.pck")
#   PackManager.ensure_maps()  # downloads maps.pck on demand

const PACK_DIR := "user://packs/"
const MAPS_PACK_NAME := "maps"
const MAPS_PACK_PATH := "packs/maps.pck"

signal packLoaded(packName: String, path: String)
signal packError(packName: String, message: String)

var _mounted: Dictionary = {}  # packName -> path
var _loading: Dictionary = {}  # packName -> HTTPRequest


func _ready() -> void:
	# Ensure pack directory exists
	if not DirAccess.dir_exists_absolute(PACK_DIR):
		DirAccess.make_dir_recursive_absolute(PACK_DIR)


func is_mounted(pack_name: String) -> bool:
	return _mounted.has(pack_name)


func is_loading(pack_name: String) -> bool:
	return _loading.has(pack_name)


func ensure(pack_name: String, remote_path: String) -> void:
	if _mounted.has(pack_name):
		return
	if _loading.has(pack_name):
		return

	var local_path := PACK_DIR + pack_name + ".pck"

	# If already downloaded, just mount it
	if FileAccess.file_exists(local_path):
		_mount(pack_name, local_path)
		return

	# Download and mount
	_download(pack_name, remote_path, local_path)


# Download and mount the maps pack (contains all non-core map scenes).
func ensure_maps() -> void:
	ensure(MAPS_PACK_NAME, MAPS_PACK_PATH)


func _mount(pack_name: String, local_path: String) -> void:
	var ok: bool = ProjectSettings.load_resource_pack(local_path)
	if not ok:
		packError.emit(pack_name, "Failed to mount pack at %s" % local_path)
		return
	_mounted[pack_name] = local_path
	packLoaded.emit(pack_name, local_path)
	# If the audio pack just loaded, populate the music DB now that resources exist
	if pack_name == "audio" and DB.isInitialized:
		DB.ReloadAudio()


func _download(pack_name: String, remote_path: String, local_path: String) -> void:
	var http := HTTPRequest.new()
	add_child(http)
	_loading[pack_name] = http

	http.download_file = local_path
	http.use_threads = false  # Web doesn't support threads

	# Godot Web HTTPRequest requires a full URL; convert relative paths
	var url := remote_path
	if LauncherCommons.isWeb and not remote_path.begins_with("http"):
		# Use the directory of the current page as the base URL.
		# e.g. "https://host:port/game/index.html" -> "https://host:port/game/"
		var href: String = JavaScriptBridge.eval("window.location.href")
		var base: String = href.get_base_dir()  # strips the filename
		if not base.ends_with("/"):
			base += "/"
		url = base + remote_path

	var err := http.request(url)
	if err != OK:
		_loading.erase(pack_name)
		packError.emit(pack_name, "HTTPRequest failed to start: %d" % err)
		return

	http.request_completed.connect(_on_download_completed.bind(pack_name, local_path, http), CONNECT_ONE_SHOT)


func _on_download_completed(result: int, response_code: int, headers: PackedStringArray, body: PackedByteArray, pack_name: String, local_path: String, http: HTTPRequest) -> void:
	_loading.erase(pack_name)
	http.queue_free()

	if result != HTTPRequest.RESULT_SUCCESS or response_code != 200:
		packError.emit(pack_name, "Download failed: result=%d code=%d" % [result, response_code])
		return

	_mount(pack_name, local_path)
