# ADR-0009：S1 Observation / Evidence 与 Foundation Ownership Rehearsal

- Status: Proposed
- Date: 2026-09-27
- Parent: [ADR-0007](0007-platform-contract-hardening.md)、[ADR-0008](0008-ownership-transfer-probe.md)
- Scope: 仅 atlas-refactor-test-ot1；13-object、1→3→1 实验，不批准 dev02 迁移
- Implementation: S1 干净平台基线、partial rollback、full forward/reverse 和最终 Gate-B 已通过；功能开发冻结。历史 STOP 全部保留，最终 runtime 绑定与审核边界见 [最终验证](../s1-final-validation.md)。此记录保持 Proposed。

## 决策目标与权责

沿用原 Atlas Architecture v1.0.2 的有限 Bootstrap、External Root 和 GitOps authority。
Argo 是这 13 个对象内容与 tracking 的唯一写入者；ceremony 仅管理精确列出的
Application 生命周期和模式，并观察结果。普通 Bootstrap/select/render 不获得迁移、
retirement、恢复权限。Helm 仍只渲染；canonical AppProject 与权限集不变。

OT-0 证明 Argo CD 3.5.1、annotation tracking、SSA 下受限窗口可双向转移两对象。
OT-1 分开证明多 owner 语义与完整 Atlas 不变量。ADR-0008/0009 均保持 Proposed；
通过实验不等于 OWNER_TRANSFER_SUPPORTED，也不构成 production/cutover approval。

## 对象与环境边界

历史布局固定 b618dea24b7c46cd36fd11a568a72c9a88f2097a，拆分布局固定
b5d0562381f5b3989578d62e2677364d8c73710d。迁移对象的完整 Git 内容必须相等。

| 新 owner | 对象 |
| --- | --- |
| secrets-foundation | atlas-secrets Namespace、platform-budget ResourceQuota、defaults LimitRange（3） |
| observability-foundation | atlas-monitoring 上述三项、monitoring-ingress NetworkPolicy（4） |
| storage-foundation | atlas-storage 上述三项、s3-default-deny / s3-clients NetworkPolicy、workload-web/s3-client-egress（6） |

跨域 Role/RoleBinding 保持 secrets-controller ownership。PVC、CRD、SealedSecret、
Deployment 会作为完整平台的一部分存在，但绝不加入 ownership-transfer 集合。

新建四节点 atlas-refactor-test-ot1，OrbStack，Cilium，control-plane / gateway /
compute / data 拓扑、node image、taint/placement、所有锁定 chart/image/tool 与 dev02
基线相同。API 与入口仅 loopback；拟用独立入口 18080/18443。保留 dev02、OT-0、
registry 和默认 kubeconfig。不导入任何旧 identity、Receipt、密钥、应用凭据。

顺序为：新实例化 → core ADOPTED → 启用独立 Sealed Secrets controller → 私钥备份 →
新凭据严格作用域密封 → 同样 monitoring/object-storage/storage-monitoring 全部启用 →
旧单 foundation owner 的完整平台 ADOPTED → 记录 BASELINE_ADOPTED。
这不是用极简 Argo 代替 Atlas，也不能通过复制 dev02 的凭据冒充等价。

用户已单独允许本次 OT-1 开发演练的同机 w1 备份例外：解析到现有 Vault 后使用
独立 atlas-refactor-ot1 子目录，目录 0700、文件 0600，create-only，不覆盖 dev02。
该例外不满足物理隔离，不推广到其他环境；未来离线备份待办保留。私钥和明文不进入
Git、日志或报告。未来发布的新三份密文必须在具体部署计划中列出 namespace/name/hash。
当前没有生成新密钥、读取旧密钥或生成/发布 OT-1 密文。

## S1 范围与首次实例化

用户已将此前“仅审查契约、暂不改普通 engine”的范围更新为完整 S1：共享 Go
Observation/Evidence、OT-1 状态机与执行准备、受限实例化 profile 一并实现并本地验证，
在现有 PR #6 中统一审核。真实集群操作仍须完整计划与独立批准；此次没有运行 OT-1。

