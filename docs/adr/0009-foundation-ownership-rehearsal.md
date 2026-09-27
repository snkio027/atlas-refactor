# ADR-0009：Foundation Ownership Split Rehearsal（OT-1）

- Status: Proposed
- Date: 2026-09-27
- Parent: [ADR-0007](0007-platform-contract-hardening.md)、[ADR-0008](0008-ownership-transfer-probe.md)
- Scope: 仅 atlas-refactor-test-ot1；13-object、1→3→1 实验，不批准 dev02 迁移
- Execution status: NOT_READY；当前交付范围契约和离线校验，尚无 OT-1 runtime proof

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

## 必须先解决的首次实例化前提

当前 Go 实现不能直接建立该等价环境：

1. `internal/atlas/config.go` 与 `internal/platform/render.go` 限定 development source
   为 dev02 的 codex/development-platform。修改这个共享分支会影响 dev02。
2. `validateDevelopmentKind` 精确固定 8080/8443；这些入口已由 dev02 占用。
   `atlas-dev verify` 也固定相同端口及重定向地址。
3. schema 3 的 baseline-v3.json 被编译期 digest 固定。不能改写该历史快照、注入
   fixtureSnapshotDigest、复制旧 identity 或跳过验证来让 OT-1 启动。
4. 当前 catalog 已是拆分布局；AH-1 正常路径不允许隐式移除旧 foundation。
   直接运行当前 atlas-dev up 不会产生要求的 pre-PR#4 baseline。

建议作为独立前置变更审查：在**同一个 Go engine** 中加入仅绑定
atlas-refactor-test-ot1 / codex/ot1-desired-state / 127.0.0.1:18080,18443 的实验实例化
profile；为这个新目标创建有来源 commit、独立 digest 的首次实例化快照。现有 dev02
配置、快照与默认行为逐字节保持。不是通用 profile framework，不增加 force、可由
命令行覆盖的 baseline hash、迁移命令或第二个 Bootstrap engine。

该前置变更须证明错误 cluster/source/port/snapshot 组合全部在外部写入前拒绝，
真实 Helm/Kustomize 与普通 Bootstrap 回归通过。它尚未实现或批准用于运行。
本 ADR 不通过声明新 profile 来掩盖当前无法运行的事实。

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
自身。Cluster UID、kubeconfig hash 在创建后追加独立 binding，再做第一步 mutation。

预期拒绝的 Argo operation 为 Failed，阶段 verifier 可以 exit 0；报告必须同时保存
operation phase、verifier exit、对象差异和无旁路写入证据。公开证据只包含经过检查的
非敏感结果；私有 GET 原文、kubeconfig、key/credentials 与日志留在 .state/Vault。

当前只有离线 scope/phase/patch guard 测试；OT-1A、OT-1B、父级 detach/reattach、fresh
bootstrap、完整平台、failure-path 均未执行。不得据此接纳 ADR-0008 或迁移 dev02。
