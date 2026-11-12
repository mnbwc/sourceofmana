package main

// Data-driven quest framework.
//
// Mirrors SoM's quest model (sources/actor/Progress.gd, sources/actor/ProgressCommons.gd):
// every character carries an integer state per quest. 0 = not started,
// 1..N = the stage the player is currently on, 255 = completed (SoM's
// REWARDS_WITREW convention). States are persisted through the AccountStore
// and pushed with UpdateQuest.
//
// Quest definitions are plain data: each stage carries the line the NPC says
// (optionally overridden per NPC, so a chain can hop between characters), an
// optional requirement (hand in items / kill a monster / talk to someone /
// nothing), and a reward. The dialogue engine in npc.go renders them.

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
)

// questStateDone mirrors ProgressCommons.*.REWARDS_WITREW: SoM marks a
// finished quest with 255 rather than deleting the entry.
const questStateDone int64 = 255

// questItem is one item requirement or reward entry.
type questItem struct {
	ItemID int64
	Count  int64
}

// questReq is a stage requirement. Kind is "", "item", "kill" or "talk".
// An empty Kind means the stage is pure dialogue / auto-advances.
type questReq struct {
	Kind      string
	Items     []questItem
	Monster   string // NPC nick, for Kind == "kill"
	KillCount int64
	TalkNPC   string // NPC nick, for Kind == "talk"
}

// questReward is what a stage pays out when it is completed.
type questReward struct {
	Exp   int64
	GP    int64
	Karma int64
	Items []questItem
}

// questStage is one step of a quest.
type questStage struct {
	// Text is the line shown at this stage. TextByNPC overrides it when the
	// stage is reached at a specific NPC (quest chains hop between NPCs).
	Text      string
	TextByNPC map[string]string
	// Req must be satisfied before the stage can be turned in.
	Req questReq
	// Reward is granted when the stage completes.
	Reward questReward
	// Auto advances the stage as soon as it is shown and its requirement is
	// satisfied, without waiting for the player to pick a choice. Used by
	// vendors and by "just talk to them" chain steps.
	Auto bool
}

// questDef is one quest.
type questDef struct {
	ID          int64
	Name        string
	Description string
	Giver       string // NPC nick that offers the quest
	Repeatable  bool   // re-offered after completion (vendors, gathering)
	Hidden      bool   // helpers that do not appear in the quest log
	// Starter is handed over when the player accepts the quest.
	Starter []questItem
	Stages  []questStage
}

// ---- quest id hashing ----

// djb2String mirrors Godot's String.hash(): h = 5381; h = h*33 + c (uint32).
// Both item and quest IDs in SoM are String.hash() of the name, so this is the
// canonical way to refer to a quest by name.
func djb2String(s string) uint32 {
	var h uint32 = 5381
	for i := 0; i < len(s); i++ {
		h = h*33 + uint32(s[i])
	}
	return h
}

// questID returns the canonical quest id for a quest name.
func questID(name string) int64 { return int64(djb2String(name)) }

// ---- catalog ----

// itemID returns the canonical item id for an item name.
func itemID(name string) int64 { return int64(djb2String(name)) }

// qi is a compact questItem literal.
func qi(name string, count int64) questItem { return questItem{ItemID: itemID(name), Count: count} }

// questCatalog is the built-in quest database, keyed by canonical quest id.
// The 17 entries mirror tools/export/quests.json; two hidden vendor quests
// cover the supplies those quests ask for (Croissant, Water Bottle).
var questCatalog = map[int64]*questDef{}

func init() {
	for _, q := range builtinQuestDefs {
		questCatalog[q.ID] = q
	}
}

