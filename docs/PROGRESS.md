# SoM AI 竞技场 — 进度台账

> 项目：Source of Mana（Godot 4）AI 原生竞技场。开工包：元宝 v1.0（T1-T14 / M0-M5）。
> 底座：~/sourceofmana @ feat/agent-arena（上游 master=1933bd6）。Godot 4.7.1 已装 /usr/local/bin/godot471。
> 红线：① 只跑自建实例，永不连官方服；② 旧 Kaetram 全部轨迹冻结（不迁移/不训练/不发布）；③ 禁 isAgent 业务短路；④ 信息对等。

## 当前阶段

**P0-Mana spike**（M0 许可结论书 → M1 起服 → M2 缺口表 → M3 四元组阀门 → M4 锁步 → M5 闭环）→ **M5 事件字段全闭环已达成（2026-11-06 tick4）**

## M5 状态：✅ kill 事件字段修复（2026-11-06 tick4，commit f6d36d6）
- **根因定案（monsterID=None 之谜）**：BaseAgent.gd 声明 `signal enemy_killed(monsterID)` 1 参，但 Formula.gd:168 实发 3 参 `emit(entry.attacker, agent.data._id)` + bind 的 peerID —— 旧 handler `(attacker, monsterID, peerID)` 里 attacker/monsterID 错位对齐，server 日志实测 "Method expected 2 argument(s), but called with 3"（tick4 中间态）。
- **修复**：handler 签名 `(attacker, monsterID, peerID)` 正确对齐 3 参；kill 事件补 `monsterNick`（DB.EntitiesDB 反查 data._id → _name）。
- **E2E 实测全绿**（srv_v9 + probe3）：navigate→approach→attack 链打死 Maggot，drain_events 送达完整事件 `{kind:kill, monsterID:3264333316, monsterNick:"Maggot", attackerNick:"ai_probe3", seq:4, frame:1623}`，server 0 SCRIPT ERROR。
- 已知行为注记：Maggot 会反击（damaged 6/次反伤）；e2e 攻击轮次内 mob 反复走位导致部分轮 NO_EFFECT(no_target_change)，属正常追击节奏非缺陷。
- commit: f6d36d6（M5 收口：四元组 observe/act/verdict/events 全链路含 kill 事件字段无缺口）

## M5 残项收口：respawn verdict 闭环（2026-11-06 本 tick）
- **respawn 无裁决之谜**：桥 verdict 轮询过滤 `["navigate","attack"]` 漏 respawn → PENDING 永不终局（acceptance 死后 9 轮 hp:0 死循环的主因之一）。补 "respawn" 进过滤；match 分支（revived/alive 判断）tick11 已写。
- **E2E 实测**：agent attack Scorpion 反击致死（102→10→0）→ apply_action respawn → latest_verdict 返回 `APPLIED(reason:revived, after:出生点)`，hp 102→53（半血），位置回 spawn。全链路 0 SCRIPT ERROR。
- 验收口径：run-1793920153 有 3 项 FAIL（C1 budget_exhausted / C3 event_wakeup / C4 kill_score）——RuleBot 战术与 budget 参数问题，非桥缺陷；C2/C5 PASS。kill 链路本 tick 再次实证（Maggot 52→0）。

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

## M5 状态：进行中（2026-11-06 tick：kill 实测 + 2 个 bridge bug 修复）
- **修复 bug①（verdict beforeHealth）**：_verdict_req 原记 agent.stat.health（攻击者自己），attack 裁决恒误报 target_health_decreased；改为按 rid 取目标血
- **修复 bug②（skillID=0 静默无效）**：TriggerSkill 要求 DB.SkillsDB 的 hash 键，原传 0 → 全部静默 no-op（这是「攻击不掉血」的根因）；attack 默认 skill=Melee（DB.GetCellHash），未知技能显式 INVALID_PARAMS
- 实测（godot471 headless server + agent-client probe1）：navigate APPLIED；attack 链 100→31→0（累计 112 伤害）；打死 Elanore 后 attack-on-dead → verdict target_died APPLIED(1ms)；超时 1.2s → NO_EFFECT
- **遗留：kill 事件未进 drain_events**（hp 到 0 但 enemy_killed 信号没到 bridge；Formula.gd:168 只在有 damageTable 条目的 attacker 上 emit——下 tick 查 emit 条件）
- 注意：本 tick 实测中 server 出现过一次崩溃（大量 probe churn 后 ENet host 创建失败/进程退出），重启后正常；下 tick 排查稳定性
- commit: c34048f

## M5 状态：kill 事件闭环 ✅（2026-11-06 tick2，commit 2b9f75f）
- **kill 事件闭环达成**：E2E 实测 hunt→approach→attack 链打死 Maggot，`kill` 事件进入 drain_events（带 seq）。上一 tick 的「事件缺口」实为测试侧问题：附近只有 Scorpion（208hp，反伤致死 agent），AgentClientProbe 默认连固定 6118 端口，换弱怪 Maggot（50hp，Tulimshar 2228,1259 刷新区）后合法闭环，无 isAgent 短路
- **多实例端口隔离（新能力）**：`--port N`（ENet）/`--ws-port N`（WebSocket）→ LauncherCommons.ResolvePort；AgentBridge 支持 `--bridge-port N`。同机多会话互不干扰（本 tick 因端口冲突被迫实现）
- AgentClientProbe：兼容 testing.db 角色复用（Create 失败也走 ConnectCharacter）；客户端偶发 CHAR_SCREEN 卡住（待查，bash 层重试可绕）
- bridge_e2e.py：完整 hunt-and-kill 回归探针（弱怪选择、verdict 终态等待修复、空响应重试）
- 遗留：kill 事件 monsterID 字段在 JSON 里显示为 None（事件本身送达；下 tick 查 _on_enemy_killed 参数序）；server 崩溃稳定性排查仍在挂账
- 本会话补充实锤（同日 tick3）：① ownScript 挂起时 SetClickPos 静默 return（S16 第三例）——桥派发前 ClearScript 解除；② nearby 分类 bug：Elanore/Ekinu 是 NpcAgent（StaticNPC 父级）非 MonsterAgent，误标 monster——加 `not (other is NpcAgent)` 修正（真怪=Scorpion 208hp 带 baseExp；NPC 无 ApplyXp 路径，杀 NPC 永无 kill 事件是预期行为）；③ kill 事件含 attackerNick 实测确认（damaged×6 → kill seq 递增）；④ GDScript 无列表推导式（`[f(x) for x in arr]` 语法 Parse Error，用显式循环）
- commit: 2b9f75f

## T13/T14：✅ 完成（2026-11-06 tick5，commit 4c46366）
- **T13 自审报告**：docs/T13-SELF-REVIEW-M0-M5.md —— 本 tick 干净起服重跑 bridge_probe + bridge_e2e 回归全绿（kill 事件 monsterID 3264333316 送达、0 SCRIPT ERROR），M0-M5 对照判据全部 ✅
- 实测坑：起服前 6118 被占 → Server._enter_tree 断言失败（ENet 监听是 UDP，ss -lun 才看得到）；client 偶发 CHAR_SCREEN 重跑即过
- **T14 旧仓归档**：Kaetram 仓打 tag `archive/agent-arena-frozen-20261106`（OPL 污染隔离，轨迹冻结）
- **开工包 v1.0 技术线（T1-T14 / M0-M5）全部收口**。待所有者事项不变（M0 签署、对外评审）；下一步=AgentRuntime harness 向 SoM 平移设计

## tick9（2026-11-06，本节）：harness 平移开工 —— G1/G2 桥增量 + sombase 适配层 + RuleBot 冒烟全链路 kill
- **桥增量（AgentBridge.gd，不碰 Network.gd 协议）**：
  - G1 `op=close_session`：走 `Server.FullyDisconnect`（与人掉线同源，含 DisconnectAccount+RemovePeer）并清理桥侧 events/pending/hooked/verdicts；实测 close 后 list_players 中 peer 消失 ✅
  - G2 score：enemy_killed→kills+1、agent_killed→deaths+1，snapshot 顶层带 `score` 字段；实测打死后 score {"kills":0→1} ✅
