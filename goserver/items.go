package main

import (
	"fmt"
	"log"
)

// Phase-4 items/economy: inventory operations, attribute allocation,
// merchant trade, and progression pushes.
//
// Wire format mirrors sources/network/server/Server.gd and
// sources/actor/Inventory.gd. All pushes are authority -> client
// (Network.gd sorted nameIDs): ItemAdded=41, ItemEquiped=42, ItemRemoved=43,
// LevelUp=44, MerchantOpened=47, MerchantTrade=48, PickupDrop=53,
// UpdateAttributes=109, UpdateSkill=116.
//
// Client -> server RPCs handled here: DropItem=33, EquipItem=37,
// PickupDrop=53, SetAttributes=74, UnequipItem=108, UseItem=117,
// MerchantTrade=48.

const (
	rpcDropItem         = 33
	rpcEquipItem        = 37
	rpcItemAdded        = 41
	rpcItemEquiped      = 42
	rpcItemRemoved      = 43
	rpcLevelUp          = 44
	rpcMerchantOpened   = 47
	rpcMerchantTrade    = 48
	rpcPickupDrop       = 53
	rpcSetAttributes    = 74
	rpcUnequipItem      = 108
	rpcUpdateAttributes = 109
	rpcUpdateSkill      = 116
	rpcUseItem          = 117
)

// itemDef is a catalog entry. In SoM this is DB.ItemsDB (presets/cells/*.tres).
type itemDef struct {
	ID        int64
	Name      string
	Slot      int64 // ActorCommons.Slot; -1 = NONE (not equipable)
	Stackable bool
	Usable    bool
	BuyPrice  int64
	SellPrice int64
	Weight    int64
}

// itemCatalog is the seeded item database. Slots match ActorCommons.Slot:
// CHEST=0, LEGS=1, FEET=2, HANDS=3, HEAD=4, NECK=5, WEAPON=6, SHIELD=7.
var itemCatalog = map[int64]*itemDef{
	1001: {ID: 1001, Name: "Potion", Slot: -1, Stackable: true, Usable: true, BuyPrice: 50, SellPrice: 25, Weight: 100},
	1002: {ID: 1002, Name: "Antidote", Slot: -1, Stackable: true, Usable: true, BuyPrice: 30, SellPrice: 15, Weight: 80},
	1003: {ID: 1003, Name: "Iron Sword", Slot: 6, Stackable: false, BuyPrice: 200, SellPrice: 100, Weight: 500},
	1004: {ID: 1004, Name: "Leather Armor", Slot: 0, Stackable: false, BuyPrice: 150, SellPrice: 75, Weight: 400},
	1005: {ID: 1005, Name: "Wooden Shield", Slot: 7, Stackable: false, BuyPrice: 120, SellPrice: 60, Weight: 350},
	1006: {ID: 1006, Name: "Cloth Boots", Slot: 2, Stackable: false, BuyPrice: 80, SellPrice: 40, Weight: 200},
}

// charBySession returns the character for a client's session, or nil.
func (s *Server) charBySession(c *Client) *Character {
	if c.session == nil {
		return nil
	}
	c.session.mu.Lock()
	charID := c.session.charID
	c.session.mu.Unlock()
	if charID == peerUnknownID {
		return nil
	}
	return s.store.findCharacterByID(charID)
}

// persist saves the store to disk.
func (s *Server) persist() {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	s.store.saveLocked()
}

// handleUseItem implements Server.UseItem(itemID, peerID).
func (s *Server) handleUseItem(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 1 {
		return
	}
	itemID := rpc.Args[0].Int
	cell := itemCatalog[itemID]
	if cell == nil || !cell.Usable {
		return
	}
	chr := s.charBySession(c)
	if chr == nil {
		return
	}
	for i := range chr.Inventory {
		if chr.Inventory[i].ItemID == itemID && chr.Inventory[i].Count > 0 {
			chr.Inventory[i].Count--
			if chr.Inventory[i].Count <= 0 {
				chr.Inventory = append(chr.Inventory[:i], chr.Inventory[i+1:]...)
			}
			s.persist()
			sendToClient(c, nil, rpcItemRemoved, []Variant{
				NewInt(itemID), NewStringName(""), NewInt(1), NewInt(int64(i)),
			})
			log.Printf("items: peer %d used item %d (%s)", c.peerID, itemID, cell.Name)
			return
		}
	}
}

