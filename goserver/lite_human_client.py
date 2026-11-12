#!/usr/bin/env python3
"""SoM Lite Human Client — tkinter GUI client for Source of Mana.

The AI headless client (headless_ai_client.py) needs no graphics; this is the
human counterpart. It reuses the exact same protocol layer (som_protocol.py)
and adds a simple screen system modelled on classic MMORPG "脱机挂" (offline
bot assistants):

  - Viewport: camera centered on the character, 100 px grid, zoom, click-to-move,
    click-a-monster-to-attack, HP bars, damage floaters, move marker.
  - Status panel: HP/MP bars, level/exp/GP/mana/stamina, position, map.
  - Equipment bar: the 8 equipment slots plus the inventory, with 装备 / 使用 /
    卸下 actions, so the starter Short Sword can be equipped before fighting.
  - Dialogue window: NPC ContextText / ContextChoice / ContextContinue /
    ContextClose, so the normal quest flow (talk to the Elder) works.
  - Target panel: selected target with 攻击 / 拾取 / 交互 / 复活 / 喝药 buttons.
  - Auto-play panel: 自动打怪 / 自动拾取 / 自动喝药 + potion threshold.
  - Stats panel: kills / loots / deaths / damage dealt.
  - Minimap, log window, chat input, screenshot button.
  - Optional AI arena: a toolbar button that runs /arena start|status|stop on
    the server. Off by default; the client boots straight into the normal game
    world, and the arena never starts unless you ask for it.

The asyncio protocol loop runs in a background thread; the tkinter UI runs on
the main thread and polls a thread-safe event queue (client.poll_events()).

Requirements:
    Python 3.8+, websockets, tkinter (Debian/Ubuntu: python3-tk, Alpine: tk).

Usage:
    python3 lite_human_client.py [ws://host:port/ws] [options]

    # Quick start (auto-create account + character)
    python3 lite_human_client.py ws://127.0.0.1:27012/ws --auto-create --nick MyHero

    # Existing account
    python3 lite_human_client.py ws://127.0.0.1:27012/ws --account me --password secret

    # Start with auto-play enabled
    python3 lite_human_client.py ws://127.0.0.1:27012/ws --auto-create --auto-play

Environment variables:
    SOM_WS_URL      WebSocket URL (default: ws://127.0.0.1:27012/ws)
    SOM_ACCOUNT     Account name
    SOM_PASSWORD    Account password
    SOM_NICK        Character nickname
    SOM_AUTO_CREATE Auto-create account/character (0/1, default: 0)
    SOM_AUTO_PLAY   Enable auto-play at startup (0/1, default: 0)
"""

import argparse
import asyncio
import logging
import math
import os
import queue
import re
import sys
import threading
import time

try:
    import tkinter as tk
    from tkinter import filedialog, messagebox
except ImportError:
    # Kept non-fatal at import time so the protocol/bridge helpers can be
    # imported (and --help shown) on headless boxes. main() checks again.
    tk = None
    filedialog = None
    messagebox = None

from som_protocol import (
    EQUIP_SLOTS,
    ITEM_POTION,
    QUEST_STATE_DONE,
    QUEST_STATE_INACTIVE,
    QuestCatalog,
    SKILL_MELEE,
    SoMClient,
    item_name,
    item_slot,
)

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%H:%M:%S",
)
log = logging.getLogger("lite-human")

# ---------------------------------------------------------------------------
# Palette
# ---------------------------------------------------------------------------

BG = "#0d1117"
PANEL = "#111722"
BORDER = "#263241"
GRID = "#1b2430"
GRID_MAJOR = "#2b3a4d"
TEXT = "#e6edf3"
DIM = "#8b9aab"
SELF = "#4aa3ff"
PLAYER = "#3fbf6f"
FRIENDLY = "#e0a83c"
MONSTER = "#d9534f"
DROP = "#f2d13d"
HP_BG = "#3a1113"
HP_OK = "#3fbf6f"
HP_WARN = "#e0a83c"
HP_LOW = "#d9534f"
BUTTON_BG = "#1c2635"
BUTTON_FG = "#e6edf3"
ACCENT = "#4aa3ff"


def _hex_to_rgb(c):
    c = c.lstrip("#")
    return tuple(int(c[i:i + 2], 16) for i in (0, 2, 4))


def _lerp_color(c1, c2, t):
    a, b = _hex_to_rgb(c1), _hex_to_rgb(c2)
    t = max(0.0, min(1.0, t))
    return "#%02x%02x%02x" % tuple(int(a[i] + (b[i] - a[i]) * t) for i in range(3))


# ---------------------------------------------------------------------------
# GUI log handler
# ---------------------------------------------------------------------------

class GuiLogHandler(logging.Handler):
    """Forward log records to the client's UI event queue."""

    def __init__(self, client_getter):
        super().__init__()
        self.client_getter = client_getter
        self.setFormatter(logging.Formatter(
            "%(asctime)s %(levelname)s %(message)s", datefmt="%H:%M:%S"))

    def emit(self, record):
        client = self.client_getter()
        if client is None:
            return
        try:
            client._on_world_event({
                "type": "log",
                "level": record.levelname,
                "message": self.format(record),
            })
        except Exception:
            pass


# ---------------------------------------------------------------------------
# Client runner — owns the asyncio loop in a background thread
# ---------------------------------------------------------------------------

# Arena templates known to the Go server (arena.go). The client only offers
# these; the server rejects anything else with UNKNOWN_TEMPLATE + "known".
ARENA_TEMPLATES = (
    ("tutorial_maggot", "教学：Maggot 3 杀 / 60 秒"),
    ("grind_maggot", "刷怪：Maggot 5 杀 / 120 秒"),
)


class ClientRunner:
    """Runs the SoMClient asyncio loop off the UI thread.

    The UI submits commands ("connect", "login", "move_to", ...) through a
    thread-safe queue; the runner executes them on the asyncio loop and posts
    status/world events back through client.ui_queue.
    """

    def __init__(self):
        self.client = None
        self.cmd_queue = queue.Queue()
        self.thread = None
        self.loop = None
        self.auto_task = None
        self._stop = False

    def start(self):
        self.thread = threading.Thread(target=self._thread_main,
                                       name="som-client", daemon=True)
        self.thread.start()

    def submit(self, cmd, **kwargs):
        self.cmd_queue.put((cmd, kwargs))

    def _thread_main(self):
        asyncio.run(self._amain())

    async def _amain(self):
        self.loop = asyncio.get_running_loop()
        while not self._stop:
            # Block on a worker thread so the asyncio loop stays free for the
            # client's recv loop and pending RPCs.
            item = await self.loop.run_in_executor(None, self._queue_get)
            if item is None:
                continue
            cmd, kwargs = item
            try:
                await self._dispatch(cmd, kwargs)
            except Exception as e:
                log.debug(f"runner command {cmd!r} failed: {e}", exc_info=True)
                if self.client is not None:
                    self.client._on_world_event(
                        {"type": "error", "message": str(e), "cmd": cmd})

    def _queue_get(self):
        try:
            return self.cmd_queue.get(timeout=0.2)
        except queue.Empty:
            return None

    async def _dispatch(self, cmd, kwargs):
        c = self.client

        if cmd == "connect":
            self.client = SoMClient(url=kwargs["url"], nick=kwargs.get("nick"))
            self.client.loop = self.loop
            await self.client.connect()
            self.auto_task = None
            self.client._on_world_event({"type": "connected",
                                         "peer": self.client.peer_id})
            return

        if c is None:
            c = self.client
        if c is None:
            return

        if cmd == "login":
            ok = await c.login(kwargs["account"], kwargs["password"])
            c._on_world_event({"type": "login_result", "ok": ok})
            if ok:
                await self._list_chars()
            return

        if cmd == "create_account":
            ok = await c.create_account(kwargs["account"], kwargs["password"])
            c._on_world_event({"type": "create_account_result", "ok": ok,
                               "account": kwargs["account"]})
            if ok:
                ok2 = await c.login(kwargs["account"], kwargs["password"])
                c._on_world_event({"type": "login_result", "ok": ok2})
                if ok2:
                    await self._list_chars()
            return

        if cmd == "create_character":
            ok = await c.create_character(kwargs["nick"])
            c._on_world_event({"type": "create_character_result", "ok": ok,
                               "nick": kwargs["nick"]})
            return

        if cmd == "connect_character":
            nick = kwargs["nick"]
            c.nick = nick
            c.world.self_nick = nick
            ok = await c.connect_character(nick)
            c._on_world_event({"type": "in_game", "nick": nick, "ok": ok})
            return

        if cmd == "disconnect":
            await c.close()
            c._on_world_event({"type": "disconnected"})
            return

        if cmd == "autoplay":
            c.auto_combat = bool(kwargs.get("combat"))
            c.auto_loot = bool(kwargs.get("loot"))
            c.auto_potion = bool(kwargs.get("potion"))
            try:
                c.potion_threshold = int(kwargs.get("threshold", 30))
            except (TypeError, ValueError):
                pass
            if c.running and c.in_game:
                if self.auto_task is None or self.auto_task.done():
                    self.auto_task = asyncio.create_task(c.auto_play_loop())
            c._on_world_event({
                "type": "autoplay",
                "combat": c.auto_combat,
                "loot": c.auto_loot,
                "potion": c.auto_potion,
                "threshold": c.potion_threshold,
            })
            return

        if cmd == "move_to":
            await c.move_to(float(kwargs["x"]), float(kwargs["y"]))
            return
        if cmd == "walk":
            await c.walk_direction(float(kwargs["dx"]), float(kwargs["dy"]))
            return
        if cmd == "stop_walk":
            await c.stop_walking()
            return
        if cmd == "attack":
            await c.attack(int(kwargs["rid"]))
            return
        if cmd == "pickup":
            await c.pickup(int(kwargs["rid"]))
            return
        if cmd == "use_item":
            await c.use_item(int(kwargs["item_id"]))
            return
        if cmd == "equip":
            await c.equip_item(int(kwargs["item_id"]),
                               item_index=int(kwargs.get("item_index", 0)))
            return
        if cmd == "unequip":
            await c.unequip_item(int(kwargs["item_id"]))
            return
        if cmd == "dialog_choice":
            await c.trigger_choice(int(kwargs["choice"]))
            return
        if cmd == "dialog_next":
            await c.trigger_next_context()
            return
        if cmd == "dialog_close":
            await c.trigger_close_context()
            return
        if cmd == "interact":
            await c.interact(int(kwargs["rid"]))
            return
        if cmd == "select":
            await c.select_target(int(kwargs["rid"]))
            return
        if cmd == "respawn":
            await c.respawn()
            return
        if cmd == "chat":
            await c.chat(kwargs.get("channel", "local"), kwargs.get("text", ""))
            return
        if cmd == "emote":
            await c.emote(int(kwargs.get("emote_id", 0)))
            return
        if cmd == "command":
            await c.command(kwargs.get("text", ""))
            return

        log.warning(f"unknown runner command: {cmd}")

    async def _list_chars(self):
        chars = await self.client.list_characters()
        self.client._on_world_event({"type": "characters", "chars": chars})

    def stop(self):
        self._stop = True
        try:
            self.submit("disconnect")
        except Exception:
            pass


