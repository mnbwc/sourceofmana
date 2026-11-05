import json, socket, time, math

def req(s, obj, timeout=10):
    for attempt in range(3):
        s.sendall((json.dumps(obj)+"\n").encode())
        s.settimeout(timeout)
        buf=b""
        while b"\n" not in buf:
            d=s.recv(65536)
            if not d: break
            buf+=d
        line=buf.split(b"\n")[0]
        if line.strip():
            return json.loads(line.decode())
        time.sleep(2.0)
    raise RuntimeError("empty bridge response for "+str(obj.get("op")))

def wait_verdict(s, pid, tries=20):
    v=None
    for i in range(tries):
        time.sleep(0.3)
        v=req(s,{"op":"latest_verdict","peerID":pid})
        if v["verdict"] and v["verdict"].get("verdict") in ("APPLIED","REJECTED","NO_EFFECT","UNVERIFIABLE"):
            return v
    return v

def persp(a, b):
    return math.hypot(a["x"]-b["x"], (a["y"]-b["y"])*1.42)

import os as _os
BRIDGE_PORT=int(_os.environ.get("BRIDGE_PORT","9101"))
s=socket.create_connection(("127.0.0.1",BRIDGE_PORT))
pid=req(s,{"op":"list_players"})["players"][0]["peerID"]
snap=req(s,{"op":"snapshot","peerID":pid})["snapshot"]
# 先走到 Maggot 刷新区（2228,1259 附近），否则附近只有 Scorpion
p0=snap["player"]["position"]
if abs(p0["x"]-2228)>400 or abs(p0["y"]-1259)>400:
    req(s,{"op":"apply_action","peerID":pid,"name":"navigate","args":{"x":2228,"y":1350}})
    for i in range(40):
        time.sleep(1.0)
        snap=req(s,{"op":"snapshot","peerID":pid})["snapshot"]
        p1=snap["player"]["position"]
        if abs(p1["x"]-2228)<250 and abs(p1["y"]-1350)<250:
            print("arrived maggot zone", p1); break
    mon=[n for n in snap["nearby"] if n["type"]=="monster" and n["alive"]]
    print("nearby now:", [(n["nick"], n["health"]) for n in mon][:6])
mon=[n for n in snap["nearby"] if n["type"]=="monster" and n["alive"] and n["health"]["max"]<=60]
if not mon:
    print("no weak monster nearby"); raise SystemExit(1)
m=min(mon,key=lambda n:n["health"]["max"])
print("hunt:", m["nick"], m["health"])

# 逐步贴近直到 persp<40
for i in range(6):
    snap=req(s,{"op":"snapshot","peerID":pid})["snapshot"]
    mons=[n for n in snap["nearby"] if n["rid"]==m["rid"] and n["alive"]]
    if not mons: break
    mm=mons[0]
    d=persp(snap["player"]["position"], mm["position"])
    print(f"approach {i}: persp={d:.0f}")
    if d < 40: break
    req(s,{"op":"apply_action","peerID":pid,"name":"navigate","args":{"x":mm["position"]["x"],"y":mm["position"]["y"]}})
    time.sleep(2.0)

cursor=0
kills=0
for rnd in range(12):
    snap=req(s,{"op":"snapshot","peerID":pid})["snapshot"]
    mons=[n for n in snap["nearby"] if n["rid"]==m["rid"] and n["alive"]]
    if not mons:
        print("KILLED!"); kills=1; break
    mm=mons[0]
    d=persp(snap["player"]["position"], mm["position"])
    if d >= 45:
        req(s,{"op":"apply_action","peerID":pid,"name":"navigate","args":{"x":mm["position"]["x"],"y":mm["position"]["y"]}})
        time.sleep(1.8)
        continue
    r=req(s,{"op":"apply_action","peerID":pid,"name":"attack","args":{"rid":m["rid"],"skillID":0}})
    if not r.get("ok"):
        print("attack err:", r); break
    v=wait_verdict(s,pid)
    ev=req(s,{"op":"drain_events","peerID":pid,"since":cursor})
    cursor=ev.get("cursor",cursor)
    print(f"rnd {rnd}: verdict={v['verdict'].get('verdict')}({v['verdict'].get('reason')}) events={[(e['kind'], e.get('value', e.get('monsterID'))) for e in ev.get('events',[])]}")
    time.sleep(0.8)

# 最终事件 flush
ev=req(s,{"op":"drain_events","peerID":pid,"since":cursor})
print("final events:", [(e['kind'],e.get('value')) for e in ev.get('events',[])])
snap=req(s,{"op":"snapshot","peerID":pid})["snapshot"]
print("me:", snap["player"]["health"], "alive:", snap["player"]["alive"])
print("CLOSE-E2E-DONE")
