#!/usr/bin/env python3
"""End-to-end Chinese-name test against a live goserver over WebSocket.

Speaks the Godot SceneMultiplayer RPC binary protocol:
  - announces the client Network node via SIMPLIFY_PATH (cache ID 5)
  - CreateAccount / LoginWithPassword with a Chinese account name
  - CreateCharacter / CharacterListing / ConnectCharacter with a Chinese name
  - negative control: an invalid-character name must be rejected

Usage:
    python3 cjk_ws_test.py [ws://127.0.0.1:27012/ws]
"""

import asyncio
import struct
import sys

import websockets

# --- Godot variant types (scene/main/multiplayer_api.cpp) ---
NIL = 0
BOOL = 1
INT = 2
FLOAT = 3
STRING = 4
VECTOR2 = 5
DICTIONARY = 27  # Variant::DICTIONARY
ARRAY = 28  # Variant::ARRAY

# --- SceneMultiplayer command types ---
CMD_REMOTE_CALL = 0
CMD_SIMPLIFY_PATH = 1

# --- RPC name IDs (sorted @rpc list of Network.gd) ---
RPC_AUTH_ERROR = 1
RPC_AUTH_TOKEN_RESULT = 2
RPC_CHARACTER_ERROR = 8
RPC_CHARACTER_INFO = 9
RPC_CHARACTER_LISTING = 10
RPC_CONNECT_CHARACTER = 19
RPC_CREATE_ACCOUNT = 25
RPC_CREATE_CHARACTER = 26
RPC_LOGIN_WITH_PASSWORD = 45

# --- Error codes ---
AUTH_ERR_OK = 0
AUTH_ERR_NAME_VALID = 9
AUTH_ERR_NAME_SIZE = 10
CHAR_ERR_OK = 0
CHAR_ERR_NAME_VALID = 8
CHAR_ERR_NAME_SIZE = 9

CACHE_ID = 5
NODE_PATH = "/root/Network"

# Server-initiated pushes to ignore while waiting for a reply.
# Mirrors serverPushNameIDs in login_test.go.
PUSH_IDS = {
    0,    # AddOnlinePlayer
    2,    # AuthTokenResult
    32,   # DropAdded
    34,   # DropRemoved
    39,   # FullUpdateEntity
    41,   # ItemAdded
    43,   # ItemRemoved
    44,   # LevelUp
    51,   # PartyInvite
    52,   # PartyUpdate
    54,   # PreloadEntity
    55,   # PreloadPlayer
    56,   # PushNotification
    57,   # RefreshEquipment
    58,   # RefreshInventory
    59,   # RefreshOnlineList
    60,   # RefreshProgress
    61,   # RemoveEntity
    62,   # RemoveOnlinePlayer
    82,   # TradeCancel
    83,   # TradeClosed
    84,   # TradeLock
    85,   # TradeOpened
    86,   # TradeRequested
    87,   # TradeRespond
    88,   # TradeSetOffer
    89,   # TradeUpdate
    109,  # UpdateAttributes
    110,  # UpdateBestiary
    112,  # UpdateEntity
    113,  # UpdatePrivateStats
    114,  # UpdatePublicStats
    115,  # UpdateQuest
    116,  # UpdateSkill
    118,  # WarpPlayer
    16,   # CommandFeedback
    20,   # ContextChoice
    21,   # ContextClose
    22,   # ContextContinue
    23,   # ContextText
    24,   # ContextThink
    91,   # TriggerChoice
    92,   # TriggerCloseContext
    93,   # TriggerCommand
    95,   # TriggerExplore
    96,   # TriggerInteract
    97,   # TriggerNextContext
    105,  # TriggerSelect
    6,    # Casted
    11,   # ChatPlayer
    12,   # ChatQuery
    13,   # ChatSystem
    35,   # Emote
    90,   # TriggerChat
    94,   # TriggerEmote
    79,   # TargetAlteration
    80,   # ThrowProjectile
    104,  # TriggerRespawn
    107,  # TriggerSkill
}


