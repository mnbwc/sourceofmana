extends NpcScript

#
func OnStart():
	var questState : int = GetQuest(ProgressCommons.Quest.TUTORIAL)
	match questState:
		ProgressCommons.TUTORIAL.INACTIVE:
			OnFirstMeeting()
		ProgressCommons.TUTORIAL.INTRO_ITEMS_GIVEN:
			OnFeelingChoice()
		ProgressCommons.TUTORIAL.POTION_GIVEN:
			OnFeelingBetter()
		ProgressCommons.TUTORIAL.CLOTHES_GIVEN:
			OnExplainUI()
		ProgressCommons.TUTORIAL.UI_EXPLAINED: # Deprecated
			Mes("你还有别的事吗？")
			SetQuest(ProgressCommons.Quest.TUTORIAL, ProgressCommons.TUTORIAL.ELANORE_DONE)
			OnSendToKael()
			Action(OnMainChoice)
		_:
			Mes("欢迎回来。我能为你做什么？")
			OnMainChoice()

# First meeting
func OnFirstMeeting():
	var waterBottleID : int = DB.GetCellHash("Water Bottle")
	var cactusSourCandyID : int = DB.GetCellHash("Cactus Sour Candy")

	Mes("你好，欢迎来到图利姆沙！")
	Mes("你及时赶到了城门。守卫在地上发现了你，几乎看不见。")
	Mes("我叫埃拉诺。你感觉怎么样？你需要喝这些水，也请吃这个，会让你感觉好些。")
	SetQuest(ProgressCommons.Quest.TUTORIAL, ProgressCommons.TUTORIAL.INTRO_ITEMS_GIVEN)
	AddItem(waterBottleID)
	AddItem(cactusSourCandyID)
	OnFeelingChoice()

func OnFeelingChoice():
	Choice("I am feeling better now, thank you.", OnFeelingBetter)
	Choice("I am still feeling a bit weak...", OnFeelingWeak)

func OnFeelingWeak():
	var pitayaID : int = DB.GetCellHash("Pitaya")

	Mes("哦……好吧。来，拿一个。除此之外，你需要花点时间，亲爱的。")
	SetQuest(ProgressCommons.Quest.TUTORIAL, ProgressCommons.TUTORIAL.POTION_GIVEN)
	AddItem(pitayaID)
	OnFeelingBetter()

func OnFeelingBetter():
	Mes("你在沙漠里闲逛什么？最近很危险，你不该一个人出去。")
	Choice("I am a mercenary soldier trying to gain local fame.", OnGiveStarterClothes)
	Choice("I am looking to start trades with natives of the region.", OnGiveStarterClothes)
	Choice("I am a scholar seeking knowledge on Mana and magic.", OnGiveStarterClothes)
	Choice("I am a sailor looking to explore new opportunities on and off land.", OnGiveStarterClothes)

func OnGiveStarterClothes():
	var cottonShirtID : int = DB.GetCellHash("Cotton Shirt")
	var linenShortsID : int = DB.GetCellHash("Shorts")

	Mes("嗯，有你在真好。城市需要有能力的人帮助维护安全。")
	Mes("最近沙漠变得更危险了。一群痴迷卡厄的狂热分子藏在东边的山里。他们一直在袭击旅人，有一次在大规模战斗中差点攻破城墙。")
	Mes("总之，这些是基础衣物，用来换掉我们捡到你时你穿的破布。")
	SetQuest(ProgressCommons.Quest.TUTORIAL, ProgressCommons.TUTORIAL.CLOTHES_GIVEN)
	AddItem(cottonShirtID, 1, "Used")
	AddItem(linenShortsID, 1, "Used")
	OnExplainUI()

