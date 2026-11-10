extends MerchantScript

#
func GetStock() -> PackedStringArray:
	return ["Short Sword", "Cleaver", "Gladius"]

func OnStart():
	Mes("需要刀吗？守卫都来找我买。")
	super.OnStart()
