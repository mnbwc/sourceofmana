#!/usr/bin/env python3
"""tick15: arena warp 完整 episode E2E — start -> bot 杀怪 x3 -> episode_end(completed)."""
import sys, time, json, subprocess, os, socket
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from sombase import SoMClient

PORT=int(os.environ.get("PORT","7325")); BRIDGE=int(os.environ.get("BRIDGE","9325"))

c = SoMClient(port=BRIDGE)
st = c.wait_ready()
pid = c.peer_id
print("PEER", pid)

# find peer id via players list
pl = c.rpc("list_players")
print("PLAYERS", pl)
pid = pl["players"][0]["peerID"]

snap0 = c.rpc("snapshot", peerID=pid)
print("BEFORE_POS", snap0["snapshot"]["player"]["position"])

r = c.rpc("arena_start", peerID=pid, template="tutorial_maggot")
print("ARENA_START", json.dumps(r, ensure_ascii=False))
assert r.get("ok"), r

time.sleep(1.5)
snap = c.rpc("snapshot", peerID=pid)
print("AFTER_POS", snap["snapshot"]["player"]["position"])

# bot loop: kill 3 maggots
deadline = time.time() + 200
kills_done = 0
cur = snap["snapshot"]
while time.time() < deadline:
    st2 = c.rpc("arena_status", peerID=pid)
    ep = st2.get("episode")
    if ep is None or ep.get("status") != "active":
        print("EPISODE_FINISHED", json.dumps(st2, ensure_ascii=False))
        break
    cur = c.rpc("snapshot", peerID=pid)["snapshot"]
    me = cur["player"]
    tgt = None
    for m in cur.get("nearby", []):
        if m.get("type") == "monster" and m.get("nick") == "Maggot":
            tgt = m; break
    if tgt is None:
        c.act("navigate", x=2230, y=1350)
        continue
    d = ((me["position"]["x"]-tgt["position"]["x"])**2 + (me["position"]["y"]-tgt["position"]["y"])**2) ** 0.5
    if d > 45:
        c.act("navigate", x=tgt["position"]["x"], y=tgt["position"]["y"])
    else:
        r = c.act("attack", rid=tgt.get("rid") or tgt.get("id"))
        if "target_died" in json.dumps(r):
            kills_done += 1
    time.sleep(0.4)

ev = c.rpc("drain_events", peerID=pid)
arena_ev = [e for e in ev.get("events", []) if e.get("kind") == "arena"]
print("ARENA_EVENTS", json.dumps(arena_ev, ensure_ascii=False))
print("KILLS_OBSERVED", kills_done)
c.close_session()
print("T15-E2E-DONE")