func OnExplainUI():
	Mes("在我让你走之前，让我给你讲几件事，帮你在外面活下来。")
	HighlightUI(UICommons.UITarget.STATINDICATOR)
	Narrate("这些是你的重要资源。随时注意它们。")
	HighlightUI(UICommons.UITarget.HEALTHBAR)
	Narrate("你的生命条显示你在死亡前能承受多少伤害。")
	HighlightUI(UICommons.UITarget.MANABAR)
	Narrate("你的法力为技能提供能量。没有它，许多能力无法使用。")
	HighlightUI(UICommons.UITarget.STAMINABAR)
	Narrate("耐力决定你能在疲劳前冲刺和执行身体动作的时间。")
	HighlightUI(UICommons.UITarget.MENUINDICATOR)
	Narrate("菜单让你访问背包、技能、任务、设置等。用它管理你携带和知道的一切。")
	HighlightUI(UICommons.UITarget.ACTION_BAR)
	Narrate("动作条让你放置技能和物品以便快速使用。把最需要的东西拖上去，战斗中每一秒都很重要。")
	HighlightUI(UICommons.UITarget.INVENTORY)
	Narrate("在背包窗口可以管理你的物品、装备、药水和找到的收集品。")
	HighlightUI(UICommons.UITarget.NONE)
	SetQuest(ProgressCommons.Quest.TUTORIAL, ProgressCommons.TUTORIAL.ELANORE_DONE)
	Mes("但我不该让你整天待在这里。你是新来的，想找到自己的立足之地。")
	OnSendToKael()
	DisplayActions(["gp_interact", "gp_target"])
	Narrate("在NPC附近使用交互键与他们交谈。你也可以先使用目标键选择一个。")
	Action(Farewell)

# Main choice loop
func GetPotionBatchCount() -> int:
	var maggotSlimeID : int = DB.GetCellHash("Maggot Slime")
	var waterBottleID : int = DB.GetCellHash("Water Bottle")
	var pitayaID : int = DB.GetCellHash("Pitaya")

	var batchCount : int = 0
	while HasItem(maggotSlimeID, (batchCount + 1) * 6) and HasItem(waterBottleID, batchCount + 1) and HasItem(pitayaID, batchCount + 1):
		batchCount += 1
	return batchCount

func OnMainChoice():
	if GetQuest(ProgressCommons.Quest.TUTORIAL) < ProgressCommons.CompletedProgress:
		Choice("Where should I go now?", OnDirections)

	var potionQuestState : int = GetQuest(ProgressCommons.Quest.ELANORE_POTION)
	var hasPotionIngredients : bool = GetPotionBatchCount() > 0
	match potionQuestState:
		ProgressCommons.ELANORE_POTION.INACTIVE:
			Choice("Can I help you with anything?", OnHelpWithPotions)
		ProgressCommons.ELANORE_POTION.STARTED:
			if hasPotionIngredients:
				Choice("I have your ingredients.", OnPotionQuestTurnIn)
			else:
				Choice("About those ingredients...", OnPotionQuestReminder)
		ProgressCommons.ELANORE_POTION.BREWED:
			if hasPotionIngredients:
				Choice("I brought you more ingredients.", OnPotionQuestTurnIn)
			else:
				Choice("About those ingredients...", OnPotionQuestReminder)

	Choice("What is Kaore?", OnExplainKaore)
	Choice("你是谁？", OnExplainSelf)

	if GetQuest(ProgressCommons.Quest.TUTORIAL) == ProgressCommons.CompletedProgress:
		Choice("Thank you again but I have to leave", Farewell)

# Help with potions
func OnHelpWithPotions():
	Mes("是的！我为我们的守卫和这座城市的人们做很多治疗药水。它们是对大多数病症的速效药，真的帮助让人们更安全。")
	Mes("治疗药水需要知识才能制作，还需要一些关键材料。也许你可以通过帮我收集这些材料来重新振作起来？")
	Choice("I'll gather them for you.", OnPotionQuestAccept)
	Choice("Maybe later.", OnPotionQuestDecline)

func OnPotionQuestAccept():
	Mes("每瓶药水我需要六个蛆虫粘液、一个水和一个火龙果。把它们带给我，我就给你做一瓶仙人掌药水作为回报。")
	SetQuest(ProgressCommons.Quest.ELANORE_POTION, ProgressCommons.ELANORE_POTION.STARTED)
	Action(OnMainChoice)

func OnPotionQuestDecline():
	Mes("你什么时候准备好，交易都有效。")
	Action(OnMainChoice)

