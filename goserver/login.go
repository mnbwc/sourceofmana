package main

// Phase-1 login chain: mirrors the SoM Godot server flow
// (sources/network/server/Server.gd + Peers.gd) with an in-memory account
// store. Client -> server RPCs are dispatched by nameID (Godot assigns RPC
// IDs by sorted @rpc method name across sources/network/Network.gd; the
// client and server share the same node script, so the IDs below are valid
// for both directions).
//
// Flow implemented (see AUTO_STATE.md):
//   LoginWithPassword / LoginWithToken / CreateAccount
//       -> AuthError(err)                 (always, even on success)
//       -> AuthTokenResult(name, token)   (only when rememberMe / token login)
//   CharacterListing
//       -> CharacterInfo(info, equipment) per character
//       -> CharacterError(err)            (ERR_OK or ERR_EMPTY_ACCOUNT)
//   CreateCharacter
//       -> CharacterInfo(...) on success
//       -> CharacterError(err)
//   ConnectCharacter
//       -> CharacterError(err); on success the peer is IN_GAME
//   DeleteCharacter / DisconnectAccount / DisconnectCharacter

import (
	"log"
	"sort"
	"sync"
)

// RPC name IDs — index of the method name in the sorted @rpc method list of
// Network.gd (118 methods, verified 2026-11-09).
const (
	rpcAddOnlinePlayer              = 0
	rpcAuthError                    = 1
	rpcAuthTokenResult              = 2
	rpcBulkCall                     = 3
	rpcCharacterError               = 8
	rpcCharacterInfo                = 9
	rpcCharacterListing             = 10
	rpcConnectCharacter             = 19
	rpcCreateAccount                = 25
	rpcCreateCharacter              = 26
	rpcDeleteCharacter              = 27
	rpcDisconnectAccount            = 28
	rpcDisconnectCharacter          = 29
	rpcLoginWithPassword            = 45
	rpcLoginWithToken               = 46
	rpcPreloadEntity                = 54
	rpcPreloadPlayer                = 55
	rpcRefreshOnlineList            = 59
	rpcRemoveOnlinePlayer           = 62
	rpcRetrieveCharacterInformation = 66
	rpcSetClickPos                  = 75
	rpcSetMovePos                   = 76
	rpcSetOrientation               = 77
	rpcWarpPlayer                   = 118

	// Phase-3 gameplay (Network.gd sorted indices).
	rpcCasted       = 6
	rpcChatPlayer   = 11
	rpcChatQuery    = 12
	rpcChatSystem   = 13
	rpcEmote        = 35
	rpcTriggerChat  = 90
	rpcTriggerEmote = 94

	// Phase-3 party (Network.gd sorted indices).
	rpcPartyInvite         = 51
	rpcPartyUpdate         = 52
	rpcTriggerPartyInvite  = 100
	rpcTriggerPartyAccept  = 98
	rpcTriggerPartyDecline = 99
	rpcTriggerPartyLeave   = 102
	rpcTriggerPartyKick    = 101
	rpcTriggerPartyPromote = 103

	// Phase-3 misc pushes (Network.gd sorted indices).
	rpcPushNotification = 56

	// Phase-3 trade (Network.gd sorted indices).
	rpcTradeCancel    = 82
	rpcTradeClosed    = 83
	rpcTradeLock      = 84
	rpcTradeOpened    = 85
	rpcTradeRequested = 86
	rpcTradeRespond   = 87
	rpcTradeSetOffer  = 88
	rpcTradeUpdate    = 89

	// Phase-3 NPC/quest (Network.gd sorted indices).
	rpcContextChoice       = 20
	rpcContextClose        = 21
	rpcContextContinue     = 22
	rpcContextText         = 23
	rpcContextThink        = 24
	rpcCommandFeedback     = 16
	rpcRefreshProgress     = 60
	rpcTriggerChoice       = 91
	rpcTriggerCloseContext = 92
	rpcTriggerCommand      = 93
	rpcTriggerExplore      = 95
	rpcTriggerInteract     = 96
	rpcTriggerNextContext  = 97
	rpcTriggerSelect       = 105
	rpcUpdateBestiary      = 110
	rpcUpdatePublicStats   = 114
	rpcUpdateQuest         = 115

	// Phase-3 combat (Network.gd sorted indices).
	rpcTargetAlteration = 79
	rpcThrowProjectile  = 80
	rpcTriggerRespawn   = 104
	rpcTriggerSkill     = 107

	// Phase-3 inventory/equipment (Network.gd sorted indices).
	rpcRefreshEquipment = 57
	rpcRefreshInventory = 58
)

