extends Control

#
@onready var nameControl : Control					= $HBoxContainer/Panel/Margin/Content/LoginContainer/Name
@onready var nameTextControl : LineEdit				= $HBoxContainer/Panel/Margin/Content/LoginContainer/Name/Container/Text
@onready var passwordControl : Control				= $HBoxContainer/Panel/Margin/Content/LoginContainer/Password
@onready var passwordLabel : Label					= $HBoxContainer/Panel/Margin/Content/LoginContainer/Password/Label
@onready var passwordTextControl : LineEdit			= $HBoxContainer/Panel/Margin/Content/LoginContainer/Password/Container/Text
@onready var confirmPasswordControl : Control		= $HBoxContainer/Panel/Margin/Content/LoginContainer/ConfirmPassword
@onready var confirmPasswordTextControl : LineEdit	= $HBoxContainer/Panel/Margin/Content/LoginContainer/ConfirmPassword/Container/Text
@onready var emailControl : Control					= $HBoxContainer/Panel/Margin/Content/LoginContainer/Email
@onready var emailTextControl : LineEdit			= $HBoxContainer/Panel/Margin/Content/LoginContainer/Email/Container/Text
@onready var resetCodeControl : Control				= $HBoxContainer/Panel/Margin/Content/LoginContainer/Code
@onready var resetCodeTextControl : LineEdit		= $HBoxContainer/Panel/Margin/Content/LoginContainer/Code/Container/Text
@onready var indicatorRow : HBoxContainer			= $HBoxContainer/Panel/Margin/Content/LoginContainer/IndicatorRow
@onready var rememberMeCheckBox : CheckBox			= $HBoxContainer/Panel/Margin/Content/LoginContainer/IndicatorRow/RememberMe
@onready var onlineIndicator : CheckBox				= $HBoxContainer/Panel/Margin/Content/LoginContainer/IndicatorRow/OnlineIndicator
@onready var panel : PanelContainer					= $HBoxContainer/Panel
@onready var content : BoxContainer					= $HBoxContainer/Panel/Margin/Content
@onready var horizontalSeparator : HSeparator		= $HBoxContainer/Panel/Margin/Content/HSeparator2
@onready var verticalSeparator : VSeparator			= $HBoxContainer/Panel/Margin/Content/VSeparator
@onready var news : Scrollable						= $HBoxContainer/Panel/Margin/Content/News
@onready var agreement : Scrollable					= $HBoxContainer/Panel/Margin/Content/Agreement
@onready var fieldControls : Array[Control]			= [nameControl, passwordControl, confirmPasswordControl, emailControl, resetCodeControl]

enum RecoveryState { NONE, REQUEST_EMAIL, ENTER_CODE }

const CompactMaxHeight : int				= 700
const TwoColumnsMinWidth : int				= 820
const TwoColumnsRatio : float				= 8.0
const HoveredFieldVariation : StringName	= &"LineEditHovered"

var nameText : String						= ""
var savedToken : String						= ""
var savedAccountName : String				= ""
var fillingFields : bool					= false
var fieldsEdited : bool						= false
var isAccountCreatorEnabled : bool			= false
var isConnectPrimary : bool					= false
var recoveryState : RecoveryState			= RecoveryState.NONE
var pendingFocusControl : Control			= null
var defaultPlaceholders : Dictionary		= {}

