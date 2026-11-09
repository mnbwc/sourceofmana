# SoM Go 服务端迁移 — 总进度

## 阶段0：协议可行性验证 ✅
- Go WebSocket 端点讲 Godot SceneMultiplayer wire 协议（SIMPLIFY_PATH / CONFIRM_PATH / RPC 包）。
- variant 编解码 + rpc 包构建/解析有单测；真 Godot 客户端握手 + RPC 回包成功。

## 阶段1：登录链 ✅（2026-11-09）
- 服务端：CreateAccount → (AuthTokenResult) → AuthError → CharacterListing → CreateCharacter →
  ConnectCharacter(IN_GAME)，含 LoginWithPassword/LoginWithToken/DeleteCharacter 等全套。
- 会话状态机 + 内存账户库（AccountStore），约束与 NetworkCommons 一致。
- 测试：`go test ./...` 全过（含 TestLoginChainLoopback 环回集成测试）。
- e2e：真 Godot 4.7.1 无头客户端跑完 5 步登录链，`run_e2e.sh` 一键复跑，连续 PASS。

## 阶段2：AOI/移动状态同步 ✅ 完成（2026-11-09）
- movement.go：服务端权威轻量世界（Entity/10Hz tick/AOI 半径 1000px 广播）。
- enterWorld 序列：WarpPlayer→FullUpdateEntity→PreloadPlayer 双向交换→UpdateEntity→
  RefreshOnlineList→AddOnlinePlayer；leaveWorld 广播 RemoveOnlinePlayer。
- 移动 RPC：SetMovePos/SetClickPos/SetOrientation。
- nameID 全表修正（Emote 空行导致 >35 全偏移 1，详见 AUTO_STATE.md 经验 7）。
- 测试：TestWorldSyncLoopback（双客户端 enter 推送顺序 + 移动广播）全绿。
- **双 Godot 客户端 e2e**：run_e2e.sh 并发启动两个真 Godot 4.7.1 进程
  （E2E_ROLE=peerA/peerB），验证互见 + 在线列表 + 移动同步，连续 8/8 PASS。

## 阶段3：全量迁移 ✅ 完成（2026-11-09）

### 已完成
- **聊天/表情/施法广播**（`gameplay.go`）：TriggerChat（LOCAL→AOI 含回声 / GLOBAL→全服 /
  PARTY→系统提示）、TriggerEmote、Casted 的 NotifyNeighbours/NotifyGlobal 通知层；
  TestGameplayLoopback 全绿。
- **持久化**（`store.go`）：AccountStore → JSON 文件（`SOM_STORE_FILE`，默认
  `som_store.json`），原子写（temp + rename），启动时 loadStore 恢复；
  Add/AddCharacter/RemoveCharacter/SetToken 自动落盘。5 个持久化测试全绿
  （含 TestPersistenceSurvivesRestart 双实例共享 store 文件）。
- **NPC/任务系统**（`npc.go`）：seed 2 NPC（Elder/Merchant）+ 2 任务；
  enterWorld 追加 PreloadEntity（NPC）+ RefreshProgress；TriggerInteract/TriggerChoice/
  TriggerNextContext/TriggerCloseContext/TriggerSelect/TriggerExplore/TriggerCommand
  全套处理器；任务进度按角色存储（Progress.SetQuest），UpdateQuest 推送；
  TestNpcQuestLoopback 全绿；**e2e 新增 npc 角色**：Elder 预加载 → 交互 → 文本+继续 →
  下一段 → 文本+选项 → 选择 → 任务激活 → 选中查看属性，全链 PASS。
- **组队系统**（`party.go`）：PartyService 数据模型（create/join/leave/kick/promote/
  invite，含队长转让与解散）；TriggerPartyInvite/Accept/Decline/Leave/Kick/Promote
  全套处理器；PartyUpdate 推送（members 为 JSON 数组字符串 + leaderRID）；
  PARTY 聊天落地（仅队员可见）；leaveWorld 清理邀请与队伍；
  TestPartyService + TestPartyLoopback 全绿；**e2e 新增 partyA/partyB 角色**：
  邀请 → 接受 → PartyUpdate → PARTY 聊天 → 晋升（leader 变更）→ 双 PASS。
