"""RuleBot 冒烟：SoM harness 适配层 + 桥 G1 close_session 实测（tick9）。

前置：自建 server + AgentClientProbe --agent-keep 在线（外部脚本负责起服/探针）。
本脚本：ping → wait_ready 认领 agent → RuleBot 冒烟 N 动作 → close_session → 断言 peer 消失。
"""
from __future__ import annotations

import json
import os
import sys
import time

sys.path.insert(0, os.path.dirname(__file__))
from sombase import SoMClient, RandomRuleBot, ApiCallCounter  # noqa: E402

BRIDGE_PORT = int(os.environ.get("BRIDGE_PORT", "9101"))
SMOKE_ACTIONS = int(os.environ.get("SMOKE_ACTIONS", "20"))


def main() -> int:
    counter = ApiCallCounter()
    c = SoMClient(port=BRIDGE_PORT, counter=counter)
    ping = c.ping()
    print("PING", json.dumps(ping))
    assert ping.get("ok"), ping

    state = c.wait_ready(timeout=40)
    if state.get("error"):
        print("WAIT_READY-FAIL", state)
        return 2
    print("READY peerID=%s nick=%s pos=%s score=%s" % (
        c.peer_id, state["player"].get("nick"),
        state["player"].get("position"), state.get("score")))

    bot = RandomRuleBot(seed=42)
    cursor = 0
    t0 = time.time()
    for i in range(SMOKE_ACTIONS):
        evs = c.events(cursor)
        cursor = max(cursor, evs.get("cursor", cursor))
        events = [e for e in evs.get("events", []) if e.get("kind") in ("kill", "died", "damaged")]
        state = c.state()
        if state.get("error"):
            print("STATE-ERROR", state)
            return 3
        if not state["player"].get("alive"):
            print("AGENT-DIED before decide (turn %d)" % i)
            break
        action, args = bot.decide(state, events)
        if action == "flee" or not state["player"].get("alive"):
            # 血量低：反向撤 200
            p = state["player"]["position"]
            args = {"x": p["x"] * 2 - (state["nearby"][0]["position"]["x"] if state["nearby"] else p["x"]),
                    "y": p["y"]}
            action = "navigate"
        if args.pop("_approach", None) is not None:
            r = c.act("navigate", **args)
            # 贴近后等导航走完再决策
            time.sleep(2.0)
        else:
            r = c.act(action, **args)
        kills = [e for e in events if e["kind"] == "kill"]
        print("TURN%02d %s %s -> %s %s%s" % (
            i, action, {k: (round(v, 1) if isinstance(v, float) else v) for k, v in args.items()},
            r.get("verdict"), r.get("reason", ""),
            (" KILL " + kills[0].get("monsterNick", "?") + " score=" + json.dumps(c.state().get("score"))) if kills else ""))
        if not state["player"].get("alive"):
            print("AGENT-DIED")
            break

    # close_session（G1）断言：peer 消失
    score = c.state().get("score")
    closed = c.close_session()
    print("CLOSE", json.dumps(closed), "final_score", json.dumps(score))
    time.sleep(1.0)
    c2 = SoMClient(port=BRIDGE_PORT)
    players = c2.rpc("list_players").get("players", [])
    gone = c.peer_id not in [p["peerID"] for p in players]
    c2._close_tcp()
    print("PEER-GONE", gone)
    print("COUNTER", json.dumps(counter.counts), "polls", counter.polls, "elapsed %.1fs" % (time.time() - t0))
    return 0 if (gone and closed.get("ok")) else 4


if __name__ == "__main__":
    sys.exit(main())
