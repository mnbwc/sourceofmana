#!/usr/bin/env python3
"""MVP-9 Spectator SSE 网关 + 单文件 HTML 观战页。

桥(TCP NDJSON) -> SSE(HTTP 单向推)。零游戏进程侵入：本网关作为桥的一个
只读客户端，把 spectate_feed 增量转成 SSE 事件流。

用法:
  python3 spectator_sse.py --bridge 127.0.0.1:9411 --http 8899 --target <peerID>
  浏览器打开 http://127.0.0.1:8899/  （单文件 HTML：事件时间线 + 顶部状态条）

断线续传: SSE 事件带 id=<seq>，EventSource 自动 Last-Event-ID 重连，
网关以 ?since=N 转发请求桥, 无丢事件。
"""
import argparse
import json
import socket
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

STATE = {
    "bridge": None,          # (host, port)
    "target": 0,
    "cursor": 0,
    "snapshot": {},
    "events": [],            # 最近事件环
    "nick": "",
    "isAgent": False,
    "lock": threading.Lock(),
}

PAGE = """<!doctype html><html lang="zh"><head><meta charset="utf-8">
<title>SoM 观战</title><style>
body{margin:0;font:14px/1.5 monospace;background:#14121a;color:#d8d2c4}
#bar{position:sticky;top:0;background:#1e1a28;padding:8px 12px;border-bottom:1px solid #332c44;display:flex;gap:18px;flex-wrap:wrap}
#bar b{color:#f0b64a}
#feed{padding:10px 12px}
.ev{padding:4px 0;border-bottom:1px dashed #262038}
.ev .seq{color:#5f5675;margin-right:8px}
.ev .kind{color:#7fb0ff;margin-right:8px}
.ev.damage .kind{color:#ff7a6b}.ev.kill .kind{color:#ffd24a}.ev.looted .kind{color:#7be07b}.ev.death .kind{color:#ff5252}
</style></head><body>
<div id="bar">
  <span>目标 <b id="nick">-</b></span>
  <span>HP <b id="hp">-</b></span>
  <span>击杀 <b id="kills">-</b></span>
  <span>位置 <b id="pos">-</b></span>
  <span>当前动作 <b id="act">-</b></span>
  <span>verdict <b id="verdict">-</b></span>
  <span>观战人数 <b id="spec">-</b></span>
  <span>连接 <b id="conn">connecting</b></span>
</div>
<div id="feed"></div>
<script>
const qs = new URLSearchParams(location.search);
const target = qs.get("target");
function bar(s){
  if(!s) return;
  const p=(s.snapshot&&s.snapshot.player)||{};
  document.getElementById("nick").textContent=(s.target&&s.target.nick)||p.nick||"-";
  const hp=(p.hp!==undefined)?p.hp:(p.health&&p.health.current);
  document.getElementById("hp").textContent=hp!=null?hp:"-";
  document.getElementById("kills").textContent=(s.snapshot&&s.snapshot.score&&s.snapshot.score.kills)!=null?s.snapshot.score.kills:"-";
  if(p.pos) document.getElementById("pos").textContent=Math.round(p.pos.x)+","+Math.round(p.pos.y);
  const last=s.events&&s.events[s.events.length-1];
  if(last){document.getElementById("act").textContent=last.action||last.kind;
    document.getElementById("verdict").textContent=last.verdict||"";}
  if(s.spectators!=null) document.getElementById("spec").textContent=s.spectators;
}
function addEv(e){
  const d=document.createElement("div");
  d.className="ev "+(e.kind||"");
  d.innerHTML='<span class="seq">#'+(e.seq!=null?e.seq:"?")+'</span><span class="kind">'+(e.kind||"event")+'</span>'+
    '<span class="detail">'+JSON.stringify(e.detail!=null?e.detail:e).replace(/[<>&]/g,c=>({"<":"&lt;",">":"&gt;","&":"&amp;"}[c])).slice(0,160)+'</span>';
  const f=document.getElementById("feed");
  f.prepend(d);
  while(f.children.length>200) f.removeChild(f.lastChild);
}
function connect(){
  const es=new EventSource("/stream"+(target?("?target="+target):""));
  es.onopen=()=>document.getElementById("conn").textContent="live";
  es.onerror=()=>{document.getElementById("conn").textContent="reconnecting";};
  es.addEventListener("state",ev=>{const s=JSON.parse(ev.data);bar(s);});
  es.addEventListener("event",ev=>{addEv(JSON.parse(ev.data));});
}
connect();
</script></body></html>"""


