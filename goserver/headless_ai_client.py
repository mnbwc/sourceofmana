#!/usr/bin/env python3
"""SoM Headless AI Client — lightweight Python client for AI agents.

Speaks the Godot SceneMultiplayer WebSocket protocol directly (no Godot
runtime, ~15 MB RSS vs ~400 MB for a headless Godot instance).

Features:
  - Full login chain: CreateAccount / LoginWithPassword / CreateCharacter /
    CharacterListing / ConnectCharacter
  - In-game loop: receives world pushes (PreloadEntity, UpdateEntity, ...)
  - AgentBridge NDJSON server: external AI can connect via TCP and control
    the character (list_players, snapshot, apply_action, drain_events, ...)
  - Built-in auto-play: auto-combat, auto-loot (optional)

Usage:
    python3 headless_ai_client.py [ws://host:port/ws] [options]

    # Quick start (auto-create account + character)
    python3 headless_ai_client.py ws://127.0.0.1:27012/ws --auto-create --nick [AI]Bot

    # Login with existing account
    python3 headless_ai_client.py ws://127.0.0.1:27012/ws --account myname --password mypass

    # With auto-play
    python3 headless_ai_client.py ws://127.0.0.1:27012/ws --auto-create --auto-play

    # Custom bridge port (for external AI control)
    python3 headless_ai_client.py ws://127.0.0.1:27012/ws --auto-create --bridge-port 19101

    # Keep running after login (don't exit)
    python3 headless_ai_client.py ws://127.0.0.1:27012/ws --auto-create --keep

Environment variables:
    SOM_WS_URL      WebSocket URL (default: ws://127.0.0.1:27012/ws)
    SOM_ACCOUNT     Account name
    SOM_PASSWORD    Account password
    SOM_NICK        Character nickname
    SOM_BRIDGE_PORT AgentBridge listen port (default: 19101)
    SOM_AUTO_PLAY   Enable auto-play (0/1, default: 0)
    SOM_KEEP        Keep running after login (0/1, default: 1)
"""

import asyncio
import json
import logging
import os
import signal
import socket
import struct
import sys
import threading
import time

try:
    import websockets
except ImportError:
    print("ERROR: 'websockets' package required. Install: pip install websockets", file=sys.stderr)
    sys.exit(1)

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%H:%M:%S",
)
log = logging.getLogger("ai-client")

# ---------------------------------------------------------------------------
# Godot variant types
# ---------------------------------------------------------------------------

NIL = 0
BOOL = 1
INT = 2
FLOAT = 3
STRING = 4
VECTOR2 = 5
DICTIONARY = 27
ARRAY = 28

# SceneMultiplayer command types
CMD_REMOTE_CALL = 0
CMD_SIMPLIFY_PATH = 1
CMD_CONFIRM_PATH = 2

# RPC name IDs (sorted @rpc list of Network.gd)
RPC_AUTH_ERROR = 1
RPC_AUTH_TOKEN_RESULT = 2
RPC_CHARACTER_ERROR = 8
RPC_CHARACTER_INFO = 9
RPC_CHARACTER_LISTING = 10
RPC_CONNECT_CHARACTER = 19
RPC_CREATE_ACCOUNT = 25
RPC_CREATE_CHARACTER = 26
RPC_LOGIN_WITH_PASSWORD = 45

# Game action RPCs
RPC_SET_CLICK_POS = 75       # SetClickPos(Vector2)
RPC_TRIGGER_SKILL = 107      # TriggerSkill(int skillID)
RPC_TRIGGER_CHAT = 90        # TriggerChat(String message)
RPC_PICKUP_DROP = 40         # PickupDrop(int dropRID)
RPC_TRIGGER_RESPAWN = 104    # TriggerRespawn()
RPC_TRIGGER_INTERACT = 96    # TriggerInteract(int targetRID)
RPC_TRIGGER_EMOTE = 94       # TriggerEmote(int emoteID)

# Server push RPCs to track (for world state)
PUSH_IDS = {
    0, 2, 6, 11, 12, 13, 16, 20, 21, 22, 23, 24, 32, 34, 35, 39, 41, 43, 44,
    51, 52, 54, 55, 56, 57, 58, 59, 60, 61, 62, 79, 80, 82, 83, 84, 85, 86,
    87, 88, 89, 90, 91, 92, 93, 94, 95, 96, 97, 104, 105, 107, 109, 110, 112,
    113, 114, 115, 116, 118,
}

# Error codes
AUTH_ERR_OK = 0
AUTH_ERR_NAME_VALID = 9
AUTH_ERR_NAME_SIZE = 10
CHAR_ERR_OK = 0
CHAR_ERR_NAME_VALID = 8
CHAR_ERR_NAME_SIZE = 9

