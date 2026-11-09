# SoM Go 服务端迁移 — 自动驱动状态（方案A）

> 本文件由自动驱动循环（每15分钟 cron → `paw send` 注入本会话）读写。
> 上次驱动时间见最下方"驱动记录"。

## 当前阶段
- 阶段0：协议可行性验证 ✅（2026-11-09，26个 Go 测试全过，Godot e2e 客户端握手+RPC 回包成功）
- 阶段1：登录链（CreateAccount→Login→CreateCharacter→IN_GAME）✅ **完成（2026-11-09）**
- 阶段2：AOI/移动状态同步 ✅ **完成（2026-11-09）**：服务端 + 双 Godot 客户端 e2e 连续 8/8 PASS
- 阶段3：战斗/技能/任务全量迁移 ✅ **完成（2026-11-09）**
  - ✅ 聊天/表情/施法广播（gameplay.go）
  - ✅ 持久化（store.go，JSON 落盘 + 原子写 + 重启恢复）
  - ✅ NPC/任务（npc.go）
  - ✅ 组队/交易（party.go + trade.go）
  - ✅ 战斗结算（combat.go，伤害/血量/死亡/重生）
  - ✅ 角色存档扩展（Level/经验/装备/背包）
- 阶段4：物品/经济/成长系统 ✅ **完成（2026-11-09）**
  - ✅ 物品目录 + 背包操作（UseItem/EquipItem/UnequipItem/DropItem/PickupDrop）
  - ✅ 属性分配（SetAttributes → UpdateAttributes）
  - ✅ 商人交易（MerchantTrade，GP 余额校验 + 堆叠）
- 阶段5：等级/经验/Buff/镜头/坐/密码管理 ✅ **完成（2026-11-09）**
  - ✅ 135 级经验表（0 基）+ 升级循环 + 击杀经验（combat.go grantKillExperience）
  - ✅ 镜头控制（CameraLookAt/CameraReset）+ 坐站（TriggerSit）
  - ✅ 视口尺寸（SetViewportSize）+ 在线列表（RequestOnlineList）
  - ✅ 密码管理（ChangePassword/RequestPasswordReset/ConfirmPasswordReset）
- 阶段6：Buff/私有属性/掉落/通知/投掷物/变身/移除实体/背包刷新 ✅ **完成（2026-11-09）**
  - ✅ Buff 系统（applyBuff/clearBuff/过期清理 + EntityBuff AOI 广播 + UpdateBuff 自身推送）
  - ✅ 技能 Buff 效果（Inma 防御-5/10s + SonicWave 移速-20/5s）
  - ✅ 地面掉落（spawnDrop/removeDrop/expireDrops + NPC 死亡掉落）
  - ✅ 私有属性同步（UpdatePrivateStats：经验/GP/法力/耐力/karma/负重/形态/精神）
  - ✅ 移除实体（RemoveEntity 广播）
  - ✅ 通知/表情/投掷物/变身/上下文/导航/教程 UI（13 个 handler）
  - ✅ nameID 全表修正（129→119 @rpc 方法，修复 15 个错误常量）
- 阶段8：SoM 客户端尚未接入的系统 ✅ **完成（2026-11-09）**
  - ✅ NPC 世界实体（spawnNpcEntities → 负 CharID 加入世界，可被技能目标化）
  - ✅ 图鉴追踪（trackKill → AddBestiary → UpdateBestiary 推送）
  - ✅ 技能进度（grantSkill/removeSkill → AddSkill/RemoveSkill → UpdateSkill 推送）
  - ✅ 属性存储（SetAttributes 上限校验 4+1×level → Character.Attributes）
  - ✅ GM 命令集（42 个 WorldCommands 全部实现）
  - ✅ 击杀/复活/传送/生成 + 私信/查询 + 屏蔽/禁言 + 检索角色信息

## 阶段1 交付物（2026-11-09 完成）
- `login.go`（~490 行）：AccountStore 内存账户库、Session 状态机（anonymous→authed→in_game）、
  全部登录链 handler：CreateAccount / LoginWithPassword / LoginWithToken / CharacterListing /
  CreateCharacter / ConnectCharacter / DeleteCharacter / DisconnectAccount / DisconnectCharacter /
  RetrieveCharacterInformation；AuthError/CharacterError 枚举与约束常量与 SoM 一致。