#
func FillWarningLabel(err : NetworkCommons.AuthError):
	if isAccountCreatorEnabled:
		FSM.EnterState(FSM.States.LOGIN_SCREEN)
		if err == NetworkCommons.AuthError.ERR_OK:
			EnableAccountCreator(false)
		else:
			EnableAccountCreator(true)
	elif recoveryState != RecoveryState.NONE:
		FSM.EnterState(FSM.States.LOGIN_SCREEN)
	else:
		if err != NetworkCommons.AuthError.ERR_OK:
			FSM.EnterState(FSM.States.LOGIN_SCREEN)

	var isWarn : bool = true
	var warn : String = ""
	match err:
		NetworkCommons.AuthError.ERR_OK:
			warn = ""
		NetworkCommons.AuthError.ERR_TOKEN:
			warn = "令牌无效，请重新输入密码"
			passwordTextControl.clear()
			ClearSavedToken()
			RequestFocus(passwordTextControl)
		NetworkCommons.AuthError.ERR_DUPLICATE_CONNECTION:
			warn = "该账号已在其他地方登录。"
		NetworkCommons.AuthError.ERR_AUTH:
			warn = "账号名或密码错误。"
			RequestFocus(passwordTextControl)
		NetworkCommons.AuthError.ERR_PASSWORD_VALID:
			warn = "密码不能包含制表符或换行符等特殊字符。"
			RequestFocus(passwordTextControl)
		NetworkCommons.AuthError.ERR_PASSWORD_SIZE:
			warn = "密码长度必须在 %d 到 %d 个字符之间。" % [NetworkCommons.PasswordMinSize, NetworkCommons.PasswordMaxSize]
			RequestFocus(passwordTextControl)
		NetworkCommons.AuthError.ERR_NAME_AVAILABLE:
			warn = "该账号名已被占用。"
			RequestFocus(nameTextControl)
		NetworkCommons.AuthError.ERR_NAME_VALID:
			warn = "账号名只能包含字母、数字和符号。"
			RequestFocus(nameTextControl)
		NetworkCommons.AuthError.ERR_NAME_SIZE:
			warn = "账号名长度必须在 %d 到 %d 个字符之间。" % [NetworkCommons.PlayerNameMinSize, NetworkCommons.PlayerNameMaxSize]
			RequestFocus(nameTextControl)
		NetworkCommons.AuthError.ERR_PASSWORD_MISMATCH:
			warn = "两次输入的密码不一致。"
			RequestFocus(confirmPasswordTextControl)
		NetworkCommons.AuthError.ERR_EMAIL_VALID:
			warn = "邮箱格式不正确，请输入有效的邮箱地址。"
			RequestFocus(emailTextControl)
		NetworkCommons.AuthError.ERR_RESET_UNAVAILABLE:
			warn = "本服务器不支持密码重置功能。"
			SetRecoveryState(RecoveryState.NONE)
		NetworkCommons.AuthError.ERR_RESET_EMAIL_SENT:
			warn = "如果该邮箱已注册，验证码已发送，请检查收件箱。"
			isWarn = false
			SetRecoveryState(RecoveryState.ENTER_CODE)
		NetworkCommons.AuthError.ERR_RESET_INVALID_CODE:
			warn = "验证码无效或已过期。"
			RequestFocus(resetCodeTextControl)
		NetworkCommons.AuthError.ERR_RESET_PASSWORD_UPDATED:
			warn = "密码重置成功，现在可以登录了。"
			isWarn = false
			SetRecoveryState(RecoveryState.NONE)
		_:
			warn = "无法连接到服务器（错误 %d）。\n请稍后重试或联系管理员。\n也可以尝试离线模式！" % [err]

	var textColor : Color = UICommons.WarnTextColor if isWarn else UICommons.TextColor

	if not warn.is_empty():
		warn = "[color=#%s]%s[/color]" % [textColor.to_html(false), warn]
	Launcher.GUI.notificationLabel.AddNotification(warn)

func RequestFocus(control : Control):
	pendingFocusControl = control
	if control.is_visible_in_tree():
		ApplyFocus.call_deferred()

func ApplyFocus():
	if pendingFocusControl:
		pendingFocusControl.release_focus()
		pendingFocusControl.grab_focus()
		pendingFocusControl = null

func RefreshControls():
	var isRecovering : bool = recoveryState != RecoveryState.NONE
	var enteringCode : bool = recoveryState == RecoveryState.ENTER_CODE

	nameControl.set_visible(not enteringCode)
	passwordControl.set_visible(recoveryState != RecoveryState.REQUEST_EMAIL)
	confirmPasswordControl.set_visible(isAccountCreatorEnabled or enteringCode)
	emailControl.set_visible(isAccountCreatorEnabled)
	resetCodeControl.set_visible(enteringCode)
	indicatorRow.set_visible(not isAccountCreatorEnabled and not isRecovering)
	news.set_visible(not isAccountCreatorEnabled and not isRecovering)
	agreement.set_visible(isAccountCreatorEnabled)

	passwordLabel.text = "新密码" if enteringCode else "密码"
	passwordTextControl.secret = true

	SetPanelExpand(not isRecovering)
	RefreshFocusNodes(isAccountCreatorEnabled)
	RefreshLayout()
	EnableButtons(true)

