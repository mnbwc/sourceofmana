extends NpcScript

#
func OnStart():
	match GetQuest(WaterPondGlobal.QUEST_ID):
		ProgressCommons.SNAKE_PIT_BITING_THIRST.INACTIVE:
			OnInactive()
		ProgressCommons.SNAKE_PIT_BITING_THIRST.STARTED:
			OnCheckProgress()
		ProgressCommons.SNAKE_PIT_BITING_THIRST.REWARDS_WITHDREW:
			OnComplete()
		_:
			OnInProgress()

# Quest states
func OnInactive():
	Mes("我的水全没了！我不知道现在怎么回图利姆沙。")
	Mes("几天前我决定直面沙漠，探索这片区域。")
	Mes("这里的洞穴有一个地下池塘，水非常清新纯净。")
	Mes("人们过去常来这里取水，但几年前这里开始到处都是蛇。")
	Mes("我以为我能击退那些蛇，但没想到有这么多！")
	Mes("我想靠近池塘，但蛇不断攻击我，我把收集的水全洒了。")
	Mes("你愿意试试吗？我太累了。我至少需要一满壶水才能回到城里。")
	Mes("这些不是毒蛇，但它们很有攻击性，而且太多了……")
	Choice("I can give it a try.", OnAccept)
	Choice("Not today.", OnDecline)

func OnCheckProgress():
	var rid : int = own.get_rid().get_id()
	if WaterPondGlobal.biteCounters.get(rid, 0) == 0:
		OnInProgress()
	else:
		OnDeliverWater()

func OnInProgress():
	Mes("池塘在洞穴深处。找到它的最佳方式就是迷路，你最终会看到的。")
	Mes("如果你找到了就帮我把壶装满，带回来。")
	Mes("但小心，如果被那些蛇咬了，你会损失一些水！")

func OnDeliverWater():
	WaterPondGlobal.StopJugTransport(own)
	ClearTracker()
	Mes("你成功了！还带着水！")
	Mes("来，拿这个。这是我仅剩的了。")
	SetQuest(WaterPondGlobal.QUEST_ID, ProgressCommons.SNAKE_PIT_BITING_THIRST.REWARDS_WITHDREW)
	AddItem(DB.GetCellHash("Pitaya"), 10)
	AddExp(50)
	AddGP(1000)
	AddKarma(1)
	Mes("我现在要回图利姆沙，这片区域太危险了。")
	Mes("我们会继续依靠城内更安全的水井。我想这水现在属于蛇了。")

func OnComplete():
	Mes("谢谢你帮我打水。也许现在我可以放松一点，回程不用那么赶了。")
	Mes("这些在沙漠里的日子是我最快乐的，但我想现在是时候回到文明世界了。")

func OnDecline():
	Mes("我理解。改变主意就回来。")

# Transitions
func OnAccept():
	SetQuest(WaterPondGlobal.QUEST_ID, ProgressCommons.SNAKE_PIT_BITING_THIRST.STARTED)
	Mes("给你壶。水源在洞穴深处。")
	Mes("给你一个重要提示：要找到它，我记得我先往北走再往东……")
	Mes("然后往南？还是我先往东走得更远再往北……")
	Mes("说实话，我只记得蛇和它们锋利的咬伤。幸好这种没有毒。")
	Mes("外面祝你好运，我还是建议你别一个人去。")
	Mes("如果你能找到别人帮忙，我相信击退蛇并收集更多水会更容易。")
