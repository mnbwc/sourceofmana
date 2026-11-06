"""SoM LLM policy 插槽（tick17）—— Kaetram LLMPlanner 平移。

OpenAI 兼容 /chat/completions；解析失败/网络失败自动降级 RandomRuleBot
（decision_source 区分 llm / rulebot_fallback:PARSE|NETWORK|NO_KEY）。
观测裁剪：nearby 按距离排序 cap 30（Kaetram F17/F30 同源教训）；
episode 上下文（arena_status 的 template/score）注入 user_msg。
"""
from __future__ import annotations

import hashlib
import json
import os
import time
import urllib.error
import urllib.request

from sombase import RandomRuleBot, persp

SYSTEM_PROMPT = """你是一个 Source of Mana MMO 里的 AI 玩家。可用动作（每次返回 JSON）：
{"reasoning": "...", "action": "navigate", "x": int, "y": int}
或 {"reasoning": "...", "action": "attack", "rid": int}
或 {"reasoning": "...", "action": "loot", "dropID": int}
或 {"reasoning": "...", "action": "say", "text": str}
或 {"reasoning": "...", "action": "wait"}   （原地回血/等动作结束）
或 {"reasoning": "...", "action": "respawn"} （死亡后复活）
规则：attack 只能打射程内目标（persp<15，目标距离见 nearby.distance，太远先 navigate 走过去再打）；血量低于 50% 就 wait 回血；只打 Maggot 等弱怪，绝不打 Scorpion（200HP 会死）。
navigate 技巧：直接把 navigate 的 x/y 设成目标怪/掉落物的坐标（±5 以内即可），不要在射程边缘来回走（服务器会自动走到点）；接近后（distance<15）立即 attack，不要再 navigate。
loot 技巧：掉落物 distance>40 时先 navigate 到它的坐标，<40 再 loot。只返回 JSON。"""


def _fmt_nearby(state: dict, me: dict, cap: int = 30) -> list[dict]:
    nb = state.get("nearby", [])
    px, py = me.get("position", {}).get("x", 0), me.get("position", {}).get("y", 0)
    try:
        nb = sorted(nb, key=lambda e: (e.get("position", {}).get("x", 0) - px) ** 2
                    + (e.get("position", {}).get("y", 0) - py) ** 2)
    except Exception:
        pass
    out = []
    for e in nb[:cap]:
        out.append({
            "type": e.get("type"), "nick": e.get("nick"), "rid": e.get("rid"),
            "x": e.get("position", {}).get("x"), "y": e.get("position", {}).get("y"),
            "hp": (e.get("health") or {}).get("current"), "hpMax": (e.get("health") or {}).get("max"),
            "distance": e.get("distance"),
        })
    return out


