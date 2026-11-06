#!/usr/bin/env bash
# tick17: SoM LLM policy 位 E2E — 起隔离三元组 → 探针在线 → LLM run + arena episode
set -u
cd "$(dirname "$0")/../.."
PORT=${PORT:-7341}; WSPORT=${WSPORT:-7342}; BRIDGE=${BRIDGE:-9341}
LOG=/tmp/som_llm_server_$PORT.log
pgrep -af "godot471 --headless --path . --server --port $PORT" >/dev/null || {
  godot471 --headless --path . --server --port $PORT --ws-port $WSPORT --bridge-port $BRIDGE >"$LOG" 2>&1 &
  sleep 6
}
pgrep -af "godot471 --headless --path . --server --port $PORT" | head -1
# agent probe (keep online)
godot471 --headless --path . --port $PORT -- --agent-client llm_t17 --agent-keep >/tmp/som_llm_probe_$PORT.log 2>&1 &
sleep 9
# env from Kaetram runtime .env
set -a; source ~/kaetram/packages/agent-runtime/.env; set +a
export BRIDGE_PORT=$BRIDGE POLICY=llm MAX_ACTIONS=18 DURATION=420
python3 tests/harness/acceptance_som.py $BRIDGE 1
grep -c "SCRIPT ERROR" $LOG || true