func SetRecoveryState(state : RecoveryState):
	recoveryState = state
	isAccountCreatorEnabled = false
	RefreshControls()

	match recoveryState:
		RecoveryState.NONE:
			passwordTextControl.clear()
			confirmPasswordTextControl.clear()
			resetCodeTextControl.clear()
		RecoveryState.ENTER_CODE:
			passwordTextControl.clear()
			confirmPasswordTextControl.clear()
	FocusDefault()

func EnableAccountCreator(enable : bool):
	recoveryState = RecoveryState.NONE
	isAccountCreatorEnabled = enable
	RefreshControls()

	if not enable:
		confirmPasswordTextControl.clear()
	FocusDefault()

func RefreshLayout():
	var viewportSize : Vector2 = get_viewport_rect().size
	var sidePanel : Scrollable = agreement if isAccountCreatorEnabled else news
	var isShort : bool = viewportSize.y < CompactMaxHeight
	var isSqueezed : bool = isShort and sidePanel.is_visible()
	var twoColumns : bool = isSqueezed and viewportSize.x >= TwoColumnsMinWidth

	content.vertical = not twoColumns
	panel.size_flags_stretch_ratio = TwoColumnsRatio if twoColumns else 1.0
	horizontalSeparator.set_visible(sidePanel.is_visible() and not isShort)
	verticalSeparator.set_visible(twoColumns)
	SetCompactFields(isSqueezed and not twoColumns)

func SetCompactFields(compact : bool):
	for control : Control in fieldControls:
		var label : Label = control.get_node("Label")
		var text : LineEdit = control.get_node("Container/Text")
		if not defaultPlaceholders.has(control):
			defaultPlaceholders[control] = text.placeholder_text
		label.set_visible(not compact)
		text.placeholder_text = label.text if compact else defaultPlaceholders[control]

func SetFieldHovered(text : LineEdit, hovered : bool):
	text.set_theme_type_variation(HoveredFieldVariation if hovered else &"")

func SetPanelExpand(expand : bool):
	if expand:
		panel.size_flags_vertical = Control.SIZE_FILL
	else:
		panel.size_flags_vertical = Control.SIZE_SHRINK_CENTER

#
func GetDefaultFocusFields() -> Array[LineEdit]:
	match recoveryState:
		RecoveryState.REQUEST_EMAIL:
			return [nameTextControl]
		RecoveryState.ENTER_CODE:
			return [resetCodeTextControl, passwordTextControl, confirmPasswordTextControl]
	if isAccountCreatorEnabled:
		return [nameTextControl, passwordTextControl, confirmPasswordTextControl, emailTextControl]
	if IsReturningPlayer():
		return [nameTextControl, passwordTextControl]
	return []

func FocusDefault():
	ApplyDefaultFocus.call_deferred()

func ApplyDefaultFocus():
	if not is_visible_in_tree():
		return

	for field in GetDefaultFocusFields():
		if field.is_visible_in_tree() and field.get_text().is_empty():
			field.grab_focus()
			return

	var focusOwner : Control = get_viewport().gui_get_focus_owner()
	if focusOwner is LineEdit:
		focusOwner.release_focus()
	if Launcher.GUI and Launcher.GUI.buttonBoxes:
		Launcher.GUI.buttonBoxes.Focus(UICommons.ButtonBox.PRIMARY)

func RefreshFocusNodes(accountCreatorEnabled : bool):
	if accountCreatorEnabled:
		nameTextControl.set_focus_previous(emailTextControl.get_path())
		passwordTextControl.set_focus_next(confirmPasswordTextControl.get_path())
		confirmPasswordTextControl.set_focus_next(emailTextControl.get_path())
		confirmPasswordTextControl.set_focus_previous(passwordTextControl.get_path())
	else:
		nameTextControl.set_focus_previous(passwordTextControl.get_path())
		passwordTextControl.set_focus_next(nameTextControl.get_path())