// NetworkCommons.AuthError (sourceofmana/sources/network/NetworkCommons.gd)
const (
	authErrOK            = 0
	authErrNoPeerData    = 1
	authErrAuth          = 5
	authErrPasswordSize  = 7
	authErrNameAvailable = 8 // name taken (SoM reuses this code for duplicates)
	authErrNameValid     = 9
	authErrNameSize      = 10
	authErrToken         = 13
)

// NetworkCommons.CharacterError
const (
	charErrOK              = 0
	charErrAlreadyLoggedIn = 1
	charErrNoPeerData      = 2
	charErrNoCharacterID   = 3
	charErrNoAccountID     = 4
	charErrMissingParams   = 6
	charErrNameAvailable   = 7 // name taken
	charErrNameValid       = 8
	charErrNameSize        = 9
	charErrSlotAvailable   = 10 // slots full
	charErrEmptyAccount    = 11
)

// Login constraints (NetworkCommons.CheckAuthInformation / CheckCharacterInformation).
const (
	accountNameMinSize = 4
	accountNameMaxSize = 16
	passwordMinSize    = 4
	passwordMaxSize    = 32
	charNameMinSize    = 3
	charNameMaxSize    = 16
	maxCharacterCount  = 4
)

// Account is an in-memory account with its characters.
type Account struct {
	Name       string
	Password   string // plaintext for the phase-1 stub; real server hashes
	Token      string // issued on rememberMe login
	Permission int    // ActorCommons.Permission: 0=USER, 1=PREMIUM, 2=GAME_MASTER, 3=ADMINISTRATOR
	Characters []*Character
}

// Character is a character record with persistent progression data.
type Character struct {
	ID        int64
	Name      string
	Level     int64
	Race      int64
	Gender    int64
	Hairstyle int64
	Haircolor int64
	Skintone  int64
	// Progression (persisted).
	Experience int64
	GP         int64 // gold pieces
	Health     int64
	Mana       int64
	Stamina    int64
	Karma      int64
	Weight     int64
	// Inventory: list of {item_id, customfield, count}.
	Inventory []ItemRecord
	// Equipment: slot -> {item_id, customfield}.
	Equipment map[int64]ItemRecord
	// Attributes: assigned attribute points (phase-8).
	Attributes attributePoints
}

// ItemRecord is a persisted inventory/equipment entry.
type ItemRecord struct {
	ItemID      int64
	Customfield string
	Count       int64
}

// AccountStore is the in-memory database, mirrored to disk (see store.go).
type AccountStore struct {
	mu         sync.Mutex
	accounts   map[string]*Account
	nextCharID int64
}

func NewAccountStore() *AccountStore {
	return &AccountStore{accounts: make(map[string]*Account), nextCharID: 1}
}

func (s *AccountStore) Get(name string) *Account {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.accounts[name]
}

// Add, AddCharacter, RemoveCharacter and SetToken persist to disk after each
// mutation; their implementations live in store.go.

// Session is the per-peer login state machine.
type Session struct {
	mu      sync.Mutex
	authed  bool
	account *Account
	charID  int64 // NetworkCommons.PeerUnknownID until ConnectCharacter
	inGame  bool
}

// peerUnknownID mirrors NetworkCommons.PeerUnknownID (-1).
const peerUnknownID = int64(-1)

func (s *Session) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authed = false
	s.account = nil
	s.charID = peerUnknownID
	s.inGame = false
}

func dictInt(d map[string]Variant, key string) int64 {
	if d == nil {
		return 0
	}
	if v, ok := d[key]; ok && v.Type == VariantInt {
		return v.Int
	}
	return 0
}

