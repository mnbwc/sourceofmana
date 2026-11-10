extends NpcScript

const DESERT_SEED_ID : int = ProgressCommons.Quest.DESERT_SEED

#
func OnStart():
	var state : ProgressCommons.DESERT_SEED = GetQuest(DESERT_SEED_ID) as ProgressCommons.DESERT_SEED
	if state == ProgressCommons.DESERT_SEED.SEEK_NINA and HasItem(DB.GetCellHash("Sandstorm Kano")):
		OnDesertSeedIntro()
	elif state == ProgressCommons.DESERT_SEED.SEEK_MANAYIR:
		OnDesertSeedReminder()
	else:
		var questState : ProgressCommons.NINA_HUNGRY = GetQuest(ProgressCommons.Quest.NINA_HUNGRY) as ProgressCommons.NINA_HUNGRY
		if questState == ProgressCommons.NINA_HUNGRY.INACTIVE:
			OnIntro()
		else:
			Mes("又见面了。")
			OnPlayerChoice()

# Intro
func OnIntro():
	Mes("你好。埃拉诺派你来的？欢迎。")
	Mes("我们站在图利姆沙古老的灵魂巨石前。我是它的守护者，用它的力量保护这座城市的人民，尽我所能。")
	OnPlayerChoice()

func OnPlayerChoice():
	if GetQuest(ProgressCommons.Quest.NINA_HUNGRY) == ProgressCommons.NINA_HUNGRY.STARTED and HasItem(DB.GetCellHash("Croissant")):
		Choice("I found a croissant for you!", OnCroissantTurnIn)
	else:
		Choice("Is someone trying to stop you?", OnExplainOpposition)
		Choice("Can you tell me more about Mana and Kaore?", OnExplainMana)
		Choice("What is a Soul Menhir?", OnExplainMenhir)
	Choice("See you.", Farewell)

# Opposition and faith
func OnExplainOpposition():
	Mes("不完全是。卡韦，或德鲁伊，一直都被允许在图利姆沙。因为没有我们，这座城市就无法挺过最严重的干旱和托诺里气候的其他危险。")
	Mes("然而，王国官方信奉萨维安教义。他们和你周围一半人一样，认为魔力也是像卡厄一样的腐蚀力量，两者都应避免，除非绝对必要。")
	Mes("我们的巨石仍然矗立，因为它让人们的生活更轻松。即使信仰对立的人也不能否认魔力的好处。他们会在公开场合反对，但当个人需要出现时，他们会找像我这样的卡韦来治愈亲人或带来需要的雨水。")
	Mes("我只希望更多人公开拥抱古老传统。萨维安信仰给我们世界带来了很多麻烦。现在他们甚至催生出了这些新信徒，瓦鲁尼安人，他们基本上崇拜卡厄。")
	Mes("这一切都不像埃米尔大部分存在时期与魔力的关系。")
	Choice("What is your people's position?", OnKahwePosition)
	Choice("I have some other topics I wanted to discuss.", OnPlayerChoice)
	Choice("Thank you for your time.", Farewell)

func OnKahwePosition():
	Mes("卡韦传承着魔力曾经是我们世界和谐善力量的时代的知识。我们寻求恢复乌鲁被摧毁时失去的平衡。")
	Choice("What are the Uru?", OnExplainUru)
	Choice("I have some other topics I wanted to discuss.", OnPlayerChoice)
	Choice("Thank you for your time.", Farewell)

func OnExplainUru():
	Mes("乌鲁，也叫魔力树，曾经是世界生命力的核心，充当魔力的天然导体。")
	Mes("在古代，有很多，每一个都充当稳定力量，防止魔力衰变成卡厄。")
	Mes("魔力树在很多年前被摧毁，那是一场毁灭性的战争之后，萨维安教义在埃米尔统治精英中扎根。")
	Mes("他们决定摧毁乌鲁，尝试创造一个没有魔力的世界。")
	Choice("Where do things stand now?", OnCurrentSituation)
	Choice("I have some other topics I wanted to discuss.", OnPlayerChoice)
	Choice("Thank you for your time.", Farewell)

func OnCurrentSituation():
	Mes("取决于你问谁。有些人，比如红女王，会说没有魔法的生活更好。")
	Mes("但她在宫殿里还是用的。她只是觉得平民最好不要有，声称太危险了。")
	Mes("如果你问我，我认为魔法是我们世界的自然组成部分。它带领我们走到今天，尽管是最危险的自然力量，我们需要它。")
	Mes("仔细想想，水也是一种危险的自然力量，但没人建议关闭水井。")
	Choice("Who is the Red Queen?", OnRedQueen)
	Choice("Is there anything I can do for you?", OnAskForHelp)

