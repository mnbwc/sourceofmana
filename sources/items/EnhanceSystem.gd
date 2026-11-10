extends RefCounted
class_name EnhanceSystem

# ============================================================
# EnhanceSystem — 强化系统
# 用材料强化装备，提升属性
# ============================================================

# 最大强化等级
const MAX_ENHANCE_LEVEL : int = 15

# 强化成功率（按等级）
const SUCCESS_RATES : Array[float] = [
	1.0,    # +0 → +1: 100%
	1.0,    # +1 → +2: 100%
	1.0,    # +2 → +3: 100%
	0.9,    # +3 → +4: 90%
	0.8,    # +4 → +5: 80%
	0.7,    # +5 → +6: 70%
	0.6,    # +6 → +7: 60%
	0.5,    # +7 → +8: 50%
	0.4,    # +8 → +9: 40%
	0.3,    # +9 → +10: 30%
	0.25,   # +10 → +11: 25%
	0.2,    # +11 → +12: 20%
	0.15,   # +12 → +13: 15%
	0.1,    # +13 → +14: 10%
	0.05,   # +14 → +15: 5%
]

# 强化费用（金币，按等级）
const ENHANCE_COSTS : Array[int] = [
	100,    # +0 → +1
	200,    # +1 → +2
	500,    # +2 → +3
	1000,   # +3 → +4
	2000,   # +4 → +5
	5000,   # +5 → +6
	10000,  # +6 → +7
	20000,  # +7 → +8
	50000,  # +8 → +9
	100000, # +9 → +10
	200000, # +10 → +11
	500000, # +11 → +12
	1000000,# +12 → +13
	2000000,# +13 → +14
	5000000,# +14 → +15
]

# 强化属性加成倍数（每级 +10%）
const ENHANCE_MULTIPLIER_PER_LEVEL : float = 0.1

# 强化失败惩罚（按等级）
# -1 = 归零，-2 = 降级，0 = 不变
const FAIL_PENALTIES : Array[int] = [
	0, 0, 0, 0, 0,  # +0 ~ +4: 失败不变
	-1, -1, -1,     # +5 ~ +7: 失败归零
	-2, -2, -2,     # +8 ~ +10: 失败降1级
	-2, -2, -2, -2  # +11 ~ +14: 失败降1级
]

# 获取强化成功率
static func GetSuccessRate(currentLevel : int) -> float:
	if currentLevel < 0 or currentLevel >= SUCCESS_RATES.size():
		return 0.0
	return SUCCESS_RATES[currentLevel]

# 获取强化费用
static func GetEnhanceCost(currentLevel : int) -> int:
	if currentLevel < 0 or currentLevel >= ENHANCE_COSTS.size():
		return 0
	return ENHANCE_COSTS[currentLevel]

# 获取失败惩罚
static func GetFailPenalty(currentLevel : int) -> int:
	if currentLevel < 0 or currentLevel >= FAIL_PENALTIES.size():
		return 0
	return FAIL_PENALTIES[currentLevel]

# 计算强化后的属性倍数
static func GetEnhanceMultiplier(enhanceLevel : int) -> float:
	return 1.0 + enhanceLevel * ENHANCE_MULTIPLIER_PER_LEVEL

# 尝试强化
# 返回: [success, newLevel, message]
static func TryEnhance(currentLevel : int, luckBonus : float = 0.0) -> Array:
	if currentLevel >= MAX_ENHANCE_LEVEL:
		return [false, currentLevel, "已达最大强化等级"]

	var rate : float = GetSuccessRate(currentLevel)
	# 运气加成
	rate = min(1.0, rate + luckBonus * 0.1)

	var roll : float = randf()
	if roll <= rate:
		# 成功
		var newLevel : int = currentLevel + 1
		return [true, newLevel, "强化成功！装备强化至 +%d" % newLevel]
	else:
		# 失败
		var penalty : int = GetFailPenalty(currentLevel)
		var newLevel : int = currentLevel + penalty
		if newLevel < 0:
			newLevel = 0
		if penalty == 0:
			return [false, currentLevel, "强化失败，等级不变"]
		elif penalty == -1:
			return [false, newLevel, "强化失败，装备已损坏！"]
		else:
			return [false, newLevel, "强化失败，等级下降至 +%d" % newLevel]

# 获取强化等级显示字符串
static func GetEnhanceLevelString(level : int) -> String:
	if level <= 0:
		return ""
	return "+%d" % level

# 获取强化等级颜色
static func GetEnhanceLevelColor(level : int) -> Color:
	return RaritySystem.GetEnhanceColor(level)

# 计算装备最终属性（基础属性 × 强化倍数）
static func CalculateEnhancedValue(baseValue : float, enhanceLevel : int) -> float:
	var multiplier : float = GetEnhanceMultiplier(enhanceLevel)
	return baseValue * multiplier