// handleDropItem implements Server.DropItem(itemID, customfield, count, index, peerID).
func (s *Server) handleDropItem(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 4 {
		return
	}
	itemID := rpc.Args[0].Int
	count := rpc.Args[2].Int
	index := rpc.Args[3].Int
	chr := s.charBySession(c)
	if chr == nil {
		return
	}
	if count <= 0 {
		return
	}
	if index < 0 || index >= int64(len(chr.Inventory)) {
		return
	}
	item := chr.Inventory[index]
	if item.ItemID != itemID || item.Count < count {
		return
	}
	item.Count -= count
	if item.Count <= 0 {
		chr.Inventory = append(chr.Inventory[:index], chr.Inventory[index+1:]...)
	}
	s.persist()
	sendToClient(c, nil, rpcItemRemoved, []Variant{
		NewInt(itemID), NewStringName(item.Customfield), NewInt(count), NewInt(index),
	})
	log.Printf("items: peer %d dropped item %d x%d", c.peerID, itemID, count)
}

// handleEquipItem implements Server.EquipItem(itemID, customfield, itemIndex, peerID).
func (s *Server) handleEquipItem(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 3 {
		return
	}
	itemID := rpc.Args[0].Int
	customfield := rpc.Args[1].Str
	itemIndex := rpc.Args[2].Int
	cell := itemCatalog[itemID]
	if cell == nil || cell.Slot < 0 {
		return
	}
	chr := s.charBySession(c)
	if chr == nil {
		return
	}
	if itemIndex < 0 || itemIndex >= int64(len(chr.Inventory)) {
		return
	}
	item := chr.Inventory[itemIndex]
	if item.ItemID != itemID || item.Customfield != customfield {
		return
	}
	chr.Inventory = append(chr.Inventory[:itemIndex], chr.Inventory[itemIndex+1:]...)
	chr.Equipment[cell.Slot] = ItemRecord{ItemID: itemID, Customfield: customfield, Count: 1}
	s.persist()
	s.notifyInstance(c, rpcItemEquiped, []Variant{
		NewInt(chr.ID), NewInt(itemID), NewStringName(customfield), NewBool(true), NewInt(itemIndex),
	})
	log.Printf("items: peer %d equipped item %d (%s) slot=%d", c.peerID, itemID, cell.Name, cell.Slot)
}

// handleUnequipItem implements Server.UnequipItem(itemID, customfield, peerID).
func (s *Server) handleUnequipItem(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 2 {
		return
	}
	itemID := rpc.Args[0].Int
	customfield := rpc.Args[1].Str
	cell := itemCatalog[itemID]
	if cell == nil || cell.Slot < 0 {
		return
	}
	chr := s.charBySession(c)
	if chr == nil {
		return
	}
	rec, ok := chr.Equipment[cell.Slot]
	if !ok || rec.ItemID != itemID || rec.Customfield != customfield {
		return
	}
	delete(chr.Equipment, cell.Slot)
	chr.Inventory = append(chr.Inventory, ItemRecord{ItemID: itemID, Customfield: customfield, Count: 1})
	s.persist()
	s.notifyInstance(c, rpcItemEquiped, []Variant{
		NewInt(chr.ID), NewInt(itemID), NewStringName(customfield), NewBool(false), NewInt(-1),
	})
	log.Printf("items: peer %d unequipped item %d (%s)", c.peerID, itemID, cell.Name)
}