func OnPotionQuestReminder():
	Mes("我还是每瓶药水需要六个蛆虫粘液、一个水和一个火龙果。有了再来。")
	OnMainChoice()

func OnPotionQuestTurnIn():
	var batchCount : int = GetPotionBatchCount()
	if batchCount > 1:
		Mes("你带来的够做 %d 瓶药水！我应该做几瓶？" % batchCount)
		Choice("Just one.", OnPotionBrew.bind(1))
		Choice("All %d of them." % batchCount, OnPotionBrew.bind(batchCount))
	else:
		OnPotionBrew(1)

func OnPotionBrew(brewCount : int):
	var removeItems : Array = [[DB.GetCellHash("Maggot Slime"), 6 * brewCount], [DB.GetCellHash("Water Bottle"), brewCount], [DB.GetCellHash("Pitaya"), brewCount]]
	ExchangeItems(removeItems, [[DB.GetCellHash("Cactus Potion"), brewCount]], OnPotionBrewed)

func OnPotionBrewed(result : ActorInventory.ExchangeResult):
	match result:
		ActorInventory.ExchangeResult.OK:
			var isFirstBrew : bool = GetQuest(ProgressCommons.Quest.ELANORE_POTION) == ProgressCommons.ELANORE_POTION.STARTED
			SetQuest(ProgressCommons.Quest.ELANORE_POTION, ProgressCommons.ELANORE_POTION.BREWED)
			if isFirstBrew:
				Mes("谢谢你的帮助。我这就开始做新的治疗药水！")
				Mes("在这里帮助人民比困在塔楼或宫殿里好得多。这座城市更多的领导者应该想想这一点。")
			else:
				Mes("谢谢，给你！")
			Action(OnPotionBrewedChoice)
		ActorInventory.ExchangeResult.MISSING_ITEMS:
			Mes("嗯，你好像没有材料了。")
			Action(OnMainChoice)
		ActorInventory.ExchangeResult.NO_SPACE:
			Mes("你的背包满了，先腾点空间。")
			Action(OnMainChoice)

func OnPotionBrewedChoice():
	Choice("I have something else to ask.", OnMainChoice)
	Choice("Goodbye.", Farewell)

# What is Kaore
func OnExplainKaore():
	Mes("魔力是连接我们所有人的生命能量。卡厄是魔力的衰败形式，它腐蚀生命而不是激活生命。还有更多内容，但大概就是这样。")
	LookAtNpc("Nina")
	Mes("如果你想了解更多，最好问我的学徒妮娜。她目前负责管理城市的灵魂巨石。你可以从她那里学到更多。")
	ResetCamera()
	OnMainChoice()

# Who are you
func OnExplainSelf():
	Mes("我叫埃拉诺。我是这座城市和周围土地的卡韦。教团外的人叫我们德鲁伊。我代表考马图亚，一个致力于世界上最古老传统的教团。")
	Mes("但你现在别操心那个。以后可以再聊。现在我就在这里帮忙配药水，接收城墙外来的伤员。")
	OnMainChoice()

func Farewell():
	if randi() % 2:
		Chat("在外面照顾好自己。")
	else:
		Chat("注意安全。")

# Directions
func OnDirections():
	var questState : int = GetQuest(ProgressCommons.Quest.TUTORIAL)
	if questState < ProgressCommons.TUTORIAL.KAEL_DONE:
		OnSendToKael()
	elif questState < ProgressCommons.CompletedProgress:
		OnSendToEkinu()
	Action(Farewell)

func OnSendToKael():
	Mes("守卫凯尔目前负责城墙内的本地巡逻，如果你在找工作应该和他谈谈。")
	LookAtNpc("Kael")
	Mes("他在这东北方向，靠近一些棕榈树。")
	Mes("沿着这堵墙走到拐角，然后走上去。")
	ResetCamera()

func OnSendToEkinu():
	Mes("守卫埃基努驻守在南边的城门。")
	LookAtNpc("Ekinu")
	Mes("告诉他你为凯尔做了什么，他会知道下一步怎么安排你。")
	ResetCamera()
