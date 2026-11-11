#!/usr/bin/env python3
"""Split a Godot 4.7 PCK v4 into core.pck + audio.pck + maps.pck.

  core.pck   — everything needed at startup (login, character screen,
               starting maps Drazil + Tulimshar, sound effects, UI, fonts)
  audio.pck  — music .ogg files + .tres resources
  maps.pck   — all other map scenes, textures, and remap files

Usage: split_pck.py <main.pck> <out_dir> <project_dir>
"""
import os
import sys
import re
import struct
import hashlib

PCK_MAGIC = b'GDPC'
PCK_VERSION = 4
GODOT_MAJOR = 4
GODOT_MINOR = 7
GODOT_PATCH = 1
PACK_FLAGS = 2  # PACK_REL_FILEBASE

# Maps that must be in core.pck:
#   Drazil   — character selection screen
#   Tulimshar — starting map for new players
CORE_MAPS = {"Drazil", "Tulimshar"}

# Map name aliases for texture matching (some textures have slightly different names)
MAP_TEXTURE_ALIASES = {
    "Tulimshar": ["Tulimshar"],
    "Drazil": ["Drazil"],
}


def _get_pad(alignment, n):
    rest = n % alignment
    return 0 if rest == 0 else alignment - rest


def read_pck_index(path):
    with open(path, 'rb') as f:
        data = f.read()
    if data[:4] != b'GDPC':
        raise ValueError(f"{path} is not a PCK file")
    ver = struct.unpack('<I', data[4:8])[0]
    major, minor, patch = struct.unpack('<III', data[8:20])
    flags = struct.unpack('<I', data[20:24])[0]
    file_base_ofs = struct.unpack('<Q', data[24:32])[0]
    dir_base_ofs = struct.unpack('<Q', data[32:40])[0]
    file_count = struct.unpack('<I', data[dir_base_ofs:dir_base_ofs+4])[0]
    print(f"  source: ver={ver} godot={major}.{minor}.{patch} flags={flags} "
          f"entries={file_count} file_base={file_base_ofs} dir_base={dir_base_ofs} "
          f"size={len(data)}")

    entries = []
    pos = dir_base_ofs + 4
    for _ in range(file_count):
        path_len = struct.unpack('<I', data[pos:pos+4])[0]
        pos += 4
        path_bytes = data[pos:pos+path_len]
        pos += path_len
        offset = struct.unpack('<Q', data[pos:pos+8])[0]
        pos += 8
        size = struct.unpack('<Q', data[pos:pos+8])[0]
        pos += 8
        md5 = data[pos:pos+16]
        pos += 16
        entry_flags = struct.unpack('<I', data[pos:pos+4])[0]
        pos += 4
        path = path_bytes.decode('utf-8', errors='replace').rstrip('\x00')
        entries.append({
            'path': path,
            'offset': offset,  # relative to file_base_ofs
            'size': size,
            'md5': md5,
            'flags': entry_flags,
        })
    assert pos == len(data), f"PCK directory size mismatch: {pos} vs {len(data)}"
    return data, entries, file_base_ofs


def get_exported_name(path):
    """Extract the resource name from a .godot/exported/ path.

    Examples:
      .godot/exported/133200997/export-46ae8d1cb606d2e4a9365c91f922bfab-Tulimshar.scn -> Tulimshar
      .godot/exported/133200997/export-6ee3dff6ff4f8a81029ba67dd3820b24-Overworld.res -> Overworld
    """
    filename = path.split('/')[-1]
    if not filename.startswith('export-'):
        return None
    # Remove "export-" prefix and extension
    name_part = filename[7:]  # after "export-"
    # Remove extension (.scn, .res, etc.)
    if '.' in name_part:
        name_part = name_part.rsplit('.', 1)[0]
    # The hash is 32 chars, then a dash, then the name
    if len(name_part) > 33 and name_part[32] == '-':
        return name_part[33:]
    return None


def is_map_scene(path, content=None):
    """Check if a .scn file is a map scene (contains TileMapLayer)."""
    if not path.startswith('.godot/exported/') or not path.endswith('.scn'):
        return False
    if content is not None:
        return b'TileMapLayer' in content
    return False


def is_map_data_res(path, content=None):
    """Check if a .res file is map data (contains MapData)."""
    if not path.startswith('.godot/exported/') or not path.endswith('.res'):
        return False
    if content is not None:
        return b'MapData' in content
    return False


def is_map_server_res(path, content=None):
    """Check if a .res file is map server data (contains MapServerData)."""
    if not path.startswith('.godot/exported/') or not path.endswith('.res'):
        return False
    if content is not None:
        return b'MapServerData' in content
    return False


