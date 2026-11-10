extends NpcScript

const DESERT_SEED_ID : int = ProgressCommons.Quest.DESERT_SEED

#
func OnStart():
	Mes("欢迎来到玛纳伊尔。")
	DisplayChoices()

func DisplayChoices():
	if GetQuest(DESERT_SEED_ID) == ProgressCommons.DESERT_SEED.SEEK_MANAYIR:
		Choice("I was sent by Nina of Tulimshar.", OnKano)
	Choice("What is this place?", OnAboutManayir)
	Choice("What are the Hantu?", OnHantus)
	Choice("再见。", Farewell)

func OnAboutManayir():
	Mes("我们是卡韦。一个德鲁伊教团，聚集在一起研究、教授和保护我们世界中魔力的秩序。")
	Mes("很久以前我们遍布整个地区，各自照料附近生长的汉特斯。那个时代已经过去了。")
	Mes("我们也在图利姆沙待了很长时间。试图阻止魔力变成政治问题。那个时代也过去了。")
	Mes("现在我们保存我们所知的知识，继续研究。")
	DisplayChoices()

func OnHantus():
	Mes("汉特，也叫魔力树，是我们世界中魔力的支柱。")
	Mes("它们通过魔力的流动连接所有生物。每一个都维持周围土地的活力，防止魔力衰变成卡厄。")
	Mes("当人们开始恐惧魔力，它们就被砍伐。传播很快。一些宗教和政治给了它教义义和目标，不久之后就没有了。")
	Mes("我们是它们的守护者。我们失败了。")
	Mes("我们存在的意义就是保存关于它们的知识。我们研究它们留下的魔力流动和痕迹。我们等待。")
	DisplayChoices()

func OnKano():
	Mes("你带着什么东西。")
	Mes("一颗卡诺，一颗魔力种子。")
	Mes("我只是在书上读到过这些，我们都是。我从没想过我们会再看到一棵。")
	Choice("Nina said you could be trusted.", OnNinaSentMe)

func OnNinaSentMe():
	Mes("好，她知道这是正确的选择。")
	Mes("但在那之前，我应该告诉你一件事。")
	Choice("What is it?", OnEndPrototype)

func OnEndPrototype():
	Mes("路到这里就结束了，暂时。")
	Mes("建造这个世界的人仍在为下一步努力。你已经追上他们了。")
	Mes("祝贺你走到这一步，感谢你与我们一起踏上这段旅程！")
	Mes("想待多久就待多久，请一定来拜访我们，在Discord上打个招呼。")
	Mes("在那之前，玛纳伊尔和托诺里地区供你继续探索。")
	Choice("I'll stay a while.", Farewell)
