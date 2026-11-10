extends NpcScript

# Behaviour
const ThresholdHandValue : int			= 17

# Card utilities
static var RANKS : PackedStringArray	= ["A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"]
static var SUITS : PackedStringArray	= [
	"[outline_size=2][outline_color=white][color=#635994]♠[/color][/outline_color][/outline_size]",
	"[outline_size=2][outline_color=white][color=#f8312f]♥[/color][/outline_color][/outline_size]",
	"[outline_size=2][outline_color=white][color=#f8312f]♦[/color][/outline_color][/outline_size]",
	"[outline_size=2][outline_color=white][color=#635994]♣[/color][/outline_color][/outline_size]"
]

# Game state
var deck : Array						= []
var playerHand : Array					= []
var dealerHand : Array					= []

# Helper
static func _CardName(card : int) -> String:
	return "%s of %s" % [RANKS[card % 13], SUITS[floori(card / 13.0)]]

static func _CardValue(card : int) -> int:
	var rank : int = card % 13
	if rank == 0: return 11
	if rank >= 10: return 10
	return rank + 1

static func _HandValue(hand : Array) -> int:
	var value : int = 0
	var aces : int = 0
	for card : int in hand:
		value += _CardValue(card)
		if card % 13 == 0:
			aces += 1
	while value > 21 and aces > 0:
		value -= 10
		aces -= 1
	return value

static func _HandStr(hand : Array) -> String:
	var parts : PackedStringArray = []
	for card : int in hand:
		parts.append(_CardName(card))
	return ", ".join(parts) + " (%d)" % _HandValue(hand)

static func _NewDeck() -> Array:
	var d : Array = []
	d.resize(52)
	for i : int in range(52):
		d[i] = i
	d.shuffle()
	return d

# General flow
func OnStart():
	Mes("欢迎，欢迎！你是来玩牌的吗？")
	Mes("你知道二十一点吗？我恰好是它的发明者！")
	Mes("坐下，坐下。让我看看你的本事。牌在等着呢。")
	DisplayChoices()

func DisplayChoices():
	Choice("Challenge Hyland", StartDealer)
	Choice("Find me a worthy opponent", StartPvP)
	Choice("Rules", ShowRules)
	Choice("Not today", Decline)

func ShowRules():
	Mes("规则很简单。我应该知道，是我定的！")
	Mes("尽量接近21点但不要超过。")
	Mes("数字牌按面值，人头牌是10点，A牌……嗯，A牌很灵活。可以是1点或11点。随你喜欢。")
	Mes("赢我，我就服你。超过21点就爆。这就是规则。")
	Mes("连守卫都能学会这个，说明它很简单。")
	DisplayChoices()

func Decline():
	Chat("如果你想玩牌，知道去哪找我。")

func OnQuit():
	if npc and npc.ownScript:
		npc.ownScript.call("LeavePvP", own)
	super.OnQuit()

# Player vs Dealer Mode
func StartDealer():
	deck = _NewDeck()
	playerHand = [deck.pop_back(), deck.pop_back()]
	dealerHand = [deck.pop_back(), deck.pop_back()]
	_DealerTurn()

func _DealerTurn():
	if _HandValue(playerHand) == 21:
		_DealerReveal()
	else:
		Mes("你的手牌：%s" % _HandStr(playerHand))
		Mes("我出的是：%s。" % _CardName(dealerHand[0]))
		Choice("Hit", _DealerHit)
		Choice("Stand", _DealerReveal)

func _DealerHit():
	playerHand.append(deck.pop_back())
	var value : int = _HandValue(playerHand)
	if value > 21:
		Mes("你的手牌：%s" % _HandStr(playerHand))
		Mes("嗬，爆了！贪婪就会这样。我见过一千次了。")
		Choice("Play again", StartDealer)
		Choice("离开", Decline)
	elif value == 21:
		_DealerReveal()
	else:
		_DealerTurn()

func _DealerReveal():
	while _HandValue(dealerHand) < ThresholdHandValue:
		dealerHand.append(deck.pop_back())
	var pv : int = _HandValue(playerHand)
	var dv : int = _HandValue(dealerHand)
	Mes("你的手牌：%s" % _HandStr(playerHand))
	Mes("我的手牌：%s" % _HandStr(dealerHand))
	if dv > 21:
		Mes("哼！老手也有失手的时候。打得好。")
	elif pv > dv:
		Mes("嘿……不错嘛。")
	elif dv > pv:
		Mes("经验又一次获胜。别难过，大多数人都输给我。")
	else:
		Mes("%d 平局！有种，我服你。" % pv)
	Choice("Play again", StartDealer)
	Choice("离开", Decline)

