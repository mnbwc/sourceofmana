import json, socket, time, sys

def req(s, obj, timeout=10):
    s.sendall((json.dumps(obj)+"\n").encode())
    s.settimeout(timeout)
    buf=b""
    while b"\n" not in buf:
        d=s.recv(65536)
        if not d: break
        buf+=d
    return json.loads(buf.split(b"\n")[0].decode())

s=socket.create_connection(("127.0.0.1",9101))
print("ping:", req(s,{"op":"ping"}))
pl=req(s,{"op":"list_players"})
print("players:", pl)
pid=pl["players"][0]["peerID"]
snap=req(s,{"op":"snapshot","peerID":pid})
p=snap["snapshot"]["player"]
print("player:", p["position"], "hp:", p["health"])
near=snap["snapshot"].get("nearby",[])
print("nearby count:", len(near))
for n in near[:6]: print("  ", n["type"], n["nick"], round(n["distance"],1), n["alive"], n.get("health"))

# navigate test
pos=p["position"]; tgt={"x":pos["x"]+80,"y":pos["y"]}
r=req(s,{"op":"apply_action","peerID":pid,"name":"navigate","args":tgt})
print("navigate ack:", r)
final=None
for i in range(15):
    time.sleep(0.3)
    v=req(s,{"op":"latest_verdict","peerID":pid})
    if v["verdict"] and v["verdict"].get("verdict") in ("APPLIED","REJECTED","NO_EFFECT","UNVERIFIABLE"):
        final=v; break
    final=v
print("navigate verdict:", final)

snap2=req(s,{"op":"snapshot","peerID":pid})
print("pos after:", snap2["snapshot"]["player"]["position"])

# events drain
ev=req(s,{"op":"drain_events","peerID":pid})
print("events:", ev["ok"], "cursor:", ev.get("cursor"), "n:", len(ev.get("events",[])))

# attack test if monster nearby
mon=[n for n in req(s,{"op":"snapshot","peerID":pid})["snapshot"]["nearby"] if n["type"]=="monster" and n["alive"]]
if mon:
    m=min(mon,key=lambda n:n["distance"])
    print("attacking:", m["rid"], m["nick"], "dist", round(m["distance"],1))
    r=req(s,{"op":"apply_action","peerID":pid,"name":"attack","args":{"rid":m["rid"],"skillID":0,"x":0,"y":0}})
    print("attack ack:", r)
    final=None
    for i in range(20):
        time.sleep(0.3)
        v=req(s,{"op":"latest_verdict","peerID":pid})
        if v["verdict"] and v["verdict"].get("verdict") in ("APPLIED","REJECTED","NO_EFFECT","UNVERIFIABLE"):
            final=v; break
        final=v
    print("attack verdict:", final)
    ev=req(s,{"op":"drain_events","peerID":pid,"since":ev.get("cursor",0)})
    print("post-attack events:", ev.get("events"))
else:
    print("no monster nearby; attack verdict path tested via TARGET_NOT_FOUND:")
    print(req(s,{"op":"apply_action","peerID":pid,"name":"attack","args":{"rid":123456,"skillID":0}}))
print("AGENT-PROBE-DONE")