CACHE_ID = 5
NODE_PATH = "/root/Network"

# ---------------------------------------------------------------------------
# Variant encoding (encode_and_compress_variants)
# ---------------------------------------------------------------------------

def enc_int(v: int) -> bytes:
    if -(1 << 7) <= v < (1 << 7):
        return bytes([INT, v & 0xFF])
    if -(1 << 15) <= v < (1 << 15):
        return bytes([INT | (1 << 6)]) + struct.pack("<h", v)
    if -(1 << 31) <= v < (1 << 31):
        return bytes([INT | (2 << 6)]) + struct.pack("<i", v)
    return bytes([INT | (3 << 6)]) + struct.pack("<q", v)

def enc_bool(v: bool) -> bytes:
    return bytes([BOOL | (0x80 if v else 0)])

def enc_string(s: str) -> bytes:
    b = s.encode("utf-8")
    out = struct.pack("<I", len(b)) + b
    pad = (-len(out)) % 4
    return out + b"\x00" * pad

def enc_uncompressed(vtype: int, payload: bytes) -> bytes:
    return struct.pack("<I", vtype) + payload

def enc_variant(v) -> bytes:
    if isinstance(v, bool):
        return enc_bool(v)
    if isinstance(v, int):
        return enc_int(v)
    if isinstance(v, str):
        return enc_uncompressed(STRING, enc_string(v))
    if isinstance(v, dict):
        payload = struct.pack("<I", len(v))
        for k, val in v.items():
            payload += enc_variant_uncompressed(k) + enc_variant_uncompressed(val)
        return enc_uncompressed(DICTIONARY, payload)
    if isinstance(v, list):
        payload = struct.pack("<I", len(v))
        for item in v:
            payload += enc_variant_uncompressed(item)
        return enc_uncompressed(ARRAY, payload)
    raise TypeError(f"unsupported variant type: {type(v)}")

def enc_variant_uncompressed(v) -> bytes:
    if isinstance(v, bool):
        return enc_uncompressed(BOOL, struct.pack("<I", 1 if v else 0))
    if isinstance(v, int):
        return enc_uncompressed(INT, struct.pack("<I", v & 0xFFFFFFFF))
    if isinstance(v, float):
        return enc_uncompressed(FLOAT, struct.pack("<f", v))
    if isinstance(v, str):
        return enc_uncompressed(STRING, enc_string(v))
    if isinstance(v, dict):
        payload = struct.pack("<I", len(v))
        for k, val in v.items():
            payload += enc_variant_uncompressed(k) + enc_variant_uncompressed(val)
        return enc_uncompressed(DICTIONARY, payload)
    if isinstance(v, list):
        payload = struct.pack("<I", len(v))
        for item in v:
            payload += enc_variant_uncompressed(item)
        return enc_uncompressed(ARRAY, payload)
    raise TypeError(f"unsupported variant type: {type(v)}")

def enc_args(args: list) -> bytes:
    if not args:
        return b""
    out = b""
    for a in args:
        out += enc_variant(a)
    return out

# ---------------------------------------------------------------------------
# Packet building / parsing
# ---------------------------------------------------------------------------

def build_rpc(cache_id: int, name_id: int, args: list) -> bytes:
    payload = enc_args(args)
    meta = CMD_REMOTE_CALL
    if cache_id <= 0xFF:
        nid = bytes([cache_id])
    elif cache_id <= 0xFFFF:
        meta |= 1 << 4
        nid = struct.pack("<H", cache_id)
    else:
        meta |= 2 << 4
        nid = struct.pack("<I", cache_id)
    if name_id <= 0xFF:
        name = bytes([name_id])
    else:
        meta |= 1 << 6
        name = struct.pack("<H", name_id)
    pkt = bytes([meta]) + nid + name
    if args:
        pkt += bytes([len(args)]) + payload
    else:
        pkt = bytes([meta | 0x80]) + nid + name
    return pkt

def build_simplify_path(cache_id: int, path: str) -> bytes:
    out = bytes([CMD_SIMPLIFY_PATH])
    md5 = b"\x00" * 32 + b"\x00"
    out += md5 + struct.pack("<I", cache_id)
    out += path.encode("utf-8") + b"\x00"
    return out