func OnRedQueen():
	Mes("她是我们城市和整个托诺里王国的统治者，不过在城墙外她没有多大权力。")
	Mes("托诺里的祖尼部落从未承认我们的王国，所以她实际上只统治这座城市。")
	Mes("又一个她不让人民接触魔法的理由。祖尼人仍在修习古老的魔法，这让他们很难对付。")
	Mes("她不想在图利姆沙也遇到同样的问题。")
	Choice("What can you tell me about the Zuni?", OnZuni)
	Choice("Is there anything I can do for you?", OnAskForHelp)

func OnZuni():
	Mes("他们在托诺里沙漠生活了很长时间。事实上，他们仍在讲述这片土地还是沙漠之前的故事。")
	Mes("他们一直和图利姆沙交易，对我们大多友好，但前提是我们不把士兵派到城墙外太远。")
	Mes("过去每次发生这种情况，它们都会击退我们直到我们放弃。我尊重它们这一点，它们有明确的界限，会保卫自己的家园。")
	Choice("It would be interesting to meet them!", OnZuniMarket)
	Choice("Bah! They seem like uncivilised nomads to me.", OnZuniDismissal)

func OnZuniDismissal():
	Mes("好……")
	OnZuniMarket()

func OnZuniMarket():
	Mes("你可以在这北边的市场找到他们的一些商人。")
	Mes("我相信他们很乐意聊天，尤其是你买他们东西的话。")
	Choice("Is there anything I can do for you?", OnAskForHelp)

# Mana and Kaore
func OnExplainMana():
	Mes("魔力，通常被称为\"生命力\"，是所有生物的本质。")
	Mes("它流经世界的生物量，形成相互连接的能量网。正确引导后，可以利用它产生魔法、治愈大地、维持生命。")
	Mes("卡厄是魔力与其源头断绝后剩下的东西。")
	Mes("它不滋养生命，而是溃烂、腐化、衰败。它渗入大地，将生物扭曲成敌对、攻击性的形态。")
	Mes("卡厄在很多方面是魔力的一种形式，是我们能感知的主要能量。")
	Mes("魔力流经所有生物，但也注入一种叫齐埃尔石的特定矿物。这块灵魂巨石就是用它做的。")
	OnPlayerChoice()

# Soul Menhir and Zielite
func OnExplainMenhir():
	Mes("灵魂巨石是古老的巨石，由巨大的齐埃尔石块雕刻而成，曾经由卡韦（在这里被称为德鲁伊）竖立，用于引导辐射魔力。")
	Mes("激活时，它们产生保护光环，驱逐卡厄感染的生物并稳定土地。")
	Mes("在卡韦的帮助下，灵魂巨石可以用来治愈受伤的人，甚至在某些情况下防止死亡，给人们多次生命机会。")
	Choice("What do you know about Zielite?", OnExplainZielite)
	Choice("I have some other topics I wanted to discuss.", OnPlayerChoice)
	Choice("Thank you for your time.", Farewell)

func OnExplainZielite():
	Mes("齐埃尔石是一种稀有矿物，与魔力有天然亲和力。它可以吸收、储存和释放魔力能量，对魔法和防护都极其宝贵。")
	Mes("虽然曾经丰富，但齐埃尔石由于萨维安教义使社会反对使用魔力时发生的清洗而变得稀缺。")
	Mes("齐埃尔石工艺仍存在于隐藏的护身符、被遗忘的废墟和少数继续矗立作为对抗卡厄堡垒的灵魂巨石中。")
	OnPlayerChoice()

# Hungry quest
func OnAskForHelp():
	var questState : int = GetQuest(ProgressCommons.Quest.NINA_HUNGRY)
	if questState == ProgressCommons.NINA_HUNGRY.STARTED:
		Mes("现在不用。我还在盼着那份点心，如果你碰巧找到的话！")
		OnPlayerChoice()
	elif questState == ProgressCommons.NINA_HUNGRY.REWARDS_WITHDREW:
		Mes("现在不用，但谢谢你的好意。")
		OnPlayerChoice()
	else:
		Mes("现在不用……")
		Mes("不过我有点饿了。倒不是要你帮我带吃的！我待会儿自己解决。")
		Choice("I'll keep an eye out for snacks on the way to the market.", OnStartHungryQuest)
		Choice("Ok bye.", Farewell)

func OnStartHungryQuest():
	Mes("你真是太好了。我一般不会开口，但一个羊角面包现在简直太美妙了。")
	Mes("你通常可以在市场的面包摊找到它们。我本想自己去，但……这块巨石不会自己守护。")
	SetQuest(ProgressCommons.Quest.NINA_HUNGRY, ProgressCommons.NINA_HUNGRY.STARTED)

func OnCroissantTurnIn():
	Mes("天哪！")
	Mes("一个羊角面包！真不敢相信。你太好了，我完全没想到！")
	ExchangeItems([DB.GetCellHash("Croissant")], [[DB.GetCellHash("Cactus Potion"), 10]], OnCroissantExchanged)

