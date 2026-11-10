extends NpcScript

# Quest ID
const questID : int = ProgressCommons.Quest.SANDSTORM_MINE_ABANDONED_TREASURE

# Required items
var chestMineKeyID : int = DB.GetCellHash("Chest Mine Key")

# Reward items
var shortSwordID : int = DB.GetCellHash("Piou Slayer")

#
func OnStart():
	match GetQuest(questID):
		ProgressCommons.SANDSTORM_MINE_ABANDONED_TREASURE.KEY_FOUND: OnTryOpen()
		ProgressCommons.SANDSTORM_MINE_ABANDONED_TREASURE.REWARDS_WITHDREW: OnEmpty()
		_: OnLocked()

func OnTryOpen():
	if not HasItem(chestMineKeyID):
		OnLocked()
		return

	ExchangeItems([chestMineKeyID], [shortSwordID], OnChestExchanged)

func OnChestExchanged(result : ActorInventory.ExchangeResult):
	match result:
		ActorInventory.ExchangeResult.OK:
			if not IsTriggering():
				Trigger()
			SetQuest(questID, ProgressCommons.SANDSTORM_MINE_ABANDONED_TREASURE.REWARDS_WITHDREW)
		ActorInventory.ExchangeResult.MISSING_ITEMS:
			OnLocked()
		ActorInventory.ExchangeResult.NO_SPACE:
			Chat("你的背包太满了，无法从这个宝箱中拿取物品。")

func OnEmpty():
	Chat("这个宝箱是空的。")

func OnLocked():
	Chat("这个宝箱被锁住了。你需要一把钥匙。")
