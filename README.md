# Source of Mana — Go Server

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://golang.org)
[![Godot](https://img.shields.io/badge/Godot-4.7.1-478CBF?logo=godot-engine&logoColor=white)](https://godotengine.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

> 2D MMORPG — 服务端从 GDScript 迁移到 Go，支持 PC / Android / Web 浏览器三端。

## 项目概述

Source of Mana (SoM) 是一个 2D MMORPG，基于 Godot 4.7.1 引擎。本项目将服务端从 GDScript 迁移到 Go，实现：

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
| Web 客户端 | 单线程 wasm + Web Worker 预编译 |

### 测试覆盖

| 测试 | 数量 | 状态 |
|------|------|------|
| Go 单元测试 | 72 | ✅ 全部通过 |
| e2e 测试（Godot 客户端） | 12 | ✅ 全部通过 |
| 浏览器 WebSocket | 1 | ✅ 通过 |

---

## 快速开始

### 1. 克隆仓库

```bash
git clone -b feat/agent-arena https://github.com/mnbwc/sourceofmana.git
cd sourceofmana
```

### 2. 启动 Go 服务器

```bash
cd goserver
go build -o goserver .
./goserver
```

服务器默认监听：
- `ws://0.0.0.0:27012` — WebSocket 游戏服务器
- `wss://0.0.0.0:27013` — WSS 安全 WebSocket（自签名证书）
- `tcp://127.0.0.1:9102` — AgentBridge

### 3. 运行测试

```bash
cd goserver
go test ./... -v
```

---

## 部署

### 系统要求

| 项目 | 最低要求 | 推荐 |
|------|---------|------|
| OS | Ubuntu 20.04 / Debian 11 / Alpine 3.18 | Ubuntu 24.04 / Alpine 3.21 |
| CPU | 1 核 | 2 核 |
| 内存 | 512 MB | 2 GB |
| 磁盘 | 1 GB | 10 GB SSD |
| Go | 1.24+ | 1.24+ |
| 网络 | 公网 IP | 公网 IP + 域名 |

### Ubuntu/Debian 一键部署

```bash
cd goserver
sudo bash deploy.sh your-domain.com
```

脚本自动完成：
1. 安装依赖（nginx, certbot, git, golang, sqlite）
2. 克隆/更新代码
3. 构建 Go 服务器
4. 配置 systemd 服务
5. 配置 nginx + WebSocket 代理
6. 获取 Let's Encrypt 证书
7. 配置防火墙

### Alpine Linux 一键部署

```bash
cd goserver
sudo sh deploy_alpine.sh your-domain.com
```

脚本自动完成：
1. 安装依赖（apk: nginx, git, go, sqlite, openssl）
2. 克隆/更新代码
3. 构建 Go 服务器（CGO_ENABLED=0，静态二进制）
4. 生成 TLS 自签名证书（ECDSA P-256）
5. 配置 nginx + WebSocket 代理
6. 配置 OpenRC 服务
7. 部署 Web 客户端

### 手动部署

详见：
- [`goserver/HANDOVER.md`](goserver/HANDOVER.md) — 部署交接文档
- [`goserver/WEB_DEPLOYMENT.md`](goserver/WEB_DEPLOYMENT.md) — Web 登录部署指南

---

## 网络架构

### 端口清单

| 端口 | 协议 | 绑定 | 用途 | 暴露 |
|------|------|------|------|------|
| 443 | TCP | 公网 | nginx HTTPS/WSS | ✅ 公网 |
| 80 | TCP | 公网 | HTTP→HTTPS 重定向 | ✅ 公网 |
| 27012 | TCP | 0.0.0.0 | Go WebSocket | ✅ 公网（AI 无头） |
| 27013 | TCP | 127.0.0.1 | Go WSS (自签名) | ❌ 仅本地 |
| 9102 | TCP | 127.0.0.1 | AgentBridge | ❌ 仅本地 |

### NAT 映射（VPS 环境）

**只需映射 3 个公网端口：**

| 公网端口 | 协议 | 指向 VPS 内部 | 说明 |
|---------|------|-------------|------|
| **11443** | TCP | 443 | 人类网页登录 (WSS) |
| **27012** | TCP | 27012 | AI 无头客户端登录 (WS) |
| **20080** | TCP | 80 | HTTP→HTTPS 重定向 |

### 数据流

```
浏览器 / PC / 安卓
    │
    │ wss://domain:11443/ws (TLS 由 nginx 终止)
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

## Web 客户端

### 预编译二进制

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

### 浏览器兼容性

| 浏览器 | 最低版本 | 说明 |
|--------|---------|------|
| Chrome | 80+ | ✅ 完全支持 |
| Firefox | 78+ | ✅ 完全支持 |
| Safari | 14+ | ✅ 完全支持 |
| Edge | 80+ | ✅ 完全支持 |

### 技术要点

- **单线程 wasm**：`thread_support=false`，无需 SharedArrayBuffer，不需要 COOP/COEP headers
- **Web Worker 预编译**：wasm 模块在 Worker 线程中预编译，避免阻塞主线程
- **进度条**：加载过程显示「初始化引擎 → 预编译资源 → 正在加载游戏资源」
- **WSS 连接**：通过 nginx 反向代理，TLS 终止在 nginx，Go 服务器使用明文 WebSocket

### 自定义导出

如需重新导出 Web 客户端：

1. 在 Godot 编辑器中打开项目
2. 选择 `Project > Export > Web`（preset: `Web`）
3. 导出到 `binary/Web_nothreads/`
4. 修改 `index.html` 和 `index.js` 以支持 Web Worker 预编译（参考现有文件）

详细导出配置见 [`export_presets.cfg`](export_presets.cfg)。

---

## 配置参考

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `BIND_IP` | 空（所有接口） | 绑定 IP 地址 |
| `WS_PORT` | 27012 | 明文 WebSocket 端口 |
| `WSS_PORT` | 27013 | TLS WebSocket 端口（留空则不启动） |
| `BRIDGE_PORT` | 9101 | AgentBridge 端口 |
| `TLS_CERT_FILE` | cert.pem | TLS 证书路径 |
| `TLS_KEY_FILE` | key.pem | TLS 私钥路径 |
| `SOM_DB_FILE` | som.db | SQLite 数据库路径 |
| `SOM_STORE_FILE` | som_store.json | JSON 迁移源（仅首次启动） |

### 数据库

SQLite 数据库包含 12 张表：

| 表 | 说明 |
|----|------|
| `players` | 玩家账号 |
| `characters` | 角色数据 |
| `inventory` | 物品栏 |
| `equipment` | 装备 |
| `skills` | 技能 |
| `quests` | 任务进度 |
| `npcs` | NPC 状态 |
| `parties` | 组队信息 |
| `trades` | 交易记录 |
| `arena` | 竞技场数据 |
| `logs` | 操作日志 |
| `config` | 服务器配置 |

---

## 开发

### 项目结构

```
sourceofmana/
├── goserver/              # Go 服务器
│   ├── main.go            # 主入口
│   ├── deploy.sh          # Ubuntu/Debian 部署脚本
│   ├── deploy_alpine.sh   # Alpine 部署脚本
│   ├── HANDOVER.md        # 部署交接文档
│   └── WEB_DEPLOYMENT.md  # Web 部署指南
├── sources/               # Godot 客户端源码
│   ├── launcher/          # 启动器
│   ├── network/           # 网络模块
│   └── ...                # 游戏逻辑
├── binary/
│   └── Web_nothreads/     # 预编译 Web 客户端
├── export_presets.cfg     # Godot 导出配置
└── project.godot          # Godot 项目配置
```

### 运行测试

```bash
# Go 单元测试
cd goserver
go test ./... -v

# e2e 测试（需要 Godot 编辑器）
# 在 Godot 编辑器中运行 test/ 目录下的场景
```

### 提交规范

```
feat: 新功能
fix: 修复
docs: 文档
refactor: 重构
test: 测试
chore: 构建/工具
```

---

## 故障排除

| 问题 | 原因 | 解决 |
|------|------|------|
| 浏览器显示安全警告 | 自签名证书 | 点击「高级」→「继续访问」 |
| WebSocket 连接失败 | nginx 未配置 Upgrade 头 | 检查 `proxy_set_header Upgrade` |
| 游戏加载卡在 100% | wasm 编译阻塞主线程 | 确认使用 `Web_nothreads` 版本 |
| `ERR_CONNECTION_REFUSED` | 服务器未启动 | 检查 `rc-service som-goserver status` |
| 数据库锁定 | 多进程访问 SQLite | 确保只有一个服务器实例运行 |

---

## 许可证

MIT License

---

## 贡献

欢迎提交 Issue 和 Pull Request！

---

## 致谢

- [Godot Engine](https://godotengine.org/) — 游戏引擎
- [gorilla/websocket](https://github.com/gorilla/websocket) — WebSocket 库
- [modernc.org/sqlite](https://modernc.org/sqlite) — 纯 Go SQLite 驱动
