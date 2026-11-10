extends NpcScript

#
const QUEST_ID : int = ProgressCommons.Quest.MINE_EXPLORATION

#
func OnStart():
	if not own or not own.progress:
		return

	var questState : int = own.progress.GetQuest(ProgressCommons.Quest.MINE_EXPLORATION)
	if questState < ProgressCommons.MINE_EXPLORATION.STRANGER_SPOTTED or questState == ProgressCommons.CompletedProgress:
		return

	Narrate("你注意到洞壁上有一条裂缝。微弱的蓝光似乎从另一边透出来。")
	Narrate("看起来你可以挤过去。")
	Narrate("一个平静的声音在你脑海中响起。")
	Mes("我没有感觉到第一个人经过这里时的那种黑暗存在。请上前。")
	Mes("数千年来第一次，我需要别人的帮助。")
	WarpInstance(npc.ownScript.mapName, Vector2(npc.ownScript.mapPos))