var builtinQuestDefs = []*questDef{
	// ---- Tutorial chain: Elanore -> Kael -> Ekinu ----
	{
		ID:          questID("Tutorial"),
		Name:        "Tutorial",
		Description: "艾拉诺带你熟悉 Tulimshar：认识守卫凯尔，再去找艾基努。",
		Giver:       "Elanore",
		Starter: []questItem{
			qi("Water Bottle", 1),
			qi("Cactus Sour Candy", 1),
			qi("Cotton Shirt", 1),
			qi("Shorts", 1),
		},
		Stages: []questStage{
			{
				Text: "艾拉诺：欢迎来到 Tulimshar，旅人。沙漠里危险重重，拿着这些补给——一瓶水、一颗糖，还有能换洗的衣裳。",
				Auto: true,
			},
			{
				Text: "艾拉诺：守卫凯尔在东门附近巡逻。去找他吧，他会教你怎么在这地方活下来。",
				Req:  questReq{Kind: "talk", TalkNPC: "Kael"},
			},
			{
				TextByNPC: map[string]string{
					"Kael": "凯尔：艾拉诺把你派来了？很好。拿上这几件装备，别在沙漠里丢了小命。",
				},
				Text: "凯尔：艾拉诺把你派来了？很好。拿上这几件装备，别在沙漠里丢了小命。",
				Req:  questReq{Kind: "talk", TalkNPC: "Ekinu"},
			},
			{
				TextByNPC: map[string]string{
					"Ekinu": "艾基努：凯尔跟我提过你。既然你已经准备好了，拿着这份酬劳，去闯你自己的路吧。",
				},
				Text:   "艾基努：既然你已经准备好了，拿着这份酬劳，去闯你自己的路吧。",
				Auto:   true,
				Reward: questReward{Exp: 20, GP: 50, Items: []questItem{qi("Desert Boots", 1), qi("Desert Goggles", 1)}},
			},
		},
	},

	// ---- Elanore's gathering quest (repeatable) ----
	{
		ID:          questID("Elanore's Potions"),
		Name:        "Elanore's Potions",
		Description: "帮艾拉诺收集 6 个 Maggot Slime、1 个 Water Bottle 和 1 个 Pitaya，换一瓶 Cactus Potion。",
		Giver:       "Elanore",
		Repeatable:  true,
		Stages: []questStage{
			{
				Text: "艾拉诺：我的药剂快用完了。帮我收集 6 个 Maggot Slime、1 个 Water Bottle 和 1 个 Pitaya，我就给你一瓶 Cactus Potion。",
				Req: questReq{Kind: "item", Items: []questItem{
					qi("Maggot Slime", 6), qi("Water Bottle", 1), qi("Pitaya", 1),
				}},
				Reward: questReward{Items: []questItem{qi("Cactus Potion", 1)}},
			},
		},
	},

	// ---- Nina's Snack ----
	{
		ID:          questID("Nina's Snack"),
		Name:        "Nina's Snack",
		Description: "妮娜饿了，去市场给她买一个 Croissant。",
		Giver:       "Nina",
		Stages: []questStage{
			{
				Text: "妮娜：我快饿扁了……能去市场给我买个 Croissant 吗？求你了。",
				Req:  questReq{Kind: "item", Items: []questItem{qi("Croissant", 1)}},
				Reward: questReward{
					GP:    100,
					Items: []questItem{qi("Cactus Potion", 10)},
				},
			},
		},
	},

	// ---- A Drop of Relief ----
	{
		ID:          questID("A Drop of Relief"),
		Name:        "A Drop of Relief",
		Description: "内森在沙暴矿井外被烤得受不了，给他一瓶 Water Bottle。",
		Giver:       "Nathan",
		Stages: []questStage{
			{
				Text:   "内森：这鬼天气要把我烤干了……给我一瓶 Water Bottle 好吗？不用喝，浇在身上就行。",
				Req:    questReq{Kind: "item", Items: []questItem{qi("Water Bottle", 1)}},
				Reward: questReward{Exp: 50, GP: 100},
			},
		},
	},

	// ---- An Old Friendship ----
	{
		ID:          questID("An Old Friendship"),
		Name:        "An Old Friendship",
		Description: "弗罗斯特想请你去找本——他就在 Tulimshar 西墙步道。",
		Giver:       "Frost",
		Stages: []questStage{
			{
				Text:   "弗罗斯特：我老了，没法再回城堡走廊拿回那些信了。帮我去找本，他就在西墙步道。",
				Req:    questReq{Kind: "talk", TalkNPC: "Ben"},
				Reward: questReward{GP: 1000},
			},
		},
	},

	// ---- Sandstorm Mine Abandoned Treasure ----
	{
		ID:          questID("Sandstorm Mine Abandoned Treasure"),
		Name:        "Sandstorm Mine Abandoned Treasure",
		Description: "一把生锈的矿井钥匙，指向废弃矿井里的宝箱。",
		Giver:       "Mine Key",
		Stages: []questStage{
			{
				Text:   "（一把生锈的 Mine Key，匙柄上刻着：'废弃矿井的宝藏'。）拿去打开深处那个旧箱子吧。",
				Req:    questReq{Kind: "talk", TalkNPC: "Abandoned Chest"},
				Reward: questReward{Items: []questItem{qi("Piou Slayer", 1)}},
			},
		},
	},

	// ---- Snake Pit Thief ----
	{
		ID:          questID("Snake Pit Thief"),
		Name:        "Snake Pit Thief",
		Description: "顺着蛇坑线索找到盗贼的宝箱。",
		Giver:       "Snake Pit Clue",
		Stages: []questStage{
			{
				Text:   "（一张皱巴巴的纸条：'蛇坑深处，盗贼把抢来的东西全藏进了宝箱。'）",
				Req:    questReq{Kind: "talk", TalkNPC: "Thiefs Chest"},
				Reward: questReward{GP: 200, Items: []questItem{qi("Snake Skin", 2)}},
			},
		},
	},

	// ---- A Miner's Helping Hand ----
	{
		ID:          questID("A Miner's Helping Hand"),
		Name:        "A Miner's Helping Hand",
		Description: "纳姆愿意把备用的 Miner Helmet 交给你，让你安全深入地底。",
		Giver:       "Naem",
		Stages: []questStage{
			{
				Text:   "纳姆：你要下到矿井深处？带上这顶备用的 Miner Helmet，能保你一命。",
				Auto:   true,
				Reward: questReward{Exp: 100, Items: []questItem{qi("Miner Helmet", 1)}},
			},
		},
	},

	// ---- Mine Exploration ----
	{
		ID:          questID("Mine Exploration"),
		Name:        "Mine Exploration",
		Description: "沙暴矿井深处有奇怪的动静：先问尼克os，再去见魔力树。",
		Giver:       "Ekinu",
		Stages: []questStage{
			{
				Text: "艾基努：沙暴矿井深处最近有奇怪的动静。去和尼克os 谈谈，他昨天刚进去过。",
				Req:  questReq{Kind: "talk", TalkNPC: "Nickos"},
			},
			{
				Text: "尼科斯：我在深处看到了一棵……会发光的树？你去见见 Mana Tree 吧，就在矿井最深处。",
				Req:  questReq{Kind: "talk", TalkNPC: "Mana Tree"},
			},
			{
				Text:   "（魔力树的枝条轻轻摆动，仿佛在欢迎你。）它把一小团温润的魔力放进你的手心。",
				Auto:   true,
				Reward: questReward{Exp: 200, GP: 300},
			},
		},
	},

	// ---- Grain in the desert ----
	{
		ID:          questID("Grain in the desert"),
		Name:        "Grain in the desert",
		Description: "里斯基姆的面粉桶空了，替他去 Tulimshar 湾看看。",
		Giver:       "Riskim",
		Stages: []questStage{
			{
				Text:   "里斯基姆：我的面粉桶空了，可我不能离开摊位。替我去 Tulimshar 湾的 Flour Barrel 看看，带点面粉回来。",
				Req:    questReq{Kind: "talk", TalkNPC: "Flour Barrel"},
				Reward: questReward{GP: 150, Items: []questItem{qi("Artis Flour Sack", 1)}},
			},
		},
	},

	// ---- Desert Seed ----
	{
		ID:          questID("Desert Seed"),
		Name:        "Desert Seed",
		Description: "魔力树想要 3 个 Pitaya，作为交换它会送你一颗沙漠的种子。",
		Giver:       "Mana Tree",
		Repeatable:  true,
		Stages: []questStage{
			{
				Text:   "（魔力树的意念在你脑海中响起：'把 3 个 Pitaya 带来，我便赠你一颗沙漠的种子。'）",
				Req:    questReq{Kind: "item", Items: []questItem{qi("Pitaya", 3)}},
				Reward: questReward{Exp: 100, GP: 150},
			},
		},
	},

	// ---- Splatyna's Offering ----
	{
		ID:          questID("Splatyna's Offering"),
		Name:        "Splatyna's Offering",
		Description: "比西斯请你去 Splatyna 的洞穴拜见史莱普蒂娜大人。",
		Giver:       "Bicies",
		Stages: []questStage{
			{
				Text:   "比西斯：史莱普蒂娜大人就在前面的洞里。带上你的敬意去见她吧。",
				Req:    questReq{Kind: "talk", TalkNPC: "Splatyna"},
				Reward: questReward{Exp: 100, GP: 200},
			},
		},
	},

	// ---- Peter's Bounty ----
	{
		ID:          questID("Peter's Bounty"),
		Name:        "Peter's Bounty",
		Description: "船舱里的 Ratto 成灾了，彼得请你杀掉 5 只。",
		Giver:       "Peter",
		Stages: []questStage{
			{
				Text:   "彼得：船舱里的 Ratto 成灾了！帮我杀掉 5 只，酬劳少不了你的。",
				Req:    questReq{Kind: "kill", Monster: "Ratto", KillCount: 5},
				Reward: questReward{Exp: 100, GP: 500},
			},
		},
	},

	// ---- When the desert gives you sand... ----
	{
		ID:          questID("When the desert gives you sand..."),
		Name:        "When the desert gives you sand...",
		Description: "埃里杜想要 12 份 Salt、12 根 Bone 和 12 份 Sulphur Powder。",
		Giver:       "Eridu",
		Stages: []questStage{
			{
				Text: "埃里杜：沙漠把一切都变成了沙子。给我 12 份 Salt、12 根 Bone 和 12 份 Sulphur Powder，我教你酿 Bottle。",
				Req: questReq{Kind: "item", Items: []questItem{
					qi("Salt", 12), qi("Bone", 12), qi("Sulphur Powder", 12),
				}},
				Reward: questReward{Exp: 100, Items: []questItem{qi("Bottle", 3)}},
			},
		},
	},

	// ---- Found a Chest in the Outskirts ----
	{
		ID:          questID("Found a Chest in the Outskirts"),
		Name:        "Found a Chest in the Outskirts",
		Description: "Tulimshar 东丘有一个被沙土半埋的 Chest。",
		Giver:       "Chest",
		Stages: []questStage{
			{
				Text:   "（一个被沙土半埋的 Chest，锁已经锈住了。你用力一掀——里面躺着一面 Desert Shield！）",
				Auto:   true,
				Reward: questReward{Items: []questItem{qi("Desert Shield", 1)}},
			},
		},
	},

	// ---- Veteran Cave Explorer ----
	{
		ID:          questID("Veteran Cave Explorer"),
		Name:        "Veteran Cave Explorer",
		Description: "洞穴里的 Snake 越来越暴躁，比西斯请你清理 8 条。",
		Giver:       "Bicies",
		Stages: []questStage{
			{
				Text:   "比西斯：洞穴深处的 Snake 最近越来越暴躁。替我清理掉 8 条，让过路人安全些。",
				Req:    questReq{Kind: "kill", Monster: "Snake", KillCount: 8},
				Reward: questReward{Exp: 300, GP: 250},
			},
		},
	},

	// ---- A Biting Thirst ----
	{
		ID:          questID("A Biting Thirst"),
		Name:        "A Biting Thirst",
		Description: "毛罗在蛇坑被蛇咬了，水全洒了——替他去 Water Point 打点水。",
		Giver:       "Mauro",
		Stages: []questStage{
			{
				Text:   "毛罗：我在 Snake Pit 的池塘边被蛇咬了，水全洒了……能替我去 Water Point 打点水吗？",
				Req:    questReq{Kind: "talk", TalkNPC: "Water Point"},
				Reward: questReward{Exp: 50, GP: 50},
			},
		},
	},

	// ---- Hidden vendor helpers ----------------------------------------------
	// These exist so the item requirements above are reachable: Nina's Snack
	// needs a Croissant, A Drop of Relief / Elanore's Potions need a Water
	// Bottle. They are repeatable, free, and hidden from the quest log.

	{
		ID:          questID("Bread for Nina"),
		Name:        "Bread for Nina",
		Description: "面包师傅刚出炉的 Croissant。",
		Giver:       "Market Baker",
		Repeatable:  true,
		Hidden:      true,
		Stages: []questStage{
			{
				Text:   "面包师傅：刚出炉的 Croissant！来一个吗？",
				Auto:   true,
				Reward: questReward{Items: []questItem{qi("Croissant", 1)}},
			},
		},
	},
	{
		ID:          questID("Water for the Road"),
		Name:        "Water for the Road",
		Description: "吉尔伽美什随身带的 Water Bottle。",
		Giver:       "Gilgames",
		Repeatable:  true,
		Hidden:      true,
		Stages: []questStage{
			{
				Text:   "吉尔伽美什：远行的人都需要水。拿一瓶 Water Bottle 吧。",
				Auto:   true,
				Reward: questReward{Items: []questItem{qi("Water Bottle", 1)}},
			},
		},
	},
}