- **交易系统**（`trade.go`）：tradeService 状态机（request/respond/open/setOffer/
  setLock/commit/cancel，含 30s 请求过期与距离校验）；`/trade <nick>` 命令解析
  （经 TriggerCommand）；TradeRespond/SetOffer/Lock/Cancel 处理器；TradeRequested/
  TradeOpened/TradeUpdate/TradeClosed 推送；leaveWorld 清理请求与会话；
  TestTradeLoopback + TestTradeDecline 全绿；**e2e 新增 tradeA/tradeB 角色**：
  请求 → 接受 → 打开 → 报价 → 双方锁定 → 成交 → 双 PASS。
- **战斗结算**（`combat.go`）：skillTable（6 技能，含 Flar/Inma/ManaBurst 等）+
  computeDamage（SoM 公式：attack+skillValue-defense，闪避→DODGE，暴击→CRIT）+
  applyDamage（血量/死亡/广播）+ killEntity（state=DEATH + TargetAlteration 广播）+
  handleTriggerRespawn（Revive + WarpTo 出生点）+ handleTriggerSkill（距离校验
  IsNear + 冷却 + Casted 广播）；TestCombatLoopback（A 施放 Flar 击杀 B，B 重生）全绿。
- **角色存档扩展**（`login.go` + `store.go`）：Character 结构体扩展
  Experience/GP/Health/Mana/Stamina/Karma/Weight/Inventory/Equipment；
  storeFileCharacter 序列化新字段；characterInfoDict 发送持久化值；
  enterWorld 连接时推送 RefreshInventory + RefreshEquipment
  （镜像 SQL.GetStorage + SQL.GetEquipment）；e2e 客户端新增 combatA/combatB 角色。

### 待办
- （无——阶段3 清单全部完成）

### 当前状态
- `go vet ./...` / `go test ./...` 全过（17 个测试）。
- `run_e2e.sh`：SOLO + PEERS + NPC + PARTY + TRADE + COMBAT 六场景 PASS（九真 Godot 进程并发）。

## 阶段4：物品/经济/成长系统 ✅ 完成（2026-11-09）

### 已完成
- **物品目录**（`items.go`）：itemCatalog 种子 6 物品（Potion/Antidote/Iron Sword/
  Leather Armor/Wooden Shield/Cloth Boots），字段 Slot/Stackable/Usable/BuyPrice/
  SellPrice/Weight 镜像 SoM ItemCell。
- **背包操作**：UseItem（消耗品堆叠扣减 + ItemRemoved 推送）、DropItem（按 index
  扣减 + ItemRemoved）、EquipItem（背包→装备槽 + ItemEquiped 实例广播）、
  UnequipItem（装备槽→背包 + ItemEquiped）、PickupDrop（堆叠增益 + ItemAdded）。
- **属性分配**：SetAttributes（5 属性）→ UpdateAttributes 回显。
- **商人交易**：MerchantTrade（sellEntries/buyEntries 数组，balance = 卖出总额 −
  买入总额，GP 余额校验 `chr.GP + balance >= 0`，堆叠购买，TradeClosed 推送）。
- **测试**：TestItemsLoopback（PickupDrop→UseItem→MerchantTrade sell/buy→
  SetAttributes 全链）、TestEquipUnequipLoopback（堆叠/商人/不可装备物品校验）全绿。
- **e2e 新增 itemA 角色**：PickupDrop×3 → 卖出 3 药水(+75 GP) → 买入 Antidote
  (-30 GP) → UseItem → SetAttributes → EquipItem(不可装备物品应拒绝)，全链 PASS。
- **enterWorld 修复**：新玩家进入时给老客户端补发 UpdateEntity（此前只发
  PreloadPlayer，导致老客户端知道对方存在但没位置）。

### 当前状态
- `go vet ./...` / `go test ./...` 全过（21 个测试）。
- `run_e2e.sh`：SOLO + PEERS + NPC + PARTY + TRADE + COMBAT + ITEMS 七场景 PASS（十真 Godot 进程并发）。

## 阶段5：等级/经验/Buff/镜头/坐/密码管理 ✅ 完成（2026-11-09）

### 已完成
- **经验表**（`progression.go`）：从 SoM `Experience.gd` 提取 135 级经验表，
  `neededExperience(level)` 镜像 `GetNeededExperienceForNextLevel`。
- **0 基等级对齐**：SoM `Stats.level` 内部从 **0** 开始（非 1），
  `Character.Level` 初始化为 0，`enterWorld` 实体 `Level: 0`，
  `neededExperience(0)=9`、`neededExperience(1)=16` 与 SoM 一致。