// handlePickupDrop implements Server.PickupDrop(dropID, peerID).
func (s *Server) handlePickupDrop(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 1 {
		return
	}
	chr := s.charBySession(c)
	if chr == nil {
		return
	}
	dropID := rpc.Args[0].Int
	// Look up the drop and verify it exists.
	s.dropMu.Lock()
	d := s.drops[dropID]
	s.dropMu.Unlock()
	if d == nil {
		log.Printf("items: peer %d pickup drop %d: drop not found", c.peerID, dropID)
		return
	}
	// Verify proximity: the drop must be within pickup range (64 px).
	e := s.world.get(chr.ID)
	if e == nil || e.MapID != d.MapID || e.Pos.dist(d.Pos) > 64 {
		log.Printf("items: peer %d pickup drop %d: too far", c.peerID, dropID)
		return
	}
	// Remove the drop from the ground.
	s.removeDrop(dropID)
	// Add the actual item to the inventory (stacking if stackable).
	added := false
	for i := range chr.Inventory {
		if chr.Inventory[i].ItemID == d.ItemID && chr.Inventory[i].Customfield == d.Customfield {
			chr.Inventory[i].Count++
			added = true
			break
		}
	}
	if !added {
		chr.Inventory = append(chr.Inventory, ItemRecord{ItemID: d.ItemID, Customfield: d.Customfield, Count: 1})
	}
	s.persist()
	sendToClient(c, nil, rpcItemAdded, []Variant{
		NewInt(d.ItemID), NewStringName(d.Customfield), NewInt(1),
	})
	log.Printf("items: peer %d picked up drop %d (item %d)", c.peerID, dropID, d.ItemID)
}

// handleSetAttributes implements Server.SetAttributes(strength, vitality,
// agility, endurance, concentration, peerID).
// Phase-8: actually stores the attribute allocation (mirrors Stats.SetAttributes).
func (s *Server) handleSetAttributes(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 5 {
		return
	}
	strength := rpc.Args[0].Int
	vitality := rpc.Args[1].Int
	agility := rpc.Args[2].Int
	endurance := rpc.Args[3].Int
	concentration := rpc.Args[4].Int
	chr := s.charBySession(c)
	if chr == nil {
		return
	}
	// Validate against the level cap (mirrors Stats.SetAttributes).
	// Each attribute must be non-negative and within the total cap.
	maxPts := maxAttributePoints(chr.Level)
	if strength < 0 || vitality < 0 || agility < 0 || endurance < 0 || concentration < 0 {
		sendToClient(c, nil, rpcCommandFeedback, []Variant{
			NewString("Attributes cannot be negative"), NewInt(int64(c.peerID)),
		})
		return
	}
	// Check for overflow: if any attribute is huge, the sum may wrap negative.
	// Use a per-stat cap of maxPts to prevent overflow in the sum.
	if strength > maxPts || vitality > maxPts || agility > maxPts || endurance > maxPts || concentration > maxPts {
		sendToClient(c, nil, rpcCommandFeedback, []Variant{
			NewString(fmt.Sprintf("Attribute value exceeds cap %d", maxPts)), NewInt(int64(c.peerID)),
		})
		return
	}
	assigned := strength + vitality + agility + endurance + concentration
	if assigned > maxPts {
		log.Printf("stats: peer %d attribute allocation %d exceeds cap %d", c.peerID, assigned, maxPts)
		sendToClient(c, nil, rpcCommandFeedback, []Variant{
			NewString(fmt.Sprintf("Attribute allocation %d exceeds cap %d", assigned, maxPts)),
			NewInt(int64(c.peerID)),
		})
		return
	}
	chr.Attributes = attributePoints{
		Strength: strength, Vitality: vitality, Agility: agility,
		Endurance: endurance, Concentration: concentration,
	}
	s.persist()
	sendToClient(c, nil, rpcUpdateAttributes, []Variant{
		NewInt(strength), NewInt(vitality), NewInt(agility), NewInt(endurance), NewInt(concentration),
	})
	log.Printf("stats: peer %d set attributes str=%d vit=%d agi=%d end=%d con=%d (cap=%d)",
		c.peerID, strength, vitality, agility, endurance, concentration, maxPts)
}