- nameID 映射：Network.gd 118 个 @rpc 方法按名排序（0 基）：AddOnlinePlayer=0, AuthError=1,
  AuthTokenResult=2, CharacterError=8, CharacterInfo=9, CharacterListing=10, ConnectCharacter=19,
  CreateAccount=25, CreateCharacter=26, …
- `login_test.go`：环回集成测试 TestLoginChainLoopback（假客户端走真实 wire 协议，7 步全过）。
- `testdata/e2e_client.gd`：真 Godot 4.7.1 无头客户端，镜像 Network.gd 全部 118 个 @rpc 桩
  （保证 nameID 对齐），5 步登录链状态机，**E2E PASS（31 帧，连续两次可重复）**。
- `run_e2e.sh`：一键 e2e（构建+起服+无头客户端+PASS/FAIL 判定）。
- 审计：`gofmt` 干净、`go vet` 通过、`go test ./...` 全过（unit + loopback）。

## 阶段2 交付物（2026-11-09 完成）
- `movement.go`（~300 行）：服务端权威轻量世界（Entity：charID=agentRID、Pos/Vel/Orient/Target/State），
  enterWorld（WarpPlayer→FullUpdateEntity→PreloadPlayer 双向交换→UpdateEntity→RefreshOnlineList→
  AddOnlinePlayer）、leaveWorld（RemoveOnlinePlayer）、SetMovePos/SetClickPos/SetOrientation、
  10Hz worldTick + AOI（半径 1000px）UpdateEntity 广播。
- `TestWorldSyncLoopback`：双假客户端验证 enter 推送顺序、Preload 交换、AddOnlinePlayer、移动广播。
- **双 Godot 客户端 e2e**：`run_e2e.sh` 并发启动 peerA/peerB 两个真 Godot 4.7.1 进程
  （E2E_ROLE 环境变量区分），验证互见（PreloadPlayer）+ 在线列表 + 移动同步（UpdateEntity）。

## 阶段3 交付物（2026-11-09，进行中）
- `gameplay.go`（~150 行）：聊天/表情/施法的通知层。
  - `notifyNeighbours`（AOI，可选 inclusive）/ `notifyGlobal`（全服）。
  - TriggerChat：LOCAL→AOI 含自身回声；GLOBAL→全服；PARTY→系统提示（组队未实装）。
  - TriggerEmote：Emote 广播 AOI（含自身）。
  - Casted：Casted 广播 AOI（含自身；伤害结算未实装）。
  - ChatQuery：占位（无待发系统消息）。
- `store.go`（~200 行）：持久化。
  - `loadStore()` 启动时从 `SOM_STORE_FILE`（默认 `som_store.json`）恢复；缺文件=空库。
  - `saveLocked()` 原子写（temp + rename），账户按名排序保证确定性。
  - Add/AddCharacter/RemoveCharacter/SetToken 变更后自动落盘。
- 测试：`TestGameplayLoopback`、`TestStorePersistenceRoundTrip`、
  `TestStorePersistenceRemove`、`TestStorePersistenceMissingFile`、
  `TestPersistenceSurvivesRestart`（双实例共享 store 文件）。
- e2e 客户端修复：移动检测改为「与 PreloadPlayer 首次观察的 x 比较」（单次 UpdateEntity 即可判定），
  聊天/表情/施法按**对方昵称/agentRID**过滤（不再把自身 LOCAL 回声误判为对方消息）。
  `run_e2e.sh` SOLO+PEERS 连续多轮 PASS。

## 关键经验（后续阶段直接复用）
1. **Godot 二进制限制**：`/tmp/godot-release` 编译时 `disable_path_overrides=yes` → 不支持
   `--path`，也不支持 CWD 加载；必须把二进制**放进项目目录**（goserver/godot-run）并从该目录启动。
2. **MultiplayerAPI 没有 rpc_id**：只有 `mp.rpc(peer, object, method, args)`；
   `Node.rpc_id(peer, method, ...)` 才是另一条路。用错会**静默不发包**（无报错）。
3. **reply 顺序**：CreateAccount(rememberMe)/CreateCharacter/CharacterListing 均先发
   AuthTokenResult 或 CharacterInfo，再发 AuthError/CharacterError —— 测试必须按序收全。
