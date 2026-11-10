extends NpcScript

# Required items
var dorianKeyID : int				= DB.GetCellHash("Dorian's Key")
var gabrielKeyID : int				= DB.GetCellHash("Gabriel's Key")
var marvinKeyID : int				= DB.GetCellHash("Marvin's Key")

var mapID : int						= "Splatyna's Chamber".hash()
const mapPosition : Vector2			= Vector2(1500, 2190)

#
func OnStart():
	match GetQuest(ProgressCommons.Quest.SPLATYNA_OFFERING):
		ProgressCommons.SPLATYNA_OFFERING.INACTIVE: Inactive()
		_: AskChoice()

func AskChoice():
	if HasItem(dorianKeyID) and HasItem(gabrielKeyID) and HasItem(marvinKeyID):
		Mes("你注意到三把不同的钥匙锁，你想怎么做？")
		Choice(tr("Open them"), TryOpen)
		Choice("离开")
	else:
		Chat("你需要三把不同的钥匙来解锁这个通道。")

func TryOpen():
	# Check and remove items to open the chest
	if GetQuest(ProgressCommons.Quest.SPLATYNA_OFFERING) != ProgressCommons.SPLATYNA_OFFERING.INACTIVE:
		if HasItem(dorianKeyID) and HasItem(gabrielKeyID) and HasItem(marvinKeyID):
			RemoveItem(dorianKeyID)
			RemoveItem(gabrielKeyID)
			RemoveItem(marvinKeyID)
			Warp(mapID, mapPosition)
			Close()
		else:
			Chat("你需要三把不同的钥匙来解锁这个通道。")

func Inactive():
	Chat("三把锁挡住了这个通道。")
