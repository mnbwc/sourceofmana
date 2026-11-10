#!/bin/bash
# deploy_web_optimized.sh — 部署 Web 优化版客户端到 Alpine VPS
# 包含：PCK 拆包 + Brotli 预压缩 + nginx Brotli 配置
#
# 使用方法：
#   1. 将 binary/Web/ 上传到 VPS /tmp/web/
#   2. 运行：sudo bash deploy_web_optimized.sh

set -e

WEB_ROOT="/var/www/som"
TMP_DIR="/tmp/web"

echo "============================================"
echo "  SoM Web 优化版部署"
echo "============================================"

# 检查文件
if [ ! -d "$TMP_DIR" ]; then
    echo "错误：请先上传 binary/Web/ 到 $TMP_DIR"
    echo "  scp -r binary/Web/* root@154.222.30.52:/tmp/web/"
    exit 1
fi

# 备份旧版本
if [ -d "$WEB_ROOT" ]; then
    echo "[1/5] 备份旧版本..."
    BACKUP="/var/www/som_backup_$(date +%Y%m%d_%H%M%S)"
    cp -r "$WEB_ROOT" "$BACKUP"
    echo "  备份到: $BACKUP"
fi

# 部署新版本
echo "[2/5] 部署新版本..."
mkdir -p "$WEB_ROOT/packs"
cp -r "$TMP_DIR"/* "$WEB_ROOT/"
chown -R www-data:www-data "$WEB_ROOT" 2>/dev/null || chown -R nginx:nginx "$WEB_ROOT"
chmod -R 755 "$WEB_ROOT"

# 验证文件
echo "[3/5] 验证文件..."
REQUIRED=(
    "index.html"
    "index.js"
    "index.side.wasm"
    "index.wasm"
    "index.pck"
    "packs/audio.pck"
    "packs/map_layers.pck"
)
for f in "${REQUIRED[@]}"; do
    if [ ! -f "$WEB_ROOT/$f" ]; then
        echo "  错误：缺少 $f"
        exit 1
    fi
    echo "  OK: $f ($(du -h "$WEB_ROOT/$f" | cut -f1))"
done

# 配置 nginx Brotli
echo "[4/5] 配置 nginx Brotli..."
NGINX_CONF="/etc/nginx/http.d/som.conf"
if [ ! -f "$NGINX_CONF" ]; then
    NGINX_CONF="/etc/nginx/conf.d/som.conf"
fi

# 检查是否已有 Brotli 配置
if ! grep -q "brotli" "$NGINX_CONF" 2>/dev/null; then
    echo "  添加 Brotli 配置到 $NGINX_CONF"
    cat >> "$NGINX_CONF" << 'BROTLI_EOF'

# === Brotli 压缩配置 ===
brotli on;
brotli_comp_level 6;
brotli_types text/plain text/css text/xml text/javascript
             application/javascript application/json application/xml
             application/rss+xml application/atom+xml
             image/svg+xml font/ttf font/otf font/woff font/woff2
             application/wasm application/octet-stream;
brotli_static on;

# packs 目录（PCK 分包）
location /packs/ {
    root /var/www/som;
    expires 7d;
    add_header Cache-Control "public, immutable";
    try_files $uri =404;
}

# index.html 不缓存
location = /index.html {
    root /var/www/som;
    add_header Cache-Control "no-cache, no-store, must-revalidate";
    expires 0;
}
BROTLI_EOF
else
    echo "  Brotli 配置已存在，跳过"
fi

# 测试 nginx 配置
nginx -t
rc-service nginx restart 2>/dev/null || systemctl restart nginx 2>/dev/null || nginx -s reload

echo "[5/5] 验证部署..."
sleep 1
curl -s -o /dev/null -w "  index.html: %{http_code}\n" http://127.0.0.1/
curl -s -o /dev/null -w "  index.pck: %{http_code}\n" http://127.0.0.1/index.pck
curl -s -o /dev/null -w "  packs/audio.pck: %{http_code}\n" http://127.0.0.1/packs/audio.pck

echo ""
echo "============================================"
echo "  部署完成！"
echo "============================================"
echo ""
echo "访问地址:"
echo "  https://<公网IP>:11443/"
echo ""
echo "验证 Brotli:"
echo "  curl -H 'Accept-Encoding: br' -I http://127.0.0.1/index.pck"
echo "  应看到 Content-Encoding: br"
echo ""