- **升级循环**：`grantExperience` 循环扣减经验表并提升等级，返回升级次数。
- **击杀经验**（`combat.go`）：`grantKillExperience` 在目标死亡时结算
  （NPC: 5+level×2，玩家: 10+level×5），推送 TargetAlteration(EXP) +
  LevelUp（AOI 广播）。
- **镜头控制**：CameraLookAt（设置观察目标）、CameraReset（复位）。
- **坐/站**：TriggerSit（IDLE↔SIT 切换 + 状态广播）。
- **视口尺寸**：SetViewportSize（记录客户端视口）。
- **在线列表**：RequestOnlineList → RefreshOnlineList 推送。
- **密码管理**：ChangePassword（校验旧密码）、RequestPasswordReset（生成重置码）、
  ConfirmPasswordReset（校验重置码 + 更新密码）。
- **测试**：TestExperienceTable（0 基表校验）、TestGrantExperience（多级升级）、
  TestProgressionLoopback（坐/镜头/在线列表/密码全链）全绿。
- **e2e 验证**：combatA 击杀 combatB 后服务器日志 `progression: Wca reached level 1 (+10 XP)`，
  升级流程端到端工作。

### 当前状态
- `go vet ./...` / `go test ./...` 全过（20 个测试）。
- `run_e2e.sh`：SOLO + PEERS + NPC + PARTY + TRADE + COMBAT + ITEMS 七场景 PASS（十真 Godot 进程并发）。

## 阶段6：Buff/私有属性/掉落/通知/投掷物/变身/移除实体/背包刷新 ✅ 完成（2026-11-09）

### 已完成
- **Buff 系统**（`buffs.go`）：Buff 结构体（Effect/Value/SkillID/Expires）+
  applyBuff（添加/刷新 + EntityBuff AOI 广播 + UpdateBuff 自身推送）+
  clearBuff（移除 + EntityBuff(enabled=false) + UpdateBuff(duration=0)）+
  buffsOn（过期清理）。SoM CellCommons.Modifier 枚举 0-23 全量镜像。
- **技能 Buff 效果**（`combat.go`）：skillProfile 扩展 BuffEffect/BuffValue/
  BuffDuration；Inma（防御-5，10s）+ SonicWave（移速-20，5s）带 buff；
  handleTriggerSkill 命中后 applyBuff。
- **地面掉落**（`buffs.go`）：Drop 结构体（ID/ItemID/Customfield/Pos/MapID/
  OwnerCharID/Expires）+ spawnDrop（DropAdded 广播）+ removeDrop（DropRemoved）+
  expireDrops（60s 生命周期）；NPC 死亡掉落（combat.go spawnNpcDrops）。
- **私有属性同步**：enterWorld 推送 UpdatePrivateStats（经验/GP/法力/耐力/
   karma/负重/形态/精神）；击杀经验后 pushPrivateStats 刷新。
- **移除实体**：leaveWorld 广播 RemoveEntity（客户端 despawn actor 节点）。
- **通知/表情/投掷物/变身/上下文/导航/教程 UI**：handlePushNotification、
  handleExpress、handleThrowProjectile、handleMorphed、handleToggleContext、
  handleClearNavigation、handleDisplayActions、handleDisplayProgressionTracker、
  handleClearProgressionTracker、handleHighlightUI、handleOpenUI、
  handleCommandModifier、handleRetrieveInventory（RefreshInventory +
  RefreshEquipment）。
- **nameID 全表修正**：从 129 方法（含非 @rpc 函数）修正为 **119 @rpc 方法**
  按名排序；修复 15 个错误常量（rpcPushNotification 56→66、rpcThrowProjectile
  80→90、rpcRetrieveCharacterInformation 66→76、rpcSetOrientation 77→87、
  rpcTriggerChat 90→100、rpcTriggerInteract 96→106 等）。
- **测试**：TestBuffs（applyBuff→EntityBuff+UpdateBuff→clearBuff）、
  TestDrops（spawnDrop→DropAdded→removeDrop→DropRemoved）、
  TestPrivateStats（enterPartyWorld→pushPrivateStats→UpdatePrivateStats）、
  TestRetrieveInventory（RefreshInventory+RefreshEquipment）、
  TestPhase6MiscHandlers（13 个杂项 handler）全绿。