func RefreshOnlineMode():
	OnlineMode(Network.Client != null, Network.ENetServer != null or Network.WebSocketServer != null)

func OnlineMode(_clientStarted : bool, serverStarted : bool):
	if onlineIndicator:
		Launcher.GUI.buttonBoxes.Rename(UICommons.ButtonBox.TERTIARY, "切换到在线" if serverStarted else "切换到离线")
		onlineIndicator.text = "离线模式中" if serverStarted else "在线游戏中"
		if onlineIndicator.button_pressed != not serverStarted:
			onlineIndicator.button_pressed = not serverStarted

func EnableButtons(state : bool):
	if Launcher.GUI and Launcher.GUI.buttonBoxes:
		Launcher.GUI.buttonBoxes.ClearAll()
		if state:
			if recoveryState == RecoveryState.REQUEST_EMAIL:
				Launcher.GUI.buttonBoxes.Bind(UICommons.ButtonBox.PRIMARY, "发送验证码", RequestReset)
				Launcher.GUI.buttonBoxes.Bind(UICommons.ButtonBox.CANCEL, "取消", SetRecoveryState.bind(RecoveryState.NONE))
			elif recoveryState == RecoveryState.ENTER_CODE:
				Launcher.GUI.buttonBoxes.Bind(UICommons.ButtonBox.PRIMARY, "重置密码", ConfirmReset)
				Launcher.GUI.buttonBoxes.Bind(UICommons.ButtonBox.CANCEL, "取消", SetRecoveryState.bind(RecoveryState.NONE))
			elif isAccountCreatorEnabled:
				Launcher.GUI.buttonBoxes.Bind(UICommons.ButtonBox.PRIMARY, "创建账号", CreateAccount)
				Launcher.GUI.buttonBoxes.Bind(UICommons.ButtonBox.CANCEL, "取消", EnableAccountCreator.bind(false))
			else:
				isConnectPrimary = IsConnectPreferred()
				if isConnectPrimary:
					Launcher.GUI.buttonBoxes.Bind(UICommons.ButtonBox.PRIMARY, "登录", Connect)
					Launcher.GUI.buttonBoxes.Bind(UICommons.ButtonBox.SECONDARY, "创建账号", EnableAccountCreator.bind(true))
				else:
					Launcher.GUI.buttonBoxes.Bind(UICommons.ButtonBox.PRIMARY, "创建账号", EnableAccountCreator.bind(true))
					Launcher.GUI.buttonBoxes.Bind(UICommons.ButtonBox.SECONDARY, "登录", Connect)
				# Sync the indicator before binding the toggle: SwitchOnlineMode used to
				# receive button_pressed captured here, so a stale value (the scene default
				# on the online web client) made the first click a no-op.
				RefreshOnlineMode()
				Launcher.GUI.buttonBoxes.Bind(UICommons.ButtonBox.TERTIARY, "切换在线", SwitchOnlineMode)
				Launcher.GUI.buttonBoxes.Bind(UICommons.ButtonBox.CANCEL, "忘记密码", SetRecoveryState.bind(RecoveryState.REQUEST_EMAIL))
			RefreshServerButtons()
		else:
			onlineIndicator.text = "连接中..."

# Every action behind these buttons needs the server, so they stay greyed out until the client is connected
func RefreshServerButtons():
	if not Launcher.GUI or not Launcher.GUI.buttonBoxes:
		return

	var reachable : bool = Network.IsServerReachable()
	Launcher.GUI.buttonBoxes.Enable(UICommons.ButtonBox.PRIMARY, reachable)
	if recoveryState == RecoveryState.NONE and not isAccountCreatorEnabled:
		Launcher.GUI.buttonBoxes.Enable(UICommons.ButtonBox.SECONDARY, reachable)
		Launcher.GUI.buttonBoxes.Enable(UICommons.ButtonBox.CANCEL, reachable)
		if not reachable and onlineIndicator:
			onlineIndicator.text = "连接中..."

	if reachable:
		Launcher.GUI.buttonBoxes.Suggest(UICommons.ButtonBox.PRIMARY)
	else:
		Launcher.GUI.buttonBoxes.ClearSuggest()

