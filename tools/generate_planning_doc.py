#!/usr/bin/env python3
"""
Generate a comprehensive game planning document from exported JSON data.
"""

import json
from pathlib import Path
from collections import defaultdict

PROJECT_ROOT = Path(__file__).resolve().parent.parent
EXPORT_DIR = PROJECT_ROOT / "tools" / "export"

# Enum mappings
SLOT_NAMES = {
    -1: "None", 0: "CHEST", 1: "LEGS", 2: "FEET", 3: "HANDS",
    4: "HEAD", 5: "NECK", 6: "WEAPON", 7: "SHIELD", 8: "BODY",
    9: "FACE", 10: "HAIR", 11: "QUEST",
}

SKILL_CATEGORY = {0: "SPELL", 1: "PHYSICAL", 2: "ABILITY"}
SKILL_MODE = {0: "SINGLE", 1: "ZONE", 2: "SELF"}

AI_BEHAVIOUR_FLAGS = {
    0: "NONE", 1: "PACIFIST", 2: "NEUTRAL", 4: "AGGRESSIVE",
    8: "IMMOBILE", 16: "FOLLOWER", 32: "LEADER", 64: "SPAWNER",
    128: "STEAL", 256: "FLEE",
}

def decode_behaviour(flags: int) -> list:
    result = []
    for bit, name in AI_BEHAVIOUR_FLAGS.items():
        if bit == 0:
            continue
        if flags & bit:
            result.append(name)
    return result if result else ["NONE"]

def load_json(filename: str):
    with open(EXPORT_DIR / filename, 'r', encoding='utf-8') as f:
        return json.load(f)

