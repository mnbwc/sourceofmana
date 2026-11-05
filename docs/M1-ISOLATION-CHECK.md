# M1 隔离检查记录（T3/T4 交付物）— 2026-11-06

## T3：release 导出 Headless Server

- 下载 Godot 4.7.1 官方 export templates（1.28GB tpz → ~/.local/share/godot/export_templates/4.7.1.stable）
- `godot471 --headless --export-release "Linux/X11 Headless Server" /tmp/som-headless-server` → **成功，产物 124MB**
- 注意：导出包默认排除 GDExtension .so（godot-sqlite/sentry/webrtc_native），首跑报缺库；从仓库拷 3 个 .so 进包后正常
- 起服验证：`./som-headless-server --headless --server` → SQLite 建库 ✅ ENet+WS+WebRTC init ✅（与编辑器模式行为一致）

## T4：八项隔离检查结果

| # | 检查项 | 结果 |
|---|---|---|
| 1 | ServerAddress 指向本地 | ✅ `data/conf/settings.cfg` Network-Local=true（客户端连 127.0.0.1）；ServerAddress 常量仅作客户端连接目标，服务端不外连 |
| 2 | Sentry 门控 | ✅ 实测未外发（无 token 配置）；出站连接快照=0 |
| 3 | Discord/Email 裁剪 | ✅ credential.cfg `Discord-Enabled=false` + Token 空 → isBotReady=false，`SendToDiscord` 直接 return（Discord.gd:82）；Email "Not configured: missing API key"；保留 SQL（登录依赖） |
| 4 | TriggerChat GLOBAL 外发 | ✅ 源码层已断：GLOBAL→SendToDiscord→isBotReady=false→return，无外发路径 |
| 5 | 单传输 | 🟡 未做（三监听并存：6118 WS + 6119 ENet + WebRTC）。**决策：保留现状**——关 ENet 需改 NetworkCommons.UseENet（上游耦合），低资源收益小，WebRTC 需信号但无出站。挂账 L7，M5 后统一裁 |
| 6 | 防火墙出站白名单 | ⚠️ 容器层面待配（本机无 iptables 权限操作记录）；**实测出站连接=0**（无外连行为） |
| 7 | 无渲染/无美术包 | ✅ headless + dedicated_server 模板；DISPLAY/WAYLAND 环境变量=0；导出 strip 美术 |
| 8 | 隔离证明快照 | ✅ 监听 `*:6118(tcp WS)/*:6119(udp ENet)`；出站非本地连接=0；进程列表=som-headless-server 单进程 |

## 验收判定

**隔离达标**（服务端无任何出站外连路径；唯一注意项=未来接入 LLM API 时的出站白名单由 harness 进程承担，不在本进程）。L7 单传输裁剪挂账。

## 遗留

- T5 资源基线（RSS/frame P95/30Hz 达成率）→ 下一步
- 导出包 exclude .so 问题：需在 export_presets.cfg include_filter 补 `addons/*/bin/*.so,addons/*/lib/**/*.so`（T4.5，避免每次手拷）
