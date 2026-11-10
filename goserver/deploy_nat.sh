#!/bin/bash
# deploy_nat.sh — 香港 NAT VPS 一键部署魔改版 SoM Go 服务器
# 使用方法：sudo bash deploy_nat.sh [域名或公网IP]
#
# 适用场景：
#   - 香港 NAT VPS（无公网 IP，通过 NAT 端口映射对外提供服务）
#   - Ubuntu 20.04+ / Debian 11+
#   - 无域名，直接用 IP + 端口访问（自签名证书）
#
# NAT VPS 说明：
#   香港 NAT VPS 通常没有公网 IP，服务商控制台提供端口映射。
#   你需要将服务商分配的公网端口映射到 VPS 的内网端口。
#   常见映射方式：公网 1xxxx → 内网 443，公网 2xxxx → 内网 27012
#   部署完成后脚本会列出需要映射的端口清单。

set -e

DOMAIN=${1:-localhost}
BRANCH="feat/auto-play-and-rpg-systems"
REPO="https://github.com/mnbwc/sourceofmana.git"
INSTALL_DIR="/opt/sourceofmana"
WEB_ROOT="/var/www/som"

echo "============================================"
echo "  Source of Mana 魔改版 — 香港 NAT VPS 部署"
echo "============================================"
echo "地址: $DOMAIN"
echo "分支: $BRANCH"
echo "安装目录: $INSTALL_DIR"
echo ""

# 检查 root 权限
if [ "$EUID" -ne 0 ]; then
    echo "错误：请使用 root 权限运行 (sudo bash deploy_nat.sh)"
    exit 1
fi

# 1. 安装依赖
echo "[1/6] 安装依赖..."
apt-get update -qq
apt-get install -y -qq \
    nginx \
    git \
    golang-go \
    sqlite3 \
    curl \
    ufw \
    openssl

# 2. 克隆/更新代码
echo "[2/6] 获取代码（魔改分支）..."
if [ -d "$INSTALL_DIR" ]; then
    echo "  更新现有代码..."
    cd "$INSTALL_DIR"
    git fetch origin
    git checkout "$BRANCH"
    git pull origin "$BRANCH"
else
    echo "  克隆仓库..."
    git clone -b "$BRANCH" "$REPO" "$INSTALL_DIR"
    cd "$INSTALL_DIR"
fi

# 3. 构建 Go 服务器
echo "[3/6] 构建 Go 服务器..."
cd "$INSTALL_DIR/goserver"
go build -o goserver .
echo "  构建完成: $(ls -lh goserver | awk '{print $5}')"

# 4. 配置 systemd
echo "[4/6] 配置 systemd 服务..."
cat > /etc/systemd/system/som-goserver.service << EOF
[Unit]
Description=Source of Mana Modded Go Server
After=network.target

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=$INSTALL_DIR/goserver
ExecStart=$INSTALL_DIR/goserver/goserver
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# 网络绑定
Environment="WS_BIND_IP=0.0.0.0"
Environment="WS_PORT=27012"
Environment="WSS_PORT=27013"
Environment="BRIDGE_BIND_IP=127.0.0.1"
Environment="BRIDGE_PORT=9102"
Environment="SOM_DB_FILE=$INSTALL_DIR/goserver/som.db"

# 安全限制
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$INSTALL_DIR/goserver

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable som-goserver
systemctl restart som-goserver
sleep 2

# 5. 生成自签名证书（NAT VPS 无域名，用 IP 访问）
echo "[5/6] 配置 TLS 证书..."
mkdir -p /etc/som-certs
if [ ! -f /etc/som-certs/fullchain.pem ]; then
    openssl req -x509 -newkey rsa:2048 \
        -keyout /etc/som-certs/privkey.pem \
        -out /etc/som-certs/fullchain.pem \
        -days 365 -nodes \
        -subj "/CN=$DOMAIN" \
        -addext "subjectAltName=DNS:$DOMAIN,IP:$DOMAIN,IP:127.0.0.1" 2>/dev/null
    echo "  自签名证书已生成"
