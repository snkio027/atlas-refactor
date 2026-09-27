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
| F8 | RV bookkeeping was mistaken for changed proof facts during List/GET | Historical mutation/STOP remains immutable | ADR-0010: canonical double reads retain identity/content/ownership/health/operation; mutation fences stay exact | `TestSemanticProofPermitsBookkeepingButRejectsControlChanges`, `TestCollectSemanticChangesAreUnknownWithoutResampling` |
| F9 | Kubernetes Namespace 审计 objectRef.namespace 可等于 Namespace 自身名称，原身份拼接漏计 4 条 Namespace 写入 | 本次原始 14 条受测资源写入逐条确认均由 Argo 发起；原执行器只匹配其中 10 条，属审计覆盖缺口 | `7918c74`：仅将 Namespace 审计身份归一到集群作用域；与 name 冲突的 namespace 拒绝；未重新执行现场 | `TestAuditRejectsSideChannelWritesAndLostHistory/namespace-wire-side-channel`，覆盖 Argo、旁路写入、冲突字段和其他 namespaced resource |
| F10 | 初次 API probe 在已有 26 个 targets 全 up 时提前结束等待，遗漏尚未被 Prometheus 发现的 SeaweedFS；随后自行发现第 27 个 target | 没有 ownership mutation；S3 测试对象已删除，Grafana 结果保留；仅补完监测验收，无集群修补 | 本地 probe 等待必需 storage target；可复用离线判定在 `experiments/foundation-ownership/platform_readiness.py` | `MonitoringReadinessTests`：缺失/重复/错 namespace/未知或 down 拒绝，27-target 真实快照通过 |
| F11 | Transitional gate imposed a global exact-revision barrier; six unchanged Apps delayed SOURCE_RELEASED | Git detach and orphan DELETE completed; 13 identities/content/tracking preserved; no window | ADR-0010: plan-proven source equivalence in transitions only; owners/platform-control/Atlas gates stay exact; 300 seconds unchanged | `TestTransitionalRevisionEquivalence`, `TestEquivalenceDoesNotExpandToOlderGitEpoch`, `TestLocalSourceClosureRejectsRemoteOrParentInput` |
| F12 | Operation completion preceded its post-sync comparison; readiness conflated two controller loops | Current STOP has tracking 3/4/6 and only observability window open; 10 App requests, 10 Argo scope writes, zero Git publication | Separate operation proof from comparison; require reconciledAt >= finishedAt on both checkpoint reads, wait only for known progress; strict refusal unchanged | `TestSuccessfulOperationWaitsForComparisonWithoutWeakeningCheckpoint`, `TestPostOperationComparisonFreshness`; one-off recovery starts with strict closure at index 7 |

| F13 | Full Atlas Gate used repository revision equality as leaf desired-state identity: five healthy unchanged Apps still reported b0768e7 after 220a113 | Only platform-control child projection changed; source closures for the five Apps were byte-identical. All 13 resources retained UID/content/SSA under the restored source; no window; STOP at 12 before full capture/Gate-B | ADR-0011: full gates reuse bounded preceding-planned source equivalence; critical owners remain exact. Schema-3 post-Receipt ADOPTED verifies durable authority/Seed ownership independently of rollout/runtime; initial adoption remains strict. No timeout/refresh/recovery change | `TestFullGateDesiredIdentity`, `TestInitialBaselineRejectsEarlierRevision`, `TestDurableHandoffDoesNotQueryBranchHeadOrLeafRollout`, `TestSchema3AuthorityDamageStillFailsClosed`, `TestFirstDevelopmentHandoffStillNeedsExactLeafRevision` |

F1–F7 与 F9 的定向回归和完整质量检查已通过；本轮 runtime 使用含 F9 修正的实现，但没有新增 Namespace 写入案例。F10/F11 的回归记录本轮新发现，未改变原执行器预算或 STOP 规则。

## 历史现场：首次 ownership mutation 后 STOP

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
- 私有 authority bundle：`.state/authority/ot1-a89a3176/`，由 `RETENTION.json` 标记冻结；包含原
  executable、plan、全部 intent/result/checkpoint、失败快照、post-STOP、冻结审计及摘要。
  原 runtime audit 和 STOP 锁保留；原集群后来按新的用户授权删除，见下文；默认 kubeconfig 未变。
- 冻结 manifest SHA256：`de618cc101c4e60b49441484c500c246b611ef798adeec420bd726887ba06274`；原始与派生证据共 792 个文件，未包含私钥或 kubeconfig 文件。
- 当时的混合状态不能通过清锁后从 baseline 重放来恢复。后续用户选择删除旧集群并重新
  创建独立新 UID；原失败与完整证据不因重建而改变。

## 当前现场：新集群重建通过，SOURCE_RELEASED 超时

