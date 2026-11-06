"""SoM AgentRuntime（tick10）— Kaetram runtime 直迁，I/O 换 sombase.SoMClient。

硬预算闸门 / 事件唤醒 / 三件套落盘（rollouts.jsonl + tools.jsonl + run.meta.json）
与 Kaetram 侧语义一致；事件分类换 SoM 桥事件（damaged/died/kill + snapshot 派生
hp_low / navigation_arrived）。
"""
from __future__ import annotations

import hashlib
import json
import os
import time
from dataclasses import asdict, dataclass, field
from types import SimpleNamespace

from sombase import SoMClient, RandomRuleBot, persp
from somllm import SOMLLMPlanner

HARNESS_VERSION = "0.1.0"
API_VERSION = "sombase-v0.1"
HEARTBEAT_SECONDS = 2.0
LOW_HP_RATIO = 0.30


@dataclass
class Budget:
    maxActions: int = 60
    maxTokens: int = 50000
    expiresAt: float = 0.0

    @classmethod
    def from_seconds(cls, max_actions: int, max_tokens: int, duration_s: float):
        return cls(max_actions, max_tokens, time.time() + duration_s)


@dataclass
class RunConfig:
    policy: str = "random_rule_bot"  # random_rule_bot | llm
    seed: int = 42
    maxActions: int = 30
    maxTokens: int = 50000
    duration_s: float = 300.0
    bridge_host: str = "127.0.0.1"
    bridge_port: int = 9101
    protocol_version: str = ""  # 由 ping 填充，进 configHash
    extra: dict = field(default_factory=dict)

    def hash(self) -> str:
        payload = json.dumps(asdict(self), sort_keys=True, default=str)
        return hashlib.sha256(payload.encode()).hexdigest()


class EventClassifier:
    """SoM 桥事件 → typed harness events（cursor 续传同 Kaetram）。"""

    def __init__(self) -> None:
        self.cursor = 0
        self._was_moving = False
        self._last_hp_ratio = 1.0

    def classify(self, events_resp: dict, state: dict) -> list[dict]:
        out: list[dict] = []
        for e in events_resp.get("events", []):
            self.cursor = max(self.cursor, int(e.get("seq", self.cursor)))
            kind = e.get("kind")
            if kind == "kill":
                out.append({"type": "kill", "monsterNick": e.get("monsterNick", "")})
            elif kind == "died":
                out.append({"type": "died"})
            elif kind == "damaged":
                out.append({"type": "damaged", "value": e.get("value", 0)})
            elif kind == "looted":
                out.append({"type": "looted"})
            elif kind == "arena":
                out.append({"type": "arena:%s" % e.get("arenaKind", "unknown"),
                            "payload": {k: e[k] for k in e
                                        if k in ("status", "endReason", "episodeId", "score", "template", "killsRequired")}})
        player = state.get("player", {})
        hp = (player.get("health") or {})
        hp_cur, hp_max = hp.get("current", 1), hp.get("max", 1) or 1
        ratio = hp_cur / hp_max if hp_max else 1.0
        if ratio < LOW_HP_RATIO and self._last_hp_ratio >= LOW_HP_RATIO:
            out.append({"type": "hp_low", "hp": hp_cur, "maxHp": hp_max})
        self._last_hp_ratio = ratio
        # navigation_arrived：nav.finished 由假变真
        moving = not (player.get("nav") or {}).get("finished", True)
        if self._was_moving and not moving:
            out.append({"type": "navigation_arrived"})
        self._was_moving = moving
        return out