// validateName mirrors NetworkCommons.CheckValid/CheckSize (minimal ASCII form).
func validateName(name string, minSize, maxSize int) int {
	if len(name) < minSize || len(name) > maxSize {
		if len(name) < minSize {
			return 1 // too small
		}
		return 2 // too large
	}
	for _, r := range name {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_'
		if !ok {
			return 3 // invalid char
		}
	}
	return 0
}

// replyCacheID resolves the cache ID to use when pushing an RPC to the
// client's Network node. The client announced that node with a SIMPLIFY_PATH
// whose ID it also uses on its side for peer=server, so echoing the incoming
// rpc.NodeID resolves correctly on the client (verified in phase 0).
func replyCacheID(c *Client, rpc *ParsedRPC) uint32 {
	if rpc != nil && rpc.NodeID&0x80000000 == 0 { // node-ID compression flag: resolve via cache
		return rpc.NodeID
	}
	for id := range c.nodeCache {
		return id
	}
	return 1
}

// sendToClient pushes an RPC to the client's Network node.
func sendToClient(c *Client, rpc *ParsedRPC, nameID uint16, args []Variant) {
	packet, err := BuildRPCPacket(replyCacheID(c, rpc), nameID, args)
	if err != nil {
		log.Printf("peer %d build rpc nameID=%d: %v", c.peerID, nameID, err)
		return
	}
	c.enqueue(packet)
	if nameID == 32 {
		log.Printf("peer %d sent DropAdded args=%s", c.peerID, formatArgs(args))
	}
}