func OnCroissantExchanged(result : ActorInventory.ExchangeResult):
	match result:
		ActorInventory.ExchangeResult.OK:
			SetQuest(ProgressCommons.Quest.NINA_HUNGRY, ProgressCommons.NINA_HUNGRY.REWARDS_WITHDREW)
			AddGP(100)
			Mes("请收下这些仙人掌药水和一点金子，弥补你的花费。")
		ActorInventory.ExchangeResult.MISSING_ITEMS:
			Mes("哦……去哪了？算了，也许下次吧。")
		ActorInventory.ExchangeResult.NO_SPACE:
			Mes("我想给你点什么作为回报，但你的背包满了。腾点空间再来找我。")

# Desert Seed quest
func OnDesertSeedIntro():
	Mes("你回来了！")
	Mes("探险怎么样？")
	Choice("You will not believe what happened. I found this...", OnRevealKano)

func OnRevealKano():
	Mes("我能看看吗？")
	Mes("……")
	Mes("你说得对。我不敢相信。这是一颗卡诺。魔力种子！")
	Mes("我只是在书上读到过这些。我从没想过我这辈子能看到一棵。")
	Mes("你是怎么找到这个的？")
	Choice("There is a Mana Tree at the bottom of the mines. It gave it to me.", OnRevealManaTree)

func OnRevealManaTree():
	Mes("你说什么？！你刚才说魔力树？活的？在沙暴矿场里？？？")
	Mes("我……")
	Mes("我需要一点时间。")
	Mes("很久以前我在那个方向感觉到过什么。我以为是深层齐埃尔石矿。我完全不知道。")
	Mes("你能和它交流吗？它告诉你怎么处理那个卡诺了吗？")
	Choice("It said to bring it to you and that you could be trusted.", OnNinaReacts)

func OnNinaReacts():
	Mes("它怎么认识我？这一切……很难理解。")
	Mes("我很高兴它信任我，但我们需要让玛纳伊尔知道这件事。他们是一个古老的德鲁伊教团，曾经是这些树的守护者。")
	Mes("他们在图利姆沙山谷西边的海滩尽头的一个小半岛上建了一座城堡，在风暴之外。")
	Mes("往南走，过了城墙再往西。")
	Mes("找到他们，给他们看卡诺。解释你看到的一切。")
	Mes("暂时别告诉别人这件事。这是一件非常强大的物品。肯定会引来不该有的注意。")
	Mes("走之前，带上这个。")
	ExchangeItems([], [DB.GetCellHash("Brass Zielite Amulet")], OnAmuletExchanged)

func OnAmuletExchanged(result : ActorInventory.ExchangeResult):
	match result:
		ActorInventory.ExchangeResult.OK:
			SetQuest(DESERT_SEED_ID, ProgressCommons.DESERT_SEED.SEEK_MANAYIR)
			Mes("这是一块齐埃尔石护身符。中心的石头是纯齐埃尔石的碎片，这块巨石也是同一种矿物做的。")
			Mes("佩戴时，它让你能引导魔力，并与灵魂巨石连接。")
			Choice("What does connecting to a Soul Menhir do?", OnExplainMenhirSync)
		ActorInventory.ExchangeResult.NO_SPACE:
			Mes("你的背包太满了装不下。腾点空间再来找我。")

func OnExplainMenhirSync():
	Mes("当你戴着齐埃尔石护身符触摸灵魂巨石时，石头与巨石的齐埃尔石产生共鸣。")
	Mes("这在你的灵魂和巨石之间建立了一种联系。如果你出了什么事，巨石会充当锚点。")
	Mes("你的灵魂在这里回归而不是消散。")
	Mes("这就是我们卡韦一直以来保护我们所照顾之人的方式。我应该早点给你一个。")
	Choice("[Touch the Soul Menhir]", OnTouchMenhir)

func OnTouchMenhir():
	Narrate("你把手放在灵魂巨石上。齐埃尔石护身符贴着你的皮肤变暖。")
	Narrate("一种奇怪的感觉传遍全身。像流水，但没有湿润感。石头短暂地闪烁着微弱的蓝光。")
	Action(SaveRespawnSoulMenhir)
	Mes("好。你现在连接上了。")
	Mes("如果你找到另一块灵魂巨石，触摸它。你会与最后一块你触摸过的巨石产生共鸣。")
	Mes("现在，你应该前往玛纳伊尔。我们不能带着卡诺在这里干等。")
	OnPlayerChoice()

func OnDesertSeedReminder():
	Mes("你找到玛纳伊尔了吗？")
	Mes("你应该往南走，然后往西。他们的塔楼在沙暴之外，在一个安静的半岛上。在那附近问问，应该有人能指给你正确的方向。")
	OnPlayerChoice()

func SaveRespawnSoulMenhir():
	var soulmenhirAgent : NpcAgent = GetNamedGlobalNPC("Soul Menhir")
	if soulmenhirAgent:
		var globalScript : SoulMenhirGlobal = soulmenhirAgent.ownScript
		if globalScript:
			globalScript.SaveRespawn(own)
