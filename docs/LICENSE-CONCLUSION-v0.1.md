# Source of Mana 许可与 ToS 结论书 v0.1（T1 / M0 交付物）

> 状态：草案（待项目所有者签署）。日期：2026-11-06。
> 依据：LICENSE.md(693行) / CC BY-SA 4.0(428行) / data/db/credits.json / data/db/agreement.json 实文。
> 未核实开放项：policies.themanaworld.org/tsc 全文（网络受限未取，登记为开放风险，不阻塞草案）。

## 一、许可分层

| 层 | 许可 | 适用 | 对本项目的义务 |
|---|---|---|---|
| 代码（sources/ 等） | MIT | 全部 GDScript/配置 | 保留版权声明与许可文本；MIT 允许修改/闭源/商用，无 copyleft |
| Godot 引擎 | MIT | 引擎二进制 | 随引擎分发声明，本项目形态（本地运行）无额外动作 |
| 美术/地图/音乐/字体（data/ 下资产） | CC BY-SA 4.0 | 精灵、瓦片、音频、DejaVuSans/Arev 字体 | **ShareAlike**：含资产的演绎材料须同许可发布；**署名**：逐文件 credits |
| 逐文件 credits | LICENSE.md 内 645 行清单 + data/db/credits.json | 每个资产文件 | 对外分发含资产的产物时须附对应署名 |

## 二、三层结论

1. **自建私有实例 + 本地研究 → ✅ 完全可行。** MIT 无任何使用限制；agreement.json 前言限定"registering for and using any of Source of Mana's services"，自建 headless server 不属于其服务。技术隔离仍强制执行（见开工包 §5）。
2. **官方服务器/平台/社区 → 🔴 禁止。** 不连 som.manasource.org；不采集官方服数据；不以官方名义办赛；公屏英语规则亦不触（我们不在官方公屏）。
3. **对外发布 → ⚠️ 分层：**
   - 结构化 JSON 轨迹（坐标/事件/数值）→ 不含美术资产，SA 不触发，低风险
   - 截图/录像/含瓦片地图的演示页/含音乐的直播 → CC BY-SA 4.0 演绎材料，须同许可 + 逐文件署名
   - 不得暗示官方背书；项目名与商标隔离

## 三、agreement.json Game Rules 关键条款（实文核实）

- No AFK Botting: "Automation of gameplay while away from the keyboard is prohibited."
- No Multiboxing: "Controlling multiple characters in combat simultaneously is forbidden."
- Social Convention → 引用 policies.themanaworld.org/tsc

**边界解释**：上述条款约束的是"using any of Source of Mana's services"（官方服生态）。本项目：自建实例、无官方账号、无官方社区互动 → 不适用。**无论解释是否成立，第 5 节技术隔离强制执行**（出站白名单=仅 LLM 供应商 API）。

## 四、义务清单（对外启用前最低 11 项）

1. MIT 版权声明与许可文本随衍生代码保留
2. README/发布页链接 github.com/sourceofmana
3. 含资产产物附逐文件署名（credits.json 可机器生成清单）
4. CC BY-SA 4.0 全文随含资产产物分发
5. 项目命名不使用"Source of Mana"官方名作背书（可注明"基于 Source of Mana 改造"）
6. 对外产物分层审查表（每类产物过 SA/署名检查后才可发布）
7. 本地训练用途说明（仅自建实例数据，无官方数据）
8. tsc 条款核对（开放风险，取到全文后补）
9. 旧 Kaetram 轨迹冻结声明（见 PROGRESS.md 红线 2）
10. 自建实例隔离证明（M1 交付）
11. 分支 B（对外）启用条件：本结论书所有者签署 + P4 冻结解除评审

## 五、结论

代码 MIT + 资产 CC BY-SA 4.0，**无任何反 AI 条款**（与 Kaetram OPL 毒丸对比：Gate-0 最大障碍已消除）。自建实例路线合法可行。本结论书待所有者签署后，M0 完成。