def parse_packet(buf: bytes):
    if len(buf) < 1:
        return 0, None, None
    meta = buf[0]
    cmd = meta & 0x07
    nid_comp = (meta >> 4) & 0x03
    name_comp = (meta >> 6) & 0x01
    byte_only = (meta >> 7) & 0x01
    ofs = 1
    if nid_comp == 0:
        ofs += 1
    elif nid_comp == 1:
        ofs += 2
    elif nid_comp == 2:
        ofs += 4
    if name_comp == 0:
        name_id = buf[ofs]
        ofs += 1
    else:
        name_id = struct.unpack_from("<H", buf, ofs)[0]
        ofs += 2
    if cmd != CMD_REMOTE_CALL:
        return cmd, None, None
    arg_count = 0
    if not byte_only:
        arg_count = buf[ofs]
        ofs += 1
    try:
        args, _ = parse_variants(buf[ofs:], arg_count)
    except (ValueError, struct.error):
        args = []
    return cmd, name_id, args

def parse_variants(buf: bytes, count: int):
    args = []
    ofs = 0
    for _ in range(count):
        v, n = parse_variant(buf[ofs:])
        args.append(v)
        ofs += n
    return args, ofs

def parse_variant(buf: bytes):
    t = buf[0] & 0x3F
    emode = buf[0] & 0xC0
    if t == BOOL:
        return bool(buf[0] & 0x80), 1
    if t == INT:
        if emode == 0:
            return struct.unpack_from("<b", buf, 1)[0], 2
        if emode == 1 << 6:
            return struct.unpack_from("<h", buf, 1)[0], 3
        if emode == 2 << 6:
            return struct.unpack_from("<i", buf, 1)[0], 5
        return struct.unpack_from("<q", buf, 1)[0], 9
    if t == FLOAT:
        return struct.unpack_from("<f", buf, 1)[0], 5
    if t == STRING:
        n = struct.unpack_from("<I", buf, 4)[0]
        s = buf[8:8 + n].decode("utf-8")
        total = 8 + n
        total += (-total) % 4
        return s, total
    if t == DICTIONARY:
        n = struct.unpack_from("<I", buf, 4)[0]
        ofs = 8
        d = {}
        for _ in range(n):
            k, klen = parse_variant_uncompressed(buf[ofs:])
            v, vlen = parse_variant_uncompressed(buf[ofs + klen:])
            d[k] = v
            ofs += klen + vlen
        return d, ofs
    if t == ARRAY:
        n = struct.unpack_from("<I", buf, 4)[0]
        ofs = 8
        arr = []
        for _ in range(n):
            v, vlen = parse_variant_uncompressed(buf[ofs:])
            arr.append(v)
            ofs += vlen
        return arr, ofs
    raise ValueError(f"unsupported variant type {t}")

def parse_variant_uncompressed(buf: bytes):
    if len(buf) < 4:
        raise ValueError("buffer too small for variant header")
    t = struct.unpack_from("<I", buf, 0)[0] & 0xFF
    if t == BOOL:
        return bool(struct.unpack_from("<I", buf, 4)[0]), 8
    if t == INT:
        return struct.unpack_from("<i", buf, 4)[0], 8
    if t == FLOAT:
        return struct.unpack_from("<f", buf, 4)[0], 8
    if t == STRING:
        n = struct.unpack_from("<I", buf, 4)[0]
        s = buf[8:8 + n].decode("utf-8")
        total = 8 + n
        total += (-total) % 4
        return s, total
    if t == DICTIONARY:
        n = struct.unpack_from("<I", buf, 4)[0]
        ofs = 8
        d = {}
        for _ in range(n):
            k, klen = parse_variant_uncompressed(buf[ofs:])
            v, vlen = parse_variant_uncompressed(buf[ofs + klen:])
            d[k] = v
            ofs += klen + vlen
        return d, ofs
    if t == ARRAY:
        n = struct.unpack_from("<I", buf, 4)[0]
        ofs = 8
        arr = []
        for _ in range(n):
            v, vlen = parse_variant_uncompressed(buf[ofs:])
            arr.append(v)
            ofs += vlen
        return arr, ofs
    raise ValueError(f"unsupported variant type {t}")

# ---------------------------------------------------------------------------
# World state
# ---------------------------------------------------------------------------

