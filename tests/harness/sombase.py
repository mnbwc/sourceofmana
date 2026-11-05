"""SoM harness 适配层 — AgentBridge TCP/NDJSON 客户端（tick9）。

向 runtime 层暴露与 Kaetram KaetramClient 相同语义的同步接口：
  observe = snapshot / drain_events(cursor)
  act     = apply_action + latest_verdict 轮询（内部封装成同步 verdict）
  生命周期 = close_session（G1，桥侧 FullyDisconnect 同源断开）

Kaetram 经验直迁：cursor 重连续传、verdict 轮询终局（0.3s×20 上限 6s）、
预算/事件唤醒/三件套落盘全部留在 runtime 层不动。
"""
from __future__ import annotations

import json
import socket
import time


class ApiCallCounter:
    """计数器：与 Kaetram harness 同形，供预算/报告复用。"""

    def __init__(self) -> None:
        self.counts: dict[str, int] = {}
        self.polls = 0

    def bump(self, kind: str) -> None:
        self.counts[kind] = self.counts.get(kind, 0) + 1

    @property
    def total(self) -> int:
        return sum(self.counts.values())


class SoMClient:
    """单 TCP 连接，按 peerID 多路。peerID 由 AgentClientProbe（--agent-keep）产生。"""

    VERDICT_POLL_INTERVAL = 0.3
    VERDICT_TIMEOUT = 6.0

    def __init__(self, host: str = "127.0.0.1", port: int = 9101,
                 peer_id: int | None = None, counter: ApiCallCounter | None = None):
        self.host, self.port = host, port
        self.peer_id = peer_id
        self.counter = counter or ApiCallCounter()
        self._sock: socket.socket | None = None
        self._buf = b""

    # ---- low level -------------------------------------------------------
    def _ensure(self) -> socket.socket:
        if self._sock is None:
            self._sock = socket.create_connection((self.host, self.port), timeout=10)
        return self._sock

    def rpc(self, op: str, **kw) -> dict:
        req = {"op": op, **kw}
        if self.peer_id is not None and "peerID" not in req:
            req["peerID"] = self.peer_id
        s = self._ensure()
        s.sendall((json.dumps(req) + "\n").encode())
        self.counter.bump(op)
        while True:
            if b"\n" in self._buf:
                line, self._buf = self._buf.split(b"\n", 1)
                return json.loads(line)
            chunk = s.recv(65536)
            if not chunk:
                raise ConnectionError("bridge closed")
            self._buf += chunk

    # ---- lifecycle -------------------------------------------------------
    def ping(self) -> dict:
        return self.rpc("ping")

    def close_session(self) -> dict:
        """G1：断开游戏 peer（桥侧与人类掉线同源 FullyDisconnect）并关 TCP。"""
        try:
            r = self.rpc("close_session")
        finally:
            self._close_tcp()
        return r

    def _close_tcp(self) -> None:
        if self._sock:
            try:
                self._sock.close()
            except OSError:
                pass
            self._sock = None
        self._buf = b""

    # ---- observe ---------------------------------------------------------
    def state(self) -> dict:
        r = self.rpc("snapshot")
        return r.get("snapshot", {"error": r.get("code", "UNKNOWN")})

    def events(self, since: int = 0) -> dict:
        return self.rpc("drain_events", since=since)

    # ---- act -------------------------------------------------------------
    def act(self, name: str, **args) -> dict:
        """同步语义：apply_action → 轮询 latest_verdict 至终局（或超时）。"""
        r = self.rpc("apply_action", name=name, args=args)
        if not r.get("ok"):
            return {**r, "verdict": "REJECTED"}
        if r.get("verdict") not in (None, "PENDING"):
            return r
        t0 = time.time()
        while time.time() - t0 < self.VERDICT_TIMEOUT:
            time.sleep(self.VERDICT_POLL_INTERVAL)
            v = self.rpc("latest_verdict")
            vd = v.get("verdict")
            if vd is not None and not v.get("provisional", False):
                return {"ok": True, **vd, "basedOnFrame": r.get("basedOnFrame")}
        return {"ok": True, "verdict": "NO_EFFECT", "reason": "verdict_timeout"}

    # ---- convenience -----------------------------------------------------
    def approach(self, pos: dict, min_dist: float = 40.0, max_rounds: int = 10) -> bool:
        """逐步 navigate 贴近目标点，直到 persp<min_dist（技能射程内）。
        e2e 教训：navigate 只到点，attack 需先进入射程（~40），否则静默 no_target_change。"""
        me = self.state().get("player", {}).get("position", {})
        for _ in range(max_rounds):
            d = ((me.get("x", 0) - pos["x"]) ** 2 + (me.get("y", 0) - pos["y"]) ** 2) ** 0.5
            if d < min_dist:
                return True
            r = self.navigate(pos["x"], pos["y"])
            if not r.get("ok"):
                return False
            time.sleep(1.5)
            me = self.state().get("player", {}).get("position", {})
        d = ((me.get("x", 0) - pos["x"]) ** 2 + (me.get("y", 0) - pos["y"]) ** 2) ** 0.5
        return d < min_dist * 2

    def navigate(self, x: float, y: float) -> dict:
        return self.act("navigate", x=x, y=y)

    def attack(self, rid: int, skill: str = "Melee") -> dict:
        return self.act("attack", rid=rid, skill=skill)

    def loot(self, drop_id: int) -> dict:
        return self.act("loot", dropID=drop_id)

    def say(self, text: str) -> dict:
        return self.act("say", text=text)

    def wait_ready(self, timeout: float = 30.0) -> dict:
        """等待 peer 进游戏（snapshot 可用）。peer_id 未知时自动从 list_players 认领。"""
        if self.peer_id is None:
            for _ in range(int(timeout / 0.5)):
                players = self.rpc("list_players").get("players", [])
                agents = [p for p in players if p.get("isAgent")]
                if agents:
                    self.peer_id = int(agents[-1]["peerID"])
                    break
                time.sleep(0.5)
            else:
                return {"ok": False, "error": "no agent peer found"}
        t0 = time.time()
        while time.time() - t0 < timeout:
            s = self.state()
            if not s.get("error"):
                return s
            time.sleep(0.5)
        return self.state()


