extends NpcScript

#
const QUEST_ID : int = ProgressCommons.Quest.TULIMSHAR_OLD_FRIENDSHIP
var sealedLettersID : int = DB.GetCellHash("Sealed Letters")
var heavyEnvelopeID : int = DB.GetCellHash("Heavy Envelope")

#
func OnStart():
	var questState : int = GetQuest(QUEST_ID)
	if questState != ProgressCommons.TULIMSHAR_OLD_FRIENDSHIP.STARTED:
		return

	ExchangeItems([], [sealedLettersID, heavyEnvelopeID], OnEnvelopesExchanged)

func OnEnvelopesExchanged(result : ActorInventory.ExchangeResult):
	match result:
		ActorInventory.ExchangeResult.OK:
			SetQuest(QUEST_ID, ProgressCommons.TULIMSHAR_OLD_FRIENDSHIP.ENVELOPES_FOUND)
			Mes("你在积灰的书籍间找到两个旧信封。")
			Mes("一个仔细密封着，另一个异常沉重。")
		ActorInventory.ExchangeResult.NO_SPACE:
			Mes("你发现了信封，但背包太满了拿不了。")
