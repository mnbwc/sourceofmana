# Source of Mana — Go 服务器部署交接文档

> 版本：2026-11-10
> 分支：`feat/agent-arena`
> 仓库：https://github.com/mnbwc/sourceofmana

## 1. 项目概述

Source of Mana (SoM) 是一个 2D MMORPG，Godot 4.7.1 引擎。本项目将其服务端从 GDScript 迁移到 Go，实现：

- **WebSocket 游戏服务器**（:27012）— 支持原生 PC/安卓客户端
- **WSS 安全 WebSocket**（:27013）— 支持浏览器客户端
- **AgentBridge**（:9101/9102）— AI agent 探索接口（TCP NDJSON）
- **SQLite 持久化** — 完整 SoM 数据库 schema（12 表 + 3 触发器）
- **竞技场** — 可配置的怪物击杀挑战
- **完整游戏玩法** — 登录/世界同步/NPC/组队/交易/战斗/物品/Buff

### 技术栈

| 组件 | 技术 |
|------|------|
| 服务器 | Go 1.24+，纯 Go（无 CGO） |
| WebSocket | github.com/gorilla/websocket |
| SQLite | modernc.org/sqlite（纯 Go 驱动） |
| TLS | 自签名证书（开发）/ Let's Encrypt（生产） |
| 客户端 | Godot 4.7.1（PC/Android/Web 导出） |

### 测试覆盖

| 测试 | 数量 | 状态 |
|------|------|------|
| Go 单元测试 | 72 | ✅ 全部通过 |
| e2e 测试（Godot 客户端） | 12 | ✅ 全部通过 |
| 浏览器 WebSocket | 1 | ✅ 通过 |

---

## 2. 系统要求

### 服务器

| 项目 | 最低要求 | 推荐 |
|------|---------|------|
| OS | Ubuntu 20.04 / Debian 11 / Alpine 3.18 | Ubuntu 24.04 / Alpine 3.21 |
| CPU | 1 核 | 2 核 |
| 内存 | 512 MB | 2 GB |
| 磁盘 | 1 GB | 10 GB SSD |
| Go | 1.24+ | 1.24+ |
| 网络 | 公网 IP | 公网 IP + 域名 |

### 客户端

| 平台 | 要求 |
|------|------|
| PC | Godot 4.7.1 原生导出（Windows/Linux/macOS） |
| Android | Godot 4.7.1 Android 导出 |
| 网页 | 现代浏览器（Chrome/Firefox/Safari/Edge） |

---

## 3. 一键部署

### 3.1 服务器一键启动

```bash
# 克隆仓库
git clone -b feat/agent-arena https://github.com/mnbwc/sourceofmana.git
cd sourceofmana/goserver

# 构建
go build -o goserver .

# 启动（开发模式 — 监听所有接口）
./goserver

# 启动（生产模式 — 只监听本地回环，通过 nginx 暴露）
BIND_IP=127.0.0.1 WS_PORT=27012 WSS_PORT=27013 BRIDGE_PORT=9102 ./goserver
```

### 3.2 完整生产部署脚本

