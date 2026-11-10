extends NpcScript

#
func OnStart():
	match GetQuest(WaterPondGlobal.QUEST_ID):
		ProgressCommons.SNAKE_PIT_BITING_THIRST.STARTED:
			OnFill()

func OnFill():
	var rid : int = own.get_rid().get_id()
	if not WaterPondGlobal.biteCounters.has(rid) and npc.get_node_or_null(own.nick) == null:
		(npc.ownScript as WaterPondGlobal).OnFillTick(own, own.position, 0)
		Mes("你把壶浸入池塘。静止几秒钟等它装满……")