// onRemoteCall dispatches client -> server RPCs (phase-1 login chain).
func (s *Server) onRemoteCall(c *Client, rpc *ParsedRPC) {
	switch rpc.NameID {
	case rpcCreateAccount:
		// (accountName, password, email, rememberMe, platform)
		s.handleCreateAccount(c, rpc)
	case rpcLoginWithPassword:
		// (accountName, password, rememberMe, platform)
		s.handleLoginWithPassword(c, rpc)
	case rpcLoginWithToken:
		// (accountName, token, platform)
		s.handleLoginWithToken(c, rpc)
	case rpcCharacterListing:
		s.handleCharacterListing(c, rpc)
	case rpcCreateCharacter:
		// (charName, traits, attributes)
		s.handleCreateCharacter(c, rpc)
	case rpcConnectCharacter:
		// (nickname)
		s.handleConnectCharacter(c, rpc)
	case rpcDeleteCharacter:
		// (charName)
		s.handleDeleteCharacter(c, rpc)
	case rpcDisconnectAccount:
		c.session.Reset()
		log.Printf("peer %d account disconnected", c.peerID)
	case rpcDisconnectCharacter:
		s.leaveWorld(c)
		c.session.mu.Lock()
		c.session.charID = peerUnknownID
		c.session.inGame = false
		c.session.mu.Unlock()
		log.Printf("peer %d character disconnected", c.peerID)
	case rpcRetrieveCharacterInformation:
		s.handleRetrieveCharacterInformation(c, rpc)
	case rpcSetMovePos:
		s.handleSetMovePos(c, rpc)
	case rpcSetClickPos:
		s.handleSetClickPos(c, rpc)
	case rpcSetOrientation:
		s.handleSetOrientation(c, rpc)
	case rpcTriggerChat:
		s.handleTriggerChat(c, rpc)
	case rpcChatQuery:
		s.handleChatQuery(c, rpc)
	case rpcTriggerEmote:
		s.handleTriggerEmote(c, rpc)
	case rpcCasted:
		s.handleCasted(c, rpc)
	case rpcTriggerInteract:
		s.handleTriggerInteract(c, rpc)
	case rpcTriggerChoice:
		s.handleTriggerChoice(c, rpc)
	case rpcTriggerNextContext:
		s.handleTriggerNextContext(c, rpc)
	case rpcTriggerCloseContext:
		s.handleTriggerCloseContext(c, rpc)
	case rpcTriggerSelect:
		s.handleTriggerSelect(c, rpc)
	case rpcTriggerExplore:
		s.handleTriggerExplore(c, rpc)
	case rpcTriggerCommand:
		s.handleTriggerCommand(c, rpc)
	case rpcTriggerPartyInvite:
		s.handleTriggerPartyInvite(c, rpc)
	case rpcTriggerPartyAccept:
		s.handleTriggerPartyAccept(c, rpc)
	case rpcTriggerPartyDecline:
		s.handleTriggerPartyDecline(c, rpc)
	case rpcTriggerPartyLeave:
		s.handleTriggerPartyLeave(c, rpc)
	case rpcTriggerPartyKick:
		s.handleTriggerPartyKick(c, rpc)
	case rpcTriggerPartyPromote:
		s.handleTriggerPartyPromote(c, rpc)
	case rpcTradeRespond:
		s.handleTradeRespond(c, rpc)
	case rpcTradeSetOffer:
		s.handleTradeSetOffer(c, rpc)
	case rpcTradeLock:
		s.handleTradeLock(c, rpc)
	case rpcTradeCancel:
		s.handleTradeCancel(c, rpc)
	case rpcTriggerSkill:
		s.handleTriggerSkill(c, rpc)
	case rpcTriggerRespawn:
		s.handleTriggerRespawn(c, rpc)
	// Phase-4 items/economy.
	case rpcUseItem:
		s.handleUseItem(c, rpc)
	case rpcDropItem:
		s.handleDropItem(c, rpc)
	case rpcEquipItem:
		s.handleEquipItem(c, rpc)
	case rpcUnequipItem:
		s.handleUnequipItem(c, rpc)
	case rpcPickupDrop:
		s.handlePickupDrop(c, rpc)
	case rpcSetAttributes:
		s.handleSetAttributes(c, rpc)
	case rpcMerchantTrade:
		s.handleMerchantTrade(c, rpc)
	// Phase-5 progression.
	case rpcRequestOnlineList:
		s.handleRequestOnlineList(c, rpc)
	case rpcSetViewportSize:
		s.handleSetViewportSize(c, rpc)
	case rpcCameraLookAt:
		s.handleCameraLookAt(c, rpc)
	case rpcCameraReset:
		s.handleCameraReset(c, rpc)
	case rpcTriggerSit:
		s.handleTriggerSit(c, rpc)
	case rpcChangePassword:
		s.handleChangePassword(c, rpc)
	case rpcRequestPasswordReset:
		s.handleRequestPasswordReset(c, rpc)
	case rpcConfirmPasswordReset:
		s.handleConfirmPasswordReset(c, rpc)
	// Phase-6: buffs/drops/private-stats/misc.
	case rpcRetrieveInventory:
		s.handleRetrieveInventory(c, rpc)
	case rpcPushNotification:
		s.handlePushNotification(c, rpc)
	case rpcExpress:
		s.handleExpress(c, rpc)
	case rpcThrowProjectile:
		s.handleThrowProjectile(c, rpc)
	case rpcMorphed:
		s.handleMorphed(c, rpc)
	case rpcToggleContext:
		s.handleToggleContext(c, rpc)
	case rpcClearNavigation:
		s.handleClearNavigation(c, rpc)
	case rpcDisplayActions:
		s.handleDisplayActions(c, rpc)
	case rpcDisplayProgressionTracker:
		s.handleDisplayProgressionTracker(c, rpc)
	case rpcClearProgressionTracker:
		s.handleClearProgressionTracker(c, rpc)
	case rpcHighlightUI:
		s.handleHighlightUI(c, rpc)
	case rpcOpenUI:
		s.handleOpenUI(c, rpc)
	case rpcCommandModifier:
		s.handleCommandModifier(c, rpc)
	default:
		log.Printf("peer %d unhandled RPC nameID=%d args=%s", c.peerID, rpc.NameID, formatArgs(rpc.Args))
	}
}

