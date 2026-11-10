extends NpcScript

# Requirements
var tributeItemID : int					= DB.GetCellHash("Apple")
const tributeItemCount : int			= 5
const playerLevelRequirement : int		= 1

#
func OnStart():
	if IsTriggering():
		Chat("专注于你的任务！")
	else:
		Mes("欢迎来到坎多尔之心，魔力来此消亡！")
		QuestionStart()

func QuestionStart():
	Choice("Let's get this started, Gabz.", StartFight)
	Choice("I changed my mind, I need to go.", Farewell)
	Choice("What is this place?", ExplainCave)
	Choice("Do you know where these waves of monsters come from?", ExplainWaves)

func ExplainCave():
	Mes("这就是坎多尔之心。这是一个洞穴系统，一直含有高浓度的卡厄。")
	Mes("当魔力流过埃米尔时，部分能量会衰减并沉积在特定区域。")
	Mes("这就是那些区域之一。魔力停止流动并变成卡厄的地方。")
	QuestionsCave()

func QuestionsCave():
	Choice("No more questions.", QuestionStart)
	Choice("Who are you? You seem off.", ExplainNpc)
	Choice("Why is the cave like this?", ExplainCaveOrigin)

func ExplainWaves():
	Mes("我们站着的棋盘是一个催化剂，利用这个洞穴的能量召唤各种生物。")
	Mes("它们会一波一波地来，试图消耗你。")
	Mes("如果你撑过足够多的波次，召唤就会停止，你就征服了坎多尔之心。")


func QuestionsWaves():
	Choice("No more questions.", QuestionStart)
	Choice("What happens if I fail?", ExplainFailure)
	Choice("Is there no way to cleanse the corruption?", ExplainCorruption)
	Choice("Why does Kaore make things this way?", ExplainKaore)

func ExplainNpc():
	Mes("我从没想过那个……我不知道！我想我一直都在这里？")
	Think("加兹有点困惑地看着你。")
	Mes("我是加兹。")
	QuestionsCave()

func ExplainCaveOrigin():
	Mes("没人真正知道。有人说它的隧道通向埃米尔的核心，里面全是卡厄。")
	Mes("其他人认为这块岩石的某种特性从四面八方吸引卡厄。")
	Mes("我不问自己太多问题。")
	QuestionsCave()

func ExplainFailure():
	Mes("你运气好，我在这里掌握着生死大权！")
	Mes("当你在这个洞穴中死亡时，我会用你的齐埃尔石护身符在灵魂巨石处恢复你的生命本质。")
	Mes("如果我不救你，我很快就没棋子可玩了。")
	QuestionsWaves()

func ExplainCorruption():
	Mes("高浓度的魔力或许能改变能量的平衡。")
	Mes("但我是来抵挡卡厄的，为什么不让我享受一下呢？")
	QuestionsWaves()
	
func ExplainKaore():
	Mes("卡厄可以让生物变得敌对，随时间推移，甚至变成亡灵。")
	Mes("它驱使它们攻击生物及其与魔力的联系。")
	Mes("如果你待太久，你可能也会感觉到。")
	QuestionsWaves()

func StartFight():
	var alivePlayerCount : int = AlivePlayerCount()
	if IsTriggering():
		if alivePlayerCount == 0:
			CallGlobal("OnCancel")
		else:
			Mes("战斗已经开始了。你现在不能再开一场。")
	elif alivePlayerCount > 0:
		if not HasItem(tributeItemID, tributeItemCount):
			Mes("开始前你需要给我带来5个苹果作为贡品。")
		elif own.stat.level < playerLevelRequirement:
			Mes("你不够强，打不了这场。到达 %d 级再来。" % playerLevelRequirement)
		else:
			RemoveItem(tributeItemID, tributeItemCount)
			Chat("啊，你准备好了！战斗将在10秒后开始，做好准备！")
			Action(Trigger)