```bash
#!/bin/bash
# deploy.sh — 一键部署 SoM Go 服务器 + nginx + Let's Encrypt
# 使用方法：sudo bash deploy.sh your-domain.com

set -e
DOMAIN=${1:-localhost}

echo "=== SoM Go 服务器一键部署 ==="
echo "域名: $DOMAIN"

# 1. 安装依赖
echo "[1/5] 安装依赖..."
apt-get update -qq
apt-get install -y -qq nginx certbot python3-certbot-nginx git golang-go

# 2. 克隆代码
echo "[2/5] 克隆代码..."
cd /opt
if [ -d sourceofmana ]; then
    cd sourceofmana && git pull
else
    git clone -b feat/agent-arena https://github.com/mnbwc/sourceofmana.git
    cd sourceofmana/goserver
fi

# 3. 构建 Go 服务器
echo "[3/5] 构建 Go 服务器..."
cd /opt/sourceofmana/goserver
go build -o goserver .

# 4. 配置 systemd 服务
echo "[4/5] 配置 systemd..."
cat > /etc/systemd/system/som-goserver.service << EOF
[Unit]
Description=Source of Mana Go Server
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/sourceofmana/goserver
ExecStart=/opt/sourceofmana/goserver/goserver
Restart=always
RestartSec=5
Environment="BIND_IP=127.0.0.1"
Environment="WS_PORT=27012"
Environment="WSS_PORT=27013"
Environment="BRIDGE_PORT=9102"
Environment="SOM_DB_FILE=/opt/sourceofmana/goserver/som.db"

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable som-goserver
systemctl start som-goserver

# 5. 配置 nginx + Let's Encrypt
echo "[5/5] 配置 nginx..."
mkdir -p /var/www/som
cat > /etc/nginx/sites-available/som << 'EOF'
server {
    listen 443 ssl http2;
    server_name DOMAIN_PLACEHOLDER;

    ssl_certificate     /etc/letsencrypt/live/DOMAIN_PLACEHOLDER/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/DOMAIN_PLACEHOLDER/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         HIGH:!aNULL:!MD5;

    # WebSocket 代理
    location /ws {
        proxy_pass http://127.0.0.1:27012;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 86400;
        proxy_send_timeout 86400;
    }

    # 健康检查
    location /health {
        proxy_pass http://127.0.0.1:27012;
        proxy_set_header Host $host;
    }

    # Godot Web 导出静态文件
    location / {
        root /var/www/som;
        index index.html;
        try_files $uri $uri/ =404;
    }
}

server {
    listen 80;
    server_name DOMAIN_PLACEHOLDER;
    return 301 https://$server_name$request_uri;
}
EOF
sed -i "s/DOMAIN_PLACEHOLDER/$DOMAIN/g" /etc/nginx/sites-available/som
ln -sf /etc/nginx/sites-available/som /etc/nginx/sites-enabled/som

# 获取 Let's Encrypt 证书
certbot --nginx -d $DOMAIN --non-interactive --agree-tos -m admin@$DOMAIN

# 重启 nginx
systemctl restart nginx

echo ""
echo "=== 部署完成 ==="
echo "服务器状态: systemctl status som-goserver"
echo "查看日志: journalctl -u som-goserver -f"
echo "游戏地址: https://$DOMAIN/"
echo "WebSocket: wss://$DOMAIN/ws"
```

### 3.3 Alpine Linux 部署

```bash
# 克隆仓库
git clone -b feat/agent-arena https://github.com/mnbwc/sourceofmana.git
cd sourceofmana/goserver

# 运行 Alpine 部署脚本
sudo sh deploy_alpine.sh your-domain.com
```

Alpine 部署脚本 (`deploy_alpine.sh`) 自动完成：
1. 安装依赖（apk: nginx, git, go, sqlite, openssl）
2. 克隆/更新代码
3. 构建 Go 服务器（`CGO_ENABLED=0`，静态二进制）
4. 生成 TLS 自签名证书（ECDSA P-256）
5. 配置 nginx + WebSocket 代理
6. 配置 OpenRC 服务（`rc-service` / `rc-update`）
7. 部署 Web 客户端到 `/var/www/som/game/`

**Alpine 与 Ubuntu/Debian 的区别：**

| 项目 | Ubuntu/Debian | Alpine |
|------|--------------|--------|
| 包管理 | `apt-get` | `apk` |
| 服务管理 | `systemctl` | `rc-service` / `rc-update` |
| libc | glibc | musl |
| CGO | 默认启用 | `CGO_ENABLED=0`（纯静态） |
| nginx 配置 | `/etc/nginx/sites-available/` | `/etc/nginx/nginx.conf` |
| 防火墙 | `ufw` | `iptables` / `nftables` |
| 日志 | `journalctl` | `logread` |

### 3.4 快速验证

```bash
# 检查服务状态
systemctl status som-goserver

# 检查端口监听
ss -tln | grep -E ':(27012|27013|9102)'

# 测试健康检查
curl http://127.0.0.1:27012/health

# 测试 WebSocket
python3 -c "
import asyncio, websockets
async def test():
    async with websockets.connect('ws://127.0.0.1:27012/ws') as ws:
        msg = await asyncio.wait_for(ws.recv(), timeout=5)
        print(f'WebSocket OK: unique_id={int.from_bytes(msg, \"little\")}')
asyncio.run(test())
"

# 测试 AgentBridge
python3 -c "
import json, socket
s = socket.create_connection(('127.0.0.1', 9102))
s.sendall(b'{\"op\":\"ping\"}\n')
buf = b''
while not buf.endswith(b'\n'):
    buf += s.recv(65536)
print('AgentBridge OK:', json.loads(buf.decode().strip()))
"
```

---

## 4. 网络架构

### 4.1 端口清单

