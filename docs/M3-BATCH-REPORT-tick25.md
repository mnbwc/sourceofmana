# M3 批测汇总报告 — tick25（2026-11-06）

来源：`tests/harness/batch_template.py`（RuleBot）+ `llm_arena_e2e.py`（LLM）+ 排行榜
`~/.local/share/SourceOfMana/arena_leaderboard.jsonl`（服务端权威计分，终局权威读法=tick22 定案）。

## grind_maggot（×10，600s，deaths≤2）

| nick | 策略 | seed | status | endReason | kills | deaths | xp | loot | 证据 |
|---|---|---|---|---|---|---|---|---|---|
| ai_batch_grind_s7381 | RuleBot | 42 | completed | kills_required_met | 10 | 1 | 10 | 3 | lineNo=5 |
| ai_ai_batch_s7 | RuleBot | 7 | failed | **max_deaths** | 2 | 3 | 2 | 2 | lineNo=9 |
| ai_ai_batch_s123 | RuleBot | 123 | expired | expired | 2 | 0 | 2 | 1 | lineNo=13 |
| ai_llmgrind / ai_llmg25 等 | LLM | — | expired/expired | 1-2 kills | — | | | | lineNo=7,8 |

要点：
- **deaths≤2 判据首次实际触发**：SEED=7 run `failed/max_deaths`（kills=2 时被 Maggot 反杀 3 次终止）——终局判定链完整可用。
- **seed 敏感性确认**：同模板 3 个 seed 三种结局（completed / failed / expired），RuleBot 挂机路线质量主导结局；M3 后续报告须按 seed 呈现分布而非单值。
- LLM 策略位（llm_arena_e2e）在 grind_maggot 尚无 completed（10-15s/决策 × 600s 预算只能 40-60 决策，且 tick24 已知预算先到为设计行为）；与 RuleBot 同 seed 对比待 LLM 专用模板（如 grind_maggot_5）。

## tutorial_maggot（×3）

| nick | 策略 | status | endReason | kills |
|---|---|---|---|---|
| ai_batch_tut_s7391 | RuleBot | completed | kills_required_met | 3 |
| ai_llmchk3 | LLM | expired | 0 | | | | | lineNo=2 |
| ai_lbfix1 | RuleBot | completed | explicit_stop | 0 |

## 排行榜聚合口径（新增）
- `tests/harness/leaderboard.py` 为查询入口（--template/--nick/--top，kills desc → xp desc → deaths asc）。
- **harness 真 bug 修复（本 tick）**：排行榜 nick 可能带 `ai_` 前缀（探针名经服务端改写，如 `ai_batch_s123` → `ai_ai_batch_s123`），batch_template.py 原按 state nick 精确匹配落空 → 误判 FAIL；已改为两种形态都试（s123 episode 复核为 `expired/expired kills=2`，非 FAIL）。
- 12→13 行历史保序（lineNo 连续），tick23 截断修复持续回归通过。

## server 侧
- 批测全程 0 SCRIPT ERROR（/tmp/s7401.log grep=0）。

## 结论与下一步（tick26）
- M3 多 seed 批测流水线成立：batch_template.py + leaderboard.py 可复跑、可聚合、终局读法权威。
- 下一步：① LLM 专用短模板（grind_maggot_5 / 5 kills）跑通 LLM vs RuleBot 同 seed 对比；② 批测汇总自动化（一个脚本跑 N seed + 出 Markdown 表，替代手工表）。
- 挂账不变：M0 许可结论书签署、P4 对外冻结解除评审（待所有者）。
