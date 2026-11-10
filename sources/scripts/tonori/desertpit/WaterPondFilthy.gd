extends NpcScript

func OnStart():
	match GetQuest(WaterPondGlobal.QUEST_ID):
		ProgressCommons.SNAKE_PIT_BITING_THIRST.STARTED:
			Mes("这水太脏了！你不觉得这是毛罗想要说的那个池塘。")