4. **无头运行 stdout 块缓冲**：被 SIGTERM 杀掉时丢失输出；调试用 `--verbose` 强制刷出。
5. **有状态服务重跑**：e2e 客户端用 `randi()` 生成唯一账户/角色名（**不要用**
   `Time.get_ticks_msec()`——它是进程内时间，每次运行几乎相同），可对运行中的服务器重复执行。
6. RPC 节点 checksum 不匹配只是警告，Godot 继续工作（Go 侧占位 MD5 即可）。
7. **nameID 提取必须容忍空行**：Network.gd 中 `Emote` 的 @rpc 与 func 之间有空行，早期提取
   漏掉导致 >索引35 的 nameID 全部偏移 1（已修正，见 /tmp/rpc_ids_119.txt 全表）。
   正确值：FullUpdateEntity=39, LoginWithPassword=45, LoginWithToken=46, PreloadEntity=54,
   PreloadPlayer=55, RefreshOnlineList=59, RemoveOnlinePlayer=62, RetrieveCharacterInformation=66,
   SetClickPos=75, SetMovePos=76, SetOrientation=77, UpdateEntity=112, WarpPlayer=118。
8. **enter-world 推送先于 CharacterError 回包**：ConnectCharacter 处理时 enterWorld 先发
   WarpPlayer/FullUpdateEntity/PreloadPlayer/RefreshOnlineList，再发 CharacterError。
   测试等待回包时必须跳过（serverPushNameIDs）。
9. **@rpc 第 4 个参数（config）被 Godot 忽略**：nameID 只按方法名字典序分配
   （scene_rpc_interface.cpp `_parse_rpc_config`：id = 排序索引，config 值不参与）。
   因此**无法在单节点上模拟多客户端**（同一节点只有一套方法集）——多客户端 e2e
   必须用**多个 Godot 进程**（本阶段采用，也更贴近真实集成场景）。
10. **多客户端 e2e 的稳定性要点**：① 双方都要持续发 SetMovePos（服务器只在 Vel≠0 时广播）；
    ② 先通过检查的客户端要**赖活几百帧**再退出，否则服务器 leaveWorld 会把实体摘掉，
    导致后进入的客户端观察不到；③ 移动检测用「与首次观察的 x 比较」而非相邻两次比较
    （对方可能只广播一次就退出）。
11. **Godot 进程内 `_process` 帧率**：无渲染 headless 下帧率不稳，登录链 5 步可能耗时
    ~400 帧；超时窗口要给足（本阶段用 1200 帧）。
12. **e2e 断言必须过滤自身回声**：LOCAL 聊天/Emote/Casted 都是 inclusive（含发送者），
    客户端若不过滤就会把自己的消息当成对方消息而**假阳性 PASS**。聊天按 nick、
    表情/施法按 agentRID 过滤。
13. **测试隔离**：`NewServer()` 现在会 `loadStore()` 读磁盘；测试必须覆盖为
    `NewAccountStore()`（内存库），否则测试间通过 som_store.json 泄漏状态。
14. **pkill 自匹配陷阱**：`pkill -9 -f goserver` 会匹配到**自身 shell 的命令行**
    （命令里含 "goserver"），导致脚本被杀。用 `pkill -9 -x goserver`（精确进程名）。
15. **PreloadEntity ≠ UpdatePublicStats**：两者 nameID 不同、参数不同。
    PreloadEntity=54 只有 5 参（agentRID/actorType/shape/nick/state）；
    UpdatePublicStats=114 有 10 参（agentRID/level/health/maxHealth/hairstyle/
    haircolor/gender/race/skintone/shape），**没有 nick**。服务端 TriggerSelect 必须发
    114 的 10 参载荷，否则 Godot 解码 10 参会错位。NPC 因此需要单独的 stats 字段。
16. **enterWorld 推送序列会随功能增长**：阶段3 追加了 PreloadEntity(NPC)×2 +
    RefreshProgress，enter-push 顺序为 WarpPlayer→FullUpdateEntity→PreloadPlayer→
    UpdateEntity→PreloadEntity×2→RefreshProgress→RefreshOnlineList。
    TestWorldSyncLoopback 的 wantOrder 必须同步更新（否则 want 5 got 8）。
17. **PartyUpdate.members 是 JSON 字符串**：客户端 `JSON.parse_string(members)`
    解析，不是 PackedStringArray。服务端必须 `json.Marshal([]int64)` 后作为
    String 发送（leaderRID 才是 int）。别和 ChatPlayer 的数组参数搞混。
