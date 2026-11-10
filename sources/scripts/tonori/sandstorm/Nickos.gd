extends NpcScript

const QUEST_ID = ProgressCommons.Quest.MINE_EXPLORATION

#
func OnStart():
	var state : ProgressCommons.MINE_EXPLORATION = GetQuest(QUEST_ID) as ProgressCommons.MINE_EXPLORATION
	if state <= ProgressCommons.MINE_EXPLORATION.STARTED:
		Mes("救命。")
	elif state <= ProgressCommons.MINE_EXPLORATION.FIND_NICKOS:
		OnMeetNickos()
	elif state <= ProgressCommons.MINE_EXPLORATION.DEFEATED:
		OnRemindStranger()
	else:
		OnComplete()

func OnMeetNickos():
	Mes("你是谁？求你了，我需要你的帮助！")
	Choice("I was sent by Watchman Ryan to find you.", OnFoundNickos)

func OnFoundNickos():
	Mes("哦。我以为他会让我等死。")
	Mes("他派我一个人下来的时候，似乎真的不在乎我会出什么事。")
	Choice("What happened to you?", OnWhatHappened)

func OnWhatHappened():
	Mes("我被……谁攻击了？")
	Mes("我没想到会在下面遇到人，但有一个穿着深色长袍的奇怪男人。")
	Mes("他似乎在检查洞穴的墙壁。我试着跟他说话，但他二话不说就攻击我。")
	Choice("Did you see where he went?", OnPursueStranger)
	Choice("Are you okay?", OnCheckNickos)

func OnCheckNickos():
	Mes("我会的。我需要喘口气。")
	Mes("你身上有什么能帮上忙的东西吗？")
	var cactusPotionID : int = DB.GetCellHash("Cactus Potion")
	var pitayaID : int = DB.GetCellHash("Pitaya")
	if HasItem(cactusPotionID):
		Choice("Yes. Try this. [x1 Cactus Potion]", OnCactusPotion)
	if HasItem(pitayaID):
		Choice("Yes. Try this. [x1 Pitaya]", OnPitaya)
	Choice("No, I'm sorry. I have nothing.", OnNothingOnMe)

func OnCactusPotion():
	var id : int = DB.GetCellHash("Cactus Potion")
	if HasItem(id):
		RemoveItem(id)
		Action(OnUsedItem)
	else:
		Action(OnNothingOnMe)

func OnPitaya():
	var id : int = DB.GetCellHash("Pitaya")
	if HasItem(id):
		RemoveItem(id)
		Action(OnUsedItem)
	else:
		Action(OnNothingOnMe)

func OnUsedItem():
	Mes("谢谢！我感觉好多了。我在这里休息一会儿，然后离开这个地方。")
	Mes("谢谢你救了我！我还担心那个人会回来结果我。")
	Choice("Did you see where he went?", OnPursueStranger)

func OnNothingOnMe():
	Mes("没关系。我在这里休息一会儿，然后就能离开这个地方了。")
	Mes("我很高兴你找到我。我还担心那个人会回来结果我。")
	Choice("Did you see where he went?", OnPursueStranger)

func OnPursueStranger():
	Mes("他沿着这条走廊走开了。我最后一次听到他的声音是在这处岩架下面。")
	Mes("听起来他像在水里行走。然后我听到石头碎裂的声音。")
	Mes("我下一次听到声音时，原来是你朝我走来。")
	SetQuest(QUEST_ID, ProgressCommons.MINE_EXPLORATION.STRANGER_SPOTTED)

func OnRemindStranger():
	Mes("听起来他像在水里行走。然后我听到石头碎裂的声音。")
	Mes("这处岩架下面某处的墙上一定有裂缝。他就是从那里消失的。")

func OnComplete():
	Mes("我很高兴你找到我。我还担心那个人会回来结果我。")