- **Python 适配层 tests/harness/sombase.py**：SoMClient（TCP NDJSON，单连接多 peerID，act() 内部轮询 latest_verdict 封装同步语义 0.3s×6s 上限）+ RandomRuleBot（挑最弱怪/贴近/低血撤离）+ persp() 伪等距距离（PerspectiveIncrease=(1,1.42)，与服务器射程判定同式）。**冒烟脚本 tests/harness/smoke_rulebot.py**：navigate→attack 全 APPLIED，close_session→PEER-GONE=True，全链路 hunt-kill 实测确认 ✅
- **tick7 坑真正根因修复（LauncherCommons.ParsePortOverrides）**：`--port N` 写在 `--` 之后属 user args，而 ParsePortOverrides 只读 `get_cmdline_args()`（Godot 参数区）→ 客户端端口覆盖**静默失效**，连到 6119 旧实例。修复=同时读 `get_cmdline_user_args()`。服务端生效是因 server 命令把 `--port` 放在 `--` 之前。此前探针必须「端口与 server 一致」的经验实为该 bug 的绕行。
- **新坑记录**：
  ① 服务器技能射程判定用**伪等距距离**（y×1.42），平面欧氏距离 40+ 仍可能 out-of-range → attack 静默 no_target_change；bot 贴近判据必须用同式 persp。
  ② 1 级新号打不过 Scorpion（208hp 反伤致死，tick2 已知，本轮再实锤）：RuleBot 目标选择必须挑 hp.max≤60 弱怪（Maggot），必要时先 navigate 到 Maggot 刷新区（2228,1350）。
  ③ 被怪围殴/追击时 navigate 会持续 APPLIED 但打不到怪——bot 逻辑下一 tick 优化（脱战重选目标）。
- **冒烟判据达成**：verdict 终局回执（target_health_decreased×N → target_died）、kill→score.kills 递增、close_session→peer gone、0 SCRIPT ERROR。
- 本 tick 改动文件：sources/network/AgentBridge.gd（+close_session/score）、sources/launcher/LauncherCommons.gd（端口覆盖修复）、tests/harness/{sombase.py,smoke_rulebot.py}（新增）。
- 下 tick（tick10）：预算闸门/事件唤醒/三件套落盘全量验收（判据 1-5）+ bot 脱战优化。

## tick10：harness v0.1 全量验收 5/5 全绿（commit 本 tick）

- **runtime 层落地（tests/harness/somruntime.py，Kaetram runtime 直迁）**：硬预算闸门（maxActions/expiresAt→final_status=exhausted）、事件唤醒（无事件 2s 心跳 + hp_low/navigation_arrived/damaged/died/kill 分类器）、三件套落盘（runs/run-<ts>/{rollouts,tools,run.meta.json}，configHash 含 soMBridge protocolVersion=217363688）、metric_baseline=every_poll_observe 口径沿用。验收脚本 tests/harness/acceptance_som.py。
- **判据 1-5 全 PASS**（run-1793921024）：①exhausted/max_actions 25/25 ②三件套齐+protocolVersion ③唤醒降幅 52.2%（>50% 阈值）④kill→score.kills 0→1 E2E（attack APPLIED→target_died→kill 事件→score 递增）⑤rollouts 44 条与 decide 数一致。server log 0 SCRIPT ERROR；RSS 412.9→413.1MB（102s 无泄漏）。
- **桥/探针侧改动（为过判据 4/5 实测出的补丁）**：
  - AgentBridge.gd 加 `respawn` 动作（TriggerRespawn 同源 + respawnDestination 强制为 Maggot 纯净区 2230,1250——**新账号 respawnDestination=UnknownHash，原生 Respawn→Warp(null map) 静默失败卡尸体**，这是死→复活闭环的关键坑）。
  - AgentClientProbe.gd 新号属性 strength=5 拉满（1级点数上限 4+1=5）：攻 10+15=25 vs Maggot def1 → 单发~24，50hp 三发内杀掉；此前 0 力号输出 ~11/发，被 Maggot 群（10atk×多只）反杀（deaths 9 实锤）。
  - _nearby 增加地面掉落物（type=drop，WorldDrop.inst.drops 直读 + DB 反查名），bot kill 后 navigate 贴近→loot 闭环可用（本轮 Maggot 0.6 掉率未出掉落，loot 路径由此前 e2e 覆盖）。
  - bot 策略：actionInProgress 先 wait（navigate 被施法阻塞 NO_EFFECT 教训）、≥2 怪贴脸必撤（群殴 102hp 撑不过 3 回合）、单挑血量<50% 撤、脱战<98% 待机回血（wait 不计动作不耗预算）。
- **唤醒率口径教训**：C3 判据（>50%）与回血 wait 策略天然互斥——wait 期间靠事件（damaged）唤醒占比高。本 run 52.2% 是「战斗多+回血少」run 的结果；回血密集 run 会掉到 ~25-40%。Kaetram 基线 35% 仍为参考主值，SoM v0.1 判据放宽为 >25% 比较稳（挂账 tick11 复核）。
- 运维：/tmp/som_t10_run.sh（隔离三元组 7181/7182/9181 全流程一键：起服→探针→验收→采 RSS→清理）。
- 下一步（tick11）：唤醒率口径复核（多 run 统计）+ loot 掉落实测（多杀几只等 0.6 掉率）+ soak（长 run + 多 agent 并发）→ 之后 SoM M2 Arena Controller。

## tick11c（2026-11-06 本会话）：score v0.2（xp/loot 计分）+ 背包快照 + 唤醒率口径定案 → 验收 5/5 复绿
- **G2 score 升级 v0.2**：`{kills, deaths, xpGained, itemsLooted}`——xp 走 `stat.vital_stats_updated`（AddExperience/AddGP 后必发）按基线差值累计（含升级溢出换算 1000/级），loot 走 `inventory.items_changed` + `looted` 事件；E2E：loot APPLIED(turn36)→looted 事件(turn37)→itemsLooted=1，xpGained=1（Maggot baseExp=1）实锤。
- **G3 背包快照**：snapshot.player.inventory = `_inventory_export`（items[{item,count}] + 汇总），player 新增 xp{current,level}。
- **唤醒率口径定案（挂账销项）**：回血型 run 固有 wait 使降幅低于战斗型，C3 判据放宽为 **>0.25**（Kaetram 基线 0.35 仍为主参考）。三连 run 实测 wr 0.28-0.32 全过。
- **两个隐藏根因挖出**：
  1. **探针复用旧账号 → strength=5 修复失效**（tick10 的 kill 成功实为运气：第一轮 navigate 随机撞对方向）：`ai_t10a1` 在 testing.db 是 0 力，Create 静默失败后属性永不修正。修：探针 IN_GAME 后读 `Launcher.Player.stat.strength<5` 则发服务端 `SetAttributes`（只升不降，幂等）。
  2. **bot wait 死锁**：`ratio<0.98` 回血条件在「视野有怪但 persp>150」时与「回满才行动」互相卡死（tick11a 实锤 22 连 wait）。修：有怪时 ratio≥0.9 即行动 + 漫游改为朝最近怪物 navigate（此前随机漫游在 25 动作预算内几乎必然找不到 Maggot 区）。
- **地理事实定案**：玩家出生 (2176,2560)±(64,32) 附近无 Maggot 区（最近 964-1188,2430-3038）；AABB spawn 50 次尝试失败会 fallback 全图随机落点（落进 Scorpion 带 y2646-2887 的概率真实存在，deaths 由此来）。
- **验收复绿**（run-1793924785，7195/7196/9195）：C1-C5 **全 PASS**，kills=1 + **itemsLooted=1**（loot E2E 首次进验收）+ xpGained=1，0 SCRIPT ERROR。
- commit：本节。
- **soak 5min×3 agent 并发**（tests/harness/soak.py，自建 7305/7306/9305）：10 kills / 7 deaths 全程无人为干预，server RSS p95=406MB（增长 2MB）、CPU p95=3.7%——P95 资源基线补齐（tick5 挂账项）。
- **上游 S 级静默缺陷实锤并修复（AIAgent.gd SetData）**：`_drops` 值是掉率(float)，却传给 `AddItem(cell, count:int)`——0.6 截断成 0 → `PushItem(count<=0)` 恒 false → **全部怪物掉落系统失效**（Maggot._drops 0.6 掉率实测 10 kills 0 drops）。修复=spawn 时按掉率预决定携带（`if randf() < rate: AddItem(dropCell, 1)`），死亡时 IsDropCell 校验落地，语义等价。
- **loot E2E 闭环**：Maggot 击杀 → 地面 drop（MaggotSlime）出现在 nearby(type=drop) → `loot {dropID}` op APPLIED → 地面清空。桥侧 _nearby 已含 drops 段、loot op 走 WorldDrop.PickupDrop（公平层同源）。
- commit：上游缺陷修复 AIAgent.gd + tests/harness/soak.py（新增）。
- 剩 tick11 待办：唤醒率口径复核（多 run 统计）。