// questJSON mirrors one entry of tools/export/quests.json.
type questJSON struct {
	File           string `json:"file"`
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Giver          string `json:"giver"`
	GiverLocation  string `json:"giverLocation"`
	Target         string `json:"target"`
	TargetLocation string `json:"targetLocation"`
	Reward         string `json:"reward"`
}

// loadQuestCatalogFromJSON merges metadata from the exported quests.json into
// the built-in catalog: name, description and giver. Quest ids and stage data
// come from the code above, so the server still works if the file is absent.
// Returns the number of entries loaded.
func loadQuestCatalogFromJSON(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	// tools/export/quests.json is a bare array, while the file written by
	// dumpQuestCatalog is an object with a "quests" key. Accept both so the
	// server can re-read its own export.
	var entries []questJSON
	if err := json.Unmarshal(data, &entries); err != nil {
		var wrapper struct {
			Quests []questJSON `json:"quests"`
		}
		if wrapErr := json.Unmarshal(data, &wrapper); wrapErr != nil {
			return 0, err
		}
		entries = wrapper.Quests
	}
	n := 0
	for _, e := range entries {
		if e.ID == 0 || e.Name == "" {
			continue
		}
		q := questCatalog[e.ID]
		if q == nil {
			// Unknown quest in the export: register a stub so the client and
			// server agree on the id.
			q = &questDef{ID: e.ID, Name: e.Name, Description: e.Description, Giver: e.Giver}
			questCatalog[e.ID] = q
		}
		if e.Name != "" {
			q.Name = e.Name
		}
		if e.Description != "" {
			q.Description = strings.Trim(e.Description, "\"")
		}
		if e.Giver != "" {
			q.Giver = e.Giver
		}
		n++
	}
	return n, nil
}

