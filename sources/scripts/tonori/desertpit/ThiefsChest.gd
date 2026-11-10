extends NpcScript

# Quest ID
const questID : int = ProgressCommons.Quest.SNAKE_PIT_THIEF

# Reward items
var scimitarID : int = DB.GetCellHash("Scimitar")

# Required items
var thiefsKeyID : int = DB.GetCellHash("Thief's Key")

#
func OnStart():
	match GetQuest(questID):
		ProgressCommons.SNAKE_PIT_THIEF.RIDDLE_SOLVED: OnTryOpen()
		ProgressCommons.SNAKE_PIT_THIEF.REWARDS_WITHDREW: OnEmpty()
		_: OnLocked()

func OnTryOpen():
	if not HasItem(thiefsKeyID):
		OnLocked()
		return

	ExchangeItems([thiefsKeyID], [scimitarID], OnChestExchanged)

func OnChestExchanged(result : ActorInventory.ExchangeResult):
	match result:
		ActorInventory.ExchangeResult.OK:
			if not IsTriggering():
				Trigger()
			SetQuest(questID, ProgressCommons.SNAKE_PIT_THIEF.REWARDS_WITHDREW)
			AddGP(200)
			AddExp(50)
			AddKarma(2)
		ActorInventory.ExchangeResult.MISSING_ITEMS:
			OnLocked()
		ActorInventory.ExchangeResult.NO_SPACE:
			Chat("你的背包太满了，无法从这个宝箱中拿取物品。")

func OnEmpty():
	Chat("这个宝箱是空的。")

func OnLocked():
	Chat("这个宝箱被锁住了。你需要一把钥匙。")
