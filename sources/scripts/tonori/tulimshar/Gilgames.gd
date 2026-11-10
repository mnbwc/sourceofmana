extends NpcScript

#
func OnStart():
	var questState : int = GetQuest(ProgressCommons.Quest.TUTORIAL)
	if questState >= ProgressCommons.TUTORIAL.EKINU_DONE:
		Mes("你是埃基努新招的？好。")
		Mes("我猜你出门前有很多问题，我能帮你什么？")
		OnTutorialResume()
	else:
		Mes("离城门远点。走开。")

# Common questions
func OnTutorialResume():
	Choice("How can I improve myself?", OnStat)
	Choice("What happens if I'm lost?", OnMinimap)
	Choice("How can I easily access everything I gathered?", OnShortcut)
	Choice("All good!", Farewell)

func Farewell():
	Chat("尽量别把自己弄死了。")

# Explanations
func OnStat():
	Mes("完成任务、战斗或参与活动都能帮助你成长。四处看看，别害怕和周围的人交谈，也别害怕探索周边区域。")
	HighlightUI(UICommons.UITarget.STAT)
	Narrate("升级时你获得一些新的属性点，可以花在与你玩法匹配的属性上，并确认你的选择。")
	Narrate("力量增加你的物理伤害、移动速度和负重。")
	Narrate("敏捷影响你的攻击速度、攻击距离以及闪避能力。")
	Narrate("体力提升你的生命值上限、再生和全局防御。")
	Narrate("耐力让你的耐力值更持久、恢复更快，并让你更快再次攻击。")
	Narrate("专注为你的法力提供养分，让你能发挥技能的最佳效果。")
	HighlightUI(UICommons.UITarget.STATINDICATOR)
	Narrate("注意你的生命资源，在生命、法力或耐力低的时候冲进荒野可能不是最明智的选择。")
	HighlightUI(UICommons.UITarget.NONE)
	Mes("你需要在沙漠中面对的一切，能用的优势都要用上。")
	OnTutorialResume()

func OnMinimap():
	Mes("地图是你的朋友，不要犹豫用它来知道该去哪。")
	HighlightUI(UICommons.UITarget.MINIMAP)
	Narrate("你可以在地图上定位任何位置并向其移动，以防迷路。")
	HighlightUI(UICommons.UITarget.NONE)
	OnTutorialResume()

func OnShortcut():
	Mes("把你喜欢的物品、药水甚至技能放在身边，沙暴时可能有用。")
	HighlightUI(UICommons.UITarget.ACTION_BAR)
	Narrate("你可以将任何可用的物品、技能甚至表情拖放到快捷栏。")
	HighlightUI(UICommons.UITarget.NONE)
	OnTutorialResume()
