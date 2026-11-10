extends NpcScript

#
const FILL_TIME : float									= 10.0
const FILL_TICKS : int									= 20
const FILL_TICK_TIME : float							= FILL_TIME / FILL_TICKS
const MOVE_TOLERANCE_SQUARED : float					= 8.0 * 8.0

var BOTTLE_ID : int										= "Bottle".hash()
var WATER_BOTTLE_ID : int								= "Water Bottle".hash()

#
func OnStart():
	if not HasItem(BOTTLE_ID):
		Mes("你需要一个瓶子才能从这口井打水。")
	elif not HasItemsSpace([[BOTTLE_ID, -1], [WATER_BOTTLE_ID, 1]]):
		Mes("你的背包没有可用空间。")
	else:
		Mes("装水时别动……")
		Action(StartFill)

func StartFill():
	OnFillTick(own.position, 0)

func OnFillTick(startPos : Vector2, tick : int):
	if own.position.distance_squared_to(startPos) > MOVE_TOLERANCE_SQUARED:
		ClearTracker()
		Notification("You moved too far from the well!")
	elif not HasItem(BOTTLE_ID):
		ClearTracker()
	elif tick >= FILL_TICKS:
		CompleteFill()
	else:
		DisplayTracker("Filling...", tick, FILL_TICKS, "%")
		AddTimer(own, FILL_TICK_TIME, OnFillTick.bind(startPos, tick + 1), own.nick)

func CompleteFill():
	NpcCommons.ClearTracker(own)
	match NpcCommons.ExchangeItems(own, [BOTTLE_ID], [WATER_BOTTLE_ID]):
		ActorInventory.ExchangeResult.OK:
			PromptNext()
		ActorInventory.ExchangeResult.NO_SPACE:
			Notification("你的背包没有可用空间。")

func PromptNext():
	if HasItem(BOTTLE_ID):
		Mes("你的瓶子现在装满了清水。你还有一个空瓶。")
		Choice("Fill another bottle", StartFill)
		Choice("离开")
	else:
		Mes("你的瓶子现在装满了清水。")
