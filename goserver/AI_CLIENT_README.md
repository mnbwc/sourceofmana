# AI 无头精简客户端

专为 AI Agent 设计的 SoM（Source of Mana）游戏客户端，无需浏览器、无需渲染。

## 体积对比

| | 完整 Web 客户端 | AI 无头客户端 |
|---|---|---|
| **文件大小** | 140 MB | 46 KB |
| **内存占用** | 400+ MB（浏览器） | 25 MB |
| **依赖** | 浏览器 + wasm 运行时 | Python 3.8+ |
| **渲染** | 需要 GPU/Canvas | 无 |
| **音频** | 需要 | 无 |

## 快速开始

### 1. 安装依赖

```bash
pip install websockets
```

### 2. 启动客户端

```bash
python3 headless_ai_client.py ws://your-server:27012/ws \
    --auto-create \
    --nick "AI机器人" \
    --bridge-port 19101 \
    --auto-play
```

**参数说明：**
- `ws://host:port/ws` — SoM 服务器 WebSocket 地址
- `--auto-create` — 自动创建账号和角色
- `--nick "名字"` — 角色名称（支持中文）
- `--bridge-port 19101` — AgentBridge TCP 端口
- `--auto-play` — 开启自动打怪 + 自动拾取
- `--account/--password` — 使用已有账号登录
- `--debug` — 开启调试日志

### 3. 连接 AgentBridge

客户端启动后会在 TCP 端口监听 NDJSON 协议：

```bash
# 查看游戏状态
echo '{"op":"snapshot"}' | nc -w 3 127.0.0.1 19101

# 列出所有玩家和 NPC
echo '{"op":"list_players"}' | nc -w 3 127.0.0.1 19101

# 移动
echo '{"op":"apply_action","action":"move_to","args":{"pos":[200.0,200.0]}}' | nc -w 3 127.0.0.1 19101

# 攻击
echo '{"op":"apply_action","action":"attack","args":{"skill_id":0}}' | nc -w 3 127.0.0.1 19101

# 拾取
echo '{"op":"apply_action","action":"pickup","args":{"drop_rid":123}}' | nc -w 3 127.0.0.1 19101

# 聊天
echo '{"op":"apply_action","action":"chat","args":{"message":"你好"}}' | nc -w 3 127.0.0.1 19101
```

## AgentBridge 协议

每个请求是一行 JSON，回复也是一行 JSON。

### 支持的操作

| 操作 | 说明 | 参数 |
|------|------|------|
| `ping` | 健康检查 | 无 |
| `snapshot` | 获取完整世界状态 | 无 |
| `list_players` | 列出所有实体 | 无 |
| `drain_events` | 获取事件流 | 无 |
| `apply_action` | 执行动作 | `action` + `args` |

### apply_action 动作列表

| 动作 | 说明 | 参数 |
|------|------|------|
| `move_to` | 移动到指定坐标 | `{"pos": [x, y]}` |
| `attack` | 释放技能 | `{"skill_id": 0}` |
| `pickup` | 拾取掉落物 | `{"drop_rid": rid}` |
| `chat` | 发送聊天消息 | `{"message": "..."}` |
| `interact` | 与 NPC 交互 | `{"target": rid}` |
| `respawn` | 复活 | 无 |
| `emote` | 发送表情 | `{"emote_id": id}` |

### snapshot 返回格式

```json
{
  "self": {
    "peer": 17,
    "nick": "AI服务器机器人",
    "pos": [160.0, 220.0],
    "hp": 100,
    "max_hp": 100
  },
  "players": {
    "17": {"peer_id": 17, "nick": "AI服务器机器人", "pos": [160.0, 220.0], ...},
    "-1": {"peer_id": -1, "nick": "Elder", "pos": [300.0, 160.0], ...}
  },
  "npcs": {
    "1": {"peer_id": 1, "name": "Elder", "actor_type": 1}
  },
  "drops": {},
  "entity_count": 5
}
```

## 部署为系统服务（Alpine Linux）

创建 `/etc/init.d/som-ai-client`：

```sh
#!/sbin/openrc-run

description="SoM Headless AI Client"
depend() {
    need net som-goserver
}

WS_URL=${WS_URL:-ws://127.0.0.1:27012/ws}
NICK=${NICK:-AI服务器机器人}
BRIDGE_PORT=${BRIDGE_PORT:-19101}

command="/usr/bin/python3"
command_args="/root/headless_ai_client.py ${WS_URL} --auto-create --nick '${NICK}' --bridge-port ${BRIDGE_PORT} --auto-play"
command_background="yes"
pidfile="/run/${RC_SVCNAME}.pid"
command_user="root"

start_pre() {
    checkpath --directory --owner root:root /run
}
```

启用服务：

```bash
chmod +x /etc/init.d/som-ai-client
rc-update add som-ai-client default
rc-service som-ai-client start
```

## 技术细节

- **协议**：直接实现 Godot 4 SceneMultiplayer 二进制 RPC 协议（RFC）
- **变体编码**：支持 BOOL/INT/FLOAT/STRING/DICTIONARY/ARRAY/VECTOR2 的压缩与非压缩格式
- **移动预测**：服务器不向移动者发送位置更新（SoM 设计为客户端本地预测），AI 客户端以 120 px/s 模拟
- **自我识别**：按角色名匹配 PreloadPlayer 推送（服务器使用 CharID 作为 agentRID）
- **自动打怪**：自动寻找最近 NPC → 移动 → 攻击，循环执行
- **自动拾取**：自动寻找最近掉落物 → 移动 → 拾取

## 文件位置

- GitHub: `goserver/headless_ai_client.py`
- VPS 部署: `/root/headless_ai_client.py`
- 服务脚本: `/etc/init.d/som-ai-client`