| 端口 | 协议 | 绑定 | 用途 | 暴露 |
|------|------|------|------|------|
| 443 | TCP | 公网 | nginx HTTPS/WSS | ✅ 公网 |
| 80 | TCP | 公网 | HTTP→HTTPS 重定向 | ✅ 公网 |
| 27012 | TCP | 127.0.0.1 | Go WebSocket | ❌ 仅本地 |
| 27013 | TCP | 127.0.0.1 | Go WSS (自签名) | ❌ 仅本地 |
| 9102 | TCP | 127.0.0.1 | AgentBridge | ❌ 仅本地 |

### 4.2 NAT 映射

**只需映射 3 个公网端口：**

| 公网端口 | 协议 | 指向 VPS 内部 | 说明 |
|---------|------|-------------|------|
| **11443** | TCP | 443 | 人类网页登录 (WSS) |
| **27012** | TCP | 27012 | AI 无头客户端登录 (WS) |
| **20080** | TCP | 80 | HTTP→HTTPS 重定向 |

> 如果使用标准端口（80/443），则只需映射 80 和 443。

### 4.3 数据流

```
浏览器 / PC / 安卓
    │
    │ wss://domain/ws (443)
    ▼
┌─────────┐
│  nginx  │  ← TLS 终止 + 静态文件
│  :443   │
└────┬────┘
     │ ws://127.0.0.1:27012
     ▼
┌─────────────┐
│ Go 服务器   │
│ :27012 (ws) │  ← 游戏逻辑 + SQLite
│ :9102 (bridge)│ ← AgentBridge
└─────────────┘
```

---

## 5. 配置参考

### 5.1 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `BIND_IP` | 空（所有接口） | 绑定 IP 地址，生产环境设为 `127.0.0.1` |
| `WS_PORT` | 27012 | 明文 WebSocket 端口 |
| `WSS_PORT` | 27013 | TLS WebSocket 端口（留空则不启动） |
| `BRIDGE_PORT` | 9101 | AgentBridge 端口 |
| `TLS_CERT_FILE` | cert.pem | TLS 证书路径（生产用 Let's Encrypt） |
| `TLS_KEY_FILE` | key.pem | TLS 私钥路径 |
| `SOM_DB_FILE` | som.db | SQLite 数据库路径 |
| `SOM_STORE_FILE` | som_store.json | JSON 迁移源（仅首次启动） |

### 5.2 systemd 配置 (Ubuntu/Debian)

```ini
[Unit]
Description=Source of Mana Go Server
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/sourceofmana/goserver
ExecStart=/opt/sourceofmana/goserver/goserver
Restart=always
RestartSec=5
Environment="BIND_IP=127.0.0.1"
Environment="WS_PORT=27012"
Environment="WSS_PORT=27013"
Environment="BRIDGE_PORT=9102"
Environment="SOM_DB_FILE=/opt/sourceofmana/goserver/som.db"

[Install]
WantedBy=multi-user.target
```

### 5.3 OpenRC 配置 (Alpine)

```sh
# /etc/init.d/som-goserver
#!/sbin/openrc-run

description="Source of Mana Go Server"
command="/opt/sourceofmana/goserver/goserver"
command_background="yes"
command_user="root"
pidfile="/run/${RC_SVCNAME}.pid"

depend() {
    need net
    after firewall
}

start_pre() {
    checkpath -d -m 0755 -o root /opt/sourceofmana/goserver
}
```

```sh
# /etc/conf.d/som-goserver
BIND_IP=0.0.0.0
WS_PORT=27012
WSS_PORT=27013
BRIDGE_PORT=9102
SOM_DB_FILE=/opt/sourceofmana/goserver/som.db
```

### 5.4 nginx 配置

```nginx
server {
    listen 443 ssl http2;
    server_name som.manasource.org;

    ssl_certificate     /etc/letsencrypt/live/som.manasource.org/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/som.manasource.org/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;

    location /ws {
        proxy_pass http://127.0.0.1:27012;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 86400;
    }

    location /health {
        proxy_pass http://127.0.0.1:27012;
    }

    location / {
        root /var/www/som;
        index index.html;
    }
}

server {
    listen 80;
    server_name som.manasource.org;
    return 301 https://$server_name$request_uri;
}
```

---

## 6. 客户端连接

### 6.1 网页版

1. 在 Godot 编辑器中导出 Web 版本（`Project > Export > Web`）
2. 将导出文件复制到 `/var/www/som/`
3. 浏览器访问 `https://your-domain/`
4. 客户端自动连接 `wss://your-domain/ws`

### 6.2 PC/安卓版

客户端默认使用 WebSocket 连接。如需指定服务器：

