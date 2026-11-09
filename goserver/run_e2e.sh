#!/usr/bin/env bash
# Phase-1/2 e2e: ensure the Go server is running, then launch two Godot
# headless clients (peerA, peerB) concurrently. Each prints its own PASS line;
# the run passes when both peers report success. Usage: ./run_e2e.sh
set -u
DIR="$(cd "$(dirname "$0")" && pwd)"
LOG=/tmp/goserver_e2e.log

if [ ! -x "$DIR/godot-run" ]; then
	echo "missing $DIR/godot-run (copy a Godot 4.7.1 binary here)"; exit 1
fi

if ! ss -tln 2>/dev/null | grep -q ':27012'; then
	(cd "$DIR" && go build -o goserver . && setsid nohup ./goserver > "$LOG" 2>&1 < /dev/null &)
	sleep 1
fi

# Single-client login chain (regression guard).
OUT_SOLO=$( (cd "$DIR" && timeout 40 ./godot-run --headless 2>&1) )
if echo "$OUT_SOLO" | grep -q "E2E PASS: solo login chain"; then
	echo "SOLO: PASS"
else
	echo "SOLO: FAIL"
	echo "$OUT_SOLO" | tail -12
fi

# Two-client world sync + NPC/quest flow + party flow (phase 3).
(cd "$DIR" && E2E_ROLE=peerA timeout 45 ./godot-run --headless > /tmp/e2e_peerA.log 2>&1) &
PA=$!
(cd "$DIR" && E2E_ROLE=peerB timeout 45 ./godot-run --headless > /tmp/e2e_peerB.log 2>&1) &
PB=$!
(cd "$DIR" && E2E_ROLE=npc timeout 45 ./godot-run --headless > /tmp/e2e_npc.log 2>&1) &
PN=$!
(cd "$DIR" && E2E_ROLE=partyA timeout 45 ./godot-run --headless > /tmp/e2e_partyA.log 2>&1) &
PPA=$!
(cd "$DIR" && E2E_ROLE=partyB timeout 45 ./godot-run --headless > /tmp/e2e_partyB.log 2>&1) &
PPB=$!
(cd "$DIR" && E2E_ROLE=tradeA timeout 45 ./godot-run --headless > /tmp/e2e_tradeA.log 2>&1) &
PTA=$!
(cd "$DIR" && E2E_ROLE=tradeB timeout 45 ./godot-run --headless > /tmp/e2e_tradeB.log 2>&1) &
PTB=$!
(cd "$DIR" && E2E_ROLE=combatA timeout 45 ./godot-run --headless > /tmp/e2e_combatA.log 2>&1) &
PCA=$!
(cd "$DIR" && E2E_ROLE=combatB timeout 45 ./godot-run --headless > /tmp/e2e_combatB.log 2>&1) &
PCB=$!
(cd "$DIR" && E2E_ROLE=itemA timeout 45 ./godot-run --headless > /tmp/e2e_itemA.log 2>&1) &
PIA=$!
(cd "$DIR" && E2E_ROLE=buffA timeout 45 ./godot-run --headless > /tmp/e2e_buffA.log 2>&1) &
PBA=$!
(cd "$DIR" && E2E_ROLE=buffB timeout 45 ./godot-run --headless > /tmp/e2e_buffB.log 2>&1) &
PBB=$!
wait $PA; RA=$?
wait $PB; RB=$?
wait $PN; RN=$?
wait $PPA; RPA=$?
wait $PPB; RPB=$?
wait $PTA; RTA=$?
wait $PTB; RTB=$?
wait $PCA; RCA=$?
wait $PCB; RCB=$?
wait $PIA; RIA=$?
wait $PBA; RBA=$?
wait $PBB; RBB=$?

if grep -q "E2E PASS" /tmp/e2e_peerA.log && grep -q "E2E PASS" /tmp/e2e_peerB.log; then
	echo "PEERS: PASS"
	echo "--- peerA ---"; grep "E2E" /tmp/e2e_peerA.log | tail -6
	echo "--- peerB ---"; grep "E2E" /tmp/e2e_peerB.log | tail -6
