#!/usr/bin/env python3
"""
Parse Godot .tres files and export game data to JSON.
Handles ext_resource, sub_resource, dictionaries, arrays, and typed dictionaries.
"""

import json
import os
import re
import sys
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parent.parent

def collect_files(dir_path: Path, extension: str) -> list:
    result = []
    for root, dirs, files in os.walk(dir_path):
        for f in files:
            if f.endswith(extension):
                result.append(Path(root) / f)
    return sorted(result)

def parse_tres(path: Path) -> dict:
    with open(path, 'r', encoding='utf-8') as f:
        content = f.read()
    
    result = {
        "properties": {},
        "sub_resources": {},
        "ext_resources": {},
    }
    
    lines = content.split('\n')
    current_section = ""
    current_sub_id = ""
    in_dict = False
    dict_key = ""
    dict_content = ""
    dict_start_pattern = None  # Track the prefix pattern to know how to close
    
    i = 0
    while i < len(lines):
        line = lines[i].strip()
        i += 1
        
        # ext_resource
        if line.startswith('[ext_resource'):
            id_match = re.search(r'\sid="([^"]+)"', line)
            path_match = re.search(r'path="([^"]+)"', line)
            if id_match and path_match:
                result["ext_resources"][id_match.group(1)] = path_match.group(1)
            continue
        
        # sub_resource
        if line.startswith('[sub_resource'):
            current_section = "sub_resource"
            id_match = re.search(r'id="?([^"\]]+)"?\]?$', line)
            if id_match:
                current_sub_id = id_match.group(1).strip().strip('"')
                result["sub_resources"][current_sub_id] = {}
            continue
        
        # resource
        if line.startswith('[resource]'):
            current_section = "resource"
            continue
        
        # other section headers
        if line.startswith('[') and not line.startswith('[ext_resource') and not line.startswith('[sub_resource') and not line.startswith('[resource'):
            current_section = ""
            continue
        
        if not line or line.startswith('#'):
            continue
        
        # Check for dictionary start
        if not in_dict:
            # Pattern 1: _stats = {
            # Pattern 2: _drops = Dictionary[...]({
            # Pattern 3: _stateSFX = Dictionary[int, AudioStream]({
            dict_match = re.match(r'^(\w+)\s*=\s*(.*)$', line)
            if dict_match and '{' in line and not line.endswith('}'):
                dict_key = dict_match.group(1)
                value_part = dict_match.group(2).strip()
                in_dict = True
                dict_content = value_part
                # Determine the closing pattern
                if re.match(r'^Dictionary\[.*\]\(\{', value_part) or re.match(r'^Array\[.*\]\(\{$', value_part):
                    dict_start_pattern = 'typed_dict'
                elif value_part.startswith('{'):
                    dict_start_pattern = 'plain_dict'
                else:
                    dict_start_pattern = None
                continue
            
            # Single-line dictionary: _stats = {"level": 1}
            single_dict_match = re.match(r'^(\w+)\s*=\s*\{(.*)\}\s*$', line)
            if single_dict_match:
                key = single_dict_match.group(1)
                dict_text = single_dict_match.group(2)
                parsed = parse_dict_content(dict_text)
                store_property(result, current_section, current_sub_id, key, parsed)
                continue
            
            # Array: _equipment = Array[...]([null, null, ...])
            array_match = re.match(r'^(\w+)\s*=\s*Array\[.*?\]\(\[(.*)\]\)$', line)
            if array_match:
                key = array_match.group(1)
                array_text = array_match.group(2)
                parsed = parse_array_content(array_text)
                store_property(result, current_section, current_sub_id, key, parsed)
                continue
            
            # Simple key = value
            simple_match = re.match(r'^(\w+)\s*=\s*(.*)$', line)
            if simple_match:
                key = simple_match.group(1)
                value = simple_match.group(2).strip()
                parsed = parse_simple_value(value)
                store_property(result, current_section, current_sub_id, key, parsed)
        
        if in_dict:
            dict_content += "\n" + line
            # Check if dict is complete
            if dict_start_pattern == 'typed_dict':
                if '})' in line or (line.endswith('}') and dict_content.count('{') <= dict_content.count('}')):
                    # Extract content between ({ and })
                    inner = extract_typed_dict_content(dict_content)
                    parsed = parse_dict_content(inner)
                    store_property(result, current_section, current_sub_id, dict_key, parsed)
                    in_dict = False
            elif dict_start_pattern == 'plain_dict':
                if line.endswith('}'):
                    inner = dict_content.strip()
                    if inner.startswith('{'):
                        inner = inner[1:]
                    if inner.endswith('}'):
                        inner = inner[:-1]
                    parsed = parse_dict_content(inner)
                    store_property(result, current_section, current_sub_id, dict_key, parsed)
                    in_dict = False
            continue
    
    return result

