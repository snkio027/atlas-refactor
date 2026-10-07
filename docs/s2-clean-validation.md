# S2 clean validation — atlas-s2-r1

2026-10-03。结果：**D1 PASS；S2 consumer Gate STOP；Runtime UNPROVEN**。
这份记录对应一次新实例验证，不能与旧实例的部分结果拼接为完整验收。
PR #9 保持 Draft；本地修复不构成再次执行批准。

## 执行绑定

| 项目 | 本次实际值 |
| --- | --- |
| Implementation | `f432cfb693cd9047c99e3406252a5fcc7226311e` |
| S2 executable SHA256 | `98e83b60e325800b9ff186f721b173581495d25995c7dd0f317250cb3553714e` |
| Review plan SHA256 | `2a7e8f12a662b8e740b2370f340f8f037651e40897aaf9e87e057dc955a47c0b` |
| Derived S2 plan SHA256 | `155003503498e2466fc24bbd547dde17a64a7968284c7728be28976b0869e3ac` |
| D1 package | `v0.1.0-d1.4`，archive checksum 与 GitHub attestation 已核验 |
| D1 plan SHA256 | `1fd5bda54cf1682617868a8df2a8f4a1200b2150e419de6cc493d3daa8c44094` |
| Installation ID | `429e00971e9cd2037e541e4d16a8efdb` |
| Cluster / deployment branch | `atlas-s2-r1` / `atlas-s2-r1` |
| Cluster UID | `218cfdc7-22d6-41ce-be53-153879729b5c` |
| Topology / loopback ports | 1 control-plane + gateway / compute / data；18080 / 18443 |
| D1 FullCommit | `9a94140c1aeecab67d144094a62f8f7775a2f2b3` |
| Infrastructure commit | `a6905a6c3b95a569569df57d4ab02b99b7b4101d` |
| Consumer commit | `b8b72de31758995895b402014647f152eeb59e13` |

