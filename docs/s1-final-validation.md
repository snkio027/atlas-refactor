# S1 干净单次验证与收口

2026-09-28：同一个候选、同一个新 plan、同一个干净四节点集群、普通 `run` 的
**单一 attempt 连续通过 29/29 checkpoint，终态 REVERSE_VERIFIED / exit 0**。
S1 runtime 验收完成，功能开发冻结，进入 PR #6 收口审核。
历史 STOP 保持失败；本结论不依赖 continuation 或跨 attempt 拼接。

## 精确绑定

| 项目 | 值 |
| --- | --- |
| 实现 | `dff5a20fd08be7ddf5c43d68f00cdf6c3aa5e2bd` |
| atlas-ot1 SHA256 | `59e680f2c0d814bc4aed4de8ce180ae521a43936fda25144cef07545ff129913` |
| Plan SHA256 | `2278dbacfcfa615d5f74683062143229accf42a7508877c90c60c7968beff16b` |
| 目标 | `atlas-refactor-test-ot1`；UID `b80928e0-71fa-450c-b3f3-3db06c6c22a7` |
| 完整平台基线 | `acf6cac57a430b91e6ead1f10146928883cb4642` |
| 最终 Git | `fa777af73e0b085813bac5d2f204bf8ba0e5508f`，远端与 runtime checkout 一致 |
| 运行时间 UTC | 2026-09-28 14:08:40.908–14:18:51.206，约 610.3 秒 |
| 阶段预算 | 300 秒；实际最慢 index 24 为 60.851 秒；总预算 160 分钟 |
| 工具链 | Go 1.27.1，CGO_ENABLED=0，-trimpath，darwin/arm64，clean VCS |
| 锁定工具 | Helm 4.2.3；Kind 0.32.0；kubectl 1.36.3；Kubernetes 1.36.1；Argo CD 3.5.1 |

审批覆盖精确旧 UID 删除、干净重建、Tier-0/凭据、平台验收与完整 29 阶段。
新 UID、专用 kubeconfig、三份密文和七个 Git 投影属于计划的机械绑定；动作范围未增加。
运行中未换 binary/plan/target、未手改 spec/tracking、未额外 refresh、未清锁续跑。

## 已验证行为

| 验收项 | 本次事实 |
| --- | --- |
| 干净重建 | 删除旧 UID ce4e11e4，仅重建 OT-1；四节点 control-plane/gateway/compute/data；loopback 18080/18443 |
| Bootstrap | 双 Seed、一次性 External Root、Latch/Receipt、GitOps ADOPTED；重复 apply 零写入 |
| Gate 0 | BASELINE_ADOPTED，ownership 与独立 Atlas Gate-B 通过 |
| Gate 12 | 部分转移后回退，原 owner 重新接管及 parent reattachment 通过 |
| Gate 23 | 完整 1→3，secrets=3 / observability=4 / storage=6；三个 owner 均 strict，窗口 0 |
| Gate 28 | 完整 3→1，13 个资源回到 capability-foundation；父应用重接、最终 Gate-B 通过 |
| 资源不变量 | 13 UID、semantic content、Argo SSA 保持；四个 Bootstrap records、四个 AppProject、Root/self UID/spec 保持 |
| 最终 runtime | 四节点 Ready；37 个 Pod（含完成 Job）及放置通过；PVC Bound、PV Retain；HTTP 301、CA 验证 HTTPS 200 |
| 最终 rollout | 24 个 App idle/Synced/Healthy，窗口 0；platform-control/foundation exact-current |

最终独立读到 23 个 App 报告最终 Git，secrets-controller 仍报告前一个 planned revision
483897c；其完整 source closure/spec 在 plan 内连续相同，符合既有 Desired Identity。
这项等价只用于只读 rollout evidence，未进入 mutation authorization。

四个 Gate 的重复 apply 均 exit 0、deniedWrites=0、identity digest 不变；kubectl mutation
审计计数分别为 72→72、96→96、118→118、132→132。成功后仅移除本 attempt 的运行锁。
默认 kubeconfig 摘要保持 `862e14c996585bf3b2c9f4cb4026bffdd56f614faea8c0a85b099b98949c651d`。

