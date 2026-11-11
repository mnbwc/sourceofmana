# Source of Mana — Web 登录部署指南

## 架构

```
浏览器 (HTML/WASM)
    │
    │ wss://som.manasource.org/ws  (TLS 由 nginx 终止)
    ▼
nginx 反向代理 (443)
    │
    │ ws://127.0.0.1:27012/ws  (明文 WebSocket)
    ▼
Go 游戏服务器 (:27012)
```

浏览器只与 nginx 通信（标准 HTTPS/WSS），nginx 负责 TLS 终止并转发到 Go 服务器的明文 WebSocket 端口。

## 0. 快速部署

### Ubuntu/Debian

```bash
cd goserver
sudo bash deploy.sh your-domain.com
```

### Alpine Linux

```bash
cd goserver
sudo sh deploy_alpine.sh your-domain.com
```

两个脚本都会自动完成：安装依赖、构建 Go 服务器、配置 nginx、配置服务、部署 Web 客户端。

## 1. 生成 TLS 证书

### 生产环境（Let's Encrypt）

```bash
# 安装 certbot
sudo apt install certbot python3-certbot-nginx

# 获取证书（需要域名已解析到本机 IP）
sudo certbot certonly --nginx -d som.manasource.org

# 证书位置：
# /etc/letsencrypt/live/som.manasource.org/fullchain.pem
# /etc/letsencrypt/live/som.manasource.org/privkey.pem
```

### 本地开发（自签名）

```bash
# RSA (Ubuntu/Debian)
openssl req -x509 -newkey rsa:2048 \
  -keyout key.pem -out cert.pem -days 365 -nodes \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"

# ECDSA P-256 (Alpine — 更轻量)
openssl ecparam -genkey -name prime256v1 -out key.pem
openssl req -new -x509 -key key.pem -out cert.pem -days 365 -sha256 \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
```

## 2. nginx 配置

```nginx
server {
    listen 443 ssl http2;
    server_name som.manasource.org;

    ssl_certificate     /etc/letsencrypt/live/som.manasource.org/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/som.manasource.org/privkey.pem;
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
    }

    # 健康检查
    location /health {
        proxy_pass http://127.0.0.1:27012;
        proxy_set_header Host $host;
    }

    # 静态文件（Godot Web 导出）
    location / {
        root /var/www/som;
        index index.html;
        try_files $uri $uri/ =404;
    }
}

# HTTP 重定向到 HTTPS
server {
    listen 80;
    server_name som.manasource.org;
    return 301 https://$server_name$request_uri;
}
```

## 3. 启动 Go 服务器

```bash
# 生产环境：只监听明文 WebSocket（nginx 处理 TLS）
cd /path/to/goserver
./goserver

# 本地开发：同时启动 WSS 端点（自签名证书）
WSS_PORT=27013 BRIDGE_PORT=9102 ./goserver
```

## 4. Godot Web 客户端配置

### 4.1 PCK 拆分架构（推荐）

完整构建流程见 [`docs/WEB_BUILD_PIPELINE.md`](../docs/WEB_BUILD_PIPELINE.md)。

Web 客户端采用 **PCK 拆分** 方案，将 72MB 的完整导出拆为三个包：

```
/var/www/som/game/
├── index.html              ← GODOT_CONFIG.mainPack = "core.pck"
├── index.js / *.wasm       ← Godot Web 引擎
├── core.pck                ← 首次必下（~30MB，含登录/选角/起始地图）
└── packs/
    ├── audio.pck           ← 按需下载（~26MB，音乐）
    └── maps.pck            ← 按需下载（~16MB，其他地图）
```

**构建命令**：

```bash
cd sourceofmana
./tools/Godot_v4.7.1-stable_linux.x86_64 --headless --export-release "Web"
cd ..
python3 sourceofmana/tools/split_pck.py binary/Web/index.pck binary/Web/ sourceofmana
cd binary/Web
rm -f index.pck && mkdir -p packs && mv audio.pck maps.pck packs/
```

**nginx Brotli 压缩**（对 `.pck` 文件）：

```nginx
location ~* \.pck$ {
    brotli_static on;
    brotli on;
    brotli_comp_level 6;
    root /var/www/som;
    try_files $uri =404;
}
```

> 浏览器 `fetch()` 会自动解压 Brotli，无需客户端代码修改。
> 验证：`curl -sI -H 'Accept-Encoding: br' https://域名/game/core.pck | grep content-encoding`