同一个 Go engine 现在认识两个固定 development 绑定。新绑定仅允许 schema 3、
atlas-refactor-test-ot1、codex/ot1-desired-state、18080/18443；错误组合在外部写入前拒绝。
新 profile 使用独立 substrate identity 和编译期 snapshot digest。正常 development
分支、8080/8443 和 baseline-v3.json 的历史字节没有变更。没有通用 hash override、
force、第二个 Bootstrap engine 或 implicit migration。

`atlas-ot1 prepare-profile` 从固定来源 65af8497c02d22a60eb8bcaecf2434790edda2df 建立
新的私有本地 clone，仅投影实验 source/port，导入固定旧 catalog/foundation；旧密文被清空。
[profile-baseline.json](../../experiments/foundation-ownership/profile-baseline.json) 记录这次
首次实例化的来源、投影标识和 19 个冻结输入摘要。真实 Helm/Kustomize 与普通 engine
验证这个新快照；正常 dev02 快照不由新输入重新计算。

完整能力启用、独立密钥备份和三份新密文仍是现场准备。`plan` 只接受已提交的完整旧布局，
输出 7 个本地 Git revision 和 29 个阶段；后续只有 catalog 与父级 Application 投影可变化，
AppProject 权限、实际 13 个资源、controller payload、Root、Seed、凭据等保持字节相同。
本地测试中的不可解密密文只是语法 fixture，绝不作为现场凭据或 runtime evidence。

## 共享 Observation 与有限执行器

`internal/observation` 与 `atlas-platform observe|verify` 只读取事实。复用锁版本的 GVK
scope，记录精确 revision、UID/RV、generation/observedGeneration、完整 spec 摘要、
Sync/Health、operation、conditions 和 blocking resources。没有数据的字段保持缺失；
UNKNOWN 不能当作不存在。API 客户端只有 GET，拒绝默认 kubeconfig、exec 插件、非
loopback API、重定向和 Secret/SealedSecret 读取。私有快照 0600、目录 0700、create-only。

OT-1 复用该观察器；预期 strict 拒绝仅由阶段校验器解释，不把通用 Observation 的
DRIFTED/DEGRADED 改为健康。`atlas-ot1 run` 独立于普通 Bootstrap/select/render；它要求
同一 clean Go binary、完整 plan SHA、精确 cluster UID/kubeconfig hash 与独立 runtime
checkout。每个请求先持久记录 intent，随后只提交一次；正常阶段自动续行。

执行器只向固定实验 Git 分支作 fast-forward 发布，并操作四个精确 foundation
Application 的 create、带 UID/RV/full-spec tests 的 syncOptions/operation patch、
带 UID/RV preconditions 和 Orphan propagation 的 DELETE。每步重新核对 Git、目标和
受测资源；不会直接写 13 个对象、放宽 AppProject、手改 tracking 或回退 Bootstrap latch。

平台完整 Gate 调用同一 engine 的 Status/Apply；重复 Apply 包装在拒绝写请求的 Runner
中，并比较 Metadata-only audit ID 集合、身份摘要、四节点/Pod/PVC/PV 和本地 HTTPS。
审计另检查 13 对象仅有 Argo 写入，允许 Kubernetes 控制器的正常 status 更新。
API 多对象读取不是事务；首尾版本或 inventory 变化会 UNKNOWN/STOP。

## 父 Application 的持续 authority

platform-control 会自动创建/修复 Git 中的子 Application。因此删除旧 owner 后仅
patch 子 App 不足以隔离窗口；selfHeal 可以重建旧 owner 或覆盖 syncOptions。

候选 Git ceremony 阶段必须先从 platform-control 的期望 Application 集合中移出
整个 foundation owner 集合，其他 Application、项目权限、Root 内容保持。
确认 parent 已比较到精确阶段 SHA、旧 App 被 Prune=confirm 保护而未自动删除后，
才以 UID/RV 前置条件非级联删除旧 App；不得批准 parent 的通用 prune。
实验中使用已入 Git、无 automated、无 finalizer 的精确 owner App 清单，逐个创建。
所有实际对象仍由 Argo SSA；禁止直接 apply 这 13 个资源。

