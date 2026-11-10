extends RefCounted
class_name AffixSystem

# ============================================================
# AffixSystem — 词条系统
# 前缀 + 后缀，随机生成装备词条
# ============================================================

enum AffixType {
	PREFIX = 0,   # 前缀（攻击向）
	SUFFIX = 1,   # 后缀（防御向）
}

# 词条定义
class AffixDef:
	var id : int = 0
	var name : String = ""
	var nameEn : String = ""
	var type : AffixType = AffixType.PREFIX
	var effect : CellCommons.Modifier = CellCommons.Modifier.None
	var minValue : float = 0.0
	var maxValue : float = 0.0
	var rarityMin : RaritySystem.Rarity = RaritySystem.Rarity.COMMON
	var weight : float = 1.0

	func _init(p_id : int, p_name : String, p_nameEn : String, p_type : AffixType, p_effect : CellCommons.Modifier, p_min : float, p_max : float, p_rarity : RaritySystem.Rarity, p_weight : float = 1.0):
		id = p_id
		name = p_name
		nameEn = p_nameEn
		type = p_type
		effect = p_effect
		minValue = p_min
		maxValue = p_max
		rarityMin = p_rarity
		weight = p_weight

# 已生成的词条实例
class AffixInstance:
	var def : AffixDef = null
	var value : float = 0.0

	func _init(p_def : AffixDef, p_value : float):
		def = p_def
		value = p_value

	func GetDescription() -> String:
		if def == null:
			return ""
		var valStr : String = _format_value()
		if def.nameEn != "":
			return "%s +%s" % [def.nameEn, valStr]
		return "+%s %s" % [valStr, CellCommons.GetModifierDisplayName(def.effect)]

	func GetDescriptionZh() -> String:
		if def == null:
			return ""
		var valStr : String = _format_value()
		if def.name != "":
			return "%s +%s" % [def.name, valStr]
		return "+%s %s" % [valStr, CellCommons.GetModifierDisplayName(def.effect)]

	func _format_value() -> String:
		if def.effect == CellCommons.Modifier.CritRate or def.effect == CellCommons.Modifier.DodgeRate:
			return "%.1f%%" % value
		if value == int(value):
			return str(int(value))
		return "%.1f" % value

# 词条池
static var _affixPool : Array[AffixDef] = []
static var _initialized : bool = false

