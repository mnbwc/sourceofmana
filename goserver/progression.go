package main

import (
	"fmt"
	"log"
	"time"
)

// Phase-5 progression: level/experience, buffs, camera/viewport sync,
// sit, and password management.
//
// Wire format mirrors sources/network/server/Server.gd:
//   LevelUp=44, UpdateSkill=116, UpdateBuff=111, EntityBuff=36,
//   CameraLookAt=4, CameraReset=5, SetViewportSize=78, TriggerSit=106,
//   ChangePassword=7, RequestPasswordReset=64, ConfirmPasswordReset=18,
//   RequestOnlineList=63.

const (
	rpcCameraLookAt         = 4
	rpcCameraReset          = 5
	rpcChangePassword       = 7
	rpcConfirmPasswordReset = 18
	rpcEntityBuff           = 36
	rpcRequestOnlineList    = 63
	rpcRequestPasswordReset = 64
	rpcSetViewportSize      = 78
	rpcTriggerSit           = 106
	rpcUpdateBuff           = 111
)

// experienceTable mirrors Experience.gd _experienceTable. Index = current
// level; value = experience needed to reach the next level. The table has 135
// entries (levels 1..135); the last entry is math.MaxInt32 (MAX_LEVEL_REACHED).
var experienceTable = []int64{
	9, 16, 25, 36, 77, 112, 153, 200,
	253, 320, 385, 490, 585, 700, 830, 970,
	1120, 1260, 1420, 1620, 1860, 1990, 2240, 2504,
	2950, 3426, 3934, 4474, 6889, 7995, 9174, 10425,
	11748, 13967, 15775, 17678, 19677, 21773, 30543, 34212,
	38065, 42102, 46323, 53026, 58419, 64041, 69892, 75973,
	102468, 115254, 128692, 142784, 157528, 178184, 196300, 215198,
	234879, 255341, 330188, 365914, 403224, 442116, 482590, 536948,
	585191, 635278, 687211, 740988, 925400, 1473746, 1594058, 1718928,
	1848355, 1982340, 2230113, 2386162, 2547417, 2713878, 3206160, 3681024,
	4022472, 4377024, 4744680, 5125440, 5767272, 6204000, 6655464, 7121664,
	7602600, 9738720, 11649960, 13643520, 18339300, 23836800, 35658000, 48687000,
	58135000, 99999999, 103000000, 107000000, 112000000, 116000000, 121000000, 125000000,
	130000000, 134000000, 139000000, 145000000, 152200000, 160840000, 171200000, 191930000,
	202290000, 214720000, 229640000, 247550000, 283370000, 301280000, 322770000, 348560000,
	379500000, 417450000, 459195000, 505114500, 555625950, 622301064, 696977191, 780614454,
	880533104, 993241342, 1120376234, 1263784392, 1425548794, 1608019039, 2147483647,
}

// maxLevelReached is returned when the level exceeds the table.
const maxLevelReached = int64(0)

// neededExperience mirrors Experience.GetNeededExperienceForNextLevel.
// SoM levels are 0-based internally (Stats.level starts at 0); the table is
// indexed directly by level, so level 0 needs 9 XP, level 1 needs 16, etc.
func neededExperience(level int64) int64 {
	if level < 0 || int(level) >= len(experienceTable) {
		return maxLevelReached
	}
	return experienceTable[level]
}

// grantExperience adds experience to a character, handling level-ups.
// Returns the number of levels gained.
func grantExperience(chr *Character, value int64) int {
	if value <= 0 {
		return 0
	}
	chr.Experience += value
	levels := 0
	for {
		needed := neededExperience(chr.Level)
		if needed == maxLevelReached || chr.Experience < needed {
			break
		}
		chr.Experience -= needed
		chr.Level++
		levels++
	}
	return levels
}

// handleRequestOnlineList implements Server.RequestOnlineList(peerID).
func (s *Server) handleRequestOnlineList(c *Client, _ *ParsedRPC) {
	names := make([]Variant, 0, len(s.world.snapshot()))
	for _, e := range s.world.snapshot() {
		names = append(names, NewString(e.Name))
	}
	sendToClient(c, nil, rpcRefreshOnlineList, []Variant{
		{Type: VariantPackedStringArray, Arr: names},
	})
}

// handleSetViewportSize implements Server.SetViewportSize(halfWidth, halfHeight, peerID).
// SoM uses this to sync the camera's viewport half-extents; we accept and log it.
func (s *Server) handleSetViewportSize(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 2 {
		return
	}
	log.Printf("camera: peer %d viewport halfsize=(%.0f, %.0f)",
		c.peerID, rpc.Args[0].Float, rpc.Args[1].Float)
}

// handleCameraLookAt implements Network.CameraLookAt(pos, peerID) — a push from
// the server to a specific client. We treat it as a no-op log (the client sets
// its camera position locally).
func (s *Server) handleCameraLookAt(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 1 {
		return
	}
	pos := rpc.Args[0]
	log.Printf("camera: peer %d look at (%.0f, %.0f)", c.peerID, pos.Vec2[0], pos.Vec2[1])
}