class WorldState:
    """Tracks in-game entities from server pushes."""

    def __init__(self):
        self.lock = threading.Lock()
        self.players = {}    # peer_id -> {nick, pos, hp, ...}
        self.npcs = {}       # peer_id -> {name, pos, hp, ...}
        self.drops = {}      # drop_rid -> {item_id, pos, count}
        self.self_peer = None
        self.self_nick = None
        self.self_pos = (0.0, 0.0)
        self.self_hp = 0
        self.self_max_hp = 0
        self.entities = {}   # peer_id -> full entity dict

    def update_from_push(self, name_id, args):
        with self.lock:
            if name_id == 55:  # PreloadPlayer
                self._handle_preload_player(args)
            elif name_id == 54:  # PreloadEntity
                self._handle_preload_entity(args)
            elif name_id == 112:  # UpdateEntity
                self._handle_update_entity(args)
            elif name_id == 61:  # RemoveEntity
                self._handle_remove_entity(args)
            elif name_id == 41:  # ItemAdded
                self._handle_item_added(args)
            elif name_id == 43:  # ItemRemoved
                self._handle_item_removed(args)
            elif name_id == 114:  # UpdatePublicStats
                self._handle_update_stats(args)
            elif name_id == 0:  # AddOnlinePlayer
                self._handle_add_online(args)

    def _handle_preload_player(self, args):
        if not args or not isinstance(args[0], dict):
            return
        d = args[0]
        pid = d.get("peer_id") or d.get("rid") or d.get("id")
        if pid is not None:
            self.players[pid] = d
            if self.self_peer and pid == self.self_peer:
                self._update_self(d)
            log.debug(f"PreloadPlayer: {d.get('nick', '?')} (peer={pid})")

    def _handle_preload_entity(self, args):
        if not args or not isinstance(args[0], dict):
            return
        d = args[0]
        pid = d.get("peer_id") or d.get("rid") or d.get("id")
        if pid is not None:
            self.npcs[pid] = d
            self.entities[pid] = d
            log.debug(f"PreloadEntity: {d.get('name', '?')} (peer={pid})")

    def _handle_update_entity(self, args):
        if not args or not isinstance(args[0], dict):
            return
        d = args[0]
        pid = d.get("peer_id") or d.get("rid") or d.get("id")
        if pid is not None:
            self.entities[pid] = d
            if pid in self.players:
                self.players[pid] = d
            if pid in self.npcs:
                self.npcs[pid] = d
            if self.self_peer and pid == self.self_peer:
                self._update_self(d)

    def _handle_remove_entity(self, args):
        if not args:
            return
        pid = args[0] if isinstance(args[0], int) else None
        if pid is not None:
            self.players.pop(pid, None)
            self.npcs.pop(pid, None)
            self.entities.pop(pid, None)

    def _handle_item_added(self, args):
        if not args or not isinstance(args[0], dict):
            return
        d = args[0]
        rid = d.get("rid") or d.get("drop_rid") or d.get("id")
        if rid is not None:
            self.drops[rid] = d

    def _handle_item_removed(self, args):
        if not args:
            return
        rid = args[0] if isinstance(args[0], int) else None
        if rid is not None:
            self.drops.pop(rid, None)

    def _handle_update_stats(self, args):
        if not args:
            return
        # args: [peer_id, health, max_health, mana, max_mana, stamina, max_stamina, ...]
        pid = args[0] if len(args) > 0 else None
        if self.self_peer and pid == self.self_peer:
            if len(args) > 1:
                self.self_hp = args[1]
            if len(args) > 2:
                self.self_max_hp = args[2]

    def _handle_add_online(self, args):
        pass

    def _update_self(self, d):
        self.self_nick = d.get("nick")
        pos = d.get("pos") or d.get("position") or d.get("click_pos")
        if pos and isinstance(pos, (list, tuple)) and len(pos) >= 2:
            self.self_pos = (float(pos[0]), float(pos[1]))
        hp = d.get("health") or d.get("hp")
        if hp is not None:
            self.self_hp = hp
        max_hp = d.get("max_health") or d.get("max_hp")
        if max_hp is not None:
            self.self_max_hp = max_hp

    def get_snapshot(self):
        with self.lock:
            return {
                "self": {
                    "peer": self.self_peer,
                    "nick": self.self_nick,
                    "pos": list(self.self_pos),
                    "hp": self.self_hp,
                    "max_hp": self.self_max_hp,
                },
                "players": {str(k): v for k, v in self.players.items()},
                "npcs": {str(k): v for k, v in self.npcs.items()},
                "drops": {str(k): v for k, v in self.drops.items()},
                "entity_count": len(self.entities),
            }

    def find_nearest_npc(self, max_dist=300):
        """Find the nearest NPC/monster to self."""
        with self.lock:
            if not self.self_pos:
                return None
            sx, sy = self.self_pos
            best = None
            best_dist = float("inf")
            for pid, npc in self.npcs.items():
                pos = npc.get("pos") or npc.get("position") or npc.get("click_pos")
                if pos and isinstance(pos, (list, tuple)) and len(pos) >= 2:
                    nx, ny = float(pos[0]), float(pos[1])
                    dist = ((sx - nx) ** 2 + (sy - ny) ** 2) ** 0.5
                    if dist < best_dist and dist <= max_dist:
                        best_dist = dist
                        best = (pid, npc)
            return best

    def find_nearest_drop(self, max_dist=100):
        """Find the nearest drop to self."""
        with self.lock:
            if not self.self_pos:
                return None
            sx, sy = self.self_pos
            best = None
            best_dist = float("inf")
            for rid, drop in self.drops.items():
                pos = drop.get("pos") or drop.get("position") or drop.get("click_pos")
                if pos and isinstance(pos, (list, tuple)) and len(pos) >= 2:
                    dx, dy = float(pos[0]), float(pos[1])
                    dist = ((sx - dx) ** 2 + (sy - dy) ** 2) ** 0.5
                    if dist < best_dist and dist <= max_dist:
                        best_dist = dist
                        best = (rid, drop)
            return best

