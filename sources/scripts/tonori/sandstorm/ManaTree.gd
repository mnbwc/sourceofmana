extends NpcScript

const QUEST_ID : int = ProgressCommons.Quest.MINE_EXPLORATION

#
func OnStart():
	match GetQuest(QUEST_ID):
		ProgressCommons.MINE_EXPLORATION.REWARDS_WITHDREW, ProgressCommons.MINE_EXPLORATION.MANA_TREE_MET:
			OnComplete()
		ProgressCommons.MINE_EXPLORATION.DEFEATED:
			OnPostFight()
		_:
			Mes("往前走。我会保护你的安全。")

func OnPostFight():
	Mes("谢谢。")
	Mes("我本可以保护自己，就像保护你一样，但他会离开，就像他刚才做的那样。")
	Mes("我的位置已经暴露了，我需要一个盟友来对抗这个新威胁。")
	Choice("Are you... A tree?", OnTree)

func OnTree():
	Mes("是的。我是一棵乌鲁，也叫魔力树。")
	Mes("我的种族是埃米尔世界上的第一批生命，直到今天，我以为我会是最后一个。")
	Mes("你刚才打的那个人是冲我的卡诺来的。你们叫它魔力种子。")
	Mes("这是一种极其罕见的东西，因为它可以长成一棵新的乌鲁。在我们历史中的这个时刻尤其珍贵，因为我是仅存的最后一棵乌鲁。")
	Action(OnTreeBranch)

func OnTreeBranch():
	Choice("How do you survive down here?", OnHowSurvive)
	Choice("What should I do now?", OnWhatNext)
	Choice("How did you end up in this cave?", OnLocation)
	Choice("What happened since that time?", OnWhatHappened)

func OnHowSurvive():
	Mes("虽然我也想看看，但我不需要阳光也能活。")
	Mes("我的根深入大地，共享赋予我们所有人力量的生命之源：魔力。")
	Mes("只要我能与它的流动相连，我就活着。")
	Mes("很多年前，这个世界到处都是像我一样的乌鲁。我们的根相互连接，魔力比现在更强大地流动。")
	Action(OnTreeBranch)

func OnWhatHappened():
	Mes("魔力可能表现出意想不到的方式。")
	Mes("当它在一段时间内扰乱了人们的生活时，一种认为它有害的信念发展了起来。")
	Mes("这迅速传播到许多文化，砍伐乌鲁以限制魔力流动成为惯例。")
	Action(OnTreeBranch)

func OnLocation():
	Mes("当乌鲁受到威胁时，一群打算保护我们的德鲁伊在这个洞穴里种下了一颗卡诺。")
	Mes("然后，他们在附近某个地方创立了玛纳伊尔教团，研究和利用靠近我根部的魔力流动。")
	Mes("我出生在这里，对世界隐藏。随着时间推移，玛纳伊尔让他们对我存在的认知逐渐消失，甚至对自己人也一样，这样我就能继续隐藏。")
	Action(OnTreeBranch)

func OnWhatNext():
	Mes("我要你永远不要向任何人提起我的存在。")
	Mes("拿走卡诺，魔力种子。")
	Mes("图利姆沙的德鲁伊妮娜是我知道你可以信任的人。她非常善于感知魔力的流动，我已经了解了她。")
	Mes("你应该告诉在附近检查矿场的守卫你发现了巨大的危险，进入下层不安全。")
	Mes("我别无选择，只能相信你会做出正确的决定。")
	Mes("无论你做什么，都不会比你在这里对付的那个人做的更糟。")
	Choice("I will go to Nina then.", OnTellNina)
	Choice("What do you know about that man?", OnXakelbael)

func OnTellNina():
	Mes("小心。你今天在这里发现的东西已经被隐藏了数千年。")
	Mes("命运使然，你现在是一位强大魔力源的守护者。")
	SetQuest(QUEST_ID, ProgressCommons.MINE_EXPLORATION.MANA_TREE_MET)
	SetQuest(ProgressCommons.Quest.DESERT_SEED, ProgressCommons.DESERT_SEED.SEEK_NINA)

func OnXakelbael():
	Mes("他曾经是玛纳伊尔的一员。现在他的意图不同了。")
	Mes("你应该和他教团的人谈谈。他们会告诉你他是谁。")
	Choice("I will go to Nina then.", OnTellNina)

func OnComplete():
	Chat("记住你带了什么。")
