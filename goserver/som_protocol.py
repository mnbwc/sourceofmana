#!/usr/bin/env python3
"""SoM shared protocol layer — Godot SceneMultiplayer over WebSocket.

Extracted from headless_ai_client.py so the AI headless client and the human
lite client (lite_human_client.py) share one protocol implementation.

Wire protocol (Godot 4 SceneRPCInterface):
  - Variant encoding: BOOL/INT/FLOAT/STRING/VECTOR2/DICTIONARY/ARRAY with
    2-bit integer compression and 4-byte alignment for strings.
  - Packets: [meta][nodeID][nameID][argCount?][variants...]
  - RPC name IDs are the sorted indices of the 119 @rpc methods in
    sources/network/Network.gd (see RPC_* constants below).

Movement is client-authoritative in SoM: the server never sends position
updates back to the mover, so both clients predict locally at 120 px/s.

RPC bug fixes vs the original headless_ai_client.py:
  - PickupDrop is nameID 53 (was 40 = HighlightUI).
  - TriggerSkill takes (targetRID, skillID) — the old code sent only one arg.
  - TriggerChat takes (channel, text) — the old code sent only the text.

Usage:
    client = SoMClient(url, nick="Hero")
    await client.connect()
    await client.login("acct", "pass")
    chars = await client.list_characters()
    await client.connect_character(chars[0]["nickname"])
    await client.move_to(300.0, 200.0)
    await client.attack(target_rid, skill_id=SKILL_MELEE)
"""

import asyncio
import logging
import queue
import struct
import threading
import time

try:
    import websockets
except ImportError:
    raise ImportError("'websockets' package required. Install: pip install websockets")

log = logging.getLogger("som-protocol")

# ---------------------------------------------------------------------------
# Godot variant types
# ---------------------------------------------------------------------------

NIL = 0
BOOL = 1
INT = 2
FLOAT = 3
STRING = 4
VECTOR2 = 5
STRING_NAME = 21
DICTIONARY = 27
ARRAY = 28
PACKED_STRING_ARRAY = 34

# SceneMultiplayer command types
CMD_REMOTE_CALL = 0
CMD_SIMPLIFY_PATH = 1
CMD_CONFIRM_PATH = 2

# ---------------------------------------------------------------------------
# RPC name IDs — sorted @rpc method list of sources/network/Network.gd
# (119 methods, verified against the Godot project 2026-11-12).
# ---------------------------------------------------------------------------

RPC_ADD_ONLINE_PLAYER = 0
RPC_AUTH_ERROR = 1
RPC_AUTH_TOKEN_RESULT = 2
RPC_CASTED = 6
RPC_CHARACTER_ERROR = 8
RPC_CHARACTER_INFO = 9
RPC_CHARACTER_LISTING = 10
RPC_CHAT_PLAYER = 11
RPC_CHAT_QUERY = 12
RPC_CHAT_SYSTEM = 13
RPC_CLEAR_NAVIGATION = 14
RPC_COMMAND_FEEDBACK = 16
RPC_CONNECT_CHARACTER = 19
RPC_CONTEXT_CHOICE = 20
RPC_CONTEXT_CLOSE = 21
RPC_CONTEXT_CONTINUE = 22
RPC_CONTEXT_TEXT = 23
RPC_CREATE_ACCOUNT = 25
RPC_CREATE_CHARACTER = 26
RPC_DROP_ADDED = 32
RPC_DROP_REMOVED = 34
RPC_EMOTE = 35
RPC_FULL_UPDATE_ENTITY = 39
RPC_ITEM_ADDED = 41
RPC_ITEM_REMOVED = 43
RPC_LEVEL_UP = 44
RPC_LOGIN_WITH_PASSWORD = 45
RPC_LOGIN_WITH_TOKEN = 46
RPC_PICKUP_DROP = 53
RPC_PRELOAD_ENTITY = 54
RPC_PRELOAD_PLAYER = 55
RPC_PUSH_NOTIFICATION = 56
RPC_REFRESH_EQUIPMENT = 57
RPC_REFRESH_INVENTORY = 58
RPC_REFRESH_ONLINE_LIST = 59
RPC_REFRESH_PROGRESS = 60
RPC_REMOVE_ENTITY = 61
RPC_REMOVE_ONLINE_PLAYER = 62
RPC_RETRIEVE_INVENTORY = 67
RPC_SET_ATTRIBUTES = 74
RPC_SET_CLICK_POS = 75
RPC_SET_MOVE_POS = 76
RPC_SET_ORIENTATION = 77
RPC_TARGET_ALTERATION = 79
RPC_THROW_PROJECTILE = 80
RPC_TOGGLE_CONTEXT = 81
RPC_TRIGGER_CHAT = 90
RPC_TRIGGER_EMOTE = 94
RPC_TRIGGER_INTERACT = 96
RPC_TRIGGER_PARTY_ACCEPT = 98
RPC_TRIGGER_RESPAWN = 104
RPC_TRIGGER_SELECT = 105
RPC_TRIGGER_SKILL = 107
RPC_UNEQUIP_ITEM = 108
RPC_UPDATE_ATTRIBUTES = 109
RPC_UPDATE_BESTIARY = 110
RPC_UPDATE_ENTITY = 112
RPC_UPDATE_PRIVATE_STATS = 113
RPC_UPDATE_PUBLIC_STATS = 114
RPC_UPDATE_QUEST = 115
RPC_UPDATE_SKILL = 116
RPC_USE_ITEM = 117
RPC_WARP_PLAYER = 118

# Chat channels (SoM ChatCommons)
CHAT_LOCAL = "local"
CHAT_GLOBAL = "global"
CHAT_PARTY = "party"
CHAT_SYSTEM = "system"

# ActorCommons.State
STATE_IDLE = 0
STATE_DEATH = 4

# ActorCommons.Alteration (TargetAlteration arg 3)
ALTERATION_HIT = 0
ALTERATION_DODGE = 1
ALTERATION_CRIT = 2
ALTERATION_EXP = 5

