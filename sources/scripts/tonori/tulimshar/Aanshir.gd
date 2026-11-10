extends NpcScript

#
func OnStart():
	Mes("被分配到内墙区域是最好的差事了。")
	Mes("我可以整天待在阴凉处休息，而其他守卫负责把野生动物挡在城墙外。")
	Mes("我要做的就是防止人们打扰我们女王照料花园。")
	OnMainChoice()

# Main choice loop
func OnMainChoice():
	Choice("What's this large building?", OnCastle)
	Choice("What's to the west?", OnWest)
	Choice("What's to the south?", OnSouth)
	Choice("I have to go.", Farewell)

# Answers
func OnCastle():
	Mes("那是红女王的城堡。")
	Mes("如果你有重要的事要说，可以请求觐见女王陛下，但我建议不要打扰她的花园。")
	Mes("你还可以找到大陆上第二大的图书馆，它的藏书量几乎与玛纳伊尔塔楼相当！")
	OnMainChoice()

func OnWest():
	Mes("啊——！")
	Mes("……抱歉。那个地方让我很不爽。")
	Mes("很危险。很多蛇，这还是友好的部分。有些土著人住在外面，我不想跟他们打交道，我听说他们吃生鸟，戴着羽毛跳舞。")
	Mes("如果非去不可，沿着路走。最好还是别去。")
	OnMainChoice()

func OnSouth():
	Mes("你可以在城墙外找到我们旧的仙人掌田和居住区，完全没有人居住。")
	Mes("沙暴吞没了一切，迫使我们在山谷尽头筑起路障自守。")
	Mes("这不太理想，因为那场沙暴一年比一年大。")
	OnMainChoice()
