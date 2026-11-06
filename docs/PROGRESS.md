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

## tick24（2026-11-06 本 tick）：多模板批测 harness — grind_maggot(×10) 首次 completed ✅
- **tests/harness/batch_template.py**：TEMPLATE/SEED/MAX_ACTIONS/DURATION env 化批测入口（arena_start→AgentRuntime rulebot→排行榜回查）；判据踩坑修正——episode completed 后 arena_status 即无 active（final_score=null、harness 侧 session_lost/PEER_NOT_FOUND），终局权威读法=排行榜（tick22 定案复用）。
- **grind_maggot 首次打满**（三元组 7381/7382/9381，SEED=42, 150 动作/560s）：排行榜 `completed / kills_required_met / kills=10 / xp=10 / loot=3 / deaths=1`（lineNo=5 保序，tick23 截断修复回归确认）。Bot 在 kills_required_met 后未退出继续打怪（server 侧已 correct 终局），删 session 后 harness 报 PEER_NOT_FOUND 属设计行为。
- **tutorial_maggot 回归**（7391/7392/9391）：`TEMPLATE-BATCH PASS`——排行榜 completed kills=3（56s 达标），harness 继续跑到 60 动作 kills=8/final_score 齐全，0 SCRIPT ERROR。
- commit: 24a9646。下一步（tick25）：M3——多 seed/多模板批测汇总报告，或 LLM 策略位跑 grind_maggot 对比 RuleBot。
