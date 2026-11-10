extends NpcScript

const QUEST_ID = ProgressCommons.Quest.SANDSTORM_NAEM_HELMET

#
func OnStart():
	Mes("我爷爷曾经在这个矿场工作。我小时候他会跟我讲矿场的事。")
	Mes("它往下很深。我想知道他们想让我们走多远；我想知道下面现在还有没有活着的东西。")
	Mes("你觉得怎么样？")
	Choice("I don't know, but I'll find out.", OnWillFindOut)

func OnWillFindOut():
	if GetQuest(QUEST_ID) == ProgressCommons.SANDSTORM_NAEM_HELMET.GIVEN:
		OnFarewell()
		return

	Mes("能不走远就不走远。来，我有个多余的头盔。")
	SetQuest(QUEST_ID, ProgressCommons.SANDSTORM_NAEM_HELMET.GIVEN)
	AddItem(DB.GetCellHash("Miner Helmet"))
	Mes("图利姆沙有一个工匠用棕榈纤维和仙人掌提取的树脂做这些。很结实。最酷了。")
	Choice("The what?", OnNevermind)

func OnNevermind():
	Mes("算了。祝你好运！")

func OnFarewell():
	Chat("祝你好运！")
