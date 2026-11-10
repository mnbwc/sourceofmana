extends NpcScript

#
func OnStart():
	Mes("你需要什么帮助？")
	OnMainChoice()

# Main choice loop
func OnMainChoice():
	Choice("Where is the port?", OnPort)
	Choice("What's along the coast?", OnEast)
	Choice("I have to go.", Farewell)

# Answers
func OnPort():
	Mes("往北走。你会到的。")
	Mes("港口是大多数货物从托诺里以外进来的地方。")
	Mes("通常很忙，如果我是你，我不会在船员工作的时候打扰他们。")
	Mes("就我个人而言，我宁愿待在城堡附近也不愿跟那群人打交道。")
	OnMainChoice()

func OnEast():
	Mes("东边有很长的海滩。如果你喜欢沙子，你会喜欢的。不过不是每个人都喜欢沙子……")
	Mes("这东北方向有一座大灯塔。从上面可以俯瞰整个海洋。")
	Mes("很难错过，有用的地标。")
	Mes("如果你继续往东走，你会找到玛纳伊尔岛，就在海滩更远处。")

	OnMainChoice()