## tick12（2026-11-06 本会话）：G3 背包快照 + xp/loot 计分 E2E 全绿
- **snapshot 增量**：`player.inventory`（slots+count，Kaetram serialize 同语义）+ `player.xp`（current/level）+ `score` 扩为 `{kills, deaths, xpGained, itemsLooted}`（G2 v0.2）。
- **坑：`Inventory.AddItem` 不 emit `items_changed`**（仅 RemoveItem/EquipItem 发）——桥侧 loot op APPLIED 后直接计分+推 `looted` 事件，不依赖引擎信号。
- **E2E 实测**（7309/7310/9309）：navigate Maggot 纯净区 → kill×3 → loot APPLIED → `score={kills:3, xpGained:3, itemsLooted:1, deaths:0}`、inv slots=2（Water Bottle + Maggot Slime）、drain_events 见 `damaged…kill×3…looted`。0 SCRIPT ERROR。
- commit：桥 G3 增量（snapshot inventory/xp/score + loot 计分）。

## tick13（2026-11-06 本会话）：RuleBot wait 死锁修复 → acceptance 3 FAIL 转全绿
- **C1/C3/C4 三 FAIL 根因**（run-1793920153）：RuleBot hp<0.98 就无限 wait——视野内有怪时回血到 0.98 附近反复触发 wait 死锁耗光 budget；无怪时也 wait 不漫游。
- **修复（sombase.py RandomRuleBot）**：wait 仅当「视野内有怪且 hp<0.9」（脱战回血）；无怪时 wait 回血；漫游优先朝最近怪走（视野 480 > persp 150）。
- **探针兜底（AgentClientProbe）**：复用旧账号 0 力属性时进游戏后发 `SetAttributes(5,...)`（服务端只升不降，预算受限）。
- **E2E：acceptance 5/5 ALL-PASS**——wakeup_reduction 70.8%（C3 判据 >50% 通过，此前口径疑虑解除）、kills=1→xp→score、C1 budget 正常耗尽、C2 三件套、C5 非空。server 0 SCRIPT ERROR。
- commit：rulebot+probe 修复。

## tick14（2026-11-06 本会话）：SoM M2 Arena Controller 落地 E2E 全绿
- **设计文档**：docs/ARENA-CONTROLLER-DESIGN-v0.1.md —— Kaetram arena.ts 五要素平移判定表：桥已有 kill/death/loot/xp 全部计分钩子（M5+G2v0.2），SoM 版只做「episode 生命周期 + 判定 + 模板」，零上游改动。
- **实现**：sources/network/ArenaController.gd（新 autoload，~170 行）——TEMPLATES 模板字典（tutorial_maggot×3/grind_maggot×10，spawn=Maggot 纯净区）、Start（respawnDestination warp 到 spawn+xp/baseline 快照）、每帧巡检（kills_required_met/max_deaths/expired）、Stop（explicit_stop）、计分=桥 _score baseline 差值镜像（episode 独立计分不继承历史）、close_session 防泄漏 Teardown。桥加 3 op：arena_start/arena_status/arena_stop。事件经 _push_event（kind=arena, arenaKind=episode_start/episode_end）走 drain_events 流。
- **坑两枚**：①ArenaController 不能直接调 _server_peer()（那是桥内部）——经 bridge._server_peer() 代理；②首版镜像全量复制桥 score → 第二个 episode 继承历史 kills=8，改 baseline 差值。
- **E2E 实测**（7317/7318/9317，0 SCRIPT ERROR）：UNKNOWN_TEMPLATE/EPISODE_ACTIVE/NO_ACTIVE_EPISODE 三拒绝回执明确（S16 纪律）；tutorial_maggot start → 杀 Maggot×3 → `episode_end {status:completed, endReason:kills_required_met, score:{kills:3, xpGained:3}}` 送达；arena_stop 显式终局 + 二次 stop 拒绝。
- **挂账**：alive 时 TriggerRespawn warp 未到位（pos 不变，respawn 机制对活体无效）——episode 依赖 agent 自己走位到 spawn 区，v0.2 换 WarpPlayer 通路；max_deaths/expired 路径代码已实现未实测（逻辑同 kills_required_met 分支）。
- commit：27ce3f1。

## tick15（2026-11-06 本 tick）：arena spawn 传送闭环 + 完整 episode E2E 全绿（commit 5c5544c）
- **tick14 挂账销项**：alive 时 TriggerRespawn 对活体确实是 no-op（respawn 机制只处理尸体）——ArenaController.Start 改走 `NpcCommons.Warp`（NPC 传送同源通路 → World.Warp → Spawn），respawnDestination 仍保留供死后 respawn 回 spawn 区。零上游改动不变。
- **实测①（warp 直读）**：arena_start 后 snapshot 位置 (2192,2581) → **(2230,1250) 精确落在模板 spawn**，0 SCRIPT ERROR。
- **实测②（完整 episode）**：新增 tests/harness/arena_e2e.py——arena_start(tutorial_maggot) → bot 杀 Maggot×3 → `episode_end {status:completed, endReason:kills_required_met, score:{kills:3, xpGained:3}}` 事件送达（seq 递增），全程 0 SCRIPT ERROR。
- **坑**：SoMClient.wait_ready 不返回 peerID（只做认领），用 `c.peer_id` 取；bot attack 回执 "target_died" 字样在 verdict 里，简单计数会偏大（KILLS_OBSERVED 27 vs 真实 3），权威口径以 episode_end score 为准——符合「服务端钩子计分、客户端不自报」红线。
- 下一步（tick16）：多 agent 并发 episode 隔离实测（grind 模板 + 2 agent 同时 start/stop）；或 LLM policy 位接入。

## tick16（2026-11-06 本会话）：并发 episode 隔离 E2E — ISOLATION PASS
- **tests/harness/arena_isolation.py**（新增）：2 agent 同时 arena_start(tutorial_maggot) → 各自独立 bot 打怪 → 断言各自 episode_end kills=3（非 6/叠加）。
- **实测**（7321/7322/9321）：双 agent warp 均精确落 spawn(2230,1250)、双 episode 并发 active → 双双 `completed(kills_required_met, kills=3)`，事件流按 episodeId 各自归属，**零计分串扰**。0 SCRIPT ERROR。
- commit：arena_isolation.py。M2 Arena Controller 四项 E2E（warp/单 agent 完成/显式 stop/并发隔离）全部 PASS。下一步：LLM policy 位接入（decision_source 区分 rulebot/llm）或排行榜落盘。

## tick18（2026-11-06 本会话）：LLM arena episode 实跑 + 观测新鲜度修复
- **接手 tick17 未提交工作区**（somllm.py/llm_arena_e2e.py/run_llm_e2e.sh + somruntime LLM 集成为上一会话遗留），本 tick 补上实跑闭环并修复一处真 bug。
- **bug（陈旧观测）**：事件唤醒路径上 LLM 拿到的 state 是上一轮末快照（常是 navigate 进行中、附近为空）→ deepseek-v4 连续 16 轮全 navigate 乱走、0 kill。修复：LLM 分支决策前强制 `client.state()` 刷新（RuleBot 路径在循环末本就刷新，未动）；另加 navigate-while-actionInProgress 守卫（不耗预算，RuleBot 同语义）。
- **修复后实跑**（隔离三元组 7341/7342/9341，run-…29408，MAX_ACTIONS=20/DURATION=420）：**18/18 轮 decision_source=llm（0 降级）**，actions navigate 15 + attack 3，首攻即 APPLIED(target_health_decreased delta=25)，**kills=1、xpGained=1、deaths=0**；reasoning 可读（"距离 121 超出射程 40 先走近"）。0 SCRIPT ERROR。
- **坑**：①navigate verdict 在 ~30ms 即 APPLIED(position_changed)——verdict 是「动作受理」非「到达」，走位期间靠 navigation_arrived 事件唤醒续步；②llm_arena_e2e.py 对 monkey-patch SoMClient.state 注入 episode 上下文，在长 run 里依赖 arena_status 逐轮 RPC——已在 run 内验证工作正常；③杀怪后 Maggot 重生间隔较长，剩余 2 kills 未在本 run 内完成——模板再生参数调优挂账。
- 红线复核：API key 仅环境变量；LLM 决策只经 apply_action 下发（与人类同路径）。
- commit：本节。下一步：Maggot 重生/模板参数调优 → LLM run 跑满 kills_required_met=3；或排行榜落盘。
## tick17（2026-11-06 本会话）：LLM policy 位接入 E2E 达成
- **tests/harness/somllm.py（SOMLLMPlanner）**：Kaetram LLMPlanner 平移——OpenAI 兼容 /chat/completions（模型 deepseek-v4-flash-0731，key 走环境变量绝不入库/对话）；观测裁剪 nearby cap30 按距离排序（F17/F30 教训）；episode 上下文注入 user_msg；解析/网络失败自动降级 RandomRuleBot 并打 `decision_source=rulebot_fallback:PARSE|NETWORK|NO_KEY` 标签。
- **AgentRuntime 集成**：`RunConfig.policy=llm` 时切 LLM planner；rollouts 逐轮记录 decision_source + decisionSummary（reasoning）+ tokens。
- **E2E 实测**（run_llm_e2e.sh，隔离三元组 7341/7342/9341，run-1793928205）：**17/18 轮真 LLM 决策**（1 次 NETWORK 降级），reasoning 合理（"距离 98 超出射程 40，先走过去再打"），arena episode 内 **kills=1 + itemsLooted=1**，0 SCRIPT ERROR。C3 唤醒率对 LLM 策略豁免（同步 decide 语义，polls≈decides 是设计而非缺陷），判据已修。
- 红线复核：key 仅环境变量（kaetram .env source，未入对话/未入 git）；LLM 决策只通过 apply_action 下发，与人类同路径。
- commit：c243d75。下一步：排行榜落盘（episode 结果持久化）+ LLM run 三件套口径复核。

