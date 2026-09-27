# S1 Failure Journal

失败知识保留在回归测试和本表中。这里的“零 mutation”指失败的 transfer/preflight
未执行 ceremony 写请求、未发布后续 Git 阶段；不否认先前真实 Bootstrap 和平台创建。
真实创建、凭据/Trust Root 的证据仍见 [首次现场记录](ot1-clean-rebuild-20260927.md)。
最终完整 29 阶段及各交接点的 Gate-B 尚未全部通过，ADR-0009 保持 Proposed。

| ID | 症状与原因 | 外部影响 | 修复 | 回归 |
| --- | --- | --- | --- | --- |
| F1 | BASELINE_ADOPTED 错要求当前 ceremony operation；纯观察应允许最近成功 operation 早于已观察 Git SHA | 原始 plan `3b3d89475fac4d4d977aca05a0bb4a9598b24f8eb7a709ab369a65ac262adbfb` 在第 0 阶段 STOP；0 请求 intent，审计区间 64 事件、0 kubectl mutation，未发布后续 Git | `76f2cba`：operation proof iff compiled Step.Sync | `TestObservedStateDoesNotInventCeremonyOperation` |
| F2 | verified typed List 的成员可能省略 TypeMeta；逐项要求字段导致 NodeList 无法读取 | 只读诊断失败，0 mutation | `76f2cba`：在 discovery/scope/list header 验证后仅补缺失字段，显式冲突仍拒绝 | `TestTypedInventoryNormalization`、`TestTypedInventoryCannotBypassDiscoveryOrScope` |
| F3 | 有效 run 错继承准备阶段的 10 分钟 deadline，完整计划可能提前中断 | 本地审查发现；未在现场触发 | `76f2cba`：从原始 parent 与开始时间推导 29×300 秒 + 15 分钟；保留调用者取消 | `TestRunDeadlineCoversPlanAndPreservesCallerControl` |
| F4 | Argo CD 3.5.1 的 hook 比较条目可能省略普通 sync status；普通资源也可能带 hookPhase | 只读预检失败，0 mutation | `4ec6005`：仅接受 hook=true 且 status 缺失；有 HookType 的实际 hook outcome 独立分类 | `TestHookComparisonOmissionDoesNotHideFailure`、`TestHookOutcomeRequiresRecognizedType`、`TestRunningHookRemainsNonReadyDuringConvergence` |
| F5 | API 编码省略空 LabelSelector.matchLabels map，Git/live 基线误报内容漂移 | 只读预检失败，0 mutation | `4ec6005`：仅在 NetworkPolicy 已知 selector 路径的首次基线比较中规范化空 map；selector 本身及后续 live semantic 不变 | `TestBaselineSelectorWireOmissionIsNarrow`、`TestNetworkPolicyEmptySelectorsKeepPeerAndPlacement` |
| F6 | Gate-B 包装器拒绝正常 Seed inventory 的 stdin manifest GET，普通 CLI ADOPTED 而包装后 UNAVAILABLE | plan `c26950f5c7d6920fb1f6571db504e2bcfbe32bbdf98557480307578ecdf76452` 第 0 阶段 STOP；2026-09-27 12:25:43–12:26:12 UTC 审计 78 事件、0 kubectl mutation，0 请求 intent，未发布后续 Git | `6880e76`：仅放行精确 `get -f - --ignore-not-found=true --show-managed-fields -o json`；Gate 错误携带实际 state/detail | `TestReadOnlySeedInventoryWithStdin`；修复后同集群只读 Status=ADOPTED，47 请求、0 denied |
| F7 | Gate-B 把 Gateway 配置命名空间 atlas-gateway 当作 Envoy 数据面 Pod 的命名空间；实际控制器与代理均在 envoy-gateway-system | plan `dd0746f0c0dc41dacb181f8163eed0f21d02e36070f0dae4babd1c640b255a73` 第 0 阶段 STOP；审计 93 事件、0 kubectl mutation，0 请求 intent，未发布后续 Git | `b300d7a`：使用实际代理命名空间，并检查 development Gateway labels 与 gateway worker；不再要求仅含配置的 namespace 必须有 Pod | `TestRuntimeGatewayUsesControllerNamespace`：正确布局通过，错 namespace/node/Gateway labels、未就绪或缺失数据面均拒绝 |
| F8 | strict restore 操作完成后，Application List/GET 采集窗口内 observability-foundation 的 RV 28396→28402；generation、managedFields 与 reconciledAt 更新，spec 相同；触发 APPLICATION_CHANGED_DURING_CAPTURE | 已有真实 mutation，详见下方当前现场；保留完整 authority evidence 和 STOP 锁 | 跨版本拒绝按既定规则生效；未放宽 fence、未重试 mutation、未自动重采或续行 | `TestAll29StatesAndNegativeOwnershipEvidence/application_list-get_version_fence_changed`；原 failed snapshot 与 post-STOP 只读快照均保留 |
| F9 | Kubernetes Namespace 审计 objectRef.namespace 可等于 Namespace 自身名称，原身份拼接漏计 4 条 Namespace 写入 | 本次原始 14 条受测资源写入逐条确认均由 Argo 发起；原执行器只匹配其中 10 条，属审计覆盖缺口 | `7918c74`：仅将 Namespace 审计身份归一到集群作用域；与 name 冲突的 namespace 拒绝；未重新执行现场 | `TestAuditRejectsSideChannelWritesAndLostHistory/namespace-wire-side-channel`，覆盖 Argo、旁路写入、冲突字段和其他 namespaced resource |

