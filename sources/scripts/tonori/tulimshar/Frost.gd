extends NpcScript

#
const QUEST_ID : int = ProgressCommons.Quest.TULIMSHAR_OLD_FRIENDSHIP
var sealedLettersID : int = DB.GetCellHash("Sealed Letters")
var heavyEnvelopeID : int = DB.GetCellHash("Heavy Envelope")

#
func OnStart():
	var questState : int = GetQuest(QUEST_ID)
	match questState:
		ProgressCommons.TULIMSHAR_OLD_FRIENDSHIP.INACTIVE:
			QuestInactive()
		ProgressCommons.TULIMSHAR_OLD_FRIENDSHIP.STARTED:
			QuestStarted()
		ProgressCommons.TULIMSHAR_OLD_FRIENDSHIP.ENVELOPES_FOUND:
			QuestEnvelopesFound()
		ProgressCommons.TULIMSHAR_OLD_FRIENDSHIP.LETTERS_DELIVERED:
			QuestRewards()
		ProgressCommons.TULIMSHAR_OLD_FRIENDSHIP.REWARDS_WITHDREW:
			QuestCompleted()

func QuestInactive():
	Mes("小心脚下。仙人掌刚长出来。")
	Mes("抱歉，我不常来客人。自从我在宫殿里当顾问以来就没有了。")
	Mes("几年前辞了。上面每个人的担子都太重了，你知道吗？")
	Mes("在这里只有我和泥土。不过耕种对城市有用，所以我不介意。")
	Mes("你有没有在脑子里反复回放同一个场景，想知道如果做了不同的决定，事情会有多不一样？")
	Choice("What's on your mind?", Lore)
	Choice("I should get going", Dismiss)

func QuestStarted():
	Mes("还在走？那些走廊不太友好。")
	Choice("Tell me about Ben again", Lore)
	Choice("Where are the letters?", Directions)
	Choice("I'm on it", Dismiss)

func QuestEnvelopesFound():
	Mes("你找到了。好，好。")
	Mes("把密封的信件交给本。你刚走我就看见他走进走廊了。")
	Mes("他直觉很敏锐，或者你专注于不被发现时发出了声音。总之，他会在那里。")

func QuestRewards():
	if HasItem(heavyEnvelopeID):
		Mes("你回来了。他……")
		Mes("这个信封。很重。")
		Mes("……")
		Mes("金子。他在这里放了金子。上面写着我的名字。")
		Mes("当女王第一次把我们安排在一起时，我们各自留了一份给对方。以防情况恶化到一个人需要离开。")
		Mes("我以为我走后他会花掉这些。如果他花了，我能理解。")
		Mes("我不配拥有这个。它应该属于一个比我更好的人。")
		Mes("来。你拿着。你一个下午为我们做的比我们几年做的还多。")
		ExchangeItems([heavyEnvelopeID], [], OnEnvelopeExchanged)
	else:
		Mes("你找到本了吗？他在西墙走廊。")

func OnEnvelopeExchanged(result : ActorInventory.ExchangeResult):
	match result:
		ActorInventory.ExchangeResult.OK:
			SetQuest(QUEST_ID, ProgressCommons.TULIMSHAR_OLD_FRIENDSHIP.REWARDS_WITHDREW)
			AddGP(1000)
		ActorInventory.ExchangeResult.MISSING_ITEMS:
			Mes("等等，信封去哪了？把它带回来给我。")

func QuestCompleted():
	Mes("今年仙人掌长得不错。")
	Mes("我一直想着我应该走到那边去。到走廊。不是今天，但我决定我会去。")
	Mes("他保留了那些信。这多少说明点什么，对吧？")

func Lore():
	Mes("曾经有一个和我一起工作的人。本。聪明人。什么都能修，什么都能建。")
	Mes("不过小声点。他宁愿把墙拆了重建，也不愿解释为什么要修。")
	Mes("我，是那个跟人打交道的人。安排班次，解决分歧，维持运转。")
	Mes("我们多年来共同守护这些城墙。在女王统治下，这不容易。")
	Choice("What happened?", Conflict)
	Choice("I should get going", Dismiss)

func Conflict():
	Mes("女王的事。或者我猜她一直在发生。总是更多要求，总是按她的方式。")
	Mes("我开始定规矩。什么事都要走流程。我想如果我能把一切都组织好，也许压力就不会压垮我们。")
	Mes("本看法不同。他说我在我们已经承受的压力上又加了一层。")
	Mes("他就自己做事。自己做决定。不告诉我，也不告诉任何人。我总是事后才知道。")
	Mes("我们都在努力维护同样的城墙。但在某个时候，我们不再讨论它，而是开始为此争吵。")
	Mes("我犯了太多错误，不敢说我对所有事都是对的。但那时候，我们两个都不肯让步。")
	Mes("他叫我走。我就走了。我对自己处理方式不自豪，但我让开了。")
	Mes("那是多年前的事了。我还是比应该更多地想起它。")
	if GetQuest(QUEST_ID) == ProgressCommons.TULIMSHAR_OLD_FRIENDSHIP.INACTIVE:
		Choice("Is there anything I can do?", Quest)
		Choice("I'm sorry to hear that", Dismiss)

func Quest():
	Mes("其实……也许有。")
	Mes("在我们附近西墙走廊的尽头，过了本的巡逻区，有一个小房间，里面有个书架。")
	Mes("那个架子上有一个信封。我们互相写的信，在女王把我们分开驻扎的时候。在一切出问题之前。")
	Mes("我不知道这会不会改变什么。但也许如果他再读一遍，他会记得我们并不总是这样的。")
	Mes("走廊有巡逻。你应该待在阴影里。如果你走进光亮处，守卫会抓住你。")
	Choice("I'll do it", Accept)
	Choice("Not now", Decline)

func Directions():
	Mes("西墙走廊的尽头，在我们左边，书架上面。")
	Mes("还有别待在光亮处。守卫接到严格命令，要把未经许可的市民赶出去。")

func Accept():
	SetQuest(QUEST_ID, ProgressCommons.TULIMSHAR_OLD_FRIENDSHIP.STARTED)
	Mes("谢谢，真的。")
	Directions()

func Decline():
	Chat("不，我理解。对陌生人要求太多了。")

func Dismiss():
	Chat("好。嗯，仙人掌不会自己照顾自己。")
