package main

// SQLite persistence for the AccountStore. The schema mirrors SoM's
// data/conf/templates/sqlite.template.db so that the Go server can share a
// database with the Godot server.
//
// The in-memory cache (AccountStore.accounts) is still the fast path for
// reads; SQLite is the source of truth. Mutations write through to SQLite
// immediately (WAL mode, busy_timeout=5000), so saveLocked() is a no-op.

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

// sqliteSchema is the SoM SQLite schema (data/conf/templates/sqlite.template.db).
const sqliteSchema = `
CREATE TABLE IF NOT EXISTS "account" (
    "account_id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "username" TEXT NOT NULL UNIQUE COLLATE NOCASE,
    "password" TEXT NOT NULL,
    "password_salt" TEXT NOT NULL,
    "email" TEXT NOT NULL,
    "created_timestamp" INTEGER NOT NULL,
    "last_timestamp" INTEGER,
    "banned_timestamp" INTEGER,
    "flags" INTEGER
);

CREATE TABLE IF NOT EXISTS "character" (
    "char_id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "account_id" INTEGER NOT NULL,
    "nickname" TEXT NOT NULL UNIQUE COLLATE NOCASE,
    "created_timestamp" INTEGER NOT NULL,
    "last_timestamp" INTEGER,
    "total_time" INTEGER,
    "pos_x" INTEGER, "pos_y" INTEGER, "pos_map" INTEGER,
    "respawn_x" INTEGER, "respawn_y" INTEGER, "respawn_map" INTEGER,
    "explore_x" INTEGER, "explore_y" INTEGER,
    FOREIGN KEY("account_id") REFERENCES "account"("account_id")
);

CREATE TABLE IF NOT EXISTS "stat" (
    char_id INTEGER PRIMARY KEY,
    level INTEGER, experience INTEGER, gp INTEGER,
    health INTEGER, mana INTEGER, stamina INTEGER, karma INTEGER,
    FOREIGN KEY (char_id) REFERENCES character(char_id)
);

CREATE TABLE IF NOT EXISTS "trait" (
    "char_id" INTEGER PRIMARY KEY,
    "hairstyle" INTEGER, "haircolor" INTEGER, "race" INTEGER,
    "skintone" INTEGER, "gender" INTEGER, "shape" INTEGER, "spirit" INTEGER,
    FOREIGN KEY("char_id") REFERENCES "character"("char_id")
);

CREATE TABLE IF NOT EXISTS "attribute" (
    char_id INTEGER PRIMARY KEY,
    strength INTEGER, vitality INTEGER, agility INTEGER,
    endurance INTEGER, concentration INTEGER,
    FOREIGN KEY (char_id) REFERENCES character(char_id)
);

CREATE TABLE IF NOT EXISTS "equipment" (
    "char_id" INTEGER PRIMARY KEY,
    "weapon" INTEGER, "shield" INTEGER, "ammunition" INTEGER,
    "hands" INTEGER, "chest" INTEGER, "neck" INTEGER,
    "feet" INTEGER, "head" INTEGER, "legs" INTEGER,
    "accessory1" INTEGER, "accessory2" INTEGER,
    "weaponCustom" TEXT, "shieldCustom" TEXT, "ammunitionCustom" TEXT,
    "handsCustom" TEXT, "chestCustom" TEXT, "neckCustom" TEXT,
    "feetCustom" TEXT, "headCustom" TEXT, "legsCustom" TEXT,
    "accessory1Custom" TEXT, "accessory2Custom" TEXT,
    FOREIGN KEY("char_id") REFERENCES "character"("char_id")
);

CREATE TABLE IF NOT EXISTS "item" (
    item_id INTEGER NOT NULL, char_id INTEGER NOT NULL,
    count INTEGER NOT NULL, storage INTEGER NOT NULL CHECK(storage IN (0, 1)),
    customfield TEXT NOT NULL,
    PRIMARY KEY (char_id, item_id, storage, customfield),
    FOREIGN KEY (char_id) REFERENCES character(char_id)
);

CREATE TABLE IF NOT EXISTS "quest" (
    char_id INTEGER NOT NULL, quest_id INTEGER NOT NULL,
    state INTEGER NOT NULL, PRIMARY KEY (char_id, quest_id),
    FOREIGN KEY (char_id) REFERENCES character(char_id)
);

CREATE TABLE IF NOT EXISTS "skill" (
    char_id INTEGER NOT NULL, skill_id INTEGER NOT NULL,
    level INTEGER, PRIMARY KEY (char_id, skill_id),
    FOREIGN KEY (char_id) REFERENCES character(char_id)
);

CREATE TABLE IF NOT EXISTS "bestiary" (
    char_id INTEGER NOT NULL, mob_id INTEGER NOT NULL,
    killed_count INTEGER DEFAULT 0, PRIMARY KEY (char_id, mob_id),
    FOREIGN KEY (char_id) REFERENCES character(char_id)
);

CREATE TABLE IF NOT EXISTS "auth_token" (
    account_id INTEGER NOT NULL, token TEXT NOT NULL,
    created_timestamp INTEGER NOT NULL,
    PRIMARY KEY (account_id, token)
);

CREATE TABLE IF NOT EXISTS "migration" (version INTEGER NOT NULL);
INSERT OR IGNORE INTO migration VALUES (0);

CREATE TRIGGER IF NOT EXISTS trg_account_delete AFTER DELETE ON account FOR EACH ROW
BEGIN DELETE FROM character WHERE account_id = OLD.account_id; END;

CREATE TRIGGER IF NOT EXISTS trg_character_delete AFTER DELETE ON character FOR EACH ROW
BEGIN
    DELETE FROM trait WHERE char_id = OLD.char_id;
    DELETE FROM attribute WHERE char_id = OLD.char_id;
    DELETE FROM stat WHERE char_id = OLD.char_id;
    DELETE FROM equipment WHERE char_id = OLD.char_id;
END;

CREATE TRIGGER IF NOT EXISTS trg_character_new AFTER INSERT ON character FOR EACH ROW
BEGIN
    INSERT INTO trait (char_id) VALUES (NEW.char_id);
    INSERT INTO attribute (char_id) VALUES (NEW.char_id);
    INSERT INTO stat (char_id, level) VALUES (NEW.char_id, 1);
    INSERT INTO equipment (char_id) VALUES (NEW.char_id);
END;
`

