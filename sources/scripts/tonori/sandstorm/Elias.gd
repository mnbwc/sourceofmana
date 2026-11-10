extends NpcScript

#
func OnStart():
	Mes("如果你沿着那边的西边通道走，你会离开图利姆沙山谷，进入祖尼台地。")
	Mes("你知道你要去哪吗？")
	Choice("I am looking for the Sandstorm Mines.", OnMines)
	Choice("I am looking for Tulimshar.", OnTulimshar)

func OnMines():
	LookAtNpc("To Mines")
	Mes("如果你沿着这条路走，你会离沙暴矿场越来越远。")
	ResetCamera()
	Mes("你应该往西走，朝山谷南边走。")

func OnTulimshar():
	LookAtNpc("To Tulimshar")
	Mes("你可以沿着那条路往北走到城墙。")
	ResetCamera()
	Mes("你不会错过的，从那里几乎可以看到城墙！")