# ---------------------------------------------------------------------------
# AgentBridge NDJSON server
# ---------------------------------------------------------------------------

class AgentBridgeServer:
    """TCP NDJSON server that exposes game state and accepts actions.

    Mirrors the Go server's AgentBridge protocol so existing AI harness
    (tests/harness/sombase.py) works with this client.
    """

    def __init__(self, client, port=19101):
        self.client = client
        self.port = port
        self.server = None
        self.running = False
        self.events = []  # list of event dicts
        self.event_seq = 0

    def start(self):
        self.server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        self.server.bind(("0.0.0.0", self.port))
        self.server.listen(16)
        self.running = True
        t = threading.Thread(target=self._accept_loop, daemon=True)
        t.start()
        log.info(f"AgentBridge listening on :{self.port}")

    def stop(self):
        self.running = False
        if self.server:
            self.server.close()

    def add_event(self, kind, data=None):
        with threading.Lock():
            self.event_seq += 1
            self.events.append({
                "seq": self.event_seq,
                "kind": kind,
                "data": data or {},
                "frame": int(time.time() * 10),
            })
            if len(self.events) > 512:
                self.events = self.events[-512:]

    def _accept_loop(self):
        while self.running:
            try:
                conn, addr = self.server.accept()
                t = threading.Thread(target=self._handle_conn, args=(conn,), daemon=True)
                t.start()
            except OSError:
                break

    def _handle_conn(self, conn):
        try:
            conn.settimeout(30)
            buf = b""
            while self.running:
                data = conn.recv(65536)
                if not data:
                    break
                buf += data
                while b"\n" in buf:
                    line, buf = buf.split(b"\n", 1)
                    line = line.strip()
                    if not line:
                        continue
                    try:
                        req = json.loads(line.decode("utf-8"))
                        reply = self._handle_request(req)
                    except json.JSONDecodeError:
                        reply = {"error": "invalid JSON"}
                    conn.sendall((json.dumps(reply) + "\n").encode("utf-8"))
        except (OSError, socket.timeout):
            pass
        finally:
            conn.close()

    def _handle_request(self, req):
        op = req.get("op", "")
        if op == "ping":
            return {"ok": True, "pong": True}
        elif op == "list_players":
            snap = self.client.world.get_snapshot()
            players = []
            for pid, p in snap["players"].items():
                players.append({
                    "peerID": int(pid),
                    "nick": p.get("nick", "?"),
                    "isAgent": False,
                })
            for pid, n in snap["npcs"].items():
                players.append({
                    "peerID": int(pid),
                    "nick": n.get("name", "?"),
                    "isAgent": True,
                })
            return {"players": players}
        elif op == "snapshot":
            return self.client.world.get_snapshot()
        elif op == "drain_events":
            with threading.Lock():
                evts = self.events[:]
                self.events = []
            return {"events": evts, "count": len(evts)}
        elif op == "apply_action":
            return self._apply_action(req)
        elif op == "latest_verdict":
            return {"verdict": "UNKNOWN", "reason": "no verdict tracked"}
        elif op == "spectate":
            return {"ok": True}
        elif op == "spectate_feed":
            return {"ok": True, "events": []}
        elif op == "spectate_stop":
            return {"ok": True}
        elif op == "close_session":
            return {"ok": True}
        elif op == "arena_start":
            return {"ok": True}
        elif op == "arena_status":
            return {"ok": False, "reason": "not in arena"}
        elif op == "arena_stop":
            return {"ok": True}
        else:
            return {"error": f"unknown op: {op}"}

    def _apply_action(self, req):
        action = req.get("action", "")
        args = req.get("args", {})
        peer_id = req.get("peerID", self.client.peer_id)

        if action == "move_to":
            pos = args.get("pos") or args.get("position") or [0, 0]
            self.client.send_rpc(RPC_SET_CLICK_POS, [float(pos[0]), float(pos[1])])
            return {"ok": True, "verdict": "APPLIED", "action": action}
        elif action == "attack":
            skill_id = args.get("skill_id", args.get("skillID", 0))
            self.client.send_rpc(RPC_TRIGGER_SKILL, [int(skill_id)])
            return {"ok": True, "verdict": "APPLIED", "action": action}
        elif action == "pickup":
            drop_rid = args.get("drop_rid", args.get("dropRID", args.get("rid", 0)))
            self.client.send_rpc(RPC_PICKUP_DROP, [int(drop_rid)])
            return {"ok": True, "verdict": "APPLIED", "action": action}
        elif action == "chat":
            msg = args.get("message", args.get("text", ""))
            self.client.send_rpc(RPC_TRIGGER_CHAT, [str(msg)])
            return {"ok": True, "verdict": "APPLIED", "action": action}
        elif action == "respawn":
            self.client.send_rpc(RPC_TRIGGER_RESPAWN, [])
            return {"ok": True, "verdict": "APPLIED", "action": action}
        elif action == "interact":
            target = args.get("target", args.get("target_rid", 0))
            self.client.send_rpc(RPC_TRIGGER_INTERACT, [int(target)])
            return {"ok": True, "verdict": "APPLIED", "action": action}
        elif action == "emote":
            emote_id = args.get("emote_id", args.get("emoteID", 0))
            self.client.send_rpc(RPC_TRIGGER_EMOTE, [int(emote_id)])
            return {"ok": True, "verdict": "APPLIED", "action": action}
        else:
            return {"error": f"unknown action: {action}"}