18. **PartyInvite/PartyUpdate 是 authority→client 推送**（nameID 51/52），
    TriggerParty* 是 any_peer→server（98-103）。PartyResolve 用 **nick** 查在线玩家
    （world.byName），不是 RID。队长转让/解散语义见 PartyService.gd。
19. **交易请求走 `/trade` 命令，不是独立 RPC**：客户端 `TriggerCommand("trade \"Nick\"")`
    → Server.TriggerCommand → CommandManager → WorldCommands.CommandTrade →
    WorldTrade.Request。服务端 RPC 面只有 TradeRespond/SetOffer/Lock/Cancel
    （87/88/84/82）+ TradeRequested/Opened/Update/Closed（86/85/89/83）。
    TradeUpdate 的 ownOffer/otherOffer 是 **Array of [itemID,count,customfield]**。
20. **按昵称发起的社会行为必须等对方可见**：partyA/tradeA 发起前要等
    `peer_nick in seen_preloads`，否则服务器 `world.byName` 查不到（对方还在登录链），
    静默返回「Player not online」。这是 e2e 并发下的常见竞态。
21. **交易锁定语义**：TradeUpdate 参数顺序是 (ownOffer, ownGP, otherOffer, otherGP,
    ownLocked, otherLocked, revision, peerID)。**对方锁了对自己就是 otherLocked=true**，
    容易写反。双方都 locked 才 commit → TradeClosed("Trade completed.")。
22. **gorilla/websocket 反复 SetReadDeadline 会损坏流**：在同一连接上多次调用
    SetReadDeadline（每次读消息前都设）会导致后续 ReadMessage 返回
    "Invalid packet received. Unable to decode state variable" 或流损坏。
    e2e 测试要为每个客户端起**专用 pushCollector goroutine**（只设一次 30s deadline），
    把推送排进 buffered channel，测试逻辑从 channel 里按 nameID 过滤。
23. **e2e 测试的技能选择**：SoM 技能 Melee(229218829) 伤害=attack(10)+skillValue(10)-defense(10)=10，
    杀 100 血要 10 次；Flar(2089110122) Attack=50 → 伤害 50，2-3 次击杀。
    e2e 测试用 Flar 减少施法次数（缩短测试时间，降低超时风险）。
    技能射程 IsNear 用 skillRange + agent._radius + target._radius（半径各 ~16px），
    Flar Range=120 → 实际 ~152px 内可命中。
24. **e2e 客户端位置竞态**：UpdateEntity 可能先于 PreloadPlayer 到达（服务端 enterWorld
    先发 PreloadPlayer 再发 UpdateEntity，但客户端 poll 顺序不保证），导致
    `if agentRID == peer_rid` 判断时 peer_rid 还是 0，位置被丢弃 → peer_pos 停留在
    (0,0)，combatA 走向原点。修复：客户端缓存 last_pos_by_rid[agentRID] = position，
    在 PreloadPlayer 设置 peer_rid 后从缓存解析 peer_pos，并等待 peer_pos 非零再行动。
25. **enterWorld 必须给老客户端补发 UpdateEntity**：新玩家进入时，服务端只给老客户端
    发 PreloadPlayer（告知存在），没发 UpdateEntity（告知位置）。老客户端因此知道
    对方存在但没位置，无法走位/攻击。修复：`sendToClient(oc, nil, rpcUpdateEntity,
    updateEntityArgs(e))` 与 PreloadPlayer 一起发给老客户端。
26. **物品堆叠是隐含契约**：SoM 的 Inventory 对 stackable 物品会合并到同一 Item 条目
    （count 累加）。服务端 PickupDrop/MerchantTrade 若不堆叠，会生成多个 count=1 的
    条目，导致 `removeItem(count=3)` 只扣一个条目（剩 2 个），余额/数量错乱。
    修复：所有增益路径（PickupDrop、MerchantTrade buy）检查已有条目并累加 count。
27. **MerchantTrade 的 GP 校验是 `chr.GP + balance < 0`**：balance 是单次交易的
    净额（卖出−买入），不是玩家最终 GP。买入时 balance 为负，但 `chr.GP + balance
    >= 0` 就该放行。写成 `if balance < 0 { reject }` 会导致纯买入永远被拒。
    另外商人 balance 按 sellEntries/buyEntries 数组计算，条目格式
    `[itemID, count, customfield]`，与 TradeSetOffer 的 `[itemID,count,customfield]`
    一致。