用户再次授权清理旧集群并用新集群验证。旧 UID `30a366bd-41f9-48b0-9330-70dffaf62662`
对应的四个 Kind 节点已删除；旧 authority bundle 校验后迁至
`.state/authority/ot1-a89a3176/`，792 个文件和原 manifest 摘要不变。旧 Trust Root 备份保留。

本次新建同名四节点 `atlas-refactor-test-ot1`，UID
`b886f730-ea3b-44b9-a904-8bd55ac345f2`，没有导入旧 identity、Receipt、密钥或凭据。
沿用已授权 OT-1 开发备份例外，新私钥另存 w1 的
`atlas-refactor-ot1/rebuild-20260927`；该同机备份仍不构成物理隔离。

- 执行实现 `8bde336fa390e6d9cd16384adb6e89c264a10919`；
  binary SHA256 `1f0a6798dd0840fa82cf4c0463dcf515ec9d946011e8302cba86b0bdcb4d4296`。
- 新 core `df8eb53`、独立 controller `5f6bb26`、完整平台基线
  `e462c253c075d5e2c9b6cabd26e6fb7c1ad644d9` 均追加到实验 Git 历史，没有强推。
  184-resource core / 346-resource 完整旧布局本地检查通过，固定镜像在四节点逐一验证。
- 从零 Bootstrap：四节点 Ready、ADOPTED、PVC/HTTPS 通过；
  第二次 apply exit 0、kubectl mutation delta 0、身份不变。
- 完整平台：24 个 App 同 SHA、idle/Synced/Healthy；S3 签名读写、HEAD、预签名、
  分片上传与删除通过，匿名和跨 bucket 请求均 403。Grafana 登录、26 看板和数据源通过；
  Prometheus 最终 27 targets 全部 up、31 rule groups、4 节点指标齐全；
  Alertmanager 本地测试告警触发/恢复通过，未验证外部通知。
  F10 是发现延迟下的本地 probe 提前退出，未修改集群来修补它。
- 新 plan `b195dc63e4ab2910bf14c2f351d7da6d7cfac91d9363ba08ecc1ff038f3fba70`；
  29 阶段、13 对象、权限、300 秒单阶段预算均未放宽。
  2026-09-27 13:50:08–13:55:48 UTC 执行，exit 2，**1/29 checkpoint 通过**。
  基线独立 Gate-B 为 VERIFIED；重复 apply 的 74 条请求只读，前后 identity digest 相同。
- 第 2 阶段 `SOURCE_RELEASED` 发布 `b0768e74b0475af7e344e2905d56aed0cbdd2ff4`，
  父 App 确认后于 13:54:25 UTC 非级联删除旧 owner。阶段截止时 cilium、envoy-gateway、
  monitoring-crds、project-bootstrap、secrets-controller、storage-monitoring 六个未改动 App
  仍观察旧 SHA，readiness precheck 未完成；300 秒到期 STOP。
- 3 条 Git 请求及 1 条受保护 Application DELETE 均 exit 0。
  独立审计区间 1008 个 audit ID，唯一 kubectl 写入为该 DELETE；
  **13 个受测对象写入为 0，tracking 接管为 0**。
  F9 的 Namespace 归一规则用于独立核对，但没有新增 Namespace 写入案例。
- STOP 后只读观察：23 个现存 App 随后全部确认阶段 SHA、idle/Synced/Healthy，
  四个 foundation owner App 均不存在，开放窗口 0。
  现有 Assess 对 SOURCE_RELEASED 当前状态为 ownership VERIFIED、Atlas NOT_APPLICABLE；
  13 UID/semantic/旧 tracking 与基线保持，identity digest 不变。
  **不覆盖原超时、不推进 checkpoint，也不宣称 detached 平台已完成 Atlas handoff。**
- 新集群、活跃 runtime 和 STOP lock 保留；默认 kubeconfig 摘要仍为
  `862e14c996585bf3b2c9f4cb4026bffdd56f614faea8c0a85b099b98949c651d`。
  私有 runtime 位于 `.state/latest/ot1/source/.state/development/atlas-refactor-test-ot1/repo/`。
- 完整 authority bundle：`.state/authority/ot1-b195dc63/`，85 个文件，69,084,766 bytes；
  manifest SHA256 `b48895b8a871c0d685b66a7bfc5fe829f7f7408d8e954542d8c2fa06bff1eb74`。
  含 executable、plan、Git bundle、原 attempt、原始审计、前后快照及准备证据；
  不含私钥、明文凭据、kubeconfig 或重复镜像缓存。

后续需要审查 Git 比较就绪预算与已 release 状态的恢复/续行契约。
本轮没有延长运行中期限、恢复旧 owner、清锁或重试 mutation；完整正反向链仍 NOT PROVEN。

## 保留规则

