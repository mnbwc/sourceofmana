extends NpcScript

const QUEST_ID = ProgressCommons.Quest.SANDSTORM_NATHAN_WATER

#
func OnStart():
	match GetQuest(QUEST_ID):
		ProgressCommons.SANDSTORM_NATHAN_WATER.INACTIVE:
			Mes("嗨！我是守卫内森。")
			Mes("我平时守图利姆沙港口，但今天我有幸在外面被烤成酥脆。")
			Mes("抱歉。我的错。")
			Mes("太阳快把我逼疯了。")
			Choice("Is this the entrance to the Sandstorm Mines?", OnEntrance)
		ProgressCommons.SANDSTORM_NATHAN_WATER.STARTED:
			Mes("哦是你！抱歉，这么多沙子我没认出你。")
			Choice("About that bottle of water.", OnWaitingWater)
		ProgressCommons.SANDSTORM_NATHAN_WATER.REWARDS_WITHDREW:
			OnComplete()
	Choice("Nevermind, I thought you were a sandman.", Farewell)

func Farewell():
	Express("I got sand everywhere... EVERYWHERE.")

# Default dialogue flow
func OnEntrance():
	Mes("确实如此！我明白为什么这地方被遗弃了。")
	Mes("风暴让我牙里都是沙子。我只能想象曾经在这里工作的矿工每天走这里是什么感受。")
	Mes("现在里面那些人得应付这个了。")
	Mes("我只希望能回去守港口。如果他们永久把我派驻这里，我可能就直接去喂哥布林了。")

	Choice("I'm actually supposed to join them.", OnJoinThem)

func OnJoinThem():
	Mes("真的？你不像矿工。你的镐呢？")

	Choice("I'm not here to mine, just scout the place.", OnJustScouting)

func OnJustScouting():
	Mes("啊，我明白了。")
	Mes("嗯，抱歉让你久等了。一个人站在这里很无聊。")
	Mes("来吧，进去。他们肯定在等你。")

	Choice("Thanks Nathan. Have a nice day!", OnNiceDay)
	if GetQuest(QUEST_ID) == ProgressCommons.SANDSTORM_NATHAN_WATER.INACTIVE:
		Choice("Is there anything I can do to make your job here easier?", OnJobEasier)

func OnNiceDay():
	Express("I'll have a great day when I can get back to staring at the sea! See you later.")

# Water quest
func OnJobEasier():
	Mes("嗯，既然你问了……我可以用一些额外的水！")
	Mes("这里有一口井，但已经干涸很久了。")
	Mes("我跟你直说吧。这甚至不是用来喝的。我身上带着呢。")
	Mes("我只想把它泼在脸上，让自己感觉没那么被热烤。")
	Mes("那是个愚蠢的请求吗？")

	Choice("Not at all. I understand what it's like to overheat in this land.", AcceptQuest)
	Choice("Yes, actually. I have more important things to do.", RefuseQuest)

func RefuseQuest():
	Mes("哦……我理解。没问题。当我没说。")
	Mes("我猜这就是他们把我派到这里的原因……我一定很烦人。")
	Action(Farewell)

func AcceptQuest():
	Mes("我就知道你会理解。补水很重要！我等你，朋友。")
	Think("内森心想")
	Mes("倒不是我能去哪，就算我想……")
	SetQuest(QUEST_ID, ProgressCommons.SANDSTORM_NATHAN_WATER.STARTED)
	if HasItem(DB.GetCellHash("Water Bottle")):
		Choice("I have one with me.", OnDeliverWater)
	Choice("I'll try to find you that!", Farewell)

func OnWaitingWater():
	Mes("你找到一瓶水了吗？")
	if HasItem(DB.GetCellHash("Water Bottle")):
		Choice("I have it.", OnDeliverWater)
	Choice("Not yet, I'll keep looking.", Farewell)

func OnDeliverWater():
	var waterBottleHash : int = DB.GetCellHash("Water Bottle")
	if HasItem(waterBottleHash):
		RemoveItem(waterBottleHash)
		SetQuest(QUEST_ID, ProgressCommons.SANDSTORM_NATHAN_WATER.REWARDS_WITHDREW)
		Mes("谢谢你，朋友！终于能松口气了！")
		Think("内森迅速把你带给他的水全泼在了自己身上。")
		Think("你看着你一路小心翼翼带来的水顺着他的身体流下，渗入下面的沙子。")
		AddExp(50)
		AddGP(100)
		Action(OnComplete)
	else:
		Action(Farewell)

func OnComplete():
	Mes("这正是我需要的。你无法想象。")
	Mes("现在我只需要有人给我再拿一个来……")
	Express("I'm joking. Mostly.")
