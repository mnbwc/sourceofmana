extends SceneTree

# Capture raw SceneMultiplayer RPC packet bytes via a WebSocket loopback.
# Run with:
#   godot --headless --path . --script testdata/dump_rpc_packet.gd

const PORT := 27010

var server_peer: WebSocketMultiplayerPeer
var client_peer: WebSocketMultiplayerPeer
var mp: SceneMultiplayer
var rpc_node: Node
var frames := 0
var sent := false
var settle := 0

# RPC methods on this SceneTree script — the packet will target /root itself.
@rpc("any_peer", "call_remote", "reliable", 0)
func CreateAccount(accountName: String, password: String, email: String, rememberMe: bool, platform: int) -> void:
	pass

@rpc("any_peer", "call_remote", "reliable", 0)
func SetMovePos(pos: Vector2) -> void:
	pass

func _initialize() -> void:
	server_peer = WebSocketMultiplayerPeer.new()
	var err := server_peer.create_server(PORT)
	if err != OK:
		print("FAIL create_server: ", err)
		quit(1)
		return

	client_peer = WebSocketMultiplayerPeer.new()
	client_peer.create_client("ws://127.0.0.1:%d" % PORT)

	mp = MultiplayerAPI.create_default_interface()
	mp.set_root_path(root.get_path())
	mp.multiplayer_peer = server_peer

func _process(_delta: float) -> bool:
	frames += 1
	server_peer.poll()
	client_peer.poll()
	mp.poll()

	var status := client_peer.get_connection_status()
	if status != MultiplayerPeer.CONNECTION_CONNECTED:
		return false

	if not sent:
		settle += 1
		if settle < 20:
			return false
		# Target the SceneTree itself (path "/root"), which has the RPC methods.
		mp.rpc(2, root, "CreateAccount", ["alice", "secret", "a@b.c", true, 0])
		mp.rpc(2, root, "SetMovePos", [Vector2(1.5, -2.25)])
		sent = true
		return false

	if client_peer.get_available_packet_count() == 0:
		return false

	var count := 0
	while client_peer.get_available_packet_count() > 0:
		var packet: PackedByteArray = client_peer.get_packet()
		print("PACKET_%d=%s" % [count, packet.hex_encode()])
		count += 1
	print("TOTAL=", count)
	quit()
	return true
