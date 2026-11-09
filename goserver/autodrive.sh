#!/bin/bash
# SoM autodrive — injects a driving prompt into the main session every 15 min.
# Mechanism: paw send -c <conversationId> (unblocked by ~/.meituan-catpaw/paw.json)
set -u

PAW=/mnt/data/catpaw/home/.meituan-catpaw/bin/paw
CONV_ID=176736d8-eb85-472f-beb2-0e4b32f0b157
WORKSPACE=/mnt/data/catpaw/home/.meituan-catpaw/333630993/desk_default_workspace
LOG=$WORKSPACE/goserver/autodrive.log
STATE=$WORKSPACE/goserver/AUTO_STATE.md
LOCK=/tmp/som-autodrive.lock

log() { echo "[$(date '+%F %T')] $*" >> "$LOG"; }

# Prevent overlapping injections (previous drive still running)
if [ -f "$LOCK" ]; then
  pid=$(cat "$LOCK" 2>/dev/null)
  if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
    log "SKIP: previous drive (pid=$pid) still running"
    exit 0
  fi
fi
echo $$ > "$LOCK"

PROMPT="【自动驱动·方案A】继续 SoM Go 服务端迁移（阶段1-3）。执行循环：
1. 读 goserver/PROGRESS.md 与 goserver/AUTO_STATE.md 确认当前阶段与下一步；
2. 编码：推进下一步（阶段1=登录链 CreateAccount→Login→CreateCharacter→IN_GAME；阶段2=AOI/移动同步；阶段3=战斗/技能/任务全量迁移）；
3. 自我审计：go vet ./... && go test ./...（goserver 下），跑 Godot e2e 客户端验证；
4. 自我规划：把结果写入 AUTO_STATE.md（时间戳+完成项+下一步），更新 PROGRESS.md；
5. 若阶段全部完成或遇到必须由用户决策的阻塞，输出明确说明。
本条为定时注入，无需等待用户确认，直接开始。"

log "DRIVE start: injecting prompt"
if "$PAW" send -c "$CONV_ID" -d "$WORKSPACE" "$PROMPT" >> "$LOG" 2>&1; then
  log "DRIVE ok"
else
  log "DRIVE FAILED exit=$?"
fi

rm -f "$LOCK"
