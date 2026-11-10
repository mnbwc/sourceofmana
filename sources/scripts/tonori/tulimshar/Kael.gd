extends NpcScript

const FIELD_POSITION : Vector2 = Vector2(2912, 1312)

#
func OnStart():
	var questState : int = GetQuest(ProgressCommons.Quest.TUTORIAL)
	if questState < ProgressCommons.TUTORIAL.ELANORE_DONE:
		OnSendToElanore()
	elif questState == ProgressCommons.TUTORIAL.ELANORE_DONE:
		OnFirstMeeting()
	elif questState == ProgressCommons.TUTORIAL.KAEL_MET:
		OnCheckProgress()
	elif questState == ProgressCommons.TUTORIAL.KAEL_DONE:
		OnSendToEkinu()
	elif questState == ProgressCommons.TUTORIAL.EKINU_DONE:
		OnComplete()

# Too early
func OnSendToElanore():
	Mes("你已经醒了？")
	LookAtNpc("Elanore")
	Mes("一定要和埃拉诺谈谈，我们捡到你之后你昏迷的时候是她照顾你的。")
	ResetCamera()

# Initial encounter
func OnFirstMeeting():
	Mes("你好，你是他们从沙子里挖出来的那个人吗？很高兴看到你能走动了。我需要你的帮助。")
	Mes("蛆虫喜欢吃仙人掌。虽然这些蛆虫异常大，但它们的嘴仍然小且灵活，能绕过任何刺吃进去。")
	Mes("一旦它们突破外部，就会钻进植物从里面吃。很可怕。")
	Choice("Why are these maggots so monstrous?", OnKaoreExplanation)
	Choice("That sounds terrifying.", OnWildFauna)
	Choice("That sounds so cool!", OnWildFauna)

func OnKaoreExplanation():
	Mes("他们说它们并不总是这样的。我太年轻不记得了，现在没人记得了。")
	Mes("在卡厄时代之前，几百年前，这个世界的一些生物不像今天这么……奇怪。")
	Mes("卡厄时代改变了它们。这种蛆虫变得又大又多。其他生物被改变得更糟。")
	Mes("他们说松鼠曾经和人类一样聪明，但随着时间推移它们失去了这一切。")
	if GetQuest(ProgressCommons.Quest.TUTORIAL) >= ProgressCommons.TUTORIAL.KAEL_DONE:
		MainChoices()
	else:
		Mes("好了，闲聊到此为止。我要你到田里去清理这些乱七八糟的东西！")
		Choice("Yes, I'll get to work!", OnFieldCleanUp)
		Choice("What do you mean? What has been happening out in the desert?", OnDesertExplanation)

func OnWildFauna():
	Mes("无论如何，它们需要被清除。")
	Mes("沙漠里发生的一切也影响了城市的蛆虫。")
	Mes("更糟的是，仙人掌似乎也受到了影响。")
	Mes("它们中有些跳来跳去，做着鬼脸，还溅到附近的庄稼！你需要帮我处理这个问题。")
	Choice("Alright. I'll get to work, then.", OnFieldCleanUp)
	Choice("What do you mean? What has been happening out in the desert?", OnDesertExplanation)

func OnDesertExplanation():
	Mes("卡厄正在影响各种生物。")
	Mes("就是这种邪恶力量把生物扭曲成……我不知道……更差的自己？大概吧。更具攻击性。")
	Mes("直到它完全控制它们，然后它们也变成某种亡灵。真是个恶毒的诅咒。")
	if GetQuest(ProgressCommons.Quest.TUTORIAL) >= ProgressCommons.TUTORIAL.KAEL_DONE:
		MainChoices()
	else:
		Mes("塔楼那边的法师……玛纳伊尔。他们发布了警告，卡厄目前正在托诺里以危险的模式聚集。")
		Choice("Alright alright, it's time to kill these things. Let's do it.", OnFieldCleanUp)
		Choice("Who are the Manayir?", OnManayir)

func OnManayir():
	Mes("他们是某个古老的教团，研究魔力，在城西的塔楼里用它做谁知道还有什么的事。")
	Mes("他们是发布卡厄警告的人。")
	Mes("玛纳伊尔教团是大约27年前宣布卡厄时代结束的人。")
	Mes("从那以后，他们一直在向图利姆沙的人们通报能量的流动情况。")
	Mes("很像天气，不同浓度的魔力和卡厄不断变化，可以影响区域内所有生物。")
	Choice("Alright alright, it's time to kill these things. Let's do it.", OnFieldCleanUp)
	Choice("What's the Age of Kaore?", OnAgeOfKaore)