class AgentRuntime:
    """observe → decide → execute 循环；会话由外部探针产生（peerID 自动认领）。"""

    def __init__(self, run_dir: str, config: RunConfig):
        self.run_dir = run_dir
        self.config = config
        os.makedirs(run_dir, exist_ok=True)
        self.rollouts_path = os.path.join(run_dir, "rollouts.jsonl")
        self.tools_path = os.path.join(run_dir, "tools.jsonl")
        self.meta_path = os.path.join(run_dir, "run.meta.json")
        self.counter = SimpleNamespace(polls=0)
        self.classifier = EventClassifier()
        self.budget = Budget.from_seconds(config.maxActions, config.maxTokens, config.duration_s)
        self.started_at = time.time()
        self.actions_used = 0
        self.tokens_used = 0
        self.turn = 0
        self.final_status = "running"
        self.exhaust_reason = ""
        self.episode_id = os.path.basename(run_dir.rstrip("/"))
        self.policy = RandomRuleBot(seed=config.seed)
        # LLM policy 位（tick17）：policy=llm 时用 SOMLLMPlanner（失败自动降级）
        self.llm = None
        if config.policy == "llm":
            self.llm = SOMLLMPlanner(fallback=self.policy)
            if not self.llm.configured:
                print("WARN: OPENAI_API_KEY missing, LLM planner will degrade")
        self.client = SoMClient(host=config.bridge_host, port=config.bridge_port)
        self.final_score: dict | None = None
        self.kills = 0

    # ---- storage ---------------------------------------------------------
    def _append(self, path: str, record: dict) -> None:
        with open(path, "a") as f:
            f.write(json.dumps(record, ensure_ascii=False, default=str) + "\n")

    def _log_tool(self, kind: str, payload: dict) -> None:
        self._append(self.tools_path, {"t": time.time(), "kind": kind, **payload})

    # ---- loop ------------------------------------------------------------
    def run(self) -> dict:
        try:
            ping = self.client.ping()
            self.config.protocol_version = str(ping.get("protocolVersion", ""))
            self._log_tool("ping", ping)
            state = self.client.wait_ready(timeout=40)
            if state.get("error"):
                raise RuntimeError("wait_ready failed: %s" % state)
            self._loop(state)
        finally:
            try:
                self.final_score = self.client.state().get("score")
                # tick19：episode 结果入 meta（arena_status → meta.episode）
                ar = self.client.rpc("arena_status")
                ep = ar.get("episode")
                if ep is None:
                    # 可能已终局（Status 只回 active）——无 id 可查时留 None
                    self.episode_final = None
                else:
                    self.episode_final = ep
            except Exception:
                self.episode_final = None
            try:
                r = self.client.close_session()
                self._log_tool("close_session", r)
            except Exception as e:
                self._log_tool("close_session", {"error": str(e)})
        return self._write_meta()

    def _budget_exceeded(self) -> str:
        if self.actions_used >= self.budget.maxActions:
            return "max_actions"
        if self.budget.expiresAt and time.time() >= self.budget.expiresAt:
            return "expired"
        return ""

    def _loop(self, state: dict) -> None:
        last_heartbeat = 0.0
        while True:
            self.counter.polls += 1
            exhausted = self._budget_exceeded()
            if exhausted:
                self.final_status = "exhausted"
                self.exhaust_reason = exhausted
                break
            events_resp = self.client.events(self.classifier.cursor)
            events = self.classifier.classify(events_resp, state)
            now = time.time()
            has_events = bool(events)
            if not has_events and now - last_heartbeat < HEARTBEAT_SECONDS:
                time.sleep(0.5)
                continue
            last_heartbeat = now
            self.turn += 1
            self.kills += sum(1 for e in events if e["type"] == "kill")

            # tick17 fix: 决策前刷新快照——事件唤醒路径上 state 可能是上一轮末的
            # （navigate 进行中/无怪），LLM 拿到的是陈旧观测。
            if self.llm is not None:
                try:
                    fresh = self.client.state()
                    if not fresh.get("error"):
                        state = fresh
                except Exception:
                    pass

            obs_summary = self._summarize_obs(state)
            decision_source = "rulebot"
            if self.llm is not None:
                action, kwargs = self.llm.decide(state, events)
                decision_source = getattr(self.llm, "_source", "llm")
                obs_summary["decisionSummary"] = getattr(self.llm, "_reasoning", "")
                self.tokens_used = self.llm.last_usage
            else:
                decision = self.policy.decide(state, events)
                action, kwargs = decision if isinstance(decision, tuple) else decision.actions[0]
            verdict = {"ok": False}
            if action == "wait":
                # 脱战回血：不计动作不耗预算，短睡让 regen 走
                time.sleep(3.0)
            elif action == "navigate" and state["player"].get("actionInProgress"):
                # 动作进行中 navigate 必被拒：不耗预算直接跳过（RuleBot 同守卫）
                verdict = {"ok": True, "verdict": "SKIPPED", "reason": "action_in_progress"}
            elif not self._budget_exceeded():
                if action == "flee":
                    # 反向撤 200（血量低）
                    p = state["player"]["position"]
                    nearest = state["nearby"][0]["position"] if state.get("nearby") else p
                    action, kwargs = "navigate", {"x": p["x"] * 2 - nearest.get("x", p["x"]),
                                                  "y": p["y"] * 2 - nearest.get("y", p["y"])}
                kwargs.pop("_approach", None)
                verdict = self.client.act(action, **kwargs)
                self._log_tool("action", {"action": action, "args": kwargs, "verdict": verdict})
                self.actions_used += 1

            self._append(self.rollouts_path, {
                "t": time.time(),
                "episode_id": self.episode_id,
                "turn": self.turn,
                "observation": obs_summary,
                "events": [e["type"] for e in events],
                "decision": {"action": action, "args": kwargs},
                "decision_source": decision_source,
                "verdicts": [verdict],
                "token_usage": self.tokens_used,
            })
            if not state["player"].get("alive", True) and action != "respawn":
                # 死亡：不再终局——respawn（Revive 半血+回出生点）后继续
                self.client.act("respawn")
                time.sleep(3.0)
                self.actions_used += 1
            state = self.client.state()
            if state.get("error"):
                self.final_status = "session_lost"
                self.exhaust_reason = state.get("error")
                break
            self._log_tool("state", {"pos": state["player"].get("position"),
                                     "hp": state["player"].get("health")})

    @staticmethod
    def _summarize_obs(state: dict) -> dict:
        p = state.get("player", {})
        return {
            "pos": p.get("position"),
            "hp": p.get("health"),
            "alive": p.get("alive"),
            "nearby_count": len(state.get("nearby", [])),
        }

    def _write_meta(self) -> dict:
        polls = self.counter.polls
        decide = self.turn
        meta = {
            "configHash": self.config.hash(),
            "apiVersion": API_VERSION,
            "harnessVersion": HARNESS_VERSION,
            "protocolVersion": self.config.protocol_version,
            "model_fingerprint": hashlib.sha256(
                (("llm:" + (self.llm.model if self.llm else "")) if self.llm
                 else "rulebot(seed=%d)" % self.config.seed).encode()).hexdigest()[:16],
            "system_prompt_hash": (self.llm.system_prompt_hash if self.llm else None),
            "started_at": self.started_at,
            "budget": asdict(self.budget),
            "final_status": self.final_status,
            "exhaust_reason": self.exhaust_reason,
            "policy": self.config.policy,
            "degraded_to_rulebot": False,
            "actions_used": self.actions_used,
            "decide_calls": decide,
            "polls": polls,
            "wakeup_reduction_vs_polls": round(1 - decide / polls, 4) if polls else None,
            "metric_baseline": "every_poll_observe",
            "cost_proxy": "tokens_used_includes_reasoning",
            "api_calls": dict(self.client.counter.counts),
            "api_calls_total": self.client.counter.total,
            "tokens_used": self.tokens_used,
            "kills": self.kills,
            "final_score": self.final_score,
            "episode_final": getattr(self, "episode_final", None),
            "duration_s": round(time.time() - self.started_at, 2),
        }
        with open(self.meta_path, "w") as f:
            json.dump(meta, f, ensure_ascii=False, indent=2, default=str)
        return meta