static func _init_pool():
	if _initialized:
		return
	_initialized = true

	# ===== 前缀（攻击向）=====
	# 力量
	_affixPool.append(AffixDef.new(1, "之力", "of Strength", AffixType.PREFIX, CellCommons.Modifier.Attack, 1.0, 10.0, RaritySystem.Rarity.COMMON))
	_affixPool.append(AffixDef.new(2, "之锋", "of Sharpness", AffixType.PREFIX, CellCommons.Modifier.Attack, 5.0, 25.0, RaritySystem.Rarity.UNCOMMON))
	_affixPool.append(AffixDef.new(3, "之怒", "of Fury", AffixType.PREFIX, CellCommons.Modifier.Attack, 15.0, 50.0, RaritySystem.Rarity.RARE))
	_affixPool.append(AffixDef.new(4, "之毁灭", "of Destruction", AffixType.PREFIX, CellCommons.Modifier.Attack, 30.0, 100.0, RaritySystem.Rarity.EPIC))

	# 暴击
	_affixPool.append(AffixDef.new(5, "之精准", "of Precision", AffixType.PREFIX, CellCommons.Modifier.CritRate, 1.0, 5.0, RaritySystem.Rarity.COMMON))
	_affixPool.append(AffixDef.new(6, "之致命", "of Lethality", AffixType.PREFIX, CellCommons.Modifier.CritRate, 3.0, 10.0, RaritySystem.Rarity.UNCOMMON))
	_affixPool.append(AffixDef.new(7, "之暗杀", "of Assassination", AffixType.PREFIX, CellCommons.Modifier.CritRate, 5.0, 15.0, RaritySystem.Rarity.RARE))

	# 魔法攻击
	_affixPool.append(AffixDef.new(8, "之魔力", "of Magic", AffixType.PREFIX, CellCommons.Modifier.MAttack, 1.0, 8.0, RaritySystem.Rarity.COMMON))
	_affixPool.append(AffixDef.new(9, "之秘术", "of Arcane", AffixType.PREFIX, CellCommons.Modifier.MAttack, 5.0, 20.0, RaritySystem.Rarity.UNCOMMON))

	# 攻击范围
	_affixPool.append(AffixDef.new(10, "之远见", "of Vision", AffixType.PREFIX, CellCommons.Modifier.AttackRange, 1.0, 3.0, RaritySystem.Rarity.COMMON))

	# 移动速度
	_affixPool.append(AffixDef.new(11, "之疾风", "of Wind", AffixType.PREFIX, CellCommons.Modifier.WalkSpeed, 1.0, 5.0, RaritySystem.Rarity.UNCOMMON))

	# ===== 后缀（防御向）=====
	# 生命
	_affixPool.append(AffixDef.new(20, "之生命", "of Life", AffixType.SUFFIX, CellCommons.Modifier.MaxHealth, 5.0, 50.0, RaritySystem.Rarity.COMMON))
	_affixPool.append(AffixDef.new(21, "之活力", "of Vitality", AffixType.SUFFIX, CellCommons.Modifier.MaxHealth, 20.0, 100.0, RaritySystem.Rarity.UNCOMMON))
	_affixPool.append(AffixDef.new(22, "之巨人", "of Giant", AffixType.SUFFIX, CellCommons.Modifier.MaxHealth, 50.0, 200.0, RaritySystem.Rarity.RARE))

	# 防御
	_affixPool.append(AffixDef.new(23, "之守护", "of Protection", AffixType.SUFFIX, CellCommons.Modifier.Defense, 1.0, 8.0, RaritySystem.Rarity.COMMON))
	_affixPool.append(AffixDef.new(24, "之坚壁", "of Fortress", AffixType.SUFFIX, CellCommons.Modifier.Defense, 5.0, 20.0, RaritySystem.Rarity.UNCOMMON))
	_affixPool.append(AffixDef.new(25, "之不朽", "of Immortality", AffixType.SUFFIX, CellCommons.Modifier.Defense, 15.0, 50.0, RaritySystem.Rarity.RARE))

	# 魔法防御
	_affixPool.append(AffixDef.new(26, "之抗性", "of Resistance", AffixType.SUFFIX, CellCommons.Modifier.MDefense, 1.0, 8.0, RaritySystem.Rarity.COMMON))
	_affixPool.append(AffixDef.new(27, "之屏障", "of Barrier", AffixType.SUFFIX, CellCommons.Modifier.MDefense, 5.0, 20.0, RaritySystem.Rarity.UNCOMMON))

	# 闪避
	_affixPool.append(AffixDef.new(28, "之灵巧", "of Agility", AffixType.SUFFIX, CellCommons.Modifier.DodgeRate, 1.0, 5.0, RaritySystem.Rarity.COMMON))
	_affixPool.append(AffixDef.new(29, "之幻影", "of Phantom", AffixType.SUFFIX, CellCommons.Modifier.DodgeRate, 3.0, 10.0, RaritySystem.Rarity.UNCOMMON))

	# 法力
	_affixPool.append(AffixDef.new(30, "之法力", "of Mana", AffixType.SUFFIX, CellCommons.Modifier.MaxMana, 5.0, 30.0, RaritySystem.Rarity.COMMON))
	_affixPool.append(AffixDef.new(31, "之智慧", "of Wisdom", AffixType.SUFFIX, CellCommons.Modifier.MaxMana, 20.0, 80.0, RaritySystem.Rarity.UNCOMMON))

	# 回复
	_affixPool.append(AffixDef.new(32, "之恢复", "of Recovery", AffixType.SUFFIX, CellCommons.Modifier.RegenHealth, 1.0, 3.0, RaritySystem.Rarity.COMMON))
	_affixPool.append(AffixDef.new(33, "之再生", "of Regeneration", AffixType.SUFFIX, CellCommons.Modifier.RegenHealth, 2.0, 8.0, RaritySystem.Rarity.UNCOMMON))

	# 耐力
	_affixPool.append(AffixDef.new(34, "之耐力", "of Stamina", AffixType.SUFFIX, CellCommons.Modifier.MaxStamina, 5.0, 30.0, RaritySystem.Rarity.COMMON))

	# 负重
	_affixPool.append(AffixDef.new(35, "之负重", "of Burden", AffixType.SUFFIX, CellCommons.Modifier.WeightCapacity, 5.0, 20.0, RaritySystem.Rarity.COMMON))

