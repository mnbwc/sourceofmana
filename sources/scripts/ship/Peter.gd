extends NpcScript

const QUEST_ID : int = ProgressCommons.Quest.SHIP_PETER_BOUNTY
const MIN_LEVEL : int = 5

const HOLD_SPAWN_POS : Vector2 = Vector2(48 * 32, 28 * 32)
const HOLD_SPAWN_MAP : StringName = "Ship Hold"

#
func OnStart():
	if own.stat.level < MIN_LEVEL:
		Mes("你太小了不能在这里闲逛，先上学再说。")
	elif npc.ownScript and npc.ownScript.IsInProgress(own):
		Mes("你在这里干什么？活在下面。快去！")
	else:
		Mes("嗯？快点，我很忙。")
		match GetQuest(QUEST_ID):
			ProgressCommons.SHIP_PETER_BOUNTY.INACTIVE:
				OnFirstMeeting()
			ProgressCommons.SHIP_PETER_BOUNTY.ALL_DONE:
				OnDailyOffer()
			_:
				OnReturnVisit()

func OnFirstMeeting():
	Mes("有什么东西一直在啃货舱的储物。我打赌又是老鼠……我需要有人去对付它们。")
	Choice("I could use the practice, I'm in.", OnAccept)
	Choice("Not my problem.", OnDecline)

func OnAccept():
	Mes("活儿在这儿。")
	OnExplainBounties()

func OnDecline():
	Chat("随你便。")

func OnReturnVisit():
	Mes("又来了？好。货舱可不会自己清理干净。")
	OnExplainBounties()

func OnExplainBounties():
	match GetQuest(QUEST_ID):
		ProgressCommons.SHIP_PETER_BOUNTY.INACTIVE:
			Mes("老鼠找到了能啃的东西。清理掉它们，你会得到报酬。")
			Choice("Consider it done.", AcceptBounty.bind(ProgressCommons.SHIP_PETER_BOUNTY.WAVE_ONE_DONE, "Ratto", 5))
		ProgressCommons.SHIP_PETER_BOUNTY.WAVE_ONE_DONE:
			Mes("我发现了一个新巢穴，下面现在更多了，也更胆大了。这次报酬更好。")
			Choice("I'll finish what I started.", AcceptBounty.bind(ProgressCommons.SHIP_PETER_BOUNTY.WAVE_TWO_DONE, "Ratto", 10))
		ProgressCommons.SHIP_PETER_BOUNTY.WAVE_TWO_DONE:
			Mes("别管老鼠了。下面现在有蝙蝠。蝙蝠，你敢信？")
			Choice("Bats it is.", AcceptBounty.bind(ProgressCommons.SHIP_PETER_BOUNTY.ALL_DONE, "Bat", 10))

	Choice("Not right now.", OnDecline)

func AcceptBounty(nextState : int, monsterName : String, monsterCount : int):
	Mes("进去后快关门，那里的空气很恶心。")
	WarpInstance(HOLD_SPAWN_MAP.hash(), HOLD_SPAWN_POS)
	Action(npc.ownScript.StartRun.bind(own, nextState, monsterName.hash(), monsterCount))

func OnDailyOffer():
	if npc.ownScript.CanRunDailyQuest(own.get_rid().get_id()):
		Mes("又是你？好。老鼠不知道什么时候放弃，它们回来了。")
		Mes("跟往常一样。下去清理干净，上来拿钱。")
		Choice("On my way.", AcceptBounty.bind(ProgressCommons.SHIP_PETER_BOUNTY.ALL_DONE, "Ratto", 10))
		Choice("Not today.", OnDecline)
	else:
		Mes("又是你？今天货舱很安静，我不需要你帮忙了，你可以走了。")
