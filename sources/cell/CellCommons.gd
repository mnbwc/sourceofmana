extends RefCounted
class_name CellCommons

enum Type
{
	ITEM = 0,
	EMOTE,
	SKILL,
	COUNT
}

#
static func CompareCell(cell : BaseCell, id : int, customfield : String) -> bool:
	return cell and \
	cell.id == id and \
	(
		cell is not ItemCell or \
		cell.customfield == customfield \
	)

static func IsSameItem(cell : BaseCell, item : Item) -> bool:
	return item and CompareCell(cell, item.cellID, item.cellCustomfield)

static func IsSameCell(cellA : BaseCell, cellB : BaseCell) -> bool:
	return cellB and CompareCell(cellA, cellB.id, cellB.customfield if cellB is ItemCell else "")

static func IsEquipment(cell : ItemCell) -> bool:
	return cell.slot >= ActorCommons.Slot.FIRST_EQUIPMENT and cell.slot < ActorCommons.Slot.LAST_EQUIPMENT

static func IsEquipped(cell : BaseCell) -> bool:
	return cell and cell is ItemCell and IsEquipment(cell) and \
	Launcher.Player and Launcher.Player.inventory and Launcher.Player.inventory.equipment and \
	IsSameItem(cell, Launcher.Player.inventory.equipment[cell.slot])

enum Modifier {
	None = 0,
	Health,
	Mana,
	Stamina,
	MaxMana,
	RegenMana,
	CritRate,
	MAttack,
	MDefense,
	MaxStamina,
	RegenStamina,
	CooldownDelay,
	MaxHealth,
	RegenHealth,
	Defense,
	CastDelay,
	DodgeRate,
	AttackRange,
	WalkSpeed,
	WeightCapacity,
	Attack,
	Hide,
	Invisible,
	Accuracy,
	Count
}

static func GetModifierDisplayName(effect : Modifier) -> String:
	match effect:
		Modifier.Health:		return "Health"
		Modifier.Mana:			return "Mana"
		Modifier.Stamina:		return "Stamina"
		Modifier.MaxHealth:		return "最大生命"
		Modifier.MaxMana:		return "最大法力"
		Modifier.MaxStamina:	return "最大耐力"
		Modifier.Attack:		return "Attack"
		Modifier.Defense:		return "Defense"
		Modifier.MAttack:		return "魔法攻击"
		Modifier.MDefense:		return "魔法防御"
		Modifier.AttackRange:	return "攻击范围"
		Modifier.CritRate:		return "暴击率"
		Modifier.DodgeRate:		return "闪避率"
		Modifier.Accuracy:		return "Accuracy"
		Modifier.CastDelay:		return "施法延迟"
		Modifier.CooldownDelay:	return "Cooldown"
		Modifier.RegenHealth:	return "生命回复"
		Modifier.RegenMana:		return "法力回复"
		Modifier.RegenStamina:	return "耐力回复"
		Modifier.WalkSpeed:		return "移动速度"
		Modifier.WeightCapacity: return "负重"
		_:						return "Unknown"

static func IsInverseModifier(effect : Modifier) -> bool:
	return effect == Modifier.CastDelay or effect == Modifier.CooldownDelay

static func IsResourceModifier(effect : Modifier) -> bool:
	return effect == Modifier.Health or effect == Modifier.Mana or effect == Modifier.Stamina

static func GetModifierColor(effect : Modifier, value : Variant) -> Color:
	var val : float = -float(value) if IsInverseModifier(effect) else float(value)
	if val > 0.0:
		return UICommons.ModifierPositiveColor
	elif val < 0.0:
		return UICommons.ModifierNegativeColor
	return UICommons.LightTextColor

static func GetModifierDiffBBCode(effect : Modifier, diff : Variant) -> String:
	if float(diff) == 0.0:
		return ""
	var diffColor : String = "#" + GetModifierColor(effect, diff).to_html(false)
	var arrow : String = "↑" if float(diff) > 0.0 else "↓"
	return " [color=%s](%s %s)[/color]" % [diffColor, FormatModifierValue(effect, diff), arrow]

static func GetPercentDiffBBCode(diffPercent : float, inverse : bool = false) -> String:
	if diffPercent == 0.0:
		return ""
	var isGood : bool = diffPercent < 0.0 if inverse else diffPercent > 0.0
	var diffColor : String = "#" + (UICommons.ModifierPositiveColor.to_html(false) if isGood else UICommons.ModifierNegativeColor.to_html(false))
	var arrow : String = "↑" if diffPercent > 0.0 else "↓"
	return " [color=%s](%s%.2f%% %s)[/color]" % [diffColor, ("+" if diffPercent > 0.0 else ""), diffPercent, arrow]

static func FormatModifierValue(effect : Modifier, value : Variant) -> String:
	match effect:
		Modifier.CritRate, Modifier.DodgeRate, Modifier.Accuracy:
			var floatVal : float = float(value) * 100.0
			return ("+" if floatVal >= 0.0 else "") + ("%.2f" % floatVal) + "%"
		Modifier.CastDelay, Modifier.CooldownDelay:
			var floatVal : float = float(value)
			return ("+" if floatVal >= 0.0 else "") + ("%.2f" % floatVal) + "s"
		_:
			var intVal : int = int(value)
			return ("+" if intVal >= 0 else "") + str(intVal)