// questDef returns the definition for a quest id.
func (w *npcWorld) questDef(questID int64) *questDef {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.catalog[questID]
}

// ---- client catalog dump -------------------------------------------------

// The lite human client is a standalone Python program: it cannot import this
// package, so the merged catalog is written out as JSON next to it. The
// client uses it to render the quest log, the side-panel tracker and the NPC
// markers. Regenerate with: SOM_DUMP_QUESTS=quests.json ./goserver

type dumpItem struct {
	ID    int64 `json:"id"`
	Count int64 `json:"count"`
}

type dumpReq struct {
	Kind       string     `json:"kind"`
	Items      []dumpItem `json:"items,omitempty"`
	Monster    string     `json:"monster,omitempty"`
	MonsterRID int64      `json:"monsterRid,omitempty"`
	KillCount  int64      `json:"killCount,omitempty"`
	TalkNPC    string     `json:"talkNpc,omitempty"`
}

type dumpReward struct {
	Exp   int64      `json:"exp"`
	GP    int64      `json:"gp"`
	Karma int64      `json:"karma"`
	Items []dumpItem `json:"items,omitempty"`
}

type dumpStage struct {
	Text      string            `json:"text"`
	TextByNPC map[string]string `json:"textByNpc,omitempty"`
	Auto      bool              `json:"auto"`
	Req       dumpReq           `json:"req"`
	Reward    dumpReward        `json:"reward"`
}

