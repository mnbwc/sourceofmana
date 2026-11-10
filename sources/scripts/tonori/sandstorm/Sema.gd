extends NpcScript

#
func OnStart():
	if randi() % 2:
		Mes("这地方感觉不对劲。")
		Mes("它有一种诡异的感觉。也许只是因为它被遗弃太久了？")
		Mes("也许这就是它最初被遗弃的原因……")
	else:
		Mes("我听到更深的地方传来一些声音。我真不想去那里……")
