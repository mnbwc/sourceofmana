extends NpcScript

var artisFlourSackID : int			= DB.GetCellHash("Artis Flour Sack")

#
func OnStart():
	match GetQuest(ProgressCommons.Quest.GRAIN_IN_THE_SAND):
		ProgressCommons.GRAIN_IN_THE_SAND.STARTED:
			OnSearch()

func OnSearch():
	Mes("一个深蓝色的火漆印，上面印着阿蒂斯的纹章。就是这个。")
	Mes("里面是精细研磨的面粉袋，正如里斯金描述的那样。")

	if not HasSpace(1):
		Mes("但你的背包太满了，装不下它们。")
		return

	SetQuest(ProgressCommons.Quest.GRAIN_IN_THE_SAND, ProgressCommons.GRAIN_IN_THE_SAND.SEARCHED_CRATES)
	AddItem(artisFlourSackID)