def extract_typed_dict_content(content: str) -> str:
    """Extract content from Dictionary[...]({ ... }) format"""
    # Find the ({ ... }) pattern
    start = content.find('({')
    if start != -1:
        end = content.rfind('})')
        if end != -1:
            return content[start+2:end]
    # Fallback: find first { and last }
    start = content.find('{')
    end = content.rfind('}')
    if start != -1 and end != -1:
        return content[start+1:end]
    return ""

def parse_dict_content(content: str) -> dict:
    result = {}
    if not content.strip():
        return result
    
    # Split by commas at depth 0
    parts = smart_split(content, ',')
    for part in parts:
        part = part.strip()
        if not part:
            continue
        # Find the colon separator (not inside quotes or parentheses)
        colon_idx = find_colon(part)
        if colon_idx == -1:
            continue
        key_text = part[:colon_idx].strip()
        value_text = part[colon_idx+1:].strip()
        
        key = parse_dict_key(key_text)
        value = parse_dict_value(value_text)
        result[str(key)] = value
    
    return result

def find_colon(text: str) -> int:
    """Find the colon separator, ignoring colons inside quotes or parentheses"""
    depth = 0
    in_quote = False
    for i, ch in enumerate(text):
        if ch == '"' and (i == 0 or text[i-1] != '\\'):
            in_quote = not in_quote
        elif not in_quote:
            if ch in '([{':
                depth += 1
            elif ch in ')]}':
                depth -= 1
            elif ch == ':' and depth == 0:
                return i
    return -1

def parse_dict_key(key_text: str):
    key_text = key_text.strip()
    if key_text.startswith('ExtResource('):
        inner = key_text[12:-1].strip().strip('"')
        return f"ext:{inner}"
    if key_text.startswith('"') and key_text.endswith('"'):
        return key_text[1:-1]
    if key_text.startswith("'") and key_text.endswith("'"):
        return key_text[1:-1]
    try:
        return int(key_text)
    except ValueError:
        pass
    return key_text

def parse_dict_value(value_text: str):
    value_text = value_text.strip()
    if value_text.startswith('ExtResource('):
        inner = value_text[12:-1].strip().strip('"')
        return f"ext:{inner}"
    if value_text.startswith('SubResource('):
        inner = value_text[12:-1].strip().strip('"')
        return f"sub:{inner}"
    if value_text == 'null':
        return None
    if value_text == 'true':
        return True
    if value_text == 'false':
        return False
    if value_text.startswith('"') and value_text.endswith('"'):
        return value_text[1:-1]
    try:
        if '.' in value_text:
            return float(value_text)
        return int(value_text)
    except ValueError:
        pass
    return value_text

def parse_array_content(content: str) -> list:
    result = []
    if not content.strip():
        return result
    parts = smart_split(content, ',')
    for part in parts:
        part = part.strip()
        if not part:
            continue
        result.append(parse_simple_value(part))
    return result

def parse_simple_value(value: str):
    value = value.strip()
    if value.startswith('ExtResource(') and value.endswith(')'):
        inner = value[12:-1].strip().strip('"')
        return f"ext:{inner}"
    if value.startswith('SubResource(') and value.endswith(')'):
        inner = value[12:-1].strip().strip('"')
        return f"sub:{inner}"
    if value == 'null':
        return None
    if value == 'true':
        return True
    if value == 'false':
        return False
    if value.startswith('Vector2(') and value.endswith(')'):
        inner = value[8:-1]
        parts = inner.split(',')
        return [float(parts[0]), float(parts[1])]
    if value.startswith('"') and value.endswith('"'):
        return value[1:-1]
    try:
        if '.' in value:
            return float(value)
        return int(value)
    except ValueError:
        pass
    return value

def smart_split(text: str, delimiter: str) -> list:
    result = []
    depth = 0
    in_quote = False
    current = ""
    for i, ch in enumerate(text):
        if ch == '"' and (i == 0 or text[i-1] != '\\'):
            in_quote = not in_quote
        elif not in_quote:
            if ch in '([{':
                depth += 1
            elif ch in ')]}':
                depth -= 1
        if ch == delimiter and depth == 0 and not in_quote:
            if current.strip():
                result.append(current.strip())
            current = ""
        else:
            current += ch
    if current.strip():
        result.append(current.strip())
    return result

def store_property(result: dict, section: str, sub_id: str, key: str, value):
    if section == "resource":
        result["properties"][key] = value
    elif section == "sub_resource":
        if sub_id in result["sub_resources"]:
            result["sub_resources"][sub_id][key] = value

def resolve_ext_name(ext_id: str, data: dict) -> str:
    ext_res = data.get("ext_resources", {})
    if ext_id in ext_res:
        path = ext_res[ext_id]
        return Path(path).stem
    return ext_id