28. **SoM 等级是 0 基的**：`Stats.level` 内部从 **0** 开始（不是 1）。
    `Experience.GetNeededExperienceForNextLevel(level)` 直接索引
    `_experienceTable[level]`，所以 level 0→1 需要 9 XP（表第 0 项），
    level 1→2 需要 16 XP（表第 1 项）。服务端 Character.Level 必须初始化为 0，
    enterWorld 实体 Level 也必须为 0，否则 neededExperience 会偏移 1 级。
    测试要按 0 基写：neededExperience(0)=9, neededExperience(1)=16,
    grantExperience(chr, 9) 从 level 0 → level 1。
29. **nameID 表必须只含 @rpc 方法**：Network.gd 中有非 @rpc 函数（Bulk/CallClient/
    CallServer/Destroy/IsServerReachable/Mode/NotifyArea/NotifyGlobal/NotifyInstance/
    NotifyNeighbours/RtcAnswer 等），早期提取误含导致 129 方法（实际 119）。
    nameID 按**方法名字典序**分配（@rpc config 第 4 参数被 Godot 忽略）。
    正确 119 方法表：AddOnlinePlayer=0, AuthError=1, AuthTokenResult=2, BulkCall=3,
    CameraLookAt=4, CameraReset=5, Casted=6, ChangePassword=7, CharacterError=8,
    CharacterInfo=9, CharacterListing=10, ChatPlayer=11, ChatQuery=12, ChatSystem=13,
    ClearNavigation=14, ClearProgressionTracker=15, CommandFeedback=16,
    CommandModifier=17, ConfirmPasswordReset=18, ConnectCharacter=19, ContextChoice=20,
    ContextClose=21, ContextContinue=22, ContextText=23, ContextThink=24,
    CreateAccount=25, CreateCharacter=26, DeleteCharacter=27, DisconnectAccount=28,
    DisconnectCharacter=29, DisplayActions=30, DisplayProgressionTracker=31,
    DropAdded=32, DropItem=33, DropRemoved=34, Emote=35, EntityBuff=36,
    EquipItem=37, Express=38, FullUpdateEntity=39, HighlightUI=40, ItemAdded=41,
    ItemEquiped=42, ItemRemoved=43, LevelUp=44, LoginWithPassword=45,
    LoginWithToken=46, MerchantOpened=47, MerchantTrade=48, Morphed=49,
    OpenUI=50, PartyInvite=51, PartyUpdate=52, PickupDrop=53, PreloadEntity=54,
    PreloadPlayer=55, PushNotification=56, RefreshEquipment=57,
    RefreshInventory=58, RefreshOnlineList=59, RefreshProgress=60,
    RemoveEntity=61, RemoveOnlinePlayer=62, RequestOnlineList=63,
    RequestPasswordReset=64, RequestRtcUpgrade=65, RetrieveCharacterInformation=66,
    RetrieveInventory=67, RtcAnswer=68, RtcCandidateToClient=69,
    RtcCandidateToServer=70, RtcConfig=71, RtcOffer=72, RtcReady=73,
    SetAttributes=74, SetClickPos=75, SetMovePos=76, SetOrientation=77,
    SetViewportSize=78, TargetAlteration=79, ThrowProjectile=80, ToggleContext=81,
    TradeCancel=82, TradeClosed=83, TradeLock=84, TradeOpened=85,
    TradeRequested=86, TradeRespond=87, TradeSetOffer=88, TradeUpdate=89,
    TriggerChat=90, TriggerChoice=91, TriggerCloseContext=92,
    TriggerCommand=93, TriggerEmote=94, TriggerExplore=95, TriggerInteract=96,
    TriggerNextContext=97, TriggerPartyAccept=98, TriggerPartyDecline=99,
    TriggerPartyInvite=100, TriggerPartyKick=101, TriggerPartyLeave=102,
    TriggerPartyPromote=103, TriggerRespawn=104, TriggerSelect=105,
    TriggerSit=106, TriggerSkill=107, UnequipItem=108, UpdateAttributes=109,
    UpdateBestiary=110, UpdateBuff=111, UpdateEntity=112, UpdatePrivateStats=113,
    UpdatePublicStats=114, UpdateQuest=115, UpdateSkill=116, UseItem=117,
    WarpPlayer=118。
