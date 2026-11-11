# Source of Mana — Web 构建与部署流水线

> 本文档记录 Source of Mana（SoM）魔改版 Web 端的完整构建、拆分、压缩与部署流程。
> 目标：**首次加载 < 35MB**，按需加载其余资源，并支持 Brotli 传输压缩。

---

## 1. 架构总览

```
┌─────────────────────────────────────────────────────────────────────┐
│                         构建机（本地沙箱）                           │
│                                                                     │
│  Godot 4.7.1 导出 ──▶ index.pck (72MB)                             │
│        │                                                            │
│        ▼                                                            │
│  tools/split_pck.py ──▶ core.pck (30MB)                             │
│                      ──▶ packs/audio.pck (26MB)                      │
│                      ──▶ packs/maps.pck (16MB)                       │
│        │                                                            │
│        ▼                                                            │
│  更新 index.html (GODOT_CONFIG: mainPack=core.pck)                  │
│        │                                                            │
│        ▼                                                            │
│  tar.gz 打包 ──▶ scp 上传到 VPS                                      │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         VPS（生产服务器）                            │
│                                                                     │
│  /var/www/som/game/                                                 │
│    ├── index.html          (GODOT_CONFIG 指向 core.pck)             │
│    ├── index.js / *.wasm   (Godot Web 引擎)                         │
│    ├── core.pck            (首次必下 ~30MB)                         │
│    └── packs/                                                        │
│         ├── audio.pck      (按需下载 ~26MB)                          │
│         └── maps.pck       (按需下载 ~16MB)                          │
│                                                                     │
│  nginx (443) ──▶ Brotli 压缩 .pck 文件（浏览器自动解压）              │
│       │                                                             │
│       ├── /game/*  → 静态文件                                       │
│       └── /ws      → Go 游戏服务器 (127.0.0.1:27012)                 │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. 前置条件

| 工具 | 版本 | 用途 | 位置 |
|------|------|------|------|
| Godot | 4.7.1 | Web 导出 | `tools/Godot_v4.7.1-stable_linux.x86_64` |
| Python | 3.8+ | PCK 拆分脚本 | 系统自带 |
| brotli | 1.1.0+ | 压缩率测试（可选） | `pip3 install brotli` |
| scp / ssh | - | 部署到 VPS | 系统自带 |

### 2.1 Godot 二进制

```bash
# 项目内已包含 Godot 4.7.1 Linux x86_64 二进制
ls -la tools/Godot_v4.7.1-stable_linux.x86_64

