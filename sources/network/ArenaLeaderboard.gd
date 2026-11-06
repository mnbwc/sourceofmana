# Arena 排行榜（tick18）——episode 结果持久化。
# 独立 JSONL 落盘（user://arena_leaderboard.jsonl），零上游改动；每行一个终局 episode。
# 读侧：tests/harness 或外部脚本按行 JSON 解析聚合。SQLite 待 M3 多局聚合再上。
# tick23 fix：Godot FileAccess.WRITE_READ 打开即截断（非追加），此前每局会把历史
# 记录清掉只剩最后一行——改为读全部已有行 → 整体重写（旧行 + 新行），保序保历史。
extends Node

const LEADERBOARD_FILE := "user://arena_leaderboard.jsonl"

func _ready() -> void:
	if not "--server" in OS.get_cmdline_args():
		set_process(false)

# 由 ArenaController._finish 调用：终局 episode 追加落盘（保历史，tick23 修复截断 bug）
func Record(episode : Dictionary) -> bool:
	var existing : Array = Entries()
	existing.append({
		"episodeId": episode.get("id", ""),
		"nick": episode.get("nick", ""),
		"template": episode.get("templateKey", ""),
		"status": episode.get("status", ""),
		"endReason": episode.get("endReason", ""),
		"score": episode.get("score", {}),
		"startedAtMsec": episode.get("startedAtMsec", 0),
		"endedAtMsec": Time.get_ticks_msec(),
		"lineNo": existing.size() + 1
	})
	var w := FileAccess.open(LEADERBOARD_FILE, FileAccess.WRITE)
	if w == null:
		printerr("[ArenaLeaderboard] cannot open %s" % LEADERBOARD_FILE)
		return false
	for entry in existing:
		w.store_line(JSON.stringify(entry))
	w.close()
	return true

func Entries() -> Array:
	var out : Array = []
	var f := FileAccess.open(LEADERBOARD_FILE, FileAccess.READ)
	if f == null:
		return out
	while not f.eof_reached():
		var line := f.get_line()
		if line.strip_edges() != "":
			var parsed = JSON.parse_string(line)
			if parsed != null:
				out.append(parsed)
	f.close()
	return out
