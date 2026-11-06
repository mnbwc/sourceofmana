#!/usr/bin/env bash
# tick17/22: SoM LLM policy 位 E2E — 起隔离三元组 → 探针在线 → LLM arena episode（含 arena_start）。
# tick21 教训：无 arena_start 的 LLM run 出生在无怪区，0 kill 是场景问题非能力问题——
# 本脚本统一走 llm_arena_e2e.py（arena_start → LLM decide → episode_end 判定）。
set -u
cd "$(dirname "$0")/../.."
PORT=${PORT:-7341}; WSPORT=${WSPORT:-7342}; BRIDGE=${BRIDGE:-9341}
LOG=/tmp/som_llm_server_$PORT.log
pgrep -af "godot471 --headless --path . --server --port $PORT" >/dev/null || {
  godot471 --headless --path . --server --port $PORT --ws-port $WSPORT --bridge-port $BRIDGE >"$LOG" 2>&1 &
  sleep 6
}
pgrep -af "godot471 --headless --path . --server --port $PORT" | head -1
# agent probe (keep online, unique nick per run to avoid EPISODE_ACTIVE residue)
NICK=${NICK:-llm_s$PORT}
godot471 --headless --path . --port $PORT -- --agent-client "$NICK" --agent-keep >/tmp/som_llm_probe_$PORT.log 2>&1 &
sleep 9
# env from Kaetram runtime .env (keys stay in env, never in repo)
set -a; source ~/kaetram/packages/agent-runtime/.env; set +a
export BRIDGE_PORT=$BRIDGE MAX_ACTIONS=${MAX_ACTIONS:-24} DURATION=${DURATION:-420}
python3 tests/harness/llm_arena_e2e.py
grep -c "SCRIPT ERROR" "$LOG" || true
