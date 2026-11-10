extends NpcScript

#
const COST_GP : int = 10000
const SNAKE_SKIN_AMOUNT : int = 10

#
func OnStart():
	Mes("世代以来，我的部落一直在用不同材料制作特殊服装。")
	Mes("也许你给我带来合适的材料，我可以为你做点什么。")
	Choice("What can you make?", OnShowRecipe)
	Choice("Where is your tribe?", OnAskTribe)
	Choice("What happened to this land?", OnAskLand)
	Choice("Maybe later.", OnFarewell)

func OnShowRecipe():
	Mes("我所有工具不在这里，让我想想……牛仔皮裤腿你有兴趣吗？")
	Mes("为此我需要一条普通牛仔短裤和大约10张蛇皮，因为操作时它们容易撕裂。")
	Mes("成交？")
	Choice("Yes, that's fine.", OnAttemptCraft)
	Choice("On second thought, maybe later.", OnNoDeal)

func OnAttemptCraft():
	var shortsID : int = DB.GetCellHash("Shorts")
	var snakeSkinID : int = DB.GetCellHash("Snake Skin")
	var chapsID : int = DB.GetCellHash("Jean Chaps")

	if not HasItem(shortsID):
		Mes("天哪，看来你没有牛仔短裤。")
		return
	if not HasItem(snakeSkinID, SNAKE_SKIN_AMOUNT):
		Mes("天哪，看来你没有足够的蛇皮。")
		return

	RemoveItem(shortsID)
	RemoveItem(snakeSkinID, SNAKE_SKIN_AMOUNT)
	AddItem(chapsID)
	Mes("给你！")
	Mes("随时再来。")

func OnAskTribe():
	Mes("我们住在这西边的沙漠悬崖。路不对外人开放。")
	Mes("我出来交易。大多数陌生人也只能走到这里。")

func OnAskLand():
	Mes("王国很久以前就撤出了这片土地。说不值得守。")
	Mes("我的族人在他们之前就到这里了，今天我们仍在这里。需要的时候我们和图利姆沙交易，但除此之外我们更愿意保持距离。")

func OnNoDeal():
	Mes("好吧，但你在别的地方绝对拿不到更好的价钱！")

func OnFarewell():
	Chat("准备好了再来。")