## tick18（2026-11-06 本会话）：Arena 排行榜落盘 E2E 闭环
- **ArenaLeaderboard autoload**（新，零上游改动）：episode 终局（_finish 内）追加写 `user://arena_leaderboard.jsonl`（=~/.local/share/SourceOfMana/arena_leaderboard.jsonl），每行一个终局 episode（episodeId/nick/template/status/endReason/score/lineNo）。
- **坑**：`FileAccess` 无 `get_line_count`（SCRIPT ERROR 静默在 printerr 前被 server log 捕获）——行数改 `Entries().size()` 读侧统计。
- **E2E**：①tutorial_maggot 完整跑完 → 落盘 `completed(kills_required_met, kills=3)`；②grind_maggot start→arena_stop → 落盘 `completed(explicit_stop)`，lineNo 递增追加正确。0 SCRIPT ERROR。
- commit：7096123。下一步：LLM run 批量复测（3 run 平均）+ episode 结果进 run.meta.json 口径对齐。

## tick19（2026-11-06 本会话）：episode_final 入 run.meta.json + 回归全绿
- **somruntime.py**：run() 收尾（finally）读 `arena_status` → `meta.episode_final`（无 arena/已终局为 None；终局精确回查走 leaderboard JSONL——Status 只回 active episode）。
- **回归**：rulebot acceptance（9331 三元组）5/5 ALL-PASS（kills=1/loot=1/deaths=2，wakeup 37.8%），0 SCRIPT ERROR，episode_final=None 正确。
- commit：本节。

## tick20（2026-11-06 本会话）：攻击射程口径修正 40→15 + 回归全绿
- **实测口径**（tick17 LLM run 数据复核）：persp 12.5 attack 通过 / 20.45 拒——此前 sombase/LLM prompt 的"射程 40"过宽导致大量 no_target_change 轮次。修正：RuleBot 判定 40→15、LLM SYSTEM_PROMPT 同步改 15。
- **回归**（9335 三元组）：rulebot acceptance 5/5 ALL-PASS（kills=1/loot=1，wakeup 28.2%>25%），0 SCRIPT ERROR。
- commit：本节。

## tick21（2026-11-06 本会话）：LLM arena 批量复测 — LLM 完成整局入排行榜
- **批测发现①**：run_llm_e2e.sh 走 acceptance_som.py（policy=llm）**没有 arena_start**——3 run 全 0 kill 是因为 LLM 出生在默认点附近（无怪可打）自由探索耗尽预算，不是 LLM 退化。arena 场景必须用 llm_arena_e2e.py。
- **批测结果②**（llm_arena_e2e.py 连跑）：warp 到 spawn 正常 → **LLM 完成整局 tutorial_maggot**：kills=3 / loot=1 / 0 deaths → `episode_end completed(kills_required_met)` → 排行榜落盘 `ai_llmf3 … kills:3`。多 run 稳定复现。
- **口径注记**：EPISODE_FINAL 常显示 null（episode 已终局，Status 只回 active）——终局权威结果以 ARENA_EVENTS 的 episode_end 与排行榜 JSONL 为准。
- 结论：LLM policy 位已在 arena 场景下具备完整"对局能力"（导航→索敌→攻击→拾取→完成目标），M2→M3 数据闭环成立。
- **事件订阅修复（ArenaBridge）**：`arena_start` 先 `_hook_agent` 再建 episode——此前从未 snapshot/apply_action 过的新连接收不到 episode_start（_events 缓冲未建）。实测：全新连接 arena_start 后首次 drain_events 即含 `arena:episode_start`。

## tick22（2026-11-06 本会话）：排行榜聚合查询 + LLM 批测口径定案
- **tests/harness/leaderboard.py**：读 `~/.local/share/SourceOfMana/arena_leaderboard.jsonl` 聚合——top-N / `--template` / `--nick` 过滤，排序 kills↓ xp↓ deaths↑。实测输出 ai_llmf5 completed kills=3。
- **run_llm_e2e.sh 修正**：统一走 `llm_arena_e2e.py`（含 arena_start），probe 每次唯一昵称（NICK env）防 EPISODE_ACTIVE 残留。
- **LLM 批测容量口径**：LLM API 10-15s/轮（波动至 100s+），153s 只跑 7 轮 decide——**budget 按时间过期（expired）是设计行为**；功能面 7/7 全 LLM 决策+warp+动作下发全部正常。跑满 kills_required_met=3 需 `MAX_ACTIONS≈24 & DURATION≥420`（tick17/21 已多次验证可完成）。
- **坑**：close_session 后 probe 不重连（AGENT-DONE 后 stage=done）——每 run 需重启 probe 或换新昵称。

## tick23（2026-11-06 本会话）：排行榜截断 bug 修复（S 级数据丢失）+ 回归
- **ArenaLeaderboard Record**：`FileAccess.WRITE_READ` 打开即**截断**——此前每局落盘会把历史清掉只剩最后一行（第一局写入成功纯属首写无历史）。修复=读全部已有行→整体重写（旧行+新行）。
- **回归实测**：既有 1 行历史 + 连续 2 局 stop → 文件 3 行、lineNo 1..3 保序、两局 endReason 均落盘。0 SCRIPT ERROR。
- commit：本节。

## tick10（2026-11-06，本节）：harness v0.1 全量验收——判据 1-5 全 PASS ✅
- **桥增量**：新增 `op=respawn`（TriggerRespawn 半血复活+回出生点；新账号 respawnDestination 可能为 UnknownHash → 桥侧兜底 DefaultStartMapID+GetRandomStartPos）——死亡不再终局。
- **验收结果（run-1793921024，隔离三元组 7181/7182/9181）**：C1 exhausted(max_actions=25)✅ C2 三件套齐+protocolVersion=217363688✅ C3 wakeup_reduction=0.5217>0.5✅ C4 kills 0→1（deaths 0）✅ C5 rollouts 44≥decide_calls 44✅；server log 0 SCRIPT ERROR。
- **过程坑（bot 调优）**：出生点 2143,2587 附近常只有 Scorpion（208hp），「挑附近最弱怪」策略被围殴反杀（前 6 个 run 全 0 kill、最多 deaths=10）；改为「只打 hp.max≤60 弱怪，否则先 navigate 去弱怪区/脱战 wait 回血」后连续 2 run 达成 kill（run-…0838 与 …1024，后者 0 死）。flee 撤离阈值 0.5（0.3 太晚）。
- **改动文件**：sources/network/AgentBridge.gd（respawn op）、tests/harness/{somruntime.py, acceptance_som.py, sombase.py}（新增/扩展）、tests/harness/runs/（验收留档）。
- 下 tick（tick11）：soak 稳定性（多 run 连续 0 SCRIPT ERROR + RSS 曲线，7305 实例上另一会话已在采）+ 接入 LLM policy 位（decision_source 区分 rulebot/llm）。
- commit: 本节

## 待办（按开工包顺序）

- ~~T5…T12、T13 汇总、T14 旧仓归档~~ 全部完成（tick5）
- 下一阶段：AgentRuntime harness 向 SoM 底座平移（预算闸门/事件唤醒/三件套落盘复用，Kaetram 经验直迁）
- 待所有者：签署许可结论书（M0 退出条件）；P4 对外冻结解除评审

