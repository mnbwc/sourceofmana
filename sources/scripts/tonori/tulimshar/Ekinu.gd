extends NpcScript

#
func OnStart():
	var questState : int = GetQuest(ProgressCommons.Quest.TUTORIAL)
	if questState < ProgressCommons.TUTORIAL.ELANORE_DONE:
		OnSendToElanore()
	elif questState < ProgressCommons.TUTORIAL.KAEL_DONE:
		OnSendToKael()
	elif questState < ProgressCommons.TUTORIAL.EKINU_DONE:
		OnKaelReport()
	else:
		OnComplete()

#
func OnSendToElanore():
	Mes("你已经醒了？")
	LookAtNpc("Elanore")
	Mes("一定要和埃拉诺谈谈，我们捡到你之后你昏迷的时候是她照顾你的。")
	ResetCamera()

func OnSendToKael():
	Mes("你看到我们要处理的那些蛆虫了吗？")
	Mes("恶心的生物。据说它们过去比现在小。")
	Mes("我们的仙人掌种植者受影响最严重。最近蛆虫数量增加对他们的庄稼造成了很大损害。")
	LookAtNpc("Kael")
	Mes("守卫凯尔驻守在东边，一个仙人掌农场旁。")
	ResetCamera()
	Mes("如果你想有所作为，报答你欠我们的恩情，你应该帮凯尔清理一些蛆虫。")

# Skip Tutorial
func OnSkipTutorial():
	Mes("你以为你要去哪？")
	Choice("I lost myself.", OnStart)
	Choice("I would like to leave this place.", OnForceHelpWithExpedition)

func OnForceHelpWithExpedition():
	Mes("嗯……我明白了。")
	Mes("为了偿还你对城市的亏欠，如果你真的想离开这里，我会让你加入城外的探险队。")
	OnExpedition()

# Kael's report
func OnKaelReport():
	Mes("你处理那些蛆虫了吗？")
	Choice("Yes, Kael said I did well. He thinks I could help with your expedition.", OnHelpWithExpedition)

func OnHelpWithExpedition():
	Mes("嗯……他说得对。")
	OnExpedition()

func OnExpedition():
	Mes("我们确实需要多一个战士，我担心把太多守卫从城里的岗位调走。")
	Mes("你要往南走，朝沙暴矿场方向。")
	Mes("它们叫那个名字是有原因的，你知道吗？沙漠那个区域容易刮大风，旋转的沙子让人很难到达。")
	Mes("但这场风暴不同。异常强烈、炎热、持久。我想知道是不是外面有什么东西在引发它……")
	Choice("Why are you heading that way?", OnWhyExpedition)
	Choice("What should I do?", OnJobExplanation)

func OnWhyExpedition():
	Mes("矿场曾经是丰富的铁矿来源，对红女王来说利润丰厚，因为整个区域是她的私人财产。")
	Mes("它们已经被遗弃几年了，因为沙暴加剧，那个区域变得更难通行，矿工们要么死于沙暴，要么死于在沙漠中游荡的愤怒生物。")
	Mes("看来我们绝望的王室现在资金不足了。她下令不惜一切代价重新开放矿场。")
	Mes("像往常一样，她根本没考虑过这对她要派去那里工作的人有什么危险。")
	Choice("What should I do?", OnJobExplanation)

