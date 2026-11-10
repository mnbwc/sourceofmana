#!/usr/bin/env python3
"""
Generate a translation worklist for missing translations.
"""

import json
import csv
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parent.parent
EXPORT_DIR = PROJECT_ROOT / "tools" / "export"

def load_translations():
    translations = {}
    for csv_file in ['ITEMS.csv', 'NPC.csv', 'QUEST.csv', 'UI-full.csv', 'UI.csv']:
        with open(PROJECT_ROOT / 'data' / 'db' / 'zh' / csv_file, 'r', encoding='utf-8') as f:
            reader = csv.reader(f)
            next(reader)
            for row in reader:
                if len(row) >= 3:
                    key = row[0].strip().strip('"')
                    en = row[1].strip().strip('"')
                    zh = row[2].strip().strip('"')
                    translations[key] = (en, zh)
    return translations

def main():
    translations = load_translations()
    
    # 1. Untranslated NPC dialogue
    dialogue = json.load(open(EXPORT_DIR / 'npc_dialogue.json'))
    untranslated = {}
    for script, lines in dialogue.items():
        for d in lines:
            text = d['text']
            if any('\u4e00' <= c <= '\u9fff' for c in text):
                continue
            if text in translations:
                continue
            if text not in untranslated:
                untranslated[text] = []
            untranslated[text].append(f"{script}:{d['line']}")
    
    # 2. Skill descriptions
    skills = json.load(open(EXPORT_DIR / 'skills.json'))
    skill_descs = []
    for s in skills:
        desc = s.get('description', '')
        if desc and desc not in translations:
            skill_descs.append((s['name'], desc))
    
    # 3. Check for any other missing translations
    # Load all CSVs and check for empty zh values
    empty_zh = []
    for csv_file in ['ITEMS.csv', 'NPC.csv', 'QUEST.csv', 'UI-full.csv', 'UI.csv']:
        with open(PROJECT_ROOT / 'data' / 'db' / 'zh' / csv_file, 'r', encoding='utf-8') as f:
            reader = csv.reader(f)
            next(reader)
            for row in reader:
                if len(row) >= 3:
                    key = row[0].strip().strip('"')
                    en = row[1].strip().strip('"')
                    zh = row[2].strip().strip('"')
                    if not zh or zh == en:
                        empty_zh.append((csv_file, key, en, zh))
    
    # Generate worklist
    lines = []
    lines.append("# Source of Mana — 翻译工作表")
    lines.append("")
    lines.append(f"生成时间: 2025-11-10")
    lines.append("")
    lines.append("---")
    lines.append("")
    
    # Section 1: NPC dialogue
    lines.append("## 1. 未翻译的 NPC 对话")
    lines.append("")
    lines.append(f"共 **{len(untranslated)}** 条")
    lines.append("")
    lines.append("| # | 英文文本 | 出现位置 |")
    lines.append("|---|---------|---------|")
    for i, (text, locations) in enumerate(sorted(untranslated.items()), 1):
        loc_str = ", ".join(locations[:3])
        if len(locations) > 3:
            loc_str += f" ... (+{len(locations)-3})"
        lines.append(f"| {i} | {text} | {loc_str} |")
    lines.append("")
    
    # Section 2: Skill descriptions
    lines.append("## 2. 未翻译的技能描述")
    lines.append("")
    lines.append(f"共 **{len(skill_descs)}** 条")
    lines.append("")
    lines.append("| # | 技能名称 | 英文描述 |")
    lines.append("|---|---------|---------|")
    for i, (name, desc) in enumerate(skill_descs, 1):
        lines.append(f"| {i} | {name} | {desc} |")
    lines.append("")
    
    # Section 3: Empty or untranslated entries in CSVs
    lines.append("## 3. CSV 中空的或未翻译的条目")
    lines.append("")
    lines.append(f"共 **{len(empty_zh)}** 条")
    lines.append("")
    lines.append("| 文件 | Key | 英文 | 中文 |")
    lines.append("|------|-----|------|------|")
    for csv_file, key, en, zh in empty_zh:
        lines.append(f"| {csv_file} | {key} | {en} | {zh} |")
    lines.append("")
    
    lines.append("---")
    lines.append("")
    lines.append("*工作表结束*")
    
    output_path = PROJECT_ROOT / "docs" / "translation_worklist.md"
    with open(output_path, 'w', encoding='utf-8') as f:
        f.write('\n'.join(lines))
    
    print(f"Translation worklist generated: {output_path}")
    print(f"Untranslated dialogue: {len(untranslated)}")
    print(f"Untranslated skill descriptions: {len(skill_descs)}")
    print(f"Empty/untranslated CSV entries: {len(empty_zh)}")

if __name__ == "__main__":
    main()
