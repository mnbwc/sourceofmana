# 中文版 MVP 交付说明

> 项目：Source of Mana 中文版 MVP（规格：《最小 MVP 范围与开工规格 v1.0》）
> 分支：`feat/agent-arena` ｜ 验收：**AC-1..AC-8 全过，12/12 PASS**（`tests/harness/mvp11_acceptance.py`，commit `676c134`）
> 日期：2026-11-06

## 一、交付范围（五件事 + 一个玩法模块）

| # | 项 | 状态 | 关键提交 |
|---|---|---|---|
| 1 | 原版 debug（D1–D9） | ✅ | 掉落截断（6405216）、排行榜截断（852fde2）、账号自踢、Sentry DSN 清空、Discord 外发移除、D7 被拒提示（148fed6） |
| 2 | AI 友好 | ✅ A1 | `[AI]` 昵称前缀 + traits.ai 持久化（010 迁移，58136de）+ 桥 isAgent 审计；**零业务短路**（AC-2 静态断言过）。A2 人类侧开关随组队 UI 后置 |
| 3 | 组队 | ✅ 核心 | PartyService + Server.Party\* 方法族 + PARTY 频道 + 桥 party op（4a8fe4a）；三种编队同入口，XP/掉落沿用原版单人规则（PM-5） |
| 4 | 观战 | ✅ | SpectatorFeed 只读桥 ops（e405b4c）+ SSE 网关 + 单文件 HTML 状态条/时间线页（93fa728）；`since=N` 断线续传；AI 与人类同一协议 |
| 5 | 汉化 | ✅ 核心路径 | 方案 A（CSV+TranslationServer）：UI 132 + NPC 961 + 任务 71 + 物品 97 ≈ **1,260 条中文**；CJK 字体 NotoSansCJK（OFL）fallback；**物品 name= 内部键零改动**（显示名分离层，MVP-6 红线落地） |
| 🧩 | AI 竞技场玩法模块 | ✅ 封装 | `ARENA_MODE` 开关（59d2090）：off=主世界零感知（arena_start 拒绝）；on=完整 episode（模板/四终局/权威计分/排行榜按原规格）；代码零删改 |

## 二、验收判据（规格 §7）

- AC-1 汉化：四 CSV tr() 实测命中中文，字体 fallback 实测 `has_char(中)=true`；桌面 IME 人工复核为**唯一挂账项**（headless 无法模拟键盘）。
- AC-2 AI 友好：人类替身+AI 同实例可见/可组队/可辨识，静态断言无 isAgent 业务短路。
- AC-3 组队：人类↔AI 邀请→接受→PARTY 聊天→离队 E2E；PARTY 不串 LOCAL/GLOBAL。
- AC-4 观战：浏览器不开客户端实时看 AI 事件流+状态条；`since=N` 续传无丢事件。
- AC-5 debug：D1–D9 全绿；**diff 无任何数值/掉落表/词条/曲线改动**（R3 清单声明）。
- AC-6 隔离：DSN 清空+Discord 外发删除+私有实例；中文版不对外发布。
- AC-7 无残留（双向）：off 主世界零 episode 语义；on 时模块语义只作用模块场次。
- AC-8 模块保留：on 跑通完整 episode（expired 落盘实证），模块代码零删改。

## 三、玩法模块登记表（规格 §6.1 实机版）

- **归属**：`feat/agent-arena` 分支，`sources/network/`（AgentBridge/ArenaController/ArenaLeaderboard）+ `tests/harness/`。
- **已实机落地**：桥五 op+spectate 三 op、verdict、事件闭环、ArenaController（tutorial_maggot/grind_maggot、四终局、baseline 差值权威计分）、排行榜 JSONL、trace、LLM 对局、并发隔离。
- **规格覆盖、未落地**：EventBus 三层断言、golden 单测、六项公平 CI、归因三件套、KAB、锁步/mapSeed/checksum。
- **许可**：模块代码全自建（MIT 相容）；新增字体 NotoSansCJK 走 OFL（`data/graphics/fonts/LICENSE-NotoSansCJK.txt`）；无上游美术复用。

## 四、挂账（下一步）

1. 桌面端 IME 中文输入人工复核（AC-1 唯一剩余人工项）
2. 人类组队完整 UI 面板 + A2「不与 AI 组队/交易」开关（当前最小 UI=系统聊天提示，方法族已就绪）
3. episode 长场差值镜像计分复查（快照 kills=2 vs episode 0 疑点）
4. 模块内专业文本（模板名/终局名/错误码）汉化——规格 §6.1 按「ID/专有名词不译」后置

## 五、许可义务

- 中文翻译为 CC BY-SA 4.0 演绎材料：同许可+署名+标明改动；`LICENSE.md` 逐文件清单已含 NotoSansCJK 登记；不暗示官方背书。
- 仅自建私有实例使用（PM-3），不推上游、不对外发布。