// handleMerchantTrade implements Server.MerchantTrade(sellEntries, buyEntries, peerID).
// sellEntries/buyEntries are arrays of [itemID, count, customfield].
func (s *Server) handleMerchantTrade(c *Client, rpc *ParsedRPC) {
	if len(rpc.Args) < 2 {
		return
	}
	chr := s.charBySession(c)
	if chr == nil {
		return
	}
	sellEntries := rpc.Args[0]
	buyEntries := rpc.Args[1]
	var balance int64
	// Validate all counts are positive and bounded before computing.
	const maxTradeCount = 10000
	if sellEntries.Type == VariantArray {
		for _, e := range sellEntries.Arr {
			if e.Type != VariantArray || len(e.Arr) < 2 {
				continue
			}
			count := e.Arr[1].Int
			if count <= 0 || count > maxTradeCount {
				sendToClient(c, nil, rpcTradeClosed, []Variant{NewString("Invalid trade count.")})
				return
			}
		}
	}
	if buyEntries.Type == VariantArray {
		for _, e := range buyEntries.Arr {
			if e.Type != VariantArray || len(e.Arr) < 2 {
				continue
			}
			count := e.Arr[1].Int
			if count <= 0 || count > maxTradeCount {
				sendToClient(c, nil, rpcTradeClosed, []Variant{NewString("Invalid trade count.")})
				return
			}
		}
	}
	// Compute balance with overflow checks.
	if sellEntries.Type == VariantArray {
		for _, e := range sellEntries.Arr {
			if e.Type != VariantArray || len(e.Arr) < 2 {
				continue
			}
			cell := itemCatalog[e.Arr[0].Int]
			if cell == nil {
				continue
			}
			// Verify the seller owns the item.
			if !s.hasItem(chr, e.Arr[0].Int, "", e.Arr[1].Int) {
				sendToClient(c, nil, rpcTradeClosed, []Variant{NewString("You don't have enough items to sell.")})
				return
			}
			balance = safeAdd(balance, safeMul(cell.SellPrice, e.Arr[1].Int))
		}
	}
	if buyEntries.Type == VariantArray {
		for _, e := range buyEntries.Arr {
			if e.Type != VariantArray || len(e.Arr) < 2 {
				continue
			}
			cell := itemCatalog[e.Arr[0].Int]
			if cell == nil {
				continue
			}
			balance = safeSub(balance, safeMul(cell.BuyPrice, e.Arr[1].Int))
		}
	}
	if chr.GP+balance < 0 {
		sendToClient(c, nil, rpcTradeClosed, []Variant{NewString("You don't have enough GP.")})
		return
	}
	chr.GP += balance
	if sellEntries.Type == VariantArray {
		for _, e := range sellEntries.Arr {
			if e.Type != VariantArray || len(e.Arr) < 2 {
				continue
			}
			removeItem(chr, e.Arr[0].Int, "", e.Arr[1].Int)
		}
	}
	if buyEntries.Type == VariantArray {
		for _, e := range buyEntries.Arr {
			if e.Type != VariantArray || len(e.Arr) < 2 {
				continue
			}
			itemID := e.Arr[0].Int
			count := e.Arr[1].Int
			// Stack onto an existing entry if the item is stackable.
			cell := itemCatalog[itemID]
			if cell != nil && cell.Stackable {
				added := false
				for i := range chr.Inventory {
					if chr.Inventory[i].ItemID == itemID && chr.Inventory[i].Customfield == "" {
						chr.Inventory[i].Count += count
						added = true
						break
					}
				}
				if added {
					continue
				}
			}
			chr.Inventory = append(chr.Inventory, ItemRecord{
				ItemID: itemID, Customfield: "", Count: count,
			})
		}
	}
	s.persist()
	sendToClient(c, nil, rpcTradeClosed, []Variant{NewString("Trade completed.")})
	log.Printf("merchant: peer %d traded balance=%d gp", c.peerID, balance)
}

// removeItem removes count of itemID from the character's inventory.
func removeItem(chr *Character, itemID int64, customfield string, count int64) {
	for i := range chr.Inventory {
		if chr.Inventory[i].ItemID == itemID && chr.Inventory[i].Customfield == customfield {
			chr.Inventory[i].Count -= count
			if chr.Inventory[i].Count <= 0 {
				chr.Inventory = append(chr.Inventory[:i], chr.Inventory[i+1:]...)
			}
			return
		}
	}
}

// notifyInstance broadcasts to all clients in the same map as c.
func (s *Server) notifyInstance(c *Client, nameID uint16, args []Variant) {
	e := s.world.get(c.session.charID)
	if e == nil {
		return
	}
	for _, other := range s.world.snapshot() {
		if other.MapID != e.MapID {
			continue
		}
		if oc := s.clientByPeer(other.PeerID); oc != nil {
			sendToClient(oc, nil, nameID, args)
		}
	}
}