30. **UpdateBuff 发给目标，不发给施法者**：applyBuff 的 UpdateBuff 推送到
    `e.PeerID`（被 buff 的实体），EntityBuff 才是 AOI 广播（含施法者）。
    e2e 中 buffA（施法者）只验证 EntityBuff；buffB（目标）验证
    EntityBuff + UpdateBuff。写反会永远收不到 UpdateBuff。
31. **e2e 角色分支必须覆盖所有 role**：新增 buffB 时忘记加
    `elif role == "buffB"` 分支 → my_nick 保持默认 "Wone"，
    导致 buffA 永远等不到 "Wbb" 可见。每个新 role 都要在 _ready()
    的 if/elif 链里加分支，并确认 my_nick/peer_nick 正确。
32. **e2e 退出前要留缓冲帧**：peerA 发送 chat/emote/cast 后立即 quit(0)，
    服务器来不及广播给 peerB，peerB 看到 chats=1 但 emotes=0/casts=0。
    修复：记录 gameplay_sent_frame，退出前等 30 帧（`frames -
    gameplay_sent_frame < 30` 时 return）。

## 阶段3 剩余清单（全量迁移）
1. ~~NPC/任务~~ ✅ 完成（npc.go + TestNpcQuestLoopback + e2e npc 角色 PASS）。
2. ~~组队~~ ✅ 完成（party.go + TestPartyService/TestPartyLoopback + e2e partyA/B PASS）。
3. ~~交易~~ ✅ 完成（trade.go + TestTradeLoopback/TestTradeDecline + e2e tradeA/B PASS）。
4. ~~战斗结算~~ ✅ 完成（combat.go + TestCombatLoopback + e2e combatA/B PASS）。
5. ~~角色存档扩展~~ ✅ 完成（login.go + store.go 新字段 + RefreshInventory/Equipment + e2e 全绿）。

## 阻塞
（无）

## 阶段4 交付物（2026-11-09 完成）
- `items.go`（~330 行）：物品目录（6 物品）+ 背包操作（UseItem/DropItem/
  EquipItem/UnequipItem/PickupDrop）+ 属性分配（SetAttributes）+ 商人交易
  （MerchantTrade，balance 计算 + GP 校验 + 堆叠）+ notifyInstance 广播。
- `login.go`：AccountStore.findCharacterByID + persistLocked 辅助。
- `movement.go`：enterWorld 给老客户端补发 UpdateEntity（修复位置缺失）。
- 测试：TestItemsLoopback + TestEquipUnequipLoopback 全绿。
- e2e 新增 itemA 角色（PickupDrop→MerchantTrade→UseItem→SetAttributes→
  EquipItem 拒绝校验）→ 七场景十进程并发 PASS。

## 阶段5 交付物（2026-11-09 完成）
- `progression.go`（~200 行）：135 级经验表（从 SoM Experience.gd 提取）+
  neededExperience + grantExperience（循环升级）+ 击杀经验（NPC: 5+level×2，
  玩家: 10+level×5）+ pushLevelUp（AOI 广播）+ broadcastEntityState。
- 处理器：CameraLookAt/CameraReset/TriggerSit（IDLE↔SIT）/SetViewportSize/
  RequestOnlineList/ChangePassword/RequestPasswordReset/ConfirmPasswordReset。
- `combat.go`：grantKillExperience 在 applyDamage 击杀时结算 XP +
  TargetAlteration(EXP) 推送 + LevelUp 广播。
- `store.go` + `movement.go`：Character.Level/实体 Level 初始化为 **0**
  （SoM 0 基等级对齐）。
- 测试：TestExperienceTable + TestGrantExperience + TestProgressionLoopback 全绿。
- e2e：combatA 击杀 combatB → 服务器日志 `progression: Wca reached level 1 (+10 XP)`。

## 驱动记录
- 2026-11-09 16:xx 方案A 建立：`~/.meituan-catpaw/paw.json` 门控解锁，`paw send -c <convId>` 注入验证成功，cron */15 注册
- 2026-11-09 05:30（本会话时钟）阶段1 完成：login.go + 环回测试全绿 + Godot e2e PASS×2；
  run_e2e.sh 固化；关键经验 6 条入档。下一轮驱动直接进入阶段2 清单第 1 项。
- 2026-11-09 05:38 修复 nameID 全表偏移（Emote 空行致 119 法），阶段2 服务端完成：
  movement.go（enterWorld/leaveWorld/移动 handler/10Hz tick/AOI 广播）+
  TestWorldSyncLoopback 全绿 + login-chain e2e 复验 PASS。剩余：双 Godot 客户端 e2e。