# 验证版本
./tools/Godot_v4.7.1-stable_linux.x86_64 --version
# 输出: 4.7.1.stable.official.8ded7ccb4
```

### 2.2 项目目录结构

```
sourceofmana/                   ← Git 仓库根目录
├── project.godot               ← Godot 项目配置
├── export_presets.cfg          ← 导出预设（Web 预设）
├── tools/
│   └── split_pck.py            ← PCK 拆分脚本
├── data/
│   ├── conf/settings.cfg       ← 默认设置（Render-Scaling 等）
│   ├── db/                     ← 游戏数据库
│   └── maps/                   ← 地图资源
├── presets/
│   ├── music/*.tres            ← 音乐资源描述
│   └── maps/                   ← 地图定义
├── sources/
│   ├── launcher/Launcher.gd    ← 启动器（PCK 加载逻辑）
│   ├── system/PackManager.gd   ← 按需下载管理器
│   └── ...
├── binary/Web/                 ← 导出输出目录（gitignore）
│   ├── index.html
│   ├── index.pck               ← 完整导出（拆分前）
│   ├── core.pck                ← 拆分后主包
│   └── packs/
│       ├── audio.pck
│       └── maps.pck
└── docs/WEB_BUILD_PIPELINE.md  ← 本文档
```

---

## 3. 构建流程

### 3.1 导出 Web 版本

```bash
cd /mnt/data/catpaw/home/.meituan-catpaw/333630993/desk_default_workspace/sourceofmana

./tools/Godot_v4.7.1-stable_linux.x86_64 --headless --export-release "Web"
```

**输出**：`../binary/Web/`（即 workspace 根目录下的 `binary/Web/`）

**关键导出配置**（`export_presets.cfg` 中的 `[preset.5]` Web 预设）：

```ini
[preset.5.options]
variant/thread_support=false              # 单线程（Web Worker 预编译）
html/custom_html_shell="html/custom_html_shell.html"
html/head_include=""
progressive_web_app/enabled=false
sentry/inject_script=false
include_filter="data/db/*,data/conf/*,data/themes/*"   # 必须包含 data/conf
exclude_filter="...,data/maps/*,..."                   # 地图走按需下载
```

> ⚠️ **重要**：`include_filter` 必须包含 `data/conf/*`，否则 `settings.cfg` 不会被打入 PCK，
> 游戏将使用硬编码默认值，导致设置界面显示异常。

### 3.2 拆分 PCK

```bash
cd /mnt/data/catpaw/home/.meituan-catpaw/333630993/desk_default_workspace

python3 sourceofmana/tools/split_pck.py \
    binary/Web/index.pck \
    binary/Web/ \
    sourceofmana
```

**脚本做什么**：

1. 读取完整 `index.pck`（Godot 4.7 PCK v4 格式）
2. 识别地图相关文件（`.scn` 含 TileMapLayer、`.res` 含 MapData/MapServerData/NavigationPolygon）
3. 识别地图纹理（`.ctex`，按文件名匹配地图名）
4. 识别地图 remap 文件（`presets/maps/*.remap`）
5. 识别音乐文件（`.oggvorbisstr` 文件名含 ` - `）
6. 分类输出：
   - **core.pck**：登录/角色/起始地图(Drazil+Tulimshar)/音效/UI/字体/设置
   - **audio.pck**：音乐 .ogg + .tres + 从项目目录补充的 .tres
   - **maps.pck**：其他所有地图场景/纹理/remap
7. 自动更新 `index.html` 的 `GODOT_CONFIG`：
   - `fileSizes` 指向 `core.pck` 的大小
   - 添加 `"mainPack":"core.pck"`
8. 自动 bump service worker 缓存版本（如果存在）

**输出示例**：

```
  source: ver=4 godot=4.7.1 flags=2 entries=892 file_base=104 dir_base=84664 size=75599284
  total entries: 892
  found 89 map-related files for 12 maps
  found 2 map textures
  found 2 map remap files
  core maps: 4 files, 1 textures, 1 remaps
  other maps: 85 files, 1 textures, 1 remaps
  ...
  audio: 63  core: 749  maps: 87  removed: 63

  core.pck: 29.0 MB
  maps.pck: 16.1 MB
  audio.pck: 26.1 MB
SPK_DONE
```

### 3.3 整理文件布局

```bash
cd binary/Web

# 删除完整导出（已拆分，不再需要）
rm -f index.pck

# 移动按需下载的 PCK 到 packs/ 子目录
mkdir -p packs
mv audio.pck maps.pck packs/
```

**最终布局**：

```
binary/Web/
├── index.html              ← GODOT_CONFIG.mainPack = "core.pck"
├── index.js
├── index.wasm
├── index.side.wasm
├── index.png / index.icon.png / index.apple-touch-icon.png
├── index.audio.worklet.js / index.audio.position.worklet.js
├── libgdsqlite.web.template_release.wasm32.nothreads.wasm
├── libsentry.web.release.wasm32.nothreads.wasm
├── core.pck                ← 首次加载（~30MB）
└── packs/
    ├── audio.pck           ← 按需下载（~26MB）
    └── maps.pck            ← 按需下载（~16MB）
```

### 3.4 验证 index.html

```bash
grep 'GODOT_CONFIG' binary/Web/index.html
```

**期望输出**：

```javascript
const GODOT_CONFIG = {"args":[],"canvasResizePolicy":2,...,
  "fileSizes":{"core.pck":30438388,"index.wasm":1508095},...
  "godotPoolSize":4,"mainPack":"core.pck"};
```

> ⚠️ 必须包含 `"mainPack":"core.pck"`，否则 Godot 会尝试加载默认的 `index.pck`。

---

## 4. Brotli 压缩传输

### 4.1 原理

nginx 对 `.pck` 文件启用 Brotli 压缩，浏览器 `fetch()` **自动解压**，
Godot 收到的是解压后的原始数据。**无需任何客户端代码修改**。

> 📖 Godot Web 的 `HTTPRequest` 在 Web 平台底层使用浏览器 `fetch()` API，
> 浏览器会自动处理 `Content-Encoding: br` 响应头并解压响应体。

### 4.2 nginx 配置

```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;

    # ... SSL 配置 ...

    # ===== PCK 文件 Brotli 压缩 =====
    # 必须放在所有 location 之前，优先级最高
    location ~* \.pck$ {
        brotli_static on;       # 优先使用预压缩的 .br 文件
        brotli on;              # 无 .br 时动态压缩
        brotli_comp_level 6;
        brotli_min_length 256;
        brotli_types
            application/octet-stream
            application/wasm
            application/javascript
            application/json
            text/plain
            text/css
            text/javascript
            font/ttf font/otf
            image/svg+xml;

        root /var/www/som;
        try_files $uri =404;
    }

    # WebSocket 代理
    location /ws {
        proxy_pass http://127.0.0.1:27012;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 86400;
    }

    # 静态文件
    location / {
        root /var/www/som;
        try_files $uri $uri/ =404;
    }
}
```

> ⚠️ **不要**添加 `add_header Content-Encoding identity;`，这会覆盖 Brotli 的
> `Content-Encoding: br` 头，导致压缩失效。

### 4.3 压缩效果

| 文件 | 原始大小 | Brotli 压缩后 | 压缩率 |
|------|---------|--------------|--------|
| core.pck | 30.9 MB | 24.6 MB | 80% |
| maps.pck | 16.3 MB | 3.5 MB | 21% |
| audio.pck | 27.4 MB | 26.4 MB | 96% |

### 4.4 验证 Brotli 生效

```bash
# 检查 Content-Encoding 头
curl -sI -H 'Accept-Encoding: br' --insecure \
    'https://your-domain.com/game/core.pck' | grep -i content-encoding
# 期望输出: Content-Encoding: br

# 测量实际传输大小
curl -s -o /dev/null -w '%{size_download}' -H 'Accept-Encoding: br' \
    --insecure 'https://your-domain.com/game/core.pck'
# 期望: ~24.6MB（压缩后）
```

> 💡 游戏加载进度显示的 MB 数是**解压后**的大小（因为浏览器先解压再交给 Godot），
> 不是网络传输大小。验证 Brotli 必须用 curl 或 DevTools Network 面板。

---

## 5. 部署到 VPS

### 5.1 一键部署脚本

```bash
cd /mnt/data/catpaw/home/.meituan-catpaw/333630993/desk_default_workspace

# 打包
cd binary/Web && tar czf ../web_export.tar.gz . && cd ../..

# 上传并解压到 VPS
python3 << 'PYEOF'
import pexpect

HOST = "154.222.30.52"
PORT = 65397
USER = "root"
PASSWORD = "***"
REMOTE_DIR = "/var/www/som/game"

# 上传
cmd = f"scp -o StrictHostKeyChecking=no -P {PORT} web_export.tar.gz {USER}@{HOST}:/tmp/"
child = pexpect.spawn(cmd, timeout=300)
child.expect(r"[Pp]assword:")
child.sendline(PASSWORD)
child.expect(pexpect.EOF)

# 解压
cmd = f"ssh -o StrictHostKeyChecking=no -p {PORT} {USER}@{HOST}"
child = pexpect.spawn(cmd, timeout=30)
child.expect(r"[Pp]assword:")
child.sendline(PASSWORD)
child.expect(r"[#$]")

child.sendline(f"rm -rf {REMOTE_DIR}/*")
child.expect(r"[#$]")
child.sendline(f"cd {REMOTE_DIR} && tar xzf /tmp/web_export.tar.gz && rm /tmp/web_export.tar.gz")
child.expect(r"[#$]", timeout=120)
child.sendline(f"chown -R nginx:nginx {REMOTE_DIR}")
child.expect(r"[#$]")
child.sendline("exit")
child.close()
PYEOF
```

### 5.2 服务器端验证

```bash
# 检查文件
ls -la /var/www/som/game/
ls -la /var/www/som/game/packs/

# 检查 Brotli
curl -sI -H 'Accept-Encoding: br' --insecure \
    'https://localhost/game/core.pck' -H 'Host: 154.222.30.52' | grep -i content-encoding

# 重载 nginx
nginx -t && nginx -s reload
```

---

## 6. 按需下载机制

游戏启动后，`PackManager` 会自动下载缺失的 PCK：

| 触发时机 | 下载文件 | 大小 |
|---------|---------|------|
| 登录/选角后进入世界 | `packs/audio.pck` | ~26MB |
| 切换到非起始地图 | `packs/maps.pck` | ~16MB |

**下载 URL 构造**（`PackManager.gd`）：

```gdscript
# 使用 window.location.href 的 base dir 拼接相对路径
var base := window.location.href.get_base_dir()
var url := base + "packs/" + pack_name
```

> ⚠️ 必须用 `window.location.href.get_base_dir()`，不能硬编码绝对路径，
> 否则在子路径部署（如 `/game/`）时会 404。

---

## 7. 常见问题与经验教训

### 7.1 设置界面显示异常（Render-Scaling 默认为 1 而非 2）

**根因**：`export_presets.cfg` 的 `exclude_filter` 排除了 `data/conf/*`，
导致 `settings.cfg` 没有被打入 PCK。

**修复**：
```ini
# export_presets.cfg Web 预设
include_filter="data/db/*,data/conf/*,data/themes/*"  # 加入 data/conf
exclude_filter="...,data/maps/*,..."                  # 移除 data/conf
```

### 7.2 Godot 导出后 project.godot 被改写

**现象**：`project.godot` 中的 `window/stretch/aspect="ignore"` 等设置在导出后被删除。

**原因**：Godot 导出流程会移除与默认值相同的项目设置。

**解决**：需要持久化的设置必须在代码中运行时设置（`Root.set_content_scale_*`），
或接受导出后的值。

### 7.3 竖屏拉伸问题

**原版行为**：`stretch/aspect="ignore"` → 1280×720 拉伸铺满屏幕。

**稳定版分支验证**：`feat/agent-arena` 分支没有任何 stretch 覆盖代码，
纯靠 `project.godot` 的 `ignore` 实现铺满。

**结论**：不要自行添加 `set_content_scale_aspect(1)`（keep）来做黑边，
这会导致 UI 缩小。保持 `ignore` 拉伸即可。

### 7.4 音乐文件重复打包

**根因**：`is_audio()` 函数没有匹配 `.godot/imported/` 下的音乐文件路径。

**修复**：检查文件名是否含 ` - `（艺术家 - 歌曲名）来识别音乐：
```python
def is_audio(path):
    if '.oggvorbisstr' in path:
        filename = path.split('/')[-1]
        # 去掉 .oggvorbisstr 后缀和 32 位 hash
        if filename.endswith('.oggvorbisstr'):
            filename = filename[:-13]
        if '-' in filename:
            parts = filename.rsplit('-', 1)
            if len(parts[1]) == 32:
                filename = parts[0]
        return ' - ' in filename
    return False
```

### 7.5 音乐 .remap 文件导致 Godot 崩溃

**现象**：包含 `presets/music/*.tres.remap` 会导致 Godot 加载失败。

**解决**：在拆分时**完全移除**音乐 .remap 文件，用原始 .tres 替代：
```python
# 移除音乐 remap
if is_music_remap(path):
    continue
```

### 7.6 PCK 目录偏移必须 4 字节对齐

**现象**：拆分后的 PCK 报错 `Couldn't load project data`。

**原因**：Godot 4.7 PCK v4 格式要求目录区（directory）起始偏移 4 字节对齐。

**修复**：写入目录前填充 `\x00` 至 4 字节对齐：
```python
pad = _get_pad(4, len(out))
out += b'\x00' * pad
```

### 7.7 浏览器缓存旧 PCK

**现象**：部署后浏览器仍加载旧版 PCK。

**解决**：
1. 强制刷新（Ctrl+Shift+R / 清除浏览器缓存）
2. 清除 IndexedDB（Godot 的 `/userfs` 数据库）
3. 或修改 `index.html` 的 `fileSizes` 值（已自动处理）

---

## 8. 完整构建 Checklist

```
□ 1. 修改代码/设置
□ 2. Godot 导出: ./tools/Godot_v4.7.1-stable_linux.x86_64 --headless --export-release "Web"
□ 3. 拆分 PCK: python3 sourceofmana/tools/split_pck.py binary/Web/index.pck binary/Web/ sourceofmana
□ 4. 删除 index.pck: rm binary/Web/index.pck
□ 5. 移动 packs: mkdir -p binary/Web/packs && mv binary/Web/audio.pck binary/Web/maps.pck binary/Web/packs/
□ 6. 验证 index.html: grep 'mainPack' binary/Web/index.html
□ 7. 本地沙箱测试（python3 -m http.server + Playwright）
□ 8. 打包: cd binary/Web && tar czf ../web_export.tar.gz .
□ 9. 部署: scp + ssh 解压到 VPS
□ 10. 验证线上: curl 检查 Content-Encoding: br
□ 11. 提交: git add + git commit + git push
```

---

## 9. 相关文件索引

| 文件 | 说明 |
|------|------|
| `tools/split_pck.py` | PCK 拆分脚本（Godot 4.7 PCK v4） |
| `export_presets.cfg` | Godot 导出预设（Web 预设） |
| `html/custom_html_shell.html` | 自定义 HTML 模板 |
| `data/conf/settings.cfg` | 默认游戏设置 |
| `sources/launcher/Launcher.gd` | 启动器（PCK 加载） |
| `sources/system/PackManager.gd` | 按需下载管理 |
| `goserver/deploy.sh` | Go 服务器一键部署 |
| `goserver/WEB_DEPLOYMENT.md` | Web 登录部署指南 |