def persp(me: dict, tgt: dict) -> float:
    """服务器技能射程判定用伪等距距离（SkillCommons.PerspectiveIncrease=(1,1.42)）。"""
    dx, dy = (me.get("x", 0) - tgt["x"]) * 1.0, (me.get("y", 0) - tgt["y"]) * 1.42
    return (dx * dx + dy * dy) ** 0.5


class RandomRuleBot:
    """seed 可复现规则 bot：事件/预算驱动的 hunt 循环（Kaetram 同名平移）。"""

    def __init__(self, seed: int = 42):
        import random
        self.rng = random.Random(seed)
        self.target_rid: int | None = None

    def decide(self, state: dict, events: list) -> tuple[str, dict]:
        me = state.get("player", {})
        hp_cur = (me.get("health") or {}).get("current", 1)
        hp_max = (me.get("health") or {}).get("max", 1) or 1
        if hp_max and hp_cur / hp_max < 0.3:
            return "flee", {}
        # 最近的可打怪
        monsters = [n for n in state.get("nearby", [])
                    if n.get("type") == "monster" and n.get("alive")]
        if self.target_rid:
            tgt = next((m for m in monsters if m["rid"] == self.target_rid), None)
            if tgt is None:
                self.target_rid = None  # 目标没了（可能已死，等 kill 事件确认）
        if self.target_rid is None and monsters:
            # 挑血量上限最弱的怪（Maggot 50hp），否则 Scorpion 208hp 反伤致死新手
            self.target_rid = int(min(monsters, key=lambda m: (m.get("health") or {}).get("max", 9999))["rid"])
        if self.target_rid is not None:
            tgt = next((m for m in monsters if m["rid"] == self.target_rid), None)
            if tgt and persp(me.get("position", {}), tgt["position"]) >= 40:
                return "navigate", {"x": tgt["position"]["x"], "y": tgt["position"]["y"],
                                    "_approach": self.target_rid}
            return "attack", {"rid": self.target_rid}
        # 漫游：附近随机点
        x, y = me.get("position", {}).get("x", 0), me.get("position", {}).get("y", 0)
        return "navigate", {"x": x + self.rng.uniform(-200, 200),
                            "y": y + self.rng.uniform(-200, 200)}
