extends NpcScript

# Quest items
var artisFlourSackID : int			= DB.GetCellHash("Artis Flour Sack")

# Reward items
var croissantID : int				= DB.GetCellHash("Croissant")
var cactusSourCandyID : int			= DB.GetCellHash("Cactus Sour Candy")

#
func OnStart():
	var questState : int = GetQuest(ProgressCommons.Quest.GRAIN_IN_THE_SAND)
	match questState:
		ProgressCommons.GRAIN_IN_THE_SAND.INACTIVE:
			OnInactive()
		ProgressCommons.GRAIN_IN_THE_SAND.SEARCHED_CRATES:
			OnReward()
		ProgressCommons.GRAIN_IN_THE_SAND.REWARDS_WITHDREW:
			OnComplete()
		_:
			OnKeepLooking()

# Quest states
func OnInactive():
	Mes("啊，欢迎！恐怕现在货架是空的。")
	Mes("图利姆沙没有好农田。你看看周围，我们在沙漠里。这里除了仙人掌什么也不长。")
	Mes("我的面粉从阿蒂斯运来，以维持我的面包店运营。我的沙暴面包养活这座城市很多年了。")
	Mes("今早码头到了一批货，但在这种高温下来回走动太累了。")
	Mes("如果我自己去，今天就累得烤不了东西了。")
	Mes("阿蒂斯面包店的桶上有蓝色火漆印。你能把里面的面粉袋带给我吗？")

	QuestChoice()

func QuestChoice(previousChoice : int = -1):
	Choice("I'll check the docks for you.", OnAccept)
	if previousChoice != 1:
		Choice("What's Sandstorm Bread?", OnAskBread)
	if previousChoice != 2:
		Choice("Does anything grow out here?", OnAskDesert)
	Choice("Maybe later.", OnDecline)

func OnKeepLooking():
	Mes("在码头有收获吗？找蓝色封蜡标记的桶。")
	Mes("我会把炉子烧热准备好。")

func OnComplete():
	Mes("即使在这种高温下，一个热面包也很难拒绝。")
	CompleteChoice()

func CompleteChoice(previousChoice : int = -1):
	if previousChoice != 0:
		Choice("Tell me about Artis.", OnAskArtis)
	if previousChoice != 1:
		Choice("How does Tulimshar survive?", OnAskCity)
	if previousChoice != 2:
		Choice("What's Sandstorm Bread?", OnAskBread)
	if previousChoice != 3:
		Choice("Does anything grow out here?", OnAskDesert)
	if previousChoice != -1:
		Choice("保重。", OnFarewell)

# Optional dialogue
func OnAskBread():
	Mes("我的招牌配方！阿蒂斯的面粉，用仙人掌汁代替普通水，加一撮沙漠盐。")
	Mes("外皮能撑过沙暴，第二天早上仍然新鲜。这就是名字的由来。")
	if IsQuestCompleted(ProgressCommons.Quest.GRAIN_IN_THE_SAND):
		CompleteChoice(2)
	else:
		QuestChoice(1)

func OnAskDesert():
	Mes("主要是仙人掌。生命力顽强的植物，我们充分利用它们。做糖、饮料，甚至药材。")
	Mes("但谷物？托诺里从来不是种谷物的地方。没有商船，这座城市会闹饥荒。")
	if IsQuestCompleted(ProgressCommons.Quest.GRAIN_IN_THE_SAND):
		CompleteChoice(3)
	else:
		QuestChoice(2)

func OnAskArtis():
	Mes("大洋彼岸的伟大港口城市，在奥罗拉海岸。肥沃的农田，技艺精湛的工匠。")
	Mes("我姐姐住在那边。她烤的饼干是最好吃的！我用的面粉都是从她的面包店运来的。")
	Mes("来自阿蒂斯的商船供应图利姆沙市场一半的货。没有它们，我们除了仙人掌几乎没别的可吃。")
	CompleteChoice(0)

func OnAskCity():
	Mes("贸易。一直是图利姆沙最大的技能。这样的城市怎么在多风的山谷沙漠中繁荣？")
	Mes("我们位于三大洲的十字路口。一切迟早都要经过图利姆沙的港口。")
	Mes("它也让我们有饭吃，但对我的生意来说，这意味着总是等下一艘船到来。")
	CompleteChoice(1)

func OnFarewell():
	Mes("随时来。总有新鲜面包等着你。")

# Transitions to next states
func OnAccept():
	SetQuest(ProgressCommons.Quest.GRAIN_IN_THE_SAND, ProgressCommons.GRAIN_IN_THE_SAND.STARTED)

	Mes("非常感谢，朋友。")
	Mes("找深蓝色的封蜡。你不会错过的。")

func OnDecline():
	Mes("没问题。炉子等着就是了。")

func OnReward():
	if not HasItem(artisFlourSackID):
		Mes("你查过码头了吗？面粉袋应该在蓝色封蜡标记的桶里。")
		return

	if not HasItemsSpace([[artisFlourSackID, -1], [cactusSourCandyID, 5], [croissantID, 5]]):
		Mes("你找到了！但看起来你的背包太满了，我没法给你任何回报。")
		Mes("腾出点空间再来。")
		return

	RemoveItem(artisFlourSackID)
	SetQuest(ProgressCommons.Quest.GRAIN_IN_THE_SAND, ProgressCommons.GRAIN_IN_THE_SAND.REWARDS_WITHDREW)

	Mes("你找到了！太好了。")
	Mes("有了这些面粉，图利姆沙在日落前就能有新鲜的沙暴面包。")

	AddItem(cactusSourCandyID, 5)
	AddItem(croissantID, 5)
	AddKarma(1)
	AddExp(30)

	Mes("来，拿我最好的。羊角面包和仙人掌糖。一个是家族秘方，另一个是图利姆沙的特产！")