### 4.2 预编译二进制（旧版）

仓库包含旧的完整导出，位于 [`binary/Web_nothreads/`](binary/Web_nothreads/)。

```bash
cp -r binary/Web_nothreads/* /var/www/som/game/
chown -R www-data:www-data /var/www/som/game/
```

### 4.3 配置服务器地址

Web 客户端默认连接 `wss://som.manasource.org:6118/ws`（SoM 原版端口）。
要连接到 Go 服务器，有两种方式：

**方式 A：命令行参数**（仅用于测试）
```
--server-address som.manasource.org --ws-port 443
```

**方式 B：修改 `NetworkCommons.gd`**（用于生产）
```gdscript
const ServerAddressTesting : String = "som.manasource.org"
const WebSocketPortTesting : int = 443  # nginx TLS 端口
```

## 5. 验证

1. 浏览器访问 `https://som.manasource.org/` — 应看到游戏登录界面
2. 打开浏览器开发者工具 → Network → WS — 应看到 WebSocket 连接到 `/ws`
3. 输入账号密码登录 — 应成功进入游戏
4. 检查 `wss://som.manasource.org/ws` 的 WebSocket 帧 — 应看到 Godot 协议数据包

**Alpine 验证命令：**

```bash
# 检查服务状态
rc-service som-goserver status
rc-service nginx status

# 检查端口监听
ss -tln | grep -E ':(27012|443)'

# 测试健康检查
curl -k https://127.0.0.1:443/health
```

## 6. 故障排除

| 问题 | 原因 | 解决 |
|------|------|------|
| `wss://` 连接失败 | 证书不受信任 | 使用 Let's Encrypt 或手动信任自签名证书 |
| WebSocket 400 | nginx 未配置 Upgrade 头 | 检查 `proxy_set_header Upgrade` |
| 连接超时 | 防火墙阻止 | 开放 443 端口 |
| RPC checksum 失败 | 客户端/服务器方法不匹配 | 确保 Go 服务器与 SoM 客户端版本一致 |
| 浏览器显示混合内容警告 | 页面 HTTP + WSS | 确保页面也通过 HTTPS 加载 |
| 游戏加载卡在 100% | wasm 编译阻塞主线程 | 确认使用 `Web_nothreads` 版本 |
| 设置界面 Render-Scaling 默认不对 | `data/conf/*` 未导出 | `export_presets.cfg` 的 `include_filter` 加入 `data/conf/*` |
| 浏览器仍加载旧 PCK | 缓存 | 强制刷新 + 清除 IndexedDB `/userfs` |
| Brotli 未生效 | `add_header Content-Encoding identity` 覆盖 | 移除该行，启用 `brotli on` |
| `ERR_CONNECTION_REFUSED` | 服务器未启动 | Alpine: `rc-service som-goserver start`；Ubuntu: `systemctl start som-goserver` |
| nginx 配置测试失败 | 配置文件语法错误 | Alpine: `nginx -t`；检查 `/etc/nginx/nginx.conf` |
| 自签名证书警告 | 浏览器不信任自签名证书 | 点击「高级」→「继续访问」，或替换为 Let's Encrypt 证书 |

## 7. 本地测试

```bash
# 1. 启动 Go 服务器（WSS 模式）
cd goserver
WSS_PORT=27013 BRIDGE_PORT=9102 ./goserver

# 2. 启动 HTTP 服务器（提供测试页面）
python3 -m http.server 8080 --directory testdata/

# 3. 浏览器访问（需要信任自签名证书）
# 先访问 https://127.0.0.1:27013/health 并信任证书
# 然后访问 http://127.0.0.1:8080/web_test.html
# 点击 "Test WebSocket"
```

## 8. Alpine 特别说明

Alpine Linux 使用 musl libc 而非 glibc，需要注意：

- **Go 编译**：使用 `CGO_ENABLED=0` 生成纯静态二进制
- **nginx 配置**：Alpine 的 nginx 配置在 `/etc/nginx/nginx.conf`（不是 `sites-available`）
- **服务管理**：使用 OpenRC（`rc-service` / `rc-update`），不是 systemd
- **证书**：推荐使用 ECDSA P-256 自签名证书（更轻量）
- **日志**：使用 `logread` 查看系统日志，不是 `journalctl`
- **防火墙**：使用 `iptables` 或 `nftables`，不是 `ufw`
