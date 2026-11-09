package main

// SceneMultiplayer packet parsing — the wire format produced by
// SceneRPCInterface::_send_rpc (modules/multiplayer/scene_rpc_interface.cpp).
//
// Packet layout:
//
//	[1B meta]
//	  bits 0..2  command type (NETWORK_COMMAND_*)
//	  bits 4..5  node ID compression (0=8bit, 1=16bit, 2=32bit)
//	  bit  6     name ID compression (0=8bit, 1=16bit)
//	  bit  7     byte-only-or-no-args flag
//	[node ID: 1/2/4 bytes]
//	  If the 32-bit form has bit 31 set, the low 31 bits are an offset into
//	  the packet where a NUL-terminated node path is stored (uncached node).
//	[name ID: 1/2 bytes]
//	[arg count: 1 byte, only when bit 7 is clear]
//	[variants: encode_and_compress_variant for each arg]

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// SceneMultiplayer command types (scene_multiplayer.h).
const (
	CmdRemoteCall   = 0
	CmdSimplifyPath = 1
	CmdConfirmPath  = 2
	CmdRaw          = 3
	CmdSpawn        = 4
	CmdDespawn      = 5
	CmdSync         = 6
	CmdSys          = 7
)

// ParsedRPC is a decoded remote-call packet.
type ParsedRPC struct {
	NodeID   uint32
	NodePath string // populated when the packet carried a full path
	NameID   uint16
	Args     []Variant
}

// ParsePacket decodes one SceneMultiplayer packet. It returns the command
// type and, for CmdRemoteCall, the decoded RPC. For CmdSimplifyPath it
// returns the cache ID and path so the caller can maintain the node cache.
func ParsePacket(buf []byte) (cmd int, rpc *ParsedRPC, pathCacheID uint32, path string, err error) {
	if len(buf) < 1 {
		return 0, nil, 0, "", errors.New("packet too small")
	}
	meta := buf[0]
	cmd = int(meta & 0x07)
	nidComp := int((meta >> 4) & 0x03)
	nameComp := int((meta >> 6) & 0x01)
	byteOnly := (meta >> 7) & 0x01

	ofs := 1
	var nodeID uint32
	switch nidComp {
	case 0:
		if len(buf) < ofs+1 {
			return 0, nil, 0, "", errors.New("packet too small for 8-bit node ID")
		}
		nodeID = uint32(buf[ofs])
		ofs++
	case 1:
		if len(buf) < ofs+2 {
			return 0, nil, 0, "", errors.New("packet too small for 16-bit node ID")
		}
		nodeID = uint32(binary.LittleEndian.Uint16(buf[ofs:]))
		ofs += 2
	case 2:
		if len(buf) < ofs+4 {
			return 0, nil, 0, "", errors.New("packet too small for 32-bit node ID")
		}
		nodeID = binary.LittleEndian.Uint32(buf[ofs:])
		ofs += 4
	default:
		return 0, nil, 0, "", fmt.Errorf("unknown node ID compression %d", nidComp)
	}

	// A 32-bit node ID with bit 31 set encodes a path offset instead of a
	// cache ID. The path is a NUL-terminated string at that offset.
	nodePath := ""
	if nidComp == 2 && nodeID&0x80000000 != 0 {
		pathOfs := int(nodeID & 0x7FFFFFFF)
		if pathOfs >= len(buf) {
			return 0, nil, 0, "", fmt.Errorf("path offset %d out of range (len %d)", pathOfs, len(buf))
		}
		end := pathOfs
		for end < len(buf) && buf[end] != 0 {
			end++
		}
		nodePath = string(buf[pathOfs:end])
	}

	var nameID uint16
	if nameComp == 0 {
		if len(buf) < ofs+1 {
			return 0, nil, 0, "", errors.New("packet too small for 8-bit name ID")
		}
		nameID = uint16(buf[ofs])
		ofs++
	} else {
		if len(buf) < ofs+2 {
			return 0, nil, 0, "", errors.New("packet too small for 16-bit name ID")
		}
		nameID = binary.LittleEndian.Uint16(buf[ofs:])
		ofs += 2
	}

	switch cmd {
	case CmdRemoteCall:
		argCount := 0
		if byteOnly == 0 {
			if len(buf) < ofs+1 {
				return 0, nil, 0, "", errors.New("packet too small for arg count")
			}
			argCount = int(buf[ofs])
			ofs++
		}
		args, _, _, err := DecodeRPCVariants(buf[ofs:], argCount)
		if err != nil {
			return cmd, nil, 0, "", fmt.Errorf("decode args: %w", err)
		}
		return cmd, &ParsedRPC{
			NodeID:   nodeID,
			NodePath: nodePath,
			NameID:   nameID,
			Args:     args,
		}, 0, "", nil
	case CmdSimplifyPath:
		// [cmd][methods_md5 32B + NUL][cache_id 4B][path NUL-terminated]
		// This command has no node/name ID fields, so parse from byte 1.
		const md5FieldLen = 33 // 32 bytes + NUL
		if len(buf) < 1+md5FieldLen+4 {
			return cmd, nil, 0, "", errors.New("packet too small for SIMPLIFY_PATH")
		}
		pathCacheID = binary.LittleEndian.Uint32(buf[1+md5FieldLen:])
		end := 1 + md5FieldLen + 4
		for end < len(buf) && buf[end] != 0 {
			end++
		}
		path = string(buf[1+md5FieldLen+4 : end])
		return cmd, nil, pathCacheID, path, nil
	default:
		return cmd, nil, 0, "", nil
	}
}