# --------------------------------------------------------------------------
# Variant encoding (encode_and_compress_variants)
# --------------------------------------------------------------------------

def enc_int(v: int) -> bytes:
    """Compressed int encoding (BOOL/INT share the compressed form)."""
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
    """Full encode_variant header (4 bytes, low byte = bare type) + payload.

    encodeAndCompressVariant overwrites byte 0 of the encode_variant output
    with the bare type so the decoder can tell compressed from uncompressed.
    """
    return struct.pack("<I", vtype) + payload


def enc_variant(v) -> bytes:
    """encode_and_compress_variant: BOOL/INT compressed, rest uncompressed."""
    if isinstance(v, bool):
        return enc_bool(v)
    if isinstance(v, int):
        return enc_int(v)
    if isinstance(v, str):
        return enc_uncompressed(STRING, enc_string(v))
    if isinstance(v, dict):
        # Dictionary elements use the uncompressed encode_variant format.
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
    """encode_variant (uncompressed) — used for dict/array elements."""
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


# --------------------------------------------------------------------------
# Packet building / parsing
# --------------------------------------------------------------------------

def build_rpc(cache_id: int, name_id: int, args: list) -> bytes:
    payload = enc_args(args)
    meta = CMD_REMOTE_CALL
    # node ID compression: 8-bit (cache_id <= 255)
    if cache_id <= 0xFF:
        nid = bytes([cache_id])
    elif cache_id <= 0xFFFF:
        meta |= 1 << 4
        nid = struct.pack("<H", cache_id)
    else:
        meta |= 2 << 4
        nid = struct.pack("<I", cache_id)
    # name ID compression: 8-bit (<= 255)
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
    md5 = b"\x00" * 32 + b"\x00"  # 32-byte MD5 + NUL
    out += md5 + struct.pack("<I", cache_id)
    out += path.encode("utf-8") + b"\x00"
    return out


def parse_packet(buf: bytes):
    """Returns (cmd, name_id, args) or (cmd, None, None) for non-RPC packets."""
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
        # Unhandled variant type in a server push we don't care about.
        args = []
    return cmd, name_id, args


# --------------------------------------------------------------------------
# Variant decoding
# --------------------------------------------------------------------------

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
        # Full encode_variant header (4 bytes) then string payload.
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
    """Decode an uncompressed encode_variant (used for dict/array elements)."""
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


# --------------------------------------------------------------------------
# Test flow
# --------------------------------------------------------------------------

async def recv_reply(ws, skip_ids):
    """Read packets until a non-push RPC reply arrives."""
    while True:
        data = await asyncio.wait_for(ws.recv(), timeout=5)
        cmd, name_id, args = parse_packet(data)
        if cmd != CMD_REMOTE_CALL or name_id is None:
            continue
        if name_id in skip_ids:
            continue
        return name_id, args


async def recv_n(ws, n, collect_ids=None):
    collect_ids = collect_ids or set()
    out = []
    while len(out) < n:
        data = await asyncio.wait_for(ws.recv(), timeout=5)
        cmd, name_id, args = parse_packet(data)
        if cmd != CMD_REMOTE_CALL or name_id is None:
            continue
        if name_id in PUSH_IDS and name_id not in collect_ids:
            continue
        out.append((name_id, args))
    return out


