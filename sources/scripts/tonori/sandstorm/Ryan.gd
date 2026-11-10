extends NpcScript

const QUEST_ID = ProgressCommons.Quest.MINE_EXPLORATION
const DESERT_SEED_ID = ProgressCommons.Quest.DESERT_SEED

#
func OnStart():
	var state : ProgressCommons.MINE_EXPLORATION = GetQuest(QUEST_ID) as ProgressCommons.MINE_EXPLORATION
	if state <= ProgressCommons.MINE_EXPLORATION.STARTED:
		OnFirstEncounter()
	elif state <= ProgressCommons.MINE_EXPLORATION.DEFEATED:
		OnRemindObjective()
	elif state == ProgressCommons.MINE_EXPLORATION.MANA_TREE_MET:
		OnWarnAboutMines()
	else:
		OnFarewell()

#
func OnFirstEncounter():
	Mes("你是谁？")
	Choice("I was sent by Watchman Ekinu to help with scouting these mines.", OnPresentation)

func OnPresentation():
	Mes("他派你来？我告诉那个沙子脑子的白痴派新兵来。")
	Mes("不是随便哪个沙漠流浪者。")
	Mes("你大概一看到蝎子就会跑，对吧？")
	if GetBestiary("Scorpion".hash()) > 0:
		Choice("Actually, I have walked all the way here and fought off a few scorpions already.", OnHarshWelcome)
	else:
		Choice("I didn't come all this way to leave that quickly.", OnHarshWelcome)

func OnHarshWelcome():
	Mes("好……随便吧。别说了。")
	Mes("你的声音很烦人。我最不需要的就是听它在这个洞穴里回荡。")
	Mes("你也看到了，我已经有一些可怜的家伙在这里了，他们拼命地挖土，看看这鬼地方还有什么值钱的金属。")
	Choice("What am I supposed to do then?", OnExplainWhatToDo)

func OnExplainWhatToDo():
	Mes("我在做了！")
	Mes("我派了这些笨蛋中的一个去侦察前方，但他好像迷路了。")
	Mes("他叫尼科斯。他已经有一阵子没回来了。")
	Mes("去找他。")
	Mes("如果他还活着就把他带回来。如果他死了，你大概也会死，然后我们就知道下面有危险了。")
	Mes("听起来怎么样？")
	Choice("Awful, but I'll do it.", OnStartQuest)
	Choice("Okay... why did I agree to this...", OnStartQuest)

func OnStartQuest():
	Mes("好。反正我也没打算给你选择。")
	Mes("走！别等我吻别。")
	SetQuest(QUEST_ID, ProgressCommons.MINE_EXPLORATION.FIND_NICKOS)

func OnRemindObjective():
	Mes("他叫尼科斯。他已经有一阵子没回来了。")
	Mes("去找他。")
	Mes("如果他还活着就把他带回来。如果他死了，你大概也会死，然后我们就知道下面有危险了。")

func OnWarnAboutMines():
	Mes("你还活着。")
	Mes("我还以为你也迷路了。下面发生什么了？")
	Choice("The lower level is completely flooded. There is no chance of getting through.", OnLieFlooded)
	Choice("There are fire-breathing scorpions! Hundreds! They might get angry if we disturb them.", OnLieScorpions)
	Choice("There are goblin nests everywhere! We could lose a lot of people clearing them out.", OnLieGoblins)
	Choice("There was a magical tree. It talked to me!", OnLieManaTree) 

func OnLieFlooded():
	Mes("被淹了？")
	Mes("我想知道水是从哪来的。")
	Mes("就在这外面有一口井，也许附近有地下水。")
	Mes("我们不该乱闯被水淹的洞穴。隧道可能坍塌，把我们全困在里面。我会回去报告矿场已经没救了。")
	Action(OnAskAboutNickos)

func OnLieScorpions():
	Mes("喷火？我从没见过。")
	Mes("我在下面看到过能吞下一个人的大蝎子。我想它们喷火也不是不可能。")
	Mes("也许我们应该避开这个地方。反正我们似乎也找不到什么有用的东西。")
	Action(OnAskAboutNickos)

func OnLieGoblins():
	Mes("哥……哥布林？你……你确定吗？")
	Mes("我……我不害怕！如果你在怀疑的话。我不怕哥布林！")
	Mes("你知道吗？我们应该离开这个地方。对，我们走吧。")
	Mes("下面太危险了。为了寻找被遗忘的矿石冒这个险不值得。")
	Action(OnAskAboutNickos)

func OnLieManaTree():
	Mes("一棵树。一棵魔法树。一棵会说话的魔法树。")
	Mes("接下来呢，你要告诉我蝎子会喷火？哥布林给你唱歌了？")
	Mes("你疯了。下面发生的事让你变傻了！")
	Mes("也许这是一个闹鬼的洞穴。我开始觉得我们根本不该下去。")
	Action(OnAskAboutNickos)

func OnAskAboutNickos():
	Mes("说到这个。尼科斯呢？你找到他了吗？")
	Choice("He's alive. Shaken up, but resting.", OnNickosAlive)
	Choice("I found his remains.", OnNickosDead)

func OnNickosAlive():
	Mes("不奇怪，我看到过他被一只朝他爬来的蛆虫吓到。")
	Mes("他最好别在这里浪费我的时间，快点回来，不然我可能就丢下他。")
	SetQuest(QUEST_ID, ProgressCommons.MINE_EXPLORATION.REWARDS_WITHDREW)
	AddGP(1000)
	OnFarewell()

func OnNickosDead():
	Mes("……")
	Mes("好。")
	Mes("他知道风险。干这行的每个人都知道。")
	Mes("……")
	Mes("我们不会去收他的尸体。别提这个。")
	Mes("如果我再失去任何人，我就再也不能带人出城墙了。")
	Mes("他的报酬是他自己该领的。就当是给你的报告酬劳。")
	Mes("至少现在我确定我们不会下去了。我派下去的第一个人死了。不是好兆头。")
	SetQuest(QUEST_ID, ProgressCommons.MINE_EXPLORATION.REWARDS_WITHDREW)
	AddGP(2000)
	OnFarewell()

func OnFarewell():
	var responses : PackedStringArray = [
		"Good. Now stop standing here.",
		"About time. I thought you'd gotten lost too.",
		"Don't expect a medal.",
		"Go find something useful to do.",
		"You're still wasting my time.",
	]
	Chat(responses[randi() % responses.size()])