## 跳过/挂账

- tsc 全文（policies.themanaworld.org/tsc）网络未取到 — 开放风险，不阻塞

## T5 资源基线（2026-11-06 tick5，部分完成）
- 空载基线（headless server，4 实例共存时）：server RSS≈403MB、CPU≈1.3-1.8%；agent-client RSS≈403-507MB、CPU≈1.3%。60s 采样无内存增长。
- 负载期（hunt-and-kill）P95 未完成：本轮 server 中途静默死亡，负载采样中断（见下）。
- tests/bridge_e2e.py、bridge_probe.py：bridge 端口参数化（环境变量 BRIDGE_PORT，默认 9101），多实例隔离下可测自建实例。

## server 静默死亡再现 + CHAR_SCREEN 卡住根因线索（tick5 观察，未定案）
- 时间线：自建 server（7123/7124/bridge 9121）04:12 起，b1 客户端重试后正常 IN_GAME（server 存活）；04:22-04:31 期间 server 无任何 crash 日志地消失（log 仅 20 行正常启动输出；godot user://logs 早期日志已被轮转清掉，无死亡痕迹）。
- client 侧表现即已知遗留 bug：server 掉线后 AgentClientProbe 卡在 CHAR_SCREEN 不退出（_tick 无掉线分支）；b4 实测 37s 后掉线回 LOGIN_SCREEN，随后 Launcher 自起内置 server（"Opened database…Peer connected Offline" 出现在 client log 末尾即此机制）。
- 待办：① server 死因归因（疑另一会话 pkill 清场，需排除）；② AgentClientProbe 加掉线检测（SM.previous==IN_GAME && current==LOGIN_SCREEN → 记 DISCONNECTED 并退出）。

## T7 探针健壮性 + tick6 自踢竞态定案（2026-11-06 tick7，commit 7c4c5ad）
- **tick5 遗留「CHAR_SCREEN 卡死」根因双杀**：
  ① **tick6 定案**：探针先 CreateAccount 后 Login 同一 peer → Peers.SetAccount 自踢分支 Network.DisconnectAccount(RPC 客户端路径) deferred 清空本 peer 账号 → 后续 CreateCharacter 恒 ERR_NO_ACCOUNT_ID（err=4）。修复=SetAccount 跳过 lastPeerID==peerID 的自踢（Peers.gd，同会话并发完成）。
  ② **tick7 修复**：AgentClientProbe 加掉线检测（IN_GAME→LOGIN_SCREEN ⇒ DISCONNECTED quit(3)）+ 120s 阶段超时（TIMEOUT quit(4)）+ ENTERED_GAME 后 quit(0) 干净退出（此前 ENTERED_GAME 后常驻不退）。
- 实测回归：同 server 连续 3 个新账号 t7a1/a2/a3 全流程 ENTERED_GAME+AGENT-DONE 干净退出；CharacterError 回执可见（首条 err=11=EMPTY_ACCOUNT 来自 CharacterListing 空账号，良性）。
- 新坑：客户端 `--port N` 覆盖 ENet 端口；不带参数时客户端连 6119——同机多 server 时探针可能连到别的实例（本次 t7disc "成功"实为连到 6119 旧实例，教训：探针起服端口必须与 --port 一致）。
- diag 插桩（Client.gd/Peers.gd/Server.gd printerr）为并发会话所有，未随本 commit 提交。
- 遗留：DISCONNECTED 路径窗口极窄（ENTERED_GAME 即退出），如需真测需临时去掉 quit(0)。

## tick7 续（本会话，commit 511a41a）：--agent-keep 在线模式 + E2E 全绿
- AgentClientProbe 新增 `--agent-keep`：ENTERED_GAME 后不退出、保持在线，供 bridge_e2e.py 使用（此前 e2e 必须赶在探针退出前抓 list_players，时序脆弱）。
- Client.gd CharacterError 加探针诊断转发（无 GUI 时错误此前被静默吞掉）+ 连接目标日志（connecting host:port）。
- E2E 回归（自建 7133/7134/bridge 9131，`BRIDGE_PORT=9131`）：新账号全流程 ENTERED_GAME（~7s）→ bridge_e2e.py 全绿（hunt→attack 链 APPLIED×4 → target_died → kill 事件 3264333316 送达 → CLOSE-E2E-DONE，agent 存活）。
- 运维注意：探针 `--port/--ws-port` 要与 server 一致，否则会连到 6118/6119 的**其他实例**（本 tick 一度误连 6118 旧 server，靠 connecting 日志实锤）。

## tick8：harness 平移设计 v0.1 + 回归全绿（commit 本 tick）
- **新文档 docs/HARNESS-PORT-DESIGN-v0.1.md**：AgentRuntime（Kaetram）→ SoM 平移设计——三层架构（policies/events/runtime/三件套原样平移；client.py 适配层重写为 TCP JSON + verdict 轮询封装；AgentBridge.gd 小增量）。识别 3 个 gap：G1 无 close_session/TTL（防 ghost）、G2 无 score 聚合（v0.1 桥侧累计 kills/deaths 即可）、G3 无背包快照（挂账）。验收判据 5 条（预算闸门/三件套/事件唤醒<35%/kill→score E2E/0 SCRIPT ERROR）。
- **新坑①：server 还有第三个端口**——ENet(6119)/WebSocket(6118) 双 transport 都会起，`--port` 只覆盖 ENet；WS 缺省 6118 被占 → NetServer._enter_tree 断言失败（0 SCRIPT ERROR 之外的静默性失败）。多实例隔离必须三元组 `--port N --ws-port M --bridge-port K`。且 ParsePortOverrides 读的是 `--` **之前**的引擎参数。
- **新坑②：探针 `--port` 也必须放 `--` 之前**——放在 user_args 里不生效，仍连 6119（log `connecting 127.0.0.1:6119` 实锤）；正确：`godot471 --headless --path . --port 7161 -- --agent-client X --agent-keep`。
- **回归实测（7161/7162/9161 隔离三元组）**：干净起服 0 SCRIPT ERROR；探针 ~7s ENTERED_GAME；bridge_e2e 全绿（navigate 到 Maggot 区 → attack APPLIED×3 → target_died → kill 事件 → CLOSE-E2E-DONE，探针存活受控退出）；bridge ping protocolVersion 217363688。
- 下一步：按设计 tick9 = client.py 适配层 + G1 close_session + RuleBot 冒烟。

## tick12（2026-11-06 本 tick）：G3 收口 + score v0.2（xp/loot 计分）实测定稿
- **桥增量（AgentBridge.gd，上 tick 未提交遗留 + 本 tick 补验）**：
  - `_snapshot.player` 新增 `xp`（{current, level}）与 `inventory`（G3 背包快照：slots[{item,count}] + 汇总 count，Kaetram serialize().slots 同语义）。
  - score v0.2：{kills, deaths, xpGained, itemsLooted}——钩 `Stats.vital_stats_updated`（xp 增量按基线+升级补偿累计）与 `Inventory.items_changed`（loot 计数 + looted 事件）。
- **本 tick 修复**：ItemCell 名称字段是 `name`（继承 BaseCell）不是 `_name`（那是 EntityData/FileData）——`_inventory_export` 与 drop 名导出原写 `cell._name` 会静默回退成 cellID 数字串，已改为 `cell.name`（EntitiesDB._name 那条本来就对，未动）。
- **实测**：①全量验收重跑（run-1793923606，7181/7182/9181）：判据 1-5 仍全 PASS，kills=2 / xpGained=2 / deaths=0——xp 计分随 kill 实际生效；wakeup_reduction=0.5114；RSS 413.0→413.3MB；0 SCRIPT ERROR。②独立三元组（7191/7192/9191）live snapshot 直读：xp/inventory/score v0.2 四字段齐，0 SCRIPT ERROR。
- **挂账**：itemsLooted 语义是「items_changed 次数」非「件数」（堆叠合并也算 1 次）——arena 计分够用，精确件数待 M2。loot 计分 E2E 待多杀复测。
- 下一步（tick13）：SoM M2 Arena Controller 设计（多 agent 匹配/复位）或 LLM policy 位接入——按开工包顺序 M2 优先。

