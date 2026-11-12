# SoM 人类玩家精简客户端

参考「网游脱机挂」的极简画面系统，为人类玩家打造的 SoM（Source of Mana）精简客户端。
与 AI 无头客户端共用同一套协议层（`som_protocol.py`），但多了一个看得见的窗口。

## 体积对比

| | 完整 Web 客户端 | AI 无头客户端 | 人类精简客户端 |
|---|---|---|---|
| **客户端文件** | 140 MB | 15 KB | 56 KB |
| **协议层** | 内置 | som_protocol.py（56 KB） | som_protocol.py（56 KB） |
| **内存占用** | 400+ MB（浏览器） | 25 MB | 60 MB |
| **依赖** | 浏览器 + wasm 运行时 | Python 3.8+ | Python 3.8+ / tkinter |
| **渲染** | GPU/Canvas | 无 | CPU（tkinter Canvas） |
| **音频** | 需要 | 无 | 无 |
| **操作** | 键鼠 | AgentBridge | 键鼠 + 点击 |

## 快速开始

### 1. 安装依赖

```bash
pip install websockets
# tkinter 通常随 Python 自带；若缺失：
#   Debian/Ubuntu: apt install python3-tk
#   Alpine: apk add py3-tkinter
```

### 2. 启动客户端

```bash
python3 lite_human_client.py ws://your-server:27012/ws \
    --account 你的账号 \
    --password 你的密码 \
    --nick "你的角色名" \
    --auto-create
```

首次使用可以只填服务器地址，在登录窗口里输入账号密码，勾选
「自动注册账号 / 创建角色」后点「连接」。

**参数说明：**

| 参数 | 环境变量 | 说明 |
|---|---|---|
| `url`（位置参数） | `SOM_WS_URL` | SoM 服务器 WebSocket 地址 |
| `--account` | `SOM_ACCOUNT` | 账号名 |
| `--password` | `SOM_PASSWORD` | 密码 |
| `--nick` | `SOM_NICK` | 角色昵称（留空则用账号名） |
| `--auto-create` | `SOM_AUTO_CREATE=1` | 账号/角色不存在时自动创建 |
| `--auto-play` | `SOM_AUTO_PLAY=1` | 进游戏后立即开启自动挂机 |
| `--debug` | — | 输出调试日志 |

### 3. 操作

| 操作 | 效果 |
|---|---|
| 左键点空地 | 移动到该坐标（本地预测 + 服务器确认） |
| 左键点 NPC / 怪物 | 选中目标并攻击 |
| 左键点掉落物 | 走过去并拾取 |
| 左键点玩家 | 选中该玩家 |
| 右键点空地 | 移动（不选中任何目标） |
| `W A S D` / 方向键 | 持续行走 |
| 鼠标滚轮 | 缩放视图（35%–250%） |
| 工具栏 `+` / `−` | 缩放视图 |
| 工具栏 `截图` | 保存当前画面为 PNG |
| 工具栏 `断开` | 断开与服务器的连接 |

## 界面布局

```
┌──────────────────────────────────────────────────────────┐
│ SoM 精简客户端  已连接 peer=15                断开 截图 100% + − │
├──────────────────────────────────────────────┬───────────┤
│                                              │ 角色状态   │
│                                              │  HP / MP  │
│            视 图 口（Canvas）                 │  等级 经验 │
│                                              │  坐标 地图 │
│   蓝点 = 自己  绿点 = 玩家  橙点 = 友好NPC     ├───────────┤
│   红点 = 怪物  黄菱形 = 掉落物                 │ 目标      │
│                                              │  攻击 拾取 │
│                                              │  交互 复活 │
│                                              │  喝药      │
│                                              ├───────────┤
│                                              │ 自动挂机   │
│                                              │  ☑ 打怪   │
│                                              │  ☑ 拾取   │
│                                              │  ☑ 喝药   │
│                                              │  阈值 30% │
│                                              ├───────────┤
│                                              │ 统计      │
│                                              │ 击杀 拾取 │
│                                              │ 死亡 伤害 │
│                                              ├───────────┤
│                                              │ 小地图    │
├──────────────────────────────────────────────┴───────────┤
│ 日志（INFO / WARNING / ERROR / CHAT / LOOT）              │
├──────────────────────────────────────────────────────────┤
│ [local ▼] [输入聊天内容……]                          [发送] │
└──────────────────────────────────────────────────────────┘
```

### 视图元素

- **网格**：100 px 细格 / 500 px 粗格，原点在画面中心时显示坐标轴。
- **相机**：平滑跟随角色（lerp 0.18），进游戏或换地图时自动回中。
- **血条**：实体头顶，绿色为友方、红色为敌方。
- **目标高亮**：当前选中目标带白色虚线描边。
- **特效**：点击移动标记（X）、伤害飘字（白色普通 / 橙色暴击 / 灰色闪避）、
  攻击连线，均带颜色渐变淡出。
- **死亡实体**：显示为灰色扁平椭圆。

### 自动挂机

勾选「自动打怪 / 自动拾取 / 自动喝药」后点「应用挂机设置」：

- **自动打怪**：寻找最近的敌对 NPC（`friendly=False`），移动至攻击距离后
  循环释放技能。
- **自动拾取**：寻找最近的掉落物，移动到 24 px 内后拾取。
- **自动喝药**：HP 低于阈值（默认 30%）时自动使用背包里的药水。

再点「停止全部挂机」可随时终止。

## 技术细节

### 协议层

与 AI 客户端共用 `som_protocol.py`：

- Godot 4 SceneMultiplayer 二进制 RPC 协议，WebSocket 传输。
- Variant 编解码：BOOL / INT（2-bit 压缩）/ FLOAT / STRING / STRING_NAME /
  VECTOR2 / DICTIONARY / ARRAY / PACKED_STRING_ARRAY / NIL。
- RPC name ID = `sources/network/Network.gd` 中 119 个 `@rpc` 方法排序后的下标。
- 实体身份：`agentRID == CharID`，正数为玩家、负数为 NPC/怪物；
  `PreloadEntity` 用正 RID 表示种子 NPC（charID 取负），
  竞技场怪物则直接用负 charID。客户端用 `friendly` 字段区分敌我。

### 线程模型

- **主线程**：tkinter 事件循环，负责全部 UI 绘制与交互。
- **后台线程**：asyncio 事件循环，负责 WebSocket 收发与自动挂机逻辑。
- **桥接**：UI 通过 `queue.Queue` 向后台提交指令；后台通过
  `world.emit` → `ui_queue` 把世界事件推给 UI 线程，
  `Application._tick()` 每 50 ms 拉取一次并刷新画面。

### 移动预测

SoM 的移動是客户端权威的：服务器不会把移动者的位置广播回来。
客户端以 120 px/s 本地推算自身位置，收到服务器位置时再校正，
因此点击移动在弱网下也不会「卡顿回弹」。

## 文件位置

- GitHub: `goserver/lite_human_client.py`
- 协议层: `goserver/som_protocol.py`
- AI 无头客户端: `goserver/headless_ai_client.py`
- AI 客户端文档: `goserver/AI_CLIENT_README.md`
