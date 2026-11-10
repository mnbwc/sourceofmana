extends NpcScript

const questID : int = ProgressCommons.Quest.SNAKE_PIT_THIEF
const bitIndex : int = 0
var thiefsKeyID : int = DB.GetCellHash("Thief's Key")

#
func OnStart():
	var state : int = GetQuest(questID)
	if state >= ProgressCommons.SNAKE_PIT_THIEF.RIDDLE_SOLVED:
		return

	Mes("线索 1")

	var newState : int = state | (1 << bitIndex)
	if newState != state:
		SetQuest(questID, newState)

	if newState == ProgressCommons.SNAKE_PIT_THIEF.ALL_CLUES_FOUND:
		OnAllCluesFound()

func OnAllCluesFound():
	SetQuest(questID, ProgressCommons.SNAKE_PIT_THIEF.RIDDLE_SOLVED)
	AddItem(thiefsKeyID, 1)
	Mes("你在线索1下面找到一把钥匙。")