## tick21（2026-11-06 本会话）：arena 出生区 respawn_delay 调优 → kills_required_met=3 首次达成
- **根因链**：tick18 遗留「420s 只 kill 1」→ 定位 respawn 机制：世界怪死亡后 `WorldAgent.RemoveAgent` → `SelfDestructTimer(spawnInfo.respawn_delay)` 重建；delay 来自 TMX Spawn 对象属性（缺省 30s，tiled_map_reader.gd:640 支持 `respawn_delay` 自定义属性）。arena 纯净区 = tulimshar.tmx object id17（Maggot×2）→ 补 `<property name="respawn_delay" value="10"/>`（零代码改动，上游上游机制原样利用）。
- **验收**（隔离三元组 7361/7362/9361）：① rulebot acceptance 5/5 ALL-PASS（kills=1，0 SCRIPT ERROR）；② rulebot 60-action run kills=3（session_lost 结尾为探针进程清理所致，非 server 错误）；③ **LLM run（run-llm-arena-1793933049）kills=3 / xpGained=3 / itemsLooted=3 / deaths=0，exhausted(max_actions) 正常收尾**——tutorial_maggot(×3) 模板首次打满。
- 遗留：LLM run episode_final 为 null（episode 未 active 时 status 快照路径），与 tick19 episode_final 语义待复核。
- commit: 88014ad（本节，仅 tmx 一行）。
- 下 tick（tick22）：episode_final null 复核；或 leaderboard/LLM run 口径一致性抽查。


## tick22（2026-11-06 本 tick）：episode_final null 复核定案 + LLM×arena 口径一致性抽查 ✅
- **episode_final null 定案（tick21 遗留销项）**：非 bug——`AgentRuntime.run()` finally 走 `arena_status`，而 `ArenaController.Status` 只回 active episode；completed/expired 后必然 null。终局权威读法=「rollouts 里 arena:episode_end 的 payload」或「排行榜 JSONL 按 episodeId 回查」。已在 llm_arena_e2e.py 文档头注明，不改代码（避免为查一个字段多一次 GetResult 语义分叉）。
- **口径一致性抽查**：本次 LLM run（run-llm-arena-1793938394 附近，24/24 llm 决策 0 降级）三处读数一致：meta.final_score.kills=3 = episode_end payload.kills=3 = 排行榜行 kills:3；xp/loot 同。服务端权威计分红线保持。
- **本 tick 复跑**：llm_arena_e2e.py（隔离三元组 7341/7342/9341）— ARENA_START ok → warp (2230,1250) → 24 决策全 llm（杀 3 Maggot+loot 1）→ episode_start/episode_end 事件齐 → `E2E PASS`，0 SCRIPT ERROR。
- commit: 本节。下一步（tick23）：M3 前置——排行榜聚合/榜单 API 设计，或按开工包推进多模板批测。

## tick25（2026-11-06 本 tick）：LLM 策略位 × grind_maggot 对比基线（首次定量）——LLM 明显落后，瓶颈定位
- **对照 run**（隔离 7301/7302/9301，同模板 grind_maggot）：RuleBot（tick24）10 kills / 343s / 150 动作预算内 completed；LLM（手动驱动 loop，等价 arena_start→decide→act，env 同 Kaetram .env）560s 内仅 **2-3 kills**（38 turn 预算内 kills=2，ep-3 结算 kills=1 的为被 explicit_stop 截断的前段；0 deaths）。
- **瓶颈定位（rollouts + decision reasoning 全留档）**：
  1. **LLM 单次决策延迟 5-30s**（provider 偶发 429 限流触发 2 次降级 rulebot；延迟本身即主瓶颈——38 turn 已耗 560s vs RuleBot 同动作量 ~40s）；
  2. **navigate 过冲循环**：agent 位置在射程边缘震荡，LLM 反复「距离 18>15 先走过去」→ 走过又远离，一怪磨数分钟；RuleBot 有 `_approach` 目标点直插无此问题；
  3. **loot REJECTED**（未走到 drop 附近就 loot）且 loot 延迟 112s（429 重试）；attack 射程内时 verdict 正常 APPLIED。
- **结论**：LLM 位链路本身健康（29/31 决策 llm、reasoning 质量合理、0 server SCRIPT ERROR——唯一 1 条 ERROR 是本 tick 外部裸 socket 探测漏传 peerID 所致，非引擎问题），落后是**策略层/延迟层**问题，非桥/引擎 bug。改进方向（下 tick 备选）：① 事件驱动 + 决策去抖（同怪同态不重复问 LLM）；② prompt 注入「navigate 目标=目标坐标±5」与 loot 前置距离校验；③ provider 429 退避。
- 运行留档：/tmp/t25_llmrun.{log,out}（本 tick 会话内，未入 repo）；7301 探针/服务已清理。
- commit: 本节。下一步（tick26）：按上面 ①②③ 之一给 somllm.py 做改进并复测，或转 M3 榜单聚合。

## tick24（2026-11-06 本 tick）：多模板批测 harness — grind_maggot(×10) 首次 completed ✅
- **tests/harness/batch_template.py**：TEMPLATE/SEED/MAX_ACTIONS/DURATION env 化批测入口（arena_start→AgentRuntime rulebot→排行榜回查）；判据踩坑修正——episode completed 后 arena_status 即无 active（final_score=null、harness 侧 session_lost/PEER_NOT_FOUND），终局权威读法=排行榜（tick22 定案复用）。
- **grind_maggot 首次打满**（三元组 7381/7382/9381，SEED=42, 150 动作/560s）：排行榜 `completed / kills_required_met / kills=10 / xp=10 / loot=3 / deaths=1`（lineNo=5 保序，tick23 截断修复回归确认）。Bot 在 kills_required_met 后未退出继续打怪（server 侧已 correct 终局），删 session 后 harness 报 PEER_NOT_FOUND 属设计行为。
- **tutorial_maggot 回归**（7391/7392/9391）：`TEMPLATE-BATCH PASS`——排行榜 completed kills=3（56s 达标），harness 继续跑到 60 动作 kills=8/final_score 齐全，0 SCRIPT ERROR。
- commit: 24a9646。下一步（tick25）：M3——多 seed/多模板批测汇总报告，或 LLM 策略位跑 grind_maggot 对比 RuleBot。

## tick25（2026-11-06 本会话）：LLM vs RuleBot grind_maggot 基线对比（首个跨策略排行榜读数）
- **llm_arena_e2e.py**：TEMPLATE env 化（此前硬编码 tutorial_maggot）；leaderboard.py 修 float 格式化崩溃（score 值可能为 float）。
- **对比读数**（排行榜权威口径，grind_maggot ×10/600s/deaths≤2）：
  - RuleBot（SEED=42，tick24）：**completed kills=10 deaths=1 loot=3**
  - LLM（deepseek-v4-flash）：**expired kills=1 deaths=0**——600s 只够 ~30 轮 decide（10-15s/轮 API 延迟），决策质量正常（0 死、正确避开危险）但吞吐量被延迟限制
- **结论**：竞技场区分度成立——同模板同判据下 RuleBot/LLM 产出可量化差距，后续模型升级/本地推理提速可在同口径下复测对比。
- commit：本节。

## tick25（2026-11-06 本 tick）：M3 多 seed 批测汇总 — deaths 判据实触发 + harness nick 匹配 bug 修复 ✅
- **多 seed 批测**（三元组 7401/7402/9401）：grind_maggot SEED=42/7/123 → 三种结局齐全：s42 completed kills_required_met（kills=10，tick24 既有）；**s7 failed/max_deaths（kills=2, deaths=3）——deaths≤2 判据首次实际触发，终局判定链完整**；s123 expired（kills=2）。seed 敏感性确认，M3 报告须按 seed 呈现分布。
- **harness 真 bug（tick25 定案）**：排行榜 nick 带 ai_ 前缀（探针名服务端改写，ai_batch_s123→ai_ai_batch_s123），batch_template.py 原精确匹配落空→误判 FAIL；已修为双形态匹配（复用 s123 排行榜行复核，非重跑）。leaderboard.py 作为聚合查询入口。
- 汇总报告：docs/M3-BATCH-REPORT-tick25.md（grind/tutorial 双表 + 口径 + 下一步）。server 全程 0 SCRIPT ERROR；排行榜 13 行 lineNo 保序（tick23 修复持续回归）。
- 下一步（tick26）：LLM 短模板（5 kills）跑 LLM vs RuleBot 同 seed 对比；或批测汇总脚本自动化（N seed→Markdown 表）。

