#!/usr/bin/env python3
"""SoM Headless AI Client — lightweight Python client for AI agents.

Speaks the Godot SceneMultiplayer WebSocket protocol directly (no Godot
runtime, ~15 MB RSS vs ~400 MB for a headless Godot instance). The protocol
layer lives in som_protocol.py and is shared with the human lite client
(lite_human_client.py).

Features:
  - Full login chain: CreateAccount / LoginWithPassword / CreateCharacter /
    CharacterListing / ConnectCharacter
  - In-game loop: receives world pushes (PreloadEntity, UpdateEntity, ...)
  - AgentBridge NDJSON server: external AI can connect via TCP and control
    the character (list_players, snapshot, apply_action, drain_events, ...)
  - Built-in auto-play: auto-combat, auto-loot, auto-potion (optional)

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

import argparse
import asyncio
import json
import logging
import os
import random
import signal
import socket
import string
import sys
import threading
import time

from som_protocol import (
    ITEM_POTION,
    RPC_PICKUP_DROP,
    RPC_TRIGGER_CHAT,
    RPC_TRIGGER_EMOTE,
    RPC_TRIGGER_INTERACT,
    RPC_TRIGGER_RESPAWN,
    RPC_TRIGGER_SKILL,
    SKILL_MELEE,
    SoMClient,
)

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%H:%M:%S",
)
log = logging.getLogger("ai-client")

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
        self.events = []
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
            return {"ok": False, "reason": "use the Go server bridge (port 9101)"}
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
            loop = self.client.loop
            if loop and loop.is_running():
                fut = asyncio.run_coroutine_threadsafe(
                    self.client.move_to(float(pos[0]), float(pos[1])), loop)

                def _log_exc(f):
                    try:
                        f.result(timeout=5)
                    except Exception as e:
                        log.debug(f"move_to coroutine error: {e}")
                fut.add_done_callback(_log_exc)
            return {"ok": True, "verdict": "APPLIED", "action": action}
        elif action == "attack":
            # TriggerSkill(targetRID, skillID). Accept either rid + skill
            # (harness format) or the legacy skill_id-only form, in which
            # case the nearest monster is targeted.
            rid = args.get("rid", args.get("target", args.get("target_rid")))
            skill_id = args.get("skill", args.get("skill_id", SKILL_MELEE))
            if isinstance(skill_id, str):
                skill_map = {"Melee": SKILL_MELEE, "melee": SKILL_MELEE}
                skill_id = skill_map.get(skill_id, SKILL_MELEE)
            if rid is None:
                target = self.client.world.find_nearest_npc(max_dist=400)
                if not target:
                    return {"error": "no target in range"}
                rid = target[0]
            self.client.send_rpc(RPC_TRIGGER_SKILL, [int(rid), int(skill_id)])
            return {"ok": True, "verdict": "APPLIED", "action": action,
                    "rid": int(rid), "skill": int(skill_id)}
        elif action == "pickup":
            drop_rid = args.get("drop_rid", args.get("dropRID", args.get("rid")))
            if drop_rid is None:
                drop = self.client.world.find_nearest_drop(max_dist=150)
                if not drop:
                    return {"error": "no drop in range"}
                drop_rid = drop[0]
            self.client.send_rpc(RPC_PICKUP_DROP, [int(drop_rid)])
            return {"ok": True, "verdict": "APPLIED", "action": action}
        elif action == "chat":
            msg = args.get("message", args.get("text", ""))
            channel = args.get("channel", "local")
            self.client.send_rpc(RPC_TRIGGER_CHAT, [str(channel), str(msg)])
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
# CLI
# ---------------------------------------------------------------------------

def parse_args():
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
    parser.add_argument("--no-keep", dest="keep", action="store_false",
                        help="Exit after login")
    parser.set_defaults(keep=True)
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
        auto_combat=args.auto_play,
        auto_loot=args.auto_play,
    )

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
            client.nick = nick
            chars = await client.list_characters()

        if not chars:
            log.error("No characters available. Use --auto-create to create one.")
            return

        # Connect to first character (or match by nick)
        nick = args.nick or chars[0].get("nickname")
        client.nick = nick
        if not await client.connect_character(nick):
            log.error("Failed to connect character")
            return

        # Start AgentBridge
        client.bridge = AgentBridgeServer(client, port=args.bridge_port)
        client.bridge.start()

        # Start auto-play
        if args.auto_play:
            asyncio.create_task(client.auto_play_loop())

        # Run
        if args.keep:
            log.info("Running... (Ctrl+C to stop)")
            while client.running:
                await asyncio.sleep(1)
                snap = client.world.get_snapshot()
                if snap["self"]["nick"]:
                    log.info(f"  HP: {snap['self']['hp']}/{snap['self']['max_hp']}  "
                             f"Pos: {snap['self']['pos']}  "
                             f"Players: {len(snap['players'])}  NPCs: {len(snap['npcs'])}  "
                             f"Drops: {len(snap['drops'])}")
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
