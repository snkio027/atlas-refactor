# S1 最终验证与冻结

2026-09-28（Asia/Shanghai）：S1 的隔离 OT-1 runtime 验证闭环完成，功能开发冻结。
最终执行终态 **REVERSE_VERIFIED / exit 0**。证据跨原运行、已审查的固定 anchor 与
最终 reverse 串联；历史 STOP 全部保持原样，最终新 attempt 只含原 indices 24..28。
PR #6 进入收口审核；ADR disposition、远端 CI 和发布准备仍是独立 Gate。

## 精确绑定

| 项目 | 值 |
| --- | --- |
| 实现 | `c7a09b4c0b51fd0f96030e7402a62371b1add554` |
| 二进制 SHA256 | `2bca3fb53ed1a5df7a0f66c66ef77de9498f5f2b709f86742fb271a10d4d707b` |
| Plan SHA256 | `91dc1dfec0417f24d1b1ab683daf559dc267bdc650f46690be5f97088bfc3e0b` |
| Toolchain | Go 1.27.1；CGO_ENABLED=0；-trimpath；darwin/arm64；clean VCS |
| 目标 | atlas-refactor-test-ot1；UID `b886f730-ea3b-44b9-a904-8bd55ac345f2` |
| 最终 Git | `28c4dc68842e7c4df0f62b871d120111d6f778c9`，runtime 与远端一致 |
| 结束时间 | 2026-09-27 17:57:03 UTC |
| 最终 manifest | `ee76df575e7745643f555af97f343b7bf20f3b923b77ddd7c3e09ada2a415e61` |

所有锁定工具/制品摘要、七个原 Git SHA、source/render hashes、权限及 13-object scope
都保存在 canonical plan 和私有证据中。新的 plan 相较 predecessor 只改变实现绑定。
最终 bundle：`.state/authority/ot1-final-91dc1dfe`，83 files、398,686,298 bytes；
引用五份校验未变的历史 bundle，保留实际 executable、plan、授权、请求、Gate 和审计。
重复的执行前只读预检已清理，运行时 checkout 未移动；明文凭据、私钥和 kubeconfig 未入 Git。

## 实际通过的闭环

| 边界 | 运行证据 |
| --- | --- |
| 干净平台基线 | 四节点 Bootstrap、双 Seed/Receipt、基线 Gate-B；首次完整 S3/监测验收见 Failure Journal |
| partial rollback | 历史 7..11 checkpoints 与新的 stage-12 read-only Gate-B；旧 stage-12 STOP 不改写 |
| forward 1→3 | 前次 13..22 checkpoints：三组 strict refusal、window adoption、strict restore，3/4/6 ownership |
| forward Gate-B | 本次 `CONTINUATION_ANCHOR_FORWARD_VERIFIED`：exact STOP/manifest/Git/UID/lock/audit + 完整 Gate-B |
| reverse 3→1 | 本次 24..27：三个 owner orphan release，source strict refusal、readoption window、strict restore |
| 最终 Git 重接与 Gate-B | 本次 28：REVERSE_VERIFIED，完整独立 Gate-B；293.4 秒内通过，阶段上限仍为 300 秒 |

最终 13 个资源均回到 capability-foundation，UID、完整 semantic content 与 Argo SSA
相对最初基线保持；开放窗口 0。source Application 是按计划重建的新 UID
`bb66a9cc-811d-4408-aecc-1c1043ddcc08`；资源 identity 与 owner App 生命周期分开验证。
四个 Bootstrap records、四个 AppProjects、Root/self UID 与 spec 不变。

最终 Gate-B：Bootstrap ADOPTED；repeat apply 47 个只读请求、exit 0、零 denied writes，
审计 mutation IDs `121 → 121`，identity digest 不变。四节点 Ready；37 个 Pod（含已完成
Job）、部署位置与必需运行组件通过；PVC Bound、PV Retain；HTTP 301、CA 验证 HTTPS 200。
24 个 App 均 idle/Synced/Healthy；18 个报告最终 SHA，6 个普通 leaf 仍报告 dd2e4cd，
其完整 desired inputs 等价。platform-control 与 foundation owner 均 exact-current。

执行器仅提交 9 个受保护 Application 写请求和 2 次原 Git 发布（6 个 Git 命令），全部
exit 0。审计增量 1,689 事件，9 个 kubectl mutation 与请求对应；26 条受测资源写入全部
来自 Argo，包含 6 条 Namespace 写入。46,969 个 anchor audit IDs 全部保留。
成功后仅移除精确 successor lock；默认 kubeconfig 摘要和所有历史证据未变。

## F13/F14 与架构一致性

- Desired Identity 由 published immutable plan 中连续不变的完整 source closure 与 App
  spec 确定。第一次变化/未知输入即截断；内容改后恢复不能跨过该边界。未来/未知 SHA 拒绝。
- 仅普通 leaf 的只读 rollout evidence 使用等价规则。BASELINE_ADOPTED、platform-control、
  foundation/active owners 仍 exact-current；mutation 的 Git/UID/RV/full-spec fence 未改变。
- Bootstrap 的 ADOPTED 只证明 durable authority；rollout/runtime 由 Observation/Gate-B
  独立证明。首次 handoff 保持严格，任何健康失败都不恢复 Seed mutation authority。
- F12 的 operation proof → reconciledAt >= finishedAt → comparison proof 原样保留。
  正常 Bootstrap、Observer 与固定 Ceremony 分离；没有直接 tracking 写、refresh、自动
  rollback、通用 resume/recovery、额外 schema 或新常驻控制器。

锁定 `task quality` 通过（Go vet/race、真实 Helm/Kustomize、348-resource 检查、Python
contracts）；三个固定入口的 tagged race/vet 通过。回归覆盖多版连续等价、intermediate
change/revert、未知/未来、UID/spec、健康与 critical-owner 拒绝。真实 F14 stage-21/22
快照本地重放通过，关键 owner 尚未 current 时仍阻止提前 Gate PASS。多版等价的边界
由回归和历史快照覆盖；没有人为注入 live stale status。

执行后独立离线复核 fresh forward anchor、全部五个 checkpoint/snapshot hash chain、
原始 Gate artifacts 与最终 terminal 均通过；另行只读复核 scope/control identity、Git、
cluster UID、审计、默认 kubeconfig 和锁清理。最终 Gate 未重跑 S3 CRUD、Grafana 登录、
Prometheus API 与外部通知；前三项功能基线及通知边界见既有平台验收记录。

## 冻结与审核边界

长期保留 internal/observation 和必要 semantic evidence。OT-1 state machine 与固定
事故入口冻结为已验证的 ceremony reference；不推广为普通 Atlas 生命周期接口。

| 决策记录 | 当前 disposition |
| --- | --- |
| ADR-0009 | 隔离 ownership 实验目标已有完整 runtime 证据；Proposed，待 owner 审核 |
| ADR-0010 | semantic double-read 与原事故边界保留；revision 规则由 ADR-0011 收敛；Proposed |
| ADR-0011 | F13/F14 最终语义及本次收口通过验证；Proposed，待 owner 审核 |
| ADR-0012 | 固定 stage-12 的历史执行/STOP 保留，工具冻结；Proposed |

PR #6 保持 draft；远端 CI/status checks 仍为空。本次不实现 CI infrastructure、不进入 S2，
不批准原 Atlas cutover 或生产发布。服务端 admission 保护、独立 recovery、供应链发布验证
和 Trust Root 物理隔离备份仍按既有边界处理；OT-1 的同机备份是已批准的开发例外。
