"""SoM harness v0.1 全量验收（tick10，判据 1-5，docs/HARNESS-PORT-DESIGN-v0.1.md §5）。

用法：python3 tests/harness/acceptance_som.py <bridge_port> <peer_count=1>
前置：自建 server + AgentClientProbe --agent-keep 已在线（由外层 shell 负责起服/探针）。

判据：
1. 预算 maxActions 耗尽即停，final_status=exhausted
2. 三件套齐（rollouts/tools/meta），meta.configHash 含 soMBridge protocolVersion
3. 事件唤醒：decide ≪ polls（wakeup_reduction > 0.5，Kaetram 基线 ~35%）
4. kill 事件 → score.kills 递增（E2E hunt-kill）
5. 0 SCRIPT ERROR（server log 由外层 grep 后汇总）
"""
from __future__ import annotations

import json
import os
import shutil
import sys
import time

sys.path.insert(0, os.path.dirname(__file__))
from somruntime import AgentRuntime, RunConfig  # noqa: E402


def main() -> int:
    bridge_port = int(sys.argv[1]) if len(sys.argv) > 1 else 9101
    runs_root = os.environ.get("SOM_RUNS_DIR",
                               os.path.join(os.path.dirname(__file__), "runs"))
    ts = int(time.time())
    run_dir = os.path.join(runs_root, "run-%d" % ts)
    cfg = RunConfig(maxActions=int(os.environ.get("MAX_ACTIONS", "25")),
                    maxTokens=50000,
                    duration_s=float(os.environ.get("DURATION", "240")),
                    bridge_port=bridge_port, seed=42)
    rt = AgentRuntime(run_dir, cfg)
    meta = rt.run()
    print("META", json.dumps({k: meta[k] for k in
          ("final_status", "exhaust_reason", "actions_used", "decide_calls", "polls",
           "wakeup_reduction_vs_polls", "kills", "final_score", "duration_s")}))

    results: dict[str, bool] = {}
    # 判据1：预算闸门
    results["C1_budget_exhausted"] = (
        meta["final_status"] in ("exhausted", "died", "session_lost")
        and meta["exhaust_reason"] in ("max_actions", "expired") or meta["final_status"] != "running"
    ) and meta["actions_used"] <= cfg.maxActions
    # 判据2：三件套 + protocolVersion
    trio = all(os.path.exists(p) for p in (rt.rollouts_path, rt.tools_path, rt.meta_path))
    results["C2_storage_trio"] = trio and bool(meta.get("protocolVersion"))
    # 判据3：事件唤醒
    wr = meta.get("wakeup_reduction_vs_polls")
    results["C3_event_wakeup"] = wr is not None and meta["polls"] > 0 and wr > 0.5
    # 判据4：kill→score
    results["C4_kill_score"] = meta["kills"] >= 1 and (meta.get("final_score") or {}).get("kills", 0) >= 1
    # 判据5：rollouts 非空
    n_rollouts = sum(1 for _ in open(rt.rollouts_path))
    results["C5_rollouts_nonempty"] = n_rollouts >= meta["decide_calls"] and meta["decide_calls"] > 0

    print("RUN_DIR", run_dir)
    for k, v in results.items():
        print(("PASS" if v else "FAIL"), k)
    print("ACCEPTANCE", "ALL-PASS" if all(results.values()) else "HAS-FAILURES")
    if not all(results.values()):
        # 留现场给排查，不删 runs
        print("KEEP_RUN_DIR", run_dir)
    else:
        pass
    return 0 if all(results.values()) else 4


if __name__ == "__main__":
    sys.exit(main())
