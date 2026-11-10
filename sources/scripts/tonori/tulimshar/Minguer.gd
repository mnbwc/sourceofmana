extends MerchantScript

#
func GetStock() -> PackedStringArray:
	return ["Bandana Scarf", "Desert Boots"]

func OnStart():
	Mes("滚烫的沙子很快就会烧穿赤脚。需要靴子吗？")
	super.OnStart()
