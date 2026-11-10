extends NpcScript

# Quest
const QUEST_ID : int				= ProgressCommons.Quest.TULIMSHAR_GLASSMAKING

# Recipe

const SET_INGREDIENT : int			= 4
const QUEST_SETS : int				= 3
const QUEST_INGREDIENT : int			= SET_INGREDIENT * QUEST_SETS
const QUEST_REWARD : int			= QUEST_SETS
const QUEST_EXP : int				= 100
const TRADE_MAX : int				= 20

# Quest items
var saltID : int					= DB.GetCellHash("Salt")
var boneID : int					= DB.GetCellHash("Bone")
var sulphurID : int					= DB.GetCellHash("Sulphur Powder")

# Reward items
var bottleID : int					= DB.GetCellHash("Bottle")

#
func OnStart():
	var questState : int = GetQuest(QUEST_ID)
	match questState:
		ProgressCommons.TULIMSHAR_GLASSMAKING.INACTIVE:
			OnInactive()
		ProgressCommons.TULIMSHAR_GLASSMAKING.REWARDS_WITHDREW:
			OnComplete()
		_:
			OnGathering()

# Helpers
func HasSets(setCount : int) -> bool:
	return HasItem(saltID, setCount * SET_INGREDIENT) and HasItem(boneID, setCount * SET_INGREDIENT) and HasItem(sulphurID, setCount * SET_INGREDIENT)

func GetTradeCount() -> int:
	var tradeCount : int = 0
	while tradeCount < TRADE_MAX and HasSets(tradeCount + 1):
		tradeCount += 1
	return tradeCount

func HasRoomFor(setCount : int, bottleCount : int) -> bool:
	return HasItemsSpace([[saltID, -setCount * SET_INGREDIENT], [boneID, -setCount * SET_INGREDIENT], [sulphurID, -setCount * SET_INGREDIENT], [bottleID, bottleCount]])

func TakeMaterials(setCount : int):
	RemoveItem(saltID, setCount * SET_INGREDIENT)
	RemoveItem(boneID, setCount * SET_INGREDIENT)
	RemoveItem(sulphurID, setCount * SET_INGREDIENT)

# Quest states
func OnInactive():
	Mes("嘿！你是新来的吧？我好像没见过你。我叫埃里杜。我是玻璃匠。")
	Mes("我的家族一直在做玻璃。在这种气候下，人们发现用玻璃携带液体很方便。所以我们主要做瓶子。")
	Mes("我们用托诺里的沙子做玻璃。我们还使用盐、骨头和硫磺粉。")
	Mes("到处都是沙子，你也看到了，但其他成分更难弄到。")

	QuestChoice()

func QuestChoice(previousChoice : int = -1):
	Choice("I could get those things for you.", OnAccept)
	if previousChoice != 1:
		Choice("Why those three ingredients?", OnAskRecipe)
	if previousChoice != 2:
		Choice("Where can I find these things?", OnAskWhere)
	Choice("Not right now.", OnDecline)

func OnGathering():
	if not HasSets(QUEST_SETS):
		OnKeepLooking()
		return

	if not HasRoomFor(QUEST_SETS, QUEST_REWARD):
		Mes("你好像带了很多东西。也许扔掉一些再回来。")
		return

	TakeMaterials(QUEST_SETS)
	SetQuest(QUEST_ID, ProgressCommons.TULIMSHAR_GLASSMAKING.REWARDS_WITHDREW)

	Mes("盐、骨头、硫磺。你好像什么都有！")
	Mes("我这就去做一些新瓶子。")
	Mes("同时，我用尽了最后的资源做了这些。拿着。")

	AddItem(bottleID, QUEST_REWARD)
	AddExp(QUEST_EXP)

	Mes("%d个瓶子。别打碎了，这可是上好的埃里杜玻璃！" % QUEST_REWARD)
	Think("埃里杜笑了。")
	Mes("随时给我带更多材料。每样 %d 份，我就给你吹更多玻璃。" % SET_INGREDIENT)

