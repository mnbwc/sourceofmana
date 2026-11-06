#!/usr/bin/env python3
"""tick24: 多模板批测（RuleBot seed 扫描）— grind_maggot(×10, 600s, deaths≤2) 定案。

前置：自建隔离 server 三元组 + AgentClientProbe --agent-keep 在线（外部负责）。
判据：episode 端到端 completed（kills_required_met），score 权威读数与排行榜一致，
0 SCRIPT ERROR（server log 由外层 grep）。输出 META JSON 供批测汇总。
"""
import json, os, sys, time
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from sombase import SoMClient
from somruntime import AgentRuntime, RunConfig

BRIDGE = int(os.environ.get("BRIDGE", "9341"))
TEMPLATE = os.environ.get("TEMPLATE", "grind_maggot")
SEED = int(os.environ.get("SEED", "42"))
MAX_ACTIONS = int(os.environ.get("MAX_ACTIONS", "150"))
DURATION = float(os.environ.get("DURATION", "560"))

c = SoMClient(port=BRIDGE)
state = c.wait_ready(timeout=40)
assert not state.get("error"), state
pid = c.peer_id
r = c.rpc("arena_start", peerID=pid, template=TEMPLATE)
if not r.get("ok") and r.get("code") == "EPISODE_ACTIVE":
    c.rpc("arena_stop", peerID=pid)
    r = c.rpc("arena_start", peerID=pid, template=TEMPLATE)
print("ARENA_START", json.dumps(r, ensure_ascii=False))
assert r.get("ok"), r
time.sleep(1.5)

cfg = RunConfig(policy="random_rule_bot", maxActions=MAX_ACTIONS,
                duration_s=DURATION, bridge_port=BRIDGE, seed=SEED)
run_id = "run-%s-s%d-%d" % (TEMPLATE, SEED, int(time.time()))
rt = AgentRuntime(os.path.join(os.path.dirname(__file__), "runs", run_id), cfg)
meta = rt.run()

ev_end = None
for l in open(rt.rollouts_path):
    rec = json.loads(l)
    for e in rec["events"]:
        if e == "arena:episode_end":
            ev_end = rec
print("EVENT_END_SAMPLE", ev_end["turn"] if ev_end else None)
print("META", json.dumps({k: meta[k] for k in
      ("final_status", "exhaust_reason", "actions_used", "decide_calls", "polls",
       "kills", "final_score", "duration_s")}, ensure_ascii=False))
# 排行榜回查（tick22 定案的终局权威读法）
lb = os.path.expanduser("~/.local/share/SourceOfMana/arena_leaderboard.jsonl")
last = None
if os.path.exists(lb):
    # 坑（tick25）：排行榜 nick 可能带 ai_ 前缀（探针名经服务端改写），两种形态都试
    cands = {state["player"].get("nick"), "ai_" + str(state["player"].get("nick"))}
    for l in open(lb):
        row = json.loads(l)
        if row.get("template") == TEMPLATE and row.get("nick") in cands:
            last = row
print("LEADERBOARD_LAST", json.dumps(last, ensure_ascii=False) if last else None)
# 判据（tick24 定案）：episode completed 后 arena_status 即无 active（final_score=null、
# harness 侧可能 session_lost/PEER_NOT_FOUND）——终局权威读法=排行榜（tick22 定案）。
ok = bool(last) and last.get("status") == "completed" and last.get("endReason") == "kills_required_met"
print("TEMPLATE-BATCH", "PASS" if ok else "FAIL")