F1–F7 的定向回归与完整 `task quality` 已通过；F9（`7918c74`）的定向回归和完整质量检查也已通过，但仅为本地修复，没有新现场执行。

## 当前现场：真实 mutation 后 STOP

- 执行实现：`b300d7a91fe9bdceed7bd69487818d78deb8c66d`；binary SHA256：
  `8ac3d907c8e124173f3eb9c1b65f856c6169f5e04346df25d0ca80e4723379df`。
- Plan SHA256：`a89a31766d6c3afea25995b80a2947c91ee83a932e8f4ff2a5da3d96b4e75f95`。
- OT-1 UID：`30a366bd-41f9-48b0-9330-70dffaf62662`；2026-09-27 12:50:33–12:56:32 UTC，
  exit 2，停于第 8 阶段 `MIXED_OBSERVABILITY_STRICT_RESTORED`。
- **7/29 个 checkpoint 通过**：包含基线 ownership、独立 Gate-B、旧 owner orphan release、
  secrets 的 strict refusal / window adoption / strict restore，以及 observability 的 strict
  refusal / window adoption。第 8 阶段已提交 strict restore 和 sync，但拒绝其跨版本快照，
  因而没有通过 checkpoint。
- 3 条 Git 请求（一次 publish + 本地 fetch/checkout）及 13 条受保护 Application 请求
  均 exit 0。审计区间 1114 条事件，13 条 kubectl 写入与请求对应；独立核对原始审计，
  14 条受测对象写入均来自 Argo，Namespace 的 4 条包含在内。F9 修复不追溯改变原 binary。
- 只读 post-STOP 检查：13 UID 与 semantic 不变；owners 为 secrets=3、observability=4、
  old-stale=6。旧 App 已不存在，storage owner 未创建；两个现存新 owner 均 strict、
  idle/Synced/Healthy，**开放窗口 0**。新快照经既有 Assess 检查通过，只是当前状态观察，
  不覆盖原 STOP，不推进 checkpoint。
- Git 留在 `f49e902217d64eb1985c51eb82743ddfafd544c3` 的 detached mixed 布局。
  未执行 partial rollback、完整 forward/reverse 或后续 Gate-B，不能声称完整 Atlas 接管。
- 私有 authority bundle：`.state/latest/ot1/`，由 `RETENTION.json` 标记冻结；包含原
  executable、plan、全部 intent/result/checkpoint、失败快照、post-STOP、冻结审计及摘要。
  原集群、原 runtime audit 和 STOP 锁保留；默认 kubeconfig 未变。
- 冻结 manifest SHA256：`de618cc101c4e60b49441484c500c246b611ef798adeec420bd726887ba06274`；原始与派生证据共 792 个文件，未包含私钥或 kubeconfig 文件。
- 下一执行决定必须针对这个已改变的混合状态。普通 run 只接受 baseline，不能清锁后从头
  重放；是否引入有界只读重采与显式 continuation，须先审查，不能默默把失败变成成功。

## 保留规则

- 本地开发和只读预检使用 `.state/latest/ot1/`，失败明确且进入回归后可替换其临时输出。
- mutation 前 STOP 经请求 intent、Git source 与审计确认后，只保留原因、影响、plan SHA、
  fix 与测试。旧 plan、binary、快照、日志和已核对的 stale lock 可清理。
- 任何重要 live mutation、后续 Git 阶段发布、凭据/Trust Root 改动、影响不明或未解决事故，
  都冻结完整 authority evidence；不能以“开发模式”自动续跑、回滚或清理。
- 最终正式 Gate PASS 保存完整不可变 bundle。中间报告不再成为长期独立文档。
