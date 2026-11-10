#!/usr/bin/env python3
"""
Extract NPC dialogue from GDScript files.
Looks for Mes(), Chat(), Narrate(), InfoChoice(), Choice() calls with string literals.
"""

import json
import os
import re
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parent.parent

def collect_files(dir_path: Path, extension: str) -> list:
    result = []
    for root, dirs, files in os.walk(dir_path):
        for f in files:
            if f.endswith(extension):
                result.append(Path(root) / f)
    return sorted(result)

def extract_dialogue(path: Path) -> dict:
    with open(path, 'r', encoding='utf-8') as f:
        content = f.read()
    
    dialogue = []
    
    # Pattern: Mes("text") or Mes("text", callable)
    # Pattern: Chat("text")
    # Pattern: Narrate("text")
    # Pattern: InfoChoice("text", callable) or InfoChoice("text")
    # Pattern: Choice("text", callable) or Choice("text")
    
    # Find all string literals that are arguments to dialogue functions
    # We'll extract the function name and the string content
    
    # Pattern for function calls with string arguments
    patterns = [
        (r'\bMes\s*\(\s*"([^"]+)"', 'Mes'),
        (r'\bChat\s*\(\s*"([^"]+)"', 'Chat'),
        (r'\bNarrate\s*\(\s*"([^"]+)"', 'Narrate'),
        (r'\bInfoChoice\s*\(\s*"([^"]+)"', 'InfoChoice'),
        (r'\bChoice\s*\(\s*"([^"]+)"', 'Choice'),
    ]
    
    for pattern, func_name in patterns:
        for match in re.finditer(pattern, content):
            text = match.group(1)
            # Get line number
            line_num = content[:match.start()].count('\n') + 1
            dialogue.append({
                "function": func_name,
                "text": text,
                "line": line_num,
            })
    
    return dialogue

def main():
    out_dir = PROJECT_ROOT / "tools" / "export"
    out_dir.mkdir(parents=True, exist_ok=True)
    
    scripts_dir = PROJECT_ROOT / "sources" / "scripts"
    files = collect_files(scripts_dir, ".gd")
    
    all_dialogue = {}
    total_lines = 0
    
    for path in files:
        dialogue = extract_dialogue(path)
        if dialogue:
            rel_path = str(path.relative_to(scripts_dir))
            all_dialogue[rel_path] = dialogue
            total_lines += len(dialogue)
            print(f"  {rel_path}: {len(dialogue)} dialogue lines")
    
    print(f"\nTotal: {total_lines} dialogue lines across {len(all_dialogue)} scripts")
    
    with open(out_dir / "npc_dialogue.json", 'w', encoding='utf-8') as f:
        json.dump(all_dialogue, f, indent=2, ensure_ascii=False)
    print(f"Saved: {out_dir / 'npc_dialogue.json'}")

if __name__ == "__main__":
    main()
