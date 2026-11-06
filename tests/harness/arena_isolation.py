#!/usr/bin/env python3
"""tick16: 多 agent 并发 episode 隔离 E2E — 2 agent 同时 arena_start,
各自独立计分, 一个 stop 一个完成, 互不串扰."""
import sys, os, json, time, socket

BRIDGE = int(os.environ.get("BRIDGE", "9321"))

def conn():
    return socket.create_connection(("127.0.0.1", BRIDGE), timeout=10)

def rpc(s, o, timeout=12):
    s.sendall((json.dumps(o) + "\n").encode()); s.settimeout(timeout)
    buf = b""
    while b"\n" not in buf:
        d = s.recv(65536)
        if not d: raise ConnectionError("closed")
        buf += d
    return json.loads(buf.split(b"\n")[0].decode())

class Agent:
    def __init__(self, nick):
        self.s = conn()
        pl = rpc(self.s, {"op": "list_players"})
        self.pid = next(p["peerID"] for p in pl["players"] if p["nick"] == nick)
        self.nick = nick
        self.kills_seen = set()
    def rpc2(self, o, timeout=12):
        o.setdefault("peerID", self.pid)
        return rpc(self.s, o, timeout)
    def snap(self):
        return self.rpc2({"op": "snapshot"})["snapshot"]
    def events(self, since):
        return self.rpc2({"op": "drain_events", "since": since})

nicks = sys.argv[1:] or None
s0 = conn()
nicks = nicks or [p["nick"] for p in rpc(s0, {"op": "list_players"})["players"]]
s0.close()
A = {n: Agent(n) for n in nicks}
print("agents:", nicks)

# 各自 start 自己的 episode
eps = {}
for n, a in A.items():
    r = a.rpc2({"op": "arena_start", "template": "tutorial_maggot"})
    print(n, "start:", r.get("ok"), r.get("episodeId"))
    eps[n] = r.get("episodeId")
time.sleep(4)
# warp 隔离验证: 位置互同 spawn 但各自独立
for n, a in A.items():
    print(n, "pos:", a.snap()["player"]["position"], "status:", a.rpc2({"op": "arena_status"})["episode"]["status"])

# 并发打怪: 每人独立 bot 循环直到自己的 episode 结束(或 120s)
cursors = {n: 0 for n in nicks}
ends = {}
t0 = time.time()
while time.time() - t0 < 150 and len(ends) < len(nicks):
    for n, a in A.items():
        if n in ends: continue
        ev = a.events(cursors[n]); cursors[n] = ev.get("cursor", cursors[n])
        for e in ev.get("events", []):
            if e.get("kind") == "arena" and e.get("arenaKind") == "episode_end" and e.get("episodeId") == eps[n]:
                ends[n] = e
                print(f"[{n}] EPISODE_END:", e.get("status"), e.get("endReason"), "kills:", e.get("score", {}).get("kills"))
        if n in ends: continue
        st = a.snap()
        p = st["player"]
        if not p.get("alive"):
            a.rpc2({"op": "apply_action", "name": "respawn"})
            time.sleep(0.5); continue
        mon = [x for x in st.get("nearby", []) if x.get("type") == "monster" and x.get("alive") and x.get("nick") == "Maggot"]
        if not mon: continue
        t = min(mon, key=lambda x: x.get("distance", 9999))
        if t["distance"] > 40:
            a.rpc2({"op": "apply_action", "name": "navigate", "args": {"x": t["position"]["x"], "y": t["position"]["y"]}})
        else:
            a.rpc2({"op": "apply_action", "name": "attack", "args": {"rid": t["rid"]}})
    time.sleep(1.2)

# 若有未完成的, stop
for n, a in A.items():
    if n not in ends:
        r = a.rpc2({"op": "arena_stop"})
        ends[n] = {"status": r.get("status"), "endReason": r.get("endReason"), "score": r.get("score")}
        print(f"[{n}] STOPPED:", r.get("status"), r.get("endReason"))

print("\n=== ISOLATION CHECK ===")
ok = True
for n in nicks:
    e = ends.get(n, {})
    k = (e.get("score") or {}).get("kills", 0)
    print(f"{n}: ep={eps[n]} status={e.get('status')} reason={e.get('endReason')} kills={k}")
    if e.get("status") != "completed": ok = False
# 关键断言: 各自 kills=3 (不是 6/互相叠加)
for n in nicks:
    k = (ends.get(n, {}).get("score") or {}).get("kills", 0)
    if k != 3: print(f"FAIL: {n} kills={k} != 3 — 计分串扰!"); ok = False
print("ISOLATION", "PASS" if ok else "FAIL")