def is_map_navigation_res(path, content=None):
    """Check if a .res file is map navigation (contains NavigationPolygon)."""
    if not path.startswith('.godot/exported/') or not path.endswith('.res'):
        return False
    if content is not None:
        return b'NavigationPolygon' in content
    return False


def is_map_related(path):
    """Check if a path is map-related (any map file type)."""
    if not path.startswith('.godot/exported/'):
        return False
    if not (path.endswith('.scn') or path.endswith('.res')):
        return False
    return True


def is_map_texture(path):
    """Check if a .ctex file is a map texture (name matches a map name)."""
    if not path.startswith('.godot/imported/') or not path.endswith('.ctex'):
        return False
    # Extract the original filename from the .ctex path
    # e.g., .godot/imported/Tulimshar.png-8ea0021570398ed7c32a68875ad938d6.ctex -> Tulimshar
    filename = path.split('/')[-1]
    if not filename.endswith('.ctex'):
        return False
    name_part = filename[:-5]  # Remove .ctex
    # The hash is 32 chars after the last dash
    if '-' in name_part:
        parts = name_part.rsplit('-', 1)
        if len(parts[1]) == 32:
            name_part = parts[0]
    # Remove .png extension if present
    if name_part.endswith('.png'):
        name_part = name_part[:-4]
    # Check if this matches any known map name
    # We'll do this check later when we have the full list of map names
    return name_part


def is_map_remap(path):
    """Check if a .remap file is map-related."""
    return path.startswith('presets/maps/') and path.endswith('.remap')


def get_map_name_from_path(path):
    """Extract map name from any map-related path."""
    # For .godot/exported/ paths
    if path.startswith('.godot/exported/'):
        return get_exported_name(path)
    # For presets/maps/ paths
    if path.startswith('presets/maps/'):
        parts = path.split('/')
        # Find the .tres filename
        for part in reversed(parts):
            if part.endswith('.tres') or part.endswith('.tres.remap'):
                return part.replace('.tres.remap', '').replace('.tres', '')
    return None


def is_core_map_name(map_name):
    """Check if a map name is in CORE_MAPS."""
    return map_name in CORE_MAPS if map_name else False


def is_audio(path):
    if '.oggvorbisstr' in path:
        filename = path.split('/')[-1]
        if filename.endswith('.oggvorbisstr'):
            filename = filename[:-13]
        if '-' in filename:
            parts = filename.rsplit('-', 1)
            if len(parts[1]) == 32:
                filename = parts[0]
        return ' - ' in filename
    if path.startswith('data/music/') and path.endswith('.import'):
        return True
    return False


def is_music_tres(path):
    """Music .tres files go to audio.pck so they can be loaded directly."""
    return path.startswith('presets/music/') and '.tres' in path


def is_music_remap(path):
    """Music .remap files are removed entirely (bypass Godot remap bug)."""
    return path.startswith('presets/music/') and '.tres.remap' in path


def is_map_layer(path):
    return path.startswith('presets/maps/layers/') and '.tmx' not in path


def write_pck(src_data, src_file_base, entries, out_path):
    """Write a PCK with the given entries, copying data from src_data.

    Godot 4.7 PCK v4 format:
    - Header: magic + version + major + minor + patch + flags + file_count + pad + dir_offset
    - File data
    - Directory: file_count + entries
    - Each entry: path_len (includes padding) + path (with padding) + offset + size + md5 + flags
    """
    out = bytearray()

    # --- Header (104 bytes) ---
    out += b'GDPC'
    out += struct.pack('<I', PCK_VERSION)
    out += struct.pack('<I', GODOT_MAJOR)
    out += struct.pack('<I', GODOT_MINOR)
    out += struct.pack('<I', GODOT_PATCH)
    out += struct.pack('<I', PACK_FLAGS)
    file_base_ofs_pos = len(out)
    out += struct.pack('<Q', 0)  # file_base_ofs placeholder
    dir_base_ofs_pos = len(out)
    out += struct.pack('<Q', 0)  # dir_base_ofs placeholder
    out += b'\x00' * 64           # reserved

    # Align to 16 bytes
    pad = _get_pad(16, len(out))
    out += b'\x00' * pad
    file_base = len(out)
    struct.pack_into('<Q', out, file_base_ofs_pos, file_base)

    # --- File data ---
    rel_offsets = []
    for e in entries:
        rel_offsets.append(len(out) - file_base)
        if 'data' in e:
            out += e['data']
        else:
            abs_offset = src_file_base + e['offset']
            out += src_data[abs_offset:abs_offset + e['size']]
        # Align each file to 16 bytes
        pad = _get_pad(16, len(out))
        out += b'\x00' * pad

    # Align to 16 bytes before directory
    pad = _get_pad(16, len(out))
    out += b'\x00' * pad

    # --- Directory ---
    dir_start = len(out)
    struct.pack_into('<Q', out, dir_base_ofs_pos, dir_start)
    out += struct.pack('<I', len(entries))
    for i, e in enumerate(entries):
        path_bytes = e['path'].encode('utf-8')
        pad_len = _get_pad(4, len(path_bytes))
        path_with_padding = path_bytes + b'\x00' * pad_len
        path_len = len(path_with_padding)
        out += struct.pack('<I', path_len)
        out += path_with_padding
        out += struct.pack('<Q', rel_offsets[i])
        out += struct.pack('<Q', e['size'])
        # MD5: use existing if available, otherwise compute
        if 'md5' in e and e['md5'] != b'\x00' * 16:
            out += e['md5']
        else:
            file_data = out[file_base + rel_offsets[i]:file_base + rel_offsets[i] + e['size']]
            out += hashlib.md5(file_data).digest()
        out += struct.pack('<I', e.get('flags', 0))

    with open(out_path, 'wb') as f:
        f.write(out)
    print(f"  wrote {out_path} with {len(entries)} entries ({len(out)} bytes)")
    return len(out)