def resolve_ext_path(ext_id: str, data: dict) -> str:
    ext_res = data.get("ext_resources", {})
    if ext_id in ext_res:
        return ext_res[ext_id]
    return ext_id

def clean_value(val, data: dict):
    if isinstance(val, dict):
        return {str(k): clean_value(v, data) for k, v in val.items()}
    if isinstance(val, list):
        return [clean_value(v, data) for v in val]
    if isinstance(val, str) and val.startswith('ext:'):
        return resolve_ext_name(val[4:], data)
    return val

def clean_drops(drops: dict, data: dict) -> list:
    result = []
    if isinstance(drops, dict):
        for key, val in drops.items():
            item_name = resolve_ext_name(key[4:], data) if isinstance(key, str) and key.startswith('ext:') else str(key)
            result.append({"item": item_name, "chance": clean_value(val, data)})
    return result

def clean_spawns(spawns: dict, data: dict) -> list:
    result = []
    if isinstance(spawns, dict):
        for key, val in spawns.items():
            entity_name = resolve_ext_name(key[4:], data) if isinstance(key, str) and key.startswith('ext:') else str(key)
            result.append({"entity": entity_name, "count": clean_value(val, data)})
    return result

def clean_skills(skills: dict, data: dict) -> list:
    result = []
    if isinstance(skills, dict):
        for key, val in skills.items():
            skill_name = resolve_ext_name(key[4:], data) if isinstance(key, str) and key.startswith('ext:') else str(key)
            result.append({"skill": skill_name, "probability": clean_value(val, data)})
    return result

def clean_equipment(equipment: list, data: dict) -> list:
    result = []
    if isinstance(equipment, list):
        for val in equipment:
            if val is None:
                result.append(None)
            elif isinstance(val, str) and val.startswith('ext:'):
                result.append(resolve_ext_name(val[4:], data))
            else:
                result.append(clean_value(val, data))
    return result

def extract_modifiers(data: dict, props: dict) -> dict:
    result = {}
    if "modifiers" not in props:
        return result
    mod_val = props["modifiers"]
    
    sub_id = None
    if isinstance(mod_val, str) and mod_val.startswith('sub:'):
        sub_id = mod_val[4:]
    
    if sub_id and sub_id in data.get("sub_resources", {}):
        sub_res = data["sub_resources"][sub_id]
        if "_modifiers" in sub_res:
            mod_list = sub_res["_modifiers"]
            if isinstance(mod_list, list):
                for mod_ref in mod_list:
                    if isinstance(mod_ref, str) and mod_ref.startswith('sub:'):
                        mod_sub_id = mod_ref[4:]
                        if mod_sub_id in data["sub_resources"]:
                            mod_data = data["sub_resources"][mod_sub_id]
                            effect = mod_data.get("_effect", 0)
                            value = mod_data.get("_value", 0)
                            effect_name = get_modifier_name(effect)
                            if effect_name in result:
                                result[effect_name] = result[effect_name] + value
                            else:
                                result[effect_name] = value
    return result

MODIFIER_NAMES = {
    0: "None", 1: "Health", 2: "Mana", 3: "Stamina",
    4: "MaxMana", 5: "RegenMana", 6: "CritRate",
    7: "MAttack", 8: "MDefense", 9: "MaxStamina",
    10: "RegenStamina", 11: "CooldownDelay",
    12: "MaxHealth", 13: "RegenHealth", 14: "Defense",
    15: "CastDelay", 16: "DodgeRate", 17: "AttackRange",
    18: "WalkSpeed", 19: "WeightCapacity", 20: "Attack",
    21: "Hide", 22: "Invisible", 23: "Accuracy",
}

def get_modifier_name(effect: int) -> str:
    return MODIFIER_NAMES.get(effect, f"Unknown({effect})")

# ============================================================
# EXPORT FUNCTIONS
# ============================================================

def export_quests(out_dir: Path):
    print("\n=== EXPORTING QUESTS ===")
    files = collect_files(PROJECT_ROOT / "presets" / "quests", ".tres")
    quests = []
    for path in files:
        data = parse_tres(path)
        props = data["properties"]
        q = {
            "file": path.stem,
            "id": props.get("id", 0),
            "name": props.get("name", ""),
            "description": props.get("description", ""),
            "giver": props.get("giver", ""),
            "giverLocation": props.get("giverLocation", ""),
            "target": props.get("target", ""),
            "targetLocation": props.get("targetLocation", ""),
            "reward": props.get("reward", ""),
        }
        quests.append(q)
        print(f"  {q['name']} (id={q['id']})")
    quests.sort(key=lambda x: x["name"])
    with open(out_dir / "quests.json", 'w', encoding='utf-8') as f:
        json.dump(quests, f, indent=2, ensure_ascii=False)
    print(f"  Saved: {out_dir / 'quests.json'}")

