extends NpcScript

#
func OnStart():
	match GetQuest(ProgressCommons.Quest.SPLATYNA_OFFERING):
		ProgressCommons.SPLATYNA_OFFERING.INACTIVE: Inactive()
		ProgressCommons.SPLATYNA_OFFERING.STARTED: OnRecap()
		_: OnFinish()

func Inactive():
	Mes("赞美斯普拉蒂娜，强大的史莱姆女王！")
	Mes("你，那边！你是来给我们黏液女士献上金子的吗？")
	InfoChoice()

func InfoChoice(previousText : int = -1):
	var questState : int = GetQuest(ProgressCommons.Quest.SPLATYNA_OFFERING)
	if previousText != 0:
		if questState == ProgressCommons.SPLATYNA_OFFERING.INACTIVE:
			Choice(tr("Sure, I'll bring her the offering."), OnAcceptQuest)
		else:
			Choice(tr("What should I do?"), OnRecap)
	if previousText != 1:
		Choice(tr("This is nonsense, I'm leaving."), OnDecline)
	if previousText != 2:
		Choice(tr("Who is Splatyna?"), OnAskAboutSplatyna)
	if previousText != 3:
		Choice(tr("What is this place?"), OnAskAboutPlace)

func OnRecap():
	Mes("你还没向斯普拉蒂娜献上财宝？")
	Mes("进入她的洞穴，找到她的追随者，把金子交给她！")
	Mes("三个忠诚的握着钥匙，记住！")
	InfoChoice(0)

func OnFinish():
	Mes("等等……我听到什么。一声尖叫！")
	Mes("斯普拉蒂娜出事了？！你干了什么？")

	Choice(tr("Nothing! Everything's fine..."), OnDeny)
	Choice(tr("She's... gone."), OnAdmit)

func OnDeny():
	Mes("好，好！我会在这里确保女王开心。")

func OnAdmit():
	Mes("没了？！不，不！你撒谎！斯普拉蒂娜不会死的！")
	Mes("快……走，趁我还没发疯！")

func OnAskAboutSplatyna():
	Mes("斯普拉蒂娜和其他史莱姆不一样。她很强大！她被卡厄祝福了！")
	Mes("她曾经是一位伟大的女王。她的灵魂被放逐，来到了这里。现在她可以再次成为伟大的女王！")
	Mes("我的女王喜欢黄金。它让她的新形态如此美丽！她更喜欢那样。")
	Mes("我们必须给她要求的金子。如果我们惹怒她，她会吞噬我们所有人！")
	InfoChoice(2)

func OnAskAboutPlace():
	Mes("这是斯普拉蒂娜的黄金王国！它曾经只是一个洞穴，但现在我的女王统治这里。")
	Mes("这里的史莱姆忠于斯普拉蒂娜。它们守护着她和她巨大的宝藏。")
	Mes("它们中有三个很特别，真正忠诚的。它们有她房间的钥匙，但它们不会轻易交给你。你得去赢得！")
	Mes("但别担心。如果你带足够的金子，也许斯普拉蒂娜会让你通过……")
	InfoChoice(3)

func OnAcceptQuest():
	Mes("很好，拿这些金子去献给斯普拉蒂娜。她会判断你的价值。")
	SetQuest(ProgressCommons.Quest.SPLATYNA_OFFERING, ProgressCommons.SPLATYNA_OFFERING.STARTED)
	Farewell()

func OnDecline():
	Chat("亵渎！你竟敢拒绝斯普拉蒂娜的供品？！")