func SaveAccountName():
	if Launcher.GUI.settingsWindow:
		Launcher.GUI.settingsWindow.set_sessionaccountname(nameText)

func IsReturningPlayer() -> bool:
	return not savedToken.is_empty() or not Conf.GetUserValue("Session-AccountName", "").is_empty()

func IsConnectPreferred() -> bool:
	return IsReturningPlayer() or not nameTextControl.get_text().is_empty()

func RefreshOnce():
	EnableAccountCreator(isAccountCreatorEnabled)
	_on_visibility_changed()

# Token persistence
func SaveToken(accountName : String, token : String):
	if not rememberMeCheckBox.button_pressed:
		return
	Conf.SetValue("auth", "account_name", Conf.Type.AUTH_TOKEN, accountName)
	Conf.SetValue("auth", "token", Conf.Type.AUTH_TOKEN, token)
	Conf.SaveType("auth_token", Conf.Type.AUTH_TOKEN)

func LoadSavedToken() -> bool:
	savedAccountName = Conf.GetString("auth", "account_name", Conf.Type.AUTH_TOKEN)
	savedToken = Conf.GetString("auth", "token", Conf.Type.AUTH_TOKEN)
	if savedAccountName.is_empty() or savedToken.is_empty():
		return false
	nameText = savedAccountName
	return true

func ClearSavedToken():
	passwordTextControl.clear()
	savedToken = ""
	savedAccountName = ""
	Conf.confFiles[Conf.Type.AUTH_TOKEN].clear()
	Conf.cache.clear()
	DirAccess.remove_absolute(Path.Profile + "auth_token" + Path.ConfExt)

func FillFieldsFromToken():
	if savedToken.is_empty():
		return

	fillingFields = true
	nameTextControl.set_text(nameText)
	passwordTextControl.set_text("令牌令牌令牌")
	fillingFields = false

#
func Connect():
	if not Network.IsServerReachable():
		FillWarningLabel(NetworkCommons.AuthError.ERR_SERVER_UNREACHABLE)
		return

	nameText = nameTextControl.get_text()
	if not savedToken.is_empty():
		if Network.LoginWithToken(savedAccountName, savedToken, NetworkCommons.GetPlatform()):
			nameText = savedAccountName
			savedToken = ""
			FSM.EnterState(FSM.States.LOGIN_PROGRESS)
		return
	var passwordText : String = passwordTextControl.get_text()
	var authError : NetworkCommons.AuthError = NetworkCommons.CheckAuthInformation(nameText, passwordText)
	FillWarningLabel(authError)
	if authError == NetworkCommons.AuthError.ERR_OK:
		if Network.LoginWithPassword(nameText, passwordText, rememberMeCheckBox.button_pressed, NetworkCommons.GetPlatform()):
			FSM.EnterState(FSM.States.LOGIN_PROGRESS)

func CreateAccount():
	if not Network.IsServerReachable():
		FillWarningLabel(NetworkCommons.AuthError.ERR_SERVER_UNREACHABLE)
		return

	nameText = nameTextControl.get_text()
	var passwordText : String = passwordTextControl.get_text()
	var confirmText : String = confirmPasswordTextControl.get_text()
	var emailText : String = emailTextControl.get_text()

	var authError : NetworkCommons.AuthError = NetworkCommons.CheckAuthInformation(nameText, passwordText)
	if authError == NetworkCommons.AuthError.ERR_OK:
		if passwordText != confirmText:
			authError = NetworkCommons.AuthError.ERR_PASSWORD_MISMATCH
	if authError == NetworkCommons.AuthError.ERR_OK:
		authError = NetworkCommons.CheckEmailInformation(emailText)

	if authError == NetworkCommons.AuthError.ERR_OK:
		if Network.CreateAccount(nameText, passwordText, emailText, rememberMeCheckBox.button_pressed, NetworkCommons.GetPlatform()):
			FSM.EnterState(FSM.States.LOGIN_PROGRESS)
	else:
		FillWarningLabel(authError)

