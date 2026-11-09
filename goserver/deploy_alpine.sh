#!/bin/sh
# deploy_alpine.sh — 一键部署 SoM Go 服务器 + nginx (Alpine Linux)
# 使用方法: sudo sh deploy_alpine.sh [DOMAIN]
#
# 前置条件:
#   - Alpine Linux 3.18+ (musl libc, OpenRC)
#   - 公网 IP
#   - 端口 80/443 未被占用
#   - NAT 映射: 外部端口 -> 内部端口 (例如 11443->443)
#
# 说明:
#   - Alpine 使用 musl libc，Go 的 CGO 默认关闭，直接编译静态二进制
#   - 使用 OpenRC 管理服务 (rc-service / rc-update)
#   - nginx 配置在 /etc/nginx/nginx.conf (Alpine 默认路径)
#   - 自签名 ECDSA P-256 证书 (开发用)
#   - 游戏 Web 客户端部署在 /var/www/som/game/

set -e

DOMAIN=${1:-localhost}
REPO="https://github.com/mnbwc/sourceofmana.git"
BRANCH="feat/agent-arena"
INSTALL_DIR="/opt/sourceofmana/goserver"
WEB_ROOT="/var/www/som"
GAME_DIR="$WEB_ROOT/game"

echo "============================================"
echo "  Source of Mana — Alpine 一键部署"
echo "============================================"
echo "域名: $DOMAIN"
echo "安装目录: $INSTALL_DIR"
echo "Web 根目录: $WEB_ROOT"
echo ""

# 检查 root 权限
if [ "$(id -u)" -ne 0 ]; then
    echo "错误：请使用 root 权限运行 (sudo sh deploy_alpine.sh $DOMAIN)"
    exit 1
fi

# 1. 安装依赖
echo "[1/7] 安装依赖..."
apk update
apk add --no-cache \
    nginx \
    git \
    go \
    sqlite \
    curl \
    openssl \
    ca-certificates

# 2. 克隆/更新代码
echo "[2/7] 获取代码..."
if [ -d "$INSTALL_DIR" ]; then
    echo "  更新现有代码..."
    cd "$INSTALL_DIR"
    git fetch origin
    git checkout "$BRANCH"
    git pull origin "$BRANCH"
else
    echo "  克隆仓库..."
    mkdir -p /opt/sourceofmana
    git clone -b "$BRANCH" "$REPO" /opt/sourceofmana
    cd "$INSTALL_DIR"
fi

# 3. 构建 Go 服务器
echo "[3/7] 构建 Go 服务器..."
cd "$INSTALL_DIR"
CGO_ENABLED=0 go build -o goserver .
echo "  构建完成: $(ls -lh goserver | awk '{print $5}')"

# 4. 生成 TLS 自签名证书 (ECDSA P-256)
echo "[4/7] 生成 TLS 证书..."
mkdir -p /root
if [ ! -f /root/server.crt ] || [ ! -f /root/server.key ]; then
    openssl ecparam -genkey -name prime256v1 -out /root/server.key
    openssl req -new -x509 -key /root/server.key -out /root/server.crt \
        -days 3650 -sha256 \
        -subj "/CN=$DOMAIN" \
        -addext "subjectAltName=DNS:$DOMAIN,IP:$(curl -s ifconfig.me 2>/dev/null || echo '127.0.0.1')"
    echo "  证书生成完成: /root/server.crt"
else
    echo "  证书已存在，跳过"
fi

# 5. 配置 nginx
echo "[5/7] 配置 nginx..."
mkdir -p "$WEB_ROOT" "$GAME_DIR"

# 主 nginx 配置
cat > /etc/nginx/nginx.conf << 'NGINX_EOF'
user nginx;
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /var/run/nginx.pid;

events {
    worker_connections 1024;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';
    access_log /var/log/nginx/access.log main;

    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 4096;

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml image/svg+xml;

    # WASM MIME type
    types {
        application/wasm wasm;
    }

    # Rate limiting zone
    limit_conn_zone $binary_remote_addr zone=addr:10m;

    server {
        listen 0.0.0.0:443 ssl;
        server_name DOMAIN_PLACEHOLDER;

        ssl_certificate /root/server.crt;
        ssl_certificate_key /root/server.key;
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305;
        ssl_prefer_server_ciphers on;

        # Security headers
        add_header X-Frame-Options "SAMEORIGIN" always;
        add_header X-Content-Type-Options "nosniff" always;
        add_header X-XSS-Protection "1; mode=block" always;

        # WebSocket 代理 (裸路径 + /ws)
        # Godot WebSocketMultiplayerPeer 连接 host:port 不带路径
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
            proxy_connect_timeout 7d;
        }

        # 健康检查
        location /health {
            proxy_pass http://127.0.0.1:27012;
            proxy_set_header Host $host;
            access_log off;
        }

        # 游戏 Web 客户端
        location /game/ {
            alias GAME_DIR_PLACEHOLDER;
            index index.html;
            try_files $uri $uri/ /game/index.html;

            # 禁用缓存 for development
            add_header Cache-Control "no-cache, no-store, must-revalidate";
            add_header Pragma "no-cache";
            add_header Expires "0";
        }

        # WebSocket at root path (Godot bare host:port)
        location = / {
            if ($http_upgrade = "websocket") {
                rewrite ^ /ws last;
            }
            root WEB_ROOT_PLACEHOLDER;
            index index.html;
            try_files $uri $uri/ /index.html;
        }

        location / {
            root WEB_ROOT_PLACEHOLDER;
            index index.html;
            try_files $uri $uri/ /index.html;
        }
    }

    # HTTP 重定向到 HTTPS
    server {
        listen 0.0.0.0:80;
        server_name DOMAIN_PLACEHOLDER;
        return 301 https://$server_name:11443$request_uri;
    }
}
NGINX_EOF