func (s *Server) handleCreateAccount(c *Client, rpc *ParsedRPC) {
	name, password := argString(rpc.Args, 0), argString(rpc.Args, 1)
	rememberMe := argBool(rpc.Args, 3)
	err := authErrOK
	nameLen := validateName(name, accountNameMinSize, accountNameMaxSize)
	passLen := validateName(password, passwordMinSize, passwordMaxSize)
	switch {
	case nameLen != 0:
		err = authErrNameSize
	case passLen != 0:
		err = authErrPasswordSize
	case s.store.Get(name) != nil:
		err = authErrNameAvailable
	default:
		acc := s.store.Add(name, password)
		s.finalizeLogin(c, acc, rememberMe)
	}
	s.sendAuthError(c, rpc, err)
	log.Printf("peer %d CreateAccount %q -> err=%d", c.peerID, name, err)
}

func (s *Server) handleLoginWithPassword(c *Client, rpc *ParsedRPC) {
	name, password := argString(rpc.Args, 0), argString(rpc.Args, 1)
	rememberMe := argBool(rpc.Args, 2)
	err := authErrOK
	acc := s.store.Get(name)
	nameLen := validateName(name, accountNameMinSize, accountNameMaxSize)
	passLen := validateName(password, passwordMinSize, passwordMaxSize)
	switch {
	case nameLen != 0:
		err = authErrNameSize
	case passLen != 0:
		err = authErrPasswordSize
	case acc == nil || acc.Password != password:
		err = authErrAuth
	default:
		s.finalizeLogin(c, acc, rememberMe)
	}
	s.sendAuthError(c, rpc, err)
	log.Printf("peer %d LoginWithPassword %q -> err=%d", c.peerID, name, err)
}

func (s *Server) handleLoginWithToken(c *Client, rpc *ParsedRPC) {
	name, token := argString(rpc.Args, 0), argString(rpc.Args, 1)
	err := authErrOK
	acc := s.store.Get(name)
	switch {
	case acc == nil:
		err = authErrToken
	case acc.Token == "" || acc.Token != token:
		err = authErrToken
	default:
		s.finalizeLogin(c, acc, false)
	}
	s.sendAuthError(c, rpc, err)
	log.Printf("peer %d LoginWithToken %q -> err=%d", c.peerID, name, err)
}

// finalizeLogin mirrors Peers.FinalizeLogin + IssueAuthToken.
func (s *Server) finalizeLogin(c *Client, acc *Account, rememberMe bool) {
	if rememberMe {
		// SetToken persists the new token to disk (it takes the store lock).
		s.store.SetToken(acc, newToken())
	}
	s.store.mu.Lock()
	token := acc.Token
	s.store.mu.Unlock()

	c.session.mu.Lock()
	c.session.authed = true
	c.session.account = acc
	c.session.charID = peerUnknownID
	c.session.inGame = false
	c.session.mu.Unlock()

	if rememberMe && token != "" {
		// AuthTokenResult(nameID=2): (accountName, token)
		sendToClient(c, nil, rpcAuthTokenResult, []Variant{NewString(acc.Name), NewString(token)})
	}
}

func (s *Server) sendAuthError(c *Client, rpc *ParsedRPC, errCode int) {
	// AuthError(nameID=1): (err: int)
	sendToClient(c, rpc, rpcAuthError, []Variant{NewInt(int64(errCode))})
}

func (s *Server) handleCharacterListing(c *Client, rpc *ParsedRPC) {
	sess := c.session
	sess.mu.Lock()
	acc := sess.account
	sess.mu.Unlock()

	errCode := charErrOK
	if acc == nil {
		errCode = charErrNoAccountID
	} else {
		s.store.mu.Lock()
		chars := append([]*Character(nil), acc.Characters...)
		s.store.mu.Unlock()
		if len(chars) == 0 {
			errCode = charErrEmptyAccount
		} else {
			for _, chr := range chars {
				sendToClient(c, rpc, rpcCharacterInfo, []Variant{characterInfoDict(chr), NewDict(map[string]Variant{})})
			}
		}
	}
	// CharacterError(nameID=8): (err: int)
	sendToClient(c, rpc, rpcCharacterError, []Variant{NewInt(int64(errCode))})
	log.Printf("peer %d CharacterListing -> err=%d (%d chars)", c.peerID, errCode, len(accCharChars(acc)))
}