else
    echo "  证书已存在，跳过"
fi

# 6. 配置 nginx
echo "[6/6] 配置 nginx..."
mkdir -p "$WEB_ROOT"

cat > /etc/nginx/sites-available/som << NGINX_EOF
server {
    listen 443 ssl http2;
    server_name $DOMAIN;

    ssl_certificate     /etc/som-certs/fullchain.pem;
    ssl_certificate_key /etc/som-certs/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         HIGH:!aNULL:!MD5;

    # 安全头
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;

    # WebSocket 代理（魔改客户端连接入口）
    location /ws {
        proxy_pass http://127.0.0.1:27012;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_read_timeout 86400;
        proxy_send_timeout 86400;
        proxy_connect_timeout 7d;
    }

    # 健康检查
    location /health {
        proxy_pass http://127.0.0.1:27012;
        proxy_set_header Host \$host;
        access_log off;
    }

    # 静态文件（预留 Web 客户端部署）
    location / {
        root $WEB_ROOT;
        index index.html;
        try_files \$uri \$uri/ =404;
    }
}

# HTTP 重定向到 HTTPS
server {
    listen 80;
    server_name $DOMAIN;
    return 301 https://\$server_name\$request_uri;
}
NGINX_EOF

ln -sf /etc/nginx/sites-available/som /etc/nginx/sites-enabled/som

# 测试并重启 nginx
nginx -t
systemctl restart nginx

# 7. 配置防火墙
echo ""
echo "[防火墙] 配置端口..."
ufw allow 80/tcp   >/dev/null 2>&1 || true
ufw allow 443/tcp  >/dev/null 2>&1 || true
ufw allow 27012/tcp >/dev/null 2>&1 || true
ufw --force enable >/dev/null 2>&1 || true
echo "  已开放: 80, 443, 27012"

# 8. 验证服务
echo ""
echo "[验证] 检查服务状态..."
sleep 1
systemctl is-active som-goserver && echo "  Go 服务器: 运行中" || echo "  Go 服务器: 未运行！"
systemctl is-active nginx && echo "  nginx: 运行中" || echo "  nginx: 未运行！"
curl -s http://127.0.0.1:27012/health && echo "  health 检查: OK" || echo "  health 检查: 失败！"

echo ""
echo "============================================"
echo "  部署完成！"
echo "============================================"
echo ""
echo "服务状态: systemctl status som-goserver"
echo "查看日志: journalctl -u som-goserver -f"
echo ""
echo "本地访问地址:"
echo "  网页登录: https://$DOMAIN/"
echo "  WebSocket: wss://$DOMAIN/ws"
echo "  直连 WS:   ws://$DOMAIN:27012/ws"
echo ""
echo "============================================"
echo "  ★ 香港 NAT VPS 端口映射清单 ★"
echo "============================================"
echo ""
echo "请在 VPS 服务商控制台配置以下端口映射："
echo ""
echo "  ┌─────────────┬─────────────┬──────────────────────────┐"
echo "  │ 公网端口    │ 内网端口    │ 用途                     │"
echo "  ├─────────────┼─────────────┼──────────────────────────┤"
echo "  │ 10080       │ 80          │ HTTP→HTTPS 重定向        │"
echo "  │ 11443       │ 443         │ 网页登录 (WSS)           │"
echo "  │ 27012       │ 27012       │ 魔改客户端直连 (WS)      │"
echo "  └─────────────┴─────────────┴──────────────────────────┘"
echo ""
echo "映射完成后，外部访问地址："
echo "  网页: https://<公网IP>:11443/"
echo "  客户端: ws://<公网IP>:27012/ws"
echo ""
echo "如果 NAT 服务商支持任意端口映射，建议："
echo "  公网 443 → 内网 443（标准 HTTPS，客户端无需指定端口）"
echo "  公网 27012 → 内网 27012（客户端直连）"
echo ""
echo "验证命令:"
echo "  curl http://127.0.0.1:27012/health"
echo "  ss -tln | grep -E ':(27012|27013|9102)'"
echo ""