# ---------------------------------------------------------------------------
# SoM Headless Client
# ---------------------------------------------------------------------------

class SoMClient:
    def __init__(self, url, account=None, password=None, nick=None,
                 bridge_port=19101, auto_play=False):
        self.url = url
        self.account = account
        self.password = password
        self.nick = nick
        self.auto_play = auto_play
        self.ws = None
        self.peer_id = None
        self.world = WorldState()
        self.bridge = None
        self.running = False
        self.rpc_lock = asyncio.Lock()
        self.in_game = False
        self.self_entity_rid = None
        self.msg_queue = asyncio.Queue()

    async def connect(self):
        """Connect to server and perform handshake."""
        log.info(f"Connecting to {self.url}...")
        self.ws = await websockets.connect(self.url, max_size=1 << 24)
        # Server pushes 4-byte LE unique_id
        raw = await asyncio.wait_for(self.ws.recv(), timeout=10)
        self.peer_id = struct.unpack("<I", raw[:4])[0]
        self.world.self_peer = self.peer_id
        log.info(f"Connected: peer_id={self.peer_id}")
        # Announce Network node
        await self.ws.send(build_simplify_path(CACHE_ID, NODE_PATH))
        # Start receive loop
        self.running = True
        self.msg_queue = asyncio.Queue()
        asyncio.create_task(self._recv_loop())

    async def _recv_loop(self):
        """Background task: receive server pushes and put into queue."""
        while self.running:
            try:
                data = await self.ws.recv()
                cmd, name_id, args = parse_packet(data)
                if cmd == CMD_REMOTE_CALL and name_id is not None:
                    self.world.update_from_push(name_id, args)
                    if self.bridge:
                        self.bridge.add_event("push", {"name_id": name_id, "args": args})
                    await self.msg_queue.put((name_id, args))
            except websockets.exceptions.ConnectionClosed:
                log.warning("Connection closed")
                self.running = False
                break
            except Exception as e:
                log.debug(f"recv error: {e}")

    async def send_rpc_async(self, name_id, args):
        async with self.rpc_lock:
            await self.ws.send(build_rpc(CACHE_ID, name_id, args))

    async def recv_reply(self, skip_ids=None, timeout=10):
        """Receive a specific reply from the message queue."""
        skip_ids = skip_ids or set()
        deadline = time.time() + timeout
        while time.time() < deadline:
            try:
                name_id, args = await asyncio.wait_for(self.msg_queue.get(), timeout=2)
                if name_id not in skip_ids:
                    return name_id, args
            except asyncio.TimeoutError:
                continue
        return None, None

    async def create_account(self, name, password, email=""):
        """Create a new account. Returns True on success."""
        log.info(f"Creating account: {name}")
        await self.send_rpc_async(RPC_CREATE_ACCOUNT, [name, password, email, False, 0])
        # Wait for AuthError(OK)
        deadline = time.time() + 10
        while time.time() < deadline:
            name_id, args = await self.recv_reply(skip_ids=PUSH_IDS, timeout=2)
            if name_id == RPC_AUTH_ERROR:
                err = args[0] if args else -1
                if err == AUTH_ERR_OK:
                    log.info(f"Account created: {name}")
                    return True
                else:
                    log.warning(f"CreateAccount failed: err={err}")
                    return False
        return False

    async def login(self, name, password):
        """Login with existing account. Returns True on success."""
        log.info(f"Logging in: {name}")
        await self.send_rpc_async(RPC_LOGIN_WITH_PASSWORD, [name, password, False, 0])
        deadline = time.time() + 10
        while time.time() < deadline:
            name_id, args = await self.recv_reply(skip_ids=PUSH_IDS, timeout=2)
            if name_id == RPC_AUTH_ERROR:
                err = args[0] if args else -1
                if err == AUTH_ERR_OK:
                    log.info(f"Login successful: {name}")
                    return True
                elif err == 7:  # ERR_NAME_AVAILABLE? No, this is "account not found"
                    log.warning("Account not found, need to create")
                    return False
                else:
                    log.warning(f"Login failed: err={err}")
                    return False
        return False

    async def create_character(self, nick, race=1, gender=0):
        """Create a new character. Returns True on success."""
        log.info(f"Creating character: {nick}")
        await self.send_rpc_async(RPC_CREATE_CHARACTER, [
            nick, {"race": race, "gender": gender}, {}
        ])
        deadline = time.time() + 10
        while time.time() < deadline:
            name_id, args = await self.recv_reply(skip_ids=PUSH_IDS, timeout=2)
            if name_id == RPC_CHARACTER_ERROR:
                err = args[0] if args else -1
                if err == CHAR_ERR_OK:
                    log.info(f"Character created: {nick}")
                    return True
                else:
                    log.warning(f"CreateCharacter failed: err={err}")
                    return False
        return False

    async def list_characters(self):
        """List characters. Returns list of character info dicts."""
        log.info("Listing characters...")
        await self.send_rpc_async(RPC_CHARACTER_LISTING, [])
        chars = []
        deadline = time.time() + 10
        while time.time() < deadline:
            name_id, args = await self.recv_reply(skip_ids=PUSH_IDS, timeout=2)
            if name_id == RPC_CHARACTER_INFO and args and isinstance(args[0], dict):
                chars.append(args[0])
            elif name_id == RPC_CHARACTER_ERROR:
                err = args[0] if args else -1
                if err == CHAR_ERR_OK:
                    break
                else:
                    break
        log.info(f"Found {len(chars)} character(s)")
        return chars

    async def connect_character(self, nick):
        """Connect to a character (enter game). Returns True on success."""
        log.info(f"Connecting character: {nick}")
        await self.send_rpc_async(RPC_CONNECT_CHARACTER, [nick])
        deadline = time.time() + 15
        while time.time() < deadline:
            name_id, args = await self.recv_reply(skip_ids=PUSH_IDS, timeout=2)
            if name_id == RPC_CHARACTER_ERROR:
                err = args[0] if args else -1
                if err == CHAR_ERR_OK:
                    self.in_game = True
                    log.info(f"In game as {nick}!")
                    return True
                else:
                    log.warning(f"ConnectCharacter failed: err={err}")
                    return False
        return False

    def send_rpc(self, name_id, args):
        """Synchronous wrapper for send_rpc_async (for bridge thread)."""
        if self.ws and self.running:
            try:
                loop = asyncio.get_event_loop()
                if loop.is_running():
                    asyncio.create_task(self.send_rpc_async(name_id, args))
                else:
                    loop.run_until_complete(self.send_rpc_async(name_id, args))
            except Exception as e:
                log.debug(f"send_rpc error: {e}")

    # ------------------------------------------------------------------
    # Auto-play
    # ------------------------------------------------------------------

    async def auto_play_loop(self):
        """Built-in auto-play: auto-combat + auto-loot."""
        if not self.auto_play:
            return
        log.info("Auto-play enabled")
        while self.running and self.in_game:
            try:
                # Auto-combat: find nearest NPC and attack
                target = self.world.find_nearest_npc(max_dist=300)
                if target:
                    pid, npc = target
                    pos = npc.get("pos") or npc.get("position") or [0, 0]
                    if pos:
                        await self.send_rpc_async(RPC_SET_CLICK_POS, [float(pos[0]), float(pos[1])])
                        await asyncio.sleep(0.5)
                        await self.send_rpc_async(RPC_TRIGGER_SKILL, [0])  # basic attack
                        log.debug(f"Auto-attack: {npc.get('name', '?')} at {pos}")
                        if self.bridge:
                            self.bridge.add_event("attack", {"target": npc.get("name"), "pos": pos})

                # Auto-loot: find nearest drop
                drop = self.world.find_nearest_drop(max_dist=100)
                if drop:
                    rid, d = drop
                    pos = d.get("pos") or d.get("position") or [0, 0]
                    if pos:
                        await self.send_rpc_async(RPC_SET_CLICK_POS, [float(pos[0]), float(pos[1])])
                        await asyncio.sleep(0.3)
                        await self.send_rpc_async(RPC_PICKUP_DROP, [int(rid)])
                        log.debug(f"Auto-loot: drop {rid} at {pos}")
                        if self.bridge:
                            self.bridge.add_event("loot", {"rid": rid, "pos": pos})

            except Exception as e:
                log.debug(f"auto-play error: {e}")

            await asyncio.sleep(0.5)

    async def run(self):
        """Main run loop."""
        # Start AgentBridge
        if self.bridge:
            self.bridge.start()

        # Start auto-play
        if self.auto_play:
            asyncio.create_task(self.auto_play_loop())

        # Keep running
        while self.running:
            await asyncio.sleep(1)
            snap = self.world.get_snapshot()
            if snap["self"]["nick"]:
                log.info(f"  HP: {snap['self']['hp']}/{snap['self']['max_hp']}  "
                         f"Pos: {snap['self']['pos']}  "
                         f"Players: {len(snap['players'])}  NPCs: {len(snap['npcs'])}  "
                         f"Drops: {len(snap['drops'])}")

    async def close(self):
        self.running = False
        if self.bridge:
            self.bridge.stop()
        if self.ws:
            await self.ws.close()


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def parse_args():
    import argparse
    parser = argparse.ArgumentParser(description="SoM Headless AI Client")
    parser.add_argument("url", nargs="?", default=os.environ.get("SOM_WS_URL", "ws://127.0.0.1:27012/ws"),
                        help="WebSocket URL")
    parser.add_argument("--account", default=os.environ.get("SOM_ACCOUNT"),
                        help="Account name")
    parser.add_argument("--password", default=os.environ.get("SOM_PASSWORD"),
                        help="Account password")
    parser.add_argument("--nick", default=os.environ.get("SOM_NICK"),
                        help="Character nickname")
    parser.add_argument("--auto-create", action="store_true",
                        help="Auto-create account and character if not exists")
    parser.add_argument("--bridge-port", type=int,
                        default=int(os.environ.get("SOM_BRIDGE_PORT", "19101")),
                        help="AgentBridge listen port")
    parser.add_argument("--auto-play", action="store_true",
                        default=os.environ.get("SOM_AUTO_PLAY") == "1",
                        help="Enable auto-play (auto-combat + auto-loot)")
    parser.add_argument("--keep", action="store_true", default=True,
                        help="Keep running after login (default: true)")
    parser.add_argument("--no-keep", dest="keep", action="store_false",
                        help="Exit after login")
    parser.add_argument("--debug", action="store_true",
                        help="Enable debug logging")
    return parser.parse_args()


