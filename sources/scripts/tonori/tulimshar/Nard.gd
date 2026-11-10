extends NpcScript

#
func OnStart():
	Mes("欢迎上船，差不多。")
	Mes("我是拉约哈恩的船长，那艘船就停靠在我身后。")
	Mes("我们目前正在享受应得的休息，远离寒冷天气，因为过去几周我们在北边的凯泽地区。")
	MainChoices()

func MainChoices():
	Choice("Can you take me somewhere by sea?", OnSailOffer)
	Choice("Tell me about yourself.", OnAbout)
	Choice("Safe travels, Captain.", Farewell)

func OnSailOffer():
	Mes("我们很快就要驶向坎多尔，但在此期间我们可以去任何你想去的地方短途旅行。")
	Mes("我们的船坚固又快速。我们几乎走遍了埃米尔的每一处海岸。有一次甚至到了瑟尔敏！")
	Mes("当你准备走时，去拉约哈恩的舵轮那里。我们可以在那里讨论你到底想去哪。")
	Mes("我很久没摆渡人出海了。我们通常运货，不运人。")
	Choice("Where can you take me?", OnDestinations)
	Choice("Let me think about it.", MainChoices)

func OnDestinations():
	Mes("坎多尔在北边，玛纳伊尔海岸在这西边，或者一直往东北到阿蒂斯。")
	Mes("如果要去其他目的地，我们需要准备更长时间。")
	Mes("我可不想无缘无故付埃斯佩里亚的港口费！赫恩斯卡尔德也一样。")
	MainChoices()

func OnAbout():
	Mes("我从阿蒂斯港口的木匠做起。那时我没什么钱，但我活干得好。")
	Mes("我曾经还教过一些学生。")
	Mes("最终我攒够了钱买了自己的船。它几乎是个废船，但我日夜整修，直到它能再次航行。")
	Mes("我找了一些朋友：加多、朱莉娅和我的老伙计魔法arpan。我们以前都在港口一起工作过。")
	Mes("我们把船命名为拉约哈恩。以它前任船长的名字。我知道不该给船改名，但我几乎把它重建了。")
	Mes("现在我们靠往返于海岸之间运送货物为生。价格公道，我是说。")
	Choice("Do you miss Artis?", OnMissOldLife)
	Choice("Tell me about your crew.", OnCrew)
	Choice("Back to it.", MainChoices)

func OnMissOldLife():
	Mes("如果问我，这是世界上最好的城市。氛围、人、酒……")
	Mes("我确实想它，但我们可以经常回去，虽然不久。很多货物进出阿蒂斯。")
	Mes("不过我想念我的学生。也许有一天我会回去教书。谁知道呢？")
	MainChoices()

func OnCrew():
	Mes("我认识他们中的一些人很久了。加多曾经是我的大副。朱莉娅是我的厨师。")
	Mes("问题是，加多喜欢争论，他是个好厨师。朱莉娅喜欢安静……可她厨艺糟糕！所以我把他们调了个位置。")
	Mes("魔法arpan……我怎么不把他带上船呢？我们做了多年好朋友。他是个有魅力的人。真的，他能用魔法迷住你！")
	Mes("其他都是我逐渐收留的。都是我喜欢的人。船上每个人跟着我都有原因。")
	Mes("他们都很有用很重要……也许除了西尔维奥。他只是来这里喝酒和调戏朱莉娅的，那个老酒鬼！")
	MainChoices()

func Farewell():
	Chat("暂时告别了！想聊天随时回来。我喜欢有好的陪伴。")