# Skill IDs from the Go server skillTable (combat.go)
SKILL_MELEE = 229218829       # Melee, range 0,  cd 0.5s
SKILL_ARCHER = 2814726266     # Archer, range 100, cd 0.5s
SKILL_FLAR = 2089110122       # Flar, magic, range 120, cd 3s
SKILL_INMA = 2089220490       # Inma, magic, lowers defense
SKILL_MANA_BURST = 2393790290 # ManaBurst, magic, range 40, cd 1s
SKILL_SONIC_WAVE = 123697108  # SonicWave, magic, slows target
SKILL_SPITFIRE = 3219153995   # Spitfire, magic, range 32, cd 1s

# Item IDs (items.go itemCatalog)
ITEM_POTION = 1001
ITEM_ANTIDOTE = 1002

# Server movement constants (movement.go)
WALK_SPEED = 120.0  # px/s
AOI_RADIUS = 1000.0 # px
ATTACK_RANGE_BASE = 32.0  # defaultAttackRange

# Error codes
AUTH_ERR_OK = 0
AUTH_ERR_NAME_SIZE = 10
AUTH_ERR_PASSWORD_SIZE = 11
AUTH_ERR_AUTH = 7
AUTH_ERR_NAME_AVAILABLE = 9
CHAR_ERR_OK = 0
CHAR_ERR_EMPTY_ACCOUNT = 9

CACHE_ID = 5
NODE_PATH = "/root/Network"

# Server pushes that are world state, not login replies.
PUSH_IDS = {
    RPC_ADD_ONLINE_PLAYER, RPC_CHAT_PLAYER, RPC_CHAT_SYSTEM, RPC_CASTED,
    RPC_CLEAR_NAVIGATION, RPC_COMMAND_FEEDBACK, RPC_CONTEXT_CHOICE,
    RPC_CONTEXT_CLOSE, RPC_CONTEXT_CONTINUE, RPC_CONTEXT_TEXT,
    RPC_DROP_ADDED, RPC_DROP_REMOVED, RPC_EMOTE, RPC_FULL_UPDATE_ENTITY,
    RPC_ITEM_ADDED, RPC_ITEM_REMOVED, RPC_LEVEL_UP, RPC_PRELOAD_ENTITY,
    RPC_PRELOAD_PLAYER, RPC_PUSH_NOTIFICATION, RPC_REFRESH_EQUIPMENT,
    RPC_REFRESH_INVENTORY, RPC_REFRESH_ONLINE_LIST, RPC_REFRESH_PROGRESS,
    RPC_REMOVE_ENTITY, RPC_REMOVE_ONLINE_PLAYER, RPC_TARGET_ALTERATION,
    RPC_THROW_PROJECTILE, RPC_TOGGLE_CONTEXT, RPC_TRIGGER_CHAT,
    RPC_TRIGGER_INTERACT, RPC_TRIGGER_RESPAWN, RPC_TRIGGER_SELECT,
    RPC_TRIGGER_SKILL, RPC_UPDATE_ATTRIBUTES, RPC_UPDATE_BESTIARY,
    RPC_UPDATE_ENTITY, RPC_UPDATE_PRIVATE_STATS, RPC_UPDATE_PUBLIC_STATS,
    RPC_UPDATE_QUEST, RPC_UPDATE_SKILL, RPC_USE_ITEM, RPC_WARP_PLAYER,
}

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
    if isinstance(v, tuple) and len(v) == 2:
        return enc_uncompressed(VECTOR2, struct.pack("<ff", float(v[0]), float(v[1])))
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
    if isinstance(v, tuple) and len(v) == 2:
        return enc_uncompressed(VECTOR2, struct.pack("<ff", float(v[0]), float(v[1])))
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
    except (ValueError, struct.error, IndexError) as e:
        # Surface decode failures instead of silently dropping the payload:
        # an unsupported Variant type here means a whole RPC is lost.
        log.warning(f"parse_packet: nameID={name_id} args={arg_count} failed: {e}")
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
        # Non-int/float variants use the full 4-byte header; the server's
        # encodeAndCompressVariant only shortens BOOL/INT.
        return struct.unpack_from("<f", buf, 4)[0], 8
    if t == VECTOR2:
        x = struct.unpack_from("<f", buf, 4)[0]
        y = struct.unpack_from("<f", buf, 8)[0]
        return (x, y), 12
    if t == STRING or t == STRING_NAME:
        n = struct.unpack_from("<I", buf, 4)[0]
        s = buf[8:8 + n].decode("utf-8", errors="replace")
        total = 8 + n
        total += (-total) % 4
        return s, total
    if t == PACKED_STRING_ARRAY:
        n = struct.unpack_from("<I", buf, 4)[0]
        ofs = 8
        arr = []
        for _ in range(n):
            slen = struct.unpack_from("<I", buf, ofs)[0]
            arr.append(buf[ofs + 4:ofs + 4 + slen].decode("utf-8", errors="replace"))
            ofs += 4 + slen
            ofs += (-ofs) % 4
        return arr, ofs
    if t == NIL:
        return None, 4
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
    if t == VECTOR2:
        x = struct.unpack_from("<f", buf, 4)[0]
        y = struct.unpack_from("<f", buf, 8)[0]
        return (x, y), 12
    if t == STRING or t == STRING_NAME:
        n = struct.unpack_from("<I", buf, 4)[0]
        s = buf[8:8 + n].decode("utf-8", errors="replace")
        total = 8 + n
        total += (-total) % 4
        return s, total
    if t == PACKED_STRING_ARRAY:
        n = struct.unpack_from("<I", buf, 4)[0]
        ofs = 8
        arr = []
        for _ in range(n):
            slen = struct.unpack_from("<I", buf, ofs)[0]
            arr.append(buf[ofs + 4:ofs + 4 + slen].decode("utf-8", errors="replace"))
            ofs += 4 + slen
            ofs += (-ofs) % 4
        return arr, ofs
    if t == NIL:
        return None, 4
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

