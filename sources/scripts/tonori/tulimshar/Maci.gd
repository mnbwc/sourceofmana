extends NpcScript

#
func OnStart():
	if own.stat and own.stat.level < 5:
		Mes("哦，新面孔！欢迎来到图利姆沙。")
		Mes("我是 %s。我帮人在这里找到立足之地。" % npc.nick)
	else:
		Mes("想知道东西在哪？尽管问。")
	OnMainChoice()

# Main choice loop
func OnMainChoice():
	Choice("Tell me about the city.", OnCityOverview)
	Choice("How is it to live here?", OnLiveHere)
	Choice("I have to go.", Farewell)

# Answers
func OnCityOverview():
	Mes("图利姆沙坐落在我们东边和西边丘陵之间的山谷中。")
	Mes("你看到周围这堵墙把荒野沙漠挡在外面，保护我们免受南边沙暴的侵袭。")
	Mes("我们大多待在那个区域，大多数人生活和工作在中央区，分布在从这里到北边港口之间。")
	Mes("北边是你能找到任何需要的东西的最佳去处，市场、游戏，应有尽有。")
	OnMainChoice()

func OnLiveHere():
	Mes("社区非常紧密。")
	Mes("我们努力互相帮助，有互相关心的人真好。")
	Mes("警告你：这里不欢迎侮辱，先跟你说清楚。在这座城市里我们互相照应。")

	OnMainChoice()

# Farewell
func Farewell():
	Chat("继续好好干 %s。" % own.nick)