# 获取符合条件的词条池
static func _get_eligible_pool(rarity : RaritySystem.Rarity, type : AffixType) -> Array[AffixDef]:
	_init_pool()
	var result : Array[AffixDef] = []
	for def in _affixPool:
		if def.rarityMin <= rarity and def.type == type:
			result.append(def)
	return result

# 随机生成一件装备的词条列表
static func GenerateAffixes(rarity : RaritySystem.Rarity, luckBonus : float = 0.0) -> Array[AffixInstance]:
	_init_pool()
	var result : Array[AffixInstance] = []

	var countRange : Vector2i = RaritySystem.GetAffixCountRange(rarity)
	var count : int = randi_range(countRange.x, countRange.y)

	# 运气加成：可能多一条词条
	if luckBonus > 0.0 and randf() < luckBonus * 0.5:
		count += 1

	var prefixes : Array[AffixDef] = _get_eligible_pool(rarity, AffixType.PREFIX)
	var suffixes : Array[AffixDef] = _get_eligible_pool(rarity, AffixType.SUFFIX)

	# 打乱顺序
	prefixes.shuffle()
	suffixes.shuffle()

	var prefixCount : int = 0
	var suffixCount : int = 0

	for i in range(count):
		var type : AffixType = AffixType.PREFIX if i % 2 == 0 else AffixType.SUFFIX
		var pool : Array[AffixDef] = prefixes if type == AffixType.PREFIX else suffixes

		if pool.is_empty():
			type = AffixType.SUFFIX if type == AffixType.PREFIX else AffixType.PREFIX
			pool = prefixes if type == AffixType.PREFIX else suffixes

		if pool.is_empty():
			continue

		# 按权重随机选择
		var def : AffixDef = _weighted_random(pool)
		if def:
			var value : float = randf_range(def.minValue, def.maxValue)
			# 稀有度倍数加成
			var multiplier : float = RaritySystem.GetRarityMultiplier(rarity)
			value *= multiplier
			result.append(AffixInstance.new(def, value))

	return result

static func _weighted_random(pool : Array[AffixDef]) -> AffixDef:
	var totalWeight : float = 0.0
	for def in pool:
		totalWeight += def.weight

	var roll : float = randf() * totalWeight
	var cumulative : float = 0.0
	for def in pool:
		cumulative += def.weight
		if roll <= cumulative:
			return def

	return pool[0] if pool.size() > 0 else null

# 将词条转换为修饰符（用于装备属性计算）
static func AffixesToModifiers(affixes : Array[AffixInstance]) -> CellModifier:
	var modifier : CellModifier = CellModifier.new()
	for affix in affixes:
		if affix and affix.def:
			var statMod : StatModifier = StatModifier.new()
			statMod._effect = affix.def.effect
			statMod._value = affix.value
			statMod._persistent = true
			modifier.Add(statMod)
	return modifier

# 生成带词条的装备名称
static func GenerateItemName(baseName : String, affixes : Array[AffixInstance]) -> String:
	if affixes.is_empty():
		return baseName

	var prefixName : String = ""
	var suffixName : String = ""

	for affix in affixes:
		if affix and affix.def:
			if affix.def.type == AffixType.PREFIX:
				prefixName = affix.def.nameEn if affix.def.nameEn != "" else affix.def.name
			else:
				suffixName = affix.def.nameEn if affix.def.nameEn != "" else affix.def.name

	var result : String = baseName
	if prefixName != "":
		result = prefixName + " " + result
	if suffixName != "":
		result = result + " " + suffixName

	return result

# 生成带词条的装备名称（中文）
static func GenerateItemNameZh(baseNameZh : String, affixes : Array[AffixInstance]) -> String:
	if affixes.is_empty():
		return baseNameZh

	var prefixName : String = ""
	var suffixName : String = ""

	for affix in affixes:
		if affix and affix.def:
			if affix.def.type == AffixType.PREFIX:
				prefixName = affix.def.name
			else:
				suffixName = affix.def.name

	var result : String = baseNameZh
	if prefixName != "":
		result = prefixName + result
	if suffixName != "":
		result = result + suffixName

	return result
