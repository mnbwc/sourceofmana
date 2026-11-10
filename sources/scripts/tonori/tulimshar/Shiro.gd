extends MerchantScript

#
func GetStock() -> PackedStringArray:
	return ["Desert Goggles", "Ghutra", "Desert Hood"]

func OnStart():
	Mes("没有合适的头部装备就冲进沙暴？你很快就会变成沙魔像的！")
	super.OnStart()