func (s *Server) handleCreateCharacter(c *Client, rpc *ParsedRPC) {
	name := argString(rpc.Args, 0)
	traits := argDict(rpc.Args, 1)
	sess := c.session
	sess.mu.Lock()
	acc := sess.account
	sess.mu.Unlock()

	errCode := charErrOK
	var created *Character
	nickLen := validateName(name, charNameMinSize, charNameMaxSize)
	switch {
	case acc == nil:
		errCode = charErrNoAccountID
	case nickLen != 0:
		errCode = charErrNameSize
	case s.accountHasName(acc, name):
		errCode = charErrNameAvailable
	case len(accCharChars(acc)) >= maxCharacterCount:
		errCode = charErrSlotAvailable
	default:
		created = s.store.AddCharacter(acc, name, traits)
		sendToClient(c, rpc, rpcCharacterInfo, []Variant{characterInfoDict(created), NewDict(map[string]Variant{})})
	}
	sendToClient(c, rpc, rpcCharacterError, []Variant{NewInt(int64(errCode))})
	log.Printf("peer %d CreateCharacter %q -> err=%d", c.peerID, name, errCode)
}

func (s *Server) handleConnectCharacter(c *Client, rpc *ParsedRPC) {
	nick := argString(rpc.Args, 0)
	sess := c.session
	sess.mu.Lock()
	acc := sess.account
	sess.mu.Unlock()

	errCode := charErrOK
	if acc == nil {
		errCode = charErrNoAccountID
	} else {
		chr := s.store.findCharacter(acc, nick)
		if chr == nil {
			errCode = charErrNoCharacterID
		} else {
			sess.mu.Lock()
			sess.charID = chr.ID
			sess.inGame = true
			sess.mu.Unlock()
			s.enterWorld(c, chr) // phase 2: WarpPlayer/PreloadPlayer/online-list sequence
		}
	}
	sendToClient(c, rpc, rpcCharacterError, []Variant{NewInt(int64(errCode))})
	log.Printf("peer %d ConnectCharacter %q -> err=%d", c.peerID, nick, errCode)
}

func (s *Server) handleDeleteCharacter(c *Client, rpc *ParsedRPC) {
	name := argString(rpc.Args, 0)
	sess := c.session
	sess.mu.Lock()
	acc := sess.account
	inGame := sess.inGame
	sess.mu.Unlock()

	errCode := charErrOK
	switch {
	case acc == nil:
		errCode = charErrNoAccountID
	case inGame:
		errCode = charErrAlreadyLoggedIn
	case s.store.findCharacter(acc, name) == nil:
		errCode = charErrNameValid
	case !s.store.RemoveCharacter(acc, name):
		errCode = charErrNameAvailable
	}
	sendToClient(c, rpc, rpcCharacterError, []Variant{NewInt(int64(errCode))})
	log.Printf("peer %d DeleteCharacter %q -> err=%d", c.peerID, name, errCode)
}

// ---- helpers ----

func argString(args []Variant, i int) string {
	if i < len(args) && (args[i].Type == VariantString || args[i].Type == VariantStringName) {
		return args[i].Str
	}
	return ""
}

func argBool(args []Variant, i int) bool {
	if i < len(args) && (args[i].Type == VariantBool || args[i].Type == VariantInt) {
		if args[i].Type == VariantBool {
			return args[i].Bool
		}
		return args[i].Int != 0
	}
	return false
}

// argVariant returns the i-th argument as-is (for array/dict payloads).
func argVariant(args []Variant, i int) Variant {
	if i < len(args) {
		return args[i]
	}
	return Variant{}
}

func argDict(args []Variant, i int) map[string]Variant {
	if i < len(args) && args[i].Type == VariantDictionary {
		return args[i].Dict
	}
	return nil
}

func accCharChars(acc *Account) []*Character {
	if acc == nil {
		return nil
	}
	return acc.Characters
}