// handleCameraReset implements Network.CameraReset(peerID).
func (s *Server) handleCameraReset(c *Client, _ *ParsedRPC) {
	log.Printf("camera: peer %d reset", c.peerID)
}

// handleTriggerSit implements Server.TriggerSit(peerID).
// SoM: player.state = SIT (2) or back to IDLE (0).
func (s *Server) handleTriggerSit(c *Client, _ *ParsedRPC) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	if e.State == 2 { // SIT -> IDLE
		e.State = 0
	} else {
		e.State = 2 // IDLE -> SIT
	}
	s.broadcastEntityState(e)
	log.Printf("sit: peer %d state=%d", c.peerID, e.State)
}

// broadcastEntityState pushes the entity's state to AOI neighbours.
func (s *Server) broadcastEntityState(e *Entity) {
	args := updateEntityArgs(e)
	for _, other := range s.world.snapshot() {
		if other.CharID == e.CharID {
			continue
		}
		if other.Pos.dist(e.Pos) > aoiRadius {
			continue
		}
		if oc := s.clientByPeer(other.PeerID); oc != nil {
			sendToClient(oc, nil, rpcUpdateEntity, args)
		}
	}
}

// handleChangePassword implements Server.ChangePassword(current, new, peerID).
func (s *Server) handleChangePassword(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 2 {
		return
	}
	current := rpc.Args[0].Str
	newPass := rpc.Args[1].Str
	c.session.mu.Lock()
	acc := c.session.account
	c.session.mu.Unlock()
	if acc == nil {
		log.Printf("password: peer %d change failed (no account)", c.peerID)
		return
	}
	if acc.Password != current {
		log.Printf("password: peer %d change failed (wrong current)", c.peerID)
		return
	}
	s.store.mu.Lock()
	acc.Password = newPass
	s.store.saveLocked()
	s.store.mu.Unlock()
	log.Printf("password: peer %d changed password", c.peerID)
}

// handleRequestPasswordReset implements Server.RequestPasswordReset(accountName, peerID).
// SoM sends an email with a verification code; we generate a server-side code
// and log it (in a real deployment this would be emailed).
func (s *Server) handleRequestPasswordReset(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 1 {
		return
	}
	accName := rpc.Args[0].Str
	// Generate a 6-digit verification code.
	code := fmt.Sprintf("%06d", s.randInt(1000000))
	// Store it with a 15-minute expiry.
	s.resetCodes[accName] = resetCode{Code: code, Expires: time.Now().Add(15 * time.Minute)}
	log.Printf("password: peer %d requested reset for %q, code=%s", c.peerID, accName, code)
}

// handleConfirmPasswordReset implements Server.ConfirmPasswordReset(accountName,
// code, newPassword, peerID).
func (s *Server) handleConfirmPasswordReset(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 3 {
		return
	}
	accName := rpc.Args[0].Str
	code := rpc.Args[1].Str
	newPass := rpc.Args[2].Str
	// Validate the verification code.
	rc, ok := s.resetCodes[accName]
	if !ok {
		log.Printf("password: peer %d confirm reset failed (no code for %q)", c.peerID, accName)
		sendToClient(c, nil, rpcCommandFeedback, []Variant{NewString("No password reset was requested for this account"), NewInt(int64(c.peerID))})
		return
	}
	if time.Now().After(rc.Expires) {
		delete(s.resetCodes, accName)
		log.Printf("password: peer %d confirm reset failed (code expired for %q)", c.peerID, accName)
		sendToClient(c, nil, rpcCommandFeedback, []Variant{NewString("Password reset code has expired"), NewInt(int64(c.peerID))})
		return
	}
	if code != rc.Code {
		log.Printf("password: peer %d confirm reset failed (wrong code for %q)", c.peerID, accName)
		sendToClient(c, nil, rpcCommandFeedback, []Variant{NewString("Invalid password reset code"), NewInt(int64(c.peerID))})
		return
	}
	acc := s.store.Get(accName)
	if acc == nil {
		log.Printf("password: peer %d confirm reset failed (no account %q)", c.peerID, accName)
		return
	}
	delete(s.resetCodes, accName)
	s.store.mu.Lock()
	acc.Password = newPass
	s.store.saveLocked()
	s.store.mu.Unlock()
	log.Printf("password: peer %d confirmed reset for %q", c.peerID, accName)
}

// pushLevelUp sends LevelUp to the entity and its neighbours.
func (s *Server) pushLevelUp(e *Entity) {
	sendToClient(s.clientByPeer(e.PeerID), nil, rpcLevelUp, []Variant{NewInt(e.CharID)})
	for _, other := range s.world.snapshot() {
		if other.CharID == e.CharID {
			continue
		}
		if other.Pos.dist(e.Pos) > aoiRadius {
			continue
		}
		if oc := s.clientByPeer(other.PeerID); oc != nil {
			sendToClient(oc, nil, rpcLevelUp, []Variant{NewInt(e.CharID)})
		}
	}
}
