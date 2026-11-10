extends RefCounted
class_name RaritySystem

# ============================================================
# RaritySystem — 稀有度系统
# 白(普通) / 绿(优秀) / 蓝(稀有) / 紫(史诗) / 橙(传说)
# ============================================================

enum Rarity {
	COMMON = 0,      # 普通（白）
	UNCOMMON = 1,    # 优秀（绿）
	RARE = 2,        # 稀有（蓝）
	EPIC = 3,        # 史诗（紫）
	LEGENDARY = 4,   # 传说（橙）
	COUNT = 5
}

# 稀有度名称（中文）
const RARITY_NAMES : Array[String] = [
	"普通", "优秀", "稀有", "史诗", "传说"
]

# 稀有度名称（英文）
const RARITY_NAMES_EN : Array[String] = [
	"Common", "Uncommon", "Rare", "Epic", "Legendary"
]

# 稀有度颜色
const RARITY_COLORS : Array[Color] = [
	Color(0.85, 0.85, 0.85),  # 白
	Color(0.2, 0.8, 0.2),      # 绿
	Color(0.2, 0.5, 1.0),      # 蓝
	Color(0.7, 0.2, 0.9),      # 紫
	Color(1.0, 0.6, 0.1),      # 橙
]

# 稀有度掉落权重
const RARITY_WEIGHTS : Array[float] = [
	100.0,  # 普通
	50.0,   # 优秀
	20.0,   # 稀有
	5.0,    # 史诗
	1.0,    # 传说
]

# 稀有度属性倍数（词条数量/数值加成）
const RARITY_MULTIPLIERS : Array[float] = [
	1.0,   # 普通
	1.2,   # 优秀
	1.5,   # 稀有
	2.0,   # 史诗
	3.0,   # 传说
]

# 稀有度词条数量范围 [min, max]
const RARITY_AFFIX_COUNTS : Array[Vector2i] = [
	Vector2i(0, 1),  # 普通: 0-1 词条
	Vector2i(1, 2),  # 优秀: 1-2 词条
	Vector2i(2, 3),  # 稀有: 2-3 词条
	Vector2i(3, 4),  # 史诗: 3-4 词条
	Vector2i(4, 6),  # 传说: 4-6 词条
]

# 获取稀有度名称
static func GetRarityName(rarity : Rarity) -> String:
	if rarity >= 0 and rarity < RARITY_NAMES.size():
		return RARITY_NAMES[rarity]
	return "未知"

# 获取稀有度名称（英文）
static func GetRarityNameEn(rarity : Rarity) -> String:
	if rarity >= 0 and rarity < RARITY_NAMES_EN.size():
		return RARITY_NAMES_EN[rarity]
	return "Unknown"

# 获取稀有度颜色
static func GetRarityColor(rarity : Rarity) -> Color:
	if rarity >= 0 and rarity < RARITY_COLORS.size():
		return RARITY_COLORS[rarity]
	return Color.WHITE

# 获取稀有度颜色（BBCode 字符串）
static func GetRarityColorString(rarity : Rarity) -> String:
	var c : Color = GetRarityColor(rarity)
	return "#" + c.to_html(false)

# 获取稀有度倍数
static func GetRarityMultiplier(rarity : Rarity) -> float:
	if rarity >= 0 and rarity < RARITY_MULTIPLIERS.size():
		return RARITY_MULTIPLIERS[rarity]
	return 1.0

# 获取稀有度词条数量范围
static func GetAffixCountRange(rarity : Rarity) -> Vector2i:
	if rarity >= 0 and rarity < RARITY_AFFIX_COUNTS.size():
		return RARITY_AFFIX_COUNTS[rarity]
	return Vector2i(0, 0)

# 随机生成一个稀有度（按权重）
static func RollRarity(luckBonus : float = 0.0) -> Rarity:
	var totalWeight : float = 0.0
	for i in range(RARITY_WEIGHTS.size()):
		var weight : float = RARITY_WEIGHTS[i]
		# 运气加成：高稀有度权重增加
		if i > 0:
			weight *= (1.0 + luckBonus * i)
		totalWeight += weight

	var roll : float = randf() * totalWeight
	var cumulative : float = 0.0
	for i in range(RARITY_WEIGHTS.size()):
		var weight : float = RARITY_WEIGHTS[i]
		if i > 0:
			weight *= (1.0 + luckBonus * i)
		cumulative += weight
		if roll <= cumulative:
			return i as Rarity

	return Rarity.COMMON

# 获取强化等级颜色
static func GetEnhanceColor(level : int) -> Color:
	if level <= 0:
		return Color(0.85, 0.85, 0.85)
	elif level <= 3:
		return Color(0.2, 0.8, 0.2)
	elif level <= 6:
		return Color(0.2, 0.5, 1.0)
	elif level <= 9:
		return Color(0.7, 0.2, 0.9)
	else:
		return Color(1.0, 0.6, 0.1)