def export_items(out_dir: Path):
    print("\n=== EXPORTING ITEMS ===")
    files = collect_files(PROJECT_ROOT / "presets" / "cells" / "items", ".tres")
    items = []
    for path in files:
        data = parse_tres(path)
        props = data["properties"]
        category = path.parent.name
        item = {
            "file": path.stem,
            "category": category,
            "id": props.get("id", 0),
            "name": props.get("name", ""),
            "slot": props.get("slot", -1),
            "buyPrice": props.get("buyPrice", 0),
            "sellPrice": props.get("sellPrice", 0),
            "weight": props.get("weight", 0.0),
            "stackable": props.get("stackable", False),
            "usable": props.get("usable", False),
            "type": props.get("type", 0),
            "customfield": props.get("customfield", ""),
        }
        mods = extract_modifiers(data, props)
        if mods:
            item["modifiers"] = mods
        items.append(item)
        print(f"  {item['name']} ({category}, id={item['id']})")
    items.sort(key=lambda x: x["name"])
    with open(out_dir / "items.json", 'w', encoding='utf-8') as f:
        json.dump(items, f, indent=2, ensure_ascii=False)
    print(f"  Saved: {out_dir / 'items.json'}")

def export_skills(out_dir: Path):
    print("\n=== EXPORTING SKILLS ===")
    files = collect_files(PROJECT_ROOT / "presets" / "cells" / "skills", ".tres")
    skills = []
    for path in files:
        data = parse_tres(path)
        props = data["properties"]
        skill = {
            "file": path.stem,
            "id": props.get("id", 0),
            "name": props.get("name", ""),
            "description": props.get("description", ""),
            "category": props.get("category", 0),
            "state": props.get("state", 0),
            "skillRange": props.get("skillRange", 0),
            "mode": props.get("mode", 0),
            "repeat": props.get("repeat", False),
            "cooldownTime": props.get("cooldownTime", 0.0),
            "castTime": props.get("castTime", 0.0),
            "skillTime": props.get("skillTime", 0.0),
            "buffTime": props.get("buffTime", 0.0),
        }
        mods = extract_modifiers(data, props)
        if mods:
            skill["modifiers"] = mods
        skills.append(skill)
        print(f"  {skill['name']} (category={skill['category']}, id={skill['id']})")
    skills.sort(key=lambda x: x["name"])
    with open(out_dir / "skills.json", 'w', encoding='utf-8') as f:
        json.dump(skills, f, indent=2, ensure_ascii=False)
    print(f"  Saved: {out_dir / 'skills.json'}")

def export_entities(out_dir: Path):
    print("\n=== EXPORTING ENTITIES ===")
    files = collect_files(PROJECT_ROOT / "presets" / "entities", ".tres")
    entities = []
    for path in files:
        data = parse_tres(path)
        props = data["properties"]
        entity = {
            "file": path.stem,
            "id": props.get("_id", 0),
            "name": props.get("_name", ""),
            "radius": props.get("_radius", 0),
            "behaviour": props.get("_behaviour", 0),
            "questID": props.get("_questID", 0),
            "questState": props.get("_questState", 0),
            "questStateMax": props.get("_questStateMax", 0),
            "isBoss": props.get("_isBoss", False),
        }
        if "_parent" in props:
            parent_val = props["_parent"]
            if isinstance(parent_val, str) and parent_val.startswith('ext:'):
                entity["parent"] = resolve_ext_name(parent_val[4:], data)
            else:
                entity["parent"] = str(parent_val)
        if "_stats" in props:
            entity["stats"] = clean_value(props["_stats"], data)
        if "_drops" in props:
            entity["drops"] = clean_drops(props["_drops"], data)
        if "_spawns" in props:
            entity["spawns"] = clean_spawns(props["_spawns"], data)
        if "_skills" in props:
            entity["skills"] = clean_skills(props["_skills"], data)
        if "_equipment" in props:
            entity["equipment"] = clean_equipment(props["_equipment"], data)
        entities.append(entity)
        print(f"  {entity['name']} (id={entity['id']}, behaviour={entity['behaviour']})")
    entities.sort(key=lambda x: x["name"])
    with open(out_dir / "entities.json", 'w', encoding='utf-8') as f:
        json.dump(entities, f, indent=2, ensure_ascii=False)
    print(f"  Saved: {out_dir / 'entities.json'}")

def main():
    out_dir = PROJECT_ROOT / "tools" / "export"
    out_dir.mkdir(parents=True, exist_ok=True)
    
    export_quests(out_dir)
    export_items(out_dir)
    export_skills(out_dir)
    export_entities(out_dir)
    
    print("\nALL EXPORTS COMPLETE")

if __name__ == "__main__":
    main()