每次窗口只使用下面的窄补丁。全部恢复 strict 后，通过 Git 恢复完整的普通子 App
投影，由 parent 接回这些 Application；验证同 UID、父 tracking、自动同步策略与
完整 spec，再执行 OT-1B。回退亦先从 parent 期望集合移出新 owners，再执行同一套
非级联 release / old strict refusal / old window / strict restore / parent reattach。

这是待验证的**临时 Tier-1 Application 配置交接**。不能把 detached 状态宣称为完整
Atlas ADOPTED，也不能引入一个新的 Root、改写现有 Root 或永久 ignoreDifferences。
若 parent prune/reattach 行为与计划不符，STOP；不手工删 tracking、不临时关闭
platform-control controller。该候选仍须完整阶段清单、渲染及运行证据才能成立。

## 阶段、窗口和 STOP

可机读阶段见 [stages.json](../../experiments/foundation-ownership/stages.json)。先验证
部分回退，再执行完整 forward/reverse；这是计划内测试，不是遇错自动 rollback。

1. BASELINE_ADOPTED：旧 owner strict、13 对象、完整平台及重复 apply 基线。
2. SOURCE_RELEASED：精确旧 owner 非级联消失；13 个旧 tracking 保留。
3. secrets strict 必须共享拒绝 → 单窗口接管 3 → strict 同步成功。
4. observability 同样接管 4；明确记录 owners={secrets:3,observability:4,old-stale:6}。
5. MIXED_ROLLBACK：移出/删除两个新 App；旧 strict 必须共享拒绝 → 旧窗口一次接回
   13 个 → strict 恢复 → parent 接回 → OT-1A/1B。
6. 再次 release 旧 owner，按 secrets、observability、storage 依次 strict refusal →
   window → strict restored。任何时刻至多一个 window；前域完成才进入后域。
7. FORWARD_VERIFIED：三个新 owner 3+4+6、完整图重新接回，OT-1A/1B。
8. REVERSE：release 三个新 App；旧 strict 必须共享拒绝 → 窗口一次接回 13 →
   strict 恢复 → parent 接回 → OT-1A/1B。最终旧 owner，保留新集群和证据。

模式变更：新 GET → UID/RV/full expected spec/idle/精确观察 SHA/上阶段 operation
info/无 finalizer → JSON Patch tests → 只 replace /spec/syncPolicy/syncOptions。
operation sync 显式完整 SHA、独立 ot1-stage info、prune=false、阶段对应 SSA/strict
选项；不能用前一次成功覆盖新阶段的失败。

每步 assert → mutate → observe → create-only evidence。任何意外，包括预期拒绝未
拒绝、额外 condition、读不可用、对象集合偏差、并发变更、5 分钟单阶段超时，都立即
STOP，保存现场并记录已执行请求/退出码/观察误差/当前窗口状态。停止是该 attempt 的
有效终态，不能用之后成功覆盖它。没有自动修复、重试 mutation、rollback、继续或
清理。窗口不是自动到期租约，超时后可能仍开启；续行必须新观察、新计划绑定与新决定。

## 两个独立 Gate

**OT-1A**：精确 13 身份；live UID 不变；全量 semantic hash 不变；逐项 owner tracking
与 destination namespace 正确；Argo SSA managedFields 存在；目标 App UID 在窗口内
保持；严格复原后成功，无共享 warning；包含部分回退和 3→1。

semantic 仅排除 status、metadata 中 UID/RV/creationTimestamp/generation/managedFields
及唯一 tracking annotation。labels、其他 annotations、spec/data、finalizers、
ownerReferences 均保留。UID 独立检查，managedFields 原始记录保留。完整平台上的
ResourceQuota status/used 可正常变化；不能把 status 更新错误归为内容变化，也不能
借此忽略 spec 或 tracking 变化。strict refusal/release 要求 semantic/UID/tracking
均不变，并通过审计区分 Kubernetes status 更新与对受测对象的非法写入。
Namespace 的 tracking namespace 随 Application destination：旧 owner 是 argocd，
新 owner 是各自域 namespace；workload-web/s3-client-egress 始终用 workload-web。

