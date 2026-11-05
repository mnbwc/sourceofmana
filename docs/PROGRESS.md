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