// BuildRPCPacket constructs a SceneMultiplayer remote-call packet targeting a
// node by cache ID. Used by the Go server to push state updates to clients.
func BuildRPCPacket(cacheID uint32, nameID uint16, args []Variant) ([]byte, error) {
	payload, raw, err := EncodeRPCVariants(args)
	if err != nil {
		return nil, err
	}

	// Determine node ID compression: 8-bit if cacheID <= 255, 16-bit if
	// <= 65535, else 32-bit.
	var nidComp int
	var nidBytes []byte
	switch {
	case cacheID <= 0xFF:
		nidComp = 0
		nidBytes = []byte{byte(cacheID)}
	case cacheID <= 0xFFFF:
		nidComp = 1
		nidBytes = make([]byte, 2)
		binary.LittleEndian.PutUint16(nidBytes, uint16(cacheID))
	default:
		nidComp = 2
		nidBytes = make([]byte, 4)
		binary.LittleEndian.PutUint32(nidBytes, cacheID)
	}

	// Name ID compression: 8-bit if <= 255, else 16-bit.
	var nameComp int
	var nameBytes []byte
	if nameID <= 0xFF {
		nameComp = 0
		nameBytes = []byte{byte(nameID)}
	} else {
		nameComp = 1
		nameBytes = make([]byte, 2)
		binary.LittleEndian.PutUint16(nameBytes, nameID)
	}

	// Arg count: omit when byteOnly is set (single PackedByteArray) or when
	// there are no args.
	byteOnly := raw || len(args) == 0

	meta := byte(CmdRemoteCall) |
		byte(nidComp<<4) |
		byte(nameComp<<6)
	if byteOnly {
		meta |= 1 << 7
	}

	out := make([]byte, 0, 1+len(nidBytes)+len(nameBytes)+1+len(payload))
	out = append(out, meta)
	out = append(out, nidBytes...)
	out = append(out, nameBytes...)
	if !byteOnly {
		out = append(out, byte(len(args)))
	}
	out = append(out, payload...)
	return out, nil
}

// BuildSimplifyPathPacket constructs a NETWORK_COMMAND_SIMPLIFY_PATH packet so
// the Go server can teach clients about node cache IDs.
func BuildSimplifyPathPacket(cacheID uint32, methodsMD5 string, nodePath string) []byte {
	out := make([]byte, 0, 1+33+4+len(nodePath)+1)
	out = append(out, byte(CmdSimplifyPath))
	// 32-byte MD5 + NUL
	md5 := make([]byte, 33)
	copy(md5, methodsMD5)
	out = append(out, md5...)
	idBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(idBytes, cacheID)
	out = append(out, idBytes...)
	out = append(out, []byte(nodePath)...)
	out = append(out, 0)
	return out
}

// BuildConfirmPathPacket constructs a NETWORK_COMMAND_CONFIRM_PATH packet.
func BuildConfirmPathPacket(cacheID uint32, valid bool) []byte {
	out := make([]byte, 6)
	out[0] = byte(CmdConfirmPath)
	if valid {
		out[1] = 1
	}
	binary.LittleEndian.PutUint32(out[2:], cacheID)
	return out
}
