#!/usr/bin/env python3
"""Build-time PCK splitter for Godot 4.7.

Reads a main PCK and splits it into:
  core.pck       — everything except map layers, audio, and music resources
  map_layers.pck — presets/maps/layers/ only
  audio.pck      — .oggvorbisstr + data/music/ + data/sounds/ + presets/music/*.tres

Music .tres files are moved to audio.pck and their .remap/.res files are
removed from core.pck. This bypasses a Godot 4.7.1 export bug where remap
entries for music resources are misaligned (shifted by one position).

At runtime, PackManager mounts audio.pck, and DB.ReloadAudio() loads the
music .tres files directly from the mounted pack.

Usage: python3 split_pck.py <main.pck> <out_dir> <project_dir>

PCK v4 layout (Godot 4.7):
  Header (28 bytes):
    magic "GDPC" (4)
    version u32 (4)
    major u32 (4)
    minor u32 (4)
    patch u32 (4)
    flags u32 (4)
    file_count u32 (4)  [note: this field is NOT the entry count in v4]
  Alignment pad (4 bytes, zeros)
  dir_offset u32 (4)  [at offset 32]
  File data ...
  Directory at dir_offset:
    file_count u32 (4)
    entries:
      path_len u32 (4)
      path bytes (path_len)
      pad to 4-byte alignment
      offset u64 (8)
      size u64 (8)
      md5 16 bytes
      flags u32 (4)
"""

import os
import re
import struct
import sys


def read_pck_index(path):
    with open(path, 'rb') as f:
        data = f.read()
    if data[:4] != b'GDPC':
        raise ValueError(f"{path} is not a PCK file")
    ver = struct.unpack('<I', data[4:8])[0]
    major, minor, patch = struct.unpack('<III', data[8:20])
    flags = struct.unpack('<I', data[20:24])[0]
    dir_offset = struct.unpack('<I', data[32:36])[0]
    file_count = struct.unpack('<I', data[dir_offset:dir_offset+4])[0]
    print(f"  source: ver={ver} godot={major}.{minor}.{patch} flags={flags} "
          f"entries={file_count} dir_offset={dir_offset} size={len(data)}")

    entries = []
    pos = dir_offset + 4
    for _ in range(file_count):
        path_len = struct.unpack('<I', data[pos:pos+4])[0]
        pos += 4
        path_bytes = data[pos:pos+path_len]
        pos += path_len
        pos += (-path_len) % 4
        offset = struct.unpack('<Q', data[pos:pos+8])[0]
        pos += 8
        size = struct.unpack('<Q', data[pos:pos+8])[0]
        pos += 8
        md5 = data[pos:pos+16]
        pos += 16
        entry_flags = struct.unpack('<I', data[pos:pos+4])[0]
        pos += 4
        path = path_bytes.decode('utf-8', errors='replace')
        entries.append({'path': path, 'offset': offset, 'size': size})
    assert pos == len(data), f"directory parse mismatch: pos={pos} filesize={len(data)}"
    return data, entries


def is_audio(path):
    return ('.oggvorbisstr' in path
            or '/data/music/' in path
            or '/data/sounds/' in path)


def is_music_tres(path):
    """Music .tres files go to audio.pck so they can be loaded directly."""
    return path.startswith('presets/music/') and '.tres' in path


def is_music_remap(path):
    """Music .remap files are removed entirely (bypass Godot remap bug)."""
    return path.startswith('presets/music/') and '.tres.remap' in path


def is_music_res(path):
    """Music .res files are removed entirely (they're corrupted by Godot bug)."""
    if not ('.godot/exported' in path and '.res' in path):
        return False
    # Check if this .res contains a music reference
    # We'll check this in the main function where we have the data
    return False  # Placeholder - actual check done in main


def is_map_layer(path):
    return path.startswith('presets/maps/layers/') and '.tmx' not in path


