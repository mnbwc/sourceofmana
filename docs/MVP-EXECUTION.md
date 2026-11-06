# Source of Mana · 最小 MVP 执行登记（MVP-1）

> 依据：《最小 MVP 范围与开工规格 v1.0》（2026-11-06 附件规格）。
> 本文件是规格在 SoM 仓的落地登记与实机核对记录。

## MVP-1 范围裁定（已落）

- `ARENA_MODE`：默认 `off`（**开关，非禁用**）。形态：project.godot 配置项，off 时 ArenaController/ArenaLeaderboard/AgentBridge 不注册（主世界零感知，AC-7 双向验收口径）。
- `EXTERNAL_MODE`：恒 `off`。
- 《游戏机制缺口表 v1.0》→「下一步策划输入」（§3.1/3.2/3.5/3.6 延后）。
- 开工包 v1.0/v1.1：不动、不落 v1.2，登记为**玩法模块规格说明书**。

## 玩法模块登记表（§6.1 实机回填，B7/B8/B9 已核）

| 项 | 实机状态 |
|---|---|
| 分支/HEAD | `feat/agent-arena`，MVP-1 时点 HEAD=`3dc507a`（commit hash 随演进更新于本表） |
| 目录 | `sources/network/AgentBridge.gd`、`ArenaController.gd`、`ArenaLeaderboard.gd`、`AgentClientProbe.gd`；harness 在 `tests/harness/` |
| 已实机落地组件 | AgentBridge（TCP NDJSON 五 op）、verdict 回执、事件闭环（damaged/died/kill/looted/arena）、ArenaController（模板 tutorial_maggot/grind_maggot、四终局、baseline 差值权威计分）、ArenaLeaderboard（JSONL+聚合）、trace logger、LLM 对局（rollouts.jsonl 轨迹）、多 agent 并发隔离 |
| 规格覆盖、未落地 | EventBus 三层断言、golden 单测、六项公平 CI、归因三件套、KAB、锁步/mapSeed/checksum/动作预算硬闸门 |
| 观战 | 模块内无已实现观战；MVP-9 的 SpectatorFeed 为统一只读入口，模块复用，不做两套 |
| 美术资产 | 模块全部自建代码，未复用上游美术（无 CC BY-SA 逐文件署名义务） |
| 代码纪律 | 零上游规则改写；上游改动仅 Launcher.gd:155 一行 + AIAgent.gd 掉落 roll debug 修复（D1，S38 系） |

## 汉化工作量实测（MVP-3/4 输入）

| 文本域 | 数量 | 位置 |
|---|---|---|
| UI（tscn 唯一 text 值） | 139 | presets/gui/*.tscn（50 场景）|
| UI（gd 赋值） | 21 | sources/gui/*.gd |
| NPC 对话字符串 | 678 | sources/scripts/tonori/**（46 文件，Mes/Say）|
| 任务文本 | 109 | presets/quests/*.tres（17 任务）|
| 实体/物品/技能名 | ~180 | presets/cells、entities `name=` 字段 |
| data/db/*.json | 21 条 | credits/news/agreement（agreement 保留英文原文）|

合计 ~1,100 条。方案 A（CSV translation + TranslationServer）。

## MVP-2 debug 清单现状（D1–D9）

| 项 | 状态 |
|---|---|
| D1 掉率 int 截断 | ✅ 已修（AIAgent.gd spawn roll，commit 6405216）|
| D2 排行榜截断 | ✅ 已修（852fde2）|
| D3 账号自踢竞态 | ✅ 已修（tick15 段）|
| D4 Sentry DSN 硬编码 | ⚠️ 确认存在：project.godot:536 + Monitoring.gd SentrySDK + 本机 sentry.dat → 本轮清空+出站快照 |
| D5 ghost/instance 残留 | ✅ 已修 |
| D6 kill 事件双写 | ✅ 已修 |
| D7 Server 静默 return | ⚠️ 本轮修（人类粗粒度被拒提示）|
| D8 GLOBAL→Discord 外发 | ⚠️ 确认 Server.gd:292；本轮显式禁用（自建实例未配 token，防御性关断）|
| D9 隔离 | ✅ 八项已过；端口/监听集合本轮补记《隔离证明》|

## 上游改动清单（R3 起始）

1. `sources/launcher/Launcher.gd:155` — 防第二实例自起服（1 行）
2. `sources/actor/agent/variants/AIAgent.gd` — 掉落按率预 roll（D1 debug 修复，非数值改动）
3. （MVP-2 追加中）DSN 清空 / Discord 禁用 / D7 提示 / isAgent 展示 / PartyService

**diff 数值零改动声明**：以上无任何数值/掉落表条目/词条规则/曲线改动（AC-5 口径）。

## 附：原 spec 附录 B 待证清账（11-06 实核）

B1 data/db 实为 3 文件 21 条，文本大头在 .tres + NPC 脚本；B2 全英文确认；B3 原版有 party UI 占位（"No party joined"）无 party 系统；B7/B8/B9 见登记表；B4/B5/B6 留 MVP-3/MVP-7 实测。