- 2026-11-09 06:2x 阶段2 双客户端 e2e 完成：读 Godot 源码确认 @rpc config 被忽略 →
  改为**多进程**方案（E2E_ROLE=peerA/peerB 并发），run_e2e.sh 一键跑；
  连续 8/8 SOLO+PEERS PASS。关键经验 9-11 入档。下一轮驱动进入阶段3。
- 2026-11-09 07:0x 阶段3 首批完成：gameplay.go（聊天/表情/施法广播）+ TestGameplayLoopback 全绿；
  修复 e2e 客户端假阳性（自身回声误判）→ SOLO+PEERS 连续 5/5 PASS；
  store.go 持久化（原子写 + 重启恢复）+ 5 个持久化测试全绿；关键经验 12-14 入档。
- 2026-11-09 07:3x 阶段3 NPC/任务完成：npc.go（2 NPC + 2 任务 + 全套 Trigger 处理器 +
  进度存储）+ enterWorld 追加 NPC 预加载/RefreshProgress + TestNpcQuestLoopback 全绿；
  e2e 新增 npc 角色（交互→选项→任务激活→选中属性）→ SOLO+PEERS+NPC 三进程并发 PASS；
  关键经验 15-16 入档。
- 2026-11-09 07:5x 阶段3 组队完成：party.go（PartyService 数据模型 + 6 个 TriggerParty*
  处理器 + PARTY 聊天 + leaveWorld 清理）+ TestPartyService/TestPartyLoopback 全绿；
  e2e 新增 partyA/partyB 角色（邀请→接受→PartyUpdate→PARTY 聊天→晋升 leader 变更）→
  SOLO+PEERS+NPC+PARTY 四场景五进程并发 PASS；关键经验 17-18 入档。
- 2026-11-09 08:1x 阶段3 交易完成：trade.go（tradeService 状态机 + /trade 命令解析 +
  TradeRespond/SetOffer/Lock/Cancel + 全套推送 + leaveWorld 清理）+
  TestTradeLoopback/TestTradeDecline 全绿；e2e 新增 tradeA/tradeB 角色
  （请求→接受→打开→报价→双方锁定→成交）→ SOLO+PEERS+NPC+PARTY+TRADE 五场景七进程并发 PASS；
  关键经验 19-21 入档。期间修复 e2e 竞态（发起方未等对方可见就邀请）。
  下一轮驱动进入阶段3 剩余清单第 4 项（战斗结算）。
- 2026-11-09 08:2x 阶段3 战斗结算完成：combat.go（skillTable + computeDamage SoM 公式 +
  applyDamage + killEntity + handleTriggerRespawn + handleTriggerSkill 距离/冷却校验）+
  TestCombatLoopback 全绿（修复 gorilla/websocket SetReadDeadline 流损坏 → 专用 pushCollector）；
  e2e 新增 combatA/combatB 角色（A 走位→Flar 击杀 B→B 重生）→ 六场景九进程并发 PASS；
  关键经验 22-24 入档。
- 2026-11-09 08:4x 阶段3 角色存档扩展完成：Character 结构体扩展 Experience/GP/Health/
  Mana/Stamina/Karma/Weight/Inventory/Equipment；storeFileCharacter 序列化新字段；
  characterInfoDict 发送持久化值；enterWorld 连接时推送 RefreshInventory +
  RefreshEquipment（镜像 SQL.GetStorage + SQL.GetEquipment）；
  login_test.go wantOrder 更新为 11 推送（含两个新 nameID 57/58）；
  e2e 修复 combatA peer_pos 竞态（UpdateEntity 先于 PreloadPlayer 到达时丢失位置 →
  last_pos_by_rid 缓存 + 等待非零位置）；**run_e2e.sh 六场景全 PASS**；
  阶段3 全量迁移完成。
- 2026-11-09 08:5x 阶段4 物品/经济/成长系统完成：items.go（物品目录 + UseItem/
  EquipItem/UnequipItem/DropItem/PickupDrop + SetAttributes + MerchantTrade）+
  TestItemsLoopback/TestEquipUnequipLoopback 全绿；e2e 新增 itemA 角色
  （PickupDrop×3→卖出+75GP→买入Antidote-30GP→UseItem→SetAttributes→
  EquipItem 拒绝校验）→ 七场景十进程并发 PASS；关键经验 25-27 入档。
  期间修复 3 个 bug：① enterWorld 未给老客户端发 UpdateEntity（位置缺失）；
  ② PickupDrop 不堆叠导致 sell 扣减错；③ MerchantTrade GP 校验应为
  `chr.GP + balance < 0` 而非 `balance < 0`。
