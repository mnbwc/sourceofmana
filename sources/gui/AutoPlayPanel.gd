extends WindowPanel

# AutoPlay UI panel - 挂机设置面板

@onready var enableCheck : CheckButton = $Layout/Margin/VBox/EnableRow/EnableCheck
@onready var combatCheck : CheckButton = $Layout/Margin/VBox/CombatRow/CombatCheck
@onready var lootCheck : CheckButton = $Layout/Margin/VBox/LootRow/LootCheck
@onready var potionCheck : CheckButton = $Layout/Margin/VBox/PotionRow/PotionCheck
@onready var thresholdSlider : HSlider = $Layout/Margin/VBox/ThresholdRow/ThresholdSlider
@onready var thresholdLabel : Label = $Layout/Margin/VBox/ThresholdRow/ThresholdLabel
@onready var statusLabel : Label = $Layout/Margin/VBox/StatusRow/StatusLabel
@onready var statsLabel : Label = $Layout/Margin/VBox/StatsRow/StatsLabel

func _ready():
	titleText = "挂机设置"
	enableCheck.text = "开启挂机"
	combatCheck.text = "自动战斗"
	lootCheck.text = "自动拾取"
	potionCheck.text = "自动喝药"
	thresholdLabel.text = "30%"

	enableCheck.toggled.connect(_on_enable_toggled)
	combatCheck.toggled.connect(_on_combat_toggled)
	lootCheck.toggled.connect(_on_loot_toggled)
	potionCheck.toggled.connect(_on_potion_toggled)
	thresholdSlider.value_changed.connect(_on_threshold_changed)

	# 初始化状态
	combatCheck.button_pressed = AutoPlay.autoCombat
	lootCheck.button_pressed = AutoPlay.autoLoot
	potionCheck.button_pressed = AutoPlay.autoPotion
	thresholdSlider.value = AutoPlay.potionThreshold * 100.0
	_update_status()

func _process(_delta):
	_update_status()

func _update_status():
	statusLabel.text = AutoPlay.get_status()
	statsLabel.text = AutoPlay.get_stats()

func _on_enable_toggled(pressed : bool):
	AutoPlay.set_enabled(pressed)
	if pressed:
		combatCheck.button_pressed = AutoPlay.autoCombat
		lootCheck.button_pressed = AutoPlay.autoLoot
		potionCheck.button_pressed = AutoPlay.autoPotion

func _on_combat_toggled(pressed : bool):
	AutoPlay.autoCombat = pressed
	if AutoPlay.is_enabled():
		AutoPlay.set_enabled(true)  # 刷新定时器状态

func _on_loot_toggled(pressed : bool):
	AutoPlay.autoLoot = pressed
	if AutoPlay.is_enabled():
		AutoPlay.set_enabled(true)

func _on_potion_toggled(pressed : bool):
	AutoPlay.autoPotion = pressed
	if AutoPlay.is_enabled():
		AutoPlay.set_enabled(true)

func _on_threshold_changed(value : float):
	AutoPlay.potionThreshold = value / 100.0
	thresholdLabel.text = "%d%%" % int(value)
