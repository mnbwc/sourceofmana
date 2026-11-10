# Source of Mana — 游戏策划文档

> 本文档由自动化工具从游戏资源文件中提取生成
>
> 生成时间: 2025-11-10

---

## 目录

1. [游戏任务数据](#1-游戏任务数据)
2. [游戏物品数据](#2-游戏物品数据)
3. [游戏技能数据](#3-游戏技能数据
4. [游戏实体数据](#4-游戏实体数据)
5. [游戏掉率数据](#5-游戏掉率数据)
6. [游戏状态机](#6-游戏状态机)
7. [NPC 对话数据](#7-npc-对话数据)
8. [附录](#8-附录)

---

## 1. 游戏任务数据

共 **17** 个任务

| # | 任务名称 | ID | 给予者 | 给予位置 | 目标 | 目标位置 | 奖励 |
|---|---------|-----|--------|---------|------|---------|------|
| 1 | A Biting Thirst | 2646362849 | Mauro | Desert Pit | Water point | Snake Pit | Unknown |
| 2 | A Drop of Relief | 1931438535 | Nathan | Sandstorm Mines | Water Bottle x1 | Tulimshar | 50 EXP, 100 GP |
| 3 | A Miner's Helping Hand | 2427453213 | Naem | Sandstorm Mines | Miner Helmet x1 | Sandstorm Mines | Miner Helmet |
| 4 | An Old Friendship | 1070913343 | Frost | Tulimshar Center | Ben | Tulimshar West Wall Pathway | 1000 GP |
| 5 | Desert Seed | 3532436717 | Mana Tree | Sandstorm Mines |  | Unknown | Unknown |
| 6 | Elanore's Potions | 3984490417 | Elanore | Tulimshar Gates | Maggot Slime x6, Water x1, Pitaya x1 | Tulimshar region | Cactus Potion |
| 7 | Found a Chest in the Outskirts | 4235969129 |  |  | Chest | Tulimshar Eastern Hills | Desert Shield |
| 8 | Grain in the desert | 2893494389 | Riskim | Tulmshar Center | Barrel | Tulimshar Bay | Unknown |
| 9 | Mine Exploration | 826825219 | Ekinu | Tulimshar |  | Sandstorm Mines | Unknown |
| 10 | Nina's Snack | 3098703797 | Nina | Tulimshar Gates | Croissant x1 | Tulimshar Market | Cactus Potion x10, 100 GP |
| 11 | Peter's Bounty | 2607006912 | Peter | Ship Second Deck | Clear the hold of Rattos | Ship Hold | 100 EXP, 500 GP |
| 12 | Sandstorm Mine Abandoned Treasure | 1083446768 | Mine Key | Abandoned Mine | Chest | Abandoned Mine | Piou Slayer |
| 13 | Snake Pit Thief | 647884116 | Snake Pit Clue | Snake Pit | Thiefs Chest | Snake Pit | Unknown |
| 14 | Splatyna's Offering | 665326971 | Bicies | Splatyna Cave Entrance | Splatyna | Splatyna's Chamber | Unknown |
| 15 | Tutorial | 1180319673 | Elanore | Tulimshar Gates |  |  | Unknown |
| 16 | Veteran Cave Explorer | 2409290506 |  |  |  |  | Unknown |
| 17 | When the desert gives you sand... | 2192634378 | Eridu | Tulimshar Center | Salt x12, Bone x12, Sulphur Powder x12 | Tonori Desert | Bottle x3, 100 EXP |

### 任务详情

#### A Biting Thirst

- **ID**: `2646362849`
- **描述**: "Mauro near a cave entrance tried to fetch water from a small pond inside, but some snakes bit him until he spilled and drank what little remained.
- **给予者**: Mauro @ Desert Pit
- **目标**: Water point @ Snake Pit
- **奖励**: Unknown

#### A Drop of Relief

- **ID**: `1931438535`
- **描述**: Nathan is stationed outside the Sandstorm Mines and suffering under the blazing heat. He doesn't need water to drink, just a bottle to pour over himself.
- **给予者**: Nathan @ Sandstorm Mines
- **目标**: Water Bottle x1 @ Tulimshar
- **奖励**: 50 EXP, 100 GP

#### A Miner's Helping Hand

- **ID**: `2427453213`
- **描述**: Naem, one of the workers in the Sandstorm Mines, offered you a spare mining helmet to keep you safe as you venture deeper.
- **给予者**: Naem @ Sandstorm Mines
- **目标**: Miner Helmet x1 @ Sandstorm Mines
- **奖励**: Miner Helmet

#### An Old Friendship

- **ID**: `1070913343`
- **描述**: Frost, a man sitting on a bench in Tulimshar, asks you to retrieve old letters from the castle corridor. Stay in the shadows to avoid the guards.
- **给予者**: Frost @ Tulimshar Center
- **目标**: Ben @ Tulimshar West Wall Pathway
- **奖励**: 1000 GP

#### Desert Seed

- **ID**: `3532436717`
- **描述**: Deep in the Sandstorm Mines, you discovered a living Mana Tree and recovered its Kano, a rare Mana Seed.
- **给予者**: Mana Tree @ Sandstorm Mines
- **目标**: (无) @ Unknown
- **奖励**: Unknown

#### Elanore's Potions

- **ID**: `3984490417`
- **描述**: Elanore needs ingredients to brew healing potions. Bring her six Maggot Slimes, one Water and one Pitaya.
- **给予者**: Elanore @ Tulimshar Gates
- **目标**: Maggot Slime x6, Water x1, Pitaya x1 @ Tulimshar region
- **奖励**: Cactus Potion

#### Found a Chest in the Outskirts

- **ID**: `4235969129`
- **描述**: Found a chest in the Tulimshar eastern outskirts.
- **给予者**: (无) @ (无)
- **目标**: Chest @ Tulimshar Eastern Hills
- **奖励**: Desert Shield

#### Grain in the desert

- **ID**: `2893494389`
- **描述**: Riskim the baker is waiting for a grain shipment marked with a blue Artis seal. Check the barrels at the docks.
- **给予者**: Riskim @ Tulmshar Center
- **目标**: Barrel @ Tulimshar Bay
- **奖励**: Unknown

#### Mine Exploration

- **ID**: `826825219`
- **描述**: Scout the Sandstorm Mines and ensure the cave system is clear of major dangers before the miners move in.
- **给予者**: Ekinu @ Tulimshar
- **目标**: (无) @ Sandstorm Mines
- **奖励**: Unknown

#### Nina's Snack

- **ID**: `3098703797`
- **描述**: Nina is hungry but too dedicated to leave her post. Find her a croissant from the market.
- **给予者**: Nina @ Tulimshar Gates
- **目标**: Croissant x1 @ Tulimshar Market
- **奖励**: Cactus Potion x10, 100 GP

#### Peter's Bounty

- **ID**: `2607006912`
- **描述**: A sailor aboard the ship is offering bounty contracts to clear rattos from the ship's hold.
- **给予者**: Peter @ Ship Second Deck
- **目标**: Clear the hold of Rattos @ Ship Hold
- **奖励**: 100 EXP, 500 GP

#### Sandstorm Mine Abandoned Treasure

- **ID**: `1083446768`
- **描述**: "A rusty key lies hidden in the depths of the abandoned mine
- **给予者**: Mine Key @ Abandoned Mine
- **目标**: Chest @ Abandoned Mine
- **奖励**: Piou Slayer

#### Snake Pit Thief

- **ID**: `647884116`
- **描述**: Investigate a hidden bedroom deep within the Snake Pit, uncover the fate of a Tulimshar thief, solve his riddle, and recover the treasure he never reclaimed.
- **给予者**: Snake Pit Clue @ Snake Pit
- **目标**: Thiefs Chest @ Snake Pit
- **奖励**: Unknown

#### Splatyna's Offering

- **ID**: `665326971`
- **描述**: "Bicies waits at the cave entrance, obsessed with a giant slime named Splatyna. He says she loves gold and keeps the corruption away.
- **给予者**: Bicies @ Splatyna Cave Entrance
- **目标**: Splatyna @ Splatyna's Chamber
- **奖励**: Unknown

#### Tutorial

- **ID**: `1180319673`
- **描述**: Get settled in Tulimshar with Elanore's help.
- **给予者**: Elanore @ Tulimshar Gates
- **目标**: (无) @ (无)
- **奖励**: Unknown

#### Veteran Cave Explorer

- **ID**: `2409290506`
- **描述**: For those who once walked these lands before the world was reborn.
- **给予者**: (无) @ (无)
- **目标**: (无) @ (无)
- **奖励**: Unknown

#### When the desert gives you sand...

- **ID**: `2192634378`
- **描述**: Eridu the glassmaker makes very useful glass bottles. He will exchange some bottles for salt, bones and sulphur powder gathered in the desert.
- **给予者**: Eridu @ Tulimshar Center
- **目标**: Salt x12, Bone x12, Sulphur Powder x12 @ Tonori Desert
- **奖励**: Bottle x3, 100 EXP

---

## 2. 游戏物品数据

共 **65** 个物品

| # | 名称 | 类别 | 槽位 | 购买价格 | 出售价格 | 重量 | 可堆叠 | 可使用 | 修饰符 |
|---|------|------|------|---------|---------|------|--------|--------|--------|
| 1 | Apple | usable | None | 10 | 1 | 5.0 | ✓ | ✓ | Health=20.0 |
| 2 | Artis Flour Sack | quest | QUEST | 0 | 0 | 50.0 | ✗ | ✗ | - |
| 3 | Bandana Scarf | neck | NECK | 5000 | 500 | 400.0 | ✗ | ✗ | - |
| 4 | Bone | common | None | 40 | 4 | 0.0 | ✓ | ✗ | - |
| 5 | Bone Knife | weapon | WEAPON | 15500 | 1550 | 400.0 | ✗ | ✗ | Attack=80.0, CastDelay=-0.1 |
| 6 | Bottle | common | None | 90 | 36 | 15.0 | ✓ | ✗ | - |
| 7 | Brass Zielite Amulet | quest | QUEST | 0 | 0 | 50.0 | ✗ | ✗ | - |
| 8 | Cactus Drink | usable | None | 55 | 6 | 5.0 | ✓ | ✓ | Health=75.0 |
| 9 | Cactus Potion | usable | None | 140 | 56 | 5.0 | ✓ | ✓ | Health=100.0 |
| 10 | Cactus Sour Candy | usable | None | 25 | 3 | 5.0 | ✓ | ✓ | Health=20.0, Stamina=50.0 |
| 11 | Chest Mine Key | common | QUEST | 0 | 0 | 15.0 | ✓ | ✗ | - |
| 12 | Cleaver | weapon | WEAPON | 3500 | 350 | 400.0 | ✗ | ✗ | Attack=100.0, CastDelay=0.5 |
| 13 | Cloud Pants | legs | LEGS | 1000 | 100 | 300.0 | ✗ | ✗ | - |
| 14 | Cotton Shirt | chest | CHEST | 1200 | 120 | 300.0 | ✗ | ✗ | Defense=1.0 |
| 15 | Croissant | usable | None | 25 | 3 | 5.0 | ✓ | ✓ | Mana=20.0, Stamina=50.0 |
| 16 | Crop Top | chest | CHEST | 1000 | 100 | 300.0 | ✗ | ✗ | - |
| 17 | Daim Boots | feet | FEET | 5100 | 510 | 300.0 | ✗ | ✗ | Defense=5.0, WalkSpeed=20.0 |
| 18 | Desert Armbands | hands | HANDS | 7100 | 710 | 300.0 | ✗ | ✗ | Defense=15.0 |
| 19 | Desert Armor | chest | CHEST | 50000 | 5000 | 5000.0 | ✗ | ✗ | Defense=30.0 |
| 20 | Desert Boots | feet | FEET | 1500 | 150 | 300.0 | ✗ | ✗ | Defense=2.0, WalkSpeed=20.0 |
| 21 | Desert Goggles | head | HEAD | 1900 | 190 | 100.0 | ✗ | ✗ | Defense=5.0 |
| 22 | Desert Hood | head | HEAD | 8000 | 800 | 300.0 | ✗ | ✗ | Defense=5.0 |
| 23 | Desert Jacket | chest | CHEST | 13500 | 1350 | 5000.0 | ✗ | ✗ | Defense=20.0 |
| 24 | Desert Shield | shield | SHIELD | 2700 | 270 | 400.0 | ✗ | ✗ | Defense=10.0, WalkSpeed=-10.0 |
| 25 | Dorian's Key | common | None | 0 | 0 | 15.0 | ✓ | ✗ | - |
| 26 | Druid Hood | head | HEAD | 2700 | 270 | 300.0 | ✗ | ✗ | - |
| 27 | Druid Robe | chest | CHEST | 2700 | 270 | 600.0 | ✗ | ✗ | - |
| 28 | Funky Hat | head | HEAD | 1900 | 190 | 300.0 | ✗ | ✗ | - |
| 29 | Gabriel's Key | common | None | 0 | 0 | 15.0 | ✓ | ✗ | - |
| 30 | Ghutra | head | HEAD | 2500 | 250 | 300.0 | ✗ | ✗ | Defense=5.0 |
| 31 | Gladius | weapon | WEAPON | 10000 | 1000 | 400.0 | ✗ | ✗ | Attack=50.0, AttackRange=16.0 |
| 32 | Gold Bracelets | hands | HANDS | 17500 | 1750 | 100.0 | ✗ | ✗ | Defense=15.0 |
| 33 | Gown | chest | CHEST | 1900 | 190 | 800.0 | ✗ | ✗ | - |
| 34 | Guard Cloud Pants | legs | LEGS | 1900 | 190 | 300.0 | ✗ | ✗ | - |
| 35 | Heavy Envelope | quest | QUEST | 0 | 0 | 25.0 | ✗ | ✗ | - |
| 36 | Jean Chaps | legs | LEGS | 13500 | 1350 | 300.0 | ✗ | ✗ | Defense=20.0 |
| 37 | Leather Armbands | hands | HANDS | 7100 | 710 | 300.0 | ✗ | ✗ | Defense=15.0 |
| 38 | Leather Boots | feet | FEET | 5100 | 510 | 300.0 | ✗ | ✗ | Defense=5.0, WalkSpeed=20.0 |
| 39 | Leather Gloves | hands | HANDS | 2900 | 290 | 300.0 | ✗ | ✗ | Defense=8.0 |
| 40 | Leather Pants | legs | LEGS | 3700 | 370 | 300.0 | ✗ | ✗ | Defense=10.0 |
| 41 | Leather Shield | shield | SHIELD | 2700 | 270 | 400.0 | ✗ | ✗ | Defense=10.0, WalkSpeed=-10.0 |
| 42 | Leather Shirt | chest | CHEST | 1900 | 190 | 300.0 | ✗ | ✗ | Defense=5.0 |
| 43 | Letter | quest | QUEST | 0 | 0 | 15.0 | ✗ | ✗ | - |
| 44 | Maggot Slime | common | None | 5 | 1 | 2.0 | ✓ | ✗ | - |
| 45 | Marvin's Key | common | None | 0 | 0 | 15.0 | ✓ | ✗ | - |
| 46 | Miner Helmet | head | HEAD | 1500 | 150 | 300.0 | ✗ | ✗ | Defense=3.0 |
| 47 | Piou Slayer | weapon | WEAPON | 2300 | 230 | 400.0 | ✗ | ✗ | Attack=20.0, CastDelay=-0.1 |
| 48 | Pitaya | usable | None | 10 | 1 | 5.0 | ✓ | ✓ | Health=15.0 |
| 49 | Raw Wood Shield | shield | SHIELD | 6000 | 600 | 400.0 | ✗ | ✗ | Defense=20.0, WalkSpeed=-20.0 |
| 50 | Rock Knife | weapon | WEAPON | 50000 | 5000 | 400.0 | ✗ | ✗ | Attack=100.0, MaxHealth=30.0, AttackRange=16.0 |
| 51 | Salt | common | None | 15 | 2 | 0.0 | ✓ | ✗ | - |
| 52 | Sandstorm Kano | quest | QUEST | 0 | 0 | 0.0 | ✗ | ✗ | - |
| 53 | Scimitar | weapon | WEAPON | 25000 | 2500 | 400.0 | ✗ | ✗ | Attack=75.0, CooldownDelay=-0.2, AttackRange=20.0 |
| 54 | Sealed Letters | quest | QUEST | 0 | 0 | 10.0 | ✗ | ✗ | - |
| 55 | Short Sword | weapon | WEAPON | 1500 | 150 | 400.0 | ✗ | ✗ | Attack=20.0 |
| 56 | Shorts | legs | LEGS | 1200 | 120 | 300.0 | ✗ | ✗ | Defense=1.0 |
| 57 | Silver Bracelets | hands | HANDS | 10500 | 1050 | 100.0 | ✗ | ✗ | Defense=15.0 |
| 58 | Snake Skin | common | None | 120 | 12 | 2.0 | ✓ | ✗ | - |
| 59 | Sulphur Powder | common | None | 30 | 3 | 0.0 | ✓ | ✗ | - |
| 60 | Tank Top | chest | CHEST | 1000 | 100 | 300.0 | ✗ | ✗ | - |
| 61 | Thief's Key | common | QUEST | 0 | 0 | 15.0 | ✓ | ✗ | - |
| 62 | Tonori Crown | head | HEAD | 20000 | 2000 | 200.0 | ✗ | ✗ | Defense=5.0 |
| 63 | Trousers | legs | LEGS | 1000 | 100 | 300.0 | ✗ | ✗ | - |
| 64 | V-Neck Tee | chest | CHEST | 1000 | 100 | 300.0 | ✗ | ✗ | - |
| 65 | Water Bottle | usable | None | 100 | 40 | 50.0 | ✓ | ✓ | Health=20.0 |

### 按类别统计

- **chest**: 9 个物品
- **common**: 11 个物品
- **feet**: 3 个物品
- **hands**: 5 个物品
- **head**: 7 个物品
- **legs**: 6 个物品
- **neck**: 1 个物品
- **quest**: 6 个物品
- **shield**: 3 个物品
- **usable**: 7 个物品
- **weapon**: 7 个物品

### 物品详情

#### Apple

- **文件**: `Apple.tres`
- **ID**: `215387671`
- **类别**: usable
- **槽位**: None
- **购买价格**: 10 GP
- **出售价格**: 1 GP
- **重量**: 5.0
- **可堆叠**: 是
- **可使用**: 是
- **修饰符**:
  - Health: 20.0

#### Artis Flour Sack

- **文件**: `ArtisFlourSack.tres`
- **ID**: `0`
- **类别**: quest
- **槽位**: QUEST
- **购买价格**: 0 GP
- **出售价格**: 0 GP
- **重量**: 50.0
- **可堆叠**: 否
- **可使用**: 否

#### Bandana Scarf

- **文件**: `BandanaScarf.tres`
- **ID**: `1956972377`
- **类别**: neck
- **槽位**: NECK
- **购买价格**: 5000 GP
- **出售价格**: 500 GP
- **重量**: 400.0
- **可堆叠**: 否
- **可使用**: 否

#### Bone

- **文件**: `Bone.tres`
- **ID**: `0`
- **类别**: common
- **槽位**: None
- **购买价格**: 40 GP
- **出售价格**: 4 GP
- **重量**: 0.0
- **可堆叠**: 是
- **可使用**: 否

#### Bone Knife

- **文件**: `BoneKnife.tres`
- **ID**: `4119861718`
- **类别**: weapon
- **槽位**: WEAPON
- **购买价格**: 15500 GP
- **出售价格**: 1550 GP
- **重量**: 400.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Attack: 80.0
  - CastDelay: -0.1

#### Bottle

- **文件**: `Bottle.tres`
- **ID**: `2850928111`
- **类别**: common
- **槽位**: None
- **购买价格**: 90 GP
- **出售价格**: 36 GP
- **重量**: 15.0
- **可堆叠**: 是
- **可使用**: 否

#### Brass Zielite Amulet

- **文件**: `BrassZieliteAmulet.tres`
- **ID**: `0`
- **类别**: quest
- **槽位**: QUEST
- **购买价格**: 0 GP
- **出售价格**: 0 GP
- **重量**: 50.0
- **可堆叠**: 否
- **可使用**: 否

#### Cactus Drink

- **文件**: `CactusDrink.tres`
- **ID**: `3877745440`
- **类别**: usable
- **槽位**: None
- **购买价格**: 55 GP
- **出售价格**: 6 GP
- **重量**: 5.0
- **可堆叠**: 是
- **可使用**: 是
- **修饰符**:
  - Health: 75.0

#### Cactus Potion

- **文件**: `CactusPotion.tres`
- **ID**: `3878004993`
- **类别**: usable
- **槽位**: None
- **购买价格**: 140 GP
- **出售价格**: 56 GP
- **重量**: 5.0
- **可堆叠**: 是
- **可使用**: 是
- **修饰符**:
  - Health: 100.0

#### Cactus Sour Candy

- **文件**: `CactusSourCandy.tres`
- **ID**: `3589756736`
- **类别**: usable
- **槽位**: None
- **购买价格**: 25 GP
- **出售价格**: 3 GP
- **重量**: 5.0
- **可堆叠**: 是
- **可使用**: 是
- **修饰符**:
  - Health: 20.0
  - Stamina: 50.0

#### Chest Mine Key

- **文件**: `ChestMineKey.tres`
- **ID**: `0`
- **类别**: common
- **槽位**: QUEST
- **购买价格**: 0 GP
- **出售价格**: 0 GP
- **重量**: 15.0
- **可堆叠**: 是
- **可使用**: 否

#### Cleaver

- **文件**: `Cleaver.tres`
- **ID**: `746948327`
- **类别**: weapon
- **槽位**: WEAPON
- **购买价格**: 3500 GP
- **出售价格**: 350 GP
- **重量**: 400.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Attack: 100.0
  - CastDelay: 0.5

#### Cloud Pants

- **文件**: `CloudPants.tres`
- **ID**: `1701922818`
- **类别**: legs
- **槽位**: LEGS
- **购买价格**: 1000 GP
- **出售价格**: 100 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否

#### Cotton Shirt

- **文件**: `CottonShirt.tres`
- **ID**: `1837303590`
- **类别**: chest
- **槽位**: CHEST
- **购买价格**: 1200 GP
- **出售价格**: 120 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 1.0

#### Croissant

- **文件**: `Croissant.tres`
- **ID**: `30320795`
- **类别**: usable
- **槽位**: None
- **购买价格**: 25 GP
- **出售价格**: 3 GP
- **重量**: 5.0
- **可堆叠**: 是
- **可使用**: 是
- **修饰符**:
  - Mana: 20.0
  - Stamina: 50.0

#### Crop Top

- **文件**: `CropTop.tres`
- **ID**: `2739365196`
- **类别**: chest
- **槽位**: CHEST
- **购买价格**: 1000 GP
- **出售价格**: 100 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否

#### Daim Boots

- **文件**: `DaimBoots.tres`
- **ID**: `1757255399`
- **类别**: feet
- **槽位**: FEET
- **购买价格**: 5100 GP
- **出售价格**: 510 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 5.0
  - WalkSpeed: 20.0

#### Desert Armbands

- **文件**: `DesertArmbands.tres`
- **ID**: `1633134452`
- **类别**: hands
- **槽位**: HANDS
- **购买价格**: 7100 GP
- **出售价格**: 710 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 15.0

#### Desert Armor

- **文件**: `DesertArmor.tres`
- **ID**: `585424461`
- **类别**: chest
- **槽位**: CHEST
- **购买价格**: 50000 GP
- **出售价格**: 5000 GP
- **重量**: 5000.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 30.0

#### Desert Boots

- **文件**: `DesertBoots.tres`
- **ID**: `586504915`
- **类别**: feet
- **槽位**: FEET
- **购买价格**: 1500 GP
- **出售价格**: 150 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 2.0
  - WalkSpeed: 20.0

#### Desert Goggles

- **文件**: `DesertGoggles.tres`
- **ID**: `906139156`
- **类别**: head
- **槽位**: HEAD
- **购买价格**: 1900 GP
- **出售价格**: 190 GP
- **重量**: 100.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 5.0

#### Desert Hood

- **文件**: `DesertHood.tres`
- **ID**: `3532052630`
- **类别**: head
- **槽位**: HEAD
- **购买价格**: 8000 GP
- **出售价格**: 800 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 5.0

#### Desert Jacket

- **文件**: `DesertJacket.tres`
- **ID**: `2470831870`
- **类别**: chest
- **槽位**: CHEST
- **购买价格**: 13500 GP
- **出售价格**: 1350 GP
- **重量**: 5000.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 20.0

#### Desert Shield

- **文件**: `DesertShield.tres`
- **ID**: `2831561157`
- **类别**: shield
- **槽位**: SHIELD
- **购买价格**: 2700 GP
- **出售价格**: 270 GP
- **重量**: 400.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 10.0
  - WalkSpeed: -10.0

#### Dorian's Key

- **文件**: `DorianKey.tres`
- **ID**: `3151885029`
- **类别**: common
- **槽位**: None
- **购买价格**: 0 GP
- **出售价格**: 0 GP
- **重量**: 15.0
- **可堆叠**: 是
- **可使用**: 否

#### Druid Hood

- **文件**: `DruidHood.tres`
- **ID**: `0`
- **类别**: head
- **槽位**: HEAD
- **购买价格**: 2700 GP
- **出售价格**: 270 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否

#### Druid Robe

- **文件**: `DruidRobe.tres`
- **ID**: `0`
- **类别**: chest
- **槽位**: CHEST
- **购买价格**: 2700 GP
- **出售价格**: 270 GP
- **重量**: 600.0
- **可堆叠**: 否
- **可使用**: 否

#### Funky Hat

- **文件**: `FunkyHat.tres`
- **ID**: `3690476111`
- **类别**: head
- **槽位**: HEAD
- **购买价格**: 1900 GP
- **出售价格**: 190 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否

#### Gabriel's Key

- **文件**: `GabrielKey.tres`
- **ID**: `2961268734`
- **类别**: common
- **槽位**: None
- **购买价格**: 0 GP
- **出售价格**: 0 GP
- **重量**: 15.0
- **可堆叠**: 是
- **可使用**: 否

#### Ghutra

- **文件**: `Ghutra.tres`
- **ID**: `3038339760`
- **类别**: head
- **槽位**: HEAD
- **购买价格**: 2500 GP
- **出售价格**: 250 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 5.0

#### Gladius

- **文件**: `Gladius.tres`
- **ID**: `1613203406`
- **类别**: weapon
- **槽位**: WEAPON
- **购买价格**: 10000 GP
- **出售价格**: 1000 GP
- **重量**: 400.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Attack: 50.0
  - AttackRange: 16.0

#### Gold Bracelets

- **文件**: `GoldBracelets.tres`
- **ID**: `2082846080`
- **类别**: hands
- **槽位**: HANDS
- **购买价格**: 17500 GP
- **出售价格**: 1750 GP
- **重量**: 100.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 15.0

#### Gown

- **文件**: `Gown.tres`
- **ID**: `2089150048`
- **类别**: chest
- **槽位**: CHEST
- **购买价格**: 1900 GP
- **出售价格**: 190 GP
- **重量**: 800.0
- **可堆叠**: 否
- **可使用**: 否

#### Guard Cloud Pants

- **文件**: `GuardCloudPants.tres`
- **ID**: `2687533109`
- **类别**: legs
- **槽位**: LEGS
- **购买价格**: 1900 GP
- **出售价格**: 190 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否

#### Heavy Envelope

- **文件**: `HeavyEnvelope.tres`
- **ID**: `0`
- **类别**: quest
- **槽位**: QUEST
- **购买价格**: 0 GP
- **出售价格**: 0 GP
- **重量**: 25.0
- **可堆叠**: 否
- **可使用**: 否

#### Jean Chaps

- **文件**: `JeanChaps.tres`
- **ID**: `1038638386`
- **类别**: legs
- **槽位**: LEGS
- **购买价格**: 13500 GP
- **出售价格**: 1350 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 20.0

#### Leather Armbands

- **文件**: `LeatherArmbands.tres`
- **ID**: `1500003666`
- **类别**: hands
- **槽位**: HANDS
- **购买价格**: 7100 GP
- **出售价格**: 710 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 15.0

#### Leather Boots

- **文件**: `LeatherBoots.tres`
- **ID**: `3383004529`
- **类别**: feet
- **槽位**: FEET
- **购买价格**: 5100 GP
- **出售价格**: 510 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 5.0
  - WalkSpeed: 20.0

#### Leather Gloves

- **文件**: `LeatherGloves.tres`
- **ID**: `162120794`
- **类别**: hands
- **槽位**: HANDS
- **购买价格**: 2900 GP
- **出售价格**: 290 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 8.0

#### Leather Pants

- **文件**: `LeatherPants.tres`
- **ID**: `3399103216`
- **类别**: legs
- **槽位**: LEGS
- **购买价格**: 3700 GP
- **出售价格**: 370 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 10.0

#### Leather Shield

- **文件**: `LeatherShield.tres`
- **ID**: `626767907`
- **类别**: shield
- **槽位**: SHIELD
- **购买价格**: 2700 GP
- **出售价格**: 270 GP
- **重量**: 400.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 10.0
  - WalkSpeed: -10.0

#### Leather Shirt

- **文件**: `LeatherShirt.tres`
- **ID**: `3402907028`
- **类别**: chest
- **槽位**: CHEST
- **购买价格**: 1900 GP
- **出售价格**: 190 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 5.0

#### Letter

- **文件**: `Letter.tres`
- **ID**: `3230422613`
- **类别**: quest
- **槽位**: QUEST
- **购买价格**: 0 GP
- **出售价格**: 0 GP
- **重量**: 15.0
- **可堆叠**: 否
- **可使用**: 否

#### Maggot Slime

- **文件**: `MaggotSlime.tres`
- **ID**: `3136767230`
- **类别**: common
- **槽位**: None
- **购买价格**: 5 GP
- **出售价格**: 1 GP
- **重量**: 2.0
- **可堆叠**: 是
- **可使用**: 否

#### Marvin's Key

- **文件**: `MarvinKey.tres`
- **ID**: `1975170517`
- **类别**: common
- **槽位**: None
- **购买价格**: 0 GP
- **出售价格**: 0 GP
- **重量**: 15.0
- **可堆叠**: 是
- **可使用**: 否

#### Miner Helmet

- **文件**: `MinerHelmet.tres`
- **ID**: `0`
- **类别**: head
- **槽位**: HEAD
- **购买价格**: 1500 GP
- **出售价格**: 150 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 3.0

#### Piou Slayer

- **文件**: `PiouSlayer.tres`
- **ID**: `629184690`
- **类别**: weapon
- **槽位**: WEAPON
- **购买价格**: 2300 GP
- **出售价格**: 230 GP
- **重量**: 400.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Attack: 20.0
  - CastDelay: -0.1

#### Pitaya

- **文件**: `Pitaya.tres`
- **ID**: `3391687821`
- **类别**: usable
- **槽位**: None
- **购买价格**: 10 GP
- **出售价格**: 1 GP
- **重量**: 5.0
- **可堆叠**: 是
- **可使用**: 是
- **修饰符**:
  - Health: 15.0

#### Raw Wood Shield

- **文件**: `RawWoodShield.tres`
- **ID**: `2754674049`
- **类别**: shield
- **槽位**: SHIELD
- **购买价格**: 6000 GP
- **出售价格**: 600 GP
- **重量**: 400.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 20.0
  - WalkSpeed: -20.0

#### Rock Knife

- **文件**: `RockKnife.tres`
- **ID**: `3712119489`
- **类别**: weapon
- **槽位**: WEAPON
- **购买价格**: 50000 GP
- **出售价格**: 5000 GP
- **重量**: 400.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Attack: 100.0
  - MaxHealth: 30.0
  - AttackRange: 16.0

#### Salt

- **文件**: `Salt.tres`
- **ID**: `0`
- **类别**: common
- **槽位**: None
- **购买价格**: 15 GP
- **出售价格**: 2 GP
- **重量**: 0.0
- **可堆叠**: 是
- **可使用**: 否

#### Sandstorm Kano

- **文件**: `SandstormKano.tres`
- **ID**: `0`
- **类别**: quest
- **槽位**: QUEST
- **购买价格**: 0 GP
- **出售价格**: 0 GP
- **重量**: 0.0
- **可堆叠**: 否
- **可使用**: 否

#### Scimitar

- **文件**: `Scimitar.tres`
- **ID**: `3601757377`
- **类别**: weapon
- **槽位**: WEAPON
- **购买价格**: 25000 GP
- **出售价格**: 2500 GP
- **重量**: 400.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Attack: 75.0
  - CooldownDelay: -0.2
  - AttackRange: 20.0

#### Sealed Letters

- **文件**: `SealedLetters.tres`
- **ID**: `0`
- **类别**: quest
- **槽位**: QUEST
- **购买价格**: 0 GP
- **出售价格**: 0 GP
- **重量**: 10.0
- **可堆叠**: 否
- **可使用**: 否

#### Short Sword

- **文件**: `ShortSword.tres`
- **ID**: `3056796356`
- **类别**: weapon
- **槽位**: WEAPON
- **购买价格**: 1500 GP
- **出售价格**: 150 GP
- **重量**: 400.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Attack: 20.0

#### Shorts

- **文件**: `LinenShorts.tres`
- **ID**: `3507746760`
- **类别**: legs
- **槽位**: LEGS
- **购买价格**: 1200 GP
- **出售价格**: 120 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 1.0

#### Silver Bracelets

- **文件**: `SilverBracelets.tres`
- **ID**: `3411966351`
- **类别**: hands
- **槽位**: HANDS
- **购买价格**: 10500 GP
- **出售价格**: 1050 GP
- **重量**: 100.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 15.0

#### Snake Skin

- **文件**: `SnakeSkin.tres`
- **ID**: `841480588`
- **类别**: common
- **槽位**: None
- **购买价格**: 120 GP
- **出售价格**: 12 GP
- **重量**: 2.0
- **可堆叠**: 是
- **可使用**: 否

#### Sulphur Powder

- **文件**: `SulphurPowder.tres`
- **ID**: `0`
- **类别**: common
- **槽位**: None
- **购买价格**: 30 GP
- **出售价格**: 3 GP
- **重量**: 0.0
- **可堆叠**: 是
- **可使用**: 否

#### Tank Top

- **文件**: `TankTop.tres`
- **ID**: `878238790`
- **类别**: chest
- **槽位**: CHEST
- **购买价格**: 1000 GP
- **出售价格**: 100 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否

#### Thief's Key

- **文件**: `ThiefsKey.tres`
- **ID**: `1577372888`
- **类别**: common
- **槽位**: QUEST
- **购买价格**: 0 GP
- **出售价格**: 0 GP
- **重量**: 15.0
- **可堆叠**: 是
- **可使用**: 否

#### Tonori Crown

- **文件**: `TonoriCrown.tres`
- **ID**: `2445605769`
- **类别**: head
- **槽位**: HEAD
- **购买价格**: 20000 GP
- **出售价格**: 2000 GP
- **重量**: 200.0
- **可堆叠**: 否
- **可使用**: 否
- **修饰符**:
  - Defense: 5.0

#### Trousers

- **文件**: `Trousers.tres`
- **ID**: `1412353772`
- **类别**: legs
- **槽位**: LEGS
- **购买价格**: 1000 GP
- **出售价格**: 100 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否

#### V-Neck Tee

- **文件**: `VNeckTee.tres`
- **ID**: `1416971591`
- **类别**: chest
- **槽位**: CHEST
- **购买价格**: 1000 GP
- **出售价格**: 100 GP
- **重量**: 300.0
- **可堆叠**: 否
- **可使用**: 否

#### Water Bottle

- **文件**: `WaterBottle.tres`
- **ID**: `3288439794`
- **类别**: usable
- **槽位**: None
- **购买价格**: 100 GP
- **出售价格**: 40 GP
- **重量**: 50.0
- **可堆叠**: 是
- **可使用**: 是
- **修饰符**:
  - Health: 20.0

---

## 3. 游戏技能数据

共 **15** 个技能

| # | 名称 | 类别 | 模式 | 范围 | 冷却时间 | 施法时间 | 技能时间 | 重复 | 修饰符 |
|---|------|------|------|------|---------|---------|---------|------|--------|
| 1 | Archer | PHYSICAL | SINGLE | 100 | 0.0 | 0.0 | 0.0 | ✓ | Stamina=-10, Attack=1.0 |
| 2 | Betsanc | SPELL | SELF | 0 | 30.0 | 5.0 | 2.0 | ✗ | Mana=-20.0, Stamina=-20, Defense=50.0, MDefense=50.0, DodgeRate=0.08 |
| 3 | Flar | SPELL | SINGLE | 120 | 3.0 | 1.0 | 0.0 | ✗ | MAttack=50, Mana=-10, Stamina=-10 |
| 4 | Inma | SPELL | SINGLE | 128 | 5.0 | 3.0 | 2.0 | ✗ | Health=50, Mana=-10, Stamina=-15 |
| 5 | Jump | ABILITY | SELF | 256 | 5.0 | 0.5 | 1.5 | ✗ | - |
| 6 | Leaf Blades | SPELL | ZONE | 40 | 1.0 | 2.0 | 2.0 | ✗ | MAttack=30, Mana=-10, Stamina=-5 |
| 7 | Lum | SPELL | SELF | 0 | 5.0 | 2.0 | 2.0 | ✗ | Health=50, Mana=-5, Stamina=-15 |
| 8 | Mana Burst | SPELL | ZONE | 40 | 1.0 | 2.0 | 2.0 | ✗ | MAttack=20.0, Mana=-8.0, Stamina=-10.0 |
| 9 | Melee | PHYSICAL | SINGLE | 0 | 0.0 | 0.0 | 0.0 | ✓ | Attack=1, Stamina=-10 |
| 10 | Morph | ABILITY | SELF | 0 | 5.0 | 10.0 | 0.0 | ✗ | Stamina=-50.0, Mana=-50.0 |
| 11 | Poison | ABILITY | SINGLE | 0 | 15.0 | 0.1 | 0.0 | ✗ | RegenHealth=-1.0, Attack=1.0 |
| 12 | Run | ABILITY | SELF | 0 | 0.0 | 0.0 | 0.0 | ✗ | - |
| 13 | Sonic Scream | SPELL | SINGLE | 64 | 3.0 | 0.0 | 0.0 | ✗ | MAttack=10, Mana=-2, Stamina=-2 |
| 14 | Sonic Wave | SPELL | SINGLE | 64 | 3.0 | 0.0 | 0.0 | ✗ | MAttack=50, Mana=-10, Stamina=-10 |
| 15 | Spitfire | SPELL | SINGLE | 32 | 1.0 | 0.0 | 0.1 | ✓ | MAttack=20.0, Mana=-5.0, Stamina=-10 |

### 技能详情

#### Archer

- **文件**: `Archer.tres`
- **ID**: `2814726266`
- **描述**: A simple bow attack, shoot one arrow at a time.
- **类别**: PHYSICAL
- **模式**: SINGLE
- **范围**: 100
- **冷却时间**: 0.0s
- **施法时间**: 0.0s
- **技能时间**: 0.0s
- **Buff时间**: 0.0s
- **重复**: 是
- **修饰符**:
  - Stamina: -10
  - Attack: 1.0

#### Betsanc

- **文件**: `Betsanc.tres`
- **ID**: `0`
- **描述**: Feel the mana warping your and raising your defense and evasion for 2 minutes.
- **类别**: SPELL
- **模式**: SELF
- **范围**: 0
- **冷却时间**: 30.0s
- **施法时间**: 5.0s
- **技能时间**: 2.0s
- **Buff时间**: 120.0s
- **重复**: 否
- **修饰符**:
  - Mana: -20.0
  - Stamina: -20
  - Defense: 50.0
  - MDefense: 50.0
  - DodgeRate: 0.08

#### Flar

- **文件**: `Flar.tres`
- **ID**: `2089110122`
- **描述**: Cast a fire projectile, igniting enemies with intense heat and power.
- **类别**: SPELL
- **模式**: SINGLE
- **范围**: 120
- **冷却时间**: 3.0s
- **施法时间**: 1.0s
- **技能时间**: 0.0s
- **Buff时间**: 0.0s
- **重复**: 否
- **修饰符**:
  - MAttack: 50
  - Mana: -10
  - Stamina: -10

#### Inma

- **文件**: `Inma.tres`
- **ID**: `2089220490`
- **描述**: Provide a light healing touch to allies or creatures.
- **类别**: SPELL
- **模式**: SINGLE
- **范围**: 128
- **冷却时间**: 5.0s
- **施法时间**: 3.0s
- **技能时间**: 2.0s
- **Buff时间**: 0.0s
- **重复**: 否
- **修饰符**:
  - Health: 50
  - Mana: -10
  - Stamina: -15

#### Jump

- **文件**: `Jump.tres`
- **ID**: `2089264065`
- **描述**: Blink to a random nearby position.
- **类别**: ABILITY
- **模式**: SELF
- **范围**: 256
- **冷却时间**: 5.0s
- **施法时间**: 0.5s
- **技能时间**: 1.5s
- **Buff时间**: 0.0s
- **重复**: 否

#### Leaf Blades

- **文件**: `LeafBlades.tres`
- **ID**: `0`
- **描述**: Unleash a mana-fueled attack, creating a focused circle of concentrated mana around you.
- **类别**: SPELL
- **模式**: ZONE
- **范围**: 40
- **冷却时间**: 1.0s
- **施法时间**: 2.0s
- **技能时间**: 2.0s
- **Buff时间**: 0.0s
- **重复**: 否
- **修饰符**:
  - MAttack: 30
  - Mana: -10
  - Stamina: -5

#### Lum

- **文件**: `Lum.tres`
- **ID**: `193463731`
- **描述**: Heal yourself with a burst of restorative energy.
- **类别**: SPELL
- **模式**: SELF
- **范围**: 0
- **冷却时间**: 5.0s
- **施法时间**: 2.0s
- **技能时间**: 2.0s
- **Buff时间**: 0.0s
- **重复**: 否
- **修饰符**:
  - Health: 50
  - Mana: -5
  - Stamina: -15

#### Mana Burst

- **文件**: `ManaBurst.tres`
- **ID**: `2393790290`
- **描述**: Unleash a mana-fueled attack, creating a focused circle of concentrated mana around you.
- **类别**: SPELL
- **模式**: ZONE
- **范围**: 40
- **冷却时间**: 1.0s
- **施法时间**: 2.0s
- **技能时间**: 2.0s
- **Buff时间**: 0.0s
- **重复**: 否
- **修饰符**:
  - MAttack: 20.0
  - Mana: -8.0
  - Stamina: -10.0

#### Melee

- **文件**: `Melee.tres`
- **ID**: `229218829`
- **描述**: Close combat attack.
- **类别**: PHYSICAL
- **模式**: SINGLE
- **范围**: 0
- **冷却时间**: 0.0s
- **施法时间**: 0.0s
- **技能时间**: 0.0s
- **Buff时间**: 0.0s
- **重复**: 是
- **修饰符**:
  - Attack: 1
  - Stamina: -10

#### Morph

- **文件**: `Morph.tres`
- **ID**: `0`
- **描述**: Transform into your spirit form.
- **类别**: ABILITY
- **模式**: SELF
- **范围**: 0
- **冷却时间**: 5.0s
- **施法时间**: 10.0s
- **技能时间**: 0.0s
- **Buff时间**: 0.0s
- **重复**: 否
- **修饰符**:
  - Stamina: -50.0
  - Mana: -50.0

#### Poison

- **文件**: `Poison.tres`
- **ID**: `0`
- **描述**: A venomous attack that remove vital forces away until it is cured.
- **类别**: ABILITY
- **模式**: SINGLE
- **范围**: 0
- **冷却时间**: 15.0s
- **施法时间**: 0.1s
- **技能时间**: 0.0s
- **Buff时间**: -1.0s
- **重复**: 否
- **修饰符**:
  - RegenHealth: -1.0
  - Attack: 1.0

#### Run

- **文件**: `Run.tres`
- **ID**: `0`
- **描述**: Toggle running to move faster at the cost of stamina.
- **类别**: ABILITY
- **模式**: SELF
- **范围**: 0
- **冷却时间**: 0.0s
- **施法时间**: 0.0s
- **技能时间**: 0.0s
- **Buff时间**: 0.0s
- **重复**: 否

#### Sonic Scream

- **文件**: `SonicScream.tres`
- **ID**: `0`
- **描述**: Releases a blue sonic wave that deals damage forward.
- **类别**: SPELL
- **模式**: SINGLE
- **范围**: 64
- **冷却时间**: 3.0s
- **施法时间**: 0.0s
- **技能时间**: 0.0s
- **Buff时间**: 0.0s
- **重复**: 否
- **修饰符**:
  - MAttack: 10
  - Mana: -2
  - Stamina: -2

#### Sonic Wave

- **文件**: `SonicWave.tres`
- **ID**: `123697108`
- **描述**: Releases a blue sonic wave that deals damage forward.
- **类别**: SPELL
- **模式**: SINGLE
- **范围**: 64
- **冷却时间**: 3.0s
- **施法时间**: 0.0s
- **技能时间**: 0.0s
- **Buff时间**: 0.0s
- **重复**: 否
- **修饰符**:
  - MAttack: 50
  - Mana: -10
  - Stamina: -10

#### Spitfire

- **文件**: `Spitfire.tres`
- **ID**: `3219153995`
- **描述**: Cast a fire spray
- **类别**: SPELL
- **模式**: SINGLE
- **范围**: 32
- **冷却时间**: 1.0s
- **施法时间**: 0.0s
- **技能时间**: 0.1s
- **Buff时间**: 0.0s
- **重复**: 是
- **修饰符**:
  - MAttack: 20.0
  - Mana: -5.0
  - Stamina: -10

---

## 4. 游戏实体数据

共 **99** 个实体

| # | 名称 | ID | 行为 | Boss | 父实体 | 等级 | 生命值 | 攻击力 | 防御力 |
|---|------|-----|------|------|--------|------|--------|--------|--------|
| 1 |  | 0 | NONE | ✗ | - | - | - | - | - |
| 2 |  | 0 | NONE | ✗ | - | - | - | - | - |
| 3 | NPC | 193464646 | PACIFIST, NEUTRAL | ✗ | Object | - | - | - | - |
| 4 | 乌龟 | 3562408837 | NONE | ✗ | - | 10 | 250 | 15 | 35 |
| 5 | 井字棋格子 | 1060916485 | NONE | ✗ | StaticObject | - | - | - | - |
| 6 | 伊莱亚斯 | 219979635 | NONE | ✗ | StaticNPC | - | - | - | - |
| 7 | 传送 | 2089709631 | PACIFIST, IMMOBILE | ✗ | - | - | - | - | - |
| 8 | 佩约特 | 3387138907 | NEUTRAL, IMMOBILE | ✗ | - | 2 | 70 | 15 | 2 |
| 9 | 信件堆 | 3922677720 | PACIFIST, IMMOBILE | ✗ | StaticObject | - | - | - | - |
| 10 | 光源触发器 | 31947249 | PACIFIST, IMMOBILE | ✗ | StaticObject | - | - | - | - |
| 11 | 凯尔 | 2089277954 | NONE | ✗ | StaticNPC | - | - | - | - |
| 12 | 加布兹 | 2089134121 | NONE | ✗ | StaticNPC | - | - | - | - |
| 13 | 加百列 | 1184402587 | AGGRESSIVE, LEADER, STEAL | ✗ | Slime | 5 | - | - | - |
| 14 | 卢莱亚 | 228607896 | NEUTRAL, FOLLOWER, STEAL | ✗ | Slime | 1 | - | - | - |
| 15 | 史莱姆 | 236582911 | NEUTRAL, STEAL | ✗ | - | 1 | - | - | - |
| 16 | 吉尔伽美 | 2550418766 | NONE | ✗ | StaticNPC | - | - | - | - |
| 17 | 哥布林 | 3045949408 | NONE | ✗ | - | 24 | 400 | 30 | 45 |
| 18 | 图利姆沙守卫 | 3639315857 | NONE | ✗ | WalkingNPC | - | - | - | - |
| 19 | 图利姆沙新兵 | 3652872819 | NONE | ✗ | WalkingNPC | - | - | - | - |
| 20 | 图利姆沙沙漠守卫 | 3905186616 | NONE | ✗ | WalkingNPC | - | - | - | - |
| 21 | 图利姆沙沙漠新兵 | 3918743578 | NONE | ✗ | WalkingNPC | - | - | - | - |
| 22 | 埃米尔 | 2089075532 | NEUTRAL, FOLLOWER, STEAL | ✗ | - | 1 | - | - | - |
| 23 | 埃里杜 | 220195358 | NONE | ✗ | StaticNPC | - | - | - | - |
| 24 | 塞玛 | 2089570059 | NONE | ✗ | StaticNPC | - | - | - | - |
| 25 | 多里安 | 2929114690 | AGGRESSIVE, LEADER, STEAL | ✗ | Slime | 5 | - | - | - |
| 26 | 妮娜 | 2089394763 | NONE | ✗ | StaticNPC | - | - | - | - |
| 27 | 安迪 | 2088932705 | NONE | ✗ | StaticNPC | - | - | - | - |
| 28 | 宝箱 | 217460284 | NONE | ✗ | StaticObject | - | - | - | - |
| 29 | 小粉 | 3391482565 | NONE | ✗ | - | 1 | - | - | - |
| 30 | 尖刺蘑菇 | 2549154767 | NONE | ✗ | - | 1 | - | - | - |
| 31 | 尼古拉斯 | 1948908174 | NONE | ✗ | StaticNPC | - | - | - | - |
| 32 | 尼科斯 | 3312816684 | NONE | ✗ | StaticNPC | - | - | - | - |
| 33 | 巨型蛆 | 2624609623 | NEUTRAL, LEADER, SPAWNER | ✗ | - | 15 | 370 | 54 | 20 |
| 34 | 帕丘亚 | 3381597015 | NONE | ✗ | StaticNPC | - | - | - | - |
| 35 | 弗罗斯特 | 221388307 | NONE | ✗ | StaticNPC | - | - | - | - |
| 36 | 强盗 | 2834092087 | NEUTRAL, STEAL | ✗ | - | 25 | 380 | 55 | 10 |
| 37 | 彼得 | 232785317 | NONE | ✗ | StaticNPC | - | - | - | - |
| 38 | 扎克贝尔 | 2715760142 | AGGRESSIVE | ✓ | - | 20 | 250 | - | 0 |
| 39 | 排水史莱姆 | 2469123693 | NEUTRAL, STEAL | ✗ | Slime | 1 | - | - | - |
| 40 | 斯普拉蒂娜 | 3314548081 | AGGRESSIVE, LEADER, SPAWNER | ✓ | Slime | 10 | - | - | - |
| 41 | 旧宝箱 | 4054136443 | NONE | ✗ | Chest | - | - | - | - |
| 42 | 明格尔 | 670207772 | NONE | ✗ | StaticNPC | - | - | - | - |
| 43 | 本 | 193452314 | NONE | ✗ | StaticNPC | - | - | - | - |
| 44 | 比西斯 | 2843189460 | NONE | ✗ | WalkingNPC | - | - | - | - |
| 45 | 毛绒绒 | 3003932433 | NONE | ✗ | - | 1 | - | - | - |
| 46 | 毛罗 | 229085321 | IMMOBILE | ✗ | StaticNPC | - | - | - | - |
| 47 | 水井 | 2089713785 | PACIFIST, IMMOBILE | ✗ | StaticObject | - | - | - | - |
| 48 | 沙暴矿井裂缝 | 434267405 | IMMOBILE | ✗ | sub:Resource_6o40j | - | - | - | - |
| 49 | 沙暴矿井钥匙 | 3478975634 | NONE | ✗ | StaticObject | - | - | - | - |
| 50 | 沙蛇 | 2399925469 | AGGRESSIVE | ✗ | - | 19 | 200 | 85 | 5 |
| 51 | 海兰 | 3097291557 | NONE | ✗ | StaticNPC | - | - | - | - |
| 52 | 海史莱姆 | 1744780248 | NEUTRAL, STEAL | ✗ | Slime | 6 | 200 | 20 | 15 |
| 53 | 淤泥史莱姆 | 2450367875 | NEUTRAL, STEAL | ✗ | Slime | 1 | - | - | - |
| 54 | 火焰哥布林 | 1836204198 | NONE | ✗ | - | 8 | 75 | 15 | 20 |
| 55 | 火焰骷髅 | 4234567318 | AGGRESSIVE | ✗ | - | 10 | - | - | - |
| 56 | 灵魂巨石 | 737772331 | NONE | ✗ | StaticNPC | - | - | - | - |
| 57 | 物体 | 3343895260 | NONE | ✗ | - | - | - | - | - |
| 58 | 猞猁 | 2089340336 | AGGRESSIVE | ✗ | - | 21 | 250 | 40 | 15 |
| 59 | 猫 | 2089390322 | NONE | ✗ | StaticNPC | - | - | - | - |
| 60 | 玛纳伊尔女德鲁伊 | 608165177 | NONE | ✗ | - | - | - | - | - |
| 61 | 玛纳伊尔德鲁伊 | 999510798 | NONE | ✗ | - | - | - | - | - |
| 62 | 玩家 | 3394588274 | NONE | ✗ | - | 1 | - | - | - |
| 63 | 瑞安 | 2089555519 | NONE | ✗ | StaticNPC | - | - | - | - |
| 64 | 白 | 236439338 | NONE | ✗ | StaticNPC | - | - | - | - |
| 65 | 皮乌 | 2089466690 | NEUTRAL, FLEE | ✗ | - | 1 | 30 | 5 | 1 |
| 66 | 盐史莱姆 | 2250973363 | NEUTRAL, STEAL | ✗ | Slime | 1 | - | - | - |
| 67 | 盗贼宝箱 | 970294591 | NONE | ✗ | Chest | - | - | - | - |
| 68 | 红女王 | 3662818686 | NONE | ✗ | StaticNPC | - | - | - | - |
| 69 | 红蝎 | 825312237 | AGGRESSIVE | ✗ | Scorpion | 15 | 500 | 55 | 5 |
| 70 | 纳姆 | 2089385766 | NONE | ✗ | StaticNPC | - | - | - | - |
| 71 | 纳德 | 2089386186 | NONE | ✗ | StaticNPC | - | - | - | - |
| 72 | 维 | 193475306 | NONE | ✗ | StaticNPC | - | - | - | - |
| 73 | 船 | 2089573209 | NONE | ✗ | - | - | - | 0.0 | 0.0 |
| 74 | 艾基努 | 219944129 | NONE | ✗ | StaticNPC | - | - | - | - |
| 75 | 艾拉诺 | 3325600555 | NONE | ✗ | StaticNPC | - | - | - | - |
| 76 | 莉拉 | 228176527 | NEUTRAL, FOLLOWER, STEAL | ✗ | Slime | 1 | - | - | - |
| 77 | 莉赞德 | 1435506622 | NEUTRAL, FOLLOWER | ✗ | - | 12 | 220 | 55 | 8 |
| 78 | 莉赞德拉 | 1435506618 | AGGRESSIVE, LEADER | ✗ | - | 14 | 300 | 65 | 15 |
| 79 | 萨威尔 | 236203253 | NONE | ✗ | StaticNPC | - | - | - | - |
| 80 | 蛆 | 3264333316 | NEUTRAL, FLEE | ✗ | - | 1 | 50 | 10 | 1 |
| 81 | 蛇 | 236646007 | AGGRESSIVE | ✗ | - | 17 | 150 | 40 | 5 |
| 82 | 蛇坑池塘 | 1190851605 | PACIFIST, IMMOBILE | ✗ | StaticObject | - | - | - | - |
| 83 | 蛇坑线索 | 1190381389 | PACIFIST, IMMOBILE | ✗ | StaticObject | - | - | - | - |
| 84 | 蝎子 | 3842739378 | NONE | ✗ | - | 4 | 200 | 22 | 0 |
| 85 | 蝙蝠 | 193452188 | NONE | ✗ | - | 3 | 90 | 10 | 2 |
| 86 | 行走NPC | 2377595219 | PACIFIST | ✗ | NPC | - | - | - | - |
| 87 | 邪恶蘑菇 | 2866154511 | NONE | ✗ | - | 1 | - | - | - |
| 88 | 里斯基姆 | 3469933044 | NONE | ✗ | StaticNPC | - | - | - | - |
| 89 | 锁 | 228382689 | NONE | ✗ | StaticObject | - | - | - | - |
| 90 | 静态NPC | 89439950 | PACIFIST, IMMOBILE | ✗ | NPC | - | - | - | - |
| 91 | 静态物体 | 1629942628 | PACIFIST, IMMOBILE | ✗ | Object | - | - | - | - |
| 92 | 面粉桶 | 1902637957 | NONE | ✗ | StaticObject | - | - | - | - |
| 93 | 马克拉特 | 3264194551 | NONE | ✗ | StaticNPC | - | - | - | - |
| 94 | 马文 | 3264744754 | AGGRESSIVE, LEADER, SPAWNER, STEAL | ✗ | Slime | 5 | - | - | - |
| 95 | 骷髅 | 890728458 | AGGRESSIVE | ✗ | - | 20 | 300 | - | 0 |
| 96 | 魔力树 | 2025440114 | PACIFIST, IMMOBILE | ✗ | - | - | - | - | - |
| 97 | 鳄鱼 | 2089009292 | AGGRESSIVE, FLEE | ✗ | - | 5 | 55 | 8 | 40 |
| 98 | 鸟 | 2088963654 | NEUTRAL, STEAL | ✗ | - | 27 | 220 | 35 | 15 |
| 99 | 鼠 | 235013903 | NEUTRAL, FLEE | ✗ | - | 5 | 85 | 10 | 2 |

### 实体详情

#### 

- **文件**: `Ally.tres`
- **ID**: `0`
- **行为**: NONE
- **Boss**: 否
- **半径**: 0

#### 

- **文件**: `Enemy.tres`
- **ID**: `0`
- **行为**: NONE
- **Boss**: 否
- **半径**: 0

#### NPC

- **文件**: `NPC.tres`
- **ID**: `193464646`
- **行为**: PACIFIST, NEUTRAL
- **Boss**: 否
- **父实体**: Object
- **半径**: 0
- **装备**:

#### 乌龟

- **文件**: `Turtle.tres`
- **ID**: `3562408837`
- **行为**: NONE
- **Boss**: 否
- **半径**: 10
- **属性**:
  - attack: 15
  - baseExp: 15
  - castAttackDelay: 1.0
  - cooldownAttackDelay: 1.5
  - defense: 35
  - level: 10
  - maxHealth: 250
  - mdefense: 40
  - walkSpeed: 30.0
- **装备**:

#### 井字棋格子

- **文件**: `TicTacToeCell.tres`
- **ID**: `1060916485`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticObject
- **半径**: 16
- **装备**:

#### 伊莱亚斯

- **文件**: `Elias.tres`
- **ID**: `219979635`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Female
  - race: Human
  - skintone: Medium
- **装备**:

#### 传送

- **文件**: `Warp.tres`
- **ID**: `2089709631`
- **行为**: PACIFIST, IMMOBILE
- **Boss**: 否
- **半径**: 16
- **装备**:

#### 佩约特

- **文件**: `Peyote.tres`
- **ID**: `3387138907`
- **行为**: NEUTRAL, IMMOBILE
- **Boss**: 否
- **半径**: 14
- **属性**:
  - attack: 15
  - baseExp: 3
  - cooldownAttackDelay: 3.0
  - defense: 2
  - level: 2
  - maxHealth: 70
- **装备**:

#### 信件堆

- **文件**: `Letter Stash.tres`
- **ID**: `3922677720`
- **行为**: PACIFIST, IMMOBILE
- **Boss**: 否
- **父实体**: StaticObject
- **半径**: 8
- **装备**:

#### 光源触发器

- **文件**: `Light Trigger.tres`
- **ID**: `31947249`
- **行为**: PACIFIST, IMMOBILE
- **Boss**: 否
- **父实体**: StaticObject
- **半径**: 0
- **装备**:

#### 凯尔

- **文件**: `Kael.tres`
- **ID**: `2089277954`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - race: Human
  - skintone: Tanned
- **装备**:

#### 加布兹

- **文件**: `Gabz.tres`
- **ID**: `2089134121`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - race: Raijin
  - skintone: Dark
- **装备**:

#### 加百列

- **文件**: `Gabriel.tres`
- **ID**: `1184402587`
- **行为**: AGGRESSIVE, LEADER, STEAL
- **Boss**: 否
- **父实体**: Slime
- **半径**: 10
- **属性**:
  - baseExp: 10
  - level: 5
  - walkSpeed: 40.0
- **装备**:

#### 卢莱亚

- **文件**: `Lulea.tres`
- **ID**: `228607896`
- **行为**: NEUTRAL, FOLLOWER, STEAL
- **Boss**: 否
- **父实体**: Slime
- **半径**: 10
- **属性**:
  - level: 1
  - walkSpeed: 40.0
- **装备**:

#### 史莱姆

- **文件**: `Slime.tres`
- **ID**: `236582911`
- **行为**: NEUTRAL, STEAL
- **Boss**: 否
- **半径**: 10
- **属性**:
  - level: 1
  - walkSpeed: 40.0
- **装备**:

#### 吉尔伽美

- **文件**: `Gilgames.tres`
- **ID**: `2550418766`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - haircolor: Golden Blonde
  - hairstyle: Tapered Nape
  - race: Human
  - skintone: Dark
- **装备**:
  - CHEST: DesertArmor
  - LEGS: GuardCloudPants
  - FEET: LeatherBoots
  - HANDS: LeatherGloves
  - NECK: BandanaScarf
  - WEAPON: BoneKnife

#### 哥布林

- **文件**: `Goblin.tres`
- **ID**: `3045949408`
- **行为**: NONE
- **Boss**: 否
- **半径**: 8
- **属性**:
  - attack: 30
  - baseExp: 100
  - defense: 45
  - level: 24
  - mattack: 70
  - maxHealth: 400
  - mdefense: 5
  - walkSpeed: 50.0
- **装备**:

#### 图利姆沙守卫

- **文件**: `Tulimshar Guard.tres`
- **ID**: `3639315857`
- **行为**: NONE
- **Boss**: 否
- **父实体**: WalkingNPC
- **半径**: 0
- **属性**:
  - gender: Female
  - haircolor: Light Violet
  - hairstyle: Imperial
  - race: Human
  - skintone: Dark
- **装备**:
  - CHEST: DesertArmor
  - LEGS: GuardCloudPants
  - FEET: LeatherBoots
  - WEAPON: Scimitar

#### 图利姆沙新兵

- **文件**: `Tulimshar Sbire.tres`
- **ID**: `3652872819`
- **行为**: NONE
- **Boss**: 否
- **父实体**: WalkingNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - haircolor: Esperia Blue
  - hairstyle: Pompadour
  - race: Human
  - skintone: Dark
- **装备**:
  - CHEST: DesertJacket
  - LEGS: GuardCloudPants
  - FEET: LeatherBoots
  - HEAD: DesertGoggles
  - WEAPON: Gladius

#### 图利姆沙沙漠守卫

- **文件**: `Tulimshar Desert Guard.tres`
- **ID**: `3905186616`
- **行为**: NONE
- **Boss**: 否
- **父实体**: WalkingNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - haircolor: Dark Brown
  - hairstyle: Center Parting
  - race: Human
  - skintone: Dark
- **装备**:
  - CHEST: DesertArmor
  - LEGS: GuardCloudPants
  - FEET: LeatherBoots
  - HEAD: DesertHood
  - WEAPON: Scimitar

#### 图利姆沙沙漠新兵

- **文件**: `Tulimshar Desert Sbire.tres`
- **ID**: `3918743578`
- **行为**: NONE
- **Boss**: 否
- **父实体**: WalkingNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - haircolor: Dark Copper
  - hairstyle: Tapered Nape
  - race: Human
  - skintone: Dark
- **装备**:
  - CHEST: DesertJacket
  - LEGS: GuardCloudPants
  - FEET: LeatherBoots
  - HEAD: Ghutra
  - WEAPON: Gladius

#### 埃米尔

- **文件**: `Emil.tres`
- **ID**: `2089075532`
- **行为**: NEUTRAL, FOLLOWER, STEAL
- **Boss**: 否
- **半径**: 10
- **属性**:
  - level: 1
  - walkSpeed: 40.0
- **装备**:

#### 埃里杜

- **文件**: `Eridu.tres`
- **ID**: `220195358`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - race: Kralog
  - skintone: Fire
- **装备**:

#### 塞玛

- **文件**: `Sema.tres`
- **ID**: `2089570059`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Female
  - race: Human
  - skintone: Medium
- **装备**:

#### 多里安

- **文件**: `Dorian.tres`
- **ID**: `2929114690`
- **行为**: AGGRESSIVE, LEADER, STEAL
- **Boss**: 否
- **父实体**: Slime
- **半径**: 10
- **属性**:
  - baseExp: 10
  - level: 5
  - walkSpeed: 40.0
- **装备**:

#### 妮娜

- **文件**: `Nina.tres`
- **ID**: `2089394763`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Female
  - race: Human
  - skintone: Medium
- **装备**:

#### 安迪

- **文件**: `Andi.tres`
- **ID**: `2088932705`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - race: Human
  - skintone: Medium
- **装备**:

#### 宝箱

- **文件**: `Chest.tres`
- **ID**: `217460284`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticObject
- **半径**: 26
- **装备**:

#### 小粉

- **文件**: `Pinkie.tres`
- **ID**: `3391482565`
- **行为**: NONE
- **Boss**: 否
- **半径**: 12
- **属性**:
  - level: 1
  - walkSpeed: 80.0
- **装备**:

#### 尖刺蘑菇

- **文件**: `Spiky Mushroom.tres`
- **ID**: `2549154767`
- **行为**: NONE
- **Boss**: 否
- **半径**: 16
- **属性**:
  - level: 1
  - walkSpeed: 80.0
- **装备**:

#### 尼古拉斯

- **文件**: `Nicolas.tres`
- **ID**: `1948908174`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - race: Kralog
  - skintone: Fire
- **装备**:

#### 尼科斯

- **文件**: `Nickos.tres`
- **ID**: `3312816684`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - race: Human
  - skintone: Medium
- **装备**:

#### 巨型蛆

- **文件**: `Giant Maggot.tres`
- **ID**: `2624609623`
- **行为**: NEUTRAL, LEADER, SPAWNER
- **Boss**: 否
- **半径**: 25
- **属性**:
  - attack: 54
  - baseExp: 35
  - castAttackDelay: 1.1
  - cooldownAttackDelay: 1.2
  - defense: 20
  - level: 15
  - maxHealth: 370
  - walkSpeed: 50.0
- **装备**:

#### 帕丘亚

- **文件**: `Pachua.tres`
- **ID**: `3381597015`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - race: Human
  - skintone: Tanned
- **装备**:

#### 弗罗斯特

- **文件**: `Frost.tres`
- **ID**: `221388307`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - haircolor: Light Violet
  - hairstyle: Side Strand
  - race: Human
  - skintone: Dark
- **装备**:
  - CHEST: sub:shirt_salamander
  - LEGS: CloudPants
  - FEET: LeatherBoots
  - HEAD: sub:ghutra_sand

#### 强盗

- **文件**: `Bandit.tres`
- **ID**: `2834092087`
- **行为**: NEUTRAL, STEAL
- **Boss**: 否
- **半径**: 10
- **属性**:
  - attack: 55
  - baseExp: 80
  - defense: 10
  - dodgeRate: 0.28
  - level: 25
  - maxHealth: 380
  - mdefense: 5
  - walkSpeed: 120.0
- **装备**:

#### 彼得

- **文件**: `Peter.tres`
- **ID**: `232785317`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - race: Human
  - skintone: Dark
- **装备**:

#### 扎克贝尔

- **文件**: `Xakelbael.tres`
- **ID**: `2715760142`
- **行为**: AGGRESSIVE
- **Boss**: 是
- **半径**: 20
- **属性**:
  - baseExp: 50
  - defense: 0
  - gender: Male
  - level: 20
  - maxHealth: 250
  - race: Raijin
  - skintone: Dark
  - vitality: 5
- **装备**:
  - CHEST: DruidRobe
  - LEGS: LeatherPants
  - FEET: DesertBoots
  - HEAD: DruidHood
- **技能**:
  - Flar (概率: 0.3)
  - SonicWave (概率: 0.3)
  - Spitfire (概率: 0.3)
  - Jump (概率: 0.2)
  - Melee (概率: 0.0)

#### 排水史莱姆

- **文件**: `Drain Slime.tres`
- **ID**: `2469123693`
- **行为**: NEUTRAL, STEAL
- **Boss**: 否
- **父实体**: Slime
- **半径**: 10
- **属性**:
  - level: 1
  - walkSpeed: 70.0
- **装备**:

#### 斯普拉蒂娜

- **文件**: `Splatyna.tres`
- **ID**: `3314548081`
- **行为**: AGGRESSIVE, LEADER, SPAWNER
- **Boss**: 是
- **父实体**: Slime
- **半径**: 26
- **属性**:
  - baseExp: 20
  - level: 10
  - walkSpeed: 50.0
- **装备**:

#### 旧宝箱

- **文件**: `Old Chest.tres`
- **ID**: `4054136443`
- **行为**: NONE
- **Boss**: 否
- **父实体**: Chest
- **半径**: 26
- **装备**:

#### 明格尔

- **文件**: `Minguer.tres`
- **ID**: `670207772`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - haircolor: Dark Brown
  - hairstyle: Center Parting
  - race: Human
  - skintone: Medium
- **装备**:
  - CHEST: TankTop
  - LEGS: LinenShorts
  - FEET: DesertBoots
  - NECK: BandanaScarf

#### 本

- **文件**: `Ben.tres`
- **ID**: `193452314`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - haircolor: Dark Brown
  - hairstyle: Tapered Nape
  - race: Human
  - skintone: Medium
- **装备**:
  - CHEST: DesertArmor
  - LEGS: GuardCloudPants
  - FEET: LeatherBoots
  - HANDS: LeatherGloves
  - WEAPON: BoneKnife

#### 比西斯

- **文件**: `Bicies.tres`
- **ID**: `2843189460`
- **行为**: NONE
- **Boss**: 否
- **父实体**: WalkingNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - haircolor: Dark Brown
  - hairstyle: Center Parting
  - race: Human
  - skintone: Dark
- **装备**:
  - CHEST: VNeckTee
  - LEGS: Trousers

#### 毛绒绒

- **文件**: `Fluffy.tres`
- **ID**: `3003932433`
- **行为**: NONE
- **Boss**: 否
- **半径**: 16
- **属性**:
  - level: 1
  - walkSpeed: 50.0
- **装备**:

#### 毛罗

- **文件**: `Mauro.tres`
- **ID**: `229085321`
- **行为**: IMMOBILE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - haircolor: Twisted Teal
  - hairstyle: Choppy
  - race: Human
  - skintone: Dark
- **装备**:
  - CHEST: TankTop
  - LEGS: CloudPants
  - FEET: LeatherBoots
  - HANDS: LeatherGloves
  - NECK: BandanaScarf

#### 水井

- **文件**: `Well.tres`
- **ID**: `2089713785`
- **行为**: PACIFIST, IMMOBILE
- **Boss**: 否
- **父实体**: StaticObject
- **半径**: 32
- **装备**:

#### 沙暴矿井裂缝

- **文件**: `SandstormMineCrack.tres`
- **ID**: `434267405`
- **行为**: IMMOBILE
- **Boss**: 否
- **父实体**: sub:Resource_6o40j
- **半径**: 10
- **装备**:

#### 沙暴矿井钥匙

- **文件**: `SandstormMineKey.tres`
- **ID**: `3478975634`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticObject
- **半径**: 10
- **装备**:

#### 沙蛇

- **文件**: `Sand Snake.tres`
- **ID**: `2399925469`
- **行为**: AGGRESSIVE
- **Boss**: 否
- **半径**: 16
- **属性**:
  - attack: 85
  - baseExp: 60
  - castAttackDelay: 0.35
  - cooldownAttackDelay: 0.3
  - defense: 5
  - dodgeRate: 0.22
  - level: 19
  - maxHealth: 200
  - walkSpeed: 120.0
- **装备**:

#### 海兰

- **文件**: `Hyland.tres`
- **ID**: `3097291557`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - race: Human
  - skintone: Dark
- **装备**:

#### 海史莱姆

- **文件**: `Sea Slime.tres`
- **ID**: `1744780248`
- **行为**: NEUTRAL, STEAL
- **Boss**: 否
- **父实体**: Slime
- **半径**: 10
- **属性**:
  - attack: 20
  - baseExp: 13
  - castAttackDelay: 1.0
  - cooldownAttackDelay: 1.5
  - defense: 15
  - level: 6
  - maxHealth: 200
  - mdefense: 2
  - walkSpeed: 40.0
- **装备**:

#### 淤泥史莱姆

- **文件**: `Sludge Slime.tres`
- **ID**: `2450367875`
- **行为**: NEUTRAL, STEAL
- **Boss**: 否
- **父实体**: Slime
- **半径**: 10
- **属性**:
  - level: 1
  - walkSpeed: 50.0
- **装备**:

#### 火焰哥布林

- **文件**: `Fire Goblin.tres`
- **ID**: `1836204198`
- **行为**: NONE
- **Boss**: 否
- **半径**: 16
- **属性**:
  - attack: 15
  - baseExp: 15
  - defense: 20
  - level: 8
  - mattack: 15
  - maxHealth: 75
  - mdefense: 20
  - walkSpeed: 80.0
- **装备**:
- **技能**:
  - Spitfire (概率: 2.0)

#### 火焰骷髅

- **文件**: `Fire Skull.tres`
- **ID**: `4234567318`
- **行为**: AGGRESSIVE
- **Boss**: 否
- **半径**: 12
- **属性**:
  - baseExp: 20
  - level: 10
  - walkSpeed: 50.0
- **装备**:
- **技能**:
  - Flar (概率: 1.0)

#### 灵魂巨石

- **文件**: `Soul Menhir.tres`
- **ID**: `737772331`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 32
- **装备**:

#### 物体

- **文件**: `Object.tres`
- **ID**: `3343895260`
- **行为**: NONE
- **Boss**: 否
- **半径**: 16
- **装备**:

#### 猞猁

- **文件**: `Lynx.tres`
- **ID**: `2089340336`
- **行为**: AGGRESSIVE
- **Boss**: 否
- **半径**: 24
- **属性**:
  - attack: 40
  - baseExp: 55
  - cooldownAttackDelay: 1.5
  - defense: 15
  - dodgeRate: 0.22
  - level: 21
  - mattack: 60
  - maxHealth: 250
  - mdefense: 25
  - walkSpeed: 160.0
- **装备**:
- **技能**:
  - SonicWave (概率: 10.0)

#### 猫

- **文件**: `Neko.tres`
- **ID**: `2089390322`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - haircolor: Dark Brown
  - hairstyle: Tapered Nape
  - race: Human
  - skintone: Medium
- **装备**:
  - CHEST: LeatherShirt
  - LEGS: LeatherPants
  - FEET: LeatherBoots
  - HANDS: LeatherGloves
  - WEAPON: Gladius

#### 玛纳伊尔女德鲁伊

- **文件**: `ManayirDruidess.tres`
- **ID**: `608165177`
- **行为**: NONE
- **Boss**: 否
- **半径**: 20
- **属性**:
  - gender: Female
  - race: Raijin
  - skintone: Light
- **装备**:
  - CHEST: DruidRobe
  - LEGS: LeatherPants
  - FEET: DesertBoots
  - HEAD: DruidHood

#### 玛纳伊尔德鲁伊

- **文件**: `ManayirDruid.tres`
- **ID**: `999510798`
- **行为**: NONE
- **Boss**: 否
- **半径**: 20
- **属性**:
  - gender: Male
  - race: Human
  - skintone: Tanned
- **装备**:
  - CHEST: DruidRobe
  - LEGS: LeatherPants
  - FEET: DesertBoots
  - HEAD: DruidHood

#### 玩家

- **文件**: `Player.tres`
- **ID**: `3394588274`
- **行为**: NONE
- **Boss**: 否
- **半径**: 10
- **属性**:
  - level: 1
  - walkSpeed: 160.0
- **装备**:

#### 瑞安

- **文件**: `Ryan.tres`
- **ID**: `2089555519`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - race: Human
  - skintone: Dark
- **装备**:

#### 白

- **文件**: `Shiro.tres`
- **ID**: `236439338`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - haircolor: Dark Brown
  - hairstyle: Tapered Nape
  - race: Human
  - skintone: Dark
- **装备**:
  - CHEST: DesertJacket
  - LEGS: Trousers
  - FEET: DesertBoots
  - HEAD: Ghutra
  - NECK: BandanaScarf

#### 皮乌

- **文件**: `Piou.tres`
- **ID**: `2089466690`
- **行为**: NEUTRAL, FLEE
- **Boss**: 否
- **半径**: 8
- **属性**:
  - attack: 5
  - baseExp: 1
  - defense: 1
  - level: 1
  - maxHealth: 30
  - walkSpeed: 30.0
- **装备**:

#### 盐史莱姆

- **文件**: `Salt Slime.tres`
- **ID**: `2250973363`
- **行为**: NEUTRAL, STEAL
- **Boss**: 否
- **父实体**: Slime
- **半径**: 10
- **属性**:
  - level: 1
  - walkSpeed: 40.0
- **装备**:

#### 盗贼宝箱

- **文件**: `Thiefs Chest.tres`
- **ID**: `970294591`
- **行为**: NONE
- **Boss**: 否
- **父实体**: Chest
- **半径**: 26
- **装备**:

#### 红女王

- **文件**: `Red Queen.tres`
- **ID**: `3662818686`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Female
  - haircolor: Pure Platinum
  - hairstyle: Long and Curly
  - race: Kralog
  - skintone: Fire
- **装备**:
  - CHEST: Gown
  - LEGS: GuardCloudPants
  - FEET: DaimBoots
  - HANDS: GoldBracelets
  - HEAD: TonoriCrown

#### 红蝎

- **文件**: `Red Scorpion.tres`
- **ID**: `825312237`
- **行为**: AGGRESSIVE
- **Boss**: 否
- **父实体**: Scorpion
- **半径**: 0
- **属性**:
  - attack: 55
  - baseExp: 30
  - defense: 5
  - level: 15
  - maxHealth: 500
  - mdefense: 5
- **装备**:
- **技能**:
  - Poison (概率: 0.25)

#### 纳姆

- **文件**: `Naem.tres`
- **ID**: `2089385766`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - race: Human
  - skintone: Medium
- **装备**:

#### 纳德

- **文件**: `Nard.tres`
- **ID**: `2089386186`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - haircolor: Navy Blue
  - hairstyle: Mane
  - race: Human
  - skintone: Tanned
- **装备**:
  - CHEST: sub:tanktop_salamander
  - LEGS: CloudPants
  - FEET: sub:boots_sand
  - HANDS: SilverBracelets

#### 维

- **文件**: `Wie.tres`
- **ID**: `193475306`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Female
  - race: Raijin
  - skintone: Light
- **装备**:

#### 船

- **文件**: `Ship.tres`
- **ID**: `2089573209`
- **行为**: NONE
- **Boss**: 否
- **半径**: 16
- **属性**:
  - attack: 0.0
  - defense: 0.0
  - walkSpeed: 20.0
- **装备**:

#### 艾基努

- **文件**: `Ekinu.tres`
- **ID**: `219944129`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - race: Human
  - skintone: Tanned
- **装备**:

#### 艾拉诺

- **文件**: `Elanore.tres`
- **ID**: `3325600555`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Female
  - race: Human
  - skintone: Tanned
- **装备**:

#### 莉拉

- **文件**: `Lilah.tres`
- **ID**: `228176527`
- **行为**: NEUTRAL, FOLLOWER, STEAL
- **Boss**: 否
- **父实体**: Slime
- **半径**: 10
- **属性**:
  - level: 1
  - walkSpeed: 40.0
- **装备**:

#### 莉赞德

- **文件**: `Lizandre.tres`
- **ID**: `1435506622`
- **行为**: NEUTRAL, FOLLOWER
- **Boss**: 否
- **半径**: 16
- **属性**:
  - attack: 55
  - baseExp: 25
  - cooldownAttackDelay: 1.5
  - defense: 8
  - level: 12
  - maxHealth: 220
  - walkSpeed: 130.0
- **装备**:

#### 莉赞德拉

- **文件**: `Lizandra.tres`
- **ID**: `1435506618`
- **行为**: AGGRESSIVE, LEADER
- **Boss**: 否
- **半径**: 16
- **属性**:
  - attack: 65
  - baseExp: 40
  - defense: 15
  - level: 14
  - maxHealth: 300
  - walkSpeed: 130.0
- **装备**:

#### 萨威尔

- **文件**: `Sawyl.tres`
- **ID**: `236203253`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Female
  - race: Human
  - skintone: Medium
- **装备**:

#### 蛆

- **文件**: `Maggot.tres`
- **ID**: `3264333316`
- **行为**: NEUTRAL, FLEE
- **Boss**: 否
- **半径**: 10
- **属性**:
  - attack: 10
  - baseExp: 1
  - defense: 1
  - level: 1
  - maxHealth: 50
  - walkSpeed: 50.0
- **装备**:

#### 蛇

- **文件**: `Snake.tres`
- **ID**: `236646007`
- **行为**: AGGRESSIVE
- **Boss**: 否
- **半径**: 16
- **属性**:
  - attack: 40
  - baseExp: 40
  - castAttackDelay: 0.1
  - cooldownAttackDelay: 0.6
  - critRate: 0.0
  - defense: 5
  - dodgeRate: 0.0
  - level: 17
  - maxHealth: 150
  - walkSpeed: 140.0
- **装备**:

#### 蛇坑池塘

- **文件**: `Snake Pit Pond.tres`
- **ID**: `1190851605`
- **行为**: PACIFIST, IMMOBILE
- **Boss**: 否
- **父实体**: StaticObject
- **半径**: 20
- **装备**:

#### 蛇坑线索

- **文件**: `Snake Pit Clue.tres`
- **ID**: `1190381389`
- **行为**: PACIFIST, IMMOBILE
- **Boss**: 否
- **父实体**: StaticObject
- **半径**: 8
- **装备**:

#### 蝎子

- **文件**: `Scorpion.tres`
- **ID**: `3842739378`
- **行为**: NONE
- **Boss**: 否
- **半径**: 18
- **属性**:
  - attack: 22
  - baseExp: 8
  - defense: 0
  - level: 4
  - maxHealth: 200
  - walkSpeed: 60.0
- **装备**:

#### 蝙蝠

- **文件**: `Bat.tres`
- **ID**: `193452188`
- **行为**: NONE
- **Boss**: 否
- **半径**: 12
- **属性**:
  - attack: 10
  - baseExp: 6
  - castAttackDelay: 0.4
  - cooldownAttackDelay: 0.3
  - defense: 2
  - dodgeRate: 0.5
  - level: 3
  - mattack: 8
  - maxHealth: 90
  - walkSpeed: 120.0
- **装备**:
- **技能**:
  - SonicScream (概率: 1.0)

#### 行走NPC

- **文件**: `WalkingNPC.tres`
- **ID**: `2377595219`
- **行为**: PACIFIST
- **Boss**: 否
- **父实体**: NPC
- **半径**: 0
- **属性**:
  - walkSpeed: 160.0
- **装备**:

#### 邪恶蘑菇

- **文件**: `Evil Mushroom.tres`
- **ID**: `2866154511`
- **行为**: NONE
- **Boss**: 否
- **半径**: 16
- **属性**:
  - level: 1
  - walkSpeed: 80.0
- **装备**:

#### 里斯基姆

- **文件**: `Riskim.tres`
- **ID**: `3469933044`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - race: Raijin
  - skintone: Light
- **装备**:

#### 锁

- **文件**: `Locks.tres`
- **ID**: `228382689`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticObject
- **半径**: 26
- **装备**:

#### 静态NPC

- **文件**: `StaticNPC.tres`
- **ID**: `89439950`
- **行为**: PACIFIST, IMMOBILE
- **Boss**: 否
- **父实体**: NPC
- **半径**: 0
- **装备**:

#### 静态物体

- **文件**: `StaticObject.tres`
- **ID**: `1629942628`
- **行为**: PACIFIST, IMMOBILE
- **Boss**: 否
- **父实体**: Object
- **半径**: 0
- **装备**:

#### 面粉桶

- **文件**: `Flour Barrel.tres`
- **ID**: `1902637957`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticObject
- **半径**: 0
- **装备**:

#### 马克拉特

- **文件**: `Maclat.tres`
- **ID**: `3264194551`
- **行为**: NONE
- **Boss**: 否
- **父实体**: StaticNPC
- **半径**: 0
- **属性**:
  - gender: Male
  - race: Kralog
  - skintone: Fire
- **装备**:

#### 马文

- **文件**: `Marvin.tres`
- **ID**: `3264744754`
- **行为**: AGGRESSIVE, LEADER, SPAWNER, STEAL
- **Boss**: 否
- **父实体**: Slime
- **半径**: 10
- **属性**:
  - baseExp: 10
  - level: 5
  - walkSpeed: 40.0
- **装备**:

#### 骷髅

- **文件**: `Skeleton.tres`
- **ID**: `890728458`
- **行为**: AGGRESSIVE
- **Boss**: 否
- **半径**: 10
- **属性**:
  - baseExp: 45
  - defense: 0
  - level: 20
  - maxHealth: 300
  - mdefense: 35
  - walkSpeed: 100.0
- **装备**:
  - CHEST: LeatherShirt
  - WEAPON: BoneKnife
  - SHIELD: DesertShield

#### 魔力树

- **文件**: `Mana Tree.tres`
- **ID**: `2025440114`
- **行为**: PACIFIST, IMMOBILE
- **Boss**: 否
- **半径**: 16
- **装备**:

#### 鳄鱼

- **文件**: `Croc.tres`
- **ID**: `2089009292`
- **行为**: AGGRESSIVE, FLEE
- **Boss**: 否
- **半径**: 10
- **属性**:
  - attack: 8
  - baseExp: 10
  - castAttackDelay: 0.4
  - cooldownAttackDelay: 0.5
  - defense: 40
  - dodgeRate: 0.2
  - level: 5
  - maxHealth: 55
  - walkSpeed: 90.0
- **装备**:

#### 鸟

- **文件**: `Bird.tres`
- **ID**: `2088963654`
- **行为**: NEUTRAL, STEAL
- **Boss**: 否
- **半径**: 20
- **属性**:
  - attack: 35
  - baseExp: 110
  - castAttackDelay: 0.25
  - cooldownAttackDelay: 0.2
  - defense: 15
  - dodgeRate: 0.28
  - level: 27
  - maxHealth: 220
  - walkSpeed: 200.0
- **装备**:

#### 鼠

- **文件**: `Ratto.tres`
- **ID**: `235013903`
- **行为**: NEUTRAL, FLEE
- **Boss**: 否
- **半径**: 10
- **属性**:
  - attack: 10
  - baseExp: 10
  - castAttackDelay: 0.3
  - cooldownAttackDelay: 0.25
  - defense: 2
  - dodgeRate: 0.33
  - level: 5
  - maxHealth: 85
  - walkSpeed: 150.0
- **装备**:

---

## 5. 游戏掉率数据

共 **12** 个实体有掉落物

| 实体 | 掉落物品 | 掉率 |
|------|---------|------|
| 佩约特 | Pitaya | 33.0% |
| 加百列 | GabrielKey | 100.0% |
| 多里安 | DorianKey | 100.0% |
| 巨型蛆 | MaggotSlime | 60.0% |
| 沙蛇 | SnakeSkin | 5.0% |
| 海史莱姆 | Salt | 33.0% |
| 火焰哥布林 | SulphurPowder | 20.0% |
| 盐史莱姆 | Apple | 70.0% |
| 蛆 | MaggotSlime | 60.0% |
| 蛇 | SnakeSkin | 10.0% |
| 马文 | MarvinKey | 100.0% |
| 骷髅 | Bone | 33.0% |

### 实体生成数据

| 实体 | 生成实体 | 数量 |
|------|---------|------|
| 巨型蛆 | Maggot | 4 |
| 斯普拉蒂娜 | Lilah | 3 |
| 斯普拉蒂娜 | Lulea | 3 |
| 马文 | Lilah | 2 |

---

## 6. 游戏状态机

### AICommons

**Behaviour**:

| 名称 | 值 |
|------|-----|
| AGGRESSIVE | 1 |
| FLEE | 1 |
| FOLLOWER | 1 |
| IMMOBILE | 1 |
| LEADER | 1 |
| NEUTRAL | 1 |
| NONE | 0 |
| PACIFIST | 1 |
| SPAWNER | 1 |
| STEAL | 1 |

**State**:

| 名称 | 值 |
|------|-----|
| ATTACK | 2 |
| COUNT | 4 |
| HALT | 3 |
| IDLE | 0 |
| WALK | 1 |

### ActorCommons

**Alteration**:

| 名称 | 值 |
|------|-----|
| CRIT | 2 |
| DODGE | 4 |
| EXP | 6 |
| GP | 7 |
| HEAL | 5 |
| HIT | 0 |
| LVL_UP | 8 |
| MISS | 3 |
| QUEST_COMPLETE | 10 |
| QUEST_UPDATE | 11 |
| SKILL_UP | 9 |
| WHISPER | 12 |

**Attribute**:

| 名称 | 值 |
|------|-----|
| AGILITY | 2 |
| CONCENTRATION | 4 |
| ENDURANCE | 3 |
| STRENGTH | 0 |
| VITALITY | 1 |

**Direction**:

| 名称 | 值 |
|------|-----|
| DOWN | 1 |
| DOWNLEFT | 2 |
| DOWNRIGHT | 8 |
| LEFT | 3 |
| RIGHT | 7 |
| UP | 5 |
| UPLEFT | 4 |
| UPRIGHT | 6 |

**Gender**:

| 名称 | 值 |
|------|-----|
| COUNT | 3 |
| FEMALE | 1 |
| MALE | 0 |
| NONBINARY | 2 |

**Permission**:

| 名称 | 值 |
|------|-----|
| ADMIN | 3 |
| GM | 2 |
| MODERATOR | 1 |
| NONE | 0 |

**Recovery**:

| 名称 | 值 |
|------|-----|
| COUNT | 4 |
| GOOD | 3 |
| NONE | 0 |
| NORMAL | 2 |
| POOR | 1 |

**Slot**:

| 名称 | 值 |
|------|-----|
| BODY | 9 |
| CHEST | 0 |
| COUNT | 13 |
| FACE | 10 |
| FEET | 3 |
| HAIR | 11 |
| HANDS | 4 |
| HEAD | 5 |
| LEGS | 2 |
| NECK | 6 |
| QUEST | 12 |
| SHIELD | 8 |
| WEAPON | 7 |

**State**:

| 名称 | 值 |
|------|-----|
| ATTACK | 4 |
| COUNT | 9 |
| DEATH | 5 |
| FROM_TRIGGER | 8 |
| IDLE | 0 |
| SIT | 3 |
| TO_TRIGGER | 6 |
| TRIGGER | 7 |
| WALK | 2 |

**Target**:

| 名称 | 值 |
|------|-----|
| ALLY | 1 |
| ENEMY | 2 |
| NONE | 0 |

**Type**:

| 名称 | 值 |
|------|-----|
| MONSTER | 2 |
| NPC | 3 |
| PLAYER | 0 |

### CellCommons

**Modifier**:

| 名称 | 值 |
|------|-----|
| Accuracy | 23 |
| Attack | 20 |
| AttackRange | 17 |
| CastDelay | 15 |
| CooldownDelay | 11 |
| Count | 24 |
| CritRate | 6 |
| Defense | 14 |
| DodgeRate | 16 |
| Health | 1 |
| Hide | 21 |
| Invisible | 22 |
| MAttack | 7 |
| MDefense | 8 |
| Mana | 2 |
| MaxHealth | 12 |
| MaxMana | 4 |
| MaxStamina | 9 |
| None | 0 |
| RegenHealth | 13 |
| RegenMana | 5 |
| RegenStamina | 10 |
| Stamina | 3 |
| WalkSpeed | 18 |
| WeightCapacity | 19 |

**Type**:

| 名称 | 值 |
|------|-----|
| COUNT | 3 |
| EMOTE | 1 |
| ITEM | 0 |
| SKILL | 2 |

### ProgressCommons

**DESERT_SEED**:

| 名称 | 值 |
|------|-----|
| SEEK_MANAYIR | 2 |
| SEEK_NINA | 1 |

**ELANORE_POTION**:

| 名称 | 值 |
|------|-----|
| STARTED | 1 |

**GRAIN_IN_THE_SAND**:

| 名称 | 值 |
|------|-----|
| SEARCHED_CRATES | 2 |
| STARTED | 1 |

**MINE_EXPLORATION**:

| 名称 | 值 |
|------|-----|
| DEFEATED | 5 |
| FIGHTING | 4 |
| FIND_NICKOS | 2 |
| MANA_TREE_MET | 6 |
| STARTED | 1 |
| STRANGER_SPOTTED | 3 |

**NINA_HUNGRY**:

| 名称 | 值 |
|------|-----|
| STARTED | 1 |

**SANDSTORM_MINE_ABANDONED_TREASURE**:

| 名称 | 值 |
|------|-----|
| KEY_FOUND | 1 |

**SANDSTORM_NAEM_HELMET**:

| 名称 | 值 |
|------|-----|

**SANDSTORM_NATHAN_WATER**:

| 名称 | 值 |
|------|-----|
| STARTED | 1 |

**SHIP_PETER_BOUNTY**:

| 名称 | 值 |
|------|-----|
| WAVE_ONE_DONE | 1 |
| WAVE_TWO_DONE | 2 |

**SNAKE_PIT_BITING_THIRST**:

| 名称 | 值 |
|------|-----|
| STARTED | 1 |

**SNAKE_PIT_THIEF**:

| 名称 | 值 |
|------|-----|
| ALL_CLUES_FOUND | 31 |
| RIDDLE_SOLVED | 32 |

**SPLATYNA_OFFERING**:

| 名称 | 值 |
|------|-----|
| STARTED | 1 |

**TULIMSHAR_EASTERN_HILLS_CHEST**:

| 名称 | 值 |
|------|-----|

**TULIMSHAR_GLASSMAKING**:

| 名称 | 值 |
|------|-----|
| STARTED | 1 |

**TULIMSHAR_OLD_FRIENDSHIP**:

| 名称 | 值 |
|------|-----|
| ENVELOPES_FOUND | 2 |
| LETTERS_DELIVERED | 3 |
| STARTED | 1 |

**TUTORIAL**:

| 名称 | 值 |
|------|-----|
| CLOTHES_GIVEN | 3 |
| ELANORE_DONE | 5 |
| INTRO_ITEMS_GIVEN | 1 |
| KAEL_DONE | 7 |
| KAEL_MET | 6 |
| POTION_GIVEN | 2 |
| UI_EXPLAINED | 4 |

**VETERAN_CAVE_EXPLORER**:

| 名称 | 值 |
|------|-----|
| UNCLAIMED | 1 |

### Skill

**TargetMode**:

| 名称 | 值 |
|------|-----|
| SELF | 2 |
| SINGLE | 0 |
| ZONE | 1 |

### CellCommons.Modifier (修饰符)

| 名称 | 值 |
|------|-----|
| None | 0 |
| Health |  |
| Mana |  |
| Stamina |  |
| MaxMana |  |
| RegenMana |  |
| CritRate |  |
| MAttack |  |
| MDefense |  |
| MaxStamina |  |
| RegenStamina |  |
| CooldownDelay |  |
| MaxHealth |  |
| RegenHealth |  |
| Defense |  |
| CastDelay |  |
| DodgeRate |  |
| AttackRange |  |
| WalkSpeed |  |
| WeightCapacity |  |
| Attack |  |
| Hide |  |
| Invisible |  |
| Accuracy |  |
| Count |  |

---

## 7. NPC 对话数据

共 **1147** 条对话，分布在 **64** 个 NPC 脚本中

| NPC 脚本 | 对话行数 |
|---------|---------|
| tonori/tulimshar/Nina.gd | 114 |
| tonori/tulimshar/Ekinu.gd | 65 |
| tonori/tulimshar/Kael.gd | 65 |
| tonori/tulimshar/Elanore.gd | 63 |
| tonori/tulimshar/Hyland.gd | 60 |
| tonori/sandstorm/Ryan.gd | 58 |
| tonori/tulimshar/Frost.gd | 55 |
| tonori/tulimshar/Eridu.gd | 44 |
| tonori/sandstorm/Nathan.gd | 39 |
| tonori/tulimshar/Riskim.gd | 38 |
| tonori/sandstorm/ManaTree.gd | 37 |
| candor/Gabz.gd | 36 |
| tonori/tulimshar/Wie.gd | 34 |
| tonori/tulimshar/Nard.gd | 33 |
| tonori/sandstorm/Nickos.gd | 28 |
| splatyna/Bicies.gd | 27 |
| tonori/manayir/Hommer.gd | 27 |
| tonori/desertpit/Mauro.gd | 26 |
| tonori/tulimshar/Ben.gd | 23 |
| ship/Peter.gd | 22 |
| tonori/desertpit/Pachua.gd | 21 |
| tonori/tulimshar/Gilgames.gd | 21 |
| tonori/sandstorm/Sawyl.gd | 19 |
| tonori/tulimshar/Aanshir.gd | 17 |
| tonori/sandstorm/Xakelbael.gd | 15 |
| tonori/tulimshar/Maci.gd | 14 |
| ship/JohanneLaLiberte-OMana.gd | 13 |
| tonori/tulimshar/Shimris.gd | 12 |
| tonori/tulimshar/Andi.gd | 11 |
| tonori/sandstorm/Naem.gd | 9 |
| generic/Barber.gd | 8 |
| tonori/sandstorm/Elias.gd | 8 |
| generic/Well.gd | 7 |
| ship/Nard-PoemAboutPoems.gd | 6 |
| splatyna/Locks.gd | 6 |
| tonori/sandstorm/Dausen.gd | 6 |
| generic/SoulMenhir.gd | 5 |
| tonori/sandstorm/DeepLevelEntrance.gd | 5 |
| tonori/sandstorm/Sema.gd | 4 |
| splatyna/Chest.gd | 3 |
| tonori/desertpit/ThiefsChest.gd | 3 |
| tonori/sandstorm/Chest.gd | 3 |
| tonori/tulimshar/FlourBarrel.gd | 3 |
| tonori/tulimshar/LetterStash.gd | 3 |
| tonori/tulimshar/PatrolGuardCaught.gd | 3 |
| generic/DryWell.gd | 2 |
| generic/MerchantScript.gd | 2 |
| ship/Nard.gd | 2 |
| tonori/desertpit/Clue1.gd | 2 |
| tonori/desertpit/Clue2.gd | 2 |
| tonori/desertpit/Clue3.gd | 2 |
| tonori/desertpit/Clue4.gd | 2 |
| tonori/desertpit/Clue5.gd | 2 |
| tonori/tulimshar/castle/Guard.gd | 2 |
| generic/Warp.gd | 1 |
| tonori/desertpit/WaterPond.gd | 1 |
| tonori/desertpit/WaterPondFilthy.gd | 1 |
| tonori/tulimshar/Barrel.gd | 1 |
| tonori/tulimshar/Maclat.gd | 1 |
| tonori/tulimshar/Minguer.gd | 1 |
| tonori/tulimshar/Neko.gd | 1 |
| tonori/tulimshar/Shiro.gd | 1 |
| tonori/tulimshar/castle/RedQueen.gd | 1 |
| tonori/tulimshar outskirts/Chest.gd | 1 |

### 对话详情

#### Gabz

*来源: `candor/Gabz.gd`*

- 💬 Mes (L13): 欢迎来到坎多尔之心，魔力来此消亡！
- 💬 Mes (L23): 这就是坎多尔之心。这是一个洞穴系统，一直含有高浓度的卡厄。
- 💬 Mes (L24): 当魔力流过埃米尔时，部分能量会衰减并沉积在特定区域。
- 💬 Mes (L25): 这就是那些区域之一。魔力停止流动并变成卡厄的地方。
- 💬 Mes (L34): 我们站着的棋盘是一个催化剂，利用这个洞穴的能量召唤各种生物。
- 💬 Mes (L35): 它们会一波一波地来，试图消耗你。
- 💬 Mes (L36): 如果你撑过足够多的波次，召唤就会停止，你就征服了坎多尔之心。
- 💬 Mes (L46): 我从没想过那个……我不知道！我想我一直都在这里？
- 💬 Mes (L48): 我是加兹。
- 💬 Mes (L52): 没人真正知道。有人说它的隧道通向埃米尔的核心，里面全是卡厄。
- 💬 Mes (L53): 其他人认为这块岩石的某种特性从四面八方吸引卡厄。
- 💬 Mes (L54): 我不问自己太多问题。
- 💬 Mes (L58): 你运气好，我在这里掌握着生死大权！
- 💬 Mes (L59): 当你在这个洞穴中死亡时，我会用你的齐埃尔石护身符在灵魂巨石处恢复你的生命本质。
- 💬 Mes (L60): 如果我不救你，我很快就没棋子可玩了。
- 💬 Mes (L64): 高浓度的魔力或许能改变能量的平衡。
- 💬 Mes (L65): 但我是来抵挡卡厄的，为什么不让我享受一下呢？
- 💬 Mes (L69): 卡厄可以让生物变得敌对，随时间推移，甚至变成亡灵。
- 💬 Mes (L70): 它驱使它们攻击生物及其与魔力的联系。
- 💬 Mes (L71): 如果你待太久，你可能也会感觉到。
- 💬 Mes (L80): 战斗已经开始了。你现在不能再开一场。
- 💬 Mes (L83): 开始前你需要给我带来5个苹果作为贡品。
- 💬 Mes (L85): 你不够强，打不了这场。到达 %d 级再来。
- 💭 Chat (L11): 专注于你的任务！
- 💭 Chat (L88): 啊，你准备好了！战斗将在10秒后开始，做好准备！
- 🔹 Choice (L17): Let's get this started, Gabz.
- 🔹 Choice (L18): I changed my mind, I need to go.
- 🔹 Choice (L19): What is this place?
- 🔹 Choice (L20): Do you know where these waves of monsters come from?
- 🔹 Choice (L29): No more questions.
- 🔹 Choice (L30): Who are you? You seem off.
- 🔹 Choice (L31): Why is the cave like this?
- 🔹 Choice (L40): No more questions.
- 🔹 Choice (L41): What happens if I fail?
- 🔹 Choice (L42): Is there no way to cleanse the corruption?
- 🔹 Choice (L43): Why does Kaore make things this way?

#### Barber

*来源: `generic/Barber.gd`*

- 💬 Mes (L5): 你今天想换发型或发色吗？
- 🔹 Choice (L6): I want to try another style
- 🔹 Choice (L7): A new color
- 🔹 Choice (L8): None
- 🔹 Choice (L21): Another style
- 🔹 Choice (L22): Perfect!
- 🔹 Choice (L35): Another color
- 🔹 Choice (L36): Perfect!

#### DryWell

*来源: `generic/DryWell.gd`*

- 💬 Mes (L5): 你把桶放下去，但捞上来是空的。
- 💬 Mes (L6): 这口井已经干了。

#### MerchantScript

*来源: `generic/MerchantScript.gd`*

- 🔹 Choice (L118): 交易
- 🔹 Choice (L119): 取消

#### SoulMenhir

*来源: `generic/SoulMenhir.gd`*

- 📖 Narrate (L7): 一块宏伟的雕刻石，包裹着流动的魔力能量。它的存在让人感到平静。
- 📖 Narrate (L8): 银色的细脉贯穿深灰色的石头，随着原生魔力的蓝光流过而变换色彩。
- 📖 Narrate (L10): 你想触摸灵魂巨石吗？
- 🔹 Choice (L11): Touch the Soul Menhir.
- 🔹 Choice (L12): 别管它。

#### Warp

*来源: `generic/Warp.gd`*

- 🔹 Choice (L8): 取消

#### Well

*来源: `generic/Well.gd`*

- 💬 Mes (L15): 你需要一个瓶子才能从这口井打水。
- 💬 Mes (L17): 你的背包没有可用空间。
- 💬 Mes (L19): 装水时别动……
- 💬 Mes (L47): 你的瓶子现在装满了清水。你还有一个空瓶。
- 💬 Mes (L51): 你的瓶子现在装满了清水。
- 🔹 Choice (L48): Fill another bottle
- 🔹 Choice (L49): 离开

#### JohanneLaLiberte-OMana

*来源: `ship/JohanneLaLiberte-OMana.gd`*

- 💬 Mes (L17): 约翰娜·拉利贝尔特，2011年
- 📖 Narrate (L5): 噢魔力，我乘船渡海只是为了寻找你。
- 📖 Narrate (L6): 噢魔力，我怎样才能成为这个英雄？你在这个悲伤的世界里想要我做什么？
- 📖 Narrate (L7): 噢魔力，这是不是在战斗后找到知识之路的关键，还是别的什么？
- 📖 Narrate (L8): 噢魔力，请听我说，告诉我如何更强，给我指明道路。
- 📖 Narrate (L9): 噢魔力，我只想停下来看看周围，看看有没有美好的人。
- 📖 Narrate (L10): 噢魔力，也许我能在这个世界上找到我的道路，与朋友们并肩作战。
- 📖 Narrate (L11): 噢魔力，从一个镇到另一个镇，我找到了他，他经过我的路，成了我最好的朋友和丈夫
- 📖 Narrate (L12): 噢魔力，我们游荡、分享，穿过林地。
- 📖 Narrate (L13): 噢魔力，他给了我信心，我感觉恐惧变少了。
- 📖 Narrate (L14): 噢魔力。在一起，我们可以帮助和保护他人免受黑暗阴影的侵袭。
- 📖 Narrate (L15): 噢魔力，现在我有了继续到最后的勇气。
- 📖 Narrate (L16): 噢魔力，事实上，正是通过你我们才能找到友谊来创造魔力世界！

#### Nard-PoemAboutPoems

*来源: `ship/Nard-PoemAboutPoems.gd`*

- 💬 Mes (L10): 纳德，2011年
- 📖 Narrate (L5): 诗让文字起舞\n文字变成音乐和目光\n在星夜下的恋人之上\n他们的眼睛在月光下闪烁
- 📖 Narrate (L6): 不管语法\n如果文字随节奏跳动\n找到它们的秩序\n像铁砧上锤子的敲击
- 📖 Narrate (L7): 不管拼写\n只管说出闪亮的话\n不然，疯狂的凯蒂克奇\n怎么向迷人的格利克奇表达爱意？
- 📖 Narrate (L8): 真或不真\n那不是问题\n押韵也不难\n如果你不喜欢\n分享你的想法就是你的使命\n无论内心是什么
- 📖 Narrate (L9): 别害羞，你很好\n打破心中的坚冰\n无论你的想法是什么\n都是好主意，嗯哼！\n让每个人\n都能写诗

#### Nard

*来源: `ship/Nard.gd`*

- 💬 Mes (L5): 你想继续航行？
- 💬 Mes (L6): 去上层甲板掌舵。我和船员负责帆。

#### Peter

*来源: `ship/Peter.gd`*

- 💬 Mes (L12): 你太小了不能在这里闲逛，先上学再说。
- 💬 Mes (L14): 你在这里干什么？活在下面。快去！
- 💬 Mes (L16): 嗯？快点，我很忙。
- 💬 Mes (L26): 有什么东西一直在啃货舱的储物。我打赌又是老鼠……我需要有人去对付它们。
- 💬 Mes (L31): 活儿在这儿。
- 💬 Mes (L38): 又来了？好。货舱可不会自己清理干净。
- 💬 Mes (L44): 老鼠找到了能啃的东西。清理掉它们，你会得到报酬。
- 💬 Mes (L47): 我发现了一个新巢穴，下面现在更多了，也更胆大了。这次报酬更好。
- 💬 Mes (L50): 别管老鼠了。下面现在有蝙蝠。蝙蝠，你敢信？
- 💬 Mes (L56): 进去后快关门，那里的空气很恶心。
- 💬 Mes (L62): 又是你？好。老鼠不知道什么时候放弃，它们回来了。
- 💬 Mes (L63): 跟往常一样。下去清理干净，上来拿钱。
- 💬 Mes (L67): 又是你？今天货舱很安静，我不需要你帮忙了，你可以走了。
- 💭 Chat (L35): 随你便。
- 🔹 Choice (L27): I could use the practice, I'm in.
- 🔹 Choice (L28): Not my problem.
- 🔹 Choice (L45): Consider it done.
- 🔹 Choice (L48): I'll finish what I started.
- 🔹 Choice (L51): Bats it is.
- 🔹 Choice (L53): Not right now.
- 🔹 Choice (L64): On my way.
- 🔹 Choice (L65): Not today.

#### Bicies

*来源: `splatyna/Bicies.gd`*

- 💬 Mes (L11): 赞美斯普拉蒂娜，强大的史莱姆女王！
- 💬 Mes (L12): 你，那边！你是来给我们黏液女士献上金子的吗？
- 💬 Mes (L30): 你还没向斯普拉蒂娜献上财宝？
- 💬 Mes (L31): 进入她的洞穴，找到她的追随者，把金子交给她！
- 💬 Mes (L32): 三个忠诚的握着钥匙，记住！
- 💬 Mes (L36): 等等……我听到什么。一声尖叫！
- 💬 Mes (L37): 斯普拉蒂娜出事了？！你干了什么？
- 💬 Mes (L43): 好，好！我会在这里确保女王开心。
- 💬 Mes (L46): 没了？！不，不！你撒谎！斯普拉蒂娜不会死的！
- 💬 Mes (L47): 快……走，趁我还没发疯！
- 💬 Mes (L50): 斯普拉蒂娜和其他史莱姆不一样。她很强大！她被卡厄祝福了！
- 💬 Mes (L51): 她曾经是一位伟大的女王。她的灵魂被放逐，来到了这里。现在她可以再次成为伟大的女王！
- 💬 Mes (L52): 我的女王喜欢黄金。它让她的新形态如此美丽！她更喜欢那样。
- 💬 Mes (L53): 我们必须给她要求的金子。如果我们惹怒她，她会吞噬我们所有人！
- 💬 Mes (L57): 这是斯普拉蒂娜的黄金王国！它曾经只是一个洞穴，但现在我的女王统治这里。
- 💬 Mes (L58): 这里的史莱姆忠于斯普拉蒂娜。它们守护着她和她巨大的宝藏。
- 💬 Mes (L59): 它们中有三个很特别，真正忠诚的。它们有她房间的钥匙，但它们不会轻易交给你。你得去赢得！
- 💬 Mes (L60): 但别担心。如果你带足够的金子，也许斯普拉蒂娜会让你通过……
- 💬 Mes (L64): 很好，拿这些金子去献给斯普拉蒂娜。她会判断你的价值。
- 💭 Chat (L69): 亵渎！你竟敢拒绝斯普拉蒂娜的供品？！
- 🔹 Choice (L19): Sure, I'll bring her the offering.
- 🔹 Choice (L21): What should I do?
- 🔹 Choice (L23): This is nonsense, I’m leaving.
- 🔹 Choice (L25): Who is Splatyna?
- 🔹 Choice (L27): What is this place?
- 🔹 Choice (L39): Nothing! Everything’s fine...
- 🔹 Choice (L40): She’s... gone.

#### Chest

*来源: `splatyna/Chest.gd`*

- 💭 Chat (L19): 这个宝箱似乎被封印了。
- 💭 Chat (L23): 周围仍有黑暗存在。
- 💭 Chat (L35): 这个宝箱是空的。

#### Locks

*来源: `splatyna/Locks.gd`*

- 💬 Mes (L19): 你注意到三把不同的钥匙锁，你想怎么做？
- 💭 Chat (L23): 你需要三把不同的钥匙来解锁这个通道。
- 💭 Chat (L35): 你需要三把不同的钥匙来解锁这个通道。
- 💭 Chat (L38): 三把锁挡住了这个通道。
- 🔹 Choice (L20): Open them
- 🔹 Choice (L21): 离开

#### Clue1

*来源: `tonori/desertpit/Clue1.gd`*

- 💬 Mes (L13): 线索 1
- 💬 Mes (L25): 你在线索1下面找到一把钥匙。

#### Clue2

*来源: `tonori/desertpit/Clue2.gd`*

- 💬 Mes (L13): 线索 2
- 💬 Mes (L25): 你在线索2下面找到一把钥匙。

#### Clue3

*来源: `tonori/desertpit/Clue3.gd`*

- 💬 Mes (L13): 线索 3
- 💬 Mes (L25): 你在线索3下面找到一把钥匙。

#### Clue4

*来源: `tonori/desertpit/Clue4.gd`*

- 💬 Mes (L13): 线索 4
- 💬 Mes (L25): 你在线索4下面找到一把钥匙。

#### Clue5

*来源: `tonori/desertpit/Clue5.gd`*

- 💬 Mes (L13): 线索 5
- 💬 Mes (L25): 你在线索5下面找到一把钥匙。

#### Mauro

*来源: `tonori/desertpit/Mauro.gd`*

- 💬 Mes (L17): 我的水全没了！我不知道现在怎么回图利姆沙。
- 💬 Mes (L18): 几天前我决定直面沙漠，探索这片区域。
- 💬 Mes (L19): 这里的洞穴有一个地下池塘，水非常清新纯净。
- 💬 Mes (L20): 人们过去常来这里取水，但几年前这里开始到处都是蛇。
- 💬 Mes (L21): 我以为我能击退那些蛇，但没想到有这么多！
- 💬 Mes (L22): 我想靠近池塘，但蛇不断攻击我，我把收集的水全洒了。
- 💬 Mes (L23): 你愿意试试吗？我太累了。我至少需要一满壶水才能回到城里。
- 💬 Mes (L24): 这些不是毒蛇，但它们很有攻击性，而且太多了……
- 💬 Mes (L36): 池塘在洞穴深处。找到它的最佳方式就是迷路，你最终会看到的。
- 💬 Mes (L37): 如果你找到了就帮我把壶装满，带回来。
- 💬 Mes (L38): 但小心，如果被那些蛇咬了，你会损失一些水！
- 💬 Mes (L43): 你成功了！还带着水！
- 💬 Mes (L44): 来，拿这个。这是我仅剩的了。
- 💬 Mes (L50): 我现在要回图利姆沙，这片区域太危险了。
- 💬 Mes (L51): 我们会继续依靠城内更安全的水井。我想这水现在属于蛇了。
- 💬 Mes (L54): 谢谢你帮我打水。也许现在我可以放松一点，回程不用那么赶了。
- 💬 Mes (L55): 这些在沙漠里的日子是我最快乐的，但我想现在是时候回到文明世界了。
- 💬 Mes (L58): 我理解。改变主意就回来。
- 💬 Mes (L63): 给你壶。水源在洞穴深处。
- 💬 Mes (L64): 给你一个重要提示：要找到它，我记得我先往北走再往东……
- 💬 Mes (L65): 然后往南？还是我先往东走得更远再往北……
- 💬 Mes (L66): 说实话，我只记得蛇和它们锋利的咬伤。幸好这种没有毒。
- 💬 Mes (L67): 外面祝你好运，我还是建议你别一个人去。
- 💬 Mes (L68): 如果你能找到别人帮忙，我相信击退蛇并收集更多水会更容易。
- 🔹 Choice (L25): I can give it a try.
- 🔹 Choice (L26): Not today.

#### Pachua

*来源: `tonori/desertpit/Pachua.gd`*

- 💬 Mes (L9): 世代以来，我的部落一直在用不同材料制作特殊服装。
- 💬 Mes (L10): 也许你给我带来合适的材料，我可以为你做点什么。
- 💬 Mes (L17): 我所有工具不在这里，让我想想……牛仔皮裤腿你有兴趣吗？
- 💬 Mes (L18): 为此我需要一条普通牛仔短裤和大约10张蛇皮，因为操作时它们容易撕裂。
- 💬 Mes (L19): 成交？
- 💬 Mes (L29): 天哪，看来你没有牛仔短裤。
- 💬 Mes (L32): 天哪，看来你没有足够的蛇皮。
- 💬 Mes (L38): 给你！
- 💬 Mes (L39): 随时再来。
- 💬 Mes (L42): 我们住在这西边的沙漠悬崖。路不对外人开放。
- 💬 Mes (L43): 我出来交易。大多数陌生人也只能走到这里。
- 💬 Mes (L46): 王国很久以前就撤出了这片土地。说不值得守。
- 💬 Mes (L47): 我的族人在他们之前就到这里了，今天我们仍在这里。需要的时候我们和图利姆沙交易，但除此之外我们更愿意保持距离。
- 💬 Mes (L50): 好吧，但你在别的地方绝对拿不到更好的价钱！
- 💭 Chat (L53): 准备好了再来。
- 🔹 Choice (L11): What can you make?
- 🔹 Choice (L12): Where is your tribe?
- 🔹 Choice (L13): What happened to this land?
- 🔹 Choice (L14): Maybe later.
- 🔹 Choice (L20): Yes, that's fine.
- 🔹 Choice (L21): On second thought, maybe later.

#### ThiefsChest

*来源: `tonori/desertpit/ThiefsChest.gd`*

- 💭 Chat (L38): 你的背包太满了，无法从这个宝箱中拿取物品。
- 💭 Chat (L41): 这个宝箱是空的。
- 💭 Chat (L44): 这个宝箱被锁住了。你需要一把钥匙。

#### WaterPond

*来源: `tonori/desertpit/WaterPond.gd`*

- 💬 Mes (L13): 你把壶浸入池塘。静止几秒钟等它装满……

#### WaterPondFilthy

*来源: `tonori/desertpit/WaterPondFilthy.gd`*

- 💬 Mes (L6): 这水太脏了！你不觉得这是毛罗想要说的那个池塘。

#### Hommer

*来源: `tonori/manayir/Hommer.gd`*

- 💬 Mes (L7): 欢迎来到玛纳伊尔。
- 💬 Mes (L18): 我们是卡韦。一个德鲁伊教团，聚集在一起研究、教授和保护我们世界中魔力的秩序。
- 💬 Mes (L19): 很久以前我们遍布整个地区，各自照料附近生长的汉特斯。那个时代已经过去了。
- 💬 Mes (L20): 我们也在图利姆沙待了很长时间。试图阻止魔力变成政治问题。那个时代也过去了。
- 💬 Mes (L21): 现在我们保存我们所知的知识，继续研究。
- 💬 Mes (L25): 汉特，也叫魔力树，是我们世界中魔力的支柱。
- 💬 Mes (L26): 它们通过魔力的流动连接所有生物。每一个都维持周围土地的活力，防止魔力衰变成卡厄。
- 💬 Mes (L27): 当人们开始恐惧魔力，它们就被砍伐。传播很快。一些宗教和政治给了它教义义和目标，不久之后就没有了。
- 💬 Mes (L28): 我们是它们的守护者。我们失败了。
- 💬 Mes (L29): 我们存在的意义就是保存关于它们的知识。我们研究它们留下的魔力流动和痕迹。我们等待。
- 💬 Mes (L33): 你带着什么东西。
- 💬 Mes (L34): 一颗卡诺，一颗魔力种子。
- 💬 Mes (L35): 我只是在书上读到过这些，我们都是。我从没想过我们会再看到一棵。
- 💬 Mes (L39): 好，她知道这是正确的选择。
- 💬 Mes (L40): 但在那之前，我应该告诉你一件事。
- 💬 Mes (L44): 路到这里就结束了，暂时。
- 💬 Mes (L45): 建造这个世界的人仍在为下一步努力。你已经追上他们了。
- 💬 Mes (L46): 祝贺你走到这一步，感谢你与我们一起踏上这段旅程！
- 💬 Mes (L47): 想待多久就待多久，请一定来拜访我们，在Discord上打个招呼。
- 💬 Mes (L48): 在那之前，玛纳伊尔和托诺里地区供你继续探索。
- 🔹 Choice (L12): I was sent by Nina of Tulimshar.
- 🔹 Choice (L13): What is this place?
- 🔹 Choice (L14): What are the Hantu?
- 🔹 Choice (L15): 再见。
- 🔹 Choice (L36): Nina said you could be trusted.
- 🔹 Choice (L41): What is it?
- 🔹 Choice (L49): I'll stay a while.

#### Chest

*来源: `tonori/sandstorm/Chest.gd`*

- 💭 Chat (L35): 你的背包太满了，无法从这个宝箱中拿取物品。
- 💭 Chat (L38): 这个宝箱是空的。
- 💭 Chat (L41): 这个宝箱被锁住了。你需要一把钥匙。

#### Dausen

*来源: `tonori/sandstorm/Dausen.gd`*

- 💬 Mes (L5): 你是埃基努派下去的人？
- 💬 Mes (L6): 我不骗你，他可能觉得你回不来了……
- 💬 Mes (L8): 往东南方向走。矿井入口在一口老井旁边。
- 💬 Mes (L9): 如果你发现自己走到了山谷外面，你就知道方向错了。
- 💬 Mes (L10): 找守卫内森，他应该驻守在矿井入口。
- 💬 Mes (L12): 祝你好运。

#### DeepLevelEntrance

*来源: `tonori/sandstorm/DeepLevelEntrance.gd`*

- 💬 Mes (L18): 我没有感觉到第一个人经过这里时的那种黑暗存在。请上前。
- 💬 Mes (L19): 数千年来第一次，我需要别人的帮助。
- 📖 Narrate (L15): 你注意到洞壁上有一条裂缝。微弱的蓝光似乎从另一边透出来。
- 📖 Narrate (L16): 看起来你可以挤过去。
- 📖 Narrate (L17): 一个平静的声音在你脑海中响起。

#### Elias

*来源: `tonori/sandstorm/Elias.gd`*

- 💬 Mes (L5): 如果你沿着那边的西边通道走，你会离开图利姆沙山谷，进入祖尼台地。
- 💬 Mes (L6): 你知道你要去哪吗？
- 💬 Mes (L12): 如果你沿着这条路走，你会离沙暴矿场越来越远。
- 💬 Mes (L14): 你应该往西走，朝山谷南边走。
- 💬 Mes (L18): 你可以沿着那条路往北走到城墙。
- 💬 Mes (L20): 你不会错过的，从那里几乎可以看到城墙！
- 🔹 Choice (L7): I am looking for the Sandstorm Mines.
- 🔹 Choice (L8): I am looking for Tulimshar.

#### ManaTree

*来源: `tonori/sandstorm/ManaTree.gd`*

- 💬 Mes (L13): 往前走。我会保护你的安全。
- 💬 Mes (L16): 谢谢。
- 💬 Mes (L17): 我本可以保护自己，就像保护你一样，但他会离开，就像他刚才做的那样。
- 💬 Mes (L18): 我的位置已经暴露了，我需要一个盟友来对抗这个新威胁。
- 💬 Mes (L22): 是的。我是一棵乌鲁，也叫魔力树。
- 💬 Mes (L23): 我的种族是埃米尔世界上的第一批生命，直到今天，我以为我会是最后一个。
- 💬 Mes (L24): 你刚才打的那个人是冲我的卡诺来的。你们叫它魔力种子。
- 💬 Mes (L25): 这是一种极其罕见的东西，因为它可以长成一棵新的乌鲁。在我们历史中的这个时刻尤其珍贵，因为我是仅存的最后一棵乌鲁。
- 💬 Mes (L35): 虽然我也想看看，但我不需要阳光也能活。
- 💬 Mes (L36): 我的根深入大地，共享赋予我们所有人力量的生命之源：魔力。
- 💬 Mes (L37): 只要我能与它的流动相连，我就活着。
- 💬 Mes (L38): 很多年前，这个世界到处都是像我一样的乌鲁。我们的根相互连接，魔力比现在更强大地流动。
- 💬 Mes (L42): 魔力可能表现出意想不到的方式。
- 💬 Mes (L43): 当它在一段时间内扰乱了人们的生活时，一种认为它有害的信念发展了起来。
- 💬 Mes (L44): 这迅速传播到许多文化，砍伐乌鲁以限制魔力流动成为惯例。
- 💬 Mes (L48): 当乌鲁受到威胁时，一群打算保护我们的德鲁伊在这个洞穴里种下了一颗卡诺。
- 💬 Mes (L49): 然后，他们在附近某个地方创立了玛纳伊尔教团，研究和利用靠近我根部的魔力流动。
- 💬 Mes (L50): 我出生在这里，对世界隐藏。随着时间推移，玛纳伊尔让他们对我存在的认知逐渐消失，甚至对自己人也一样，这样我就能继续隐藏。
- 💬 Mes (L54): 我要你永远不要向任何人提起我的存在。
- 💬 Mes (L55): 拿走卡诺，魔力种子。
- 💬 Mes (L56): 图利姆沙的德鲁伊妮娜是我知道你可以信任的人。她非常善于感知魔力的流动，我已经了解了她。
- 💬 Mes (L57): 你应该告诉在附近检查矿场的守卫你发现了巨大的危险，进入下层不安全。
- 💬 Mes (L58): 我别无选择，只能相信你会做出正确的决定。
- 💬 Mes (L59): 无论你做什么，都不会比你在这里对付的那个人做的更糟。
- 💬 Mes (L64): 小心。你今天在这里发现的东西已经被隐藏了数千年。
- 💬 Mes (L65): 命运使然，你现在是一位强大魔力源的守护者。
- 💬 Mes (L70): 他曾经是玛纳伊尔的一员。现在他的意图不同了。
- 💬 Mes (L71): 你应该和他教团的人谈谈。他们会告诉你他是谁。
- 💭 Chat (L75): 记住你带了什么。
- 🔹 Choice (L19): Are you... A tree?
- 🔹 Choice (L29): How do you survive down here?
- 🔹 Choice (L30): What should I do now?
- 🔹 Choice (L31): How did you end up in this cave?
- 🔹 Choice (L32): What happened since that time?
- 🔹 Choice (L60): I will go to Nina then.
- 🔹 Choice (L61): What do you know about that man?
- 🔹 Choice (L72): I will go to Nina then.

#### Naem

*来源: `tonori/sandstorm/Naem.gd`*

- 💬 Mes (L7): 我爷爷曾经在这个矿场工作。我小时候他会跟我讲矿场的事。
- 💬 Mes (L8): 它往下很深。我想知道他们想让我们走多远；我想知道下面现在还有没有活着的东西。
- 💬 Mes (L9): 你觉得怎么样？
- 💬 Mes (L17): 能不走远就不走远。来，我有个多余的头盔。
- 💬 Mes (L20): 图利姆沙有一个工匠用棕榈纤维和仙人掌提取的树脂做这些。很结实。最酷了。
- 💬 Mes (L24): 算了。祝你好运！
- 💭 Chat (L27): 祝你好运！
- 🔹 Choice (L10): I don't know, but I'll find out.
- 🔹 Choice (L21): The what?

#### Nathan

*来源: `tonori/sandstorm/Nathan.gd`*

- 💬 Mes (L9): 嗨！我是守卫内森。
- 💬 Mes (L10): 我平时守图利姆沙港口，但今天我有幸在外面被烤成酥脆。
- 💬 Mes (L11): 抱歉。我的错。
- 💬 Mes (L12): 太阳快把我逼疯了。
- 💬 Mes (L15): 哦是你！抱歉，这么多沙子我没认出你。
- 💬 Mes (L26): 确实如此！我明白为什么这地方被遗弃了。
- 💬 Mes (L27): 风暴让我牙里都是沙子。我只能想象曾经在这里工作的矿工每天走这里是什么感受。
- 💬 Mes (L28): 现在里面那些人得应付这个了。
- 💬 Mes (L29): 我只希望能回去守港口。如果他们永久把我派驻这里，我可能就直接去喂哥布林了。
- 💬 Mes (L34): 真的？你不像矿工。你的镐呢？
- 💬 Mes (L39): 啊，我明白了。
- 💬 Mes (L40): 嗯，抱歉让你久等了。一个人站在这里很无聊。
- 💬 Mes (L41): 来吧，进去。他们肯定在等你。
- 💬 Mes (L52): 嗯，既然你问了……我可以用一些额外的水！
- 💬 Mes (L53): 这里有一口井，但已经干涸很久了。
- 💬 Mes (L54): 我跟你直说吧。这甚至不是用来喝的。我身上带着呢。
- 💬 Mes (L55): 我只想把它泼在脸上，让自己感觉没那么被热烤。
- 💬 Mes (L56): 那是个愚蠢的请求吗？
- 💬 Mes (L62): 哦……我理解。没问题。当我没说。
- 💬 Mes (L63): 我猜这就是他们把我派到这里的原因……我一定很烦人。
- 💬 Mes (L67): 我就知道你会理解。补水很重要！我等你，朋友。
- 💬 Mes (L69): 倒不是我能去哪，就算我想……
- 💬 Mes (L76): 你找到一瓶水了吗？
- 💬 Mes (L86): 谢谢你，朋友！终于能松口气了！
- 💬 Mes (L96): 这正是我需要的。你无法想象。
- 💬 Mes (L97): 现在我只需要有人给我再拿一个来……
- 🔹 Choice (L13): Is this the entrance to the Sandstorm Mines?
- 🔹 Choice (L16): About that bottle of water.
- 🔹 Choice (L19): Nevermind, I thought you were a sandman.
- 🔹 Choice (L31): I'm actually supposed to join them.
- 🔹 Choice (L36): I'm not here to mine, just scout the place.
- 🔹 Choice (L43): Thanks Nathan. Have a nice day!
- 🔹 Choice (L45): Is there anything I can do to make your job here easier?
- 🔹 Choice (L58): Not at all. I understand what it's like to overheat in this land.
- 🔹 Choice (L59): Yes, actually. I have more important things to do.
- 🔹 Choice (L72): I have one with me.
- 🔹 Choice (L73): I'll try to find you that!
- 🔹 Choice (L78): I have it.
- 🔹 Choice (L79): Not yet, I'll keep looking.

#### Nickos

*来源: `tonori/sandstorm/Nickos.gd`*

- 💬 Mes (L9): 救命。
- 💬 Mes (L18): 你是谁？求你了，我需要你的帮助！
- 💬 Mes (L22): 哦。我以为他会让我等死。
- 💬 Mes (L23): 他派我一个人下来的时候，似乎真的不在乎我会出什么事。
- 💬 Mes (L27): 我被……谁攻击了？
- 💬 Mes (L28): 我没想到会在下面遇到人，但有一个穿着深色长袍的奇怪男人。
- 💬 Mes (L29): 他似乎在检查洞穴的墙壁。我试着跟他说话，但他二话不说就攻击我。
- 💬 Mes (L34): 我会的。我需要喘口气。
- 💬 Mes (L35): 你身上有什么能帮上忙的东西吗？
- 💬 Mes (L61): 谢谢！我感觉好多了。我在这里休息一会儿，然后离开这个地方。
- 💬 Mes (L62): 谢谢你救了我！我还担心那个人会回来结果我。
- 💬 Mes (L66): 没关系。我在这里休息一会儿，然后就能离开这个地方了。
- 💬 Mes (L67): 我很高兴你找到我。我还担心那个人会回来结果我。
- 💬 Mes (L71): 他沿着这条走廊走开了。我最后一次听到他的声音是在这处岩架下面。
- 💬 Mes (L72): 听起来他像在水里行走。然后我听到石头碎裂的声音。
- 💬 Mes (L73): 我下一次听到声音时，原来是你朝我走来。
- 💬 Mes (L77): 听起来他像在水里行走。然后我听到石头碎裂的声音。
- 💬 Mes (L78): 这处岩架下面某处的墙上一定有裂缝。他就是从那里消失的。
- 💬 Mes (L81): 我很高兴你找到我。我还担心那个人会回来结果我。
- 🔹 Choice (L19): I was sent by Watchman Ryan to find you.
- 🔹 Choice (L24): What happened to you?
- 🔹 Choice (L30): Did you see where he went?
- 🔹 Choice (L31): Are you okay?
- 🔹 Choice (L39): Yes. Try this. [x1 Cactus Potion]
- 🔹 Choice (L41): Yes. Try this. [x1 Pitaya]
- 🔹 Choice (L42): No, I'm sorry. I have nothing.
- 🔹 Choice (L63): Did you see where he went?
- 🔹 Choice (L68): Did you see where he went?

#### Ryan

*来源: `tonori/sandstorm/Ryan.gd`*

- 💬 Mes (L20): 你是谁？
- 💬 Mes (L24): 他派你来？我告诉那个沙子脑子的白痴派新兵来。
- 💬 Mes (L25): 不是随便哪个沙漠流浪者。
- 💬 Mes (L26): 你大概一看到蝎子就会跑，对吧？
- 💬 Mes (L33): 好……随便吧。别说了。
- 💬 Mes (L34): 你的声音很烦人。我最不需要的就是听它在这个洞穴里回荡。
- 💬 Mes (L35): 你也看到了，我已经有一些可怜的家伙在这里了，他们拼命地挖土，看看这鬼地方还有什么值钱的金属。
- 💬 Mes (L39): 我在做了！
- 💬 Mes (L40): 我派了这些笨蛋中的一个去侦察前方，但他好像迷路了。
- 💬 Mes (L41): 他叫尼科斯。他已经有一阵子没回来了。
- 💬 Mes (L42): 去找他。
- 💬 Mes (L43): 如果他还活着就把他带回来。如果他死了，你大概也会死，然后我们就知道下面有危险了。
- 💬 Mes (L44): 听起来怎么样？
- 💬 Mes (L49): 好。反正我也没打算给你选择。
- 💬 Mes (L50): 走！别等我吻别。
- 💬 Mes (L54): 他叫尼科斯。他已经有一阵子没回来了。
- 💬 Mes (L55): 去找他。
- 💬 Mes (L56): 如果他还活着就把他带回来。如果他死了，你大概也会死，然后我们就知道下面有危险了。
- 💬 Mes (L59): 你还活着。
- 💬 Mes (L60): 我还以为你也迷路了。下面发生什么了？
- 💬 Mes (L67): 被淹了？
- 💬 Mes (L68): 我想知道水是从哪来的。
- 💬 Mes (L69): 就在这外面有一口井，也许附近有地下水。
- 💬 Mes (L70): 我们不该乱闯被水淹的洞穴。隧道可能坍塌，把我们全困在里面。我会回去报告矿场已经没救了。
- 💬 Mes (L74): 喷火？我从没见过。
- 💬 Mes (L75): 我在下面看到过能吞下一个人的大蝎子。我想它们喷火也不是不可能。
- 💬 Mes (L76): 也许我们应该避开这个地方。反正我们似乎也找不到什么有用的东西。
- 💬 Mes (L80): 哥……哥布林？你……你确定吗？
- 💬 Mes (L81): 我……我不害怕！如果你在怀疑的话。我不怕哥布林！
- 💬 Mes (L82): 你知道吗？我们应该离开这个地方。对，我们走吧。
- 💬 Mes (L83): 下面太危险了。为了寻找被遗忘的矿石冒这个险不值得。
- 💬 Mes (L87): 一棵树。一棵魔法树。一棵会说话的魔法树。
- 💬 Mes (L88): 接下来呢，你要告诉我蝎子会喷火？哥布林给你唱歌了？
- 💬 Mes (L89): 你疯了。下面发生的事让你变傻了！
- 💬 Mes (L90): 也许这是一个闹鬼的洞穴。我开始觉得我们根本不该下去。
- 💬 Mes (L94): 说到这个。尼科斯呢？你找到他了吗？
- 💬 Mes (L99): 不奇怪，我看到过他被一只朝他爬来的蛆虫吓到。
- 💬 Mes (L100): 他最好别在这里浪费我的时间，快点回来，不然我可能就丢下他。
- 💬 Mes (L106): ……
- 💬 Mes (L107): 好。
- 💬 Mes (L108): 他知道风险。干这行的每个人都知道。
- 💬 Mes (L109): ……
- 💬 Mes (L110): 我们不会去收他的尸体。别提这个。
- 💬 Mes (L111): 如果我再失去任何人，我就再也不能带人出城墙了。
- 💬 Mes (L112): 他的报酬是他自己该领的。就当是给你的报告酬劳。
- 💬 Mes (L113): 至少现在我确定我们不会下去了。我派下去的第一个人死了。不是好兆头。
- 🔹 Choice (L21): I was sent by Watchman Ekinu to help with scouting these mines.
- 🔹 Choice (L28): Actually, I have walked all the way here and fought off a few scorpions already.
- 🔹 Choice (L30): I didn't come all this way to leave that quickly.
- 🔹 Choice (L36): What am I supposed to do then?
- 🔹 Choice (L45): Awful, but I'll do it.
- 🔹 Choice (L46): Okay... why did I agree to this...
- 🔹 Choice (L61): The lower level is completely flooded. There is no chance of getting through.
- 🔹 Choice (L62): There are fire-breathing scorpions! Hundreds! They might get angry if we disturb them.
- 🔹 Choice (L63): There are goblin nests everywhere! We could lose a lot of people clearing them out.
- 🔹 Choice (L64): There was a magical tree. It talked to me!
- 🔹 Choice (L95): He's alive. Shaken up, but resting.
- 🔹 Choice (L96): I found his remains.

#### Sawyl

*来源: `tonori/sandstorm/Sawyl.gd`*

- 💬 Mes (L5): 如果你沿着这条路往东走，你会离开图利姆沙山谷，朝玛纳伊尔海岸方向前进。
- 💬 Mes (L6): 你迷路了吗？
- 💬 Mes (L21): 你会在正南方向找到它们。
- 💬 Mes (L23): 它们已经被遗弃很久了，但我刚才看到一群人往那个方向去了。
- 💬 Mes (L24): 看起来像图利姆沙的人。不知道他们在搞什么……
- 💬 Mes (L28): 图利姆沙山谷是一个低洼的保护区，图利姆沙城在这里躲避开阔的托诺里沙漠。
- 💬 Mes (L29): 城墙把城市封闭在山谷和开阔海洋之间。
- 💬 Mes (L30): 除了吹过山谷的沙尘暴，这片区域相比托诺里更远的地区相对安全，那里各种危险都藏在滚烫的沙子下。
- 💬 Mes (L34): 这是托诺里靠近大海的安静区域。它可能是这片沙漠中气候最温和的地方。
- 💬 Mes (L35): 最显著的特征是从海滩延伸出来的半岛。
- 💬 Mes (L36): 我们的语言中叫它纳瓦。那是玛纳伊尔教团的家。
- 💬 Mes (L42): 他们是一个由智慧法师和魔力专家组成的强大组织。他们存在了很长时间，几乎和我的族人一样久。
- 💬 Mes (L43): 我们的古老传说讲述他们如何在古代统治图利姆沙。现在他们大多独来独往。
- 💬 Mes (L44): 我们偶尔和他们交易，我们一直与他们分享我们对灵体——魔力——的知识和崇敬。
- 🔹 Choice (L11): I am looking for the Sandstorm Mines.
- 🔹 Choice (L13): What can you tell me about this valley?
- 🔹 Choice (L15): What can you tell me about the Manayir Coast?
- 🔹 Choice (L17): Thank you for your time.
- 🔹 Choice (L38): Manyir Order?

#### Sema

*来源: `tonori/sandstorm/Sema.gd`*

- 💬 Mes (L6): 这地方感觉不对劲。
- 💬 Mes (L7): 它有一种诡异的感觉。也许只是因为它被遗弃太久了？
- 💬 Mes (L8): 也许这就是它最初被遗弃的原因……
- 💬 Mes (L10): 我听到更深的地方传来一些声音。我真不想去那里……

#### Xakelbael

*来源: `tonori/sandstorm/Xakelbael.gd`*

- 💬 Mes (L8): 你比我以为的更强。
- 💬 Mes (L9): 我们会再见的……
- 💬 Mes (L20): 我来这里是为了卡诺。没想到会在黑暗中遇到一棵活着的乌鲁。它是怎么活下来的？
- 💬 Mes (L21): 这是最后一个还是还有更多？
- 💬 Mes (L27): 你们图利姆沙人来打扰这里，时机不太好。
- 💬 Mes (L28): 我之前已经对付过你们中的一个了，但你跟得太远了，我不会对你这么宽容了。
- 💬 Mes (L34): 你不该在这里。我也不该在，但这不重要。
- 💬 Mes (L35): 我得到了我想要的。
- 💬 Mes (L36): 我没时间满足你的好奇心，也不能让你离开这里。
- 🔹 Choice (L22): What are you talking about?
- 🔹 Choice (L23): Kano? Uru?
- 🔹 Choice (L24): 你是谁？
- 🔹 Choice (L29): What is this place?
- 🔹 Choice (L30): Can you explain what's happening?
- 🔹 Choice (L31): I don't even know who you are.

#### Chest

*来源: `tonori/tulimshar outskirts/Chest.gd`*

- 💭 Chat (L21): 这个宝箱是空的。

#### Aanshir

*来源: `tonori/tulimshar/Aanshir.gd`*

- 💬 Mes (L5): 被分配到内墙区域是最好的差事了。
- 💬 Mes (L6): 我可以整天待在阴凉处休息，而其他守卫负责把野生动物挡在城墙外。
- 💬 Mes (L7): 我要做的就是防止人们打扰我们女王照料花园。
- 💬 Mes (L19): 那是红女王的城堡。
- 💬 Mes (L20): 如果你有重要的事要说，可以请求觐见女王陛下，但我建议不要打扰她的花园。
- 💬 Mes (L21): 你还可以找到大陆上第二大的图书馆，它的藏书量几乎与玛纳伊尔塔楼相当！
- 💬 Mes (L25): 啊——！
- 💬 Mes (L26): ……抱歉。那个地方让我很不爽。
- 💬 Mes (L27): 很危险。很多蛇，这还是友好的部分。有些土著人住在外面，我不想跟他们打交道，我听说他们吃生鸟，戴着羽毛跳舞。
- 💬 Mes (L28): 如果非去不可，沿着路走。最好还是别去。
- 💬 Mes (L32): 你可以在城墙外找到我们旧的仙人掌田和居住区，完全没有人居住。
- 💬 Mes (L33): 沙暴吞没了一切，迫使我们在山谷尽头筑起路障自守。
- 💬 Mes (L34): 这不太理想，因为那场沙暴一年比一年大。
- 🔹 Choice (L12): What's this large building?
- 🔹 Choice (L13): What's to the west?
- 🔹 Choice (L14): What's to the south?
- 🔹 Choice (L15): I have to go.

#### Andi

*来源: `tonori/tulimshar/Andi.gd`*

- 💬 Mes (L14): 你看到我的粉笔了吗？
- 💬 Mes (L15): 我打赌是守卫拿去自己玩了。
- 💬 Mes (L18): 总有一天，我要去海上探索世界。
- 💬 Mes (L19): 我听说北方有一座城市，有冰冷的白色沙子。
- 💬 Mes (L22): 你会画完美的圆吗？我会！我爸爸教我的。
- 💬 Mes (L23): 薇不会画圆。她试着画时看起来像胖方块。
- 💬 Mes (L24): 它们看起来很好笑，但我不喜欢笑它们，所以她总是想当X。
- 💬 Mes (L25): 我开始叫她\
- 💭 Chat (L9): 哦，我真的很喜欢那块石头！
- 💭 Chat (L10): 小心藏匿的海盗，他们很危险。
- 💭 Chat (L11): 请别擦掉粉笔字！

#### Barrel

*来源: `tonori/tulimshar/Barrel.gd`*

- 💭 Chat (L7): 一个密封的桶。

#### Ben

*来源: `tonori/tulimshar/Ben.gd`*

- 💬 Mes (L21): 你是谁？
- 💬 Mes (L24): 你看起来想说什么。但你手里空空的。
- 💬 Mes (L25): 有东西给我看的时候再来。我有活要干。
- 💬 Mes (L28): 什么。他派你带着这些来？
- 💬 Mes (L30): 我认识这个笔迹。
- 💬 Mes (L31): 这些很旧了。在女王把我们分开驻扎的时候，在我们共用同一堵墙之前。
- 💬 Mes (L32): 弗罗斯特管人。我管石头。就这么分工。
- 💬 Mes (L33): 有效果，持续了一阵子。
- 💬 Mes (L34): 然后女王不断施压。更多要求，总是按她的方式。没有空间去好好干活。
- 💬 Mes (L35): 弗罗斯特开始定规矩。什么事都要走流程。我告诉他，在已有压力之上不需要再加压了。
- 💬 Mes (L36): 他说我把他排斥在外，自己做决定不告诉任何人。
- 💬 Mes (L37): 也许是的，言语不是我的工具。从来不是。
- 💬 Mes (L38): 我只想建造东西，让人们别争论我该怎么建。
- 💬 Mes (L39): 总之。我叫他走。他走了。就这样。
- 💬 Mes (L40): ……
- 💬 Mes (L41): 但显然不是那样。因为我在这里，守着没人想攻破的城墙。
- 💬 Mes (L42): 除了你……但我想是弗罗斯特派你来的，所以不一样。
- 💬 Mes (L49): 把这个信封带回给他。他会知道里面是什么。
- 💬 Mes (L50): 告诉他城墙还在。他会……嗯。就这么告诉他。
- 💬 Mes (L52): 那些信去哪了？有了再来。
- 💬 Mes (L55): 又是你。走廊现在对你开放了，我是认真的。
- 💬 Mes (L56): 我旁边图书馆后面有一条通道。通向城墙外。没多少人知道。
- 🔹 Choice (L22): I was sent by Frost. He gave me these...

#### Ekinu

*来源: `tonori/tulimshar/Ekinu.gd`*

- 💬 Mes (L17): 你已经醒了？
- 💬 Mes (L19): 一定要和埃拉诺谈谈，我们捡到你之后你昏迷的时候是她照顾你的。
- 💬 Mes (L23): 你看到我们要处理的那些蛆虫了吗？
- 💬 Mes (L24): 恶心的生物。据说它们过去比现在小。
- 💬 Mes (L25): 我们的仙人掌种植者受影响最严重。最近蛆虫数量增加对他们的庄稼造成了很大损害。
- 💬 Mes (L27): 守卫凯尔驻守在东边，一个仙人掌农场旁。
- 💬 Mes (L29): 如果你想有所作为，报答你欠我们的恩情，你应该帮凯尔清理一些蛆虫。
- 💬 Mes (L33): 你以为你要去哪？
- 💬 Mes (L38): 嗯……我明白了。
- 💬 Mes (L39): 为了偿还你对城市的亏欠，如果你真的想离开这里，我会让你加入城外的探险队。
- 💬 Mes (L44): 你处理那些蛆虫了吗？
- 💬 Mes (L48): 嗯……他说得对。
- 💬 Mes (L52): 我们确实需要多一个战士，我担心把太多守卫从城里的岗位调走。
- 💬 Mes (L53): 你要往南走，朝沙暴矿场方向。
- 💬 Mes (L54): 它们叫那个名字是有原因的，你知道吗？沙漠那个区域容易刮大风，旋转的沙子让人很难到达。
- 💬 Mes (L55): 但这场风暴不同。异常强烈、炎热、持久。我想知道是不是外面有什么东西在引发它……
- 💬 Mes (L60): 矿场曾经是丰富的铁矿来源，对红女王来说利润丰厚，因为整个区域是她的私人财产。
- 💬 Mes (L61): 它们已经被遗弃几年了，因为沙暴加剧，那个区域变得更难通行，矿工们要么死于沙暴，要么死于在沙漠中游荡的愤怒生物。
- 💬 Mes (L62): 看来我们绝望的王室现在资金不足了。她下令不惜一切代价重新开放矿场。
- 💬 Mes (L63): 像往常一样，她根本没考虑过这对她要派去那里工作的人有什么危险。
- 💬 Mes (L67): 你的任务是侦察该区域，确保旧的洞穴系统没有重大危险。如果一切正常，我们将开始派矿工进来，重启整个作业。
- 💬 Mes (L73): 但走之前，你不能赤脚半瞎地穿越沙漠。
- 💬 Mes (L74): 戴上护目镜和靴子。外面沙子会烧你的脚，还会进眼睛！
- 💬 Mes (L82): 你连武器都没有，对吧？
- 💬 Mes (L83): 来，拿这把剑。总比空手走出去好。
- 💬 Mes (L86): 穿上合适的靴子你也会比赤脚快得多。
- 💬 Mes (L87): 如果你遇到危险，记住逃跑，死人没有尊严可言。
- 💬 Mes (L93): 你现在应该准备好了。
- 💬 Mes (L95): 还有一件事。离开城市之前，跟那边的吉尔伽美什说句话。他在守城门，总是愿意和新兵聊天。
- 💬 Mes (L98): 其他守卫已经先走了。问城门外的守卫道森，他会告诉你去哪。
- 💬 Mes (L114): 先下手为强。
- 💬 Mes (L115): 如果你的对手不是你的对手，赶紧逃命。死人没有尊严。
- 💬 Mes (L125): \
- 💬 Mes (L126): 事实上，她声称自己是托诺里的女王。好像她能控制城墙外的事情似的。
- 💬 Mes (L127): 她声称自己是铂金王朝的后裔，数千年前图利姆沙的统治者。实际上我们都知道她父亲是个仙人掌种植者，她能当女王只是因为他是个非常聪明的人。
- 💬 Mes (L132): 是的。他成功说服了临终的老国王他是他的私生子。
- 💬 Mes (L133): 老人没有其他孩子，喜欢上了他。临终前他立他为继承人。他实际上是个非常聪明的人，也是好领袖，所以宫殿里的人也就接受了。
- 💬 Mes (L134): 现在他也死了，他那个没用的女儿在编造她所谓王室血统的故事。事实是没人喜欢她，她需要一个借口让别人继续让她戴那顶王冠。
- 💬 Mes (L138): 那说得轻了！
- 💬 Mes (L139): 我想看她半埋在沙子里，蝎子戳着她的尸体。
- 💬 Mes (L140): 不过别到处说是我告诉你的。
- 💬 Mes (L141): 她是个冷酷无情、自私自利的人。她把所有时间都花在皇家宫殿里照料她的花园。
- 💬 Mes (L142): 她关心那个愚蠢的花园胜过关心我们任何人的生命。
- 💬 Mes (L155): 准备出发时向城门外的守卫道森报告。
- 💬 Mes (L157): 你很幸运，我们在哥布林之前找到了你，但很高兴看到你好转了。
- 💭 Chat (L148): 保持警惕。
- 💭 Chat (L150): 平安回来。
- 📖 Narrate (L90): 按住奔跑键冲刺，更快地移动或逃离战斗。
- 📖 Narrate (L91): 冲刺消耗耐力，所以需要时再用。
- 📖 Narrate (L119): 按住奔跑键冲刺，更快地移动或逃离战斗。
- 📖 Narrate (L120): 冲刺消耗耐力，所以需要时再用。
- 🔹 Choice (L34): I lost myself.
- 🔹 Choice (L35): I would like to leave this place.
- 🔹 Choice (L45): Yes, Kael said I did well. He thinks I could help with your expedition.
- 🔹 Choice (L56): Why are you heading that way?
- 🔹 Choice (L57): What should I do?
- 🔹 Choice (L64): What should I do?
- 🔹 Choice (L107): What should I do again?
- 🔹 Choice (L108): What can you tell me about the Red Queen?
- 🔹 Choice (L109): What happens if I'm in danger?
- 🔹 Choice (L110): Alright, let's get moving.
- 🔹 Choice (L128): It seems that you really dislike her.
- 🔹 Choice (L129): Her father was a cactus farmer?
- 🔹 Choice (L135): It seems that you really dislike her.
- 🔹 Choice (L143): We should get going before she gets angry then!

#### Elanore

*来源: `tonori/tulimshar/Elanore.gd`*

- 💬 Mes (L16): 你还有别的事吗？
- 💬 Mes (L21): 欢迎回来。我能为你做什么？
- 💬 Mes (L29): 你好，欢迎来到图利姆沙！
- 💬 Mes (L30): 你及时赶到了城门。守卫在地上发现了你，几乎看不见。
- 💬 Mes (L31): 我叫埃拉诺。你感觉怎么样？你需要喝这些水，也请吃这个，会让你感觉好些。
- 💬 Mes (L44): 哦……好吧。来，拿一个。除此之外，你需要花点时间，亲爱的。
- 💬 Mes (L50): 你在沙漠里闲逛什么？最近很危险，你不该一个人出去。
- 💬 Mes (L60): 嗯，有你在真好。城市需要有能力的人帮助维护安全。
- 💬 Mes (L61): 最近沙漠变得更危险了。一群痴迷卡厄的狂热分子藏在东边的山里。他们一直在袭击旅人，有一次在大规模战斗中差点攻破城墙。
- 💬 Mes (L62): 总之，这些是基础衣物，用来换掉我们捡到你时你穿的破布。
- 💬 Mes (L69): 在我让你走之前，让我给你讲几件事，帮你在外面活下来。
- 💬 Mes (L86): 但我不该让你整天待在这里。你是新来的，想找到自己的立足之地。
- 💬 Mes (L131): 是的！我为我们的守卫和这座城市的人们做很多治疗药水。它们是对大多数病症的速效药，真的帮助让人们更安全。
- 💬 Mes (L132): 治疗药水需要知识才能制作，还需要一些关键材料。也许你可以通过帮我收集这些材料来重新振作起来？
- 💬 Mes (L137): 每瓶药水我需要六个蛆虫粘液、一个水和一个火龙果。把它们带给我，我就给你做一瓶仙人掌药水作为回报。
- 💬 Mes (L142): 你什么时候准备好，交易都有效。
- 💬 Mes (L146): 我还是每瓶药水需要六个蛆虫粘液、一个水和一个火龙果。有了再来。
- 💬 Mes (L152): 你带来的够做 %d 瓶药水！我应该做几瓶？
- 💬 Mes (L168): 谢谢你的帮助。我这就开始做新的治疗药水！
- 💬 Mes (L169): 在这里帮助人民比困在塔楼或宫殿里好得多。这座城市更多的领导者应该想想这一点。
- 💬 Mes (L171): 谢谢，给你！
- 💬 Mes (L174): 嗯，你好像没有材料了。
- 💬 Mes (L177): 你的背包满了，先腾点空间。
- 💬 Mes (L186): 魔力是连接我们所有人的生命能量。卡厄是魔力的衰败形式，它腐蚀生命而不是激活生命。还有更多内容，但大概就是这样。
- 💬 Mes (L188): 如果你想了解更多，最好问我的学徒妮娜。她目前负责管理城市的灵魂巨石。你可以从她那里学到更多。
- 💬 Mes (L194): 我叫埃拉诺。我是这座城市和周围土地的卡韦。教团外的人叫我们德鲁伊。我代表考马图亚，一个致力于世界上最古老传统的教团。
- 💬 Mes (L195): 但你现在别操心那个。以后可以再聊。现在我就在这里帮忙配药水，接收城墙外来的伤员。
- 💬 Mes (L214): 守卫凯尔目前负责城墙内的本地巡逻，如果你在找工作应该和他谈谈。
- 💬 Mes (L216): 他在这东北方向，靠近一些棕榈树。
- 💬 Mes (L217): 沿着这堵墙走到拐角，然后走上去。
- 💬 Mes (L221): 守卫埃基努驻守在南边的城门。
- 💬 Mes (L223): 告诉他你为凯尔做了什么，他会知道下一步怎么安排你。
- 💭 Chat (L200): 在外面照顾好自己。
- 💭 Chat (L202): 注意安全。
- 📖 Narrate (L71): 这些是你的重要资源。随时注意它们。
- 📖 Narrate (L73): 你的生命条显示你在死亡前能承受多少伤害。
- 📖 Narrate (L75): 你的法力为技能提供能量。没有它，许多能力无法使用。
- 📖 Narrate (L77): 耐力决定你能在疲劳前冲刺和执行身体动作的时间。
- 📖 Narrate (L79): 菜单让你访问背包、技能、任务、设置等。用它管理你携带和知道的一切。
- 📖 Narrate (L81): 动作条让你放置技能和物品以便快速使用。把最需要的东西拖上去，战斗中每一秒都很重要。
- 📖 Narrate (L83): 在背包窗口可以管理你的物品、装备、药水和找到的收集品。
- 📖 Narrate (L89): 在NPC附近使用交互键与他们交谈。你也可以先使用目标键选择一个。
- 🔹 Choice (L38): I am feeling better now, thank you.
- 🔹 Choice (L39): I am still feeling a bit weak...
- 🔹 Choice (L51): I am a mercenary soldier trying to gain local fame.
- 🔹 Choice (L52): I am looking to start trades with natives of the region.
- 🔹 Choice (L53): I am a scholar seeking knowledge on Mana and magic.
- 🔹 Choice (L54): I am a sailor looking to explore new opportunities on and off land.
- 🔹 Choice (L105): Where should I go now?
- 🔹 Choice (L111): Can I help you with anything?
- 🔹 Choice (L114): I have your ingredients.
- 🔹 Choice (L116): About those ingredients...
- 🔹 Choice (L119): I brought you more ingredients.
- 🔹 Choice (L121): About those ingredients...
- 🔹 Choice (L123): What is Kaore?
- 🔹 Choice (L124): 你是谁？
- 🔹 Choice (L127): Thank you again but I have to leave
- 🔹 Choice (L133): I'll gather them for you.
- 🔹 Choice (L134): Maybe later.
- 🔹 Choice (L153): Just one.
- 🔹 Choice (L154): All %d of them.
- 🔹 Choice (L181): I have something else to ask.
- 🔹 Choice (L182): Goodbye.

#### Eridu

*来源: `tonori/tulimshar/Eridu.gd`*

- 💬 Mes (L54): 嘿！你是新来的吧？我好像没见过你。我叫埃里杜。我是玻璃匠。
- 💬 Mes (L55): 我的家族一直在做玻璃。在这种气候下，人们发现用玻璃携带液体很方便。所以我们主要做瓶子。
- 💬 Mes (L56): 我们用托诺里的沙子做玻璃。我们还使用盐、骨头和硫磺粉。
- 💬 Mes (L57): 到处都是沙子，你也看到了，但其他成分更难弄到。
- 💬 Mes (L75): 你好像带了很多东西。也许扔掉一些再回来。
- 💬 Mes (L81): 盐、骨头、硫磺。你好像什么都有！
- 💬 Mes (L82): 我这就去做一些新瓶子。
- 💬 Mes (L83): 同时，我用尽了最后的资源做了这些。拿着。
- 💬 Mes (L88): %d个瓶子。别打碎了，这可是上好的埃里杜玻璃！
- 💬 Mes (L90): 随时给我带更多材料。每样 %d 份，我就给你吹更多玻璃。
- 💬 Mes (L93): 以我的观察还差。每样材料 %d 份。少了配比就不行。
- 💬 Mes (L94): 慢慢来。炉子不会跑，我也不会。
- 💬 Mes (L97): 又来买玻璃？见到你真好。
- 💬 Mes (L101): 还需要别的吗？
- 💬 Mes (L117): 沙子就是玻璃。我们把它熔化，然后冷却得到的液体。
- 💬 Mes (L118): 我有很多，你也看到了。
- 💬 Mes (L119): 我们用盐降低熔化温度，这样就不会太热而损坏我们的陶罐。
- 💬 Mes (L120): 然后，我们加入钙。就是骨头，磨成粉。这很重要，这样玻璃接触水时不会融化。
- 💬 Mes (L121): 最后，硫磺去除气泡并消除绿色。
- 💬 Mes (L122): 比例不对，玻璃就会脆弱。或者更糟，很丑。
- 💬 Mes (L130): 史莱姆通常有高浓度的盐。有时盐晶体在它们体内形成。不比让海水蒸发容易，但可能更快。
- 💬 Mes (L131): 硫磺更难搞。沙暴那边的火焰哥布林身上有。我觉得是它们的身体产生的。也许用来喷火？
- 💬 Mes (L132): 骨头需要从在某些区域游荡的复活尸体上获取。我听说它们是卡厄制造的。总比挖坟好。
- 💬 Mes (L145): 好。你什么都齐了。
- 💬 Mes (L146): 把它们带到这里来，等它们冷却之前我就把 %d 个瓶子交到你手上。
- 💬 Mes (L149): 随你便。反正炉子一直烧着。
- 💬 Mes (L153): 一个瓶子需要 %d 盐、%d 骨头和 %d 硫磺粉。两倍就能做两瓶。很简单。
- 💬 Mes (L162): 让我称称你带来的东西……
- 💬 Mes (L174): 你的背包快撑破了。你怎么往里面装玻璃？去腾点东西出来！
- 💬 Mes (L182): %d个瓶子。小心别打碎了！
- 💬 Mes (L184): 一瓶。现在别把它打碎了！
- 💭 Chat (L139): 小心碎玻璃。
- 🔹 Choice (L62): I could get those things for you.
- 🔹 Choice (L64): Why those three ingredients?
- 🔹 Choice (L66): Where can I find these things?
- 🔹 Choice (L67): Not right now.
- 🔹 Choice (L106): I brought more material.
- 🔹 Choice (L108): What do you need for another bottle?
- 🔹 Choice (L110): Why those three ingredients?
- 🔹 Choice (L112): Where would I find them?
- 🔹 Choice (L113): Keep well, Eridu.
- 🔹 Choice (L163): Just the one bottle.
- 🔹 Choice (L165): All of it, %d bottles.
- 🔹 Choice (L166): On second thought, I'll hold on to it.

#### FlourBarrel

*来源: `tonori/tulimshar/FlourBarrel.gd`*

- 💬 Mes (L12): 一个深蓝色的火漆印，上面印着阿蒂斯的纹章。就是这个。
- 💬 Mes (L13): 里面是精细研磨的面粉袋，正如里斯金描述的那样。
- 💬 Mes (L16): 但你的背包太满了，装不下它们。

#### Frost

*来源: `tonori/tulimshar/Frost.gd`*

- 💬 Mes (L24): 小心脚下。仙人掌刚长出来。
- 💬 Mes (L25): 抱歉，我不常来客人。自从我在宫殿里当顾问以来就没有了。
- 💬 Mes (L26): 几年前辞了。上面每个人的担子都太重了，你知道吗？
- 💬 Mes (L27): 在这里只有我和泥土。不过耕种对城市有用，所以我不介意。
- 💬 Mes (L28): 你有没有在脑子里反复回放同一个场景，想知道如果做了不同的决定，事情会有多不一样？
- 💬 Mes (L33): 还在走？那些走廊不太友好。
- 💬 Mes (L39): 你找到了。好，好。
- 💬 Mes (L40): 把密封的信件交给本。你刚走我就看见他走进走廊了。
- 💬 Mes (L41): 他直觉很敏锐，或者你专注于不被发现时发出了声音。总之，他会在那里。
- 💬 Mes (L45): 你回来了。他……
- 💬 Mes (L46): 这个信封。很重。
- 💬 Mes (L47): ……
- 💬 Mes (L48): 金子。他在这里放了金子。上面写着我的名字。
- 💬 Mes (L49): 当女王第一次把我们安排在一起时，我们各自留了一份给对方。以防情况恶化到一个人需要离开。
- 💬 Mes (L50): 我以为我走后他会花掉这些。如果他花了，我能理解。
- 💬 Mes (L51): 我不配拥有这个。它应该属于一个比我更好的人。
- 💬 Mes (L52): 来。你拿着。你一个下午为我们做的比我们几年做的还多。
- 💬 Mes (L55): 你找到本了吗？他在西墙走廊。
- 💬 Mes (L63): 等等，信封去哪了？把它带回来给我。
- 💬 Mes (L66): 今年仙人掌长得不错。
- 💬 Mes (L67): 我一直想着我应该走到那边去。到走廊。不是今天，但我决定我会去。
- 💬 Mes (L68): 他保留了那些信。这多少说明点什么，对吧？
- 💬 Mes (L71): 曾经有一个和我一起工作的人。本。聪明人。什么都能修，什么都能建。
- 💬 Mes (L72): 不过小声点。他宁愿把墙拆了重建，也不愿解释为什么要修。
- 💬 Mes (L73): 我，是那个跟人打交道的人。安排班次，解决分歧，维持运转。
- 💬 Mes (L74): 我们多年来共同守护这些城墙。在女王统治下，这不容易。
- 💬 Mes (L79): 女王的事。或者我猜她一直在发生。总是更多要求，总是按她的方式。
- 💬 Mes (L80): 我开始定规矩。什么事都要走流程。我想如果我能把一切都组织好，也许压力就不会压垮我们。
- 💬 Mes (L81): 本看法不同。他说我在我们已经承受的压力上又加了一层。
- 💬 Mes (L82): 他就自己做事。自己做决定。不告诉我，也不告诉任何人。我总是事后才知道。
- 💬 Mes (L83): 我们都在努力维护同样的城墙。但在某个时候，我们不再讨论它，而是开始为此争吵。
- 💬 Mes (L84): 我犯了太多错误，不敢说我对所有事都是对的。但那时候，我们两个都不肯让步。
- 💬 Mes (L85): 他叫我走。我就走了。我对自己处理方式不自豪，但我让开了。
- 💬 Mes (L86): 那是多年前的事了。我还是比应该更多地想起它。
- 💬 Mes (L92): 其实……也许有。
- 💬 Mes (L93): 在我们附近西墙走廊的尽头，过了本的巡逻区，有一个小房间，里面有个书架。
- 💬 Mes (L94): 那个架子上有一个信封。我们互相写的信，在女王把我们分开驻扎的时候。在一切出问题之前。
- 💬 Mes (L95): 我不知道这会不会改变什么。但也许如果他再读一遍，他会记得我们并不总是这样的。
- 💬 Mes (L96): 走廊有巡逻。你应该待在阴影里。如果你走进光亮处，守卫会抓住你。
- 💬 Mes (L101): 西墙走廊的尽头，在我们左边，书架上面。
- 💬 Mes (L102): 还有别待在光亮处。守卫接到严格命令，要把未经许可的市民赶出去。
- 💬 Mes (L106): 谢谢，真的。
- 💭 Chat (L110): 不，我理解。对陌生人要求太多了。
- 💭 Chat (L113): 好。嗯，仙人掌不会自己照顾自己。
- 🔹 Choice (L29): What's on your mind?
- 🔹 Choice (L30): I should get going
- 🔹 Choice (L34): Tell me about Ben again
- 🔹 Choice (L35): Where are the letters?
- 🔹 Choice (L36): I'm on it
- 🔹 Choice (L75): What happened?
- 🔹 Choice (L76): I should get going
- 🔹 Choice (L88): Is there anything I can do?
- 🔹 Choice (L89): I'm sorry to hear that
- 🔹 Choice (L97): I'll do it
- 🔹 Choice (L98): Not now

#### Gilgames

*来源: `tonori/tulimshar/Gilgames.gd`*

- 💬 Mes (L7): 你是埃基努新招的？好。
- 💬 Mes (L8): 我猜你出门前有很多问题，我能帮你什么？
- 💬 Mes (L11): 离城门远点。走开。
- 💬 Mes (L25): 完成任务、战斗或参与活动都能帮助你成长。四处看看，别害怕和周围的人交谈，也别害怕探索周边区域。
- 💬 Mes (L36): 你需要在沙漠中面对的一切，能用的优势都要用上。
- 💬 Mes (L40): 地图是你的朋友，不要犹豫用它来知道该去哪。
- 💬 Mes (L47): 把你喜欢的物品、药水甚至技能放在身边，沙暴时可能有用。
- 💭 Chat (L21): 尽量别把自己弄死了。
- 📖 Narrate (L27): 升级时你获得一些新的属性点，可以花在与你玩法匹配的属性上，并确认你的选择。
- 📖 Narrate (L28): 力量增加你的物理伤害、移动速度和负重。
- 📖 Narrate (L29): 敏捷影响你的攻击速度、攻击距离以及闪避能力。
- 📖 Narrate (L30): 体力提升你的生命值上限、再生和全局防御。
- 📖 Narrate (L31): 耐力让你的耐力值更持久、恢复更快，并让你更快再次攻击。
- 📖 Narrate (L32): 专注为你的法力提供养分，让你能发挥技能的最佳效果。
- 📖 Narrate (L34): 注意你的生命资源，在生命、法力或耐力低的时候冲进荒野可能不是最明智的选择。
- 📖 Narrate (L42): 你可以在地图上定位任何位置并向其移动，以防迷路。
- 📖 Narrate (L49): 你可以将任何可用的物品、技能甚至表情拖放到快捷栏。
- 🔹 Choice (L15): How can I improve myself?
- 🔹 Choice (L16): What happens if I'm lost?
- 🔹 Choice (L17): How can I easily access everything I gathered?
- 🔹 Choice (L18): All good!

#### Hyland

*来源: `tonori/tulimshar/Hyland.gd`*

- 💬 Mes (L58): 欢迎，欢迎！你是来玩牌的吗？
- 💬 Mes (L59): 你知道二十一点吗？我恰好是它的发明者！
- 💬 Mes (L60): 坐下，坐下。让我看看你的本事。牌在等着呢。
- 💬 Mes (L70): 规则很简单。我应该知道，是我定的！
- 💬 Mes (L71): 尽量接近21点但不要超过。
- 💬 Mes (L72): 数字牌按面值，人头牌是10点，A牌……嗯，A牌很灵活。可以是1点或11点。随你喜欢。
- 💬 Mes (L73): 赢我，我就服你。超过21点就爆。这就是规则。
- 💬 Mes (L74): 连守卫都能学会这个，说明它很简单。
- 💬 Mes (L96): 你的手牌：%s
- 💬 Mes (L97): 我出的是：%s。
- 💬 Mes (L105): 你的手牌：%s
- 💬 Mes (L106): 嗬，爆了！贪婪就会这样。我见过一千次了。
- 💬 Mes (L119): 你的手牌：%s
- 💬 Mes (L120): 我的手牌：%s
- 💬 Mes (L122): 哼！老手也有失手的时候。打得好。
- 💬 Mes (L124): 嘿……不错嘛。
- 💬 Mes (L126): 经验又一次获胜。别难过，大多数人都输给我。
- 💬 Mes (L128): %d 平局！有种，我服你。
- 💬 Mes (L136): 好吧，看看还有谁够胆。等一下。
- 💬 Mes (L145): 耐心……好的牌手学会等待。
- 💬 Mes (L158): 哈！比赛开始了！让我看看你们俩的本事。
- 💬 Mes (L162): 你的手牌：%s
- 💬 Mes (L164): 二十一点！
- 💬 Mes (L178): 你的手牌：%s
- 💬 Mes (L179): 爆了！早该听你的直觉。
- 💬 Mes (L183): 你的手牌：%s
- 💬 Mes (L197): 你的部分完成了。现在等你的对手……
- 💬 Mes (L206): 还在想……好玩家会花时间思考！
- 💬 Mes (L216): 你的分数：%d%s
- 💬 Mes (L217): 对手的手牌：%s%s
- 💬 Mes (L220): 平局！势均力敌。在我的桌上可不常见。
- 💬 Mes (L222): 胜利属于你！干得好，牌今天眷顾你。
- 💬 Mes (L224): 运气不好。牌很善变，但总会回来的。
- 💬 Mes (L231): 中途走人？在我那个年代，那得请一轮酒。
- 💭 Chat (L78): 如果你想玩牌，知道去哪找我。
- 💭 Chat (L155): 不玩？真可惜。好吧，也许你会改变主意。
- 🔹 Choice (L64): Challenge Hyland
- 🔹 Choice (L65): Find me a worthy opponent
- 🔹 Choice (L66): Rules
- 🔹 Choice (L67): Not today
- 🔹 Choice (L98): Hit
- 🔹 Choice (L99): Stand
- 🔹 Choice (L107): Play again
- 🔹 Choice (L108): 离开
- 🔹 Choice (L129): Play again
- 🔹 Choice (L130): 离开
- 🔹 Choice (L137): Check
- 🔹 Choice (L138): 取消
- 🔹 Choice (L146): Check
- 🔹 Choice (L147): 取消
- 🔹 Choice (L167): Hit
- 🔹 Choice (L168): Stand
- 🔹 Choice (L198): Check
- 🔹 Choice (L199): Forfeit
- 🔹 Choice (L207): Check
- 🔹 Choice (L208): Forfeit
- 🔹 Choice (L226): Play again
- 🔹 Choice (L227): 离开
- 🔹 Choice (L232): Play again
- 🔹 Choice (L233): 离开

#### Kael

*来源: `tonori/tulimshar/Kael.gd`*

- 💬 Mes (L21): 你已经醒了？
- 💬 Mes (L23): 一定要和埃拉诺谈谈，我们捡到你之后你昏迷的时候是她照顾你的。
- 💬 Mes (L28): 你好，你是他们从沙子里挖出来的那个人吗？很高兴看到你能走动了。我需要你的帮助。
- 💬 Mes (L29): 蛆虫喜欢吃仙人掌。虽然这些蛆虫异常大，但它们的嘴仍然小且灵活，能绕过任何刺吃进去。
- 💬 Mes (L30): 一旦它们突破外部，就会钻进植物从里面吃。很可怕。
- 💬 Mes (L36): 他们说它们并不总是这样的。我太年轻不记得了，现在没人记得了。
- 💬 Mes (L37): 在卡厄时代之前，几百年前，这个世界的一些生物不像今天这么……奇怪。
- 💬 Mes (L38): 卡厄时代改变了它们。这种蛆虫变得又大又多。其他生物被改变得更糟。
- 💬 Mes (L39): 他们说松鼠曾经和人类一样聪明，但随着时间推移它们失去了这一切。
- 💬 Mes (L43): 好了，闲聊到此为止。我要你到田里去清理这些乱七八糟的东西！
- 💬 Mes (L48): 无论如何，它们需要被清除。
- 💬 Mes (L49): 沙漠里发生的一切也影响了城市的蛆虫。
- 💬 Mes (L50): 更糟的是，仙人掌似乎也受到了影响。
- 💬 Mes (L51): 它们中有些跳来跳去，做着鬼脸，还溅到附近的庄稼！你需要帮我处理这个问题。
- 💬 Mes (L56): 卡厄正在影响各种生物。
- 💬 Mes (L57): 就是这种邪恶力量把生物扭曲成……我不知道……更差的自己？大概吧。更具攻击性。
- 💬 Mes (L58): 直到它完全控制它们，然后它们也变成某种亡灵。真是个恶毒的诅咒。
- 💬 Mes (L62): 塔楼那边的法师……玛纳伊尔。他们发布了警告，卡厄目前正在托诺里以危险的模式聚集。
- 💬 Mes (L67): 他们是某个古老的教团，研究魔力，在城西的塔楼里用它做谁知道还有什么的事。
- 💬 Mes (L68): 他们是发布卡厄警告的人。
- 💬 Mes (L69): 玛纳伊尔教团是大约27年前宣布卡厄时代结束的人。
- 💬 Mes (L70): 从那以后，他们一直在向图利姆沙的人们通报能量的流动情况。
- 💬 Mes (L71): 很像天气，不同浓度的魔力和卡厄不断变化，可以影响区域内所有生物。
- 💬 Mes (L76): 我太年轻了不记得，但在几十年前，大家的生活要困难得多。
- 💬 Mes (L77): 卡厄曾经比魔力更充足，在几百年里，人们比今天更艰难地应对变异甚至亡灵生物。
- 💬 Mes (L78): 我们抱怨蛆虫和佩约特，但和我父母经历过的相比，我们过得轻松多了。
- 💬 Mes (L85): 不过你不会徒手清理那片田。
- 💬 Mes (L86): 来，拿这把剑。虽然不怎么样，但对付蛆虫和佩约特够用了。
- 💬 Mes (L89): 先从这北边田里的佩约特开始。
- 💬 Mes (L91): 它们是卡厄浸染的仙人掌，对我们无害，但现在没法耕种了。
- 💬 Mes (L92): 它们只是跳来跳去，溅到附近的庄稼来嘲弄人……
- 💬 Mes (L94): 除掉5株佩约特和5只蛆虫应该就足以保护我们的庄稼一天了。
- 💬 Mes (L112): 还没完。我还需要你清除田里 %d 株佩约特。
- 💬 Mes (L114): 佩约特处理得不错。不过还需要你处理 %d 只蛆虫。
- 💬 Mes (L118): 干得漂亮！
- 💬 Mes (L119): 你处理得非常快，比我预期的好得多。
- 💬 Mes (L120): 你应该向守卫埃基努报告，在遇见埃拉诺的城门附近。告诉他我对你在这里的工作很满意。
- 💬 Mes (L121): 我们很快要去沙漠做一次小型探险，你可能是帮助我们的人选。
- 💬 Mes (L129): 准备好了向守卫埃基努报告。
- 💬 Mes (L130): 他会知道接下来该做什么。
- 💬 Mes (L135): 见到你真好！你也看到了，我还在被这些恶心的白色蛆虫包围着。
- 💬 Mes (L149): 瞄准、射击，然后捂住耳朵，因为击中它们时会发出难听的声音。
- 💭 Chat (L146): 在外面注意背后。
- 📖 Narrate (L96): 进度窗口会分组显示你所有进行中的任务和怪物遭遇。
- 📖 Narrate (L98): 你可以在菜单栏的魔力百科标签中查看你消灭的怪物数量。
- 📖 Narrate (L151): 在敌人附近按交互键自动锁定并攻击。
- 📖 Narrate (L152): 你也可以使用目标动作手动切换目标。
- 📖 Narrate (L153): 每次攻击消耗耐力。耐力耗尽后攻击效果会降低。
- 📖 Narrate (L154): 坐下有助于耐力恢复。
- 🔹 Choice (L31): Why are these maggots so monstrous?
- 🔹 Choice (L32): That sounds terrifying.
- 🔹 Choice (L33): That sounds so cool!
- 🔹 Choice (L44): Yes, I'll get to work!
- 🔹 Choice (L45): What do you mean? What has been happening out in the desert?
- 🔹 Choice (L52): Alright. I'll get to work, then.
- 🔹 Choice (L53): What do you mean? What has been happening out in the desert?
- 🔹 Choice (L63): Alright alright, it's time to kill these things. Let's do it.
- 🔹 Choice (L64): Who are the Manayir?
- 🔹 Choice (L72): Alright alright, it's time to kill these things. Let's do it.
- 🔹 Choice (L73): What's the Age of Kaore?
- 🔹 Choice (L79): Alright alright, it's time to kill these things. Let's do it.
- 🔹 Choice (L140): Why are these maggots so monstrous?
- 🔹 Choice (L141): How can I fight them?
- 🔹 Choice (L142): What has been happening out in the desert?
- 🔹 Choice (L143): I have to leave.

#### LetterStash

*来源: `tonori/tulimshar/LetterStash.gd`*

- 💬 Mes (L20): 你在积灰的书籍间找到两个旧信封。
- 💬 Mes (L21): 一个仔细密封着，另一个异常沉重。
- 💬 Mes (L23): 你发现了信封，但背包太满了拿不了。

#### Maci

*来源: `tonori/tulimshar/Maci.gd`*

- 💬 Mes (L6): 哦，新面孔！欢迎来到图利姆沙。
- 💬 Mes (L7): 我是 %s。我帮人在这里找到立足之地。
- 💬 Mes (L9): 想知道东西在哪？尽管问。
- 💬 Mes (L20): 图利姆沙坐落在我们东边和西边丘陵之间的山谷中。
- 💬 Mes (L21): 你看到周围这堵墙把荒野沙漠挡在外面，保护我们免受南边沙暴的侵袭。
- 💬 Mes (L22): 我们大多待在那个区域，大多数人生活和工作在中央区，分布在从这里到北边港口之间。
- 💬 Mes (L23): 北边是你能找到任何需要的东西的最佳去处，市场、游戏，应有尽有。
- 💬 Mes (L27): 社区非常紧密。
- 💬 Mes (L28): 我们努力互相帮助，有互相关心的人真好。
- 💬 Mes (L29): 警告你：这里不欢迎侮辱，先跟你说清楚。在这座城市里我们互相照应。
- 💭 Chat (L35): 继续好好干 %s。
- 🔹 Choice (L14): Tell me about the city.
- 🔹 Choice (L15): How is it to live here?
- 🔹 Choice (L16): I have to go.

#### Maclat

*来源: `tonori/tulimshar/Maclat.gd`*

- 💬 Mes (L8): 在太阳把你烤熟之前，先给自己弄件轻的衣服！

#### Minguer

*来源: `tonori/tulimshar/Minguer.gd`*

- 💬 Mes (L8): 滚烫的沙子很快就会烧穿赤脚。需要靴子吗？

#### Nard

*来源: `tonori/tulimshar/Nard.gd`*

- 💬 Mes (L5): 欢迎上船，差不多。
- 💬 Mes (L6): 我是拉约哈恩的船长，那艘船就停靠在我身后。
- 💬 Mes (L7): 我们目前正在享受应得的休息，远离寒冷天气，因为过去几周我们在北边的凯泽地区。
- 💬 Mes (L16): 我们很快就要驶向坎多尔，但在此期间我们可以去任何你想去的地方短途旅行。
- 💬 Mes (L17): 我们的船坚固又快速。我们几乎走遍了埃米尔的每一处海岸。有一次甚至到了瑟尔敏！
- 💬 Mes (L18): 当你准备走时，去拉约哈恩的舵轮那里。我们可以在那里讨论你到底想去哪。
- 💬 Mes (L19): 我很久没摆渡人出海了。我们通常运货，不运人。
- 💬 Mes (L24): 坎多尔在北边，玛纳伊尔海岸在这西边，或者一直往东北到阿蒂斯。
- 💬 Mes (L25): 如果要去其他目的地，我们需要准备更长时间。
- 💬 Mes (L26): 我可不想无缘无故付埃斯佩里亚的港口费！赫恩斯卡尔德也一样。
- 💬 Mes (L30): 我从阿蒂斯港口的木匠做起。那时我没什么钱，但我活干得好。
- 💬 Mes (L31): 我曾经还教过一些学生。
- 💬 Mes (L32): 最终我攒够了钱买了自己的船。它几乎是个废船，但我日夜整修，直到它能再次航行。
- 💬 Mes (L33): 我找了一些朋友：加多、朱莉娅和我的老伙计魔法arpan。我们以前都在港口一起工作过。
- 💬 Mes (L34): 我们把船命名为拉约哈恩。以它前任船长的名字。我知道不该给船改名，但我几乎把它重建了。
- 💬 Mes (L35): 现在我们靠往返于海岸之间运送货物为生。价格公道，我是说。
- 💬 Mes (L41): 如果问我，这是世界上最好的城市。氛围、人、酒……
- 💬 Mes (L42): 我确实想它，但我们可以经常回去，虽然不久。很多货物进出阿蒂斯。
- 💬 Mes (L43): 不过我想念我的学生。也许有一天我会回去教书。谁知道呢？
- 💬 Mes (L47): 我认识他们中的一些人很久了。加多曾经是我的大副。朱莉娅是我的厨师。
- 💬 Mes (L48): 问题是，加多喜欢争论，他是个好厨师。朱莉娅喜欢安静……可她厨艺糟糕！所以我把他们调了个位置。
- 💬 Mes (L49): 魔法arpan……我怎么不把他带上船呢？我们做了多年好朋友。他是个有魅力的人。真的，他能用魔法迷住你！
- 💬 Mes (L50): 其他都是我逐渐收留的。都是我喜欢的人。船上每个人跟着我都有原因。
- 💬 Mes (L51): 他们都很有用很重要……也许除了西尔维奥。他只是来这里喝酒和调戏朱莉娅的，那个老酒鬼！
- 💭 Chat (L55): 暂时告别了！想聊天随时回来。我喜欢有好的陪伴。
- 🔹 Choice (L11): Can you take me somewhere by sea?
- 🔹 Choice (L12): Tell me about yourself.
- 🔹 Choice (L13): Safe travels, Captain.
- 🔹 Choice (L20): Where can you take me?
- 🔹 Choice (L21): Let me think about it.
- 🔹 Choice (L36): Do you miss Artis?
- 🔹 Choice (L37): Tell me about your crew.
- 🔹 Choice (L38): Back to it.

#### Neko

*来源: `tonori/tulimshar/Neko.gd`*

- 💬 Mes (L8): 需要刀吗？守卫都来找我买。

#### Nina

*来源: `tonori/tulimshar/Nina.gd`*

- 💬 Mes (L17): 又见面了。
- 💬 Mes (L22): 你好。埃拉诺派你来的？欢迎。
- 💬 Mes (L23): 我们站在图利姆沙古老的灵魂巨石前。我是它的守护者，用它的力量保护这座城市的人民，尽我所能。
- 💬 Mes (L37): 不完全是。卡韦，或德鲁伊，一直都被允许在图利姆沙。因为没有我们，这座城市就无法挺过最严重的干旱和托诺里气候的其他危险。
- 💬 Mes (L38): 然而，王国官方信奉萨维安教义。他们和你周围一半人一样，认为魔力也是像卡厄一样的腐蚀力量，两者都应避免，除非绝对必要。
- 💬 Mes (L39): 我们的巨石仍然矗立，因为它让人们的生活更轻松。即使信仰对立的人也不能否认魔力的好处。他们会在公开场合反对，但当个人需要出现时，他们会找像我这样的卡韦来治愈亲人或带来需要的雨水。
- 💬 Mes (L40): 我只希望更多人公开拥抱古老传统。萨维安信仰给我们世界带来了很多麻烦。现在他们甚至催生出了这些新信徒，瓦鲁尼安人，他们基本上崇拜卡厄。
- 💬 Mes (L41): 这一切都不像埃米尔大部分存在时期与魔力的关系。
- 💬 Mes (L47): 卡韦传承着魔力曾经是我们世界和谐善力量的时代的知识。我们寻求恢复乌鲁被摧毁时失去的平衡。
- 💬 Mes (L53): 乌鲁，也叫魔力树，曾经是世界生命力的核心，充当魔力的天然导体。
- 💬 Mes (L54): 在古代，有很多，每一个都充当稳定力量，防止魔力衰变成卡厄。
- 💬 Mes (L55): 魔力树在很多年前被摧毁，那是一场毁灭性的战争之后，萨维安教义在埃米尔统治精英中扎根。
- 💬 Mes (L56): 他们决定摧毁乌鲁，尝试创造一个没有魔力的世界。
- 💬 Mes (L62): 取决于你问谁。有些人，比如红女王，会说没有魔法的生活更好。
- 💬 Mes (L63): 但她在宫殿里还是用的。她只是觉得平民最好不要有，声称太危险了。
- 💬 Mes (L64): 如果你问我，我认为魔法是我们世界的自然组成部分。它带领我们走到今天，尽管是最危险的自然力量，我们需要它。
- 💬 Mes (L65): 仔细想想，水也是一种危险的自然力量，但没人建议关闭水井。
- 💬 Mes (L70): 她是我们城市和整个托诺里王国的统治者，不过在城墙外她没有多大权力。
- 💬 Mes (L71): 托诺里的祖尼部落从未承认我们的王国，所以她实际上只统治这座城市。
- 💬 Mes (L72): 又一个她不让人民接触魔法的理由。祖尼人仍在修习古老的魔法，这让他们很难对付。
- 💬 Mes (L73): 她不想在图利姆沙也遇到同样的问题。
- 💬 Mes (L78): 他们在托诺里沙漠生活了很长时间。事实上，他们仍在讲述这片土地还是沙漠之前的故事。
- 💬 Mes (L79): 他们一直和图利姆沙交易，对我们大多友好，但前提是我们不把士兵派到城墙外太远。
- 💬 Mes (L80): 过去每次发生这种情况，它们都会击退我们直到我们放弃。我尊重它们这一点，它们有明确的界限，会保卫自己的家园。
- 💬 Mes (L85): 好……
- 💬 Mes (L89): 你可以在这北边的市场找到他们的一些商人。
- 💬 Mes (L90): 我相信他们很乐意聊天，尤其是你买他们东西的话。
- 💬 Mes (L95): 魔力，通常被称为\
- 💬 Mes (L96): 它流经世界的生物量，形成相互连接的能量网。正确引导后，可以利用它产生魔法、治愈大地、维持生命。
- 💬 Mes (L97): 卡厄是魔力与其源头断绝后剩下的东西。
- 💬 Mes (L98): 它不滋养生命，而是溃烂、腐化、衰败。它渗入大地，将生物扭曲成敌对、攻击性的形态。
- 💬 Mes (L99): 卡厄在很多方面是魔力的一种形式，是我们能感知的主要能量。
- 💬 Mes (L100): 魔力流经所有生物，但也注入一种叫齐埃尔石的特定矿物。这块灵魂巨石就是用它做的。
- 💬 Mes (L105): 灵魂巨石是古老的巨石，由巨大的齐埃尔石块雕刻而成，曾经由卡韦（在这里被称为德鲁伊）竖立，用于引导辐射魔力。
- 💬 Mes (L106): 激活时，它们产生保护光环，驱逐卡厄感染的生物并稳定土地。
- 💬 Mes (L107): 在卡韦的帮助下，灵魂巨石可以用来治愈受伤的人，甚至在某些情况下防止死亡，给人们多次生命机会。
- 💬 Mes (L113): 齐埃尔石是一种稀有矿物，与魔力有天然亲和力。它可以吸收、储存和释放魔力能量，对魔法和防护都极其宝贵。
- 💬 Mes (L114): 虽然曾经丰富，但齐埃尔石由于萨维安教义使社会反对使用魔力时发生的清洗而变得稀缺。
- 💬 Mes (L115): 齐埃尔石工艺仍存在于隐藏的护身符、被遗忘的废墟和少数继续矗立作为对抗卡厄堡垒的灵魂巨石中。
- 💬 Mes (L122): 现在不用。我还在盼着那份点心，如果你碰巧找到的话！
- 💬 Mes (L125): 现在不用，但谢谢你的好意。
- 💬 Mes (L128): 现在不用……
- 💬 Mes (L129): 不过我有点饿了。倒不是要你帮我带吃的！我待会儿自己解决。
- 💬 Mes (L134): 你真是太好了。我一般不会开口，但一个羊角面包现在简直太美妙了。
- 💬 Mes (L135): 你通常可以在市场的面包摊找到它们。我本想自己去，但……这块巨石不会自己守护。
- 💬 Mes (L139): 天哪！
- 💬 Mes (L140): 一个羊角面包！真不敢相信。你太好了，我完全没想到！
- 💬 Mes (L148): 请收下这些仙人掌药水和一点金子，弥补你的花费。
- 💬 Mes (L150): 哦……去哪了？算了，也许下次吧。
- 💬 Mes (L152): 我想给你点什么作为回报，但你的背包满了。腾点空间再来找我。
- 💬 Mes (L156): 你回来了！
- 💬 Mes (L157): 探险怎么样？
- 💬 Mes (L161): 我能看看吗？
- 💬 Mes (L162): ……
- 💬 Mes (L163): 你说得对。我不敢相信。这是一颗卡诺。魔力种子！
- 💬 Mes (L164): 我只是在书上读到过这些。我从没想过我这辈子能看到一棵。
- 💬 Mes (L165): 你是怎么找到这个的？
- 💬 Mes (L169): 你说什么？！你刚才说魔力树？活的？在沙暴矿场里？？？
- 💬 Mes (L170): 我……
- 💬 Mes (L171): 我需要一点时间。
- 💬 Mes (L172): 很久以前我在那个方向感觉到过什么。我以为是深层齐埃尔石矿。我完全不知道。
- 💬 Mes (L173): 你能和它交流吗？它告诉你怎么处理那个卡诺了吗？
- 💬 Mes (L177): 它怎么认识我？这一切……很难理解。
- 💬 Mes (L178): 我很高兴它信任我，但我们需要让玛纳伊尔知道这件事。他们是一个古老的德鲁伊教团，曾经是这些树的守护者。
- 💬 Mes (L179): 他们在图利姆沙山谷西边的海滩尽头的一个小半岛上建了一座城堡，在风暴之外。
- 💬 Mes (L180): 往南走，过了城墙再往西。
- 💬 Mes (L181): 找到他们，给他们看卡诺。解释你看到的一切。
- 💬 Mes (L182): 暂时别告诉别人这件事。这是一件非常强大的物品。肯定会引来不该有的注意。
- 💬 Mes (L183): 走之前，带上这个。
- 💬 Mes (L190): 这是一块齐埃尔石护身符。中心的石头是纯齐埃尔石的碎片，这块巨石也是同一种矿物做的。
- 💬 Mes (L191): 佩戴时，它让你能引导魔力，并与灵魂巨石连接。
- 💬 Mes (L194): 你的背包太满了装不下。腾点空间再来找我。
- 💬 Mes (L197): 当你戴着齐埃尔石护身符触摸灵魂巨石时，石头与巨石的齐埃尔石产生共鸣。
- 💬 Mes (L198): 这在你的灵魂和巨石之间建立了一种联系。如果你出了什么事，巨石会充当锚点。
- 💬 Mes (L199): 你的灵魂在这里回归而不是消散。
- 💬 Mes (L200): 这就是我们卡韦一直以来保护我们所照顾之人的方式。我应该早点给你一个。
- 💬 Mes (L207): 好。你现在连接上了。
- 💬 Mes (L208): 如果你找到另一块灵魂巨石，触摸它。你会与最后一块你触摸过的巨石产生共鸣。
- 💬 Mes (L209): 现在，你应该前往玛纳伊尔。我们不能带着卡诺在这里干等。
- 💬 Mes (L213): 你找到玛纳伊尔了吗？
- 💬 Mes (L214): 你应该往南走，然后往西。他们的塔楼在沙暴之外，在一个安静的半岛上。在那附近问问，应该有人能指给你正确的方向。
- 📖 Narrate (L204): 你把手放在灵魂巨石上。齐埃尔石护身符贴着你的皮肤变暖。
- 📖 Narrate (L205): 一种奇怪的感觉传遍全身。像流水，但没有湿润感。石头短暂地闪烁着微弱的蓝光。
- 🔹 Choice (L28): I found a croissant for you!
- 🔹 Choice (L30): Is someone trying to stop you?
- 🔹 Choice (L31): Can you tell me more about Mana and Kaore?
- 🔹 Choice (L32): What is a Soul Menhir?
- 🔹 Choice (L33): See you.
- 🔹 Choice (L42): What is your people's position?
- 🔹 Choice (L43): I have some other topics I wanted to discuss.
- 🔹 Choice (L44): Thank you for your time.
- 🔹 Choice (L48): What are the Uru?
- 🔹 Choice (L49): I have some other topics I wanted to discuss.
- 🔹 Choice (L50): Thank you for your time.
- 🔹 Choice (L57): Where do things stand now?
- 🔹 Choice (L58): I have some other topics I wanted to discuss.
- 🔹 Choice (L59): Thank you for your time.
- 🔹 Choice (L66): Who is the Red Queen?
- 🔹 Choice (L67): Is there anything I can do for you?
- 🔹 Choice (L74): What can you tell me about the Zuni?
- 🔹 Choice (L75): Is there anything I can do for you?
- 🔹 Choice (L81): It would be interesting to meet them!
- 🔹 Choice (L82): Bah! They seem like uncivilised nomads to me.
- 🔹 Choice (L91): Is there anything I can do for you?
- 🔹 Choice (L108): What do you know about Zielite?
- 🔹 Choice (L109): I have some other topics I wanted to discuss.
- 🔹 Choice (L110): Thank you for your time.
- 🔹 Choice (L130): I'll keep an eye out for snacks on the way to the market.
- 🔹 Choice (L131): Ok bye.
- 🔹 Choice (L158): You will not believe what happened. I found this...
- 🔹 Choice (L166): There is a Mana Tree at the bottom of the mines. It gave it to me.
- 🔹 Choice (L174): It said to bring it to you and that you could be trusted.
- 🔹 Choice (L192): What does connecting to a Soul Menhir do?
- 🔹 Choice (L201): [Touch the Soul Menhir]

#### PatrolGuardCaught

*来源: `tonori/tulimshar/PatrolGuardCaught.gd`*

- 💬 Mes (L8): 站住别动。
- 💬 Mes (L9): bones议员的命令。未经许可不得走这些走廊。
- 💬 Mes (L10): 我不管你来干什么。转身。

#### Riskim

*来源: `tonori/tulimshar/Riskim.gd`*

- 💬 Mes (L25): 啊，欢迎！恐怕现在货架是空的。
- 💬 Mes (L26): 图利姆沙没有好农田。你看看周围，我们在沙漠里。这里除了仙人掌什么也不长。
- 💬 Mes (L27): 我的面粉从阿蒂斯运来，以维持我的面包店运营。我的沙暴面包养活这座城市很多年了。
- 💬 Mes (L28): 今早码头到了一批货，但在这种高温下来回走动太累了。
- 💬 Mes (L29): 如果我自己去，今天就累得烤不了东西了。
- 💬 Mes (L30): 阿蒂斯面包店的桶上有蓝色火漆印。你能把里面的面粉袋带给我吗？
- 💬 Mes (L43): 在码头有收获吗？找蓝色封蜡标记的桶。
- 💬 Mes (L44): 我会把炉子烧热准备好。
- 💬 Mes (L47): 即使在这种高温下，一个热面包也很难拒绝。
- 💬 Mes (L64): 我的招牌配方！阿蒂斯的面粉，用仙人掌汁代替普通水，加一撮沙漠盐。
- 💬 Mes (L65): 外皮能撑过沙暴，第二天早上仍然新鲜。这就是名字的由来。
- 💬 Mes (L72): 主要是仙人掌。生命力顽强的植物，我们充分利用它们。做糖、饮料，甚至药材。
- 💬 Mes (L73): 但谷物？托诺里从来不是种谷物的地方。没有商船，这座城市会闹饥荒。
- 💬 Mes (L80): 大洋彼岸的伟大港口城市，在奥罗拉海岸。肥沃的农田，技艺精湛的工匠。
- 💬 Mes (L81): 我姐姐住在那边。她烤的饼干是最好吃的！我用的面粉都是从她的面包店运来的。
- 💬 Mes (L82): 来自阿蒂斯的商船供应图利姆沙市场一半的货。没有它们，我们除了仙人掌几乎没别的可吃。
- 💬 Mes (L86): 贸易。一直是图利姆沙最大的技能。这样的城市怎么在多风的山谷沙漠中繁荣？
- 💬 Mes (L87): 我们位于三大洲的十字路口。一切迟早都要经过图利姆沙的港口。
- 💬 Mes (L88): 它也让我们有饭吃，但对我的生意来说，这意味着总是等下一艘船到来。
- 💬 Mes (L92): 随时来。总有新鲜面包等着你。
- 💬 Mes (L98): 非常感谢，朋友。
- 💬 Mes (L99): 找深蓝色的封蜡。你不会错过的。
- 💬 Mes (L102): 没问题。炉子等着就是了。
- 💬 Mes (L106): 你查过码头了吗？面粉袋应该在蓝色封蜡标记的桶里。
- 💬 Mes (L110): 你找到了！但看起来你的背包太满了，我没法给你任何回报。
- 💬 Mes (L111): 腾出点空间再来。
- 💬 Mes (L117): 你找到了！太好了。
- 💬 Mes (L118): 有了这些面粉，图利姆沙在日落前就能有新鲜的沙暴面包。
- 💬 Mes (L125): 来，拿我最好的。羊角面包和仙人掌糖。一个是家族秘方，另一个是图利姆沙的特产！
- 🔹 Choice (L35): I'll check the docks for you.
- 🔹 Choice (L37): What's Sandstorm Bread?
- 🔹 Choice (L39): Does anything grow out here?
- 🔹 Choice (L40): Maybe later.
- 🔹 Choice (L52): Tell me about Artis.
- 🔹 Choice (L54): How does Tulimshar survive?
- 🔹 Choice (L56): What's Sandstorm Bread?
- 🔹 Choice (L58): Does anything grow out here?
- 🔹 Choice (L60): 保重。

#### Shimris

*来源: `tonori/tulimshar/Shimris.gd`*

- 💬 Mes (L5): 你需要什么帮助？
- 💬 Mes (L16): 往北走。你会到的。
- 💬 Mes (L17): 港口是大多数货物从托诺里以外进来的地方。
- 💬 Mes (L18): 通常很忙，如果我是你，我不会在船员工作的时候打扰他们。
- 💬 Mes (L19): 就我个人而言，我宁愿待在城堡附近也不愿跟那群人打交道。
- 💬 Mes (L23): 东边有很长的海滩。如果你喜欢沙子，你会喜欢的。不过不是每个人都喜欢沙子……
- 💬 Mes (L24): 这东北方向有一座大灯塔。从上面可以俯瞰整个海洋。
- 💬 Mes (L25): 很难错过，有用的地标。
- 💬 Mes (L26): 如果你继续往东走，你会找到玛纳伊尔岛，就在海滩更远处。
- 🔹 Choice (L10): Where is the port?
- 🔹 Choice (L11): What's along the coast?
- 🔹 Choice (L12): I have to go.

#### Shiro

*来源: `tonori/tulimshar/Shiro.gd`*

- 💬 Mes (L8): 没有合适的头部装备就冲进沙暴？你很快就会变成沙魔像的！

#### Wie

*来源: `tonori/tulimshar/Wie.gd`*

- 💬 Mes (L10): 你想玩井字棋吗？棋盘是我自己画的！
- 💬 Mes (L12): 我不会画圆，就画方块代替。但安迪总是取笑我。
- 💬 Mes (L13): 想玩井字棋吗？
- 💬 Mes (L15): 前几天我看到一群海盗在跟一些守卫玩井字棋。
- 💬 Mes (L16): 想玩吗？
- 💬 Mes (L18): 你想玩井字棋吗？
- 💬 Mes (L25): 嘿，已经有人在等了！你应该去和他们玩！
- 💬 Mes (L30): 还没人来……想继续等吗？
- 💬 Mes (L37): 嘿，别作弊！玩你的游戏去！
- 💬 Mes (L39): 嘘，他们在玩！不过你可以看。
- 💬 Mes (L46): 所以你选一个格子放你的标记，然后对方走。
- 💬 Mes (L47): 三个连成一线就赢了！横着、竖着，或者斜着都行。
- 💬 Mes (L48): 如果整个棋盘下满还没人赢，就是平局。不过那样挺无聊的。
- 💬 Mes (L59): 你是X！选一个格子，快！
- 💬 Mes (L61): 等一下，已经有人在玩了！排队等着。
- 💬 Mes (L68): 现在我们等人出现……
- 💬 Mes (L72): 耶，你们都在！玩得开心！
- 💬 Mes (L75): 不行不行，棋盘现在被占了！
- 💭 Chat (L52): 好吧……改变主意就回来！
- 💭 Chat (L79): 我相信很快就会有人来……大概吧！
- 💭 Chat (L85): 啊，这么快？好吧……
- 🔹 Choice (L19): Play against you
- 🔹 Choice (L20): Play against someone else
- 🔹 Choice (L21): How do you play?
- 🔹 Choice (L22): Maybe later
- 🔹 Choice (L26): I'll play them!
- 🔹 Choice (L27): How do you play?
- 🔹 Choice (L28): Maybe later
- 🔹 Choice (L31): I'll wait a bit more
- 🔹 Choice (L32): How do you play?
- 🔹 Choice (L33): I have to go
- 🔹 Choice (L40): Ok ok
- 🔹 Choice (L41): How do you play?
- 🔹 Choice (L43): I have to go

#### Guard

*来源: `tonori/tulimshar/castle/Guard.gd`*

- 💬 Mes (L5): 站住！你不能继续前进。
- 💬 Mes (L6): 前方是红女王的宝座厅。现在转身回去。

#### RedQueen

*来源: `tonori/tulimshar/castle/RedQueen.gd`*

- 💬 Mes (L5): 别烦我。

---

## 8. 附录

### A. 装备槽位 (ActorCommons.Slot)

| 值 | 名称 |
|-----|------|
| -1 | None |
| 0 | CHEST |
| 1 | LEGS |
| 2 | FEET |
| 3 | HANDS |
| 4 | HEAD |
| 5 | NECK |
| 6 | WEAPON |
| 7 | SHIELD |
| 8 | BODY |
| 9 | FACE |
| 10 | HAIR |
| 11 | QUEST |

### B. 技能类别 (SkillCell.Category)

| 值 | 名称 |
|-----|------|
| 0 | SPELL |
| 1 | PHYSICAL |
| 2 | ABILITY |

### C. 技能模式 (Skill.TargetMode)

| 值 | 名称 |
|-----|------|
| 0 | SINGLE |
| 1 | ZONE |
| 2 | SELF |

### D. AI 行为标志 (AICommons.Behaviour)

| 值 | 名称 |
|-----|------|
| 0 | NONE |
| 1 | PACIFIST |
| 2 | NEUTRAL |
| 4 | AGGRESSIVE |
| 8 | IMMOBILE |
| 16 | FOLLOWER |
| 32 | LEADER |
| 64 | SPAWNER |
| 128 | STEAL |
| 256 | FLEE |

### E. 任务进度常量 (ProgressCommons)

```
UnknownProgress = 0
CompletedProgress = 255
```

---

*文档结束*