def fix_html_filesizes(web_dir, new_pck_size, main_pack="core.pck"):
    """Update GODOT_CONFIG fileSizes in index.html to match split PCK size.

    Replaces the ``fileSizes`` entry for the main pack and ensures the
    ``mainPack`` key points at the split pack (core.pck).
    """
    html_path = os.path.join(web_dir, "index.html")
    if not os.path.exists(html_path):
        print(f"  WARNING: index.html not found at {html_path}")
        return
    with open(html_path, 'r') as f:
        html = f.read()
    new_html = html
    # Replace fileSizes for the main pack (handle both index.pck and core.pck)
    new_html = re.sub(r'"(?:index|core)\.pck":\d+',
                      f'"{main_pack}":{new_pck_size}', new_html)
    # Ensure mainPack is set
    if '"mainPack"' in new_html:
        new_html = re.sub(r'"mainPack":"[^"]*"',
                          f'"mainPack":"{main_pack}"', new_html)
    else:
        new_html = re.sub(r'("godotPoolSize":\d+)',
                          rf'\1,"mainPack":"{main_pack}"', new_html)
    if new_html != html:
        with open(html_path, 'w') as f:
            f.write(new_html)
        print(f"  Fixed index.html: {main_pack} = {new_pck_size}, mainPack = {main_pack}")
    else:
        print(f"  index.html already correct")


def bump_service_worker(web_dir):
    """Bump CACHE_VERSION to force browser cache invalidation."""
    sw_path = os.path.join(web_dir, "index.service.worker.js")
    if not os.path.exists(sw_path):
        print(f"  (no service worker at {sw_path}, skipping cache bump)")
        return
    core_pck = os.path.join(web_dir, 'core.pck')
    pck_size = os.path.getsize(core_pck) if os.path.exists(core_pck) else 0
    new_version = f"som_{pck_size}"
    with open(sw_path, 'r') as f:
        sw = f.read()
    new_sw = re.sub(r"CACHE_VERSION = '[^']*'", f"CACHE_VERSION = '{new_version}'", sw)
    if new_sw != sw:
        with open(sw_path, 'w') as f:
            f.write(new_sw)
        print(f"  Bumped CACHE_VERSION to {new_version}")
    else:
        print(f"  CACHE_VERSION already correct")