# ---------------------------------------------------------------------------
# Application
# ---------------------------------------------------------------------------

class Application:
    def __init__(self, args):
        self.args = args
        self.root = tk.Tk()
        self.root.title("SoM 精简客户端")
        self.root.geometry("1220x780")
        self.root.minsize(900, 600)
        self.root.configure(bg=BG)

        self.runner = ClientRunner()
        self.runner.start()
        self.client = None
        self.log_handler = GuiLogHandler(lambda: self.client)
        logging.getLogger().addHandler(self.log_handler)
        logging.getLogger("som-protocol").addHandler(self.log_handler)

        # viewport state
        self.cam = [300.0, 160.0]
        self.zoom = 1.0
        self.view_w = 800
        self.view_h = 600
        self.effects = []
        self.target_kind = None
        self.target_id = None
        self.keys_pressed = set()
        self.log_lines = 0

        # equipment / inventory panel state
        self._eq_sig = None
        self._inv_items = []

        # dialogue window state
        self.dlg_window = None
        self.dlg_can_continue = False

        # quest catalog / quest log state
        qpath = self.args.quests or _default_quests_path()
        self.quests = QuestCatalog(qpath if os.path.exists(qpath) else None)
        if self.quests.loadable:
            log.info("quest catalog: %d quests from %s",
                     len(self.quests.quests), self.quests.path)
        else:
            log.warning("quest catalog: not loaded (looked at %s); "
                        "quest log and NPC markers are disabled", qpath)
        self.quest_log_window = None
        self.quest_log_text = None
        self.quest_log_filter = None
        self._quest_tracker_sig = None
        self._quest_active = []
        self.cam_hold = None

        # optional AI arena state
        self.arena_window = None
        self.arena_episode = None
        self.arena_status_text = "未开启"

        self._show_login()
        self._main_ready = False
        self.root.after(50, self._tick)
        self.root.protocol("WM_DELETE_WINDOW", self._on_close)

    # -- login window -------------------------------------------------------

    def _show_login(self):
        f = tk.Frame(self.root, bg=BG)
        f.place(relx=0.5, rely=0.5, anchor="center")
        self.login_frame = f

        tk.Label(f, text="SoM 精简客户端", font=("TkDefaultFont", 20, "bold"),
                 fg=ACCENT, bg=BG).grid(row=0, column=0, columnspan=2, pady=(0, 4))
        tk.Label(f, text="人类玩家精简客户端 · 参考脱机挂的简单画面系统",
                 font=("TkDefaultFont", 9), fg=DIM, bg=BG).grid(
                     row=1, column=0, columnspan=2, pady=(0, 16))

        self.var_url = tk.StringVar(value=self.args.url)
        self.var_account = tk.StringVar(value=self.args.account or "")
        self.var_password = tk.StringVar(value=self.args.password or "")
        self.var_nick = tk.StringVar(value=self.args.nick or "")
        self.var_auto = tk.BooleanVar(value=self.args.auto_create)

        self._login_row(f, 2, "服务器", self.var_url)
        self._login_row(f, 3, "账号", self.var_account)
        self._login_row(f, 4, "密码", self.var_password, show="*")
        self._login_row(f, 5, "昵称", self.var_nick)

        tk.Checkbutton(f, text="自动注册账号 / 创建角色", variable=self.var_auto,
                       fg=TEXT, bg=BG, selectcolor=PANEL,
                       activebackground=BG, activeforeground=TEXT).grid(
                           row=6, column=0, columnspan=2, pady=(10, 6), sticky="w")

        self.btn_connect = tk.Button(f, text="连接", width=14, command=self._do_connect,
                                     bg=ACCENT, fg="#06131f",
                                     activebackground="#6fb6ff", activeforeground="#06131f",
                                     relief="flat", font=("TkDefaultFont", 10, "bold"))
        self.btn_connect.grid(row=7, column=0, columnspan=2, pady=(6, 10))

        tk.Label(f, text="默认进入正常游戏世界；AI 竞技场可在主界面【AI竞技场】按钮中手动开启",
                 font=("TkDefaultFont", 8), fg=DIM, bg=BG, wraplength=360,
                 justify="center").grid(row=9, column=0, columnspan=2, pady=(0, 4))

        self.login_status = tk.StringVar(value="")
        tk.Label(f, textvariable=self.login_status, font=("TkDefaultFont", 9),
                 fg=DIM, bg=BG, wraplength=360, justify="center").grid(
                     row=8, column=0, columnspan=2)

        self._pending = None

    def _login_row(self, parent, row, label, var, show=None):
        tk.Label(parent, text=label, width=8, anchor="e", fg=DIM, bg=BG,
                 font=("TkDefaultFont", 10)).grid(row=row, column=0, sticky="e",
                                                  padx=(0, 8), pady=4)
        e = tk.Entry(parent, textvariable=var, width=34, show=show,
                     bg=PANEL, fg=TEXT, insertbackground=TEXT,
                     relief="flat", highlightthickness=1,
                     highlightbackground=BORDER, highlightcolor=ACCENT)
        e.grid(row=row, column=1, sticky="w", pady=4)

    def _login_status(self, text, kind="info"):
        self.login_status.set(text)
        color = {"ok": HP_OK, "error": HP_LOW, "info": DIM}.get(kind, DIM)
        # Label color is fixed; the text itself carries the meaning.
        log.info(text)

    def _do_connect(self):
        account = self.var_account.get().strip()
        password = self.var_password.get().strip()
        nick = self.var_nick.get().strip() or account
        if not account or not password:
            self._login_status("请填写账号和密码", "error")
            return
        self._pending = {
            "account": account,
            "password": password,
            "nick": nick,
            "auto_create": bool(self.var_auto.get()),
        }
        self.btn_connect.configure(state="disabled")
        self._login_status("连接服务器...")
        self.runner.submit("connect", url=self.var_url.get().strip(), nick=nick)

    # -- main window --------------------------------------------------------

    def _show_main(self):
        self.login_frame.destroy()
        self._main_ready = True
        self._build_ui()
        if getattr(self.args, "auto_play", False):
            self.var_combat.set(True)
            self.var_loot.set(True)
            self.var_potion.set(True)
        self._apply_autoplay(initial=True)

    def _build_ui(self):
        root = self.root
        # toolbar
        top = tk.Frame(root, bg=PANEL, height=38)
        top.pack(side="top", fill="x")
        top.pack_propagate(False)
        tk.Label(top, text=" SoM 精简客户端", fg=ACCENT, bg=PANEL,
                 font=("TkDefaultFont", 11, "bold")).pack(side="left", padx=(10, 6))
        self.conn_status = tk.StringVar(value="已连接")
        tk.Label(top, textvariable=self.conn_status, fg=DIM, bg=PANEL,
                 font=("TkDefaultFont", 9)).pack(side="left", padx=6)
        tk.Button(top, text="−", width=3, command=lambda: self._set_zoom(self.zoom / 1.2),
                  bg=BUTTON_BG, fg=BUTTON_FG, relief="flat").pack(side="right", padx=2)
        tk.Button(top, text="+", width=3, command=lambda: self._set_zoom(self.zoom * 1.2),
                  bg=BUTTON_BG, fg=BUTTON_FG, relief="flat").pack(side="right", padx=2)
        self.zoom_label = tk.StringVar(value="100%")
        tk.Label(top, textvariable=self.zoom_label, fg=DIM, bg=PANEL,
                 font=("TkDefaultFont", 9), width=6).pack(side="right", padx=4)
        tk.Button(top, text="截图", command=self._screenshot,
                  bg=BUTTON_BG, fg=BUTTON_FG, relief="flat").pack(side="right", padx=4)
        tk.Button(top, text="断开", command=self._disconnect,
                  bg=BUTTON_BG, fg=BUTTON_FG, relief="flat").pack(side="right", padx=4)
        self.arena_label = tk.StringVar(value="AI竞技场: 未开启")
        tk.Button(top, textvariable=self.arena_label, command=self._open_arena,
                  bg=BUTTON_BG, fg=BUTTON_FG, relief="flat",
                  font=("TkDefaultFont", 9)).pack(side="right", padx=4)

        # middle
        middle = tk.Frame(root, bg=BG)
        middle.pack(side="top", fill="both", expand=True)

        self.canvas = tk.Canvas(middle, bg=BG, highlightthickness=0)
        self.canvas.pack(side="left", fill="both", expand=True)
        self.canvas.bind("<Configure>", self._on_canvas_resize)
        self.canvas.bind("<Button-1>", self._on_canvas_click)
        self.canvas.bind("<Button-3>", self._on_canvas_right_click)
        self.canvas.bind("<MouseWheel>", self._on_mouse_wheel)
        self.canvas.bind("<Button-4>", self._on_mouse_wheel)
        self.canvas.bind("<Button-5>", self._on_mouse_wheel)
        self.canvas.bind("<Motion>", self._on_canvas_motion)

        side = tk.Frame(middle, bg=PANEL, width=270)
        side.pack(side="right", fill="y")
        side.pack_propagate(False)
        # The panel holds more than one screenful (status, equipment, target,
        # auto-play, stats, minimap), so it scrolls vertically.
        side_canvas = tk.Canvas(side, bg=PANEL, highlightthickness=0)
        side_canvas.pack(side="left", fill="both", expand=True)
        side_scroll = tk.Scrollbar(side, orient="vertical",
                                   command=side_canvas.yview,
                                   bg=BUTTON_BG, troughcolor=PANEL,
                                   relief="flat", width=10)
        side_scroll.pack(side="right", fill="y")
        side_canvas.configure(yscrollcommand=side_scroll.set)
        side_inner = tk.Frame(side_canvas, bg=PANEL)
        side_win = side_canvas.create_window((0, 0), window=side_inner,
                                             anchor="nw")

        def _side_configure(event):
            side_canvas.configure(scrollregion=side_canvas.bbox("all"))
            side_canvas.itemconfigure(side_win, width=event.width)

        side_inner.bind("<Configure>", _side_configure)

        def _side_wheel(event):
            up = getattr(event, "num", 0) == 4 or getattr(event, "delta", 0) > 0
            side_canvas.yview_scroll(-1 if up else 1, "units")

        self._build_side_panel(side_inner)
        # Bind the wheel on every descendant: tkinter does not propagate
        # events from child widgets to their parents.
        widgets = [side_inner]
        while widgets:
            w = widgets.pop()
            for seq in ("<MouseWheel>", "<Button-4>", "<Button-5>"):
                w.bind(seq, _side_wheel)
            widgets.extend(w.winfo_children())

        # bottom: log + chat
        bottom = tk.Frame(root, bg=PANEL, height=150)
        bottom.pack(side="bottom", fill="x")
        bottom.pack_propagate(False)

        self.log_text = tk.Text(bottom, bg="#0a0e14", fg=TEXT, height=8,
                                wrap="word", relief="flat", state="disabled",
                                font=("TkFixedFont", 9))
        self.log_text.pack(side="top", fill="both", expand=True, padx=(6, 0), pady=(6, 0))
        for tag, color in (("INFO", DIM), ("WARNING", HP_WARN),
                           ("ERROR", HP_LOW), ("GOOD", HP_OK), ("CHAT", ACCENT),
                           ("LOOT", DROP)):
            self.log_text.tag_config(tag, foreground=color)
        sb = tk.Scrollbar(self.log_text, command=self.log_text.yview)
        sb.pack(side="right", fill="y")
        self.log_text.configure(yscrollcommand=sb.set)

        chat_row = tk.Frame(bottom, bg=PANEL)
        chat_row.pack(side="bottom", fill="x", padx=6, pady=6)
        self.var_channel = tk.StringVar(value="local")
        tk.OptionMenu(chat_row, self.var_channel, "local", "global", "party",
                      "system").pack(side="left")
        self.chat_entry = tk.Entry(chat_row, bg="#0a0e14", fg=TEXT,
                                   insertbackground=TEXT, relief="flat")
        self.chat_entry.pack(side="left", fill="x", expand=True, padx=6)
        self.chat_entry.bind("<Return>", lambda e: self._send_chat())
        tk.Button(chat_row, text="发送", command=self._send_chat,
                  bg=BUTTON_BG, fg=BUTTON_FG, relief="flat").pack(side="left")

        # keyboard walk
        root.bind("<KeyPress>", self._on_key_press)
        root.bind("<KeyRelease>", self._on_key_release)

    def _build_side_panel(self, parent):
        # -- status --
        self._side_section(parent, "角色状态")
        bar = tk.Canvas(parent, width=238, height=14, bg=PANEL, highlightthickness=0)
        bar.pack(fill="x", padx=12, pady=(0, 2))
        self.hp_bar = bar
        self.hp_text = tk.StringVar(value="HP -/-")
        tk.Label(parent, textvariable=self.hp_text, fg=DIM, bg=PANEL,
                 font=("TkFixedFont", 9)).pack(anchor="w", padx=14)
        mp = tk.Canvas(parent, width=238, height=10, bg=PANEL, highlightthickness=0)
        mp.pack(fill="x", padx=12, pady=(6, 2))
        self.mp_bar = mp
        self.mp_text = tk.StringVar(value="MP -/-")
        tk.Label(parent, textvariable=self.mp_text, fg=DIM, bg=PANEL,
                 font=("TkFixedFont", 9)).pack(anchor="w", padx=14)

        self.status_vars = {}
        grid = tk.Frame(parent, bg=PANEL)
        grid.pack(fill="x", padx=12, pady=6)
        rows = [("level", "等级"), ("exp", "经验"), ("gp", "GP"),
                ("mana", "法力"), ("stamina", "体力"), ("pos", "坐标"),
                ("map", "地图"), ("potion", "药水")]
        for i, (key, label) in enumerate(rows):
            tk.Label(grid, text=label, width=6, anchor="e", fg=DIM, bg=PANEL,
                     font=("TkDefaultFont", 9)).grid(row=i, column=0, sticky="e")
            var = tk.StringVar(value="-")
            tk.Label(grid, textvariable=var, anchor="w", fg=TEXT, bg=PANEL,
                     font=("TkFixedFont", 9)).grid(row=i, column=1, sticky="w", padx=(6, 0))
            self.status_vars[key] = var

        # -- equipment / inventory -- (kept high: it is the core of the UI)
        self._side_section(parent, "装备 / 背包")
        eq_grid = tk.Frame(parent, bg=PANEL)
        eq_grid.pack(fill="x", padx=10)
        self.eq_buttons = {}
        for i, slot in enumerate(EQUIP_SLOTS):
            btn = tk.Button(eq_grid, text="空", width=12, anchor="w",
                            bg=BUTTON_BG, fg=DIM, relief="flat",
                            font=("TkDefaultFont", 8),
                            command=lambda s=slot: self._unequip_slot(s))
            btn.grid(row=i // 2, column=i % 2, padx=2, pady=2, sticky="ew")
            self.eq_buttons[slot] = btn
        eq_grid.grid_columnconfigure(0, weight=1)
        eq_grid.grid_columnconfigure(1, weight=1)

        inv_row = tk.Frame(parent, bg=PANEL)
        inv_row.pack(fill="x", padx=10, pady=(6, 2))
        self.inv_list = tk.Listbox(inv_row, height=4, bg="#0a0e14", fg=TEXT,
                                   relief="flat", highlightthickness=1,
                                   highlightbackground=BORDER, highlightcolor=ACCENT,
                                   selectbackground=ACCENT,
                                   selectforeground="#06131f",
                                   font=("TkFixedFont", 9))
        self.inv_list.pack(side="left", fill="both", expand=True)
        self.inv_list.bind("<Double-Button-1>", lambda e: self._equip_selected())
        inv_btns = tk.Frame(inv_row, bg=PANEL)
        inv_btns.pack(side="left", fill="y", padx=(4, 0))
        for text, cmd in (("装备", self._equip_selected), ("使用", self._use_selected)):
            tk.Button(inv_btns, text=text, width=4, command=cmd, bg=BUTTON_BG,
                      fg=BUTTON_FG, relief="flat",
                      font=("TkDefaultFont", 8)).pack(pady=1)

        # -- target --
        self._side_section(parent, "目标")
        self.target_text = tk.StringVar(value="无目标")
        tk.Label(parent, textvariable=self.target_text, anchor="w", fg=TEXT,
                 bg=PANEL, font=("TkDefaultFont", 9), wraplength=240,
                 justify="left").pack(fill="x", padx=12)
        tbar = tk.Canvas(parent, width=238, height=10, bg=PANEL, highlightthickness=0)
        tbar.pack(fill="x", padx=12, pady=(4, 2))
        self.target_bar = tbar
        self.target_hp = tk.StringVar(value="")
        tk.Label(parent, textvariable=self.target_hp, fg=DIM, bg=PANEL,
                 font=("TkFixedFont", 9)).pack(anchor="w", padx=14)
        btnrow = tk.Frame(parent, bg=PANEL)
        btnrow.pack(fill="x", padx=10, pady=6)
        for text, cmd in (("攻击", self._attack), ("拾取", self._pickup),
                          ("交互", self._interact)):
            tk.Button(btnrow, text=text, width=5, command=cmd, bg=BUTTON_BG,
                      fg=BUTTON_FG, relief="flat").pack(side="left", expand=True, padx=2)
        btnrow2 = tk.Frame(parent, bg=PANEL)
        btnrow2.pack(fill="x", padx=10, pady=(0, 6))
        for text, cmd in (("复活", self._respawn), ("喝药", self._drink)):
            tk.Button(btnrow2, text=text, width=5, command=cmd, bg=BUTTON_BG,
                      fg=BUTTON_FG, relief="flat").pack(side="left", expand=True, padx=2)

        # -- quests --
        self._side_section(parent, "任务")
        self.quest_list = tk.Listbox(parent, height=5, bg="#0a0e14", fg=TEXT,
                                     relief="flat", highlightthickness=1,
                                     highlightbackground=BORDER, highlightcolor=ACCENT,
                                     selectbackground=ACCENT,
                                     selectforeground="#06131f",
                                     font=("TkFixedFont", 9))
        self.quest_list.pack(fill="x", padx=10, pady=(0, 2))
        qbtn = tk.Frame(parent, bg=PANEL)
        qbtn.pack(fill="x", padx=10, pady=(0, 6))
        for text, cmd in (("任务日志", self._show_quest_log),
                          ("追踪目标", self._track_selected_quest)):
            tk.Button(qbtn, text=text, command=cmd, bg=BUTTON_BG, fg=BUTTON_FG,
                      relief="flat", font=("TkDefaultFont", 8)).pack(
                          side="left", expand=True, padx=1)

        # -- auto-play --
        self._side_section(parent, "自动挂机")
        self.var_combat = tk.BooleanVar(value=False)
        self.var_loot = tk.BooleanVar(value=False)
        self.var_potion = tk.BooleanVar(value=False)
        for text, var in (("自动打怪", self.var_combat), ("自动拾取", self.var_loot),
                          ("自动喝药", self.var_potion)):
            tk.Checkbutton(parent, text=text, variable=var, fg=TEXT, bg=PANEL,
                           selectcolor=PANEL, activebackground=PANEL,
                           activeforeground=TEXT).pack(anchor="w", padx=12)
        thresh_row = tk.Frame(parent, bg=PANEL)
        thresh_row.pack(fill="x", padx=12, pady=(4, 2))
        tk.Label(thresh_row, text="喝药阈值 %", fg=DIM, bg=PANEL,
                 font=("TkDefaultFont", 9)).pack(side="left")
        self.var_threshold = tk.StringVar(value="30")
        tk.Spinbox(thresh_row, from_=5, to=95, width=5, textvariable=self.var_threshold,
                   bg="#0a0e14", fg=TEXT, insertbackground=TEXT, relief="flat",
                   buttonbackground=BUTTON_BG).pack(side="left", padx=6)
        tk.Button(parent, text="应用挂机设置", command=self._apply_autoplay,
                  bg=ACCENT, fg="#06131f", relief="flat",
                  font=("TkDefaultFont", 9, "bold")).pack(fill="x", padx=12, pady=6)
        tk.Button(parent, text="停止全部挂机", command=self._stop_autoplay,
                  bg=BUTTON_BG, fg=BUTTON_FG, relief="flat").pack(fill="x", padx=12)

        # -- stats --
        self._side_section(parent, "统计")
        self.stats_vars = {}
        sgrid = tk.Frame(parent, bg=PANEL)
        sgrid.pack(fill="x", padx=12, pady=4)
        for i, (key, label) in enumerate((("kills", "击杀"), ("loots", "拾取"),
                                          ("deaths", "死亡"), ("dmg_dealt", "伤害"))):
            tk.Label(sgrid, text=label, width=6, anchor="e", fg=DIM, bg=PANEL,
                     font=("TkDefaultFont", 9)).grid(row=i, column=0, sticky="e")
            var = tk.StringVar(value="0")
            tk.Label(sgrid, textvariable=var, anchor="w", fg=TEXT, bg=PANEL,
                     font=("TkFixedFont", 9)).grid(row=i, column=1, sticky="w", padx=(6, 0))
            self.stats_vars[key] = var

        # -- minimap --
        self._side_section(parent, "小地图")
        mm = tk.Canvas(parent, width=238, height=150, bg="#0a0e14", highlightthickness=0)
        mm.pack(fill="x", padx=12, pady=(0, 10))
        self.minimap = mm

    def _side_section(self, parent, title):
        tk.Frame(parent, bg=BORDER, height=1).pack(fill="x", padx=10, pady=(10, 0))
        tk.Label(parent, text=title, anchor="w", fg=ACCENT, bg=PANEL,
                 font=("TkDefaultFont", 10, "bold")).pack(fill="x", padx=12, pady=(6, 4))

    # -- autoplay -----------------------------------------------------------

    def _apply_autoplay(self, initial=False):
        if self.client is None:
            return
        try:
            threshold = int(self.var_threshold.get())
        except (ValueError, AttributeError):
            threshold = 30
        self.runner.submit(
            "autoplay",
            combat=bool(self.var_combat.get()),
            loot=bool(self.var_loot.get()),
            potion=bool(self.var_potion.get()),
            threshold=threshold,
        )
        if not initial:
            self._append_log(
                f"挂机设置: 打怪={self.var_combat.get()} 拾取={self.var_loot.get()} "
                f"喝药={self.var_potion.get()} 阈值={threshold}%", "INFO")

    def _stop_autoplay(self):
        self.var_combat.set(False)
        self.var_loot.set(False)
        self.var_potion.set(False)
        self._apply_autoplay()

    # -- actions ------------------------------------------------------------

    def _attack(self):
        if self.client is None:
            return
        if self.target_kind == "npc" and self.target_id is not None:
            self.runner.submit("attack", rid=self.target_id)
            return
        target = self.client.world.find_nearest_npc(max_dist=400)
        if target:
            pid, npc = target
            self.target_kind = "npc"
            self.target_id = pid
            self.runner.submit("attack", rid=pid)
            self._update_target_frame()
        else:
            self._append_log("附近没有可攻击的目标", "WARNING")

    def _pickup(self):
        if self.client is None:
            return
        if self.target_kind == "drop" and self.target_id is not None:
            self.runner.submit("pickup", rid=self.target_id)
            return
        drop = self.client.world.find_nearest_drop(max_dist=200)
        if drop:
            rid, d = drop
            pos = d.get("pos")
            if pos:
                self.runner.submit("move_to", x=pos[0], y=pos[1])
            self.runner.submit("pickup", rid=rid)
        else:
            self._append_log("附近没有可拾取的掉落", "WARNING")

    def _interact(self):
        if self.client is None or self.target_kind != "npc" or self.target_id is None:
            self._append_log("请先点击选择一个 NPC", "WARNING")
            return
        npc = self.client.world.npcs.get(self.target_id)
        rid = npc.get("rid") if npc else None
        if rid is None:
            rid = -self.target_id
        self.runner.submit("interact", rid=int(rid))

    def _respawn(self):
        if self.client is None:
            return
        self.runner.submit("respawn")
        self._append_log("已请求复活", "INFO")

    def _drink(self):
        if self.client is None:
            return
        self.runner.submit("use_item", item_id=ITEM_POTION)
        self._append_log("使用药水", "INFO")

    # -- equipment / inventory ----------------------------------------------

    SLOT_LABELS = {
        "head": "头部", "neck": "项链", "chest": "胸甲", "hands": "手套",
        "weapon": "武器", "shield": "盾牌", "legs": "护腿", "feet": "靴子",
    }

    def _update_equipment(self):
        """Refresh the equipment bar + inventory list from world state.

        Only rebuilds when the (equipment, inventory) signature changes so the
        50 ms UI tick stays cheap.
        """
        world = self.client.world if self.client else None
        if world is None or not hasattr(self, "eq_buttons"):
            return
        eq_sig = tuple(sorted(
            (slot, rec.get("item_id"))
            for slot, rec in (world.equipment or {}).items()))
        inv_sig = tuple(sorted((world.inventory or {}).items()))
        sig = (eq_sig, inv_sig)
        if sig == self._eq_sig:
            return
        self._eq_sig = sig

        for slot, btn in self.eq_buttons.items():
            rec = (world.equipment or {}).get(slot)
            item_id = rec.get("item_id") if rec else None
            label = self.SLOT_LABELS.get(slot, slot)
            if item_id is None or item_id == -1:
                btn.configure(text=f"{label}: 空", fg=DIM)
            else:
                btn.configure(text=f"{label}: {item_name(item_id)}", fg=TEXT)

        inv = world.inventory or {}
        self._inv_items = sorted(inv)
        self.inv_list.delete(0, "end")
        for item_id in self._inv_items:
            slot = item_slot(item_id)
            tag = "" if slot < 0 else "  [可装备]"
            self.inv_list.insert(
                "end", f"{item_name(item_id)} x{inv[item_id]}{tag}")

    # -- quests -------------------------------------------------------------

    def _update_quest_tracker(self):
        """Rebuild the side-panel quest list when its inputs change.

        The list shows every active quest plus a one-line objective (where to
        go, or collection/kill progress), so the player always knows what to
        do next without opening the log.
        """
        if not hasattr(self, "quest_list"):
            return
        world = self.client.world if self.client else None
        if world is None:
            return
        states = world.quests or {}
        sig = (tuple(sorted(states.items())),
               tuple(sorted((world.inventory or {}).items())),
               tuple(sorted((world.bestiary or {}).items())))
        if sig == self._quest_tracker_sig:
            return
        self._quest_tracker_sig = sig

        active = []
        for qid in sorted(states):
            try:
                state = int(states.get(qid) or 0)
            except (TypeError, ValueError):
                continue
            if state <= QUEST_STATE_INACTIVE or state >= QUEST_STATE_DONE:
                continue
            active.append((qid, state))
        self._quest_active = active

        self.quest_list.delete(0, "end")
        if not active:
            self.quest_list.insert("end", "（暂无进行中的任务）")
        else:
            for qid, state in active:
                obj = self.quests.objective(qid, state, world)
                self.quest_list.insert("end", f"{self.quests.name(qid)}  ·  {obj}")
        # Keep the open quest log's progress lines live as items/kills change.
        if self.quest_log_text is not None:
            self._refresh_quest_log()

    def _quest_markers(self, world):
        """Map npc nick -> marker char for the world overlay.

        '!' = the NPC offers a quest the player can still take.
        '?' = the NPC is the current objective: either the person to talk to,
              or the giver the finished stage can be handed in to.
        """
        markers = {}
        if not self.quests.loadable:
            return markers
        states = world.quests or {}
        for nick, qids in self.quests.by_giver.items():
            for qid in qids:
                q = self.quests.get(qid)
                if q is None or q.get("hidden"):
                    continue
                try:
                    state = int(states.get(qid, 0) or 0)
                except (TypeError, ValueError):
                    state = 0
                if state <= QUEST_STATE_INACTIVE or (
                        state >= QUEST_STATE_DONE and q.get("repeatable")):
                    markers[nick] = "!"
        for qid in states:
            try:
                state = int(states.get(qid) or 0)
            except (TypeError, ValueError):
                continue
            if state <= QUEST_STATE_INACTIVE or state >= QUEST_STATE_DONE:
                continue
            q = self.quests.get(qid)
            stage = self.quests.stage(qid, state)
            if q is None or stage is None:
                continue
            req = stage.get("req") or {}
            kind = req.get("kind") or ""
            if kind == "talk":
                npc = req.get("talkNpc")
                if npc:
                    markers[npc] = "?"
            elif kind in ("item", "kill", ""):
                giver = q.get("giver")
                if giver and self.quests.stage_ready(qid, state, world):
                    markers[giver] = "?"
        return markers

    def _track_selected_quest(self):
        """Pan the camera to the selected quest's objective NPC."""
        if self.client is None:
            return
        sel = self.quest_list.curselection()
        if not sel or not self._quest_active:
            self._append_log("请先在任务列表里选中一个任务", "WARNING")
            return
        idx = sel[0]
        if idx >= len(self._quest_active):
            return
        qid, state = self._quest_active[idx]
        world = self.client.world
        q = self.quests.get(qid)
        stage = self.quests.stage(qid, state)
        if q is None or stage is None:
            return
        req = stage.get("req") or {}
        nick = req.get("talkNpc") if req.get("kind") == "talk" else q.get("giver")
        target = None
        for npc in world.npcs.values():
            if npc.get("name") == nick and npc.get("pos"):
                target = npc
                break
        if target is None:
            self._append_log(f"找不到目标 NPC：{nick}", "WARNING")
            return
        pos = target["pos"]
        self.cam_hold = ((pos[0], pos[1]), time.time() + 4.0)
        self._append_log(f"追踪 {self.quests.name(qid)}：前往 {nick} "
                         f"({int(pos[0])},{int(pos[1])})", "INFO")

    def _quest_for_dialog(self, author, text):
        """The quest being offered or discussed at `author`, if we can tell."""
        if not self.quests.loadable or not author or not text:
            return None
        for q in self.quests.for_npc(author):
            for st in q.get("stages") or []:
                if st.get("text") == text:
                    return q
                if (st.get("textByNpc") or {}).get(author) == text:
                    return q
        # Completion lines are wrapped as 【name】... rather than stage text.
        if text.startswith("【") and "】" in text:
            name = text[1:text.index("】")]
            for q in self.quests.for_npc(author):
                if q.get("name") == name:
                    return q
        return None

    # -- quest log window ---------------------------------------------------

    def _show_quest_log(self):
        if self.quest_log_window is not None and self.quest_log_window.winfo_exists():
            self.quest_log_window.lift()
            self._refresh_quest_log()
            return
        win = tk.Toplevel(self.root)
        self.quest_log_window = win
        win.title("任务日志")
        win.geometry("640x500")
        win.configure(bg=BG)
        win.protocol("WM_DELETE_WINDOW", self._close_quest_log)

        bar = tk.Frame(win, bg=PANEL)
        bar.pack(side="top", fill="x", padx=8, pady=8)
        tk.Label(bar, text="显示：", fg=DIM, bg=PANEL,
                 font=("TkDefaultFont", 9)).pack(side="left")
        self.quest_log_filter = tk.StringVar(value="全部")
        tk.OptionMenu(bar, self.quest_log_filter, "全部", "进行中", "已完成", "未接",
                      command=lambda _: self._refresh_quest_log()).pack(side="left")
        tk.Label(bar, text="金色 ! = 可接取    蓝色 ? = 可交付 / 该去找的 NPC",
                 fg=DIM, bg=PANEL, font=("TkDefaultFont", 8)).pack(side="right")

        txt = tk.Text(win, bg="#0a0e14", fg=TEXT, wrap="word", relief="flat",
                      state="disabled", font=("TkFixedFont", 9), padx=10, pady=8)
        txt.pack(side="top", fill="both", expand=True, padx=8, pady=(0, 8))
        for tag, color in (("title", ACCENT), ("done", HP_OK), ("active", "#ffd27a"),
                           ("idle", DIM), ("obj", TEXT)):
            txt.tag_config(tag, foreground=color)
        self.quest_log_text = txt
        self._refresh_quest_log()

    def _close_quest_log(self):
        if self.quest_log_window is not None:
            try:
                self.quest_log_window.destroy()
            except tk.TclError:
                pass
        self.quest_log_window = None
        self.quest_log_text = None

    def _refresh_quest_log(self):
        txt = self.quest_log_text
        if txt is None:
            return
        world = self.client.world if self.client else None
        states = (world.quests if world else {}) or {}
        filt = self.quest_log_filter.get() if self.quest_log_filter else "全部"

        rows = []
        if not self.quests.loadable:
            rows.append(("未找到 quests.json，任务日志不可用。", "idle"))
            rows.append((f"预期位置：{self.quests.path or '(未设置)'}", "idle"))
        else:
            for qid in sorted(self.quests.quests):
                q = self.quests.quests[qid]
                try:
                    state = int(states.get(qid, 0) or 0)
                except (TypeError, ValueError):
                    state = 0
                if state <= QUEST_STATE_INACTIVE:
                    label, tag = "未接", "idle"
                elif state >= QUEST_STATE_DONE:
                    label, tag = "已完成", "done"
                else:
                    label, tag = f"阶段 {state}", "active"
                if filt == "进行中" and tag != "active":
                    continue
                if filt == "已完成" and tag != "done":
                    continue
                if filt == "未接" and tag != "idle":
                    continue
                rows.append((f"{q.get('name') or f'任务 #{qid}'}  [{label}]", "title"))
                if q.get("description"):
                    rows.append((q["description"].strip(), "idle"))
                rows.append((f"给予：{q.get('giver', '?')}    "
                             f"奖励：{self.quests.reward_text(qid)}", "idle"))
                stage = self.quests.stage(qid, state)
                if stage is not None:
                    rows.append((f"当前：{self.quests.objective(qid, state, world)}",
                                 "obj"))
                    if stage.get("text"):
                        rows.append((stage["text"], "idle"))
                if q.get("repeatable"):
                    rows.append(("（可重复）", "idle"))
                rows.append(("", "idle"))

        txt.configure(state="normal")
        txt.delete("1.0", "end")
        for text, tag in rows:
            txt.insert("end", text + "\n", tag)
        txt.configure(state="disabled")

    def _equip_selected(self):
        if self.client is None:
            return
        sel = self.inv_list.curselection()
        if not sel:
            self._append_log("请先在背包里选中一件装备", "WARNING")
            return
        item_id = self._inv_items[sel[0]]
        if item_slot(item_id) < 0:
            self._append_log(f"{item_name(item_id)} 不能装备", "WARNING")
            return
        self.runner.submit("equip", item_id=int(item_id), item_index=0)
        self._append_log(f"装备 {item_name(item_id)}...", "INFO")

    def _use_selected(self):
        if self.client is None:
            return
        sel = self.inv_list.curselection()
        if not sel:
            self._append_log("请先在背包里选中一件物品", "WARNING")
            return
        item_id = self._inv_items[sel[0]]
        self.runner.submit("use_item", item_id=int(item_id))
        self._append_log(f"使用 {item_name(item_id)}...", "INFO")

    def _unequip_slot(self, slot):
        if self.client is None:
            return
        rec = (self.client.world.equipment or {}).get(slot)
        item_id = rec.get("item_id") if rec else None
        if item_id is None or item_id == -1:
            return
        self.runner.submit("unequip", item_id=int(item_id))
        self._append_log(f"卸下 {item_name(item_id)}...", "INFO")

    # -- dialogue -----------------------------------------------------------

    def _build_dialog(self):
        d = tk.Toplevel(self.root)
        d.title("对话")
        d.configure(bg=PANEL)
        d.transient(self.root)
        d.protocol("WM_DELETE_WINDOW", self._close_dialog)
        d.withdraw()

        self.dlg_quest = tk.StringVar(value="")
        tk.Label(d, textvariable=self.dlg_quest, anchor="w", fg="#ffd27a",
                 bg=PANEL, font=("TkDefaultFont", 9, "bold")).pack(
                     fill="x", padx=12, pady=(10, 2))
        self.dlg_author = tk.StringVar(value="")
        tk.Label(d, textvariable=self.dlg_author, anchor="w", fg=ACCENT,
                 bg=PANEL, font=("TkDefaultFont", 11, "bold")).pack(
                     fill="x", padx=12, pady=(0, 2))
        self.dlg_text = tk.StringVar(value="")
        tk.Label(d, textvariable=self.dlg_text, anchor="w", fg=TEXT, bg=PANEL,
                 font=("TkDefaultFont", 10), wraplength=400,
                 justify="left").pack(fill="x", padx=12, pady=(0, 8))
        self.dlg_choices = tk.Frame(d, bg=PANEL)
        self.dlg_choices.pack(fill="x", padx=12)
        row = tk.Frame(d, bg=PANEL)
        row.pack(fill="x", padx=12, pady=(8, 10))
        self.btn_dlg_next = tk.Button(row, text="继续", width=8,
                                      command=self._dialog_next, bg=BUTTON_BG,
                                      fg=BUTTON_FG, relief="flat")
        self.btn_dlg_next.pack(side="left", padx=(0, 6))
        tk.Button(row, text="关闭", width=8, command=self._close_dialog,
                  bg=BUTTON_BG, fg=BUTTON_FG, relief="flat").pack(side="left")
        self.dlg_window = d

    def _show_dialog(self):
        if self.dlg_window is None:
            self._build_dialog()
        self.dlg_window.deiconify()
        self.dlg_window.lift()

    def _set_dialog_choices(self, choices):
        for w in self.dlg_choices.winfo_children():
            w.destroy()
        for i, text in enumerate(choices):
            tk.Button(self.dlg_choices, text=str(text), anchor="w",
                      command=lambda c=i: self._dialog_choice(c),
                      bg=BUTTON_BG, fg=BUTTON_FG, relief="flat",
                      font=("TkDefaultFont", 9)).pack(fill="x", pady=1)

    def _dialog_choice(self, choice_id):
        self._append_log(f"选择对话选项 #{choice_id}", "INFO")
        self.runner.submit("dialog_choice", choice=choice_id)

    def _dialog_next(self):
        self.runner.submit("dialog_next")

    def _close_dialog(self):
        if self.dlg_window is not None:
            self.dlg_window.withdraw()
        self.runner.submit("dialog_close")

    # -- AI arena (optional) -----------------------------------------------

    def _open_arena(self):
        if self.arena_window is not None and self.arena_window.winfo_exists():
            self.arena_window.lift()
            return
        if self.client is None:
            self._append_log("尚未进入世界，无法开启竞技场", "WARNING")
            return
        d = tk.Toplevel(self.root)
        d.title("AI 竞技场（可选）")
        d.configure(bg=PANEL)
        d.transient(self.root)
        d.geometry("340x260")
        self.arena_window = d

        tk.Label(d, text="AI 竞技场为可选项，不影响正常游戏流程",
                 fg=DIM, bg=PANEL, font=("TkDefaultFont", 9),
                 wraplength=320, justify="left").pack(fill="x", padx=12, pady=(10, 6))

        tk.Label(d, text="场地模板", fg=TEXT, bg=PANEL,
                 font=("TkDefaultFont", 9)).pack(anchor="w", padx=12)
        self.var_arena_template = tk.StringVar(value=ARENA_TEMPLATES[0][0])
        for key, label in ARENA_TEMPLATES:
            tk.Radiobutton(d, text=f"{key}  —  {label}", anchor="w",
                           variable=self.var_arena_template, value=key,
                           fg=TEXT, bg=PANEL, selectcolor=PANEL,
                           activebackground=PANEL, activeforeground=TEXT,
                           font=("TkDefaultFont", 9)).pack(fill="x", padx=12)

        self.arena_detail = tk.StringVar(value="未开启")
        tk.Label(d, textvariable=self.arena_detail, anchor="w", fg=TEXT,
                 bg=PANEL, font=("TkFixedFont", 9), wraplength=320,
                 justify="left").pack(fill="x", padx=12, pady=(8, 6))

        row = tk.Frame(d, bg=PANEL)
        row.pack(fill="x", padx=12, pady=(0, 10))
        for text, cmd in (("开始", self._arena_start), ("状态", self._arena_status),
                          ("停止", self._arena_stop)):
            tk.Button(row, text=text, width=7, command=cmd, bg=BUTTON_BG,
                      fg=BUTTON_FG, relief="flat").pack(side="left", expand=True, padx=2)
        # Ask the server for the real state instead of resetting the label.
        self.runner.submit("command", text="/arena status")

    def _refresh_arena_label(self):
        # The toolbar only exists after the main UI is built; feedback can
        # arrive earlier (e.g. a notice while still in the login screen).
        if getattr(self, "arena_label", None) is not None:
            self.arena_label.set(f"AI竞技场: {self.arena_status_text}")

    def _arena_start(self):
        if self.client is None:
            return
        template = self.var_arena_template.get()
        self._append_log(f"请求开启 AI 竞技场: {template}", "INFO")
        self.runner.submit("command", text=f"/arena start {template}")

    def _arena_status(self):
        if self.client is None:
            return
        self.runner.submit("command", text="/arena status")

    def _arena_stop(self):
        if self.client is None:
            return
        self.runner.submit("command", text="/arena stop")

    def _handle_system_text(self, text):
        """CommandFeedback from the server (/arena replies, notices...)."""
        if not text:
            return
        self._append_log(f"[系统] {text}", "INFO")
        if text.startswith("Arena") or text.startswith("Usage: /arena"):
            self._handle_arena_text(text)

    def _handle_arena_text(self, text):
        """Parse the server's /arena feedback (CommandFeedback)."""
        line = text.strip()

        m = re.match(r"^Arena started:\s*(\S+)\s*—\s*(.*)$", line)
        if m:
            self.arena_status_text = "进行中 0/?"
            self._append_log(f"AI 竞技场已开启: {m.group(1)}", "GOOD")
            if m.group(2):
                self._append_log(f"  目标: {m.group(2)}", "INFO")
        elif (m := re.match(
                r"^Arena (\S+): kills (\d+)/(\S+), deaths (\d+), xp (\S+)", line)):
            status, kills, required = m.group(1), m.group(2), m.group(3)
            # The server keeps a finished episode for the record; only an
            # active run blocks a new one.
            self.arena_status_text = (
                f"进行中 {kills}/{required}" if status == "active"
                else f"已结束 {kills}/{required}")
        elif line == "Arena stopped.":
            self.arena_status_text = "已停止"
        elif line.startswith("Arena start failed:"):
            self.arena_status_text = "开启失败"
        elif line.startswith("Arena stop failed:"):
            self.arena_status_text = "停止失败"
        elif "no episode" in line or line.startswith("Usage: /arena"):
            self.arena_status_text = "未开启"
        # Anything else is a notice/usage hint: log it, keep the last state.

        if self.arena_window is not None and self.arena_window.winfo_exists():
            self.arena_detail.set(self.arena_status_text)
        self._refresh_arena_label()

    def _send_chat(self):
        text = self.chat_entry.get().strip()
        if not text or self.client is None:
            return
        if text.startswith("/"):
            self.runner.submit("command", text=text)
        else:
            self.runner.submit("chat", channel=self.var_channel.get(), text=text)
        self.chat_entry.delete(0, "end")

    def _disconnect(self):
        if self.client is not None:
            self.runner.submit("disconnect")
        self.conn_status.set("已断开")

    def _screenshot(self):
        path = filedialog.asksaveasfilename(
            title="保存截图",
            defaultextension=".eps",
            filetypes=[("PostScript", "*.eps"), ("所有文件", "*")],
            initialfile="som_lite_client.eps",
        )
        if not path:
            return
        try:
            self.canvas.postscript(file=path, colormode="color")
            self._append_log(f"截图已保存: {path}", "GOOD")
        except Exception as e:
            self._append_log(f"截图失败: {e}", "ERROR")

    # -- viewport -----------------------------------------------------------

    def _on_canvas_resize(self, event):
        self.view_w = max(50, event.width)
        self.view_h = max(50, event.height)

    def _set_zoom(self, z):
        self.zoom = max(0.35, min(2.5, z))
        self.zoom_label.set(f"{int(self.zoom * 100)}%")

    def _on_mouse_wheel(self, event):
        if hasattr(event, "delta") and event.delta:
            self._set_zoom(self.zoom * (1.1 if event.delta > 0 else 1 / 1.1))
        else:
            self._set_zoom(self.zoom * (1.1 if event.num == 4 else 1 / 1.1))

    def _on_canvas_motion(self, event):
        wx, wy = self._screen_to_world(event.x, event.y)
        self.conn_status.set(f"坐标 ({int(wx)}, {int(wy)})  缩放 {int(self.zoom*100)}%")

    def _world_to_screen(self, wx, wy):
        sx = (wx - self.cam[0]) * self.zoom + self.view_w / 2
        sy = (wy - self.cam[1]) * self.zoom + self.view_h / 2
        return sx, sy

    def _screen_to_world(self, sx, sy):
        wx = (sx - self.view_w / 2) / self.zoom + self.cam[0]
        wy = (sy - self.view_h / 2) / self.zoom + self.cam[1]
        return wx, wy

    def _hit_test(self, wx, wy):
        if self.client is None:
            return None
        best = None
        best_d = 20.0 / self.zoom
        for pid, npc in self.client.world.npcs.items():
            pos = npc.get("pos")
            if not pos:
                continue
            d = math.hypot(wx - pos[0], wy - pos[1])
            if d < best_d:
                best, best_d = ("npc", pid), d
        for pid, p in self.client.world.players.items():
            pos = p.get("pos")
            if not pos:
                continue
            d = math.hypot(wx - pos[0], wy - pos[1])
            if d < best_d:
                best, best_d = ("player", pid), d
        for rid, dr in self.client.world.drops.items():
            pos = dr.get("pos")
            if not pos:
                continue
            d = math.hypot(wx - pos[0], wy - pos[1])
            if d < best_d:
                best, best_d = ("drop", rid), d
        return best

    def _on_canvas_click(self, event):
        if self.client is None:
            return
        wx, wy = self._screen_to_world(event.x, event.y)
        hit = self._hit_test(wx, wy)
        if hit is None:
            self.runner.submit("move_to", x=wx, y=wy)
            self._add_effect({"kind": "marker", "pos": (wx, wy), "t0": time.time(),
                              "dur": 0.7})
            return
        kind, eid = hit
        if kind == "drop":
            self.target_kind, self.target_id = "drop", eid
            pos = self.client.world.drops[eid].get("pos")
            if pos:
                self.runner.submit("move_to", x=pos[0], y=pos[1])
                self.runner.submit("pickup", rid=eid)
        elif kind == "npc":
            self.target_kind, self.target_id = "npc", eid
            npc = self.client.world.npcs[eid]
            rid = npc.get("rid")
            if rid is not None:
                self.runner.submit("select", rid=int(rid))
            self.runner.submit("attack", rid=eid)
            self._add_effect({"kind": "marker", "pos": (wx, wy), "t0": time.time(),
                              "dur": 0.4})
        elif kind == "player":
            self.target_kind, self.target_id = "player", eid
            self.runner.submit("select", rid=eid)
        self._update_target_frame()

    def _on_canvas_right_click(self, event):
        if self.client is None:
            return
        wx, wy = self._screen_to_world(event.x, event.y)
        self.runner.submit("move_to", x=wx, y=wy)
        self._add_effect({"kind": "marker", "pos": (wx, wy), "t0": time.time(),
                          "dur": 0.7})

    def _on_key_press(self, event):
        if self._focus_is_entry():
            return
        k = event.keysym.lower()
        if k in ("w", "a", "s", "d", "up", "down", "left", "right"):
            self.keys_pressed.add(k)
            self._update_walk()

    def _on_key_release(self, event):
        k = event.keysym.lower()
        if k in self.keys_pressed:
            self.keys_pressed.discard(k)
            self._update_walk()

    def _focus_is_entry(self):
        w = self.root.focus_get()
        return isinstance(w, (tk.Entry, tk.Text, tk.Spinbox))

    def _update_walk(self):
        if self.client is None:
            return
        dx = dy = 0
        if "w" in self.keys_pressed or "up" in self.keys_pressed:
            dy -= 1
        if "s" in self.keys_pressed or "down" in self.keys_pressed:
            dy += 1
        if "a" in self.keys_pressed or "left" in self.keys_pressed:
            dx -= 1
        if "d" in self.keys_pressed or "right" in self.keys_pressed:
            dx += 1
        if dx == 0 and dy == 0:
            self.runner.submit("stop_walk")
            return
        length = math.hypot(dx, dy)
        self.runner.submit("walk", dx=dx / length, dy=dy / length)

    def _add_effect(self, eff):
        self.effects.append(eff)
        if len(self.effects) > 120:
            self.effects = self.effects[-120:]

    # -- drawing ------------------------------------------------------------

    def _tick(self):
        client = self.runner.client
        if client is not None:
            for ev in client.poll_events(64):
                self._handle_event(ev)
        if self._main_ready:
            self._update_status()
            self._update_camera()
            self._draw()
            self._draw_minimap()
        self.root.after(50, self._tick)

    def _update_camera(self):
        if self.client is None:
            return
        if self.cam_hold:
            pos, until = self.cam_hold
            if time.time() > until:
                self.cam_hold = None
            else:
                self.cam[0] += (pos[0] - self.cam[0]) * 0.18
                self.cam[1] += (pos[1] - self.cam[1]) * 0.18
                return
        pos = self.client.world.self_pos
        if pos:
            tx, ty = pos
            self.cam[0] += (tx - self.cam[0]) * 0.18
            self.cam[1] += (ty - self.cam[1]) * 0.18

    def _draw(self):
        cv = self.canvas
        cv.delete("all")
        w, h = self.view_w, self.view_h
        cv.create_rectangle(0, 0, w, h, fill=BG, outline="")
        cam_x, cam_y = self.cam
        zoom = self.zoom
        x0 = cam_x - (w / 2) / zoom
        x1 = cam_x + (w / 2) / zoom
        y0 = cam_y - (h / 2) / zoom
        y1 = cam_y + (h / 2) / zoom

        # grid
        step = 100
        gx = math.floor(x0 / step) * step
        while gx <= x1:
            sx = (gx - cam_x) * zoom + w / 2
            major = abs(gx) % 500 == 0
            cv.create_line(sx, 0, sx, h, fill=GRID_MAJOR if major else GRID)
            gx += step
        gy = math.floor(y0 / step) * step
        while gy <= y1:
            sy = (gy - cam_y) * zoom + h / 2
            major = abs(gy) % 500 == 0
            cv.create_line(0, sy, w, sy, fill=GRID_MAJOR if major else GRID)
            gy += step
        # origin axes
        ox, oy = self._world_to_screen(0, 0)
        cv.create_line(ox, 0, ox, h, fill="#3a2a12", width=1)
        cv.create_line(0, oy, w, oy, fill="#3a2a12", width=1)

        world = self.client.world if self.client else None
        if world is None:
            return

        # drops
        for rid, d in world.drops.items():
            pos = d.get("pos")
            if not pos:
                continue
            sx, sy = self._world_to_screen(pos[0], pos[1])
            if not (-20 < sx < w + 20 and -20 < sy < h + 20):
                continue
            r = 5 * zoom
            cv.create_polygon(sx, sy - r, sx + r, sy, sx, sy + r, sx - r, sy,
                              fill=DROP, outline="#7a6418")

        # npcs
        markers = self._quest_markers(world) if self.quests.loadable else {}
        for pid, npc in world.npcs.items():
            pos = npc.get("pos")
            if not pos:
                continue
            sx, sy = self._world_to_screen(pos[0], pos[1])
            if not (-40 < sx < w + 40 and -40 < sy < h + 40):
                continue
            friendly = npc.get("friendly", False)
            dead = npc.get("state") == 4
            color = FRIENDLY if friendly else MONSTER
            r = 11 * zoom
            if dead:
                cv.create_oval(sx - r, sy - r * 0.5, sx + r, sy + r * 0.5,
                               fill="#3a3f46", outline="#22262c")
                self._draw_label(sx, sy - r * 0.6, npc.get("name") or "?",
                                 "#6b7683", zoom)
                continue
            cv.create_oval(sx - r, sy - r, sx + r, sy + r, fill=color,
                           outline="#0a0e14", width=2)
            if self.target_kind == "npc" and self.target_id == pid:
                cv.create_oval(sx - r - 4, sy - r - 4, sx + r + 4, sy + r + 4,
                               outline="#ffffff", width=1, dash=(3, 3))
            self._draw_label(sx, sy - r - 6, npc.get("name") or "?", TEXT, zoom)
            marker = markers.get(npc.get("name"))
            if marker:
                color = "#ffd27a" if marker == "!" else "#7ae0ff"
                cv.create_text(sx, sy - r - 6 - 12 * zoom, text=marker,
                               fill=color,
                               font=("TkDefaultFont", int(12 * min(1.4, zoom)), "bold"),
                               anchor="s")
            hp, max_hp = npc.get("hp"), npc.get("max_hp")
            if hp is not None and max_hp:
                self._draw_bar(sx, sy + r + 4, 36 * zoom, hp / max_hp)

        # players
        for pid, p in world.players.items():
            pos = p.get("pos")
            if not pos:
                continue
            sx, sy = self._world_to_screen(pos[0], pos[1])
            if not (-40 < sx < w + 40 and -40 < sy < h + 40):
                continue
            r = 11 * zoom
            cv.create_oval(sx - r, sy - r, sx + r, sy + r, fill=PLAYER,
                           outline="#0a0e14", width=2)
            if self.target_kind == "player" and self.target_id == pid:
                cv.create_oval(sx - r - 4, sy - r - 4, sx + r + 4, sy + r + 4,
                               outline="#ffffff", width=1, dash=(3, 3))
            self._draw_label(sx, sy - r - 6, p.get("nick") or "?", TEXT, zoom)
            hp, max_hp = p.get("hp"), p.get("max_hp")
            if hp is not None and max_hp:
                self._draw_bar(sx, sy + r + 4, 36 * zoom, hp / max_hp)

        # self
        if world.self_pos:
            sx, sy = self._world_to_screen(world.self_pos[0], world.self_pos[1])
            r = 12 * zoom
            cv.create_oval(sx - r, sy - r, sx + r, sy + r, fill=SELF,
                           outline="#ffffff", width=2)
            self._draw_label(sx, sy - r - 6, world.self_nick or "我", "#ffffff", zoom)
            if world.self_max_hp:
                self._draw_bar(sx, sy + r + 4, 44 * zoom,
                               world.self_hp / world.self_max_hp)

        # effects
        now = time.time()
        self.effects = [e for e in self.effects if now - e["t0"] < e["dur"]]
        for eff in self.effects:
            age = now - eff["t0"]
            t = age / eff["dur"]
            if eff["kind"] == "marker":
                sx, sy = self._world_to_screen(*eff["pos"])
                r = 6 + 8 * t
                color = _lerp_color("#ff5555", "#0d1117", t)
                cv.create_line(sx - r, sy - r, sx + r, sy + r, fill=color, width=2)
                cv.create_line(sx - r, sy + r, sx + r, sy - r, fill=color, width=2)
            elif eff["kind"] == "floater":
                sx, sy = self._world_to_screen(*eff["pos"])
                sy -= 20 + 26 * t
                color = _lerp_color(eff.get("color", "#ffffff"), "#0d1117", t)
                cv.create_text(sx, sy, text=eff.get("text", ""), fill=color,
                               font=("TkDefaultFont", int(10 + 4 * (1 - t)), "bold"),
                               anchor="center")
            elif eff["kind"] == "line":
                x1, y1 = self._world_to_screen(*eff["from"])
                x2, y2 = self._world_to_screen(*eff["to"])
                color = _lerp_color(eff.get("color", "#ffd27a"), "#0d1117", t)
                cv.create_line(x1, y1, x2, y2, fill=color, width=2)

        # quest marker legend (bottom-left), only when a catalog is loaded
        if markers:
            cv.create_text(10, h - 26, text="!  可接取任务", fill="#ffd27a",
                           font=("TkDefaultFont", 9, "bold"), anchor="w")
            cv.create_text(10, h - 12, text="?  可交付 / 该去找的 NPC", fill="#7ae0ff",
                           font=("TkDefaultFont", 9, "bold"), anchor="w")

    def _draw_label(self, x, y, text, color, zoom):
        size = max(8, int(9 * min(1.4, zoom)))
        font = ("TkDefaultFont", size)
        self.canvas.create_text(x + 1, y + 1, text=text, fill="#000000",
                                font=font, anchor="s")
        self.canvas.create_text(x, y, text=text, fill=color, font=font, anchor="s")

    def _draw_bar(self, x, y, width, pct):
        pct = max(0.0, min(1.0, pct))
        self.canvas.create_rectangle(x - width / 2, y, x + width / 2, y + 4,
                                     fill=HP_BG, outline="")
        if pct > 0:
            color = HP_OK if pct > 0.5 else HP_WARN if pct > 0.25 else HP_LOW
            self.canvas.create_rectangle(x - width / 2, y,
                                         x - width / 2 + width * pct, y + 4,
                                         fill=color, outline="")

    def _draw_minimap(self):
        mm = self.minimap
        mm.delete("all")
        w, h = 238, 150
        mm.create_rectangle(0, 0, w, h, fill="#0a0e14", outline="")
        world = self.client.world if self.client else None
        if world is None:
            return
        pts = []
        if world.self_pos:
            pts.append(world.self_pos)
        for npc in world.npcs.values():
            if npc.get("pos"):
                pts.append(npc["pos"])
        for p in world.players.values():
            if p.get("pos"):
                pts.append(p["pos"])
        if not pts:
            return
        xs = [p[0] for p in pts]
        ys = [p[1] for p in pts]
        margin = 60
        x0, x1 = min(xs) - margin, max(xs) + margin
        y0, y1 = min(ys) - margin, max(ys) + margin
        scale = min(w / max(1, x1 - x0), h / max(1, y1 - y0))

        def mmxy(p):
            return ((p[0] - x0) * scale, (p[1] - y0) * scale)

        for npc in world.npcs.values():
            pos = npc.get("pos")
            if not pos:
                continue
            mx, my = mmxy(pos)
            color = FRIENDLY if npc.get("friendly") else MONSTER
            mm.create_oval(mx - 2, my - 2, mx + 2, my + 2, fill=color, outline="")
        for p in world.players.values():
            pos = p.get("pos")
            if not pos:
                continue
            mx, my = mmxy(pos)
            mm.create_oval(mx - 2, my - 2, mx + 2, my + 2, fill=PLAYER, outline="")
        if world.self_pos:
            mx, my = mmxy(world.self_pos)
            mm.create_oval(mx - 3, my - 3, mx + 3, my + 3, fill=SELF,
                           outline="#ffffff")

    # -- status / target frames ---------------------------------------------

    def _update_status(self):
        world = self.client.world
        # HP / MP bars
        self._paint_bar_canvas(self.hp_bar, 238, 14,
                               world.self_hp, world.self_max_hp)
        self.hp_text.set(f"HP {world.self_hp or 0}/{world.self_max_hp or 0}")
        mana = world.self_mana
        mp_pct = 1.0 if mana is None else (mana / 100.0 if mana <= 100 else 0.0)
        self._paint_bar_canvas(self.mp_bar, 238, 10, mana if mana is not None else 0, 100)
        self.mp_text.set(f"MP {mana if mana is not None else '-'}")

        self.status_vars["level"].set(str(world.self_level))
        self.status_vars["exp"].set(str(int(world.self_exp)))
        self.status_vars["gp"].set(str(int(world.self_gp)))
        self.status_vars["mana"].set(str(int(world.self_mana or 0)))
        self.status_vars["stamina"].set(str(int(world.self_stamina or 0)))
        pos = world.self_pos
        self.status_vars["pos"].set(
            f"{int(pos[0])},{int(pos[1])}" if pos else "-")
        self.status_vars["map"].set(str(world.map_id))
        self.status_vars["potion"].set(str(world.inventory.get(ITEM_POTION, 0)))

        stats = world.stats
        for key, var in self.stats_vars.items():
            var.set(str(int(stats.get(key, 0))))

        self._update_target_frame()
        self._update_equipment()
        self._update_quest_tracker()

    def _paint_bar_canvas(self, cv, w, h, value, max_value):
        cv.delete("all")
        cv.create_rectangle(0, 0, w, h, fill=HP_BG, outline="")
        try:
            pct = float(value) / float(max_value) if max_value else 0.0
        except (TypeError, ValueError):
            pct = 0.0
        pct = max(0.0, min(1.0, pct))
        if pct > 0:
            color = HP_OK if pct > 0.5 else HP_WARN if pct > 0.25 else HP_LOW
            cv.create_rectangle(0, 0, w * pct, h, fill=color, outline="")

    def _update_target_frame(self):
        world = self.client.world if self.client else None
        if world is None:
            return
        name = "无目标"
        hp_text = ""
        pct = 0.0
        if self.target_kind == "npc" and self.target_id is not None:
            npc = world.npcs.get(self.target_id)
            if npc:
                name = f"{npc.get('name') or '?'}  Lv{npc.get('level', '?')}"
                hp, max_hp = npc.get("hp"), npc.get("max_hp")
                if hp is not None and max_hp:
                    hp_text = f"HP {hp}/{max_hp}"
                    pct = hp / max_hp
                else:
                    hp_text = "HP -/-"
                pos = npc.get("pos")
                if pos and world.self_pos:
                    dist = math.hypot(pos[0] - world.self_pos[0],
                                      pos[1] - world.self_pos[1])
                    name += f"  ({int(dist)}px)"
            else:
                self.target_kind = self.target_id = None
        elif self.target_kind == "player" and self.target_id is not None:
            p = world.players.get(self.target_id)
            if p:
                name = f"{p.get('nick') or '?'}  Lv{p.get('level', '?')}"
                hp, max_hp = p.get("hp"), p.get("max_hp")
                if hp is not None and max_hp:
                    hp_text = f"HP {hp}/{max_hp}"
                    pct = hp / max_hp
            else:
                self.target_kind = self.target_id = None
        elif self.target_kind == "drop" and self.target_id is not None:
            d = world.drops.get(self.target_id)
            if d:
                name = f"掉落 #{self.target_id}"
                hp_text = f"物品 {d.get('item_id')}"
            else:
                self.target_kind = self.target_id = None
        self.target_text.set(name)
        self.target_hp.set(hp_text)
        cv = self.target_bar
        cv.delete("all")
        cv.create_rectangle(0, 0, 238, 10, fill=HP_BG, outline="")
        if pct > 0:
            color = HP_OK if pct > 0.5 else HP_WARN if pct > 0.25 else HP_LOW
            cv.create_rectangle(0, 0, 238 * pct, 10, fill=color, outline="")

    # -- events -------------------------------------------------------------

    def _handle_event(self, ev):
        etype = ev.get("type")
        world = self.client.world if self.client else None

        if etype == "log":
            level = ev.get("level", "INFO")
            self._append_log(ev.get("message", ""), level)
        elif etype == "connected":
            if hasattr(self, "conn_status"):
                self.conn_status.set(f"已连接 peer={ev.get('peer')}")
            if self._pending is not None:
                self._login_status("已连接，登录中...")
                self.runner.submit("login",
                                   account=self._pending["account"],
                                   password=self._pending["password"])
        elif etype == "login_result":
            if ev.get("ok"):
                self._login_status("登录成功，获取角色列表...")
            elif self._pending and self._pending.get("auto_create"):
                self._login_status("登录失败，尝试注册新账号...")
                self.runner.submit("create_account",
                                   account=self._pending["account"],
                                   password=self._pending["password"])
            else:
                self._login_status("登录失败：账号或密码错误", "error")
                self.btn_connect.configure(state="normal")
        elif etype == "create_account_result":
            if ev.get("ok"):
                # The runner already logs in and lists characters after creating
                # the account; nothing more to do here.
                self._login_status("账号已创建，登录中...")
            else:
                self._login_status("账号已存在或注册失败", "error")
                self.btn_connect.configure(state="normal")
        elif etype == "characters":
            chars = ev.get("chars") or []
            if chars:
                nick = self._pending["nick"] if self._pending else ""
                match = next((c for c in chars if c.get("nickname") == nick), None)
                chosen = match or chars[0]
                self._login_status(f"进入角色 {chosen.get('nickname')}...")
                self.runner.submit("connect_character", nick=chosen.get("nickname"))
            elif self._pending and self._pending.get("auto_create"):
                self._login_status("创建角色...")
                self.runner.submit("create_character", nick=self._pending["nick"])
            else:
                self._login_status("该账号没有角色", "error")
                self.btn_connect.configure(state="normal")
        elif etype == "create_character_result":
            if ev.get("ok"):
                self._login_status("角色已创建，进入世界...")
                self.runner.submit("connect_character", nick=ev.get("nick"))
            else:
                self._login_status("创建角色失败", "error")
                self.btn_connect.configure(state="normal")
        elif etype == "in_game":
            if ev.get("ok"):
                self.client = self.runner.client
                self._show_main()
            else:
                self._login_status("进入世界失败", "error")
                self.btn_connect.configure(state="normal")
        elif etype == "error":
            self._append_log(f"错误: {ev.get('message', '?')}", "ERROR")
        elif etype == "disconnected":
            self.conn_status.set("连接已断开")
        elif etype == "system":
            self._handle_system_text(ev.get("text") or "")

        # world events
        if world is None:
            return
        if etype == "warp":
            pos = ev.get("pos")
            if pos:
                self.cam[0], self.cam[1] = pos[0], pos[1]
            self._append_log(f"传送至地图 {ev.get('map_id')} ({int(pos[0])},{int(pos[1])})"
                             if pos else f"传送至地图 {ev.get('map_id')}", "INFO")
        elif etype == "damage":
            pos = ev.get("pos")
            if pos:
                alt = ev.get("alt")
                text = str(ev.get("value", ""))
                if alt == 1:
                    text, color = "闪避", "#8b9aab"
                elif alt == 2:
                    color = "#ff9f43"
                else:
                    color = "#ffffff"
                self._add_effect({"kind": "floater", "pos": tuple(pos),
                                  "text": text, "color": color,
                                  "t0": time.time(), "dur": 0.9})
        elif etype == "exp":
            if ev.get("value"):
                self._append_log(f"获得经验 +{ev['value']} (总计 {ev.get('total')})", "GOOD")
        elif etype == "levelup":
            if ev.get("rid") == world.self_peer:
                pos = ev.get("pos") or world.self_pos
                self._add_effect({"kind": "floater", "pos": tuple(pos),
                                  "text": "LEVEL UP!", "color": "#ffd27a",
                                  "t0": time.time(), "dur": 1.4})
                self._append_log("升级了！", "GOOD")
        elif etype == "death":
            name = ev.get("name") or "目标"
            self._append_log(f"{name} 死亡", "WARNING")
        elif etype == "self_dead":
            self._append_log("你已死亡，请点击【复活】", "ERROR")
        elif etype == "chat":
            channel = ev.get("channel", "?")
            name = ev.get("name", "?")
            text = ev.get("text", "")
            self._append_log(f"[{channel}] {name}: {text}", "CHAT")
        elif etype == "drop_added":
            self._append_log(f"掉落出现 #{ev.get('rid')}", "LOOT")
        elif etype == "auto_potion":
            self._append_log("自动喝药", "INFO")
        elif etype == "autoplay":
            self._append_log(
                f"挂机已更新: 打怪={ev.get('combat')} 拾取={ev.get('loot')} "
                f"喝药={ev.get('potion')} 阈值={ev.get('threshold')}%", "INFO")
        elif etype == "equipment" or etype == "inventory" or etype == "item_equiped":
            self._eq_sig = None  # force the next _update_equipment() to rebuild
        elif etype == "quest":
            qid = ev.get("quest_id")
            state = ev.get("state")
            state_text = self.quests.state_label(state)
            name = self.quests.name(qid)
            if int(state or 0) >= QUEST_STATE_DONE:
                self._append_log(f"【{name}】任务完成！奖励："
                                 f"{self.quests.reward_text(qid)}", "GOOD")
            elif int(state or 0) > QUEST_STATE_INACTIVE:
                self._append_log(f"【{name}】已接取（{state_text}）", "GOOD")
            else:
                self._append_log(f"【{name}】已放弃", "WARNING")
            self._quest_tracker_sig = None
            if self.quest_log_text is not None:
                self._refresh_quest_log()
        elif etype in ("dialog", "dialog_choice"):
            self._show_dialog()
            author = ev.get("author") or ""
            text = ev.get("text") or ""
            self.dlg_author.set(author)
            self.dlg_text.set(text)
            q = self._quest_for_dialog(author, text)
            self.dlg_quest.set(f"任务：{q['name']}" if q else "")
            if etype == "dialog":
                self._set_dialog_choices([])
                self.btn_dlg_next.configure(state="disabled")
            else:
                self._set_dialog_choices(ev.get("choices") or [])
                self.btn_dlg_next.configure(state="disabled")
        elif etype == "dialog_continue":
            self._show_dialog()
            self.dlg_can_continue = True
            self.btn_dlg_next.configure(state="normal")
        elif etype == "dialog_close":
            self.dlg_can_continue = False
            if self.dlg_window is not None:
                self.btn_dlg_next.configure(state="disabled")
                self.dlg_window.withdraw()
            self.dlg_quest.set("")

    def _append_log(self, text, tag="INFO"):
        if not hasattr(self, "log_text"):
            log.info(text)
            return
        self.log_text.configure(state="normal")
        self.log_text.insert("end", text + "\n", tag)
        self.log_lines += 1
        if self.log_lines > 600:
            self.log_text.delete("1.0", "150.0")
            self.log_lines = 450
        self.log_text.see("end")
        self.log_text.configure(state="disabled")

    # -- lifecycle ----------------------------------------------------------

    def _on_close(self):
        try:
            self.runner.stop()
        except Exception:
            pass
        try:
            self.root.destroy()
        except Exception:
            pass

    def run(self):
        self.root.mainloop()


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def parse_args():
    parser = argparse.ArgumentParser(description="SoM Lite Human Client (tkinter GUI)")
    parser.add_argument("url", nargs="?",
                        default=os.environ.get("SOM_WS_URL", "ws://127.0.0.1:27012/ws"),
                        help="WebSocket URL")
    parser.add_argument("--account", default=os.environ.get("SOM_ACCOUNT"),
                        help="Account name")
    parser.add_argument("--password", default=os.environ.get("SOM_PASSWORD"),
                        help="Account password")
    parser.add_argument("--nick", default=os.environ.get("SOM_NICK"),
                        help="Character nickname")
    parser.add_argument("--auto-create", action="store_true",
                        default=os.environ.get("SOM_AUTO_CREATE") == "1",
                        help="Auto-create account and character if missing")
    parser.add_argument("--auto-play", action="store_true",
                        default=os.environ.get("SOM_AUTO_PLAY") == "1",
                        help="Enable auto-play at startup")
    parser.add_argument("--quests",
                        default=os.environ.get("SOM_QUESTS_JSON") or None,
                        help="Path to quests.json (default: quests.json next to "
                             "this script); the quest log/markers are disabled "
                             "when the file is missing")
    parser.add_argument("--debug", action="store_true",
                        help="Enable debug logging")
    return parser.parse_args()


def _default_quests_path():
    return os.path.join(os.path.dirname(os.path.abspath(__file__)), "quests.json")


def main():
    args = parse_args()
    if args.debug:
        logging.getLogger().setLevel(logging.DEBUG)
    if tk is None:
        print(
            "错误：需要 tkinter 才能运行人类精简客户端。\n"
            "  Debian/Ubuntu: sudo apt-get install python3-tk\n"
            "  Alpine:        apk add tk\n"
            "  Fedora:        sudo dnf install python3-tkinter",
            file=sys.stderr,
        )
        sys.exit(1)
    app = Application(args)
    app.run()


if __name__ == "__main__":
    main()