else
	echo "PEERS: FAIL (peerA=$RA peerB=$RB)"
	echo "--- peerA ---"; grep "E2E" /tmp/e2e_peerA.log | tail -12
	echo "--- peerB ---"; grep "E2E" /tmp/e2e_peerB.log | tail -12
	exit 1
fi

if grep -q "E2E PASS" /tmp/e2e_npc.log; then
	echo "NPC: PASS"
	echo "--- npc ---"; grep "E2E" /tmp/e2e_npc.log | tail -8
else
	echo "NPC: FAIL ($RN)"
	echo "--- npc ---"; grep "E2E" /tmp/e2e_npc.log | tail -12
	exit 1
fi

if grep -q "E2E PASS" /tmp/e2e_partyA.log && grep -q "E2E PASS" /tmp/e2e_partyB.log; then
	echo "PARTY: PASS"
	echo "--- partyA ---"; grep "E2E" /tmp/e2e_partyA.log | tail -6
	echo "--- partyB ---"; grep "E2E" /tmp/e2e_partyB.log | tail -6
else
	echo "PARTY: FAIL (partyA=$RPA partyB=$RPB)"
	echo "--- partyA ---"; grep "E2E" /tmp/e2e_partyA.log | tail -12
	echo "--- partyB ---"; grep "E2E" /tmp/e2e_partyB.log | tail -12
	exit 1
fi

if grep -q "E2E PASS" /tmp/e2e_tradeA.log && grep -q "E2E PASS" /tmp/e2e_tradeB.log; then
	echo "TRADE: PASS"
	echo "--- tradeA ---"; grep "E2E" /tmp/e2e_tradeA.log | tail -6
	echo "--- tradeB ---"; grep "E2E" /tmp/e2e_tradeB.log | tail -6
else
	echo "TRADE: FAIL (tradeA=$RTA tradeB=$RTB)"
	echo "--- tradeA ---"; grep "E2E" /tmp/e2e_tradeA.log | tail -12
	echo "--- tradeB ---"; grep "E2E" /tmp/e2e_tradeB.log | tail -12
	exit 1
fi

if grep -q "E2E PASS" /tmp/e2e_combatA.log && grep -q "E2E PASS" /tmp/e2e_combatB.log; then
	echo "COMBAT: PASS"
	echo "--- combatA ---"; grep "E2E" /tmp/e2e_combatA.log | tail -6
	echo "--- combatB ---"; grep "E2E" /tmp/e2e_combatB.log | tail -6
else
	echo "COMBAT: FAIL (combatA=$RCA combatB=$RCB)"
	echo "--- combatA ---"; grep "E2E" /tmp/e2e_combatA.log | tail -12
	echo "--- combatB ---"; grep "E2E" /tmp/e2e_combatB.log | tail -12
	exit 1
fi

if grep -q "E2E PASS" /tmp/e2e_itemA.log; then
	echo "ITEMS: PASS"
	echo "--- itemA ---"; grep "E2E" /tmp/e2e_itemA.log | tail -8
else
	echo "ITEMS: FAIL (itemA=$RIA)"
	echo "--- itemA ---"; grep "E2E" /tmp/e2e_itemA.log | tail -12
	exit 1
fi

if grep -q "E2E PASS" /tmp/e2e_buffA.log && grep -q "E2E PASS" /tmp/e2e_buffB.log; then
	echo "BUFFS: PASS"
	echo "--- buffA ---"; grep "E2E" /tmp/e2e_buffA.log | tail -8
	echo "--- buffB ---"; grep "E2E" /tmp/e2e_buffB.log | tail -8
else
	echo "BUFFS: FAIL (buffA=$RBA buffB=$RBB)"
	echo "--- buffA ---"; grep "E2E" /tmp/e2e_buffA.log | tail -12
	echo "--- buffB ---"; grep "E2E" /tmp/e2e_buffB.log | tail -12
	exit 1
fi