class SOMLLMPlanner:
    """decide(state, events) -> (action, kwargs)；失败降级 RuleBot。"""

    def __init__(self, fallback: RandomRuleBot | None = None, model: str | None = None):
        self.api_key = os.environ.get("OPENAI_API_KEY", "")
        self.base_url = os.environ.get("OPENAI_BASE_URL",
                                       "https://discovery-api.intern-ai.org.cn/v1").rstrip("/")
        self.model = model or os.environ.get("SOM_LLM_MODEL", "deepseek-v4-flash-0731")
        self.configured = bool(self.api_key)
        self.fallback = fallback or RandomRuleBot()
        self.last_usage = 0
        self.last_latency_ms = 0.0
        self.system_prompt_hash = hashlib.sha256(SYSTEM_PROMPT.encode()).hexdigest()[:16]

    def decide(self, state: dict, events: list):
        action, kwargs = self._decide(state, events)
        return self._guard(state, action, kwargs)

    def _decide(self, state: dict, events: list):
        if not self.configured:
            return self._fb(state, events, "NO_KEY")
        try:
            return self._decide_llm(state, events)
        except Exception as e:
            return self._fb(state, events, "PARSE" if "PARSE_FALLBACK" in str(e) else "NETWORK",
                            err=str(e))

    def _guard(self, state: dict, action: str, kwargs: dict):
        """tick26 前置距离守卫：射程外 attack / 拾取距离外 loot 客户端侧改写为
        navigate 到目标坐标（服务器会走到点），省一次注定 REJECTED/no-op 的决策。"""
        if action not in ("attack", "loot"):
            return action, kwargs
        rid = int(kwargs.get("rid", kwargs.get("dropID", 0)) or 0)
        tgt = next((e for e in state.get("nearby", []) if int(e.get("rid", -1)) == rid), None)
        if tgt is None:
            return action, kwargs
        dist = float(tgt.get("distance", 0) or 0)
        limit = 15.0 if action == "attack" else 40.0  # attack 射程 / PickupSquaredDistance=48
        pos = tgt.get("position") or {}
        if dist >= limit and pos.get("x") is not None:
            self._reasoning = "[guard] %s dist=%.1f>=%.0f → navigate 目标坐标" % (action, dist, limit)
            return "navigate", {"x": int(pos["x"]), "y": int(pos["y"]),
                                "_approach": rid if action == "attack" else -rid}
        return action, kwargs

    def _fb(self, state, events, why: str, err: str = ""):
        a = self.fallback.decide(state, events)
        self._source = "rulebot_fallback:%s_FALLBACK" % why if why != "NO_KEY" else "rulebot_fallback:NO_KEY"
        self._reasoning = "LLM fallback(%s %s): %s" % (why, err[:120], a[0])
        return a

    def _decide_llm(self, state: dict, events: list):
        me = state.get("player", {})
        ep = state.get("episode")
        goal = ""
        if ep and ep.get("status") == "active":
            tpl = ep.get("template")
            if isinstance(tpl, dict):
                brief, need = tpl.get("brief", ""), tpl.get("killsRequired", "?")
            else:
                brief, need = str(tpl), 3  # 桥 arena_status 只回模板 id 字符串
            goal = "\n当前任务: %s (目标 %s kill，进度 %s)\n" % (
                brief, need, (ep.get("score") or {}).get("kills", 0))
        user_msg = (
            "状态: pos=%s hp=%s alive=%s actionInProgress=%s%s\n"
            "附近实体(按距离排序): %s\n最近事件: %s" % (
                me.get("position"), me.get("health"), me.get("alive"),
                me.get("actionInProgress"), goal,
                json.dumps(_fmt_nearby(state, me), ensure_ascii=False),
                json.dumps([e.get("type") for e in events], ensure_ascii=False)))
        body = json.dumps({
            "model": self.model,
            "messages": [{"role": "system", "content": SYSTEM_PROMPT},
                         {"role": "user", "content": user_msg}],
            "max_tokens": 3000,
            "temperature": 0.2,
            **({"reasoning_effort": "none"} if os.environ.get("LLM_THINKING", "off") == "off" else {}),
        }).encode()
        req = urllib.request.Request(
            self.base_url + "/chat/completions", data=body,
            headers={"Content-Type": "application/json",
                     "Authorization": "Bearer " + self.api_key})
        last_err = None
        data = None
        t0 = time.time()
        # tick26：429/网络指数退避（2s→8s→32s），共 4 次尝试；上限 ~42s vs 此前立即降级
        delays = [2, 8, 32]
        for attempt in range(4):
            try:
                with urllib.request.urlopen(req, timeout=120) as resp:
                    data = json.load(resp)
                break
            except urllib.error.HTTPError as e:
                last_err = e
                if e.code == 429 and attempt < 3:
                    retry_after = e.headers.get("Retry-After") if e.headers else None
                    wait_s = delays[attempt]
                    if retry_after:
                        try:
                            wait_s = float(retry_after)
                        except ValueError:
                            pass
                    time.sleep(wait_s)
                    continue
                break
            except Exception as e:
                last_err = e
                if attempt < 3:
                    time.sleep(delays[attempt])
                    continue
        if data is None:
            raise RuntimeError("NETWORK_FALLBACK: %s" % last_err)
        self.last_latency_ms = (time.time() - t0) * 1000
        usage = data.get("usage", {})
        self.last_usage += usage.get("total_tokens", 0)
        content = data["choices"][0]["message"]["content"]
        try:
            parsed = json.loads(content.strip().removeprefix("```json").removesuffix("```").strip())
        except Exception:
            raise RuntimeError("PARSE_FALLBACK: content=%r" % (content,))
        action = parsed.get("action", "wait")
        kwargs = {k: v for k, v in parsed.items()
                  if k not in ("action", "reasoning")}
        self._source = "llm"
        self._reasoning = str(parsed.get("reasoning", ""))
        return action, kwargs