- 本地开发和只读预检使用 `.state/latest/ot1/`，失败明确且进入回归后可替换其临时输出。
- mutation 前 STOP 经请求 intent、Git source 与审计确认后，只保留原因、影响、plan SHA、
  fix 与测试。旧 plan、binary、快照、日志和已核对的 stale lock 可清理。
- 任何重要 live mutation、后续 Git 阶段发布、凭据/Trust Root 改动、影响不明或未解决事故，
  都冻结完整 authority evidence；不能以“开发模式”自动续跑、回滚或清理。
- 最终正式 Gate PASS 保存完整不可变 bundle。中间报告不再成为长期独立文档。

F8/F11 have a Proposed semantic-evidence correction. The exact 53c82ce continuation was executed once and stopped on F12; original STOP outcomes remain immutable. No generic continuation/recovery mechanism is added.

## Current exact continuation: F12 after live mutation

- Executed implementation: 53c82cea9aa275c6e1e84f05a7e9351aa9da8870.
- Plan: 3e8db44c2a6c56b52ff7268be628bb34089dfd7227fe996f5912b6e6cb92ed47.
- Same cluster UID: b886f730-ea3b-44b9-a904-8bd55ac345f2.
- 2026-09-27 14:55:41–14:56:57 UTC, exit 2. Fresh anchor and lock handoff passed; indices 2–5 produced VERIFIED checkpoints. Index 6 MIXED_OBSERVABILITY_ADOPTED_WINDOW remains STOP.
- Post-STOP diagnostic reads: 25 Apps idle/Synced/Healthy at b0768e7; all 13 resource UIDs/content and Argo SSA preserved; tracking secrets=3, observability=4, old-stale=6. One observability window remains open. Bootstrap records and four projects unchanged. No retroactive checkpoint.
- Audit: 10 bounded kubectl owner-App requests and 10 scope writes, all by Argo; no Git publication. One accepted double-read proof contained benign RV churn. Transitional prior-revision equivalence was not exercised by a new Git publication in this run.
- Private authority bundle: .state/authority/ot1-continuation-3e8db44c; 73 files, 51,388,491 bytes. Manifest SHA256: 57a0e0f74aaa11220a0f37625da0f2b0035060b5faf6e66d6259355534a7a806. Predecessor bundle is referenced, not copied.
- Successor STOP lock preserved: f3c6b273117d2104869a43ef3f017ebedcba5135657edc3e75d03c78c6c95f47. No automatic rollback, window closure, retry or rebuild.
- The F12 operation/comparison split has regression coverage, including the post-operation freshness fence. It does not authorize replaying the old SOURCE_RELEASED entry against this new state. No arbitrary-stage resume or additional evidence model is introduced. Full OT-1/reattach/rollback and remote CI/release gates remain unproven.

## F12 root-fix execution and full-gate STOP

- Implementation: 8994edf7aee7065142012604c7f38b9eef8d9c46; canonical plan: 8aace8f17eeed6736700f07b3e9c9c77258ba2c4f35a8790b45201a636f8b0a2.
- Same cluster b886f730-ea3b-44b9-a904-8bd55ac345f2. Fresh anchor and lock handoff passed; first mutation closed observability strict. Original indices 7..11 produced five VERIFIED checkpoints. Successful post-operation comparison was proven at 7/10/11; strict refusal at 9 remained valid.
- Index 12 MIXED_ROLLBACK_VERIFIED published 220a113, then stopped at 2026-09-27 15:47:04 UTC / exit 2 / 300 seconds before full capture and Gate-B. Five healthy Apps still reported b0768e7: cilium, envoy-gateway, monitoring-operator, platform-credentials, secrets-crds.
- Later independent reads showed all 24 Apps at 220a113 and Synced/Healthy. They do not rewrite STOP or establish Gate-B. No refresh/retry/rollback/rebuild or additional continuation was executed.
- Four nodes; four unchanged AppProjects and Bootstrap records; all 13 resource UIDs/content/Argo SSA preserved; tracking capability-foundation=13; zero transfer windows. Default kubeconfig unchanged.
- Ten bounded kubectl mutations, one Git publication (three Git commands), all request exits zero. All 30 scoped resource writes in the retained audit were by Argo.
- Immutable private bundle: .state/authority/ot1-f12-8aace8f1; 86 files, 68,659,530 bytes; manifest 3ada495c8740562e9b06fe4b855b245b6e062c2d48a9ec9c9f6c5c40d2d96f18. Predecessors are referenced rather than copied.
- STOP lock retained: d7e4a9d78fc284f55b1e34b3aafe9b67998fd7789eb0201814cec2c280a7b1d2. This one-off F12 executable cannot re-enter the new STOP.
- Locked task quality and tagged race/vet passed. F12 has real runtime evidence; full mixed rollback Gate-B, forward/reverse completion and final S1 acceptance remain unproven. Remote CI/release gates are separate.
