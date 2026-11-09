#!/bin/bash
# deploy.sh — 一键部署 SoM Go 服务器 + nginx + Let's Encrypt
# 使用方法：sudo bash deploy.sh your-domain.com
#
# 前置条件：
#   - Ubuntu 20.04+ / Debian 11+
#   - 域名已解析到本机公网 IP
#   - 80/443 端口未被占用

set -e

DOMAIN=${1:-localhost}
BRANCH="feat/agent-arena"
REPO="https://github.com/mnbwc/sourceofmana.git"
INSTALL_DIR="/opt/sourceofmana/goserver"
WEB_ROOT="/var/www/som"

echo "============================================"
echo "  Source of Mana — Go 服务器一键部署"
echo "============================================"
echo "域名: $DOMAIN"
echo "安装目录: $INSTALL_DIR"
echo "Web 根目录: $WEB_ROOT"
echo ""

# 检查 root 权限
if [ "$EUID" -ne 0 ]; then
    echo "错误：请使用 root 权限运行 (sudo bash deploy.sh $DOMAIN)"
    exit 1
fi

# 1. 安装依赖
echo "[1/6] 安装依赖..."
apt-get update -qq
apt-get install -y -qq \
    nginx \
    certbot \
    python3-certbot-nginx \
    git \
    golang-go \
    sqlite3 \
    curl

# 2. 克隆/更新代码
echo "[2/6] 获取代码..."
if [ -d "$INSTALL_DIR" ]; then
    echo "  更新现有代码..."
    cd "$INSTALL_DIR"
    git fetch origin
    git checkout "$BRANCH"
    git pull origin "$BRANCH"
else
    echo "  克隆仓库..."
    git clone -b "$BRANCH" "$REPO" /opt/sourceofmana
    cd "$INSTALL_DIR"
fi

# 3. 构建 Go 服务器
echo "[3/6] 构建 Go 服务器..."
cd "$INSTALL_DIR"
go build -o goserver .
echo "  构建完成: $(ls -lh goserver | awk '{print $5}')"

# 4. 配置 systemd
echo "[4/6] 配置 systemd 服务..."
cat > /etc/systemd/system/som-goserver.service << EOF
[Unit]
Description=Source of Mana Go Server
After=network.target

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/goserver
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# 网络绑定：游戏端口对外（网页 + AI 无头），管理端口仅本地
Environment="WS_BIND_IP=0.0.0.0"
Environment="WS_PORT=27012"
Environment="WSS_PORT=27013"
Environment="BRIDGE_BIND_IP=127.0.0.1"
Environment="BRIDGE_PORT=9102"
Environment="SOM_DB_FILE=$INSTALL_DIR/som.db"

# 安全限制
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$INSTALL_DIR

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable som-goserver
systemctl restart som-goserver
sleep 2

# 5. 配置 nginx
echo "[5/6] 配置 nginx..."
mkdir -p "$WEB_ROOT"

cat > /etc/nginx/sites-available/som << 'NGINX_EOF'
server {
    listen 443 ssl http2;
    server_name DOMAIN_PLACEHOLDER;

    ssl_certificate     /etc/letsencrypt/live/DOMAIN_PLACEHOLDER/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/DOMAIN_PLACEHOLDER/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # 安全头
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

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
        proxy_connect_timeout 7d;
    }

    # 健康检查
    location /health {
        proxy_pass http://127.0.0.1:27012;
        proxy_set_header Host $host;
        access_log off;
    }

    # Godot Web 导出静态文件
    location / {
        root /var/www/som;
        index index.html;
        try_files $uri $uri/ =404;

        # WASM 文件需要正确的 MIME 类型
        location ~ \.wasm$ {
            add_header Content-Type application/wasm;
        }
    }
}

# HTTP 重定向到 HTTPS
server {
    listen 80;
    server_name DOMAIN_PLACEHOLDER;
    return 301 https://$server_name$request_uri;
}
NGINX_EOF

sed -i "s/DOMAIN_PLACEHOLDER/$DOMAIN/g" /etc/nginx/sites-available/som
ln -sf /etc/nginx/sites-available/som /etc/nginx/sites-enabled/som

# 6. 获取 Let's Encrypt 证书
echo "[6/6] 获取 Let's Encrypt 证书..."
if [ "$DOMAIN" = "localhost" ]; then
    echo "  跳过 Let's Encrypt（本地开发模式）"
    # 生成本地自签名证书用于 nginx
    mkdir -p /etc/letsencrypt/live/$DOMAIN
    openssl req -x509 -newkey rsa:2048 \
        -keyout /etc/letsencrypt/live/$DOMAIN/privkey.pem \
        -out /etc/letsencrypt/live/$DOMAIN/fullchain.pem \
        -days 365 -nodes \
        -subj "/CN=localhost" \
        -addext "subjectAltName=DNS:localhost,IP:127.0.0.1" 2>/dev/null
fi

# 测试 nginx 配置
nginx -t

# 重启 nginx
systemctl restart nginx

# 如果域名不是 localhost，获取真实证书
if [ "$DOMAIN" != "localhost" ]; then
    echo "  申请 Let's Encrypt 证书..."
    certbot --nginx -d "$DOMAIN" --non-interactive --agree-tos -m "admin@$DOMAIN" || {
        echo "  警告：证书申请失败，请检查域名解析和 80 端口可达性"
    }
fi

# 7. 配置防火墙
echo ""
echo "[防火墙] 配置端口..."
if command -v ufw >/dev/null 2>&1; then
    ufw allow 80/tcp   >/dev/null 2>&1 || true
    ufw allow 443/tcp  >/dev/null 2>&1 || true
    ufw allow 27012/tcp >/dev/null 2>&1 || true
    ufw --force enable >/dev/null 2>&1 || true
    echo "  已开放: 80, 443, 27012"
    ufw status | grep -E "80|443|27012" || true
else
    echo "  未安装 ufw，请手动在 NAT/安全组开放: 80, 443, 27012"
fi

echo ""
echo "============================================"
echo "  部署完成！"
echo "============================================"
echo ""
echo "服务状态: systemctl status som-goserver"
echo "查看日志: journalctl -u som-goserver -f"
echo ""
echo "人类网页登录: https://$DOMAIN/"
echo "人类 WebSocket: wss://$DOMAIN/ws"
echo "AI 无头登录:   ws://$DOMAIN:27012/ws"
echo ""
echo "NAT/安全组需映射的端口:"
echo "  80    (TCP) — HTTP→HTTPS + 证书续期"
echo "  443   (TCP) — 人类网页登录 (WSS)"
echo "  27012 (TCP) — AI 无头客户端登录 (WS)"
echo ""
echo "验证命令:"
echo "  curl http://127.0.0.1:27012/health"
echo "  ss -tln | grep -E ':(27012|27013|9102)'"
echo ""