冻结实现的 [Quality run 36855041208](https://github.com/snkio027/atlas-refactor/actions/runs/36855041208)
在首次外部写入前已经 success。新实例没有复用旧 kubeconfig、Trust Root、凭据、密文或运行记录。
新 Trust Root 完成独立备份与读回验证；隔离级别是这次单独批准的
`same-host-development-exception`，不是物理隔离备份或灾难恢复证明。

## 实际结果

| 阶段 | 结果 |
| --- | --- |
| 一次 D1 install | PASS / exit 0，约 15 分 24 秒；有限 Bootstrap handoff，26 个 Application 收敛 |
| 一次 D1 read-only verify | PASS / exit 0 |
| D1 基线 | 352 个持久资源身份、tracking/SSA owner，13 份冻结文件和 Bootstrap authority 记录 |
| 隔离 SeaweedFS 4.47 fixture | PASS；对象读写、跨桶与管理权限负例，清理该临时容器 |
| 唯一 S2 plan 派生 | 全部固定相等条件、新 UID / certificate / D1 parent 关系通过 |
| Infrastructure compile ×2 | 字节一致，实际文件摘要重算相符，363 个唯一 owner 资源 |
| 四节点镜像 / 独立 Binding 凭据 | 完成；没有替换原 D1 identity |
| Infrastructure publication / Gate | PASS；原 parent 的 exact lease 发布并取得 receipt |
| Consumer publication | 完成；以上一阶段为 exact parent，取得 receipt |
| Consumer Gate | **STOP**：四个 HTTPRoute 持续 OutOfSync |
| Metrics convergence / live functional probe | 未执行 |
| Repeat compile / publish / successful deploy | 未执行；只能在完整 PASS 后执行 |
| S2 final.json | 不存在；原 terminal.json 为 STOP，deploy exit 1 |

S2 deploy 只调用一次，12:06:06–12:19:08 UTC。确认持续不收敛后取消了唯一精确匹配的进程，
没有等待至 15 分钟 Gate 上限。原程序在取消等待时使用通用错误 `S2 convergence timed out`；
外层执行记录为 `timeout=false`，另有 operator-stop 原因。不能把它描述成真实超时或已通过的 Gate。

两个 WebService 的 Deployment/Service 已 Synced、Pod Healthy，但其 HTTPRoute 未 Synced，
所以不能由 Pod 健康推导 Runtime PASS。没有执行真实 HTTPS→S3 写入 probe，也没有重复部署。
危险的 CreateBucket/DeleteBucket/PutBucketCORS 负例仅在合成 fixture 执行。

## 原因与编译器修正

锁定 Gateway API v1.6.1 在 API 中补入以下字段；S2 compiler 原来省略它们：

| 位置 | API 的实际默认值 |
| --- | --- |
| `parentRefs[]` | `group: gateway.networking.k8s.io`、`kind: Gateway` |
| HTTPS `backendRefs[]` | `group: ""`、`kind: Service`、`weight: 1` |
| HTTP redirect `rules[].matches` | `path.type: PathPrefix`、`path.value: /` |

四个路由在 Argo 报告成功同步后仍为 OutOfSync；逐项 Git/live spec 比对确认这些差异。
修复只让 compiler 显式输出同一默认语义，不改变 hostname、listener、backend、端口、资源 identity
或授权。没有增加 ignoreDifferences、放宽 Gate、手工 patch live spec/tracking 或创建 continuation。

`TestConsumerRoutesMatchGatewayAPIDefaults` 使用本次 API 返回的四个路由 spec 作为非敏感 fixture，
覆盖 bound/unbound 的 HTTP/HTTPS：旧实现四项全部失败，修复后通过。
这证明生成结果与已观测 API 表示一致；**修正后的 Argo/runtime 收敛仍待新的执行决定验证**。

后续根因修复把「schema 合法」与「CRD 默认值不改变新增内容」分开检查，并在 infrastructure
编译／plan 阶段提前验证不依赖密文的 consumer。schema 绑定不可变 D1 base，历史未改动子树
保持原样；缺失默认值、版本错配或 consumer 编译错误在返回输出前拒绝。
实现边界与回归见 [编译时拒绝 CRD 默认值遗漏](s2-workloads.md#编译时拒绝-crd-默认值遗漏)。
这项源代码修复没有重跑本次执行，也不改变上表的 STOP 或任何历史 plan/证据。

辅助证据采集曾因把 JSON 多文档流作为单个 JSON 解码而退出；修正本地读取器后完成基线。
这个辅助问题没有改变产品 executable、计划或外部状态，D1 install/verify 未重跑，S2 当时尚未生成 plan。
原日志保留；不将其隐藏为无瑕疵的操作过程。

## 现场保留与未证明事项

STOP 后只读复核：四节点 Ready，352 个既有持久资源 UID/owner、13 份 D1 冻结文件以及
Identity/Latch/Receipt/External Root 不变。部署分支保留 consumer commit；原 intent、receipt、
terminal、确切二进制/plan、新实例凭据与密钥备份、元数据 audit 完整保留在私有实例目录。
原 `atlas-d1-r2` 与其 STOP/分支不变；默认 kubeconfig、hosts 未改写。
公开仓库只保存脱敏结论和非敏感路由 fixture，不附入凭据、私钥或其关联摘要。

未证明：真实 HTTPS→S3、unbound 网络拒绝、跨项目 Secret 授权拒绝、metrics discovery、S2 后 D1
功能回归，以及成功后的幂等/零写入审计 Gate。禁止把本地修复或 fixture PASS 当成这些 Gate 的替代。
本次仍是同机、有公开镜像缓存的开发实例，不构成独立干净机器、HA 或强多租户生产保证。


## r2：metrics 读取通路失败（53634fe）

`atlas-s2-r2` 的唯一 S2 attempt 使用 `53634feaf85b3762fb06e7479f2bc18c276151b7`，
plan `65ac3a6f8788ab5f33d9efea869b5dd6c0989f3033934d41fea3c453dd69874b`，
cluster UID `495d7cba-cf78-4c0b-b7ce-70ab935122ba`。
D1 曾在旧集群并行运行时失败；经用户批准删除旧集群后重试通过，因此 r2 不能声称完整首装单次 PASS。
原始 D1 STOP 与重试记录分别保留。

S2 infrastructure `bffe2c5e73637647ebb6e1467b7c711821aa9b49` 和 consumer
`b0076a2361edc1df63046ae0674a1a1ffb4e9c8e` 均成功发布并通过 Gate。
CRD default-stability 修正生效；Project / Workload / Binding VERIFIED。
随后 metrics 查询失败，deploy exit 1，Runtime UNPROVEN；未执行功能 probe 或幂等验收。

根因是 `kubectl get --raw .../services/.../proxy/api/v1/query` 的 API Service proxy
连接来自 Cilium `remote-node` identity，不满足冻结的 monitoring-ingress 策略。
现场捕获到访问 Prometheus 9090 的 Policy denied SYN；同一查询经临时 loopback
port-forward 返回成功且目标 `up=1`。这是观察通路不符合平台网络边界，不是 scrape 未就绪。

修正为精确绑定实例 kubeconfig/context、校验工具后，对锁定 Prometheus Service 建立
临时 `127.0.0.1` port-forward；只发 GET，禁止环境代理/重定向，限制响应大小与等待时间。
同一私有 helper 也替换 D1 S3 兼容性读取的重复转发代码，持续排空 stdout 并在所有退出路径
取消和回收子进程。它不修改 D1 release、NetworkPolicy、GitOps 输出、权限、Gate 或超时上限。
普通 Observe 仍不启动转发；转发仅属于已批准 deploy/probe 的读通路。

回归：`TestServiceForwardProcess` 用真实子进程验证输出管道持续排空、HTTP 查询、提前退出、
错误地址/端口、取消与重复清理；`TestMetricGate` 验证全部副本 up、缺失/失败抓取、HTTP/JSON
失败、重定向与响应上限；未知连接失败保持 fatal。旧 STOP、发布 intent/receipt、私有备份和
完整证据不改写；用户已另行批准修正、清理旧集群并以新实例重新验证。


## r3：AppProject 权限与依赖发布竞争（2b42ce7）

`atlas-s2-r3` 在按用户授权删除全部旧本机集群后创建。实现 `2b42ce7964c357a2bd500a2855416983cf0079f6`，
cluster UID `323b905f-c267-4f5a-a68f-e7a5fda5af55`，D1 FullCommit
`f4793802f21763b717cadafb09202d843288c730`。D1 首次安装与独立验证 PASS，四节点 Ready；
S2 sole plan `3c4c37e99c19b8cdcd7dee850c9ead4f18803671eec944de024bc8e3556180ed` 的
infrastructure `61bd0286ca001dd75cd637900911f0fcc934ac77` 发布后 Gate STOP。
consumer 未发布；metrics、HTTPS→Web→S3 和幂等验收未执行。

2026-10-03 22:29:11 UTC，secrets-controller 已尝试同步 demo 中的 Role/RoleBinding，
明确报 namespace demo is not permitted。22:30:24 新 Project Application 被创建并进入 Unknown；
22:31:10 project-bootstrap 才应用新增 destination。22:31:42 S2 STOP；22:32:38 Argo
自行把 Project 收敛为 Synced/Healthy。后续自然收敛不改写原 attempt 失败。
Project 原始 condition.message 未被旧程序保存，不能声称拥有其具体错误原文。
同 namespace 的 RBAC 拒绝、应用时间线和冻结 Git 树共同证明了授权发布竞争。

根因：一个 Git commit 同时开放权限、namespace 和独立自动调谐的使用者；sync-wave
只约束各自同步，不构成这些 Application 之间的完成屏障。依赖升级不改变这项协议缺口。
另一个确定问题是 Gate 遍历 map 后立即返回首个未通过应用，混合错误/进度下结果不确定。

修正见 [ADR-0016](adr/0016-s2-publication-prerequisites.md)：固定四阶段与逐阶段真实只读门禁、
完整 predecessor receipt 链、UNKNOWN 不放宽、全快照 fatal 优先、失败诊断私有保留。
回归 `TestPublicationPrerequisitesUnderAdversarialReconcileOrder` 验证任一新可见资源的权限与
跨应用 namespace 前置条件来自前一个已通过阶段；真实 Kustomize 覆盖四阶段。
`TestPublicationCannotSkipPrerequisiteReceipts` / `TestPublicationRejectsBrokenReceiptChain`
覆盖缺失、乱序与六类 receipt 漂移；`TestApplicationGateFatalAlwaysWinsOverProgressAndMissing`
覆盖反复乱序快照和诊断不泄漏；consumer preflight 覆盖所有首次可发布阶段。

旧 STOP、分支、集群、Trust Root 和完整私有证据不改动。该修正尚无新实例 Runtime PASS；
需要新实现与四阶段计划的独立执行决定，不能清锁续跑 r3。

## r4：创建与首次 controller 状态之间的观察窗口（2aec769）

按精确总计划 `98579b7b7dcefffc825abdf7b33649a0ab8adb14410061494aab88f306d4ec7c`
删除 r3 四节点并保留历史证据后，创建 `atlas-s2-r4`，cluster UID
`895c2b95-4624-4e08-a63b-c5fb20834d6e`。实现
`2aec76954585277d242a9a8cb85740bea6ec5d33`，唯一 S2 plan
`a57c4ff1bbd62c606197dc09ce36450a9987baa7f39918441a0c5a092b62c96a`。

D1 从零安装、独立 verify、352 项身份/owner 基线、13 个冻结文件、隔离 provider fixture、
三个阶段双编译及 consumer preflight 均 PASS。D1 FullCommit
`a094f85a8de392d32da705b400924eb4b3ece216`；permissions
`de4a523eeb0383a77136acf7b7104d83c5117513` 发布并通过 Gate；project
`fd3c34d06ccce2897f4677a38bd47f59a5da4053` 发布后 STOP。

2026-10-07 16:12:54.434 UTC，审计记录 Project Application 创建成功（201）。
16:12:55.395 S2 已因 `REVISION_EVIDENCE_MISSING / SYNC_UNKNOWN / HEALTH_UNKNOWN`
退出；下一条 controller update 为 16:12:55.867。STOP fact 记录 generation=1、
精确 spec 摘要、正确 tracking/SSA、UID 与 RV，未有 revision、sync、health 或 operation。
相同 UID 随后自然达到 Synced/Healthy，不能改写原 attempt 的 STOP。

根因是把新对象的创建成功与异步 controller 首次写入 status 视为同一步。
历史 fc8ce63 修复边界见 ADR-0016：当前 receipted 阶段首次声明且身份/ownership 完整的空 status
仅允许 Pending；UNKNOWN facts 保留，不放行写入或降低终态要求。回归覆盖
`TestNewApplicationFirstObservationWaitsWithoutInventingProof`、
`TestFirstObservationCannotMaskUnknownDriftOrPreviousState`、
`TestFirstObservationDisabledAfterGateStopOrCompletion`、
`TestFirstObservationRequiresBoundCurrentPublication` 和混合 fatal 优先。

infrastructure/consumer 未发布，S2 metrics、功能 probe、final 和重复部署未执行。
STOP 后只读检查：四节点 Ready，352 项旧资源身份/owner、13 个冻结 D1 文件及 Bootstrap
authority 不变。保留 r4 集群、分支、密钥备份和完整 evidence；不清锁续跑。
当前 S2 Runtime 仍未通过。上述本地修复不构成新执行授权。

## r4 后的观察协议修正（Proposed ADR-0017，未执行新 runtime）

用户授权结合工程实践重新设计，改动限于 S2 观察协议与回归，详见
[ADR-0017](adr/0017-s2-rollout-gates.md)。r4 的 STOP、已发布 Git、密钥备份和运行现场不改写。
原 fc8ce63 的 generation=1/空 status/reason 三元组特判已替换，四阶段编译及外部写入保持原范围。

| 问题 | 修正 / 长期回归 |
| --- | --- |
| 创建与首次 comparison 被视为原子操作 | 发布契约限定初始化 Waiting；`TestRolloutAllowsSkippedRepeatedAndPartialInitialObservations` |
| 新 UID 未跨轮询/阶段保留 | 会话首次 UID + immutable phase Gate；`TestRolloutRejectsIdentityLossAndComparisonRegression`、`TestRolloutContractUsesIntroductionAndPriorGateUIDs` |
| 旧 target 更新被一律误判或无限等待 | 仅允许 exact predecessor → target；`TestRolloutTargetSpecConvergenceIsMonotoneAndPlanBounded`、`TestResourceConvergenceRejectsThirdContentAndBackwardTransition` |
| 新 spec 仍使用旧 source comparison | 核对 Argo comparedTo；`TestRolloutCannotUseStaleComparisonAfterSourceChange` |
| receipt / Gate 绑定或只读取消边界不完整 | `TestRolloutContractRejectsCurrentReceiptDrift`、`TestRolloutContractRequiresCompletePredecessorEvidence`、`TestReadPollCancellationAndDeadlineNeverAllowLateSuccess` |

契约测试使用真正的本地 compiler 输出及锁定 resource model；API fixture 与时序测试均不访问
现有集群。`FuzzRolloutCannotInventApplicationProof` 验证删减证据不能构造 Ready。
本地通过不构成 S2 Runtime PASS；下一次验收必须使用新实现与新精确计划。

本轮本地验证：Go 1.27.1，完整 `task quality` PASS（race/vet、锁定 kubectl 隔离 API fixture、
真实 Helm/Kustomize、S1/D1/OT-1 契约）；256 种删减证据组合及 10 秒 fuzz PASS。
未执行新的 live mutation；r4 的 157 份冻结证据与 2 份备份均通过原摘要复核。


## r5：四阶段通过，Probe 入口单次观察 STOP（fc59e33）

2026-10-08，按独立批准的总计划
`1971bea222db964f31b99e487044b98ab18086a40920d6792daf27295c168ce6`
删除精确 r4 四节点并保留其证据/Git/备份，创建 `atlas-s2-r5`。实现
`fc59e33e8383968db383614a8b358228a9203882`，cluster UID
`fbb4877b-6de7-4cd9-9c15-69935fcfb919`；唯一 S2 plan
`0744bd4b9276fbd11b290f5b4bb3cb9868b5558d64fca398386c7e30d47352e7`。

D1 首装及独立 verify、352 项旧身份/owner 基线、13 份冻结文件、隔离 provider fixture、
三阶段双编译和 consumer preflight 全部 PASS。permissions、project、infrastructure、consumer
均完成 publication/Gate；Project / Workload / Binding VERIFIED。r4 创建窗口修正已在本轮
实际走通，未执行手工 refresh、spec/tracking patch、回滚或 continuation。

S2 在 2026-10-07 22:56:05 UTC 返回 exit 1（非超时）：
`observation changed during capture: argoproj.io/Application/argocd/envoy-gateway`。
metrics 等待已经返回；Probe 入口单次 Observe 将 closing proof 变化的 Pending 直接返回，
没有复用发布 Gate 的有界只读等待。审计中该 S2 区间 admin pods/exec 和 SubjectAccessReview
均为 0；结合后观察之前必有两类 exec 的控制流，确认 STOP 在功能写入前。原双读 raw pair
未保存，不能宣称已知道 envoy-gateway 的具体变化字段，也不放宽 semantic proof。

Runtime UNPROVEN；final.json 不存在；HTTPS→S3、隔离功能及成功后的幂等未执行。
STOP 后 4 节点 Ready、30 Applications；旧持久身份/owner、冻结 D1 文件、Bootstrap authority、
历史分支和主机默认配置均未变。170 份私有证据及 2 份独立备份冻结保留。r5 的开发备份
例外是 same-host，不构成物理隔离。公开报告不附入凭据、私钥、kubeconfig 或其关联摘要。

修正限定于 [ADR-0017 的 Probe 读边界](adr/0017-s2-rollout-gates.md)：同一 contract/UID 会话、
同一个 15 分钟总 deadline、前后只读轮询、期间功能操作一次；明确错误、取消和 deadline
立即失败。共享观察记录器在失败时保存本次 Rejected，而非上一份 Ready。
`TestProbeClosingChangesRetryReadsNotFunctionalEffects` 使用真实 semantic proof 和生产轮询器
复现前/后采集变化；`TestProbeObservationFailuresStopWithoutRepeatingEffects` 覆盖双侧六类
fatal 并检查失败报告；`TestProbeFunctionalFailureNeverRetriesEvenIfPending` 防止重复功能写入；
`TestProbeDeadlineSpansBothReadsAndFunctionalPass` 用 Go 虚拟时钟证明后观察不重置预算；
取消及 UID inventory 回归也必须通过。fixture 不访问任何现有集群，不构成 Runtime PASS。
