#!/usr/bin/env python3
"""tick17: LLM policy + arena episode E2E — arena_start → LLM run → episode_end 事件与 score 判定.

前置：自建 server（隔离三元组）+ AgentClientProbe --agent-keep 在线。
判据：episode_start/episode_end 事件进事件流、LLM 决策占比、0 降级或降级有因、0 SCRIPT ERROR。
"""
import json, os, sys, time
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from sombase import SoMClient
from somruntime import AgentRuntime, RunConfig

BRIDGE = int(os.environ.get("BRIDGE", "9341"))
TEMPLATE = os.environ.get("TEMPLATE", "tutorial_maggot")
c = SoMClient(port=BRIDGE)
state = c.wait_ready(timeout=40)
assert not state.get("error"), state
pid = c.peer_id
r = c.rpc("arena_start", peerID=pid, template=TEMPLATE)
if not r.get("ok") and r.get("code") == "EPISODE_ACTIVE":
    c.rpc("arena_stop", peerID=pid)  # 上个 run 遗留 episode：显式收尾后重开
    r = c.rpc("arena_start", peerID=pid, template=TEMPLATE)
print("ARENA_START", json.dumps(r, ensure_ascii=False))
assert r.get("ok"), r
time.sleep(1.5)
snap = c.rpc("snapshot", peerID=pid)["snapshot"]
print("AFTER_WARP_POS", snap["player"]["position"])

# episode 上下文注入观测（桥 snapshot 暂不携带 episode；arena_status 主动取）
orig_state = SoMClient.state
def state_with_episode(self):
    s = orig_state(self)
    ep = c.rpc("arena_status", peerID=self.peer_id).get("episode")
    if ep:
        s["episode"] = ep
    return s
SoMClient.state = state_with_episode

cfg = RunConfig(policy=os.environ.get("POLICY", "llm"),
                maxActions=int(os.environ.get("MAX_ACTIONS", "24")),
                duration_s=float(os.environ.get("DURATION", "420")),
                bridge_port=BRIDGE, seed=7)
rt = AgentRuntime(os.path.join(os.path.dirname(__file__), "runs",
                               "run-%s-arena-%d" % (cfg.policy, int(time.time()))), cfg)
meta = rt.run()
SoMClient.state = orig_state

final = c.rpc("arena_status", peerID=pid)
print("EPISODE_FINAL", json.dumps(final, ensure_ascii=False))
print("META", json.dumps({k: meta[k] for k in
      ("final_status", "exhaust_reason", "actions_used", "decide_calls", "polls",
       "kills", "final_score", "duration_s")}))
# 决策源统计 + arena 事件
from collections import Counter
srcs = Counter()
arena_ev = []
for l in open(rt.rollouts_path):
    rec = json.loads(l)
    srcs[rec["decision_source"]] += 1
    arena_ev += [e for e in rec["events"] if str(e).startswith("arena")]
print("SOURCES", dict(srcs))
print("ARENA_EVENTS", arena_ev)
ev = c.rpc("drain_events", peerID=pid, since=0)
print("ARENA_EV_RAW", json.dumps([e for e in ev.get("events", []) if e.get("kind") == "arena"],
                                  ensure_ascii=False))
c.close_session()
ok = ("arena:episode_start" in arena_ev and "arena:episode_end" in arena_ev
      and srcs.get("llm", 0) > 0)
print("E2E", "PASS" if ok else "FAIL")
