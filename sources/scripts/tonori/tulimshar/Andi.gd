extends NpcScript

#
func OnStart():
	match randi_range(0, 5):
		0: SeenClay()
		1: GoingToSea()
		2: Circles()
		3: Chat("哦，我真的很喜欢那块石头！")
		4: Chat("小心藏匿的海盗，他们很危险。")
		5: Chat("请别擦掉粉笔字！")

func SeenClay():
	Mes("你看到我的粉笔了吗？")
	Mes("我打赌是守卫拿去自己玩了。")

func GoingToSea():
	Mes("总有一天，我要去海上探索世界。")
	Mes("我听说北方有一座城市，有冰冷的白色沙子。")

func Circles():
	Mes("你会画完美的圆吗？我会！我爸爸教我的。")
	Mes("薇不会画圆。她试着画时看起来像胖方块。")
	Mes("它们看起来很好笑，但我不喜欢笑它们，所以她总是想当X。")
	Mes("我开始叫她\"狂野X\"。")