type dumpQuest struct {
	ID          int64       `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Giver       string      `json:"giver"`
	Repeatable  bool        `json:"repeatable"`
	Hidden      bool        `json:"hidden"`
	Starter     []dumpItem  `json:"starter,omitempty"`
	Stages      []dumpStage `json:"stages"`
}

func dumpItems(in []questItem) []dumpItem {
	if len(in) == 0 {
		return nil
	}
	out := make([]dumpItem, 0, len(in))
	for _, it := range in {
		out = append(out, dumpItem{ID: it.ItemID, Count: it.Count})
	}
	return out
}

// rosterRIDs maps an NPC nick to its lowest rid, matching npcByNickLocked
// so the client's kill-quest progress uses the same key as trackKill.
func rosterRIDs() map[string]int64 {
	out := make(map[string]int64)
	for _, n := range spawnNpcs {
		if cur, ok := out[n.nick]; !ok || n.rid < cur {
			out[n.nick] = n.rid
		}
	}
	for i, m := range fieldMonsters {
		rid := monsterRIDBase + int64(i)
		if cur, ok := out[m.nick]; !ok || rid < cur {
			out[m.nick] = rid
		}
	}
	return out
}

// dumpQuestCatalog writes the merged catalog (built-ins + quests.json) as JSON.
func dumpQuestCatalog(path string) error {
	ids := make([]int64, 0, len(questCatalog))
	for id := range questCatalog {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	roster := rosterRIDs()
	out := make([]dumpQuest, 0, len(ids))
	for _, id := range ids {
		q := questCatalog[id]
		dq := dumpQuest{
			ID: q.ID, Name: q.Name, Description: q.Description,
			Giver: q.Giver, Repeatable: q.Repeatable, Hidden: q.Hidden,
			Starter: dumpItems(q.Starter),
		}
		for _, st := range q.Stages {
			dq.Stages = append(dq.Stages, dumpStage{
				Text:      st.Text,
				TextByNPC: st.TextByNPC,
				Auto:      st.Auto,
				Req: dumpReq{
					Kind:       st.Req.Kind,
					Items:      dumpItems(st.Req.Items),
					Monster:    st.Req.Monster,
					MonsterRID: roster[st.Req.Monster],
					KillCount:  st.Req.KillCount,
					TalkNPC:    st.Req.TalkNPC,
				},
				Reward: dumpReward{
					Exp:   st.Reward.Exp,
					GP:    st.Reward.GP,
					Karma: st.Reward.Karma,
					Items: dumpItems(st.Reward.Items),
				},
			})
		}
		out = append(out, dq)
	}
	data, err := json.MarshalIndent(map[string]any{"quests": out}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// ---- requirement checks ----

// countItem sums the stacks of itemID in the character's inventory.
func countItem(chr *Character, itemID int64) int64 {
	var total int64
	for _, it := range chr.Inventory {
		if it.ItemID == itemID && it.Customfield == "" {
			total += it.Count
		}
	}
	return total
}

// stageReqSatisfied reports whether the character meets stage k's requirement
// at the given NPC. It also returns a short human-readable hint listing what
// is missing.
func (s *Server) stageReqSatisfied(chr *Character, q *questDef, k int, npcNick string) (bool, string) {
	if chr == nil || k < 1 || k > len(q.Stages) {
		return false, ""
	}
	req := q.Stages[k-1].Req
	switch req.Kind {
	case "":
		return true, ""
	case "talk":
		// Fulfilled by interacting with the target NPC; the caller compares
		// TalkNPC against the NPC the player is actually talking to.
		return req.TalkNPC == npcNick, ""
	case "item":
		var missing []string
		ok := true
		for _, it := range req.Items {
			have := countItem(chr, it.ItemID)
			if have < it.Count {
				ok = false
				missing = append(missing, itemName(it.ItemID))
			}
		}
		return ok, strings.Join(missing, "、")
	case "kill":
		npc := s.npcs.npcByNick(req.Monster)
		if npc == nil {
			return false, req.Monster
		}
		have := s.progress.GetBestiary(chr.ID, npc.RID)
		if have < req.KillCount {
			return false, req.Monster
		}
		return true, ""
	}
	return false, ""
}

// itemName resolves an item id to its catalog name (or the raw id).
func itemName(itemID int64) string {
	if cell := itemCatalog[itemID]; cell != nil {
		return cell.Name
	}
	return itoa(itemID)
}

// ---- state transitions ----

// acceptQuest starts a quest: hands over the starter items and sets state 1.
func (s *Server) acceptQuest(c *Client, q *questDef) {
	charID := c.session.charID
	chr := s.store.findCharacterByID(charID)
	if chr != nil && len(q.Starter) > 0 {
		s.store.mu.Lock()
		for _, it := range q.Starter {
			s.addItem(chr, it.ItemID, "", it.Count)
		}
		s.store.mu.Unlock()
	}
	s.progress.SetQuest(charID, q.ID, 1)
	s.persist()
	s.pushQuestUpdate(c, q.ID, 1)
	if chr != nil {
		if e := s.world.get(charID); e != nil {
			if oc := s.clientByPeer(e.PeerID); oc != nil {
				for _, it := range q.Starter {
					sendToClient(oc, nil, rpcItemAdded, []Variant{
						NewInt(it.ItemID), NewStringName(""), NewInt(it.Count),
					})
				}
			}
		}
	}
	log.Printf("quest: char %d accepted %s (%d)", charID, q.Name, q.ID)
}

// completeStage finishes stage k: takes the required items, pays the reward
// and advances the state. Returns the new state (255 = done).
func (s *Server) completeStage(c *Client, q *questDef, k int) int64 {
	charID := c.session.charID
	st := q.Stages[k-1]
	chr := s.store.findCharacterByID(charID)
	if chr != nil && len(st.Req.Items) > 0 {
		s.store.mu.Lock()
		for _, it := range st.Req.Items {
			removeItem(chr, it.ItemID, "", it.Count)
		}
		s.store.mu.Unlock()
		// The client only learns about inventory changes through pushes, so
		// without this it would keep showing the consumed items.
		if e := s.world.get(charID); e != nil {
			if oc := s.clientByPeer(e.PeerID); oc != nil {
				sendToClient(oc, nil, rpcRefreshInventory,
					[]Variant{inventoryArray(chr)})
			}
		}
	}
	s.grantQuestReward(c, charID, st.Reward)
	next := int64(k + 1)
	if next > int64(len(q.Stages)) {
		next = questStateDone
	}
	s.progress.SetQuest(charID, q.ID, next)
	s.persist()
	s.pushQuestUpdate(c, q.ID, next)
	log.Printf("quest: char %d %s stage %d -> state %d", charID, q.Name, k, next)
	return next
}

// grantQuestReward pays out a reward: GP, karma, experience and items.
func (s *Server) grantQuestReward(c *Client, charID int64, r questReward) {
	chr := s.store.findCharacterByID(charID)
	if chr == nil {
		return
	}
	s.store.mu.Lock()
	if r.GP > 0 {
		chr.GP += r.GP
	}
	if r.Karma > 0 {
		chr.Karma += r.Karma
	}
	for _, it := range r.Items {
		s.addItem(chr, it.ItemID, "", it.Count)
	}
	s.store.mu.Unlock()

	levels := 0
	if r.Exp > 0 {
		levels = grantExperience(chr, r.Exp)
	}
	s.persist()
	if r.Exp > 0 {
		s.pushPrivateStats(charID)
	}
	if e := s.world.get(charID); e != nil {
		if levels > 0 {
			e.Level = chr.Level
			s.pushLevelUp(e)
		}
		if oc := s.clientByPeer(e.PeerID); oc != nil {
			for _, it := range r.Items {
				sendToClient(oc, nil, rpcItemAdded, []Variant{
					NewInt(it.ItemID), NewStringName(""), NewInt(it.Count),
				})
			}
		}
	}
	if r.GP > 0 {
		log.Printf("quest: char %d +%d GP", charID, r.GP)
	}
}

// pushQuestUpdate sends a single UpdateQuest to the client.
func (s *Server) pushQuestUpdate(c *Client, questID int64, state int64) {
	sendToClient(c, nil, rpcUpdateQuest, []Variant{NewInt(questID), NewInt(state)})
}

// stageText resolves the line for a stage at a given NPC.
func stageText(st questStage, npcNick string) string {
	if st.TextByNPC != nil {
		if t, ok := st.TextByNPC[npcNick]; ok && t != "" {
			return t
		}
	}
	return st.Text
}

// reqHint renders the outstanding requirement of a stage for reminders.
func (s *Server) reqHint(chr *Character, q *questDef, k int) string {
	if chr == nil || k < 1 || k > len(q.Stages) {
		return ""
	}
	req := q.Stages[k-1].Req
	switch req.Kind {
	case "item":
		parts := make([]string, 0, len(req.Items))
		for _, it := range req.Items {
			have := countItem(chr, it.ItemID)
			parts = append(parts, fmt.Sprintf("%s %d/%d", itemName(it.ItemID), have, it.Count))
		}
		return strings.Join(parts, "、")
	case "kill":
		npc := s.npcs.npcByNick(req.Monster)
		if npc == nil {
			return fmt.Sprintf("击杀 %s %d/%d", req.Monster, 0, req.KillCount)
		}
		have := s.progress.GetBestiary(chr.ID, npc.RID)
		return fmt.Sprintf("击杀 %s %d/%d", req.Monster, have, req.KillCount)
	case "talk":
		return fmt.Sprintf("去找 %s", req.TalkNPC)
	}
	return ""
}