def _pos_of(d) -> tuple:
    """Extract a (x, y) position from an entity dict."""
    pos = d.get("pos") or d.get("position") or d.get("click_pos")
    if pos and isinstance(pos, (list, tuple)) and len(pos) >= 2:
        try:
            return (float(pos[0]), float(pos[1]))
        except (TypeError, ValueError):
            return None
    return None


class WorldState:
    """Tracks in-game entities from server pushes.

    Emits UI events through ``emit`` (a callable set by SoMClient): every
    event is a dict with at least a ``type`` key.
    """

    def __init__(self):
        self.lock = threading.RLock()
        self.emit = None          # callable(event_dict) — set by SoMClient
        self.players = {}         # peer_id -> {nick, pos, hp, level, ...}
        self.npcs = {}            # peer_id -> {name, pos, hp, max_hp, state, ...}
        self.drops = {}           # drop_rid -> {item_id, pos, count}
        self.entities = {}        # peer_id -> merged entity dict (all types)
        self.self_peer = None
        self.self_nick = None
        self.self_pos = (0.0, 0.0)
        self.self_hp = 0
        self.self_max_hp = 0
        self.self_level = 0
        self.self_exp = 0
        self.self_gp = 0
        self.self_mana = 0
        self.self_stamina = 0
        self.self_karma = 0
        self.map_id = 0
        self.inventory = {}       # item_id -> count
        self.skills = {}          # skill_id -> level
        self.quests = {}
        self.bestiary = {}
        self.online_names = []
        self.stats = {"kills": 0, "loots": 0, "deaths": 0, "dmg_dealt": 0}
        self.chat_log = []        # [{channel, name, text, t}]

    # -- event helper -------------------------------------------------------

    def _emit(self, event: dict):
        if self.emit:
            try:
                self.emit(event)
            except Exception as e:
                log.debug(f"emit error: {e}")

    # -- push dispatch ------------------------------------------------------

    def update_from_push(self, name_id, args):
        with self.lock:
            if name_id == RPC_PRELOAD_PLAYER:
                self._handle_preload_player(args)
            elif name_id == RPC_PRELOAD_ENTITY:
                self._handle_preload_entity(args)
            elif name_id == RPC_WARP_PLAYER:
                self._handle_warp(args)
            elif name_id == RPC_UPDATE_ENTITY:
                self._handle_update_entity(args)
            elif name_id == RPC_FULL_UPDATE_ENTITY:
                self._handle_full_update(args)
            elif name_id == RPC_REMOVE_ENTITY:
                self._handle_remove_entity(args)
            elif name_id == RPC_DROP_ADDED:
                self._handle_drop_added(args)
            elif name_id == RPC_DROP_REMOVED:
                self._handle_drop_removed(args)
            elif name_id == RPC_ITEM_ADDED:
                self._handle_item_added(args)
            elif name_id == RPC_ITEM_REMOVED:
                self._handle_item_removed(args)
            elif name_id == RPC_TARGET_ALTERATION:
                self._handle_target_alteration(args)
            elif name_id == RPC_CASTED:
                self._handle_casted(args)
            elif name_id == RPC_CHAT_PLAYER:
                self._handle_chat_player(args)
            elif name_id == RPC_CHAT_SYSTEM:
                self._handle_chat_system(args)
            elif name_id == RPC_PUSH_NOTIFICATION:
                self._handle_push_notification(args)
            elif name_id == RPC_UPDATE_PUBLIC_STATS:
                self._handle_update_stats(args)
            elif name_id == RPC_UPDATE_PRIVATE_STATS:
                self._handle_private_stats(args)
            elif name_id == RPC_LEVEL_UP:
                self._handle_level_up(args)
            elif name_id == RPC_REFRESH_INVENTORY:
                self._handle_refresh_inventory(args)
            elif name_id == RPC_REFRESH_EQUIPMENT:
                pass  # equipment is cosmetic; the lite client does not render it
            elif name_id == RPC_REFRESH_ONLINE_LIST:
                self._handle_refresh_online_list(args)
            elif name_id == RPC_ADD_ONLINE_PLAYER:
                self._handle_add_online(args)
            elif name_id == RPC_REMOVE_ONLINE_PLAYER:
                self._handle_remove_online(args)
            elif name_id == RPC_REFRESH_PROGRESS:
                self._handle_refresh_progress(args)
            elif name_id == RPC_UPDATE_SKILL:
                self._handle_update_skill(args)
            elif name_id == RPC_UPDATE_QUEST:
                self._handle_update_quest(args)
            elif name_id == RPC_UPDATE_BESTIARY:
                self._handle_update_bestiary(args)
            elif name_id == RPC_CONTEXT_TEXT:
                self._handle_context_text(args)
            elif name_id == RPC_CONTEXT_CHOICE:
                self._handle_context_choice(args)
            elif name_id == RPC_COMMAND_FEEDBACK:
                self._handle_command_feedback(args)

    # -- entity handlers ----------------------------------------------------

    def _handle_warp(self, args):
        # WarpPlayer: [mapID(int), position(vec2)]
        if not args or len(args) < 2:
            return
        pos = args[1]
        if pos and isinstance(pos, (list, tuple)) and len(pos) >= 2:
            self.map_id = int(args[0])
            self.self_pos = (float(pos[0]), float(pos[1]))
            self._emit({"type": "warp", "map_id": self.map_id, "pos": list(self.self_pos)})
            log.debug(f"WarpPlayer: map={args[0]} pos=({pos[0]},{pos[1]})")

    def _handle_preload_player(self, args):
        # preloadPlayerArgs: [agentRID(=CharID), spirit, currentShape, nick,
        #   level, health, hairstyle, haircolor, gender, race, skintone, equipment]
        # The server preloads every world entity this way: positive CharID for
        # players, negative CharID for seeded NPCs.
        if not args:
            return
        if isinstance(args[0], dict):
            d = args[0]
            pid = d.get("peer_id") or d.get("rid") or d.get("id")
            if pid is not None:
                self._ingest_preload(pid, d)
            return
        if len(args) < 4 or not isinstance(args[0], int):
            return
        pid = args[0]
        d = {
            "peer_id": pid,
            "nick": args[3] if len(args) > 3 else None,
            "level": args[4] if len(args) > 4 else 0,
            "health": args[5] if len(args) > 5 else 0,
        }
        self._ingest_preload(pid, d)

    def _ingest_preload(self, pid, d):
        if pid < 0:
            # NPC arriving via PreloadPlayer (negative CharID namespace).
            npc = self.npcs.get(pid)
            if npc is None:
                npc = {"char_id": pid, "rid": -pid, "friendly": False}
                self.npcs[pid] = npc
            npc.update(d)
            npc["char_id"] = pid
            self.entities[pid] = npc
            return
        self.players[pid] = d
        self.entities[pid] = d
        if self.self_nick and d.get("nick") == self.self_nick:
            self.self_peer = pid
            self._update_self(d)
            log.info(f"Self identified: {d['nick']} (rid={pid})")

    def _handle_preload_entity(self, args):
        # PreloadEntity payload: [agentRID, actorType, currentShape, nick,
        #   defaultState]. Seeded NPCs use their positive RID (CharID = -RID);
        # arena monsters use their negative CharID directly.
        if not args:
            return
        if isinstance(args[0], dict):
            d = args[0]
            pid = d.get("peer_id") or d.get("rid") or d.get("id")
            if pid is not None:
                self._ingest_npc_preload(pid, d)
            return
        if len(args) < 4 or not isinstance(args[0], int):
            return
        pid = args[0]
        d = {
            "name": args[3] if len(args) > 3 else None,
            "actor_type": args[1] if len(args) > 1 else 0,
            "state": args[4] if len(args) > 4 else 0,
        }
        self._ingest_npc_preload(pid, d)

    def _ingest_npc_preload(self, eid, d):
        if eid > 0:
            rid, char_id, friendly = eid, -eid, True
        else:
            rid, char_id, friendly = -eid, eid, False
        npc = self.npcs.get(char_id)
        if npc is None:
            npc = {"char_id": char_id, "rid": rid, "friendly": friendly}
            self.npcs[char_id] = npc
        npc.update(d)
        npc["char_id"] = char_id
        npc["rid"] = rid
        npc["friendly"] = friendly
        self.entities[char_id] = npc

    def _handle_update_entity(self, args):
        # updateEntityArgs: [agentRID, velocity(vec2), position(vec2), frameID]
        if not args:
            return
        if isinstance(args[0], dict):
            d = args[0]
            pid = d.get("peer_id") or d.get("rid") or d.get("id")
            if pid is not None:
                self._merge_entity(pid, d)
            return
        if len(args) < 3 or not isinstance(args[0], int):
            return
        pid = args[0]
        pos = args[2] if len(args) > 2 else None
        self._merge_entity(pid, {
            "peer_id": pid,
            "pos": list(pos) if pos and isinstance(pos, (list, tuple)) and len(pos) >= 2 else None,
        })

    def _handle_full_update(self, args):
        # fullUpdateArgs: [agentRID, velocity, position, orientation,
        #   agentState, skillCastID, isRunning, frameID]
        if not args or not isinstance(args[0], int):
            return
        pid = args[0]
        pos = args[2] if len(args) > 2 else None
        d = {
            "peer_id": pid,
            "pos": list(pos) if pos and isinstance(pos, (list, tuple)) and len(pos) >= 2 else None,
            "state": args[4] if len(args) > 4 else 0,
            "running": args[6] if len(args) > 6 else False,
        }
        self._merge_entity(pid, d)
        if d.get("state") == STATE_DEATH:
            ent = self.entities.get(pid, {})
            self._emit({"type": "death", "rid": pid, "pos": d.get("pos"),
                        "name": ent.get("name") or ent.get("nick")})
            if self.self_peer and pid == self.self_peer:
                self.stats["deaths"] += 1
                self._emit({"type": "self_dead"})

    def _merge_entity(self, pid, d):
        # UpdateEntity/FullUpdateEntity carry agentRID = CharID: positive for
        # players, negative for NPCs/monsters. Each dict is shared between the
        # type-specific map and the merged entities view.
        if pid < 0:
            npc = self.npcs.get(pid)
            if npc is None:
                npc = {"char_id": pid, "rid": -pid, "friendly": False}
                self.npcs[pid] = npc
            npc.update(d)
            npc["char_id"] = pid
            self.entities[pid] = npc
            return
        player = self.players.get(pid)
        if player is None:
            player = {"peer_id": pid}
            self.players[pid] = player
        player.update(d)
        player["peer_id"] = pid
        self.entities[pid] = player
        if self.self_peer and pid == self.self_peer:
            self._update_self(player)

    def _handle_remove_entity(self, args):
        if not args:
            return
        pid = args[0] if isinstance(args[0], int) else None
        if pid is None:
            return
        for key in (pid, -pid):
            self.players.pop(key, None)
            self.npcs.pop(key, None)
            self.entities.pop(key, None)

    # -- item / drop handlers ----------------------------------------------

    def _handle_drop_added(self, args):
        # DropAdded: [dropID, itemID, customfield, pos(vec2)]
        if not args or not isinstance(args[0], int):
            return
        rid = args[0]
        pos = args[3] if len(args) > 3 else None
        d = {
            "rid": rid,
            "item_id": args[1] if len(args) > 1 else 0,
            "customfield": args[2] if len(args) > 2 else "",
            "pos": list(pos) if pos and isinstance(pos, (list, tuple)) and len(pos) >= 2 else None,
        }
        self.drops[rid] = d
        self._emit({"type": "drop_add", "rid": rid, "item_id": d["item_id"], "pos": d["pos"]})

    def _handle_drop_removed(self, args):
        if not args:
            return
        rid = args[0] if isinstance(args[0], int) else None
        if rid is not None:
            self.drops.pop(rid, None)

    def _handle_item_added(self, args):
        # ItemAdded: [itemID, customfield, count]
        if not args or not isinstance(args[0], int):
            return
        item_id = args[0]
        count = args[2] if len(args) > 2 else 1
        self.inventory[item_id] = self.inventory.get(item_id, 0) + int(count)
        self._emit({"type": "item_add", "item_id": item_id, "count": count})

    def _handle_item_removed(self, args):
        # ItemRemoved: [itemID, customfield, count, ?]
        if not args or not isinstance(args[0], int):
            return
        item_id = args[0]
        count = args[2] if len(args) > 2 else 1
        left = self.inventory.get(item_id, 0) - int(count)
        if left > 0:
            self.inventory[item_id] = left
        else:
            self.inventory.pop(item_id, None)
        self._emit({"type": "item_remove", "item_id": item_id, "count": count})

    def _handle_refresh_inventory(self, args):
        # RefreshInventory: [Array[Dictionary]] with item_id/customfield/count
        if not args:
            return
        arr = args[0] if isinstance(args[0], list) else None
        if arr is None:
            return
        inv = {}
        for entry in arr:
            if isinstance(entry, dict):
                iid = entry.get("item_id")
                cnt = entry.get("count", 1)
                if iid is not None:
                    inv[int(iid)] = inv.get(int(iid), 0) + int(cnt)
        self.inventory = inv

    # -- combat handlers ----------------------------------------------------

    def _handle_target_alteration(self, args):
        # TargetAlteration: [agentRID, targetRID, value, alteration, skillID, true]
        if not args or len(args) < 4 or not isinstance(args[0], int):
            return
        attacker, target, value, alt = args[0], args[1], args[2], args[3]
        tpos = None
        if isinstance(target, int):
            ent = self.entities.get(target)
            if ent:
                tpos = _pos_of(ent)
        if alt == ALTERATION_EXP:
            if self.self_peer and attacker == self.self_peer:
                self.self_exp += int(value)
                self.stats["kills"] += 1
                self._emit({"type": "exp", "value": int(value), "total": self.self_exp})
            return
        # Damage / dodge / crit on an entity.
        if isinstance(target, int) and target in self.npcs:
            npc = self.npcs[target]
            hp = npc.get("hp")
            if hp is not None:
                npc["hp"] = max(0, int(hp) - int(value))
        if isinstance(target, int) and self.self_peer and target == self.self_peer:
            self.self_hp = max(0, self.self_hp - int(value))
        if isinstance(attacker, int) and self.self_peer and attacker == self.self_peer:
            self.stats["dmg_dealt"] += int(value)
        self._emit({
            "type": "damage",
            "attacker": attacker,
            "target": target,
            "value": int(value),
            "alt": int(alt),
            "pos": tpos,
        })

    def _handle_casted(self, args):
        # Casted: [agentRID, skillID, cooldown]
        if not args or not isinstance(args[0], int):
            return
        caster = args[0]
        skill_id = args[1] if len(args) > 1 else 0
        ent = self.entities.get(caster)
        self._emit({
            "type": "cast",
            "caster": caster,
            "skill_id": skill_id,
            "pos": _pos_of(ent) if ent else None,
        })

    def _handle_update_stats(self, args):
        # UpdatePublicStats: [agentRID, level, health, maxHealth, hairstyle,
        #   haircolor, gender, race, skintone, currentShape]
        # agentRID is the CharID for players (positive) and arena monsters
        # (negative), but the positive RID for seeded NPCs (TriggerSelect).
        if not args or not isinstance(args[0], int):
            return
        pid = args[0]
        d = {
            "level": args[1] if len(args) > 1 else 0,
            "hp": args[2] if len(args) > 2 else 0,
            "max_hp": args[3] if len(args) > 3 else 0,
        }
        if self.self_peer and pid == self.self_peer:
            self.self_level = d["level"]
            self.self_hp = d["hp"]
            self.self_max_hp = d["max_hp"]
            return
        if pid < 0:
            npc = self.npcs.get(pid)
            if npc is not None:
                npc.update(d)
                self._emit({"type": "npc_stats", "rid": pid, **d})
            return
        # Positive: a player CharID first, then a seeded-NPC RID.
        if pid in self.players:
            self.players[pid].update(d)
            return
        npc = self.npcs.get(-pid)
        if npc is not None:
            npc.update(d)
            self._emit({"type": "npc_stats", "rid": pid, **d})

    def _handle_private_stats(self, args):
        # UpdatePrivateStats: [exp, gp, mana, stamina, karma, weight, shape, spirit]
        def num(i):
            return args[i] if len(args) > i and isinstance(args[i], (int, float)) else 0
        self.self_exp = num(0)
        self.self_gp = num(1)
        self.self_mana = num(2)
        self.self_stamina = num(3)
        self.self_karma = num(4)

    def _handle_level_up(self, args):
        # LevelUp: [charID]
        rid = args[0] if args and isinstance(args[0], int) else None
        ent = self.entities.get(rid) if rid is not None else None
        self._emit({"type": "levelup", "rid": rid, "pos": _pos_of(ent) if ent else None})
        if self.self_peer and rid == self.self_peer:
            self.self_level += 1

    # -- chat / notification handlers ---------------------------------------

    def _handle_chat_player(self, args):
        # ChatPlayer: [channel, name, text, charID]
        if not args or len(args) < 3:
            return
        msg = {
            "channel": args[0] if len(args) > 0 else "",
            "name": args[1] if len(args) > 1 else "",
            "text": args[2] if len(args) > 2 else "",
            "t": time.time(),
        }
        self.chat_log.append(msg)
        if len(self.chat_log) > 200:
            self.chat_log = self.chat_log[-200:]
        self._emit({"type": "chat", **msg})

    def _handle_chat_system(self, args):
        # ChatSystem: [channel, text, peerID]
        if not args or len(args) < 2:
            return
        msg = {
            "channel": args[0] if len(args) > 0 else CHAT_SYSTEM,
            "name": "",
            "text": args[1] if len(args) > 1 else "",
            "t": time.time(),
        }
        self.chat_log.append(msg)
        if len(self.chat_log) > 200:
            self.chat_log = self.chat_log[-200:]
        self._emit({"type": "system", **msg})

    def _handle_push_notification(self, args):
        # PushNotification: [text, peerID]
        if not args:
            return
        text = args[0] if isinstance(args[0], str) else str(args[0])
        self._emit({"type": "notify", "text": text})

    def _handle_command_feedback(self, args):
        # CommandFeedback: [text, peerID]
        if not args:
            return
        text = args[0] if isinstance(args[0], str) else str(args[0])
        self._emit({"type": "system", "channel": CHAT_SYSTEM, "name": "", "text": text, "t": time.time()})

    def _handle_context_text(self, args):
        # ContextText: [author, text]
        if not args or len(args) < 2:
            return
        self._emit({
            "type": "dialog",
            "author": args[0] if isinstance(args[0], str) else "",
            "text": args[1] if isinstance(args[1], str) else "",
        })

    def _handle_context_choice(self, args):
        # ContextChoice: [Array[String]]
        choices = args[0] if args and isinstance(args[0], list) else []
        self._emit({"type": "dialog_choice", "choices": list(choices)})

    # -- online list / progress handlers ------------------------------------

    def _handle_refresh_online_list(self, args):
        # RefreshOnlineList: PackedStringArray of names
        if args and isinstance(args[0], list):
            self.online_names = list(args[0])

    def _handle_add_online(self, args):
        if args and isinstance(args[0], str) and args[0] not in self.online_names:
            self.online_names.append(args[0])
            self._emit({"type": "online", "name": args[0]})

    def _handle_remove_online(self, args):
        if args and isinstance(args[0], str):
            if args[0] in self.online_names:
                self.online_names.remove(args[0])
            self._emit({"type": "offline", "name": args[0]})

    def _handle_refresh_progress(self, args):
        # RefreshProgress: [skills dict, quests dict, bestiary dict]
        if not args:
            return
        if isinstance(args[0], dict):
            self.skills = {int(k): int(v) for k, v in args[0].items()}
        if len(args) > 1 and isinstance(args[1], dict):
            self.quests = {int(k): int(v) for k, v in args[1].items()}
        if len(args) > 2 and isinstance(args[2], dict):
            self.bestiary = {int(k): int(v) for k, v in args[2].items()}

    def _handle_update_skill(self, args):
        # UpdateSkill: [skillID, level]
        if not args or not isinstance(args[0], int):
            return
        sid = args[0]
        lvl = args[1] if len(args) > 1 else 0
        self.skills[sid] = int(lvl)
        self._emit({"type": "skill", "skill_id": sid, "level": lvl})

    def _handle_update_quest(self, args):
        # UpdateQuest: [questID, state]
        if not args or not isinstance(args[0], int):
            return
        qid = args[0]
        state = args[1] if len(args) > 1 else 0
        self.quests[qid] = int(state)
        self._emit({"type": "quest", "quest_id": qid, "state": state})

    def _handle_update_bestiary(self, args):
        # UpdateBestiary: [npcRID, total]
        if not args or not isinstance(args[0], int):
            return
        npc_rid = args[0]
        total = args[1] if len(args) > 1 else 0
        self.bestiary[npc_rid] = int(total)

    # -- self ---------------------------------------------------------------

    def _update_self(self, d):
        nick = d.get("nick")
        if nick is not None:
            self.self_nick = nick
        pos = _pos_of(d)
        if pos:
            self.self_pos = pos
        hp = d.get("health") or d.get("hp")
        if hp is not None:
            self.self_hp = hp
        max_hp = d.get("max_health") or d.get("max_hp")
        if max_hp is not None:
            self.self_max_hp = max_hp

    # -- queries ------------------------------------------------------------

    def get_snapshot(self):
        with self.lock:
            return {
                "self": {
                    "peer": self.self_peer,
                    "nick": self.self_nick,
                    "pos": list(self.self_pos),
                    "hp": self.self_hp,
                    "max_hp": self.self_max_hp,
                    "level": self.self_level,
                    "exp": self.self_exp,
                    "gp": self.self_gp,
                    "mana": self.self_mana,
                    "stamina": self.self_stamina,
                    "map_id": self.map_id,
                },
                "players": {str(k): v for k, v in self.players.items()},
                "npcs": {str(k): v for k, v in self.npcs.items()},
                "drops": {str(k): v for k, v in self.drops.items()},
                "inventory": dict(self.inventory),
                "online": list(self.online_names),
                "stats": dict(self.stats),
                "entity_count": len(self.entities),
            }

    def find_nearest_npc(self, max_dist=300, monsters_only=True):
        """Find the nearest NPC/monster to self.

        monsters_only prefers arena monsters (negative charID) over friendly
        NPCs such as Elder/Merchant.
        """
        with self.lock:
            if not self.self_pos:
                return None
            sx, sy = self.self_pos
            best = None
            best_dist = float("inf")
            fallback = None
            fallback_dist = float("inf")
            seen = set()
            for pid, npc in self.npcs.items():
                if id(npc) in seen:
                    continue
                seen.add(id(npc))
                if npc.get("state") == STATE_DEATH:
                    continue
                pos = _pos_of(npc)
                if not pos:
                    continue
                nx, ny = pos
                dist = ((sx - nx) ** 2 + (sy - ny) ** 2) ** 0.5
                if dist > max_dist:
                    continue
                is_monster = not npc.get("friendly", False)
                if monsters_only and not is_monster:
                    if dist < fallback_dist:
                        fallback_dist = dist
                        fallback = (pid, npc)
                    continue
                if dist < best_dist:
                    best_dist = dist
                    best = (pid, npc)
            return best or fallback

    def find_nearest_drop(self, max_dist=150):
        with self.lock:
            if not self.self_pos:
                return None
            sx, sy = self.self_pos
            best = None
            best_dist = float("inf")
            for rid, drop in self.drops.items():
                pos = _pos_of(drop)
                if not pos:
                    continue
                dx, dy = pos
                dist = ((sx - dx) ** 2 + (sy - dy) ** 2) ** 0.5
                if dist < best_dist and dist <= max_dist:
                    best_dist = dist
                    best = (rid, drop)
            return best

    def entity(self, rid):
        with self.lock:
            return self.entities.get(rid)

