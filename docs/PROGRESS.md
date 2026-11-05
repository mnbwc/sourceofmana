# SoM AI 竞技场 — 进度台账

> 项目：Source of Mana（Godot 4）AI 原生竞技场。开工包：元宝 v1.0（T1-T14 / M0-M5）。
> 底座：~/sourceofmana @ feat/agent-arena（上游 master=1933bd6）。Godot 4.7.1 已装 /usr/local/bin/godot471。
> 红线：① 只跑自建实例，永不连官方服；② 旧 Kaetram 全部轨迹冻结（不迁移/不训练/不发布）；③ 禁 isAgent 业务短路；④ 信息对等。

## 当前阶段

**P0-Mana spike**（M0 许可结论书 → M1 起服 → M2 缺口表 → M3 四元组阀门 → M4 锁步 → M5 闭环）

## 已完成

- 2026-11-06 T2：切 feat/agent-arena 分支；godot471 4.7.1 实机验证（--headless --import + --server 起服、ENet 6119 监听、pyenet CONNECT ✅，见会话记录）
- 2026-11-06 T1：M0 许可结论书草案 docs/LICENSE-CONCLUSION-v0.1.md（LICENSE.md/CC BY-SA 4.0/credits.json/agreement.json 实文核实；tsc 未取到=开放风险登记）
- S17 版本机制已核清（会话内完成）：Network.gd:624 `ComputeProtocolVersion(self)` 传 Network 节点自身 → **Network.gd 加 @rpc 必改哈希；独立 AgentBridge.gd autoload 不影响** → M3 通道 1（独立 autoload+复用 6108）可行

## 进行中

- T7/T8（M3）：AgentBridge.gd autoload 探针 → 四元组阀门

## M1 状态：✅ 完成（T3 导出成功 + T4 隔离八项过 + T5 基线 225MB）

## 待办（按开工包顺序）

- T5 资源基线、T7-T9 M3 阀门、T10 契约冻结、T11 M4 锁步、T12 M5 四元组闭环、T13 汇总、T14 旧仓 tag 归档
- 待所有者：签署许可结论书（M0 退出条件）；P4 对外冻结解除评审

## 跳过/挂账

- tsc 全文（policies.themanaworld.org/tsc）网络未取到 — 开放风险，不阻塞