```bash
# PC 客户端
./sourceofmana --server-address your-domain --ws-port 443

# 或修改 NetworkCommons.gd 中的默认地址
# ServerAddressTesting = "your-domain"
# WebSocketPortTesting = 443
```

### 6.3 URL 参数（网页）

```
https://your-domain/?server=your-domain&port=443
```

---

## 7. 运维手册

### 7.1 日常管理

**Ubuntu/Debian (systemd):**

```bash
# 启动服务
systemctl start som-goserver

# 停止服务
systemctl stop som-goserver

# 重启服务
systemctl restart som-goserver

# 查看状态
systemctl status som-goserver

# 查看日志
journalctl -u som-goserver -f
```

**Alpine (OpenRC):**

```bash
# 启动服务
rc-service som-goserver start

# 停止服务
rc-service som-goserver stop

# 重启服务
rc-service som-goserver restart

# 查看状态
rc-service som-goserver status

# 查看日志
logread -f | grep goserver
```

**通用命令:**

```bash
# 查看实时连接数
ss -tn | grep 27012 | wc -l

# 查看数据库大小
ls -lh /opt/sourceofmana/goserver/som.db
```

### 7.2 备份

```bash
# 备份数据库
sqlite3 /opt/sourceofmana/goserver/som.db ".backup '/backup/som-$(date +%Y%m%d).db'"

# 备份整个项目
tar -czf /backup/som-$(date +%Y%m%d).tar.gz /opt/sourceofmana/goserver/
```

### 7.3 更新部署

**Ubuntu/Debian:**

```bash
cd /opt/sourceofmana/goserver
git pull
go build -o goserver .
systemctl restart som-goserver
```

**Alpine:**

```bash
cd /opt/sourceofmana/goserver
git pull
CGO_ENABLED=0 go build -o goserver .
rc-service som-goserver restart
```

### 7.4 故障排除

| 问题 | 原因 | 解决 |
|------|------|------|
| `address already in use` | 端口被占用 | `ss -tlnp \| grep 27012` 找到进程并杀死 |
| `certificate verify failed` | 自签名证书不受信 | 使用 Let's Encrypt 或客户端加载 `server.crt` |
| WebSocket 400 | nginx 未配置 Upgrade 头 | 检查 `proxy_set_header Upgrade` |
| 连接超时 | 防火墙阻止 | 开放 443/80 端口 |
| RPC checksum 失败 | 客户端/服务器版本不匹配 | 确保两端代码版本一致 |
| `no such table: account` | SQLite 未初始化 | 检查 `som.db` 是否存在，重启服务器自动创建 |

### 7.5 性能监控

```bash
# CPU/内存占用
ps aux | grep goserver

# 网络流量
iftop -i eth0

# 连接数
watch -n 1 'ss -tn | grep 27012 | wc -l'

# 数据库查询性能
sqlite3 /opt/sourceofmana/goserver/som.db "PRAGMA journal_mode;"
```

---

## 8. 测试

### 8.1 单元测试

```bash
cd goserver
go test -v ./...
```

### 8.2 e2e 测试

```bash
cd goserver
./run_e2e.sh
```

### 8.3 浏览器测试

```bash
# 启动测试页面
cd goserver/testdata
python3 -m http.server 8080

# 浏览器访问 http://127.0.0.1:8080/web_test.html
# 点击 "Test WebSocket" 验证连接
```

---

## 9. 项目结构

```
sourceofmana/
├── sources/                    # SoM GDScript 源码
│   ├── network/
│   │   ├── Network.gd          # 网络入口
│   │   ├── NetworkCommons.gd   # 端口/地址常量
│   │   ├── AgentBridge.gd      # SoM 版 AgentBridge
│   │   ├── client/Client.gd    # 客户端实现
│   │   └── server/Server.gd    # 服务端实现
│   └── launcher/
│       └── LauncherCommons.gd  # 平台检测 + 启动参数
├── goserver/                   # Go 服务器（本项目）
│   ├── main.go                 # 入口 + WebSocket 端点
│   ├── tls.go                  # TLS 证书管理
│   ├── bridge.go               # AgentBridge TCP NDJSON
│   ├── arena.go                # 竞技场模块
│   ├── login.go                # 登录链
│   ├── movement.go             # 世界同步 + AOI
│   ├── combat.go               # 战斗系统
│   ├── items.go                # 物品/经济
│   ├── sqlite_store.go         # SQLite 持久化
│   ├── store.go                # 账号存储（内存 + SQLite）
│   ├── variant.go              # Godot Variant 编解码
│   ├── rpc_packet.go           # Godot RPC 包解析
│   ├── testdata/               # e2e 测试客户端
│   ├── run_e2e.sh              # e2e 测试脚本
│   └── HANDOVER.md             # 本文档
├── data/
│   └── conf/templates/sqlite.template.db  # SoM 数据库模板
└── docs/                       # 项目文档
```