func OnJobExplanation():
	Mes("你的任务是侦察该区域，确保旧的洞穴系统没有重大危险。如果一切正常，我们将开始派矿工进来，重启整个作业。")

	var tutorialState : int = GetQuest(ProgressCommons.Quest.TUTORIAL)
	if tutorialState < ProgressCommons.TUTORIAL.EKINU_DONE:
		var desertBootsID : int = DB.GetCellHash("Desert Boots")
		var desertGogglesID : int = DB.GetCellHash("Desert Goggles")
		Mes("但走之前，你不能赤脚半瞎地穿越沙漠。")
		Mes("戴上护目镜和靴子。外面沙子会烧你的脚，还会进眼睛！")
		SetQuest(ProgressCommons.Quest.TUTORIAL, ProgressCommons.TUTORIAL.EKINU_DONE)
		AddItem(desertBootsID)
		AddItem(desertGogglesID)
		AddExp(20)

		if tutorialState < ProgressCommons.TUTORIAL.KAEL_MET:
			var shortSwordID : int = DB.GetCellHash("Short Sword")
			Mes("你连武器都没有，对吧？")
			Mes("来，拿这把剑。总比空手走出去好。")
			AddItem(shortSwordID)

		Mes("穿上合适的靴子你也会比赤脚快得多。")
		Mes("如果你遇到危险，记住逃跑，死人没有尊严可言。")

		DisplayActions(["gp_run"])
		Narrate("按住奔跑键冲刺，更快地移动或逃离战斗。")
		Narrate("冲刺消耗耐力，所以需要时再用。")

		Mes("你现在应该准备好了。")
		LookAtNpc("Gilgames")
		Mes("还有一件事。离开城市之前，跟那边的吉尔伽美什说句话。他在守城门，总是愿意和新兵聊天。")
		ResetCamera()

	Mes("其他守卫已经先走了。问城门外的守卫道森，他会告诉你去哪。")
	if GetQuest(ProgressCommons.Quest.MINE_EXPLORATION) < ProgressCommons.MINE_EXPLORATION.STARTED:
		SetQuest(ProgressCommons.Quest.MINE_EXPLORATION, ProgressCommons.MINE_EXPLORATION.STARTED)

	OnMainChoice()

# Main choice loop
func OnMainChoice():
	if GetQuest(ProgressCommons.Quest.MINE_EXPLORATION) == ProgressCommons.MINE_EXPLORATION.STARTED:
		Choice("What should I do again?", OnJobExplanation)
	Choice("What can you tell me about the Red Queen?", OnRedQueen)
	Choice("What happens if I'm in danger?", OnStrangerDanger)
	Choice("Alright, let's get moving.", Farewell)

# Attack and flee
func OnStrangerDanger():
	Mes("先下手为强。")
	Mes("如果你的对手不是你的对手，赶紧逃命。死人没有尊严。")
	DisplayActions(["gp_target"])
	DisplayActions(["gp_interact"])
	DisplayActions(["gp_run"])
	Narrate("按住奔跑键冲刺，更快地移动或逃离战斗。")
	Narrate("冲刺消耗耐力，所以需要时再用。")
	OnMainChoice()

# Red Queen chain
func OnRedQueen():
	Mes("\"红女王\"，正式名称是卡罗琳娜一世女王，是这座城市无用的统治者。")
	Mes("事实上，她声称自己是托诺里的女王。好像她能控制城墙外的事情似的。")
	Mes("她声称自己是铂金王朝的后裔，数千年前图利姆沙的统治者。实际上我们都知道她父亲是个仙人掌种植者，她能当女王只是因为他是个非常聪明的人。")
	Choice("It seems that you really dislike her.", OnDislikeQueen)
	Choice("Her father was a cactus farmer?", OnRedQueenFather)

func OnRedQueenFather():
	Mes("是的。他成功说服了临终的老国王他是他的私生子。")
	Mes("老人没有其他孩子，喜欢上了他。临终前他立他为继承人。他实际上是个非常聪明的人，也是好领袖，所以宫殿里的人也就接受了。")
	Mes("现在他也死了，他那个没用的女儿在编造她所谓王室血统的故事。事实是没人喜欢她，她需要一个借口让别人继续让她戴那顶王冠。")
	Choice("It seems that you really dislike her.", OnDislikeQueen)

func OnDislikeQueen():
	Mes("那说得轻了！")
	Mes("我想看她半埋在沙子里，蝎子戳着她的尸体。")
	Mes("不过别到处说是我告诉你的。")
	Mes("她是个冷酷无情、自私自利的人。她把所有时间都花在皇家宫殿里照料她的花园。")
	Mes("她关心那个愚蠢的花园胜过关心我们任何人的生命。")
	Choice("We should get going before she gets angry then!", OnMainChoice)

# Expedition started
func Farewell():
	if randi() % 2:
		Chat("保持警惕。")
	else:
		Chat("平安回来。")

func OnComplete():
	var questState : int = GetQuest(ProgressCommons.Quest.MINE_EXPLORATION)
	if questState == ProgressCommons.MINE_EXPLORATION.STARTED:
		Mes("准备出发时向城门外的守卫道森报告。")
	else:
		Mes("你很幸运，我们在哥布林之前找到了你，但很高兴看到你好转了。")

	OnMainChoice()
