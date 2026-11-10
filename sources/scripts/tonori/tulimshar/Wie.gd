extends NpcScript

#
func OnStart():
	var globalScript : TicTacToeGlobal = (npc.ownScript as TicTacToeGlobal)
	match globalScript.startStep:
		TicTacToeGlobal.State.NONE:
			if steps.is_empty():
				match randi_range(0, 2):
					0: Mes("你想玩井字棋吗？棋盘是我自己画的！")
					1:
						Mes("我不会画圆，就画方块代替。但安迪总是取笑我。")
						Mes("想玩井字棋吗？")
					2:
						Mes("前几天我看到一群海盗在跟一些守卫玩井字棋。")
						Mes("想玩吗？")
			else:
				Mes("你想玩井字棋吗？")
			Choice("Play against you", StartPvE)
			Choice("Play against someone else", StartPvP)
			Choice("How do you play?", ShowRules)
			Choice("Maybe later", Decline)
		TicTacToeGlobal.State.X:
			if globalScript.playerX != own:
				Mes("嘿，已经有人在等了！你应该去和他们玩！")
				Choice("I'll play them!", StartPvP)
				Choice("How do you play?", ShowRules)
				Choice("Maybe later", Decline)
			else:
				Mes("还没人来……想继续等吗？")
				Choice("I'll wait a bit more", WaitPvP)
				Choice("How do you play?", ShowRules)
				Choice("I have to go", CancelPvP)
		TicTacToeGlobal.State.O:
			var isPlaying : bool = globalScript.playerX == own or globalScript.playerO == own
			if isPlaying:
				Mes("嘿，别作弊！玩你的游戏去！")
			else:
				Mes("嘘，他们在玩！不过你可以看。")
			Choice("Ok ok", OnQuit)
			Choice("How do you play?", ShowRules)
			if isPlaying:
				Choice("I have to go", CancelPvP)

func ShowRules():
	Mes("所以你选一个格子放你的标记，然后对方走。")
	Mes("三个连成一线就赢了！横着、竖着，或者斜着都行。")
	Mes("如果整个棋盘下满还没人赢，就是平局。不过那样挺无聊的。")
	OnStart()

func Decline():
	Chat("好吧……改变主意就回来！")

# Player vs NPC
func StartPvE():
	var globalScript : TicTacToeGlobal = (npc.ownScript as TicTacToeGlobal)
	if globalScript.startStep == TicTacToeGlobal.State.NONE and globalScript.StartPvE(own):
		DisplayActions(["gp_target", "gp_untarget"])
		Mes("你是X！选一个格子，快！")
	else:
		Mes("等一下，已经有人在玩了！排队等着。")

# Player vs Player
func StartPvP():
	var globalScript : TicTacToeGlobal = (npc.ownScript as TicTacToeGlobal)
	match globalScript.StartPvP(own):
		TicTacToeGlobal.State.X:
			Mes("现在我们等人出现……")
			return
		TicTacToeGlobal.State.O:
			DisplayActions(["gp_target", "gp_untarget"])
			Mes("耶，你们都在！玩得开心！")
			return
		TicTacToeGlobal.State.NONE:
			Mes("不行不行，棋盘现在被占了！")

func WaitPvP():
	if (npc.ownScript as TicTacToeGlobal).playerX == own:
		Chat("我相信很快就会有人来……大概吧！")

func CancelPvP():
	var globalScript : TicTacToeGlobal = (npc.ownScript as TicTacToeGlobal)
	if globalScript.playerX == own:
		globalScript.LeaveQueue(own)
		Chat("啊，这么快？好吧……")