**OT-1B**：独立现有 Bootstrap status=ADOPTED；Identity、Latch、Receipt、Signal 的 UID
与持久内容，以及 Root UID/spec 不变；全部 AppProject UID、spec/权限不变；全部
Application 精确清单/full expected spec/对应阶段完整 SHA、idle/Synced/Healthy、
没有 condition、foundation App 重新归 platform-control 配置管理；四节点 Ready、
原平台 workload/PVC/HTTPS 仍通过；同一已审查 Go binary 重复 apply exit 0，记录
runner 命令日志和 Metadata-only 审计差值，kubectl mutation=0，前后身份不变。

operation 的最近成功 SHA 可能早于当前无内容差异的 Git SHA，沿用 schema 3 已有
Receipt 后 CRD 规则；不能为通过本实验改弱首次 adoption 或当前 observed revision。
各 Git 阶段先锁定 SHA，再收集其证据；禁止在一次观察期间移动来源分支。

## Plan、证据与批准绑定

执行前必须形成 create-only plan，包含：ceremony/attempt ID、实现 SHA 与 binary hash、
各阶段 desired Git SHA / render hash /完整 App spec、scope inventory hash、phase graph
hash、toolchain/artifact/image hashes、精确 cluster/context/API exposure、允许请求列表、
timeout/stop 行为与批准记录。Plan SHA 是最终 plan 文件字节 hash，不把自身 hash 写入
自身。首次启动准备与 transfer plan 分开：cluster 创建后形成独立 target binding，transfer plan
在编译时包含 cluster UID 和 kubeconfig hash；没有完整绑定就不能生成可运行 transfer plan。

预期拒绝的 Argo operation 为 Failed，阶段 verifier 可以 exit 0；报告必须同时保存
operation phase、verifier exit、对象差异和无旁路写入证据。公开证据只包含经过检查的
非敏感结果；私有 GET 原文、kubeconfig、key/credentials 与日志留在 .state/Vault。

当前已验证本地真实渲染、普通 Bootstrap 回归、共享观察器、合成阶段链及请求 guard；
fresh Bootstrap 和完整平台已有现场证明。完整 OT-1A/OT-1B、父级 reattach、partial
rollback 与 full reverse 已在后续固定执行中通过，见最终验证。历史跨版本采集等 STOP
保持不变；runtime 成功不自动接纳 ADR-0008，也不批准迁移 dev02。
具体入口、plan/evidence 格式、授权边界及限制见 [S1 实现说明](../s1-observation-ownership.md)。

## 开发保留策略

create-only、SHA 绑定和不可覆盖终态用于实际 authority execution。日常本地开发与只读
预检使用 latest 工作状态；已理解且确认没有外部 mutation 的失败进入回归测试和
[Failure Journal](../s1-failures.md)，临时 plan/binary/快照可清理。重要 live/Git mutation、
Trust Root/凭据变化、影响不明的 STOP 与最终 Gate PASS 仍保留完整 authority evidence。
锁用于互斥和阻止未经审查的重入，不兼任永久档案；执行器失败仍留锁，调用者仅在明确
零影响、当前状态复核及用户授权后清理摘要匹配的旧锁。已发生 mutation 的恢复决定不变。

历史混合状态、版本 fence、审计匹配修正及干净重建后的 SOURCE_RELEASED 超时见 [Failure Journal](../s1-failures.md)。

## Proposed evidence and continuation revision

[ADR-0010](0010-semantic-evidence-and-source-released-continuation.md) proposes semantic double-read evidence, transitional source revision equivalence and one explicit SOURCE_RELEASED continuation. Historical attempts retain the exact-RV/exact-revision rules above and their immutable STOP outcomes. Mutation fences remain exact; this proposal does not approve live execution.