def main():
    if len(sys.argv) < 4:
        print("usage: split_pck.py <main.pck> <out_dir> <project_dir>")
        sys.exit(1)
    main_pck = sys.argv[1]
    out_dir = sys.argv[2]
    project_dir = sys.argv[3]
    os.makedirs(out_dir, exist_ok=True)

    src_data, entries, src_file_base = read_pck_index(main_pck)
    print(f"  total entries: {len(entries)}")

    # First pass: identify all map-related files and their map names
    map_files = {}  # path -> map_name
    map_names = set()

    for e in entries:
        path = e['path']
        if path.startswith('.godot/exported/') and (path.endswith('.scn') or path.endswith('.res')):
            abs_offset = src_file_base + e['offset']
            content = src_data[abs_offset:abs_offset + e['size']]
            name = get_exported_name(path)
            if name:
                # Check if this is a map file by content
                if is_map_scene(path, content) or is_map_data_res(path, content) or \
                   is_map_server_res(path, content) or is_map_navigation_res(path, content):
                    map_files[path] = name
                    map_names.add(name)

    print(f"  found {len(map_files)} map-related files for {len(map_names)} maps")

    # Identify map textures by matching names
    map_textures = {}  # path -> map_name
    for e in entries:
        path = e['path']
        if path.startswith('.godot/imported/') and path.endswith('.ctex'):
            tex_name = is_map_texture(path)
            if tex_name:
                # Check if this texture name matches any map name
                for map_name in map_names:
                    if tex_name == map_name or tex_name.startswith(map_name):
                        map_textures[path] = map_name
                        break

    print(f"  found {len(map_textures)} map textures")

    # Identify map remap files
    map_remaps = {}  # path -> map_name
    for e in entries:
        path = e['path']
        if is_map_remap(path):
            name = get_map_name_from_path(path)
            if name:
                map_remaps[path] = name

    print(f"  found {len(map_remaps)} map remap files")

    # Classify map files into core and other
    core_map_files = {}  # path -> map_name
    other_map_files = {}  # path -> map_name

    for path, name in map_files.items():
        if is_core_map_name(name):
            core_map_files[path] = name
        else:
            other_map_files[path] = name

    core_map_textures = {}
    other_map_textures = {}
    for path, name in map_textures.items():
        if is_core_map_name(name):
            core_map_textures[path] = name
        else:
            other_map_textures[path] = name

    core_map_remaps = {}
    other_map_remaps = {}
    for path, name in map_remaps.items():
        if is_core_map_name(name):
            core_map_remaps[path] = name
        else:
            other_map_remaps[path] = name

    print(f"  core maps: {len(core_map_files)} files, {len(core_map_textures)} textures, {len(core_map_remaps)} remaps")
    print(f"  other maps: {len(other_map_files)} files, {len(other_map_textures)} textures, {len(other_map_remaps)} remaps")

    # Identify music .res files (corrupted by Godot bug)
    music_res_paths = set()
    for e in entries:
        path = e['path']
        if '.godot/exported' in path and path.endswith('.res'):
            abs_offset = src_file_base + e['offset']
            content = src_data[abs_offset:abs_offset + e['size']]
            if b'data/music' in content and b'FileData' in content:
                music_res_paths.add(path)

    # Read music .tres files from project
    music_tres_entries = []
    music_dir = os.path.join(project_dir, 'presets', 'music')
    if os.path.isdir(music_dir):
        for fname in os.listdir(music_dir):
            if fname.endswith('.tres'):
                fpath = os.path.join(music_dir, fname)
                with open(fpath, 'rb') as f:
                    content = f.read()
                music_tres_entries.append({
                    'path': f'presets/music/{fname}',
                    'size': len(content),
                    'data': content
                })
                print(f"  added music .tres: {fname} ({len(content)} bytes)")

    # Build file sets
    other_map_paths = set(other_map_files.keys()) | set(other_map_textures.keys()) | set(other_map_remaps.keys())
    core_map_paths = set(core_map_files.keys()) | set(core_map_textures.keys()) | set(core_map_remaps.keys())

    # Audio entries
    audio_entries = [e for e in entries if is_audio(e['path'])]

    # Core entries: everything except audio, other map files, and music remaps
    core_entries = []
    for e in entries:
        path = e['path']
        if is_audio(path):
            continue
        if is_music_remap(path):
            continue
        if path in music_res_paths:
            continue
        if path in other_map_paths:
            continue
        core_entries.append(e)

    # Maps entries: other map files + their textures + remaps
    maps_entries = []
    seen = set()
    for e in entries:
        path = e['path']
        if path in other_map_paths and path not in seen:
            seen.add(path)
            maps_entries.append(e)

    # Removed entries
    removed = [e for e in entries if is_music_remap(e['path']) or e['path'] in music_res_paths]

    print(f"\n  audio: {len(audio_entries)}  core: {len(core_entries)}  maps: {len(maps_entries)}  removed: {len(removed)}")

    # Add music .tres files to audio pack
    audio_entries.extend(music_tres_entries)

    # Write packs
    write_pck(src_data, src_file_base, audio_entries, os.path.join(out_dir, 'audio.pck'))
    maps_size = write_pck(src_data, src_file_base, maps_entries, os.path.join(out_dir, 'maps.pck'))
    core_size = write_pck(src_data, src_file_base, core_entries, os.path.join(out_dir, 'core.pck'))

    # Fix HTML and service worker for new PCK size
    fix_html_filesizes(out_dir, core_size, 'core.pck')
    bump_service_worker(out_dir)

    print(f"\n  core.pck: {core_size/1024/1024:.1f} MB")
    print(f"  maps.pck: {maps_size/1024/1024:.1f} MB")
    print(f"  audio.pck: {sum(e['size'] for e in audio_entries)/1024/1024:.1f} MB")
    print("SPK_DONE")


if __name__ == '__main__':
    main()
