#!/usr/bin/env python3
"""Soak: N agents concurrent hunt-kill on one server; sample server RSS/CPU; count loot/kills.
Env: BRIDGE_PORT (default 9305), DURATION (default 180s)."""
import json, socket, time, os, subprocess, sys

PORT = int(os.environ.get("BRIDGE_PORT", "9305"))
DURATION = int(os.environ.get("DURATION", "180"))
PID = os.environ.get("SERVER_PID", "")

def conn():
    s = socket.create_connection(("127.0.0.1", PORT), timeout=10)
    return s

def make_session(nick):
    s = conn()
    pl = rpc(s, {"op": "list_players"})
    pid = next(p["peerID"] for p in pl["players"] if p["nick"] == nick)
    return (s, pid)

def rpc(s, obj, timeout=10):
    if isinstance(s, tuple):
        obj.setdefault("peerID", s[1])
        s = s[0]
    s.sendall((json.dumps(obj) + "\n").encode()); s.settimeout(timeout)
    buf = b""
    while b"\n" not in buf:
        d = s.recv(65536)
        if not d: raise ConnectionError("closed")
        buf += d
    return json.loads(buf.split(b"\n")[0].decode())

def make_session(nick):
    s = conn()
    pl = rpc(s, {"op": "list_players"})
    pid = next(p["peerID"] for p in pl["players"] if p["nick"] == nick)
    return (s, pid)

def server_stats():
    if not PID: return None
    try:
        out = subprocess.check_output(["ps", "-o", "rss=,pcpu=", "-p", PID]).decode().split()
        return {"rss_mb": int(out[0]) / 1024, "cpu": float(out[1])}
    except Exception:
        return None

def hunt_step(s):
    st = rpc(s, {"op": "snapshot"}).get("snapshot", {})
    p = st.get("player") or {}
    if not p.get("alive"):
        r = rpc(s, {"op": "apply_action", "name": "respawn"})
        return ("respawn", p)
    nb = st.get("nearby") or []
    mon = [n for n in nb if n.get("type") == "monster" and n.get("alive")]
    if not mon:
        return ("roam", p)
    t = min(mon, key=lambda n: n.get("distance", 9999))
    if t["distance"] > 40:
        rpc(s, {"op": "apply_action", "name": "navigate", "args": {"x": t["position"]["x"], "y": t["position"]["y"]}})
        return ("nav", p)
    rpc(s, {"op": "apply_action", "name": "attack", "args": {"rid": t["rid"]}})
    return ("atk", p)

def main():
    nicks = [a for a in sys.argv[1:]]
    if not nicks:
        s0 = conn()
        nicks = [p["nick"] for p in rpc(s0, {"op": "list_players"})["players"]]
        s0.close()
    sessions = [make_session(n) for n in nicks]
    print("sessions:", nicks)
    stats = {n: {"acts": 0, "kills": 0, "deaths": 0, "loot": []} for n in nicks}
    rss = []
    t0 = time.time()
    while time.time() - t0 < DURATION:
        for s, n in zip(sessions, nicks):
            try:
                ev = rpc(s, {"op": "drain_events", "since": stats[n].get("cursor", 0)})
                cur = ev.get("cursor", stats[n].get("cursor", 0))
                stats[n]["cursor"] = cur
                for e in ev.get("events", []):
                    k = e.get("kind")
                    if k == "kill": stats[n]["kills"] += 1
                    if k == "died": stats[n]["deaths"] += 1
                    if k in ("loot", "item_gained", "drop"): stats[n]["loot"].append(e)
            except Exception as ex:
                print(n, "ev err", ex)
            try:
                act, p = hunt_step(s)
                stats[n]["acts"] += 1
            except Exception as ex:
                print(n, "act err", ex)
        st = server_stats()
        if st: rss.append((time.time(), st["rss_mb"], st["cpu"]))
        time.sleep(1.5)
    print("\n=== SOAK RESULT ===")
    for n in nicks:
        s = stats[n]
        print(f"{n}: acts={s['acts']} kills={s['kills']} deaths={s['deaths']} loot_events={len(s['loot'])}")
    if rss:
        vals = sorted(r[1] for r in rss)
        cpus = sorted(r[2] for r in rss)
        p95 = vals[int(len(vals) * 0.95) - 1] if len(vals) >= 20 else vals[-1]
        cp95 = cpus[int(len(cpus) * 0.95) - 1] if len(cpus) >= 20 else cpus[-1]
        print(f"server RSS p95={p95:.0f}MB  CPU p95={cp95:.1f}%  samples={len(rss)}  growth={vals[-1]-vals[0]:.0f}MB")

if __name__ == "__main__":
    main()