- **e2e 新增 buffA/buffB 角色**：buffA 走位→Inma 施法→验证 EntityBuff
  （modDefense=14，enabled=true）；buffB 被动验证 EntityBuff + UpdateBuff
  （自身 stat panel 更新）→ 双 PASS。

### 当前状态
- `go vet ./...` / `go test ./...` 全过（25 个测试）。
- `run_e2e.sh`：SOLO + PEERS + NPC + PARTY + TRADE + COMBAT + ITEMS + BUFFS 八场景 PASS（十二真 Godot 进程并发）。

## 阶段8：SoM 客户端尚未接入的系统 ✅ 完成（2026-11-09）

### 已完成
- **NPC 世界实体**（`phase8.go`）：spawnNpcEntities 在服务器启动时将种子 NPC
  （Elder/Merchant）作为世界实体加入（负 CharID = -RID），使其可被技能目标化；
  Entity 新增 NpcRID 字段支持 spawn 出来的 NPC 实例的图鉴追踪。
- **图鉴追踪**（`phase8.go`）：trackKill 在 NPC 死亡时调用 progress.AddBestiary
  并推送 UpdateBestiary 给击杀者；GM `bestiary <name> <count>` 命令可手动设置。
- **技能进度**（`phase8.go`）：grantSkill/removeSkill → progress.AddSkill/RemoveSkill
  → UpdateSkill 推送；GM `skill <name> <level>` 命令可授予/修改技能。
- **属性存储**（`items.go` + `phase8.go`）：handleSetAttributes 校验属性点上限
  （Formula.GetMaxAttributePoints = 4 + 1×level），存储到 Character.Attributes，
  推送 UpdateAttributes；超额分配返回 CommandFeedback 错误。
- **GM 命令集**（`phase8.go`）：42 个 WorldCommands 全部实现——
  spawn/warp/jump/goto/recall/recallnpc/disablenpc/godmode/hide/invisible/
  stat/setstat/level/experience/gp/health/mana/stamina/speed/localbroadcast/
  broadcast/quest/bestiary/item/skill/killall/kill/revive/permission/ipcheck/
  kick/ban/unban/banlist/ipban/ipunban/ipbanlist/whisper/w/query/q/deploy。
  命令经 TriggerCommand RPC 分发，未识别命令返回 "Unknown command" 反馈。
- **击杀/复活/传送**：kill → killEntity + trackKill；revive → 恢复半血 + WarpPlayer
  回出生点；warp/jump/goto/recall → WarpPlayer + FullUpdateEntity 广播。
- **私信/查询**：whisper → ChatPlayer 双向推送；query → UpdatePublicStats 推送目标属性。
- **屏蔽/禁言**：kick/ban 关闭连接；banlist/ipban/ipunban/ipbanlist 管理命令。
- **检索角色信息**（`phase8.go`）：handleRetrieveCharacterInformation 推送
  UpdatePublicStats + UpdatePrivateStats + RefreshProgress + RefreshInventory
  + RefreshEquipment + UpdateAttributes（镜像 Server.gd RetrieveCharacterInformation）。
- **测试**（`phase8_test.go`，27 个新测试）：
  TestBestiaryTracking、TestSkillProgression、TestGMSpawnAndKillAll、
  TestGMWarpAndJump、TestGMGotoAndRecall、TestGMBroadcast、TestGMWhisperAndQuery、
  TestGMItemAndQuest、TestGMStatCommands、TestGMKillAndRevive、
  TestRetrieveCharacterInformation、TestGMCommandFeedback、TestGMPermissionAndBan、
  TestGMDisableNpc、TestGMGodmodeAndHide、TestProgressStoreBestiary、
  TestMaxAttributePoints、TestHashString、TestGMInvisible、TestGMRecallNpc、
  TestGMDeploy、TestGMQuerySelf、TestGMWhisperSelf、TestGMKick、TestGMExperience、
  TestGMSpeed、TestGMIpCheck、TestGMBanList、TestGMIpBan、TestProgressStoreData 全绿。
- **回归修复**：TestItemsLoopback（SetAttributes 上限校验 + 超额拒绝）、
  TestWorldSyncLoopback（NPC 实体推送顺序 + PreloadPlayer 按 nick 查找）已更新。

### 当前状态
- `go vet ./...` / `go test ./...` 全过（59 个测试）。
- `run_e2e.sh`：SOLO + PEERS + NPC + PARTY + TRADE + COMBAT + ITEMS + BUFFS 八场景 PASS（十二真 Godot 进程并发）。