func OnKeepLooking():
	Mes("以我的观察还差。每样材料 %d 份。少了配比就不行。" % QUEST_INGREDIENT)
	Mes("慢慢来。炉子不会跑，我也不会。")

func OnComplete():
	Mes("又来买玻璃？见到你真好。")
	CompleteChoice()

func OnCompleteContinue():
	Mes("还需要别的吗？")
	CompleteChoice()

func CompleteChoice(previousChoice : int = -1):
	if GetTradeCount() > 0:
		Choice("I brought more material.", OnTrade)
	elif previousChoice != 0:
		Choice("What do you need for another bottle?", OnTradeReminder)
	if previousChoice != 1:
		Choice("Why those three ingredients?", OnAskRecipe)
	if previousChoice != 2:
		Choice("Where would I find them?", OnAskWhere)
	Choice("Keep well, Eridu.", OnFarewell)

# Optional dialogue
func OnAskRecipe():
	Mes("沙子就是玻璃。我们把它熔化，然后冷却得到的液体。")
	Mes("我有很多，你也看到了。")
	Mes("我们用盐降低熔化温度，这样就不会太热而损坏我们的陶罐。")
	Mes("然后，我们加入钙。就是骨头，磨成粉。这很重要，这样玻璃接触水时不会融化。")
	Mes("最后，硫磺去除气泡并消除绿色。")
	Mes("比例不对，玻璃就会脆弱。或者更糟，很丑。")
	if IsQuestCompleted(QUEST_ID):
		CompleteChoice(1)
	else:
		QuestChoice(1)

func OnAskWhere():

	Mes("史莱姆通常有高浓度的盐。有时盐晶体在它们体内形成。不比让海水蒸发容易，但可能更快。")
	Mes("硫磺更难搞。沙暴那边的火焰哥布林身上有。我觉得是它们的身体产生的。也许用来喷火？")
	Mes("骨头需要从在某些区域游荡的复活尸体上获取。我听说它们是卡厄制造的。总比挖坟好。")
	if IsQuestCompleted(QUEST_ID):
		CompleteChoice(2)
	else:
		QuestChoice(2)

func OnFarewell():
	Chat("小心碎玻璃。")

# Transitions to next states
func OnAccept():
	SetQuest(QUEST_ID, ProgressCommons.TULIMSHAR_GLASSMAKING.STARTED)

	Mes("好。你什么都齐了。")
	Mes("把它们带到这里来，等它们冷却之前我就把 %d 个瓶子交到你手上。" % QUEST_REWARD)

func OnDecline():
	Mes("随你便。反正炉子一直烧着。")

# Repeatable exchange
func OnTradeReminder():
	Mes("一个瓶子需要 %d 盐、%d 骨头和 %d 硫磺粉。两倍就能做两瓶。很简单。" % [SET_INGREDIENT, SET_INGREDIENT, SET_INGREDIENT])
	CompleteChoice(0)

func OnTrade():
	var tradeCount : int = GetTradeCount()
	if tradeCount <= 0:
		OnTradeReminder()
		return

	Mes("让我称称你带来的东西……")
	Choice("Just the one bottle.", OnExchange.bind(1))
	if tradeCount > 1:
		Choice("All of it, %d bottles." % tradeCount, OnExchange.bind(tradeCount))
	Choice("On second thought, I'll hold on to it.", OnCompleteContinue)

func OnExchange(bottleCount : int):
	if bottleCount <= 0 or not HasSets(bottleCount):
		OnTradeReminder()
		return

	if not HasRoomFor(bottleCount, bottleCount):
		Mes("你的背包快撑破了。你怎么往里面装玻璃？去腾点东西出来！")
		Action(OnCompleteContinue)
		return

	TakeMaterials(bottleCount)
	AddItem(bottleID, bottleCount)

	if bottleCount > 1:
		Mes("%d个瓶子。小心别打碎了！" % bottleCount)
	else:
		Mes("一瓶。现在别把它打碎了！")

	Action(OnCompleteContinue)