## 平台功能与发布链路

完整平台基线验收包括：S3 签名 CRUD/HEAD、预签名 GET、分片上传与清理；匿名和跨 bucket
均 403；Grafana 登录、26 个看板、Prometheus 数据源；27 个 targets 全 UP、31 组规则、
四节点指标；Alertmanager 本地告警触发/恢复。未验证外部通知。
这些 API 测试在 ceremony 前执行；四次 Gate-B 复核 runtime/存储/HTTPS，不重跑全部功能 API。

| 发布阶段 | Git 确认至全部目标父/子比较完成 |
| --- | ---: |
| 1 SOURCE_RELEASED | 9.071 s |
| 12 MIXED_ROLLBACK_VERIFIED | 7.502 s |
| 13 SECOND_SOURCE_RELEASED | 8.501 s |
| 23 FORWARD_VERIFIED | 9.873 s |
| 24 REVERSE_TARGETS_RELEASED | 17.192 s |
| 28 REVERSE_VERIFIED | 6.910 s |

实际 49 个 foundation 请求、11 次 normal refresh（6 parent + 5 child，批准上限 16）、
6 次 Git 发布（18 条 Git 请求），全部成功。已完成比较的 child 没有重复 refresh。
基线 6,650 个 audit ID 全部保留，最终 8,809 个；增量 2,159 条中 60 条 kubectl mutation
与请求的对象、动词、顺序和总数逐一对应。92 条受测资源写入（含 22 条 Namespace）全部
来自 Argo。Metadata-only 审计没有请求正文；补丁 UID/RV/full-spec tests 保存在请求 intent。
独立关联允许宿主机/VM 的 ±1 秒时间差，并要求精确序列，不能把两个时钟当成一个。

## 实际测试与证明边界

- 执行前及收口文档完成后，锁定 `task quality` 均通过：Go vet/race、真实 Helm/Kustomize、平台资源检查及 Python contracts。
- 三个历史固定入口的 tagged race/vet 通过；没有在本次执行中使用这些入口。
- 时序回归把 F12、连续 Desired Identity、六次发布及三个采集入口组合起来：合法竞争重采，
  未知/不可用/真实漂移 STOP，同一 deadline，写请求序列不变，Gate 输入只执行一次。
- 保留的 F16 原始样本缺少 closing Projects/Nodes，仍被拒绝；后续完整样本可验证，未改写旧 STOP。
- 独立离线检查重新 Assess 全部 29 原始 snapshot、四组 Gate artifacts、checkpoint 摘要链和 terminal。
  单独只读复核 live UID/content/SSA、控制记录、Git、窗口、节点及审计，结果 PASS。

本次 runtime 证明整条链可一次完成；没有人为注入 live status race，也不推断确切重采次数。
采集竞争的写入不重放保证由组合回归与本次实际请求审计共同支撑。双读不是全局原子快照。

## 证据与冻结边界

最终 bundle、manifest 和历史结果见 [证据索引](s1-evidence-index.md)；失败经验精简到
[Failure Journal](s1-failures.md)。新 Trust Root 另存 w1 的独立目录，已验证密钥对；
`physicallyIsolated=false`，仅为批准的 OT-1 开发例外。三份严格密文已公开发布；明文、
私钥和 kubeconfig 未入 Git 或最终 bundle。

保留长期能力 internal/observation 和必要 semantic evidence；OT-1 状态机与旧固定入口
冻结为 ceremony reference，不扩成通用 resume/recovery/rollout controller。

剩余非 S1 Gate：PR #6 收口审核/合并、ADR-0009～0013 disposition、远端 CI/发布供应链、
Trust Root 物理隔离、服务端 admission、独立 recovery、原 Atlas parity/cutover。
本次未将这些项标为通过；PR 保持 draft，远端 status checks 为空，ADR 仍 Proposed。
后续路线为 S2 Project + Workload + Binding，尚未开始实现。