async def main():
    url = sys.argv[1] if len(sys.argv) > 1 else "ws://127.0.0.1:27012/ws"
    account_name = "玩家测试"
    char_name = "张三"
    password = "密码1234"
    invalid_name = "玩家@测试"

    async with websockets.connect(url, max_size=1 << 20) as ws:
        # Server pushes the 4-byte LE unique_id on connect.
        raw = await asyncio.wait_for(ws.recv(), timeout=5)
        peer_id = struct.unpack("<I", raw[:4])[0]
        print(f"connected: peer_id={peer_id}")

        await ws.send(build_simplify_path(CACHE_ID, NODE_PATH))

        # 1. CreateAccount -> AuthTokenResult + AuthError(OK)
        await ws.send(build_rpc(CACHE_ID, RPC_CREATE_ACCOUNT,
                                [account_name, password, "zh@test.com", True, 0]))
        replies = await recv_n(ws, 2, collect_ids={RPC_AUTH_TOKEN_RESULT})
        tok_id, tok_args = replies[0]
        auth_id, auth_args = replies[1]
        assert tok_id == RPC_AUTH_TOKEN_RESULT, f"want AuthTokenResult, got {tok_id}"
        assert tok_args[0] == account_name, f"token name mismatch: {tok_args[0]!r}"
        assert auth_id == RPC_AUTH_ERROR and auth_args[0] == AUTH_ERR_OK, \
            f"CreateAccount failed: id={auth_id} err={auth_args}"
        print(f"PASS: CreateAccount({account_name!r}) -> AuthError(OK)")

        # 2. LoginWithPassword -> AuthError(OK)
        await ws.send(build_rpc(CACHE_ID, RPC_LOGIN_WITH_PASSWORD,
                                [account_name, password, False, 0]))
        name_id, args = await recv_reply(ws, PUSH_IDS)
        assert name_id == RPC_AUTH_ERROR and args[0] == AUTH_ERR_OK, \
            f"Login failed: id={name_id} err={args}"
        print(f"PASS: LoginWithPassword({account_name!r}) -> AuthError(OK)")

        # 3. CreateCharacter -> CharacterInfo + CharacterError(OK)
        await ws.send(build_rpc(CACHE_ID, RPC_CREATE_CHARACTER,
                                [char_name, {"race": 1, "gender": 0}, {}]))
        replies = await recv_n(ws, 2, collect_ids={RPC_CHARACTER_INFO})
        info_id, info_args = replies[0]
        err_id, err_args = replies[1]
        assert info_id == RPC_CHARACTER_INFO, f"want CharacterInfo, got {info_id}"
        assert info_args[0]["nickname"] == char_name, \
            f"nickname mismatch: {info_args[0].get('nickname')!r}"
        assert err_id == RPC_CHARACTER_ERROR and err_args[0] == CHAR_ERR_OK, \
            f"CreateCharacter failed: id={err_id} err={err_args}"
        print(f"PASS: CreateCharacter({char_name!r}) -> CharacterError(OK)")

        # 4. CharacterListing -> CharacterInfo(nickname) + CharacterError(OK)
        await ws.send(build_rpc(CACHE_ID, RPC_CHARACTER_LISTING, []))
        replies = await recv_n(ws, 2, collect_ids={RPC_CHARACTER_INFO})
        info_id, info_args = replies[0]
        err_id, err_args = replies[1]
        assert info_id == RPC_CHARACTER_INFO and info_args[0]["nickname"] == char_name
        assert err_id == RPC_CHARACTER_ERROR and err_args[0] == CHAR_ERR_OK
        print(f"PASS: CharacterListing -> nickname={char_name!r}")

        # 5. ConnectCharacter -> CharacterError(OK)
        await ws.send(build_rpc(CACHE_ID, RPC_CONNECT_CHARACTER, [char_name]))
        name_id, args = await recv_reply(ws, PUSH_IDS)
        assert name_id == RPC_CHARACTER_ERROR and args[0] == CHAR_ERR_OK, \
            f"ConnectCharacter failed: id={name_id} err={args}"
        print(f"PASS: ConnectCharacter({char_name!r}) -> CharacterError(OK)")

        # 6. Negative control: invalid characters must be rejected
        await ws.send(build_rpc(CACHE_ID, RPC_CREATE_CHARACTER,
                                [invalid_name, {"race": 1, "gender": 0}, {}]))
        replies = await recv_n(ws, 2, collect_ids={RPC_CHARACTER_INFO})
        err_id, err_args = replies[-1]
        assert err_id == RPC_CHARACTER_ERROR and err_args[0] != CHAR_ERR_OK, \
            f"invalid name unexpectedly accepted: err={err_args}"
        print(f"PASS: CreateCharacter({invalid_name!r}) rejected with err={err_args[0]}")

    print("\nALL CHINESE-NAME TESTS PASSED")


if __name__ == "__main__":
    asyncio.run(main())
