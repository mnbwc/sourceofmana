extends NpcScript

#
func OnStart():
	Mes("如果你沿着这条路往东走，你会离开图利姆沙山谷，朝玛纳伊尔海岸方向前进。")
	Mes("你迷路了吗？")
	DisplayChoices()

func DisplayChoices(hideChoice : int = -1):
	if hideChoice != 1:
		Choice("I am looking for the Sandstorm Mines.", OnMines)
	if hideChoice != 2:
		Choice("What can you tell me about this valley?", OnValley)
	if hideChoice != 3:
		Choice("What can you tell me about the Manayir Coast?", OnManayir)
	if hideChoice != 4:
		Choice("Thank you for your time.", Farewell)

func OnMines():
	LookAtNpc("To Mines")
	Mes("你会在正南方向找到它们。")
	ResetCamera()
	Mes("它们已经被遗弃很久了，但我刚才看到一群人往那个方向去了。")
	Mes("看起来像图利姆沙的人。不知道他们在搞什么……")
	DisplayChoices()

func OnValley():
	Mes("图利姆沙山谷是一个低洼的保护区，图利姆沙城在这里躲避开阔的托诺里沙漠。")
	Mes("城墙把城市封闭在山谷和开阔海洋之间。")
	Mes("除了吹过山谷的沙尘暴，这片区域相比托诺里更远的地区相对安全，那里各种危险都藏在滚烫的沙子下。")
	DisplayChoices()

func OnManayir():
	Mes("这是托诺里靠近大海的安静区域。它可能是这片沙漠中气候最温和的地方。")
	Mes("最显著的特征是从海滩延伸出来的半岛。")
	Mes("我们的语言中叫它纳瓦。那是玛纳伊尔教团的家。")

	Choice("Manyir Order?", OnManayirOrder)
	DisplayChoices(3)

func OnManayirOrder():
	Mes("他们是一个由智慧法师和魔力专家组成的强大组织。他们存在了很长时间，几乎和我的族人一样久。")
	Mes("我们的古老传说讲述他们如何在古代统治图利姆沙。现在他们大多独来独往。")
	Mes("我们偶尔和他们交易，我们一直与他们分享我们对灵体——魔力——的知识和崇敬。")
	DisplayChoices()
