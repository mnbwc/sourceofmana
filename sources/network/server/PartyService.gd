extends Node
class_name PartyService

# MVP-8 组队最小边界（规格 §3.1）：只做社交与共同在场。
# XP 与掉落沿用原版单人规则，不新增任何分配规则。
# rid = agentRID（PlayerAgent.get_rid().get_id()，全局唯一）。

var parties : Dictionary = {}          # partyId -> {"leader": int rid, "members": {rid: true}}
var ridToParty : Dictionary = {}       # rid -> partyId
var invites : Dictionary = {}          # target rid -> inviter rid（一次一个，新邀请覆盖旧）
var _nextId : int = 1

func CreateParty(leaderRID : int) -> int:
	if ridToParty.has(leaderRID):
		return 0
	var pid : int = _nextId
	_nextId += 1
	parties[pid] = {"leader": leaderRID, "members": {leaderRID: true}}
	ridToParty[leaderRID] = pid
	return pid

func Disband(pid : int) -> void:
	if not parties.has(pid):
		return
	for rid in parties[pid]["members"].keys():
		ridToParty.erase(rid)
	parties.erase(pid)

func AddInvite(targetRID : int, inviterRID : int) -> bool:
	if ridToParty.has(targetRID):
		return false    # 已在队伍
	invites[targetRID] = inviterRID
	return true

func PopInvite(targetRID : int) -> int:
	var inviter : int = invites.get(targetRID, 0)
	invites.erase(targetRID)
	return inviter

func Join(pid : int, rid : int) -> bool:
	if not parties.has(pid) or ridToParty.has(rid):
		return false
	if parties[pid]["members"].size() >= ActorCommons.MaxCharacterCount:
		return false
	parties[pid]["members"][rid] = true
	ridToParty[rid] = pid
	return true

func Leave(rid : int) -> int:
	var pid : int = ridToParty.get(rid, 0)
	if pid == 0:
		return 0
	var party : Dictionary = parties[pid]
	party["members"].erase(rid)
	ridToParty.erase(rid)
	if party["members"].is_empty():
		Disband(pid)
	elif party["leader"] == rid:
		# 队长离队 → 转让给第一个成员
		party["leader"] = party["members"].keys()[0]
	return pid

func Kick(pid : int, leaderRID : int, targetRID : int) -> bool:
	if not parties.has(pid) or parties[pid]["leader"] != leaderRID:
		return false
	if not parties[pid]["members"].has(targetRID) or targetRID == leaderRID:
		return false
	return Leave(targetRID) == pid

func Promote(pid : int, leaderRID : int, targetRID : int) -> bool:
	if not parties.has(pid) or parties[pid]["leader"] != leaderRID:
		return false
	if not parties[pid]["members"].has(targetRID):
		return false
	parties[pid]["leader"] = targetRID
	return true

func GetParty(rid : int) -> Dictionary:
	var pid : int = ridToParty.get(rid, 0)
	return parties.get(pid, {})

func GetPartyID(rid : int) -> int:
	return ridToParty.get(rid, 0)

func GetMembers(pid : int) -> Array:
	if not parties.has(pid):
		return []
	return parties[pid]["members"].keys()

func GetLeader(pid : int) -> int:
	return parties.get(pid, {}).get("leader", 0)