---

## 10. Web 客户端部署

### 10.1 预编译二进制

预编译的 Web 客户端位于 [`binary/Web_nothreads/`](binary/Web_nothreads/)，包含：

| 文件 | 大小 | 说明 |
|------|------|------|
| `index.html` | 10 KB | 游戏 HTML（含加载进度条 + Web Worker 预编译） |
| `index.js` | 2.9 MB | Godot JS glue 代码（已修改支持预编译） |
| `index.pck` | 90 MB | 游戏数据包 |
| `index.wasm` | 1.5 MB | 主 wasm 模块 |
| `index.side.wasm` | 44 MB | Side 模块（引擎） |
| `libgdsqlite...wasm` | 2.5 MB | SQLite GDExtension |
| `libsentry...wasm` | 2.6 MB | Sentry GDExtension |

### 10.2 技术要点

- **单线程 wasm**：`thread_support=false`，无需 SharedArrayBuffer，不需要 COOP/COEP headers
- **Web Worker 预编译**：wasm 模块在 Worker 线程中预编译，避免阻塞主线程
- **进度条**：加载过程显示「初始化引擎 → 预编译资源 → 正在加载游戏资源」
- **WSS 连接**：通过 nginx 反向代理，TLS 终止在 nginx，Go 服务器使用明文 WebSocket

### 10.3 部署到服务器

```bash
# 复制 Web 客户端到 nginx 根目录
cp -r binary/Web_nothreads/* /var/www/som/game/

# 确保文件权限正确
chown -R nginx:nginx /var/www/som/game/  # Alpine
chown -R www-data:www-data /var/www/som/game/  # Ubuntu/Debian
```

### 10.4 浏览器兼容性

| 浏览器 | 最低版本 | 说明 |
|--------|---------|------|
| Chrome | 80+ | ✅ 完全支持 |
| Firefox | 78+ | ✅ 完全支持 |
| Safari | 14+ | ✅ 完全支持 |
| Edge | 80+ | ✅ 完全支持 |

---

## 11. 常见问题

### Q: Go 服务器和 SoM 原版服务器有什么区别？

A: Go 服务器是 SoM 原版 GDScript 服务器的完整重写，功能对等：
- ✅ 登录链、世界同步、NPC、组队、交易、战斗、物品、Buff
- ✅ AgentBridge（AI agent 探索）
- ✅ 竞技场
- ✅ SQLite 持久化
- ✅ 内存占用仅 8 MB（原版 331 MB）

### Q: 支持 ENet 协议吗？

A: 不支持。Go 服务器只支持 WebSocket。PC/安卓客户端默认已配置为 WebSocket 模式（`UseWebSocket = true`）。

### Q: 如何添加新的竞技场模板？

A: 编辑 `goserver/arena.go` 中的 `arenaTemplates` map：

```go
var arenaTemplates = map[string]ArenaTemplate{
    "my_template": {
        KillsRequired: 10,
        SpawnPos:      Vec2{X: 2230, Y: 1250},
        Monsters:      []string{"Maggot", "Rat"},
        MaxActive:     5,
        TimeLimit:     90,
    },
}
```

### Q: 如何修改数据库 schema？

A: 编辑 `goserver/sqlite_store.go` 中的 `sqliteSchema` 常量，然后删除 `som.db` 重新初始化。

### Q: 如何调试 AgentBridge？

A: 使用 Python 脚本连接：

```python
import json, socket
s = socket.create_connection(('127.0.0.1', 9102))
s.sendall(b'{"op":"ping"}\n')
buf = b''
while not buf.endswith(b'\n'):
    buf += s.recv(65536)
print(json.loads(buf.decode().strip()))
```

### Q: 生产环境需要哪些安全措施？

A:
1. 使用 Let's Encrypt 证书（不用自签名）
2. Go 服务器绑定 `127.0.0.1`（不暴露公网）
3. 配置防火墙只开放 80/443
4. 定期备份 `som.db`
5. 启用 systemd 自动重启
6. 监控日志和连接数

---

## 12. 联系方式

- 仓库：https://github.com/mnbwc/sourceofmana
- 分支：`feat/agent-arena`
- 文档：`goserver/HANDOVER.md`