// dbFileDefault is used when SOM_DB_FILE is not set.
const dbFileDefault = "som.db"

// dbFile returns the configured SQLite database path.
func dbFile() string {
	if p := os.Getenv("SOM_DB_FILE"); p != "" {
		return p
	}
	return dbFileDefault
}

// openSQLite opens (and if necessary creates) the SQLite database at path,
// applies the SoM schema, and configures WAL mode for concurrent access.
func openSQLite(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// WAL mode allows concurrent readers with a single writer.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, err
	}
	// Wait up to 5s for a locked database instead of failing immediately.
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, err
	}
	// Create schema if not exists.
	if _, err := db.Exec(sqliteSchema); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// importFromJSON reads accounts/characters from a JSON store file and inserts
// them into SQLite. It is a no-op if the JSON file does not exist.
func importFromJSON(db *sql.DB, jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var f storeFileFormat
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	now := time.Now().Unix()
	for _, fa := range f.Accounts {
		res, err := db.Exec(
			`INSERT INTO account (username, password, password_salt, email, created_timestamp, flags)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			fa.Name, fa.Password, "", "", now, fa.Permission,
		)
		if err != nil {
			return err
		}
		accID, _ := res.LastInsertId()
		for _, fc := range fa.Characters {
			res, err := db.Exec(
				`INSERT INTO character (account_id, nickname, created_timestamp) VALUES (?, ?, ?)`,
				accID, fc.Name, now,
			)
			if err != nil {
				return err
			}
			charID, _ := res.LastInsertId()
			// Insert stat row (trigger creates one with level=1; overwrite with JSON values).
			_, err = db.Exec(
				`UPDATE stat SET level=?, experience=?, gp=?, health=?, mana=?, stamina=?, karma=? WHERE char_id=?`,
				fc.Level, fc.Experience, fc.GP, fc.Health, fc.Mana, fc.Stamina, fc.Karma, charID,
			)
			if err != nil {
				return err
			}
			// Insert trait row.
			_, err = db.Exec(
				`UPDATE trait SET race=?, gender=?, hairstyle=?, haircolor=?, skintone=? WHERE char_id=?`,
				fc.Race, fc.Gender, fc.Hairstyle, fc.Haircolor, fc.Skintone, charID,
			)
			if err != nil {
				return err
			}
			// Insert equipment row from JSON equipment list.
			eqMap := make(map[int64]ItemRecord)
			for _, eq := range fc.Equipment {
				eqMap[eq.Slot] = ItemRecord{ItemID: eq.ItemID, Customfield: eq.Customfield}
			}
			_, err = db.Exec(
				`UPDATE equipment SET
					chest=?, legs=?, feet=?, hands=?, head=?, neck=?, weapon=?, shield=?,
					chestCustom=?, legsCustom=?, feetCustom=?, handsCustom=?, headCustom=?, neckCustom=?, weaponCustom=?, shieldCustom=?
				 WHERE char_id=?`,
				sqlInt64(eqMap[0].ItemID), sqlInt64(eqMap[1].ItemID), sqlInt64(eqMap[2].ItemID),
				sqlInt64(eqMap[3].ItemID), sqlInt64(eqMap[4].ItemID), sqlInt64(eqMap[5].ItemID),
				sqlInt64(eqMap[6].ItemID), sqlInt64(eqMap[7].ItemID),
				eqMap[0].Customfield, eqMap[1].Customfield, eqMap[2].Customfield,
				eqMap[3].Customfield, eqMap[4].Customfield, eqMap[5].Customfield,
				eqMap[6].Customfield, eqMap[7].Customfield,
				charID,
			)
			if err != nil {
				return err
			}
			// Insert inventory items (storage=0 for inventory).
			for _, it := range fc.Inventory {
				_, err = db.Exec(
					`INSERT INTO item (item_id, char_id, count, storage, customfield) VALUES (?, ?, ?, 0, ?)`,
					it.ItemID, charID, it.Count, it.Customfield,
				)
				if err != nil {
					return err
				}
			}
		}
	}
	log.Printf("sqlite: imported %d account(s) from %s", len(f.Accounts), jsonPath)
	return nil
}

// sqlInt64 converts an int64 to a nullable SQL value (0 becomes NULL).
// SQLite treats NULL as "no value" for optional columns.
func sqlInt64(v int64) interface{} {
	if v == 0 {
		return nil
	}
	return v
}

// loadFromSQLite reads all accounts and characters from SQLite into memory.
func loadFromSQLite(db *sql.DB) (*AccountStore, error) {
	s := NewAccountStore()
	s.db = db

	rows, err := db.Query(
		`SELECT a.account_id, a.username, a.password, a.flags,
		        c.char_id, c.nickname,
		        s.level, s.experience, s.gp, s.health, s.mana, s.stamina, s.karma,
		        t.race, t.gender, t.hairstyle, t.haircolor, t.skintone
		 FROM account a
		 LEFT JOIN character c ON c.account_id = a.account_id
		 LEFT JOIN stat s ON s.char_id = c.char_id
		 LEFT JOIN trait t ON t.char_id = c.char_id
		 ORDER BY a.account_id, c.char_id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accMap := make(map[int64]*Account)
	for rows.Next() {
		var accID int64
		var username, password string
		var flags sql.NullInt64
		var charID sql.NullInt64
		var nickname sql.NullString
		var level, experience, gp, health, mana, stamina, karma sql.NullInt64
		var race, gender, hairstyle, haircolor, skintone sql.NullInt64

		if err := rows.Scan(
			&accID, &username, &password, &flags,
			&charID, &nickname,
			&level, &experience, &gp, &health, &mana, &stamina, &karma,
			&race, &gender, &hairstyle, &haircolor, &skintone,
		); err != nil {
			return nil, err
		}

		acc, ok := accMap[accID]
		if !ok {
			acc = &Account{
				Name:       username,
				Password:   password,
				Permission: int(flags.Int64),
			}
			accMap[accID] = acc
			s.accounts[username] = acc
		}

		if charID.Valid {
			c := &Character{
				ID:         int64(charID.Int64),
				Name:       nickname.String,
				Level:      level.Int64,
				Race:       race.Int64,
				Gender:     gender.Int64,
				Hairstyle:  hairstyle.Int64,
				Haircolor:  haircolor.Int64,
				Skintone:   skintone.Int64,
				Experience: experience.Int64,
				GP:         gp.Int64,
				Health:     health.Int64,
				Mana:       mana.Int64,
				Stamina:    stamina.Int64,
				Karma:      karma.Int64,
				Inventory:  []ItemRecord{},
				Equipment:  map[int64]ItemRecord{},
			}
			acc.Characters = append(acc.Characters, c)
			if c.ID >= s.nextCharID {
				s.nextCharID = c.ID + 1
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load tokens.
	tokRows, err := db.Query(
		`SELECT a.username, t.token
		 FROM auth_token t JOIN account a ON a.account_id = t.account_id`,
	)
	if err != nil {
		return nil, err
	}
	defer tokRows.Close()
	for tokRows.Next() {
		var username, token string
		if err := tokRows.Scan(&username, &token); err != nil {
			return nil, err
		}
		if acc, ok := s.accounts[username]; ok {
			acc.Token = token
		}
	}
	if err := tokRows.Err(); err != nil {
		return nil, err
	}

	// Load inventory items.
	invRows, err := db.Query(
		`SELECT a.username, i.item_id, i.count, i.customfield
		 FROM item i
		 JOIN character c ON c.char_id = i.char_id
		 JOIN account a ON a.account_id = c.account_id
		 WHERE i.storage = 0`,
	)
	if err != nil {
		return nil, err
	}
	defer invRows.Close()
	for invRows.Next() {
		var username string
		var itemID, count int64
		var customfield string
		if err := invRows.Scan(&username, &itemID, &count, &customfield); err != nil {
			return nil, err
		}
		if acc, ok := s.accounts[username]; ok {
			for _, c := range acc.Characters {
				c.Inventory = append(c.Inventory, ItemRecord{
					ItemID:      itemID,
					Customfield: customfield,
					Count:       count,
				})
			}
		}
	}
	if err := invRows.Err(); err != nil {
		return nil, err
	}

	// Load equipment.
	eqRows, err := db.Query(
		`SELECT a.username, e.char_id,
		        e.weapon, e.shield, e.ammunition, e.hands, e.chest, e.neck,
		        e.feet, e.head, e.legs, e.accessory1, e.accessory2,
		        e.weaponCustom, e.shieldCustom, e.ammunitionCustom,
		        e.handsCustom, e.chestCustom, e.neckCustom,
		        e.feetCustom, e.headCustom, e.legsCustom,
		        e.accessory1Custom, e.accessory2Custom
		 FROM equipment e
		 JOIN character c ON c.char_id = e.char_id
		 JOIN account a ON a.account_id = c.account_id`,
	)
	if err != nil {
		return nil, err
	}
	defer eqRows.Close()
	for eqRows.Next() {
		var username string
		var charID int64
		var weapon, shield, ammunition, hands, chest, neck, feet, head, legs, acc1, acc2 sql.NullInt64
		var weaponC, shieldC, ammoC, handsC, chestC, neckC, feetC, headC, legsC, acc1C, acc2C sql.NullString
		if err := eqRows.Scan(
			&username, &charID,
			&weapon, &shield, &ammunition, &hands, &chest, &neck,
			&feet, &head, &legs, &acc1, &acc2,
			&weaponC, &shieldC, &ammoC, &handsC, &chestC, &neckC,
			&feetC, &headC, &legsC, &acc1C, &acc2C,
		); err != nil {
			return nil, err
		}
		if acc, ok := s.accounts[username]; ok {
			for _, c := range acc.Characters {
				if c.ID == charID {
					setSlot := func(slot int64, id sql.NullInt64, custom sql.NullString) {
						if id.Valid && id.Int64 != 0 {
							c.Equipment[slot] = ItemRecord{
								ItemID:      id.Int64,
								Customfield: custom.String,
							}
						}
					}
					setSlot(0, chest, chestC)
					setSlot(1, legs, legsC)
					setSlot(2, feet, feetC)
					setSlot(3, hands, handsC)
					setSlot(4, head, headC)
					setSlot(5, neck, neckC)
					setSlot(6, weapon, weaponC)
					setSlot(7, shield, shieldC)
				}
			}
		}
	}
	if err := eqRows.Err(); err != nil {
		return nil, err
	}

	return s, nil
}

// ---- write-through helpers ----

// dbAddAccount inserts an account into SQLite.
func (s *AccountStore) dbAddAccount(acc *Account) {
	if s.db == nil {
		return
	}
	_, err := s.db.Exec(
		`INSERT INTO account (username, password, password_salt, email, created_timestamp, flags)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		acc.Name, acc.Password, "", "", time.Now().Unix(), acc.Permission,
	)
	if err != nil {
		log.Printf("sqlite: add account %q: %v", acc.Name, err)
	}
}

// dbAddCharacter inserts a character into SQLite.
func (s *AccountStore) dbAddCharacter(acc *Account, c *Character) {
	if s.db == nil {
		return
	}
	// Find account ID.
	var accID int64
	err := s.db.QueryRow(`SELECT account_id FROM account WHERE username = ?`, acc.Name).Scan(&accID)
	if err != nil {
		log.Printf("sqlite: find account %q: %v", acc.Name, err)
		return
	}
	res, err := s.db.Exec(
		`INSERT INTO character (account_id, nickname, created_timestamp) VALUES (?, ?, ?)`,
		accID, c.Name, time.Now().Unix(),
	)
	if err != nil {
		log.Printf("sqlite: add character %q: %v", c.Name, err)
		return
	}
	charID, _ := res.LastInsertId()
	c.ID = charID
	// The trigger creates stat/trait/attribute/equipment rows; update them.
	_, err = s.db.Exec(
		`UPDATE stat SET level=?, experience=?, gp=?, health=?, mana=?, stamina=?, karma=? WHERE char_id=?`,
		c.Level, c.Experience, c.GP, c.Health, c.Mana, c.Stamina, c.Karma, charID,
	)
	if err != nil {
		log.Printf("sqlite: update stat for char %q: %v", c.Name, err)
	}
	_, err = s.db.Exec(
		`UPDATE trait SET race=?, gender=?, hairstyle=?, haircolor=?, skintone=? WHERE char_id=?`,
		c.Race, c.Gender, c.Hairstyle, c.Haircolor, c.Skintone, charID,
	)
	if err != nil {
		log.Printf("sqlite: update trait for char %q: %v", c.Name, err)
	}
}

// dbRemoveCharacter deletes a character from SQLite.
func (s *AccountStore) dbRemoveCharacter(c *Character) {
	if s.db == nil {
		return
	}
	_, err := s.db.Exec(`DELETE FROM character WHERE char_id = ?`, c.ID)
	if err != nil {
		log.Printf("sqlite: remove character %q: %v", c.Name, err)
	}
}

// dbSetToken updates the auth token in SQLite.
func (s *AccountStore) dbSetToken(acc *Account, token string) {
	if s.db == nil {
		return
	}
	var accID int64
	err := s.db.QueryRow(`SELECT account_id FROM account WHERE username = ?`, acc.Name).Scan(&accID)
	if err != nil {
		log.Printf("sqlite: find account %q: %v", acc.Name, err)
		return
	}
	_, err = s.db.Exec(
		`INSERT OR REPLACE INTO auth_token (account_id, token, created_timestamp) VALUES (?, ?, ?)`,
		accID, token, time.Now().Unix(),
	)
	if err != nil {
		log.Printf("sqlite: set token for %q: %v", acc.Name, err)
	}
}