func OnAgeOfKaore():
	Mes("我太年轻了不记得，但在几十年前，大家的生活要困难得多。")
	Mes("卡厄曾经比魔力更充足，在几百年里，人们比今天更艰难地应对变异甚至亡灵生物。")
	Mes("我们抱怨蛆虫和佩约特，但和我父母经历过的相比，我们过得轻松多了。")
	Choice("Alright alright, it's time to kill these things. Let's do it.", OnFieldCleanUp)

# Task assignment
func OnFieldCleanUp():
	var shortSwordID : int = DB.GetCellHash("Short Sword")

	Mes("不过你不会徒手清理那片田。")
	Mes("来，拿这把剑。虽然不怎么样，但对付蛆虫和佩约特够用了。")
	AddItem(shortSwordID)
	OnFightTutorial()
	Mes("先从这北边田里的佩约特开始。")
	LookAtPosition(FIELD_POSITION)
	Mes("它们是卡厄浸染的仙人掌，对我们无害，但现在没法耕种了。")
	Mes("它们只是跳来跳去，溅到附近的庄稼来嘲弄人……")
	ResetCamera()
	Mes("除掉5株佩约特和5只蛆虫应该就足以保护我们的庄稼一天了。")
	HighlightUI(UICommons.UITarget.PROGRESS)
	Narrate("进度窗口会分组显示你所有进行中的任务和怪物遭遇。")
	HighlightUI(UICommons.UITarget.NONE)
	Narrate("你可以在菜单栏的魔力百科标签中查看你消灭的怪物数量。")
	SetQuest(ProgressCommons.Quest.TUTORIAL, ProgressCommons.TUTORIAL.KAEL_MET)
	npc.ownScript.TrackPlayer(own)

# Progress check
func OnCheckProgress():
	var kaelGlobal : KaelGlobal = npc.ownScript as KaelGlobal
	kaelGlobal.TrackPlayer(own)

	var peyoteKills : int = kaelGlobal.GetKillCount(own, kaelGlobal.peyoteID)
	var maggotKills : int = kaelGlobal.GetKillCount(own, kaelGlobal.maggotID)
	if peyoteKills >= KaelGlobal.PEYOTE_REQUIRED and maggotKills >= KaelGlobal.MAGGOT_REQUIRED:
		OnTaskComplete()
	elif peyoteKills < KaelGlobal.PEYOTE_REQUIRED:
		Mes("还没完。我还需要你清除田里 %d 株佩约特。" % (KaelGlobal.PEYOTE_REQUIRED - peyoteKills))
	else:
		Mes("佩约特处理得不错。不过还需要你处理 %d 只蛆虫。" % (KaelGlobal.MAGGOT_REQUIRED - maggotKills))

# Task completion
func OnTaskComplete():
	Mes("干得漂亮！")
	Mes("你处理得非常快，比我预期的好得多。")
	Mes("你应该向守卫埃基努报告，在遇见埃拉诺的城门附近。告诉他我对你在这里的工作很满意。")
	Mes("我们很快要去沙漠做一次小型探险，你可能是帮助我们的人选。")
	SetQuest(ProgressCommons.Quest.TUTORIAL, ProgressCommons.TUTORIAL.KAEL_DONE)
	AddExp(20)
	npc.ownScript.UntrackPlayer(own)
	ClearTracker()

#
func OnSendToEkinu():
	Mes("准备好了向守卫埃基努报告。")
	Mes("他会知道接下来该做什么。")
	MainChoices()

#
func OnComplete():
	Mes("见到你真好！你也看到了，我还在被这些恶心的白色蛆虫包围着。")
	MainChoices()

#
func MainChoices():
	Choice("Why are these maggots so monstrous?", OnKaoreExplanation)
	Choice("How can I fight them?", OnFightTutorial)
	Choice("What has been happening out in the desert?", OnDesertExplanation)
	Choice("I have to leave.", Farewell)

func Farewell():
	Chat("在外面注意背后。")

func OnFightTutorial():
	Mes("瞄准、射击，然后捂住耳朵，因为击中它们时会发出难听的声音。")
	DisplayActions(["gp_interact", "gp_target", "gp_sit"])
	Narrate("在敌人附近按交互键自动锁定并攻击。")
	Narrate("你也可以使用目标动作手动切换目标。")
	Narrate("每次攻击消耗耐力。耐力耗尽后攻击效果会降低。")
	Narrate("坐下有助于耐力恢复。")