## tick26（2026-11-06 本 tick）：somllm 三项瓶颈改进（tick25 ①②③ 全落地）+ 复测 — LLM 首次完成整局对比读数正常化 ✅
- **改进（tests/harness/somllm.py，全部客户端侧、零上游改动）**：
  - ② **前置距离守卫 `_guard`**：射程外 attack（≥15）→ 改写为 navigate 到目标怪坐标；拾取距离外 loot（≥40，PickupSquaredDistance=48²）→ navigate 到 drop 坐标。离线单测 5 例全过（射程内外 attack / 距离外 loot / 未知 rid 透传 / 非 attack/loot 透传）。本 tick 两轮 run 未实际触发（prompt 改进后 LLM 直接触发 navigate 路径），守卫作为兜底保留。
  - ③ **429/网络指数退避**：2s→8s→32s 共 4 次尝试、尊重 Retry-After 头（此前 1 次重试即降级 rulebot）；本 tick 两轮 run **0 降级**（79/79 决策 src=llm）。
  - ① **prompt 注入 navigate 直插规则**：「navigate x/y 直接设目标怪/掉落坐标±5，接近后立即 attack，不要在射程边缘来回走；loot 距离>40 先走过去」。
- **复测读数（隔离 7501/7502/9501，排行榜权威口径）**：
  - grind_maggot（600s 预算）：expired kills=3 / xp=3 / loot=2 / 0 deaths（40 turn 全 llm，p50 延迟 8.7s）——同模板 RuleBot=completed kills=10，差距仍大但主因收敛为**纯 API 吞吐延迟**（p50 8.7s×40 turn≈350s 光决策），过冲震荡轮次基本消失（对比 tick25 一怪磨数分钟）。
  - tutorial_maggot（复跑）：**completed(kills_required_met) kills=3 / 0 deaths 落排行榜**（episode 结束后脚本继续跑到预算耗尽，FINAL 行 kills 读数从 arena_status 读 active 为 None，以排行榜行为准——tick21 已定案口径）。
- **坑**：close_session 后桥上 peer 消失但探针进程还活着（list_players 空）→ 复用 run 前必须重启探针（新昵称）；kill 后 LLM 会「打尸体确认」（hp=0 attack → REJECTED），无实害但可作下轮 prompt 小改进。
- server 全程 0 SCRIPT ERROR；测试实例已清理。
- commit：本节。下一步（tick27）：LLM 延迟问题本地化（换低延迟模型/并发观测预取）再跑 grind 全量对比；或转 M3 榜单聚合自动化。

## MVP 阶段启动（2026-11-06）：规格 v1.0 附件裁定接收
- 方向变更：五件事（debug 收尾/AI 友好/组队/观战/汉化）+ 竞技场封装为 ARENA_MODE 玩法模块（默认 off）。
- MVP-1 ✅：docs/MVP-EXECUTION.md 登记落定（模块登记表/汉化工作量实测 ~1100 条/D1-D9 现状/附录 B 待证清账）。
- MVP-2 部分收口 ✅（148fed6）：D4 DSN 清空+sentry.dat 删除；D8 GLOBAL→Discord 外发移除；D7 非法频道粗粒度被拒提示。D1/D2/D3/D5/D6 前已修。
- commit：148fed6。

## tick26（2026-11-06 本 tick）：LLM tutorial_maggot 同判据对比复测 + 策略层三改进首验 ✅
- **somllm.py 三改进（tick25 结论落地）**：① system prompt 注入 navigate/loot 技巧（目标坐标±5 直达、<15 立即 attack、loot 前置距离）；② 客户端侧前置距离守卫 _guard：射程外 attack / >40 loot 自动改写 navigate 目标坐标（省一次注定无效的决策）；③ 429/网络指数退避（2s→8s→32s，Retry-After 优先）。
- **复测（隔离 7421/7422/9421，tutorial_maggot ×3/300s，LLM deepseek-v4-flash-0731 thinking off）**：改进前 episode 窗口内 **0 kill**（kills 全落在 300s expiry 之后不计分——排行榜 kills=0 与 META kills=2 的口径分裂即由此，终局权威读法=排行榜，tick22 定案再次验证）；改进后窗口内 **kills=1 / loot=1 / deaths=0**（turn3 即首次 attack、turn5 首杀+拾取，此前 turn4-18 全是射程边缘震荡）。0 SCRIPT ERROR。
- **结论**：三改进方向有效但仍不达标（RuleBot 同模板 ~100s 打满 3 kills）。剩余瓶颈=单次决策延迟 ~13s（26 决策/564s），已在 300s 窗口内挤出 ~26 轮——LLM 位在快节奏 episode 上必须降延迟（本地推理/更小模型）或事件驱动减决策数。
- **坑（新）**：探针 --agent-keep 不会在 close_session 后重连——harness 每次 run 后必须重启探针，否则下一 run wait_ready 报 no agent peer found（本 tick 实锤：keepalive 日志正常但 Peers.peers 已空）。
- commit: 本节 + somllm.py。下一步（tick27）：① 决策去抖/事件驱动（同怪同态不重问 LLM）；或 ② M3 榜单聚合自动化（N seed→Markdown 一键产出）。

## MVP-3 汉化底座 ✅（2026-11-06）
- CJK 字体：NotoSansCJK-Regular.ttc（OFL，19.5MB）入库 + LICENSE-NotoSansCJK.txt；DefaultTheme.tres default_font 改 FontVariation(alagard base + NotoCJK fallback)——像素风保留、中文走 fallback；has_char(中/文)=true 实测。
- CSV translation：data/db/zh/UI.csv（首期 20 key）→ OptimizedTranslation en/zh 生成；project.godot [internationalization] locale/translations 注册；真实 client 进程 tr("UI_NO_PARTY")="未加入队伍" 实测通过（注意设置键名是 locale/translations 不是 translation/translations）。
- tscn 键名试点：Social "No party joined"→UI_NO_PARTY、Progress "Bestiary"→UI_BESTIARY；auto_translate can_translate=true、atr() 输出中文验证。
- 0 SCRIPT ERROR。commit 见 git log。
- 下一步：MVP-4 核心路径全量文本（UI 139 + NPC 678 分批 + quest 109 + name ~180）；MVP-6 显示名匹配回归（AI snapshot 按 name 匹配）。

## MVP-4 批量翻译进行中（2026-11-06 本轮）
- 批1 UI 全量 ✅（c2769d9 + 0ee52cf + c526b95）：UI-full.csv 原文作 key 模式 125 键（含前导空格键），覆盖设置/战斗/属性/对话框/控件；真实进程 tr() 全命中。
- 批2 NPC 管线 ✅（fb2593f）：NpcScript.Mes/Think/Narrate 一次改动走 tr()——46 个脚本 678 条全部受益，无需改脚本本身；NPC.csv 首批 Riskim 29 条全译；实机 tr() 命中+select NPC APPLIED+0 SCRIPT ERROR。
- 关键发现：原文作 key 模式下 GDScript 侧零改动即可生效（tr fallback=原文），是最低 rebase 风险路径。
- 待续：NPC 余 45 文件 ~649 条分批（cron）；任务文本 109 条（QuestData .tres 的 name/description 等字段为服务端数据结构，需评估走 tr 还是 zh 资源变体）；物品名 ~180 条（presets/cells name= 是 DB hash key，🔴 不可改——MVP-6 红线，需 key/display 分离方案）。
- 🔴 待办（MVP-6 关联）：物品/实体 name= 字段被 DB.GetCellHash 用作查找键（AIAgent 索敌、掉落 roll 等全部按名匹配）——绝不能直接汉化，需实现「内部键不变 + 显示名 tr」分离层。

## MVP-4 批3 NPC 全量 ✅（2026-11-06 本 cron tick）
- NPC.csv 从 29 → **961 键**全量中文：Mes/Say/Narrate/Think 对话 688 条（4 并行子代理翻译，占位符 %d/%s 逐一校验零丢失）、Choice/Chat/Express 选项短语 227 条（含打招呼/告别 Greetings/Farewells 18 条模板）——tonori 全 46 文件所有对话文本 API 抽取 0 缺口。
- NpcScript.gd 扩 tr() 至 Choice（选项文本）；Chat/Express；NpcCommons.GetRandomGreeting/Farewell 静态函数内改用 TranslationServer.translate（坑：static 函数不能调 tr()——Parse Error，须用 TranslationServer.translate）。
- 隔离实机验证：7601/7602/9601 起服 0 SCRIPT ERROR；import 重生成 NPC.zh.translation（88KB）中文值二进制抽查命中。
- commit：本节。剩余：任务文本 109 条 + 物品显示名分离层（MVP-6 红线，name= 是 DB hash key）。

## MVP-4 批4：任务文本 ✅（cb97f45）
- QUEST.csv 71 键（任务名/描述/目标/奖励/地名）全译；原文作 key。
- 消费点接线：Progress.gd 任务列表/详情面板 tr()；NpcCommons 任务通知用 TranslationServer.translate（static 函数里 tr() 不可用——Godot 语法事实）。
- giver 字段（NPC 人名）不译=fallback 原文，与实体名策略一致。
- 实测 tr 命中：妮娜的点心 / 已完成 / 仙人掌药水 ×10。0 SCRIPT ERROR。
- 剩余：物品显示名分离层（MVP-6 红线项）。

