# SoM 系统缺口表 v0.1（T6 / M2 交付物）

> 逐项回答开工包 §11 M2 清单。全部为本地源码实读证据（文件+符号）。日期：2026-11-06。

| # | 问题 | 答案 | 证据 |
|---|---|---|---|
| 1 | 任务系统可查询/可 seed 化？ | quests = `Progress.quests: Dictionary[int questID → int state]`，结构简单可查询；seed 化：无随机任务逻辑，确定性 ✅。`progress.quests` 经 `RefreshProgress` 可推送 | sources/actor/Progress.gd:15-26 |
| 2 | 掉落表与 WorldDrop 结构 | `WorldInstance.drops: Dictionary[dropID → Drop]`；`Drop.item{cellID,cellCustomfield,count}`；`PickupDrop(dropID, agent)→bool`（距离 `PickupSquaredDistance` 校验+背包满通知）。**返回 bool，天然可当裁决回执** ✅ | sources/world/WorldDrop.gd:40-52 |
| 3 | ExportInventory 字段 | `Array[Dictionary]`（每 item `item.Export()`）；装备另 `ExportEquipment()→Dictionary{slotName→cellID+Custom}` | sources/actor/Inventory.gd:342-348, 362-371 |
| 4 | TriggerChat(LOCAL) 能否路由到 agent | `NotifyLocal` 走 `Network` 广播（agent 是同 World 内 peer，能收）；GLOBAL 已按隔离清单禁用 Discord 外发 | sources/network/server/Server.gd:288-294 |
| 5 | 多 agent 同图？ | 同一 World 多 peer（MaxPlayerCount=128），`WorldInstance.players` 支持 | sources/world/WorldInstance.gd:24 区域 |
| 6 | S21 地图/碰撞/导航数据来源 | **已解答：`presets/maps/navigations/*.tres`（9 张 NavigationPolygon 资源，独立于 tmx）+ `presets/maps/server/*.tres`（40 张 MapServerData：flags/spawns）**。headless 导出 exclude `data/maps/*` 只排原始 tmx，导航/服务器数据在 presets/ 下不受影响 ✅ 无缺口 | sources/world/WorldNavigation.gd:5-19; sources/db/Instantiate.gd:57-72; export_presets.cfg preset.6 |
| 7 | CommandManager 路径与命令集 | `sources/debug/CommandManager.gd`（63 行）；命令注册于 `sources/world/WorldCommands.gd`（43 条：spawn/warp/godmode/stat…）；权限 NONE<MODERATOR<GM<ADMIN；**release 构建强制权限检查**（`OS.is_debug_build()` 分支）→ agent 账号给 NONE 权限即可白名单化调试命令 | CommandManager.gd:16-40; WorldCommands.gd:5-20 |
| 8 | craft 语义 | 待下一步核 `Skill.gd`/crafting 脚本（低优先，竞技场不依赖） | — |
| 9 | progress.quests 阶段结构 | 见 #1，int state 可枚举 | Progress.gd |
| 10 | 事件真实命名（EventBus 源头） | 信号：`Entity.entity_died`、`BaseAgent.agent_killed`、`BaseAgent.enemy_killed(monsterID)`；出站推送：`RequestStatsUpdate / stat.buffs.NotifyAll / RefreshProgress / RefreshInventory`；击杀/拾取/死亡事件可从这些信号钩出 ✅ | sources/actor/entity/Entity.gd:17; sources/actor/agent/BaseAgent.gd:5-8 |

## 结论

- M2 十项中 8 项已答清，无架构级缺口；craft 一项低优先挂账。
- **S21 排除**：导航/服务器数据在 presets/ 下，headless 导出无缺数据风险。
- `PickupDrop→bool`、`CommandManager→bool` 证明上游多处已是返回 bool 的可裁决入口——裁决回执改造比预期更容易（Kaetram 全 void + 静默的悲观在 Mana 不成立，只有 Server.gd 的 Trigger* 系 void）。

## 下一步

T7/T8：AgentBridge.gd autoload 挂探针，冲 1 天四元组阀门（S17 已核清：独立 autoload 不影响 ProtocolVersion 哈希）。
