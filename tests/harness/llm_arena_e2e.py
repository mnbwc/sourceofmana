#!/usr/bin/env python3
"""tick17: LLM policy + arena episode E2E — arena_start(warp 到 Maggot 区) → LLM run → episode_end 判定."""
import json, os, sys, time
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from sombase import SoMClient
from somruntime import AgentRuntime, RunConfig
from somllm import SOMLLMPlanner

BRIDGE = int(os.environ.get("BRIDGE", "9341"))
c = SoMClient(port=BRIDGE)
state = c.wait_ready(timeout=40)
assert not state.get("error"), state
pid = c.peer_id
r = c.rpc("arena_start", peerID=pid, template="tutorial_maggot")
print("ARENA_START", json.dumps(r, ensure_ascii=False))
assert r.get("ok"), r
time.sleep(1.5)
snap = c.rpc("snapshot", peerID=pid)["snapshot"]
print("AFTER_WARP_POS", snap["player"]["position"])
# 把 episode 上下文注入观测（桥 snapshot 暂不携带；planner 读 state['episode']）
orig_state = SoMClient.state
def state_with_episode(self):
    s = orig_state(self)
    st = c.rpc("arena_status", peerID=self.peer_id).get("episode")
    if st:
        s["episode"] = st
    return s
SoMClient.state = state_with_episode

cfg = RunConfig(policy="llm", maxActions=int(os.environ.get("MAX_ACTIONS", "14")),
                duration_s=float(os.environ.get("DURATION", "360")), bridge_port=BRIDGE, seed=7)
rt = AgentRuntime(os.path.join(os.path.dirname(__file__), "runs",
                               "run-llm-arena-%d" % int(time.time())), cfg)
meta = rt.run()
SoMClient.state = orig_state
# 终局：episode 状态
final = c.rpc("arena_status", peerID=pid)
print("EPISODE_FINAL", json.dumps(final, ensure_ascii=False))
print("META", json.dumps({k: meta[k] for k in
      ("final_status", "exhaust_reason", "actions_used", "decide_calls", "polls",
       "kills", "final_score", "duration_s")}))
ev = c.rpc("drain_events", peerID=pid)
arena_ev = [e for e in ev.get("events", []) if e.get("kind") == "arena"]
print("ARENA_EVENTS", json.dumps(arena_ev, ensure_ascii=False))
c.close_session()