async def main():
    args = parse_args()
    if args.debug:
        logging.getLogger().setLevel(logging.DEBUG)

    client = SoMClient(
        url=args.url,
        account=args.account,
        password=args.password,
        nick=args.nick,
        bridge_port=args.bridge_port,
        auto_play=args.auto_play,
    )

    # Setup signal handlers
    def shutdown(sig, frame):
        log.info("Shutting down...")
        asyncio.create_task(client.close())
        sys.exit(0)
    signal.signal(signal.SIGINT, shutdown)
    signal.signal(signal.SIGTERM, shutdown)

    try:
        await client.connect()

        # Login or create account
        if args.account and args.password:
            ok = await client.login(args.account, args.password)
            if not ok and args.auto_create:
                log.info("Login failed, creating account...")
                await client.create_account(args.account, args.password)
                await client.login(args.account, args.password)
        elif args.auto_create:
            # Generate random account name (alphanumeric only, 4-16 chars)
            import random
            import string
            account = "".join(random.choices(string.ascii_letters + string.digits, k=8))
            password = "botpassword123"
            log.info(f"Auto-creating account: {account}")
            await client.create_account(account, password)
            await client.login(account, password)
            client.account = account
            client.password = password
        else:
            log.error("No account/password provided. Use --auto-create or --account/--password")
            return

        # List characters
        chars = await client.list_characters()

        # Create character if needed
        if not chars and args.auto_create:
            nick = args.nick or f"AI_{client.account}"
            log.info(f"No characters found, creating: {nick}")
            await client.create_character(nick)
            chars = await client.list_characters()

        if not chars:
            log.error("No characters available. Use --auto-create to create one.")
            return

        # Connect to first character (or match by nick)
        nick = args.nick or chars[0].get("nickname")
        if not await client.connect_character(nick):
            log.error("Failed to connect character")
            return

        # Start AgentBridge
        client.bridge = AgentBridgeServer(client, port=args.bridge_port)

        # Run
        if args.keep:
            log.info("Running... (Ctrl+C to stop)")
            await client.run()
        else:
            log.info("Login complete, exiting")

    except KeyboardInterrupt:
        log.info("Interrupted")
    except Exception as e:
        log.error(f"Fatal error: {e}", exc_info=True)
    finally:
        await client.close()


if __name__ == "__main__":
    asyncio.run(main())
