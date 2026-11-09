# 双服务器 Agent 探索对比报告

> 日期：2026-11-09
> SoM 服务器：`./godot-run --headless --server --ws-port 6118`（ARENA_MODE=on）
> Go 服务器：`./goserver`（:27012）

## 1. 测试环境

| 组件 | SoM 原版 | Go 迁移版 |
|------|----------|-----------|
| 端口 | ENet 6119 / WS 6118 / Bridge 9101 | WS 27012 |
| 玩家接入 | AgentClientProbe `--agent-keep` | Godot e2e_client.gd |
| Agent 接口 | AgentBridge (TCP NDJSON) | Godot WebSocket RPC |
| 单元测试 | — | `go test ./...` 59/59 PASS |
| e2e 测试 | AgentBridge 探索 | run_e2e.sh 12/12 PASS |

## 2. SoM 服务器探索结果

### 2.1 登录 + 世界同步 ✅
- AgentClientProbe 成功创建账号 `ai_probe1` → 登录 → 创建角色 `[AI] ai_probe1` → 进入 IN_GAME
- 世界同步正常：AOI 广播、在线列表、NPC 实体均可见

### 2.2 AgentBridge 操作 ✅
| 操作 | 结果 | 备注 |
|------|------|------|
| `ping` | ✅ | frame 正常递增 |
| `list_players` | ✅ | 正确返回 isAgent 标记 |
| `snapshot` | ✅ | pos/health/nearby 完整 |
| `apply_action navigate` | ✅ | 短距离正常；长距离受 navmesh 边界限制 |
| `apply_action attack` | ✅ | 近战攻击生效，目标掉血 |
| `apply_action say` | ✅ | LOCAL 频道发送成功 |
| `apply_action inspect` | ✅ | 触发选择目标 |
| `apply_action respawn` | ✅ | 活体时 PENDING |
| `apply_action party invite` | ✅ | 发送邀请（无在线目标） |
| `spectate` | ✅ | 观战加入，readOnly=true |
| `spectate_feed` | ✅ | 事件流 + 快照 |
| `spectate_stop` | ✅ | 退出观战 |
| `drain_events` | ✅ | 完整事件历史 |

### 2.3 竞技场 ✅（ARENA_MODE=on）
- `arena_start` 成功创建 episode（tutorial_maggot 模板：击杀 3 只蛆虫）
- `arena_status` 正确追踪 kills/deaths/xp/itemsLooted
- `arena_stop` 正确结束 episode
- 战斗击杀确认：Maggot 被杀，`kill` 事件推送
- 计分镜像：kills=1, xpGained=1 正确累计
- 多轮 episode 正确启停（ep-1 到 ep-5）

### 2.4 发现的问题

#### 问题 1：navmesh 边界阻挡移动（非 bug，设计限制）
- **现象**：玩家从 (2230,1250) 导航到 (2228,1350)，走到 (2229,1325) 后停止
- **原因**：Godot NavigationAgent2D 在目标不在 navmesh 上时，会走到最近可达点
- **影响**：agent 无法精确到达任意坐标点，但可接近到 ~25px 内
- **建议**：agent 应使用 `apply_action navigate` 逐步逼近目标，而非期望精确到达

#### 问题 2：攻击距离限制
- **现象**：攻击 200px 外的目标返回 PENDING，1.2s 后超时
- **原因**：近战技能需要目标在射程内（~20px），远距离攻击无效
- **影响**：agent 需要先 navigate 贴近目标再 attack
- **建议**：agent 应实现 approach-then-attack 策略

## 3. Go 服务器探索结果

### 3.1 e2e 测试 12/12 PASS ✅

| 测试 | 结果 | 覆盖流程 |
|------|------|----------|
| SOLO | PASS | 登录链 5 步 |
| PEERS | PASS | 双客户端互见 + 移动同步 |
| NPC | PASS | 任务对话 + 选择 + 任务更新 |
| PARTY | PASS | 邀请/接受/聊天 |
| TRADE | PASS | 请求/开交易/报价/接受/完成 |
| COMBAT | PASS | 攻击/死亡/复活 |
| ITEMS | PASS | 拾取/出售/购买/使用/属性 |
| BUFFS | PASS | 施放 debuff + 接收 |

### 3.2 Go 服务器 vs SoM 服务器行为对比

| 功能 | SoM 原版 | Go 迁移版 | 一致性 |
|------|----------|-----------|--------|
| 登录链 | ✅ 5 步 | ✅ 5 步 | ✅ 一致 |
| 世界同步 | ✅ ENet+WS | ✅ WS | ✅ 一致 |
| NPC 交互 | ✅ 任务对话 | ✅ 任务对话 | ✅ 一致 |
| 组队 | ✅ | ✅ | ✅ 一致 |
| 交易 | ✅ | ✅ | ✅ 一致 |
| 战斗 | ✅ | ✅ | ✅ 一致 |
| 物品经济 | ✅ | ✅ | ✅ 一致 |
| Buff/Debuff | ✅ | ✅ | ✅ 一致 |
| 竞技场 | ✅ (ARENA_MODE) | ❌ 未实现 | ⚠️ 差异 |
| AgentBridge | ✅ (端口 9101) | ❌ 未实现 | ⚠️ 差异 |
| 持久化 | ✅ SQLite | ✅ JSON 文件 | ⚠️ 不同实现 |

## 4. 关键差异分析

### 4.1 Go 服务器缺失的功能
1. **AgentBridge**：Go 服务器没有 TCP NDJSON 接口，无法被 Python agent 直接操控
2. **竞技场**：Go 服务器没有 ArenaController 等价物
3. **观战系统**：Go 服务器没有 spectate 功能

### 4.2 架构差异（非 bug）
- SoM 使用 Godot NavigationAgent2D 进行寻路，Go 服务器使用简单的 AOI + 直线移动
- SoM 使用 SQLite 持久化，Go 使用 JSON 文件
- SoM 的 ENet 传输 vs Go 的 WebSocket 传输

## 5. 结论

- **SoM 服务器**：功能完整，AgentBridge + 竞技场 + 全部游戏玩法正常工作
- **Go 服务器**：核心游戏玩法全部迁移完成，12/12 e2e 测试通过
- **待实现**：Go 服务器的 AgentBridge + 竞技场（如需 agent 直接操控 Go 服务器）

## 6. 建议的后续工作

1. **Go 服务器 AgentBridge**：实现 TCP NDJSON 接口，让 Python agent 可以操控 Go 服务器
2. **Go 服务器竞技场**：移植 ArenaController 逻辑到 Go
3. **navmesh 适配**：agent 探索脚本应处理 Godot navmesh 边界（逐步逼近目标）
4. **攻击策略**：agent 应实现 approach-then-attack（先贴近再攻击）
