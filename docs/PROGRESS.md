# SoM AI 竞技场 — 进度台账

> 项目：Source of Mana（Godot 4）AI 原生竞技场。开工包：元宝 v1.0（T1-T14 / M0-M5）。
> 底座：~/sourceofmana @ feat/agent-arena（上游 master=1933bd6）。Godot 4.7.1 已装 /usr/local/bin/godot471。
> 红线：① 只跑自建实例，永不连官方服；② 旧 Kaetram 全部轨迹冻结（不迁移/不训练/不发布）；③ 禁 isAgent 业务短路；④ 信息对等。

## 当前阶段

**P0-Mana spike**（M0 许可结论书 → M1 起服 → M2 缺口表 → M3 四元组阀门 → M4 锁步 → M5 闭环）

## 已完成

- 2026-11-06 T2：切 feat/agent-arena 分支；godot471 4.7.1 实机验证（--headless --import + --server 起服、ENet 6119 监听、pyenet CONNECT ✅，见会话记录）
- 2026-11-06 T1：M0 许可结论书草案 docs/LICENSE-CONCLUSION-v0.1.md（LICENSE.md/CC BY-SA 4.0/credits.json/agreement.json 实文核实；tsc 未取到=开放风险登记）
- S17 版本机制已核清（会话内完成）：Network.gd:624 `ComputeProtocolVersion(self)` 传 Network 节点自身 → **Network.gd 加 @rpc 必改哈希；独立 AgentBridge.gd autoload 不影响** → M3 通道 1（独立 autoload+复用 6108）可行

## 进行中

- （M3 收口中）四元组实测闭环已达成，待 M4

## M3 里程碑：✅ 核心闭环达成（2026-11-06）

- **无头 agent 客户端**：AgentClientProbe.gd autoload，`godot471 --headless --path . -- --agent-client <name>` 全自动 连接→注册→登录→建角色→ENTERED_GAME（约 7s）
- **Launcher.gd 上游改动 1 行**：--agent-client 时跳过自起 server（已记清单）
- **四元组实测**：observe(list_players+snapshot: pos/health 112/112/level/alive) → apply_action(navigate, ACTION_OK) → **位置真实移动 dx=33.4 dy=16.6**，服务器 0 脚本错误
- **verdict 仍 PENDING_PROBE**：下一阶段（M4 锁步）做状态增量裁决回执 + events 订阅
- commit: "feat(m3): headless agent client probe + bridge snapshot/navigate E2E verified"

## 环境事实（复用）

- 导出模板已装 ~/.local/share/godot/export_templates/4.7.1.stable/（linux_* 8 个）；导出命令：`godot471 --headless --path . --export-release "Linux/X11 Headless Server" ~/binary/Server/SourceOfMana.x86_64`（154MB embed pck）
- 导出目录须手工补 GDExtension 库：mkdir -p addons/{godot-sqlite/bin,sentry/bin/linux/x86_64} 并 cp libgdsqlite…so / libsentry…so（pck 里的 .gdextension 找 bin/*.so 相对路径）
- 导出包验证过的启动：`./SourceOfMana.x86_64 --headless --server --port 6119`（SQLite 建库+ENet/WebSocket/WebRTC init）
- GDScript 坑：`\x00` 转义非法（用 "\0"）；StreamPeerTCP 无 STATUS_DISCONNECTED（判非 CONNECTED/CONNECTING）；put_utf8_string 带 32 位长度前缀（NDJSON 用 put_data(to_utf8_buffer())）；Util 只有 PrintLog/PrintInfo 无 PrintErr（用 printerr）；server 类 class_name=NetServer（非 Server）
- 同机并发注意：另一个会话曾占用 9101（godot471 --server 探针进程），起服前先 `ss -ltn | grep 9101`

## M1 状态：✅ 完成（T3 导出成功 + T4 隔离八项过 + T5 基线 225MB）
## M2 状态：✅ 完成（缺口表 v0.1，S21 解答：导航数据在 presets/maps/navigations）
## M3 状态：✅ 通道 1 已定案并落地 AgentBridge（独立 autoload 不影响 ProtocolVersion 实测确认：bridge 起、协议版本不变、WS 客户端兼容路径未动）
## M4 状态：✅ 裁决回执 + events 闭环（2026-11-06 自审实机验证）
- verdict：navigate=position_changed APPLIED（33ms）；attack=target_health_decreased APPLIED（delta=12，打 Elanore 实测）；超时 1.2s → NO_EFFECT/REJECTED
- events：agent_damaged/agent_killed/enemy_killed 信号钩子 + drain_events（seq cursor，512 上限）
- nearby：±480 实体扫描（monster/player/npc，距离排序）→ attack 目标选择闭环
- say 动作只走 LOCAL 频道（Discord 红线）
- 探针脚本 tests/bridge_probe.py（ping/players/snapshot/navigate+verdict/attack+verdict/events 全绿）
- 待 M5：kill 事件实测（需打死一只怪）、明确被拒回执场景、资源基线 P95

## 待办（按开工包顺序）

- T5 资源基线、T7-T9 M3 阀门、T10 契约冻结、T11 M4 锁步、T12 M5 四元组闭环、T13 汇总、T14 旧仓 tag 归档
- 待所有者：签署许可结论书（M0 退出条件）；P4 对外冻结解除评审

## 跳过/挂账

- tsc 全文（policies.themanaworld.org/tsc）网络未取到 — 开放风险，不阻塞
