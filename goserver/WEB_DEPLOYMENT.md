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
openssl req -x509 -newkey rsa:2048 \
  -keyout key.pem -out cert.pem -days 365 -nodes \
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

### 4.1 导出 Web 版本

在 Godot 编辑器中选择 `Project > Export > Web`，导出到 `binary/Web/`。

### 4.2 配置服务器地址

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

### 4.3 部署静态文件

```bash
# 复制 Godot Web 导出文件到 nginx 根目录
cp binary/Web/* /var/www/som/

# 确保文件权限正确
chown -R www-data:www-data /var/www/som
```

## 5. 验证

1. 浏览器访问 `https://som.manasource.org/` — 应看到游戏登录界面
2. 打开浏览器开发者工具 → Network → WS — 应看到 WebSocket 连接到 `/ws`
3. 输入账号密码登录 — 应成功进入游戏
4. 检查 `wss://som.manasource.org/ws` 的 WebSocket 帧 — 应看到 Godot 协议数据包

## 6. 故障排除

| 问题 | 原因 | 解决 |
|------|------|------|
| `wss://` 连接失败 | 证书不受信任 | 使用 Let's Encrypt 或手动信任自签名证书 |
| WebSocket 400 | nginx 未配置 Upgrade 头 | 检查 `proxy_set_header Upgrade` |
| 连接超时 | 防火墙阻止 | 开放 443 端口 |
| RPC checksum 失败 | 客户端/服务器方法不匹配 | 确保 Go 服务器与 SoM 客户端版本一致 |
| 浏览器显示混合内容警告 | 页面 HTTP + WSS | 确保页面也通过 HTTPS 加载 |

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
