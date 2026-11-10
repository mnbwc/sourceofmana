extends NpcScript

const DESERT_SEED_ID : int = ProgressCommons.Quest.DESERT_SEED

#
func OnStart():
	Narrate("一块宏伟的雕刻石，包裹着流动的魔力能量。它的存在让人感到平静。")
	Narrate("银色的细脉贯穿深灰色的石头，随着原生魔力的蓝光流过而变换色彩。")
	if GetQuest(DESERT_SEED_ID) >= ProgressCommons.DESERT_SEED.SEEK_MANAYIR:
		Narrate("你想触摸灵魂巨石吗？")
		Choice(tr("Touch the Soul Menhir."), Touch)
		Choice("别管它。")

func Touch():
	var globalScript : SoulMenhirGlobal = npc.ownScript
	if globalScript:
		globalScript.SaveRespawn(own)