def write_pck(src_data, entries, out_path):
    """Write a PCK with the given entries, copying data from src_data."""
    out = bytearray()
    out += b'GDPC'
    out += struct.pack('<I', 4)       # version
    out += struct.pack('<I', 4)       # major
    out += struct.pack('<I', 7)       # minor
    out += struct.pack('<I', 1)       # patch
    out += struct.pack('<I', 0)       # flags
    out += struct.pack('<I', len(entries))  # file_count (header field)
    out += struct.pack('<I', 0)       # alignment pad
    dir_offset_pos = len(out)
    out += struct.pack('<I', 0)       # dir_offset placeholder

    # Write file data
    offsets = []
    for e in entries:
        offsets.append(len(out))
        if 'data' in e:
            out += e['data']
        else:
            out += src_data[e['offset']:e['offset'] + e['size']]

    # Write directory (must be 4-byte aligned)
    padding_needed = (4 - (len(out) % 4)) % 4
    out += b'\x00' * padding_needed
    dir_start = len(out)
    out += struct.pack('<I', len(entries))
    for i, e in enumerate(entries):
        path_bytes = e['path'].encode('utf-8')
        path_len = len(path_bytes)
        out += struct.pack('<I', path_len)
        out += path_bytes
        out += b'\x00' * ((-path_len) % 4)
        out += struct.pack('<Q', offsets[i])
        out += struct.pack('<Q', e['size'])
        out += b'\x00' * 16          # md5 (unused)
        out += struct.pack('<I', 0)  # flags

    # Back-patch dir_offset
    struct.pack_into('<I', out, dir_offset_pos, dir_start)

    with open(out_path, 'wb') as f:
        f.write(out)
    print(f"  wrote {out_path} with {len(entries)} entries ({len(out)} bytes)")
    return len(out)


def fix_html_filesizes(web_dir, new_pck_size):
    """Update GODOT_CONFIG fileSizes in index.html to match split PCK size."""
    html_path = os.path.join(web_dir, "index.html")
    if not os.path.exists(html_path):
        print(f"  WARNING: index.html not found at {html_path}")
        return
    with open(html_path, 'r') as f:
        html = f.read()
    new_html = re.sub(r'"index\.pck":\d+', f'"index.pck":{new_pck_size}', html)
    if new_html != html:
        with open(html_path, 'w') as f:
            f.write(new_html)
        print(f"  Fixed fileSizes in index.html: index.pck = {new_pck_size}")
    else:
        print(f"  fileSizes already correct in index.html")


def bump_service_worker(web_dir):
    """Bump CACHE_VERSION to force browser cache invalidation."""
    sw_path = os.path.join(web_dir, "index.service.worker.js")
    if not os.path.exists(sw_path):
        print(f"  WARNING: index.service.worker.js not found at {sw_path}")
        return
    with open(sw_path, 'r') as f:
        sw = f.read()
    new_version = f"som_{os.path.getsize(os.path.join(web_dir, 'index.pck'))}"
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

    src_data, entries = read_pck_index(main_pck)
    print(f"  total entries: {len(entries)}")

    # Identify music .res files (corrupted by Godot bug)
    music_res_paths = set()
    for e in entries:
        path = e['path']
        if '.godot/exported' in path and path.endswith('.res'):
            content = src_data[e['offset']:e['offset'] + e['size']]
            # Music .res files contain 'data/music' and 'FileData'
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

    audio_entries = [e for e in entries if is_audio(e['path'])]
    layer_entries = [e for e in entries if is_map_layer(e['path'])]
    core_entries = [e for e in entries if not is_audio(e['path']) and not is_map_layer(e['path']) and not is_music_remap(e['path']) and e['path'] not in music_res_paths]
    removed = [e for e in entries if is_music_remap(e['path']) or e['path'] in music_res_paths]
    print(f"  audio: {len(audio_entries)}  layers: {len(layer_entries)}  core: {len(core_entries)}  removed: {len(removed)}")
    assert len(audio_entries) + len(layer_entries) + len(core_entries) + len(removed) == len(entries), \
        "classification mismatch"

    # Add music .tres files to audio pack
    audio_entries.extend(music_tres_entries)

    write_pck(src_data, audio_entries, os.path.join(out_dir, 'audio.pck'))
    write_pck(src_data, layer_entries, os.path.join(out_dir, 'map_layers.pck'))
    core_size = write_pck(src_data, core_entries, os.path.join(out_dir, 'core.pck'))

    # Fix HTML and service worker for new PCK size
    fix_html_filesizes(out_dir, core_size)
    bump_service_worker(out_dir)

    print("SPK_DONE")


if __name__ == '__main__':
    main()
