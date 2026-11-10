extends MerchantScript

#
func GetStock() -> PackedStringArray:
	return ["Shorts", "Cotton Shirt", "Tank Top"]

func OnStart():
	Mes("在太阳把你烤熟之前，先给自己弄件轻的衣服！")
	super.OnStart()