func (s *AccountStore) findCharacter(acc *Account, name string) *Character {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range acc.Characters {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// findCharacterByID scans all accounts for a character with the given ID.
// Used by in-game handlers that only have the session's charID.
func (s *AccountStore) findCharacterByID(charID int64) *Character {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, acc := range s.accounts {
		for _, c := range acc.Characters {
			if c.ID == charID {
				return c
			}
		}
	}
	return nil
}

// findAccountByID returns the account that owns the character with the given ID.
func (s *AccountStore) findAccountByID(charID int64) *Account {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, acc := range s.accounts {
		for _, c := range acc.Characters {
			if c.ID == charID {
				return acc
			}
		}
	}
	return nil
}

// persistLocked writes the store to disk. Callers must hold s.mu.
func (s *AccountStore) persistLocked() {
	s.saveLocked()
}

func (s *Server) accountHasName(acc *Account, name string) bool {
	return s.store.findCharacter(acc, name) != nil
}

// characterInfoDict builds the info Dictionary the client's
// CharacterPanel.AddCharacter expects (SELECT * row over character+stat+
// trait+attribute; the client fills missing keys with defaults).
func characterInfoDict(c *Character) Variant {
	return NewDict(map[string]Variant{
		"nickname":   NewString(c.Name),
		"level":      NewInt(c.Level),
		"char_id":    NewInt(c.ID),
		"race":       NewInt(c.Race),
		"gender":     NewInt(c.Gender),
		"hairstyle":  NewInt(c.Hairstyle),
		"haircolor":  NewInt(c.Haircolor),
		"skintone":   NewInt(c.Skintone),
		"health":     NewInt(c.Health),
		"mana":       NewInt(c.Mana),
		"stamina":    NewInt(c.Stamina),
		"experience": NewInt(c.Experience),
		"gp":         NewInt(c.GP),
		"karma":      NewInt(c.Karma),
		"weight":     NewInt(c.Weight),
	})
}

// equipmentSlotNames maps ActorCommons.Slot equipment IDs to the lowercase
// keys SoM's Inventory.ImportEquipment expects.
var equipmentSlotNames = map[int64]string{
	0: "chest", 1: "legs", 2: "feet", 3: "hands",
	4: "head", 5: "neck", 6: "weapon", 7: "shield",
}

// equipmentDict builds the equipment Dictionary for RefreshEquipment:
// each slot key maps to the item's cellID (or DB.UnknownHash=-1 when empty),
// with a "<slot>Custom" key for the custom field.
func equipmentDict(c *Character) Variant {
	d := map[string]Variant{}
	for slot, name := range equipmentSlotNames {
		if item, ok := c.Equipment[slot]; ok {
			d[name] = NewInt(item.ItemID)
			d[name+"Custom"] = NewString(item.Customfield)
		} else {
			d[name] = NewInt(-1) // DB.UnknownHash
			d[name+"Custom"] = NewString("")
		}
	}
	return NewDict(d)
}

// inventoryArray builds the Array[Dictionary] for RefreshInventory.
func inventoryArray(c *Character) Variant {
	arr := make([]Variant, 0, len(c.Inventory))
	for _, it := range c.Inventory {
		arr = append(arr, NewDict(map[string]Variant{
			"item_id":     NewInt(it.ItemID),
			"customfield": NewString(it.Customfield),
			"count":       NewInt(it.Count),
		}))
	}
	return Variant{Type: VariantArray, Arr: arr}
}

var tokenCounter uint64

// newToken generates a placeholder auth token (crypto strength not needed
// for the phase-1 in-memory stub).
func newToken() string {
	tokenCounter++
	return "tok-" + itoa(int64(tokenCounter)) + "-" + itoa(int64(len(sortedNamesCache)))
}

// sortedNamesCache exists only to keep sort imported for the documented
// nameID derivation below.
var sortedNamesCache = func() []string {
	names := []string{
		"AddOnlinePlayer", "AuthError", "AuthTokenResult", "BulkCall",
		"CharacterError", "CharacterInfo", "CharacterListing", "ConnectCharacter",
		"CreateAccount", "CreateCharacter", "DeleteCharacter", "DisconnectAccount",
		"DisconnectCharacter", "LoginWithPassword", "LoginWithToken",
		"PreloadEntity", "PreloadPlayer", "RefreshOnlineList", "RemoveOnlinePlayer",
		"RetrieveCharacterInformation", "SetClickPos", "SetMovePos", "SetOrientation",
		"WarpPlayer",
	}
	sort.Strings(names)
	return names
}()