func RequestReset():
	if not Network.IsServerReachable():
		FillWarningLabel(NetworkCommons.AuthError.ERR_SERVER_UNREACHABLE)
		return

	nameText = nameTextControl.get_text()
	if nameText.is_empty():
		nameTextControl.grab_focus()
		return
	Network.RequestPasswordReset(nameText)

func ConfirmReset():
	if not Network.IsServerReachable():
		FillWarningLabel(NetworkCommons.AuthError.ERR_SERVER_UNREACHABLE)
		return

	var codeText : String = resetCodeTextControl.get_text()
	var newPassword : String = passwordTextControl.get_text()
	var confirmText : String = confirmPasswordTextControl.get_text()

	if not NetworkCommons.CheckResetCode(codeText):
		FillWarningLabel(NetworkCommons.AuthError.ERR_RESET_INVALID_CODE)
		return

	var passwordErr : NetworkCommons.AuthError = NetworkCommons.CheckPasswordInformation(newPassword)
	if passwordErr != NetworkCommons.AuthError.ERR_OK:
		FillWarningLabel(passwordErr)
		return

	if newPassword != confirmText:
		FillWarningLabel(NetworkCommons.AuthError.ERR_PASSWORD_MISMATCH)
		return

	Network.ConfirmPasswordReset(nameText, codeText, newPassword)

func Close():
	if recoveryState != RecoveryState.NONE:
		SetRecoveryState(RecoveryState.NONE)
	elif isAccountCreatorEnabled:
		EnableAccountCreator(false)
	else:
		Launcher.GUI.quitBox.Toggle()

#
func _on_text_focus_entered():
	if Launcher.Action:
		Launcher.Action.Enable(false)

func _on_text_focus_exited():
	if Launcher.Action:
		Launcher.Action.Enable(true)

func _on_text_submitted(_newText):
	Launcher.GUI.buttonBoxes.Call(UICommons.ButtonBox.PRIMARY)

#
func _on_visibility_changed():
	if visible:
		if not fieldsEdited:
			LoadSavedToken()
			FillFieldsFromToken()
		EnableButtons(true)
		if pendingFocusControl and pendingFocusControl.is_visible_in_tree():
			ApplyFocus.call_deferred()
		else:
			FocusDefault()

# Toggle between online and offline (single-player) mode.
# Offline means "run a local server" (see Network.Mode), so the server must be
# launched only when we are currently online. Read the live mode instead of
# trusting the value captured when the button got bound, which can be stale.
func SwitchOnlineMode(_toggled : bool = false):
	var launchServer : bool = not (Network.ENetServer != null or Network.WebSocketServer != null)
	EnableButtons(true)
	if Launcher.Mode(true, launchServer):
		EnableButtons(false)

func _on_name_text_changed(_newText : String):
	if recoveryState == RecoveryState.NONE and not isAccountCreatorEnabled and isConnectPrimary != IsConnectPreferred():
		EnableButtons(true)
	if not fillingFields:
		fieldsEdited = true
		if not savedToken.is_empty():
			savedToken = ""
			passwordTextControl.clear()

func _on_password_text_changed(_newText : String):
	if not fillingFields:
		fieldsEdited = true
		savedToken = ""

func _on_remember_me_toggled(toggled_on : bool):
	if not toggled_on:
		ClearSavedToken()

func _ready():
	Launcher.launchModeUpdated.connect(OnlineMode)
	get_viewport().size_changed.connect(RefreshLayout)
	for control : Control in fieldControls:
		var text : LineEdit = control.get_node("Container/Text")
		text.mouse_entered.connect(SetFieldHovered.bind(text, true))
		text.mouse_exited.connect(SetFieldHovered.bind(text, false))
	RefreshLayout()
	if LoadSavedToken():
		rememberMeCheckBox.button_pressed = true