# 替换占位符
EXTERNAL_IP=$(curl -s ifconfig.me 2>/dev/null || echo "$DOMAIN")
sed -i "s/DOMAIN_PLACEHOLDER/$DOMAIN/g" /etc/nginx/nginx.conf
sed -i "s|GAME_DIR_PLACEHOLDER|$GAME_DIR|g" /etc/nginx/nginx.conf
sed -i "s|WEB_ROOT_PLACEHOLDER|$WEB_ROOT|g" /etc/nginx/nginx.conf

# 测试 nginx 配置
nginx -t

# 6. 配置 OpenRC 服务
echo "[6/7] 配置 OpenRC 服务..."

# Go 服务器服务
cat > /etc/init.d/som-goserver << 'INIT_EOF'
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
INIT_EOF
chmod +x /etc/init.d/som-goserver

# 环境变量
cat > /etc/conf.d/som-goserver << ENV_EOF
BIND_IP=0.0.0.0
WS_PORT=27012
WSS_PORT=27013
BRIDGE_PORT=9102
SOM_DB_FILE=/opt/sourceofmana/goserver/som.db
ENV_EOF

# 启用服务
rc-update add som-goserver default 2>/dev/null || true
rc-update add nginx default 2>/dev/null || true

# 启动服务
rc-service som-goserver restart 2>/dev/null || rc-service som-goserver start
sleep 2
rc-service nginx restart 2>/dev/null || rc-service nginx start

# 7. 部署 Web 客户端
echo "[7/7] 部署 Web 客户端..."
if [ -d "$INSTALL_DIR/../binary/Web_nothreads" ]; then
    cp -r "$INSTALL_DIR/../binary/Web_nothreads/"* "$GAME_DIR/"
    echo "  Web 客户端已部署到 $GAME_DIR"
else
    echo "  警告: 未找到 Web 客户端二进制文件"
    echo "  请手动复制 binary/Web_nothreads/ 到 $GAME_DIR"
fi

# 创建简单的首页
cat > "$WEB_ROOT/index.html" << 'LANDING_EOF'
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Source of Mana — Go Server</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #0f0c29, #302b63, #24243e);
            color: #fff;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .container {
            text-align: center;
            max-width: 600px;
            padding: 2rem;
        }
        h1 {
            font-size: 2.5rem;
            margin-bottom: 0.5rem;
            background: linear-gradient(90deg, #00d2ff, #3a7bd5);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .subtitle {
            color: #888;
            margin-bottom: 2rem;
            font-size: 1.1rem;
        }
        .btn {
            display: inline-block;
            padding: 1rem 2.5rem;
            background: linear-gradient(90deg, #00d2ff, #3a7bd5);
            color: #fff;
            text-decoration: none;
            border-radius: 8px;
            font-size: 1.1rem;
            font-weight: 600;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 10px 30px rgba(0, 210, 255, 0.3);
        }
        .info {
            margin-top: 2rem;
            padding: 1rem;
            background: rgba(255,255,255,0.05);
            border-radius: 8px;
            font-size: 0.9rem;
            color: #aaa;
        }
        .info code {
            color: #00d2ff;
            background: rgba(0,210,255,0.1);
            padding: 0.2rem 0.4rem;
            border-radius: 4px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Source of Mana</h1>
        <p class="subtitle">Go Server · Web Client</p>
        <a href="/game/?server=SERVER_PLACEHOLDER&port=11443" class="btn">🎮 启动游戏客户端</a>
        <div class="info">
            <p>服务器地址: <code>SERVER_PLACEHOLDER</code></p>
            <p>WebSocket 端口: <code>11443</code> (WSS)</p>
            <p>AI 无头端口: <code>27012</code> (WS)</p>
        </div>
    </div>
</body>
</html>
LANDING_EOF
sed -i "s/SERVER_PLACEHOLDER/$EXTERNAL_IP/g" "$WEB_ROOT/index.html"

# 防火墙提示
echo ""
echo "============================================"
echo "  部署完成！"
echo "============================================"
echo ""
echo "服务状态: rc-service som-goserver status"
echo "查看日志: logread -f | grep goserver"
echo "nginx 日志: tail -f /var/log/nginx/access.log"
echo ""
echo "游戏地址:"
echo "  人类网页登录: https://$EXTERNAL_IP:11443/"
echo "  游戏客户端:   https://$EXTERNAL_IP:11443/game/?server=$EXTERNAL_IP&port=11443"
echo "  人类 WebSocket: wss://$EXTERNAL_IP:11443/ws"
echo "  AI 无头登录:   ws://$EXTERNAL_IP:27012/ws"
echo ""
echo "NAT/安全组需映射的端口:"
echo "  11443 -> 443 (TCP) — 人类网页登录 (WSS)"
echo "  27012 -> 27012 (TCP) — AI 无头客户端登录 (WS)"
echo "  20080 -> 80 (TCP) — HTTP->HTTPS 重定向"
echo ""
echo "验证命令:"
echo "  curl -k https://127.0.0.1:443/health"
echo "  curl -k https://127.0.0.1:443/"
echo ""
echo "注意: 自签名证书，浏览器会显示安全警告。"
echo "      生产环境请替换为 Let's Encrypt 证书。"
echo ""
