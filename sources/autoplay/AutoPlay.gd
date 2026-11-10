extends Node
class_name AutoPlay

# ============================================================
# AutoPlay — 自动打怪 + 自动拾取
# 解放双手，挂机无忧
# ============================================================

# --- 配置 ---
var enabled : bool = false
var autoCombat : bool = true
var autoLoot : bool = true
var autoPotion : bool = true
var potionThreshold : float = 0.3  # HP低于30%自动喝药

# 打怪范围（像素）
var combatRange : float = 300.0
# 拾取范围（像素）
var lootRange : float = 100.0
# 自动打怪间隔（秒）
var combatInterval : float = 0.5
# 自动拾取间隔（秒）
var lootInterval : float = 0.3

# --- 内部 ---
var _combatTimer : Timer
var _lootTimer : Timer
var _potionTimer : Timer
var _currentTargetRID : int = -1
var _lastMoveTime : float = 0.0
var _moveCooldown : float = 0.8  # 移动冷却，防止抖动

# --- 状态显示 ---
var statusText : String = ""
var killCount : int = 0
var lootCount : int = 0

func _ready():
	# 战斗定时器
	_combatTimer = Timer.new()
	_combatTimer.wait_time = combatInterval
	_combatTimer.one_shot = false
	_combatTimer.timeout.connect(_on_combat_tick)
	_combatTimer.name = "AutoCombatTimer"
	add_child(_combatTimer)

	# 拾取定时器
	_lootTimer = Timer.new()
	_lootTimer.wait_time = lootInterval
	_lootTimer.one_shot = false
	_lootTimer.timeout.connect(_on_loot_tick)
	_lootTimer.name = "AutoLootTimer"
	add_child(_lootTimer)

	# 喝药定时器
	_potionTimer = Timer.new()
	_potionTimer.wait_time = 1.0
	_potionTimer.one_shot = false
	_potionTimer.timeout.connect(_on_potion_tick)
	_potionTimer.name = "AutoPotionTimer"
	add_child(_potionTimer)

	# 默认关闭
	set_enabled(false)

# --- 开关 ---
func set_enabled(enable : bool):
	if enabled == enable:
		return
	enabled = enable
	if enable:
		_combatTimer.start() if autoCombat else _combatTimer.stop()
		_lootTimer.start() if autoLoot else _lootTimer.stop()
		_potionTimer.start() if autoPotion else _potionTimer.stop()
		print("[AutoPlay] 已开启")
	else:
		_combatTimer.stop()
		_lootTimer.stop()
		_potionTimer.stop()
		_currentTargetRID = -1
		print("[AutoPlay] 已关闭")
	statusText = "挂机中" if enabled else "已停止"

func toggle():
	set_enabled(not enabled)

func is_enabled() -> bool:
	return enabled

# --- 自动战斗 ---
func _on_combat_tick():
	if not enabled or not autoCombat:
		return
	if not Launcher.Player or not ActorCommons.IsAlive(Launcher.Player):
		return
	if not FSM.IsGameState():
		return

	# 如果当前目标还活着，继续打
	if _currentTargetRID != -1:
		var currentTarget = Entities.Get(_currentTargetRID)
		if currentTarget and ActorCommons.IsAlive(currentTarget):
			_attack_target(currentTarget)
			return

	# 寻找新目标
	var target = _find_nearest_monster()
	if target:
		_currentTargetRID = target.agentRID
		_attack_target(target)
	else:
		_currentTargetRID = -1

func _find_nearest_monster() -> Entity:
	var nearest : Entity = null
	var nearestDist : float = combatRange * combatRange

	for entityRID in Entities.entities:
		var entity = Entities.entities[entityRID]
		if not entity or entity == Launcher.Player:
			continue
		if entity.type != ActorCommons.Type.MONSTER:
			continue
		if not ActorCommons.IsAlive(entity):
			continue

		var dist : float = Launcher.Player.position.distance_squared_to(entity.position)
		if dist < nearestDist:
			nearestDist = dist
			nearest = entity

	return nearest

func _attack_target(target : Entity):
	if not target or not ActorCommons.IsAlive(target):
		_currentTargetRID = -1
		return

	# 距离太远就先靠近（客户端通过 MoveTo 发送移动请求）
	var dist : float = Launcher.Player.position.distance_to(target.position)
	if dist > ActorCommons.TargetMaxDistance:
		var now : float = Time.get_ticks_msec() / 1000.0
		if now - _lastMoveTime > _moveCooldown:
			_lastMoveTime = now
			Launcher.Action.MoveTo(target.position)
		return

	# 直接发攻击请求（服务器处理朝向和命中）
	var meleeSkillID : int = DB.GetCellHash(SkillCommons.SkillMeleeName)
	Network.TriggerSkill(target.agentRID, meleeSkillID)

# --- 自动拾取 ---
func _on_loot_tick():
	if not enabled or not autoLoot:
		return
	if not Launcher.Player or not ActorCommons.IsAlive(Launcher.Player):
		return
	if not FSM.IsGameState():
		return

	# 找最近的掉落物
	var nearestID : int = -1
	var nearestDist : float = lootRange * lootRange

	for dropID in Launcher.Map.drops:
		var drop = Launcher.Map.drops.get(dropID)
		if drop != null:
			var dist : float = Launcher.Player.position.distance_squared_to(drop.position)
			if dist < nearestDist:
				nearestDist = dist
				nearestID = dropID

	if nearestID != -1:
		var drop = Launcher.Map.drops.get(nearestID)
		if drop:
			# 距离够近直接捡
			if Launcher.Player.position.distance_squared_to(drop.position) < ActorCommons.PickupSquaredDistance:
				Network.PickupDrop(nearestID)
				lootCount += 1
			else:
				# 先走过去（客户端通过 MoveTo 发送移动请求）
				Launcher.Action.MoveTo(drop.position)

# --- 自动喝药 ---
func _on_potion_tick():
	if not enabled or not autoPotion:
		return
	if not Launcher.Player or not ActorCommons.IsAlive(Launcher.Player):
		return

	var hpRatio : float = float(Launcher.Player.stat.health) / float(Launcher.Player.stat.current.maxHealth)
	if hpRatio < potionThreshold:
		# 查找背包里的药水
		for item in Launcher.Player.inventory.items:
			if item:
				var cell = DB.GetItem(item.cellID, item.cellCustomfield)
				if cell and cell is ItemCell and cell.usable:
					# 检查是否有恢复HP的效果
					if _is_potion(cell):
						Network.TriggerSkill(Launcher.Player.agentRID, cell.id)
						break

func _is_potion(cell : ItemCell) -> bool:
	# 简单判断：名字包含 Potion 或 药水
	var name : String = cell.name.to_lower()
	return name.contains("potion") or name.contains("药水") or cell.slot == ActorCommons.Slot.CONSUMABLE

# --- 通知回调 ---
func _on_player_killed(targetRID : int):
	if enabled:
		killCount += 1

func get_status() -> String:
	if not enabled:
		return "已停止"
	var parts : Array[String] = []
	parts.append("挂机中")
	if autoCombat:
		parts.append("自动战斗")
	if autoLoot:
		parts.append("自动拾取")
	if autoPotion:
		parts.append("自动喝药")
	return " | ".join(parts)

func get_stats() -> String:
	return "击杀: %d | 拾取: %d" % [killCount, lootCount]
