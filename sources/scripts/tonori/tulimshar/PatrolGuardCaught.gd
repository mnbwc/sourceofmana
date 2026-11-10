extends NpcScript

#
const ENTRANCE_POS : Vector2 = Vector2(1536, 1600) # tile (48, 50)

#
func OnStart():
	Mes("站住别动。")
	Mes("bones议员的命令。未经许可不得走这些走廊。")
	Mes("我不管你来干什么。转身。")
	Action(Escort)

func Escort():
	if own.state == ActorCommons.State.TRIGGER:
		own.SetState(ActorCommons.State.TRIGGER)

	var entranceMapID : int = "Tulimshar Center".hash()
	Action(NpcCommons.Warp.bind(own, entranceMapID, ENTRANCE_POS))
