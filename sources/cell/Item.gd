extends RefCounted
class_name Item

var cellID : int					= DB.UnknownHash
var cellCustomfield : String		= ""
var count : int						= 0

# RPG 系统扩展
var rarity : int					= 0  # RaritySystem.Rarity
var enhanceLevel : int				= 0  # 强化等级 0-15
var affixes : Array[Dictionary]		= [] # 词条列表 [{id, value}, ...]

func _init(_cell : ItemCell, _count : int = 1):
	cellID = _cell.id
	cellCustomfield = _cell.customfield
	count = _count

# 获取稀有度颜色
func GetRarityColor() -> Color:
	return RaritySystem.GetRarityColor(rarity)

# 获取稀有度名称
func GetRarityName() -> String:
	return RaritySystem.GetRarityName(rarity)

# 获取强化等级字符串
func GetEnhanceString() -> String:
	return EnhanceSystem.GetEnhanceLevelString(enhanceLevel)

# 获取强化颜色
func GetEnhanceColor() -> Color:
	return EnhanceSystem.GetEnhanceLevelColor(enhanceLevel)

# 计算最终属性值（基础 × 强化倍数）
func GetEnhancedValue(baseValue : float) -> float:
	return EnhanceSystem.CalculateEnhancedValue(baseValue, enhanceLevel)

# 生成显示名称（带稀有度和强化前缀）
func GetDisplayName(baseName : String) -> String:
	var prefix : String = ""
	if enhanceLevel > 0:
		prefix = "+%d " % enhanceLevel
	return prefix + baseName

# 导出为字典（用于数据库存储和网络传输）
func Export() -> Dictionary:
	var result : Dictionary = {
		"item_id": cellID,
		"customfield": cellCustomfield,
		"count": count,
	}
	if rarity > 0:
		result["rarity"] = rarity
	if enhanceLevel > 0:
		result["enhance_level"] = enhanceLevel
	if not affixes.is_empty():
		result["affixes"] = affixes
	return result

# 从字典导入
func Import(data : Dictionary):
	cellID = data.get("item_id", DB.UnknownHash)
	cellCustomfield = data.get("customfield", "")
	count = data.get("count", 1)
	rarity = data.get("rarity", 0)
	enhanceLevel = data.get("enhance_level", 0)
	if data.has("affixes"):
		affixes = data["affixes"]