- 2026-11-09 09:0x 阶段5 等级/经验/Buff/镜头/坐/密码管理完成：progression.go
  （135 级经验表 + neededExperience + grantExperience + 击杀经验 +
  CameraLookAt/CameraReset/TriggerSit/SetViewportSize/RequestOnlineList +
  ChangePassword/RequestPasswordReset/ConfirmPasswordReset）+
  TestExperienceTable/TestGrantExperience/TestProgressionLoopback 全绿；
  e2e 验证 combatA 击杀 combatB 后 `progression: Wca reached level 1 (+10 XP)`；
  期间修复 **0 基等级对齐 bug**：SoM Stats.level 从 0 开始（非 1），
  Character.Level/enterWorld 实体 Level 初始化为 0，neededExperience(0)=9；
  七场景十进程并发 PASS。阶段5 完成。
- 2026-11-09 09:4x 阶段6 Buff/私有属性/掉落/通知/投掷物/变身/移除实体/
  背包刷新完成：buffs.go（Buff 系统 + 地面掉落 + 私有属性 + 13 个杂项 handler）+
  combat.go（技能 buff 效果 + NPC 掉落）+ movement.go（RemoveEntity + 私有属性推送）+
  login.go（13 个 handler 接线）+ main.go（Server 字段）+ gameplay.go（broadcastToMap）；
  **关键修复：nameID 全表从 129 方法修正为 119 @rpc 方法**（修复 15 个错误常量）；
  TestBuffs/TestDrops/TestPrivateStats/TestRetrieveInventory/TestPhase6MiscHandlers 全绿；
  e2e 新增 buffA/buffB 角色（buffA 走位→Inma→EntityBuff；buffB 被动→EntityBuff+UpdateBuff）→
  八场景十二进程并发 PASS；关键经验 29-32 入档。阶段6 完成。
- 2026-11-09 10:5x 阶段8 SoM 客户端尚未接入的系统完成：phase8.go（NPC 世界实体 +
  图鉴追踪 + 技能进度 + 属性存储 + 42 个 GM 命令 + 检索角色信息）+
  items.go（SetAttributes 上限校验）+ npc.go（progressStore 图鉴/技能方法）+
  combat.go（trackKill 接入击杀流程）+ movement.go（Entity.NpcRID 字段）+
  login.go（Character.Attributes 字段 + RetrieveCharacterInformation 接线）+
  main.go（randSrc + spawnNpcEntities 调用）；
  **关键修复：trackKill 对 spawn 出的 NPC 实例无法追踪**（npcByCharID 只认种子 NPC
  的负 CharID，spawn 的 NPC 用 RID*1000+i+1 编码）→ Entity 新增 NpcRID 字段，
  trackKill 优先读 NpcRID；
  TestBestiaryTracking/TestSkillProgression/TestGMSpawnAndKillAll/TestGMWarpAndJump/
  TestGMGotoAndRecall/TestGMBroadcast/TestGMWhisperAndQuery/TestGMItemAndQuest/
  TestGMStatCommands/TestGMKillAndRevive/TestRetrieveCharacterInformation/
  TestGMCommandFeedback/TestGMPermissionAndBan/TestGMDisableNpc/TestGMGodmodeAndHide/
  TestProgressStoreBestiary/TestMaxAttributePoints/TestHashString/TestGMInvisible/
  TestGMRecallNpc/TestGMDeploy/TestGMQuerySelf/TestGMWhisperSelf/TestGMKick/
  TestGMExperience/TestGMSpeed/TestGMIpCheck/TestGMBanList/TestGMIpBan/
  TestProgressStoreData 全绿（27 个新测试）；
  回归修复：TestItemsLoopback（SetAttributes 上限 4 点 + 超额拒绝）、
  TestWorldSyncLoopback（NPC 实体推送顺序 16 推送 + PreloadPlayer 按 nick 查找）；
  `go vet ./...` / `go test ./...` 全过（59 个测试）；阶段8 完成。
