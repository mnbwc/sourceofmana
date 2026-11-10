extends NpcScript

# Reward items
var appleID : int					= DB.GetCellHash("Apple")

# Required items
var dorianKeyID : int				= DB.GetCellHash("Dorian's Key")
var gabrielKeyID : int				= DB.GetCellHash("Gabriel's Key")
var marvinKeyID : int				= DB.GetCellHash("Marvin's Key")

#
func OnStart():
	match GetQuest(ProgressCommons.Quest.SPLATYNA_OFFERING):
		ProgressCommons.SPLATYNA_OFFERING.INACTIVE: Inactive()
		ProgressCommons.SPLATYNA_OFFERING.STARTED: TryOpen()
		_: Empty()

func Inactive():
	Chat("这个宝箱似乎被封印了。")

func TryOpen():
	if IsMonsterAlive("Splatyna"):
		Chat("周围仍有黑暗存在。")
	else:
		# Chest is not open, try to open it
		if not IsTriggering():
			Trigger()

		# Chest is opened, you can withdraw your reward
		if HasSpace(1):
			SetQuest(ProgressCommons.Quest.SPLATYNA_OFFERING, ProgressCommons.SPLATYNA_OFFERING.REWARDS_WITHDREW)
			AddItem(appleID, 5)

func Empty():
	Chat("这个宝箱是空的。")