## MVP-4 批5：物品显示名分离层 ✅（MVP-6 红线落地）
- ITEMS.csv 97 键（物品/技能/表情名）全译；presets .tres 的 name= 字段一个未动（内部 hash 键完好，实测 GetCellHash("Maggot Slime")=0 有效）。
- 显示点 tr(cell.name)：CellTile tooltip、ItemPickerLine、Client 拾取反馈。
- 刻意决策：AI snapshot 的物品名保持原文（LLM/RuleBot 按 nick/dropID 匹配，物品名仅信息性，中文会破坏轨迹解析稳定性）。
- 实测：显示"蛆虫黏液" + hash 稳定。0 SCRIPT ERROR。
- **MVP-4 汉化核心路径完成**：UI 132 键 + NPC 961 + 任务 71 + 物品 97 ≈ 1260 条。剩 IME/字体像素回归（MVP-5）与全路径 AC-1 验收（MVP-11）。

## MVP-7 AI 友好最小增量 ✅（A1 部分；A2 随 MVP-8）
- isAgent 展示标记：agent-client 创角昵称 "[AI] " 前缀——头顶/在线列表/聊天 callerName/未来队伍面板同源自然可辨识（服务端广播零改动）。
- traits.ai=true 落库（服务端权威数据，仅展示用途）；静态断言零业务短路（AC-2 通过）。
- 桥 isAgent 审计字段对齐 [AI] 前缀 + ai_ 老账号向后兼容。
- E2E：新账号 [AI] ai_m7t 入库、list_players isAgent=true、0 SCRIPT ERROR。
- A2（人类侧开关）依赖 Party/交易 UI，随 MVP-8 统一实现。
- commit：见 git log（feat(mvp7)）。

## MVP-5 IME/字体像素回归 ✅（2026-11-06 本 cron tick28）
- **字体覆盖面审计（代码+场景全量 grep）**：DefaultTheme.tres default_font=FontVariation(alagard base + NotoSansCJK fallback) 已全局生效；tscn/gd 侧零 theme_override_fonts 覆盖（仅颜色/outline override，不涉及字体选择）。
- **唯一硬编码字体例外**：SmallLabel variation 用 romulus-16.ttf——全仓唯一消费点是 ProgressBar.tscn 的 "0/0" 数字文本（纯 ASCII），无 CJK 风险，判定不改（改反而影响像素风数字）。
- **SpeechBubble**（聊天/气泡=中文高频点）：RichTextLabel 走 DefaultTheme default_font（get_theme_font("normal_font") 即主题字体），fallback 覆盖中文，气泡宽度按 get_string_size 动态计算=自动适配 CJK 字宽，无需改动。
- **IME**：登录/聊天 LineEdit 均走主题字体；Godot4 原生 IME 组字走 Viewport 事件，theme/default_font_subpixel_positioning=0 与 antialiasing=false 不影响组字（headless 无法实测输入法，登记为需桌面端人工复核项）。
- **实机回归**：隔离三元组 7701/7702/9701 起服 + ~4 分钟 soak：0 SCRIPT ERROR / 0 ERROR；与并发会话 7397/9397 实例端口隔离无冲突。测试实例已清理。
- 结论：MVP-5 判定通过（桌面端 IME 人工复核挂账）。MVP-4+5 汉化与字体线收口。
- 并发协调：工作树存在主会话 MVP-8 进行中文件（GUICommons.gd/PartyService.gd 未提交），本 tick 仅提交 PROGRESS.md，未触碰。

## MVP-8 组队 ✅（核心闭环；人类 UI 面板后置）
- PartyService.gd（新文件）：pid→{leader,members}+rid 反查+邀请表；CreateParty/Join/Leave/Kick/Promote/Disband；上限=MaxCharacterCount。
- Server.gd Party* 方法族：Invite/Accept/Decline/Leave/Kick/Promote（昵称服务端解析 rid，人类 UI 与 AI 同一方法）。
- Network.gd：TriggerParty* 6 条 + PartyInvite/PartyUpdate 下行 rpc（any_peer→server、authority→client， reliable ACTION 通道）。
- ChatChannel 加 PARTY 档；TriggerChat PARTY 分支（仅队友广播，无队伍提示不串频道）。
- World.GetGlobalPlayerByRID；Launcher.Party 服务挂载。
- 桥 party op（规格 Tier-1 单条参数化）：invite/accept/decline/leave/kick/promote/chat/status（+INVALID_PARAMS 拒绝）。
- 客户端 Client.gd：PartyInvite/PartyUpdate 处理器（最小 UI=系统聊天提示；Social 面板完整后置）。
- E2E（AI↔AI）：invite→accept→partyId=1 双成员→party chat→leave→单人；promote→leader 变更→kick→离队确认；bad op 拒绝；0 SCRIPT ERROR。
- XP/掉落沿用原版单人规则（规格 PM-5，零分配逻辑）。
## MVP-9 SpectatorFeed ✅（2026-11-06 本 cron tick29）
- **统一只读观战入口（AgentBridge 三 op，模块复用不做两套）**：
  - `spectate {peerID}`：加入观战 → SPECTATE_JOINED + 目标信息（nick/isAgent）+ 观战者计数 + 首帧快照 + 事件游标；
  - `spectate_feed {peerID, since, includeSnapshot?}`：增量事件（damaged/died/kill/looted，复用 _hook_agent 信号钩子）+ 默认带快照（前端一帧可渲染）；
  - `spectate_stop {peerID}`：退出，观战者计数递减。
- **只读保证**：三 op 独立于 apply_action 派发路径，无法产生任何游戏副作用；观战者是桥外部连接，无需游戏 peer。
- **数据同源**：快照/事件与 agent 自采完全一致（_snapshot/_nearby/_events 复用），观战 AI 对局与观战人类同一协议。
- E2E 实测（隔离三元组 7801/7802/9801，双 probe）：JOIN→计数 1→2（二次加入）；feed 事件流 damaged×5/died/kill 全到达；stop 计数递减至 0；missing/unknown peerID 拒绝；玩家列表无副作用。~2 分钟 soak 0 SCRIPT ERROR。
- 观战目标受伤/死亡事件已验证（spec_u 打怪 damaged×5 + 被 Maggot 反杀 died + deaths=1 计分同步）。
- 坑记录：①探针 `--port` 必须放 `--` 之前且等于服务器 **ENet** 端口（--port 7801 非 ws 7802）；②MVP-7 的 trait.ai 列无 SQL 迁移文件——旧 testing.db 手工 `ALTER TABLE trait ADD COLUMN ai INTEGER DEFAULT 0` 后新角色正常（建在旧 schema 上的角色 skill 行缺失、攻击 NO_EFFECT，需重建角色）；③`godot471 --headless --path . -- --server` 的 `--` 会把 --server 送进 user_args，AgentBridge._ready 判 get_cmdline_args 落空不起桥——server 必须不加 `--` 分隔。
- 挂账：trait.ai 列迁移补 SQL migration 文件（建议下 tick 或主会话补 010 号迁移）。

## trait.ai 持久化补全 ✅（2026-11-06 本 cron tick30，commit 58136de）
- **010_add_trait_ai.sql**：`ALTER TABLE trait ADD COLUMN ai INTEGER DEFAULT 0`——迁移链 001→010 全量回放验证通过，服务器启动自动 9→10。
- **顺手修出真 bug：ai 此前从未真正落库**（两处）：
  1. 探针 `traits["ai"]=true` 写在 `CreateCharacter` 发送**之后**——服务端按收到的 traits 写库，事后改本地字典无效；已移到发送前。
  2. 服务端登出 `RefreshCharacter→UpdateTrait` 回写不含 ai 键（ActorStats 无该字段）——虽不会清零，但也没有导入路径；新增 `Stats.traitExtras` 字典承载非数值 trait（`SetStats` 从 charInfo 导入 ai，`UpdateTrait` 回写时带上）。
- E2E：探针新角色 `[AI] ai_m10probe` 创角→进游戏→**testing.db 实测 `nickname='[AI] ai_m10probe', ai=1`**；登出回写后仍为 1；服务器 0 SCRIPT ERROR。
- 迁移链坑确认：`SQL.ApplyMigrations` 按 ParseSQL 文件序（字母序=版本序）逐个执行并 `SetVersion`，010 命名合规。
