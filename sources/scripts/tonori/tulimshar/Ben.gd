extends NpcScript

#
const QUEST_ID : int = ProgressCommons.Quest.TULIMSHAR_OLD_FRIENDSHIP
var sealedLettersID : int = DB.GetCellHash("Sealed Letters")

#
func OnStart():
	var questState : int = GetQuest(QUEST_ID)
	match questState:
		ProgressCommons.TULIMSHAR_OLD_FRIENDSHIP.ENVELOPES_FOUND:
			ReceiveLetters()
		ProgressCommons.TULIMSHAR_OLD_FRIENDSHIP.LETTERS_DELIVERED, \
		ProgressCommons.TULIMSHAR_OLD_FRIENDSHIP.REWARDS_WITHDREW:
			FreeRoaming()
		_:
			TulimsharWestWallLightTrigerGlobal.CallGuard(own)

func ReceiveLetters():
	if HasItem(sealedLettersID):
		Mes("你是谁？")
		Choice("I was sent by Frost. He gave me these...", GiveLetters)
	else:
		Mes("你看起来想说什么。但你手里空空的。")
		Mes("有东西给我看的时候再来。我有活要干。")

func GiveLetters():
	Mes("什么。他派你带着这些来？")
	Think("本翻看着信件，在手中来回移动。")
	Mes("我认识这个笔迹。")
	Mes("这些很旧了。在女王把我们分开驻扎的时候，在我们共用同一堵墙之前。")
	Mes("弗罗斯特管人。我管石头。就这么分工。")
	Mes("有效果，持续了一阵子。")
	Mes("然后女王不断施压。更多要求，总是按她的方式。没有空间去好好干活。")
	Mes("弗罗斯特开始定规矩。什么事都要走流程。我告诉他，在已有压力之上不需要再加压了。")
	Mes("他说我把他排斥在外，自己做决定不告诉任何人。")
	Mes("也许是的，言语不是我的工具。从来不是。")
	Mes("我只想建造东西，让人们别争论我该怎么建。")
	Mes("总之。我叫他走。他走了。就这样。")
	Mes("……")
	Mes("但显然不是那样。因为我在这里，守着没人想攻破的城墙。")
	Mes("除了你……但我想是弗罗斯特派你来的，所以不一样。")
	ExchangeItems([sealedLettersID], [], OnLettersExchanged)

func OnLettersExchanged(result : ActorInventory.ExchangeResult):
	match result:
		ActorInventory.ExchangeResult.OK:
			SetQuest(QUEST_ID, ProgressCommons.TULIMSHAR_OLD_FRIENDSHIP.LETTERS_DELIVERED)
			Mes("把这个信封带回给他。他会知道里面是什么。")
			Mes("告诉他城墙还在。他会……嗯。就这么告诉他。")
		ActorInventory.ExchangeResult.MISSING_ITEMS:
			Mes("那些信去哪了？有了再来。")

func FreeRoaming():
	Mes("又是你。走廊现在对你开放了，我是认真的。")
	Mes("我旁边图书馆后面有一条通道。通向城墙外。没多少人知道。")
