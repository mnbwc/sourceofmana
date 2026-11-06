#!/usr/bin/env python3
"""Arena leaderboard 聚合查询（读 user://arena_leaderboard.jsonl）。
用法: python3 tests/harness/leaderboard.py [--top N] [--template KEY] [--nick SUBSTR]"""
import json, os, sys, glob, argparse

def find_file():
    cands = glob.glob(os.path.expanduser("~/.local/share/SourceOfMana/arena_leaderboard.jsonl"))
    cands += glob.glob(os.path.expanduser("~/.local/share/godot/app_userdata/*/arena_leaderboard.jsonl"))
    for c in cands:
        if os.path.exists(c):
            return c
    return None

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--top", type=int, default=10)
    ap.add_argument("--template", default=None)
    ap.add_argument("--nick", default=None)
    a = ap.parse_args()
    path = find_file()
    if not path:
        print("NO_LEADERBOARD_FILE")
        return 1
    rows = []
    for line in open(path):
        line = line.strip()
        if not line: continue
        try: rows.append(json.loads(line))
        except Exception: pass
    if a.template:
        rows = [r for r in rows if r.get("template") == a.template]
    if a.nick:
        rows = [r for r in rows if a.nick in r.get("nick", "")]
    # 排序: kills desc, xpGained desc, deaths asc
    rows.sort(key=lambda r: (-(r.get("score") or {}).get("kills", 0),
                             -(r.get("score") or {}).get("xpGained", 0),
                             (r.get("score") or {}).get("deaths", 0)))
    print(f"LEADERBOARD {path} rows={len(rows)}")
    for i, r in enumerate(rows[:a.top], 1):
        s = r.get("score") or {}
        print(f"{i:3d}. {r.get('nick','?'):16s} {r.get('template','?'):18s} {r.get('status','?'):10s} "
              f"kills={s.get('kills',0):3d} deaths={s.get('deaths',0):2d} xp={s.get('xpGained',0):5d} "
              f"loot={s.get('itemsLooted',0):3d} ep={r.get('episodeId','?')}")
    return 0

if __name__ == "__main__":
    sys.exit(main())