# ---------------------------------------------------------------------------
# SoM client
# ---------------------------------------------------------------------------

class SoMClient:
    """WebSocket client for the SoM Go server.

    Args:
        url: WebSocket URL (ws://host:port/ws or bare host:port).
        nick: character nickname (used for self-identification).
        auto_combat / auto_loot / auto_potion: auto-play toggles.
        potion_threshold: HP percentage that triggers auto-potion.
        preferred_skill: skill ID used by auto-combat.
    """

    def __init__(self, url, account=None, password=None, nick=None,
                 auto_combat=False, auto_loot=False, auto_potion=False,
                 potion_threshold=30, preferred_skill=SKILL_MELEE):
        self.url = url if "://" in url else "ws://" + url
        self.account = account
        self.password = password
        self.nick = nick
        self.auto_combat = auto_combat
        self.auto_loot = auto_loot
        self.auto_potion = auto_potion
        self.potion_threshold = potion_threshold
        self.preferred_skill = preferred_skill
        self.ws = None
        self.peer_id = None
        self.world = WorldState()
        self.bridge = None
        self.running = False
        self.rpc_lock = asyncio.Lock()
        self.in_game = False
        self.msg_queue = asyncio.Queue()
        self.loop = None
        self.ui_queue = queue.Queue()   # thread-safe events for UI consumers
        self.world.emit = self._on_world_event

    # -- event bridge -------------------------------------------------------

    def _on_world_event(self, event: dict):
        """WorldState event -> UI queue (thread-safe)."""
        try:
            self.ui_queue.put_nowait(event)
        except queue.Full:
            pass

    def poll_events(self, max_events=64):
        """Drain pending UI events (called from the UI thread)."""
        out = []
        for _ in range(max_events):
            try:
                out.append(self.ui_queue.get_nowait())
            except queue.Empty:
                break
        return out

    # -- connection ---------------------------------------------------------

    async def connect(self):
        log.info(f"Connecting to {self.url}...")
        self.ws = await websockets.connect(self.url, max_size=1 << 24)
        raw = await asyncio.wait_for(self.ws.recv(), timeout=10)
        self.peer_id = struct.unpack("<I", raw[:4])[0]
        self.world.self_peer = None
        self.world.self_nick = self.nick
        log.info(f"Connected: peer_id={self.peer_id}, nick={self.nick}")
        await self.ws.send(build_simplify_path(CACHE_ID, NODE_PATH))
        self.running = True
        self.loop = asyncio.get_running_loop()
        self.msg_queue = asyncio.Queue()
        asyncio.create_task(self._recv_loop())

    async def _recv_loop(self):
        while self.running:
            try:
                data = await self.ws.recv()
                cmd, name_id, args = parse_packet(data)
                if cmd == CMD_REMOTE_CALL and name_id is not None:
                    self.world.update_from_push(name_id, args)
                    await self.msg_queue.put((name_id, args))
            except websockets.exceptions.ConnectionClosed:
                log.warning("Connection closed")
                self.running = False
                self._on_world_event({"type": "disconnected"})
                break
            except Exception as e:
                log.debug(f"recv error: {e}")

    # -- RPC helpers --------------------------------------------------------

    async def send_rpc_async(self, name_id, args):
        async with self.rpc_lock:
            await self.ws.send(build_rpc(CACHE_ID, name_id, args))

    def send_rpc(self, name_id, args):
        """Synchronous fire-and-forget RPC (for non-async contexts)."""
        if self.ws and self.running and self.loop and self.loop.is_running():
            asyncio.run_coroutine_threadsafe(self.send_rpc_async(name_id, args), self.loop)

    async def recv_reply(self, skip_ids=None, timeout=10):
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

    # -- login chain --------------------------------------------------------

    async def create_account(self, name, password, email=""):
        log.info(f"Creating account: {name}")
        await self.send_rpc_async(RPC_CREATE_ACCOUNT, [name, password, email, False, 0])
        deadline = time.time() + 10
        while time.time() < deadline:
            name_id, args = await self.recv_reply(skip_ids=PUSH_IDS, timeout=2)
            if name_id == RPC_AUTH_ERROR:
                err = args[0] if args else -1
                if err == AUTH_ERR_OK:
                    log.info(f"Account created: {name}")
                    return True
                log.warning(f"CreateAccount failed: err={err}")
                return False
        return False

    async def login(self, name, password):
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
                if err == AUTH_ERR_AUTH:
                    log.warning("Account not found or wrong password")
                    return False
                log.warning(f"Login failed: err={err}")
                return False
        return False

    async def create_character(self, nick, race=1, gender=0):
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
                log.warning(f"CreateCharacter failed: err={err}")
                return False
        return False

    async def list_characters(self):
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
                break
        log.info(f"Found {len(chars)} character(s)")
        return chars

    async def connect_character(self, nick):
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
                log.warning(f"ConnectCharacter failed: err={err}")
                return False
        return False

    # -- movement -----------------------------------------------------------

    async def move_to(self, x, y):
        """Click-to-move: send SetClickPos and predict locally."""
        log.debug(f"move_to: ({x}, {y}) from {self.world.self_pos}")
        await self.send_rpc_async(RPC_SET_CLICK_POS, [(float(x), float(y))])
        self._cancel_prediction()
        self._predict_task = asyncio.create_task(self._predict_move(x, y))

    async def walk_direction(self, dx, dy):
        """Hold-to-walk: send SetMovePos(direction) and predict locally."""
        await self.send_rpc_async(RPC_SET_MOVE_POS, [(float(dx), float(dy))])
        self._cancel_prediction()
        self._predict_task = asyncio.create_task(self._predict_walk(dx, dy))

    async def stop_walking(self):
        await self.send_rpc_async(RPC_SET_MOVE_POS, [(0.0, 0.0)])
        self._cancel_prediction()

    def _cancel_prediction(self):
        task = getattr(self, "_predict_task", None)
        if task and not task.done():
            task.cancel()

    async def _predict_move(self, tx, ty):
        interval = 0.05
        try:
            while self.running:
                sx, sy = self.world.self_pos
                dx, dy = tx - sx, ty - sy
                dist = (dx * dx + dy * dy) ** 0.5
                if dist < 2.0:
                    self.world.self_pos = (float(tx), float(ty))
                    break
                step = WALK_SPEED * interval
                if step >= dist:
                    self.world.self_pos = (float(tx), float(ty))
                    break
                self.world.self_pos = (sx + dx / dist * step, sy + dy / dist * step)
                self._sync_self_pos()
                await asyncio.sleep(interval)
        except asyncio.CancelledError:
            pass
        except Exception as e:
            log.debug(f"predict error: {e}")

    async def _predict_walk(self, dx, dy):
        interval = 0.05
        try:
            while self.running:
                sx, sy = self.world.self_pos
                self.world.self_pos = (sx + dx * WALK_SPEED * interval,
                                       sy + dy * WALK_SPEED * interval)
                self._sync_self_pos()
                await asyncio.sleep(interval)
        except asyncio.CancelledError:
            pass
        except Exception as e:
            log.debug(f"walk predict error: {e}")

    def _sync_self_pos(self):
        if self.world.self_peer is not None:
            p = self.world.players.get(self.world.self_peer)
            if isinstance(p, dict):
                p["pos"] = list(self.world.self_pos)

    # -- actions ------------------------------------------------------------

    async def attack(self, target_rid, skill_id=None):
        """TriggerSkill(targetRID, skillID) — melee by default."""
        if skill_id is None:
            skill_id = self.preferred_skill
        await self.send_rpc_async(RPC_TRIGGER_SKILL, [int(target_rid), int(skill_id)])

    async def pickup(self, drop_rid):
        """PickupDrop(dropID)."""
        await self.send_rpc_async(RPC_PICKUP_DROP, [int(drop_rid)])

    async def use_item(self, item_id):
        """UseItem(itemID) — e.g. ITEM_POTION."""
        await self.send_rpc_async(RPC_USE_ITEM, [int(item_id)])

    async def interact(self, target_rid):
        """TriggerInteract(targetRID) — talk to an NPC."""
        await self.send_rpc_async(RPC_TRIGGER_INTERACT, [int(target_rid)])

    async def select_target(self, target_rid):
        """TriggerSelect(agentRID) — ask the server for full stats of a target."""
        await self.send_rpc_async(RPC_TRIGGER_SELECT, [int(target_rid)])

    async def respawn(self):
        await self.send_rpc_async(RPC_TRIGGER_RESPAWN, [])

    async def emote(self, emote_id):
        await self.send_rpc_async(RPC_TRIGGER_EMOTE, [int(emote_id)])

    async def chat(self, channel, text):
        """TriggerChat(channelName, text)."""
        await self.send_rpc_async(RPC_TRIGGER_CHAT, [str(channel), str(text)])

    # -- auto-play ----------------------------------------------------------

    async def auto_play_loop(self):
        """Built-in auto-play: auto-combat + auto-loot + auto-potion."""
        if not (self.auto_combat or self.auto_loot or self.auto_potion):
            return
        log.info("Auto-play enabled "
                 f"(combat={self.auto_combat} loot={self.auto_loot} potion={self.auto_potion})")
        while self.running and self.in_game:
            try:
                if self.auto_combat:
                    await self._auto_combat_step()
                if self.auto_loot:
                    await self._auto_loot_step()
                if self.auto_potion:
                    self._auto_potion_step()
            except Exception as e:
                log.debug(f"auto-play error: {e}")
            await asyncio.sleep(0.3)

    async def _auto_combat_step(self):
        target = self.world.find_nearest_npc(max_dist=400)
        if not target:
            return
        pid, npc = target
        pos = _pos_of(npc)
        if not pos:
            return
        sx, sy = self.world.self_pos
        dist = ((sx - pos[0]) ** 2 + (sy - pos[1]) ** 2) ** 0.5
        reach = ATTACK_RANGE_BASE + 16 + 16  # melee-ish reach; ranged skills add more
        if dist > reach:
            await self.move_to(pos[0], pos[1])
            await asyncio.sleep(0.4)
        await self.attack(pid)
        self._on_world_event({"type": "auto_attack", "target": pid,
                              "name": npc.get("name"), "pos": list(pos)})

    async def _auto_loot_step(self):
        drop = self.world.find_nearest_drop(max_dist=150)
        if not drop:
            return
        rid, d = drop
        pos = _pos_of(d)
        if not pos:
            return
        sx, sy = self.world.self_pos
        dist = ((sx - pos[0]) ** 2 + (sy - pos[1]) ** 2) ** 0.5
        if dist > 24:
            await self.move_to(pos[0], pos[1])
            await asyncio.sleep(0.3)
        await self.pickup(rid)
        self.world.stats["loots"] += 1
        self._on_world_event({"type": "auto_loot", "rid": rid, "pos": list(pos)})

    def _auto_potion_step(self):
        if self.world.self_hp <= 0 or self.world.self_max_hp <= 0:
            return
        pct = 100.0 * self.world.self_hp / self.world.self_max_hp
        if pct >= self.potion_threshold:
            return
        if self.world.inventory.get(ITEM_POTION, 0) <= 0:
            return
        self.send_rpc(RPC_USE_ITEM, [ITEM_POTION])
        self._on_world_event({"type": "auto_potion"})

    async def run(self):
        """Keep the client alive; auto-play tasks are started separately."""
        while self.running:
            await asyncio.sleep(1)

    async def close(self):
        self.running = False
        self._cancel_prediction()
        if self.ws:
            try:
                await self.ws.close()
            except Exception:
                pass
