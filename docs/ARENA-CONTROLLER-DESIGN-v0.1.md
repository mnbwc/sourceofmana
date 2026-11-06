# SoM M2 Arena Controller 设计 v0.1（Kaetram arena.ts 平移）

日期：2026-11-06（tick13 设计，本文件）。参照：`~/kaetram/packages/agent-api/src/arena.ts`（冻结仓，只读）。

## 1. 平移判定

Kaetram ArenaController 的五要素全部可在 SoM 桥侧等价实现，**零新增上游改动**：

| Kaetram 要素 | SoM 等价物 | 落点 |
|---|---|---|
| `DifficultyTemplate`（spawn/mob/killsRequired/maxDurationS/maxDeaths） | `_ARENA_TEMPLATES` 常量字典（GDScript Dictionary） | ArenaArena.gd（新增文件） |
| `player.killCallback` wrap | 桥已钩 `enemy_killed`（_on_enemy_killed，含 monsterID） | AgentBridge 现有钩子 |
| `player.deathICallback` wrap | 桥已钩 `agent_killed`（_on_agent_killed） | 同上 |
| `inventory.addCallback` wrap | 桥 loot op 直接计分（AddItem 无信号，tick12 定案） | 同上 |
| `player.teleport(spawn)` | `_server_peer().TriggerWarp`（与人类传送同源） | Server.gd 现有 RPC |
| xp snapshot（per skill） | `agent.stat.experience` + level（单技能制，更简单） | 桥 _snapshot 已导出 xp |
| tick() 过期巡检 | AgentBridge._process 已有帧循环，加 arena 巡检 | 同上 |
| emit（socket+trace 双写） | 桥 `_push_event`（drain_events 流）——SoM 无独立 events socket，drain_events 即 events stream | 同上 |

**结论：SoM 版更薄**——桥已有 kill/death/loot/xp 全部计分钩子（M5+G2v0.2 完成），Arena Controller 只剩「episode 生命周期 + 判定 + 模板」三件事，独立 autoload 文件，不碰上游。

## 2. 文件与数据结构

新增 `sources/network/ArenaController.gd`（autoload，与 AgentBridge 同目录同模式，`--server` 才激活）：

```gdscript
const TEMPLATES := {
    "tutorial_maggot": {
        "label": "新手：蛆虫×3",
        "spawn": Vector2(2230, 1250),      # Maggot 纯净区（world spawn id17）
        "targetNick": "Maggot",
        "killsRequired": 3,
        "maxDurationS": 300,
        "maxDeaths": 3,
        "brief": "击杀 3 只蛆虫。安全区，适合热身。"
    },
    "grind_maggot": {
        "label": "进阶：蛆虫×10",
        "spawn": Vector2(2230, 1250),
        "targetNick": "Maggot",
        "killsRequired": 10,
        "maxDurationS": 600,
        "maxDeaths": 2,
        "brief": "击杀 10 只蛆虫，注意血量与 respawn。"
    }
}

var _episodes : Dictionary = {}   # peerID -> episode dict
```

episode dict：`{id, peerID, nick, templateKey, score{kills,deaths,itemsLooted,xpGained,xpBaseline,levelBaseline}, startedAtMsec, status}`。

**计分来源=桥的 `_score` 字典**（不重复挂钩子）：ArenaController 每帧把 `_score[peerID]` 的增量镜像进 episode.score；xp 用 start 时 snapshot 差值（比 Kaetram 单技能简化）。

## 3. 桥 op 增量（AgentBridge.gd，仅此一处改动）

- `arena_start {template}` → 校验模板/无进行中 episode → TriggerWarp 到 spawn（公平层同源）→ 建 episode，返 `episodeId`。
- `arena_status` → 当前 episode + score。
- `arena_stop` → 显式结束，返权威终局分。
- 桥 `_push_event` 增加 `arena` kind（episode_start/score/episode_end）。

结束判定（ArenaController._process）：
- `kills >= killsRequired` → `completed (kills_required_met)`
- `deaths > maxDeaths` → `failed (max_deaths)`
- `elapsed >= maxDurationS` → `expired`
- `arena_stop` → `completed (explicit_stop)`

结束即推 `episode_end`（含终局分）；close_session 时若 episode 活跃按 explicit_stop 收尾（防 ghost 泄漏，同 Kaetram teardownSession）。

## 4. 验收判据（对齐 Kaetram golden 口径）

1. `arena_start` 后 agent 被 warp 到模板 spawn（snapshot.position 校验）。
2. kill 目标怪 N 只 → `episode_end status=completed endReason=kills_required_met`，score.kills=N。
3. 死亡超限 → `failed (max_deaths)`。
4. 到时 → `expired`，xpGained 诚实（快照差值）。
5. 非目标怪 kill 不计数（打 Scorpion 不涨 Maggot episode 的 kills）。
6. 双开拒绝（EPISODE_ACTIVE）；未知模板拒绝（UNKNOWN_TEMPLATE）——明确被拒回执（S16 纪律）。
7. 全程 0 SCRIPT ERROR。

## 5. 不做（v0.1 边界）

- 排行榜/多人匹配（M3+）；观战；damageDealt（SoM damageTable 结构不同，挂账）。
- itemsLooted 精确件数（沿用 items_changed 语义，与桥一致）。