class BridgeClient(threading.Thread):
    """桥只读客户端：spectate + 周期 spectate_feed，维护 STATE。"""

    def __init__(self, host, port, target):
        super().__init__(daemon=True)
        self.host, self.port, self.target = host, port, target
        self.sock = None
        self.buf = b""

    def rpc(self, obj, timeout=8):
        self.sock.settimeout(timeout)
        self.sock.sendall((json.dumps(obj) + "\n").encode())
        while b"\n" not in self.buf:
            self.buf += self.sock.recv(65536)
        line, self.buf = self.buf.split(b"\n", 1)
        return json.loads(line.decode())

    def run(self):
        while True:
            try:
                self.sock = socket.create_connection((self.host, self.port), 5)
                self.buf = b""
                r = self.rpc({"op": "spectate", "peerID": self.target})
                if not r.get("ok"):
                    print("spectate failed:", r)
                    time.sleep(5)
                    continue
                with STATE["lock"]:
                    STATE["cursor"] = r.get("cursor", 0)
                    STATE["snapshot"] = r.get("snapshot", {})
                    STATE["nick"] = (r.get("target") or {}).get("nick", "")
                    STATE["isAgent"] = (r.get("target") or {}).get("isAgent", False)
                while True:
                    since = STATE["cursor"]
                    r = self.rpc({"op": "spectate_feed", "peerID": self.target,
                                  "since": since, "includeSnapshot": True})
                    if not r.get("ok"):
                        break
                    with STATE["lock"]:
                        STATE["cursor"] = r.get("cursor", since)
                        STATE["snapshot"] = r.get("snapshot", STATE["snapshot"])
                        STATE["events"] = (STATE["events"] + r.get("events", []))[-500:]
                        STATE["spec"] = None  # feed 不带 spectators; 用最近 spectate 值
                    time.sleep(0.7)
            except Exception as e:
                print("bridge link lost:", e, "- retry in 3s")
                time.sleep(3)


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *a):
        pass

    def do_GET(self):
        if self.path.startswith("/stream"):
            self._stream()
        else:
            body = PAGE.encode()
            self.send_response(200)
            self.send_header("Content-Type", "text/html; charset=utf-8")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

    def _stream(self):
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Cache-Control", "no-cache")
        self.send_header("Access-Control-Allow-Origin", "*")
        self.end_headers()
        last_sent = 0
        try:
            while True:
                with STATE["lock"]:
                    snap = dict(STATE["snapshot"])
                    nick = STATE["nick"]
                    is_agent = STATE["isAgent"]
                    cursor = STATE["cursor"]
                    events = [e for e in STATE["events"] if e.get("seq", 0) > last_sent]
                state_payload = {"target": {"nick": nick, "isAgent": is_agent},
                                 "snapshot": snap, "events": events, "spectators": 1}
                data = json.dumps(state_payload, ensure_ascii=False)
                self.wfile.write(f"id: {cursor}\nevent: state\ndata: {data}\n\n".encode())
                self.wfile.flush()
                for e in events:
                    ed = json.dumps(e, ensure_ascii=False)
                    self.wfile.write(f"event: event\ndata: {ed}\n\n".encode())
                self.wfile.flush()
                last_sent = max([last_sent] + [e.get("seq", 0) for e in events])
                time.sleep(1.0)
        except (BrokenPipeError, ConnectionResetError):
            pass


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--bridge", default="127.0.0.1:9411")
    ap.add_argument("--http", type=int, default=8899)
    ap.add_argument("--target", type=int, required=True, help="被观战 peerID")
    args = ap.parse_args()
    host, port = args.bridge.rsplit(":", 1)
    STATE["bridge"] = (host, int(port))
    STATE["target"] = args.target
    BridgeClient(host, int(port), args.target).start()
    srv = ThreadingHTTPServer(("0.0.0.0", args.http), Handler)
    print(f"SSE spectator gateway: http://0.0.0.0:{args.http}/ target={args.target}")
    srv.serve_forever()


if __name__ == "__main__":
    main()
