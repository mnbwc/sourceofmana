extends NpcScript

const QUEST_ID = ProgressCommons.Quest.MINE_EXPLORATION

func OnStart():
	match GetQuest(QUEST_ID):
		ProgressCommons.MINE_EXPLORATION.DEFEATED:
			Mes("你比我以为的更强。")
			Mes("我们会再见的……")
		_:
			OnMeetXakelbael()

func OnQuit():
	super.OnQuit()
	if GetQuest(QUEST_ID) == ProgressCommons.MINE_EXPLORATION.DEFEATED:
		npc.ownScript.TriggerManaTree.call_deferred()
		npc.ownScript.RunAway.call_deferred()

func OnMeetXakelbael():
	Mes("我来这里是为了卡诺。没想到会在黑暗中遇到一棵活着的乌鲁。它是怎么活下来的？")
	Mes("这是最后一个还是还有更多？")
	Choice("What are you talking about?", OnBadMoment)
	Choice("Kano? Uru?", OnBadMoment)
	Choice("你是谁？", OnBadMoment)

func OnBadMoment():
	Mes("你们图利姆沙人来打扰这里，时机不太好。")
	Mes("我之前已经对付过你们中的一个了，但你跟得太远了，我不会对你这么宽容了。")
	Choice("What is this place?", OnNotSupposedToBeHere)
	Choice("Can you explain what's happening?", OnNotSupposedToBeHere)
	Choice("I don't even know who you are.", OnNotSupposedToBeHere)

func OnNotSupposedToBeHere():
	Mes("你不该在这里。我也不该在，但这不重要。")
	Mes("我得到了我想要的。")
	Mes("我没时间满足你的好奇心，也不能让你离开这里。")
	Action(npc.ownScript.StartFight)
