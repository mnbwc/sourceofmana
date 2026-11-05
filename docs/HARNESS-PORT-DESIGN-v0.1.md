# AgentRuntime → SoM 平移设计 v0.1（2026-11-06，tick8）

目标：把 Kaetram 侧已验收的 AgentRuntime harness（~/kaetram/packages/agent-runtime，v0.1+P0-P4）平移到 SoM 底座，复用硬预算闸门 / 事件唤醒 / 三件套落盘 / configHash 归因，只替换 I/O 适配层。

## 1. 底座差异（Kaetram HTTP API vs SoM AgentBridge TCP/JSON）

| 维度 | Kaetram | SoM |
|---|---|---|
| 传输 | HTTP POST/GET（:9100） | TCP 长连接 + 换行 JSON（bridge 端口，默认 9101） |
| 会话 | POST /v1/sessions 创建 | 由探针（AgentClientProbe --agent-keep）登录游戏产生，bridge 以 peerID 标识 |
| 观测 | GET /state | op=snapshot（player/nearby/health） |
| 事件 | GET /events?since=N | op=drain_events（cursor 语义同 since） |
| 动作 | POST actions/* | op=apply_action（navigate/inspect/attack/loot/say） |
| 回执 | HTTP 响应即时 verdict | ack=PENDING + op=latest_verdict 轮询终局（APPLIED/REJECTED/NO_EFFECT/UNVERIFIABLE） |
| 生命周期 | DELETE /v1/sessions/:id + TTL reaper | 无对应 op——**gap G1**：探针退出即断线，bridge 无 destroy/TTL |
| 计分 | Arena Controller 4 端点 + score 事件 | **gap G2**：无 episode/arena 概念，score 未聚合 |

## 2. 平移架构（三层，只动适配层）

```
policies.py / events.py / runtime.py / 存储三件套   ← 原样平移（Kaetram 经验直迁）
sombase client.py（适配层）                          ← 重写：TCP JSON，verdict 轮询封装成同步语义
AgentBridge.gd（已上线，402 行）                     ← 小增量：G1 session 管理 + kill 事件确认
```

- **client.py（SoM 适配）**：单 TCP 连接多路 peerID；`state(peerID)`→snapshot、`drain_events` 带 cursor 重连续传、`act()` 发 apply_action 后内部轮询 latest_verdict（0.3s 间隔，上限 6s）再返回，向 runtime 暴露与 Kaetram 相同的 `decide→execute` 同步接口。budget 闸门/事件唤醒判定全部在 runtime 层，不动。
- **events.py 映射**：SoM 已有事件 = damaged / killed / enemy_killed（monsterNick 反查）；hp_low 从 snapshot.player.health 派生；navigation_arrived 从位置差派生（Kaetram 同法）。inventory_changed：**gap G3**——SoM 桥尚无背包快照，v0.1 先不做（loot 动作 verdict 即间接信号）。
- **policies.py**：不变。obs 组装函数换成 SoM snapshot 字段名（nick/position/health/alive/distance）。

## 3. SoM 侧小增量（本设计批准的改动面，均不碰 Network.gd 协议哈希）

1. **G1 session 管理**：bridge 加 `op=close_session{peerID}`（触发 Peers 断开该 peer）+ 空闲 TTL 记录（只暴露，reaper 在 Python 侧做，避免 GDScript 复杂度）。防 ghost——Kaetram 第三种 ghost 教训直迁。
2. **G2 score 事件**：enemy_killed 处理器已存在，桥侧按 peerID 累计 kills/deaths 进 snapshot 顶层 `score` 字段即可满足 v0.1；完整 Arena Controller（模板/expiry）推迟到 SoM M2。
3. **G3 背包**：挂账，不进 v0.1。
4. 红线保持：say 只走 LOCAL 频道（Discord 外发红线已实现，平移后在 policies 的 prompt 里同样声明）。

## 4. 探针/运维约定（直迁自 tick7 教训）

- 起 harness run 前：`pgrep -af godot471` + `ss -lun` 查端口；自建实例端口三元组（ENet/WS/bridge）必须与探针 `--port` 一致，否则静默连到 6119 旧实例（t7disc 实锤）。
- 探针 `--agent-keep` 保持在线；run 结束由 close_session 或探针退出收尾，三件套 meta 记 probe PID。
- 公平层/trace/golden：事件双出口断言（EVENTBUS-OK 模式）平移——SoM 侧 bridge `_push_event` 已是单收口，断言检查每类事件 drain 可见即可。

## 5. 验收判据（SoM harness v0.1 → 转 SoM M2）

1. RuleBot（seed 可复现）在 SoM 完整 rollout：预算 maxActions 耗尽即停，final_status=exhausted。
2. 三件套落盘齐（rollouts.jsonl/tools.jsonl/run.meta.json），configHash 含 soMBridge protocolVersion。
3. 事件唤醒有效：decide 数 ≪ poll 数（Kaetram 基线 ~35%）。
4. kill 事件 → enemy_killed → score.kills 递增，E2E 至少 1 次完整 hunt-kill-loot。
5. 0 SCRIPT ERROR（server log 判据沿用 T13）。

## 6. 排期（建议按 tick 粒度）

- tick9：client.py 适配层 + G1 close_session（桥侧 ~20 行）+ RuleBot 冒烟 20 动作。
- tick10：预算/唤醒/三件套全量验收（判据 1-5）+ soak（server 长 run）。
- 之后：SoM M2 Arena Controller（score 聚合升级 + 模板）。