# Player vs Player Mode
func StartPvP():
	var result : int = npc.ownScript.call("JoinPvP", own)
	if result == 0:
		Mes("好吧，看看还有谁够胆。等一下。")
		Choice("Check", _PvPCheck)
		Choice("取消", _PvPCancel)
	else:
		_PvPBegin()

func _PvPCheck():
	var game : Dictionary = npc.ownScript.call("GetGame", own)
	if game.is_empty():
		Mes("耐心……好的牌手学会等待。")
		Choice("Check", _PvPCheck)
		Choice("取消", _PvPCancel)
	elif game.has("result"):
		_PvPShowResult(game)
	else:
		_PvPBegin()

func _PvPCancel():
	npc.ownScript.call("LeavePvP", own)
	Chat("不玩？真可惜。好吧，也许你会改变主意。")
func _PvPBegin():
	playerHand = npc.ownScript.call("GetHand", own)
	Mes("哈！比赛开始了！让我看看你们俩的本事。")
	_PvPTurn()

func _PvPTurn():
	Mes("你的手牌：%s" % _HandStr(playerHand))
	if _HandValue(playerHand) == 21:
		Mes("二十一点！")
		_PvPStand()
	else:
		Choice("Hit", _PvPHit)
		Choice("Stand", _PvPStand)

func _PvPHit():
	var card : int = npc.ownScript.call("DrawCard", own)
	if card < 0:
		_PvPStand()
		return
	playerHand.append(card)
	var value : int = _HandValue(playerHand)
	if value > 21:
		Mes("你的手牌：%s" % _HandStr(playerHand))
		Mes("爆了！早该听你的直觉。")
		npc.ownScript.call("FinishHand", own, value, true)
		_PvPWait()
	elif value == 21:
		Mes("你的手牌：%s" % _HandStr(playerHand))
		_PvPStand()
	else:
		_PvPTurn()

func _PvPStand():
	npc.ownScript.call("FinishHand", own, _HandValue(playerHand), false)
	_PvPWait()

func _PvPWait():
	var game : Dictionary = npc.ownScript.call("GetGame", own)
	if game.has("result"):
		_PvPShowResult(game)
	else:
		Mes("你的部分完成了。现在等你的对手……")
		Choice("Check", _PvPWaitCheck)
		Choice("Forfeit", _PvPForfeit)

func _PvPWaitCheck():
	var game : Dictionary = npc.ownScript.call("GetGame", own)
	if game.has("result"):
		_PvPShowResult(game)
	else:
		Mes("还在想……好玩家会花时间思考！")
		Choice("Check", _PvPWaitCheck)
		Choice("Forfeit", _PvPForfeit)

func _PvPShowResult(game : Dictionary):
	var isP1 : bool = game.get("p1") == own
	var myVal : int = game.get("p1Value") if isP1 else game.get("p2Value")
	var oppBust : bool = game.get("p2Busted") if isP1 else game.get("p1Busted")
	var oppHand : Array = game.get("p2Hand") if isP1 else game.get("p1Hand")
	var myBust : bool = game.get("p1Busted") if isP1 else game.get("p2Busted")
	Mes("你的分数：%d%s" % [myVal, " (Bust)" if myBust else ""])
	Mes("对手的手牌：%s%s" % [_HandStr(oppHand), " (Bust)" if oppBust else ""])
	var result : String = game.get("result", "")
	if result == "draw":
		Mes("平局！势均力敌。在我的桌上可不常见。")
	elif (result == "p1" and isP1) or (result == "p2" and not isP1):
		Mes("胜利属于你！干得好，牌今天眷顾你。")
	else:
		Mes("运气不好。牌很善变，但总会回来的。")
	npc.ownScript.call("CleanupGame", own)
	Choice("Play again", StartPvP)
	Choice("离开", Decline)

func _PvPForfeit():
	npc.ownScript.call("ForfeitPvP", own)
	Mes("中途走人？在我那个年代，那得请一轮酒。")
	Choice("Play again", StartPvP)
	Choice("离开", Decline)