def generate_document():
    quests = load_json("quests.json")
    items = load_json("items.json")
    skills = load_json("skills.json")
    entities = load_json("entities.json")
    dialogue = load_json("npc_dialogue.json")
    
    lines = []
    
    # Header
    lines.append("# Source of Mana — 游戏策划文档")
    lines.append("")
    lines.append("> 本文档由自动化工具从游戏资源文件中提取生成")
    lines.append(">")
    lines.append(f"> 生成时间: 2025-11-10")
    lines.append("")
    lines.append("---")
    lines.append("")
    
    # Table of Contents
    lines.append("## 目录")
    lines.append("")
    lines.append("1. [游戏任务数据](#1-游戏任务数据)")
    lines.append("2. [游戏物品数据](#2-游戏物品数据)")
    lines.append("3. [游戏技能数据](#3-游戏技能数据")
    lines.append("4. [游戏实体数据](#4-游戏实体数据)")
    lines.append("5. [游戏掉率数据](#5-游戏掉率数据)")
    lines.append("6. [游戏状态机](#6-游戏状态机)")
    lines.append("7. [NPC 对话数据](#7-npc-对话数据)")
    lines.append("8. [附录](#8-附录)")
    lines.append("")
    lines.append("---")
    lines.append("")
    
    # ============================================================
    # 1. QUESTS
    # ============================================================
    lines.append("## 1. 游戏任务数据")
    lines.append("")
    lines.append(f"共 **{len(quests)}** 个任务")
    lines.append("")
    
    # Quest summary table
    lines.append("| # | 任务名称 | ID | 给予者 | 给予位置 | 目标 | 目标位置 | 奖励 |")
    lines.append("|---|---------|-----|--------|---------|------|---------|------|")
    for i, q in enumerate(quests, 1):
        name = q["name"] or "(未命名)"
        lines.append(f"| {i} | {name} | {q['id']} | {q['giver']} | {q['giverLocation']} | {q['target']} | {q['targetLocation']} | {q['reward']} |")
    lines.append("")
    
    # Quest details
    lines.append("### 任务详情")
    lines.append("")
    for q in quests:
        lines.append(f"#### {q['name']}")
        lines.append("")
        lines.append(f"- **ID**: `{q['id']}`")
        lines.append(f"- **描述**: {q['description'] or '(无)'}")
        lines.append(f"- **给予者**: {q['giver'] or '(无)'} @ {q['giverLocation'] or '(无)'}")
        lines.append(f"- **目标**: {q['target'] or '(无)'} @ {q['targetLocation'] or '(无)'}")
        lines.append(f"- **奖励**: {q['reward'] or '(无)'}")
        lines.append("")
    
    lines.append("---")
    lines.append("")
    
    # ============================================================
    # 2. ITEMS
    # ============================================================
    lines.append("## 2. 游戏物品数据")
    lines.append("")
    lines.append(f"共 **{len(items)}** 个物品")
    lines.append("")
    
    # Group by category
    by_category = defaultdict(list)
    for item in items:
        by_category[item["category"]].append(item)
    
    # Summary table
    lines.append("| # | 名称 | 类别 | 槽位 | 购买价格 | 出售价格 | 重量 | 可堆叠 | 可使用 | 修饰符 |")
    lines.append("|---|------|------|------|---------|---------|------|--------|--------|--------|")
    for i, item in enumerate(items, 1):
        slot = SLOT_NAMES.get(item["slot"], str(item["slot"]))
        mods = ", ".join(f"{k}={v}" for k, v in item.get("modifiers", {}).items()) if item.get("modifiers") else "-"
        lines.append(f"| {i} | {item['name']} | {item['category']} | {slot} | {item['buyPrice']} | {item['sellPrice']} | {item['weight']} | {'✓' if item['stackable'] else '✗'} | {'✓' if item['usable'] else '✗'} | {mods} |")
    lines.append("")
    
    # Category breakdown
    lines.append("### 按类别统计")
    lines.append("")
    for cat, cat_items in sorted(by_category.items()):
        lines.append(f"- **{cat}**: {len(cat_items)} 个物品")
    lines.append("")
    
    # Item details
    lines.append("### 物品详情")
    lines.append("")
    for item in items:
        lines.append(f"#### {item['name']}")
        lines.append("")
        lines.append(f"- **文件**: `{item['file']}.tres`")
        lines.append(f"- **ID**: `{item['id']}`")
        lines.append(f"- **类别**: {item['category']}")
        lines.append(f"- **槽位**: {SLOT_NAMES.get(item['slot'], str(item['slot']))}")
        lines.append(f"- **购买价格**: {item['buyPrice']} GP")
        lines.append(f"- **出售价格**: {item['sellPrice']} GP")
        lines.append(f"- **重量**: {item['weight']}")
        lines.append(f"- **可堆叠**: {'是' if item['stackable'] else '否'}")
        lines.append(f"- **可使用**: {'是' if item['usable'] else '否'}")
        if item.get("modifiers"):
            lines.append(f"- **修饰符**:")
            for mod_name, mod_val in item["modifiers"].items():
                lines.append(f"  - {mod_name}: {mod_val}")
        lines.append("")
    
    lines.append("---")
    lines.append("")
    
    # ============================================================
    # 3. SKILLS
    # ============================================================
    lines.append("## 3. 游戏技能数据")
    lines.append("")
    lines.append(f"共 **{len(skills)}** 个技能")
    lines.append("")
    
    lines.append("| # | 名称 | 类别 | 模式 | 范围 | 冷却时间 | 施法时间 | 技能时间 | 重复 | 修饰符 |")
    lines.append("|---|------|------|------|------|---------|---------|---------|------|--------|")
    for i, s in enumerate(skills, 1):
        cat = SKILL_CATEGORY.get(s["category"], str(s["category"]))
        mode = SKILL_MODE.get(s["mode"], str(s["mode"]))
        mods = ", ".join(f"{k}={v}" for k, v in s.get("modifiers", {}).items()) if s.get("modifiers") else "-"
        lines.append(f"| {i} | {s['name']} | {cat} | {mode} | {s['skillRange']} | {s['cooldownTime']} | {s['castTime']} | {s['skillTime']} | {'✓' if s['repeat'] else '✗'} | {mods} |")
    lines.append("")
    
    # Skill details
    lines.append("### 技能详情")
    lines.append("")
    for s in skills:
        lines.append(f"#### {s['name']}")
        lines.append("")
        lines.append(f"- **文件**: `{s['file']}.tres`")
        lines.append(f"- **ID**: `{s['id']}`")
        lines.append(f"- **描述**: {s['description'] or '(无)'}")
        lines.append(f"- **类别**: {SKILL_CATEGORY.get(s['category'], str(s['category']))}")
        lines.append(f"- **模式**: {SKILL_MODE.get(s['mode'], str(s['mode']))}")
        lines.append(f"- **范围**: {s['skillRange']}")
        lines.append(f"- **冷却时间**: {s['cooldownTime']}s")
        lines.append(f"- **施法时间**: {s['castTime']}s")
        lines.append(f"- **技能时间**: {s['skillTime']}s")
        lines.append(f"- **Buff时间**: {s['buffTime']}s")
        lines.append(f"- **重复**: {'是' if s['repeat'] else '否'}")
        if s.get("modifiers"):
            lines.append(f"- **修饰符**:")
            for mod_name, mod_val in s["modifiers"].items():
                lines.append(f"  - {mod_name}: {mod_val}")
        lines.append("")
    
    lines.append("---")
    lines.append("")
    
    # ============================================================
    # 4. ENTITIES
    # ============================================================
    lines.append("## 4. 游戏实体数据")
    lines.append("")
    lines.append(f"共 **{len(entities)}** 个实体")
    lines.append("")
    
    # Summary table
    lines.append("| # | 名称 | ID | 行为 | Boss | 父实体 | 等级 | 生命值 | 攻击力 | 防御力 |")
    lines.append("|---|------|-----|------|------|--------|------|--------|--------|--------|")
    for i, e in enumerate(entities, 1):
        behaviours = ", ".join(decode_behaviour(e["behaviour"]))
        stats = e.get("stats", {})
        level = stats.get("level", "-")
        hp = stats.get("maxHealth", "-")
        atk = stats.get("attack", "-")
        defense = stats.get("defense", "-")
        boss = "✓" if e.get("isBoss") else "✗"
        parent = e.get("parent", "-")
        lines.append(f"| {i} | {e['name']} | {e['id']} | {behaviours} | {boss} | {parent} | {level} | {hp} | {atk} | {defense} |")
    lines.append("")
    
    # Entity details
    lines.append("### 实体详情")
    lines.append("")
    for e in entities:
        lines.append(f"#### {e['name']}")
        lines.append("")
        lines.append(f"- **文件**: `{e['file']}.tres`")
        lines.append(f"- **ID**: `{e['id']}`")
        lines.append(f"- **行为**: {', '.join(decode_behaviour(e['behaviour']))}")
        lines.append(f"- **Boss**: {'是' if e.get('isBoss') else '否'}")
        if e.get("parent"):
            lines.append(f"- **父实体**: {e['parent']}")
        lines.append(f"- **半径**: {e['radius']}")
        
        if e.get("stats"):
            lines.append(f"- **属性**:")
            for stat_name, stat_val in e["stats"].items():
                lines.append(f"  - {stat_name}: {stat_val}")
        
        if e.get("equipment"):
            lines.append(f"- **装备**:")
            for i, eq in enumerate(e["equipment"]):
                slot_name = SLOT_NAMES.get(i, f"slot_{i}")
                if eq:
                    lines.append(f"  - {slot_name}: {eq}")
        
        if e.get("skills"):
            lines.append(f"- **技能**:")
            for sk in e["skills"]:
                lines.append(f"  - {sk['skill']} (概率: {sk['probability']})")
        
        lines.append("")
    
    lines.append("---")
    lines.append("")
    
    # ============================================================
    # 5. DROPS
    # ============================================================
    lines.append("## 5. 游戏掉率数据")
    lines.append("")
    
    entities_with_drops = [e for e in entities if e.get("drops")]
    lines.append(f"共 **{len(entities_with_drops)}** 个实体有掉落物")
    lines.append("")
    
    lines.append("| 实体 | 掉落物品 | 掉率 |")
    lines.append("|------|---------|------|")
    for e in entities_with_drops:
        for drop in e["drops"]:
            chance_pct = f"{drop['chance']*100:.1f}%" if isinstance(drop['chance'], (int, float)) else str(drop['chance'])
            lines.append(f"| {e['name']} | {drop['item']} | {chance_pct} |")
    lines.append("")
    
    # Spawn data
    entities_with_spawns = [e for e in entities if e.get("spawns")]
    if entities_with_spawns:
        lines.append("### 实体生成数据")
        lines.append("")
        lines.append("| 实体 | 生成实体 | 数量 |")
        lines.append("|------|---------|------|")
        for e in entities_with_spawns:
            for spawn in e["spawns"]:
                lines.append(f"| {e['name']} | {spawn['entity']} | {spawn['count']} |")
        lines.append("")
    
    lines.append("---")
    lines.append("")
    
    # ============================================================
    # 6. STATE MACHINES
    # ============================================================
    lines.append("## 6. 游戏状态机")
    lines.append("")
    
    # Load state machine data
    with open(EXPORT_DIR / "state_machines.json", 'r', encoding='utf-8') as f:
        state_machines = json.load(f)
    
    for class_name, enums in state_machines.items():
        lines.append(f"### {class_name}")
        lines.append("")
        for enum_name, values in enums.items():
            lines.append(f"**{enum_name}**:")
            lines.append("")
            lines.append("| 名称 | 值 |")
            lines.append("|------|-----|")
            for name, val in values.items():
                lines.append(f"| {name} | {val} |")
            lines.append("")
    
    # Load modifiers
    with open(EXPORT_DIR / "modifiers.json", 'r', encoding='utf-8') as f:
        modifiers = json.load(f)
    
    lines.append("### CellCommons.Modifier (修饰符)")
    lines.append("")
    lines.append("| 名称 | 值 |")
    lines.append("|------|-----|")
    for mod in modifiers:
        lines.append(f"| {mod['name']} | {mod.get('value', '')} |")
    lines.append("")
    
    lines.append("---")
    lines.append("")
    
    # ============================================================
    # 7. NPC DIALOGUE
    # ============================================================
    lines.append("## 7. NPC 对话数据")
    lines.append("")
    
    total_lines = sum(len(d) for d in dialogue.values())
    lines.append(f"共 **{total_lines}** 条对话，分布在 **{len(dialogue)}** 个 NPC 脚本中")
    lines.append("")
    
    # Summary table
    lines.append("| NPC 脚本 | 对话行数 |")
    lines.append("|---------|---------|")
    for script_path, dialogues in sorted(dialogue.items(), key=lambda x: -len(x[1])):
        lines.append(f"| {script_path} | {len(dialogues)} |")
    lines.append("")
    
    # Detailed dialogue
    lines.append("### 对话详情")
    lines.append("")
    for script_path, dialogues in sorted(dialogue.items()):
        npc_name = Path(script_path).stem
        lines.append(f"#### {npc_name}")
        lines.append("")
        lines.append(f"*来源: `{script_path}`*")
        lines.append("")
        for d in dialogues:
            func_label = {
                "Mes": "💬 Mes",
                "Chat": "💭 Chat",
                "Narrate": "📖 Narrate",
                "InfoChoice": "ℹ️ InfoChoice",
                "Choice": "🔹 Choice",
            }.get(d["function"], d["function"])
            lines.append(f"- {func_label} (L{d['line']}): {d['text']}")
        lines.append("")
    
    lines.append("---")
    lines.append("")
    
    # ============================================================
    # 8. APPENDIX
    # ============================================================
    lines.append("## 8. 附录")
    lines.append("")
    
    lines.append("### A. 装备槽位 (ActorCommons.Slot)")
    lines.append("")
    lines.append("| 值 | 名称 |")
    lines.append("|-----|------|")
    for val, name in SLOT_NAMES.items():
        lines.append(f"| {val} | {name} |")
    lines.append("")
    
    lines.append("### B. 技能类别 (SkillCell.Category)")
    lines.append("")
    lines.append("| 值 | 名称 |")
    lines.append("|-----|------|")
    for val, name in SKILL_CATEGORY.items():
        lines.append(f"| {val} | {name} |")
    lines.append("")
    
    lines.append("### C. 技能模式 (Skill.TargetMode)")
    lines.append("")
    lines.append("| 值 | 名称 |")
    lines.append("|-----|------|")
    for val, name in SKILL_MODE.items():
        lines.append(f"| {val} | {name} |")
    lines.append("")
    
    lines.append("### D. AI 行为标志 (AICommons.Behaviour)")
    lines.append("")
    lines.append("| 值 | 名称 |")
    lines.append("|-----|------|")
    for val, name in AI_BEHAVIOUR_FLAGS.items():
        lines.append(f"| {val} | {name} |")
    lines.append("")
    
    lines.append("### E. 任务进度常量 (ProgressCommons)")
    lines.append("")
    lines.append("```")
    lines.append("UnknownProgress = 0")
    lines.append("CompletedProgress = 255")
    lines.append("```")
    lines.append("")
    
    lines.append("---")
    lines.append("")
    lines.append("*文档结束*")
    
    # Write to file
    output_path = PROJECT_ROOT / "docs" / "game_planning_document.md"
    output_path.parent.mkdir(parents=True, exist_ok=True)
    with open(output_path, 'w', encoding='utf-8') as f:
        f.write('\n'.join(lines))
    
    print(f"Document generated: {output_path}")
    print(f"Total lines: {len(lines)}")

if __name__ == "__main__":
    generate_document()
