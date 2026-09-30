# ADR-0007：平台能力编译契约加固（AH-1～AH-4）

- Status: Proposed
- Date: 2026-09-27
- Scope: 独立 atlas-refactor 候选；本地实现、验证与审核，不批准 dev02 部署或生产切换
- Parent: [ADR-0006](0006-declarative-platform-capabilities.md)

## 控制权与问题

沿用原 Atlas Architecture v1.0.2、GitOps v1.0.3 §1–3、§8、§11、§19：
Git 定义期望状态；Bootstrap 有限实例化；Argo 调谐定义；领域控制器调谐运行时。
canonical AppProject、Root → Control → Leaf、Tier-0 判断门禁和密钥备份要求不变。
本记录不是原 Atlas 的 superseding ADR，不修改 Shell authority 或新建控制面执行器。

已有目录允许 select 缩减能力集合，把删除混入启用操作；两份资源 scope 列表把未知类型
隐式当作 namespaced；一个 capability-foundation 还混合管理三个域。以上问题会在新增
APIService、集群级 CR 或仅部署密钥控制器时暴露。此轮先收紧编译契约，不扩充平台服务。

## AH-1：启用与退役分开

select 比较旧、新依赖闭包，要求 old ⊆ new。删除 capability、以空集清空、删除仍需要的
依赖，均在执行外部工具或写文件前拒绝。顶层名称改变但闭包不变仍允许。
plan 展示 lifecycle、removedCapabilities 与 retirementSupported=false；readyToEnable
同时要求不发生退役且静态密文条件满足，仍不代表部署批准、解密或运行时就绪。

已有生成的 platform Application 清单也是本地比较输入，避免手改 enabled.json 后 render
绕过删除检查；缺失或不可读的旧投影按失败处理。该防护不是服务器端锁：同时修改目录、
选择和投影仍须代码审核。没有以本地配置冒充已部署清单，也没有普通 force/remove 开关。
资源内容变更的删除风险仍须 diff 审核；单调选择不等于每个 Kubernetes 对象永不删除。

退役须另有 consumer 盘点、数据处置、凭据撤销、网络绑定移除、权限收缩、观察与回滚契约。
Prune=confirm / Delete=false 保留，不能把删除保护当成已实现退役或备份。

## AH-2：单一版本绑定资源模型

内置资源来自 platform/development/kubernetes-api-scope.json，绑定锁定 Kubernetes 1.36.1。
88 个 GVK 条目由该 tag 的官方 OpenAPI 中 top-level GET /{name} 操作提取：
操作 x-kubernetes-action=get，取 x-kubernetes-group-version-kind；URL 含
/namespaces/{namespace}/ 时是 Namespaced，否则是 Cluster。排除子资源和 watch。
这是可管理对象的受审查 scope 表，不声称穷尽 create-only API 或提供内置 schema 校验。

来源：[固定版本 OpenAPI](https://raw.githubusercontent.com/kubernetes/kubernetes/v1.36.1/api/openapi-spec/swagger.json)。
源文件 SHA-256：dcede2063da1d7ad62ecb5af8adb6d7fabd0b52385a7fa0048afb491dac90450。
SHA 记录提取来源，不是发布者签名；普通命令不下载该文件。表作为本地编译输入纳入 bundle，
升级 Kubernetes 时必须重新审核版本与 scope，不能只放宽字符串比较。
后续 X1 metrics-server 前补齐离线 scopegen：校验 locked swagger 的实际 SHA，再确定性
生成 registry 并做 byte equality。当前 source.sha256 是 provenance，未证明表与源的派生关系。

CRD 的 scope 来自 spec.scope；只注册 served=true 的精确版本，并使用该版本结构 schema。
未知 GVK、冲突/重复定义、非法 scope、未提供 schema 的 served CRD 及版本不匹配均拒绝。
权限投影、namespace 检查与 CRD 结构子集检查共用 ResourceModel。APIService 和集群级 CR
由模型进入 clusterResourceWhitelist，不能进入 namespaceResourceWhitelist；带 namespace 的
集群对象拒绝。Namespaced 对象可显式声明 namespace 或使用 Application destination。
CEL、admission、控制器行为仍需真实 API 验证。

## AH-3：permissionDomain 是元数据

catalog schema 升为 2；enabled、制品锁仍为 schema 1。每个组件必须声明 permissionDomain，
目录显式列出 security / observability / storage → platform-project。缺失域、未知域和指向
非 canonical Project 的映射拒绝。plan 展示每个组件的域和实际 AppProject。

这三个域当前共享同一个权限并集。AppProject 的 destination namespaces × allowed kinds
允许交叉组合，不是逐组件最小权限，也不是域间强隔离。真正拆分 AppProject 是独立信任边界
决策；本 ADR 未授权。platform-credentials 仍是跨域物化单元，其 provider/consumer 重构留待 B1。

## AH-4：foundation 独立叶子

| 叶子 | 基础对象 |
| --- | --- |
| secrets-foundation | atlas-secrets Namespace / Quota / LimitRange（3） |
| observability-foundation | atlas-monitoring Namespace / Quota / LimitRange / monitoring-ingress（4） |
| storage-foundation | atlas-storage Namespace / Quota / LimitRange、三条 S3 网络策略（6） |

三个叶子都是 platform-control 的直接子 Application，wave=-100；不引入额外父应用。
它们恰好包含旧 foundation 的 13 个资源，内容保持一致。Sealed Secrets 的全部 RBAC
继续由 secrets-controller 拥有，包括 atlas-monitoring / atlas-storage 的四个 Role/RoleBinding；
namespace 位置不决定 controller 资源的归属。只选择 monitoring-crds 时，已启用的叶子不会包含引用
Sealed Secrets ServiceAccount 的 RoleBinding。

secrets-controller 的依赖闭包仅为 secrets-foundation → secrets-crds → secrets-controller。
platform-credentials 仍需另外两域 namespace，因此明确依赖另外两个 foundation。
namespaceCapabilities 绑定 foundation 选择与 chart additionalNamespaces；仅选择
secrets-controller 时监听和渲染 RBAC 的范围为 atlas-secrets / core workload-web。
之后增加 foundation，只为同一 secrets-controller Application 扩充 RBAC 和监听参数，
不进行 RBAC owner 转移。未选择 controller 时可离线渲染完整候选，但其资源不会进入
任何 foundation；Application 不启用就不会发布该候选。全部启用时权限并集不变，应用增加两个。

## 现有 dev02 迁移门禁

本 PR 的清单是新布局候选，不能直接推送到 dev02 跟踪的 codex/development-platform。
dev02 只有 capability-foundation 的 13 个对象需要转移 owner；四个跨域 RBAC
保留 secrets-controller owner。Prune=confirm 和 FailOnSharedResource=true 不会自动完成这种迁移。
普通 select/render 遇到旧投影时会报告 retirement/ownership migration unsupported。
本次代码变更中的离线投影重建仅用于展示受审查的新布局，没有操作已部署资源。

后续迁移必须另行提出具体、可回滚的 Git 阶段与精确目标，至少证明：

1. 盘点这 13 个对象的 UID、tracking、当前 owner 和保护策略；保存旧 commit 与观察。
2. 规定旧 owner 退出与新 owner 接管的顺序；任一时刻每个对象只有一个同步 owner，
   不靠去掉 FailOnSharedResource 或强制 prune 规避冲突，不删除重建 Namespace/RBAC。
3. 先确认 namespace 与 RoleBinding 已就绪，再验证 controller 当前配置确实监听新增 namespace。
   上游 controller 会跳过启动时缺失的 namespace；sync-wave 不是跨应用当前 SHA 屏障。
   若提前启动而跳过，迁移方案必须包含另行审查的 rollout 续行步骤，不在本工具中自动 refresh/restart。
4. 验证全部受影响 App 的精确 SHA、idle/Sync/Health，Secret 物化、权限拒绝和服务 API；
   资源 UID、Bootstrap Identity / Root / Latch / Receipt / Signal 保持，重复 apply 零写入。
5. 为每个中间阶段定义停止条件和反向 owner 转移；不能假设 git revert 就能恢复归属。
   非级联删除旧 Application 后的自然接管与反向接管都必须实测；删除 App 不等于清除资源
   tracking。禁止手工构造/patch tracking-id；保持 FailOnSharedResource，若自然接管失败则停止。

静态核查发现：锁定 Argo CD v3.5.1 的 [共享资源判定](https://github.com/argoproj/argo-cd/blob/v3.5.1/controller/state.go#L734-L743)
直接比较 live tracking owner 与新 App 名称，此处不查询旧 App 是否存在；
[同步入口](https://github.com/argoproj/argo-cd/blob/v3.5.1/controller/sync.go#L140-L145)
在 FailOnSharedResource=true 时据此拒绝。因此，仅删除旧 App 而保留旧 tracking
可能仍阻断接管；M1 → M2 及其对称 rollback 目前是待验证假设，不是获批可执行路径。

本轮没有运行此迁移，也未改变 Trust Root、公开密文、冻结 baseline 或历史 evidence。
既有实测结论仍绑定 d1fbd2d，不能用于证明此候选的 26 个 Application 已运行成功。

## 后续方向与验证

后续按 A5 → P1 → W1 → B1 顺序设计，之后才按需求评估 X1 metrics-server、X2 Loki/Alloy、
X3 PostgreSQL。术语和观察边界见 [Typed Platform Contract 方向](../typed-platform-contracts.md)。
本轮未实现这些 API、CRD、controller 或新组件；ADR 保持 Proposed。

验证要求：task quality、真实 Helm 最小/完整 namespace 选择及 controller RBAC 归属、正反向权限投影、GVK/CRD
version fail-closed、单调 select 零写入拒绝、重复生成字节稳定、Bootstrap 身份/写权回归。
本地通过不替代迁移演练与 owner review。

本次本地验证：task quality 通过（348 个 GitOps 资源）；实际 Helm 覆盖仅 secrets、
分别增加监测/存储域、仅 monitoring-crds、完整能力、增加后重复 select 字节不变及减少时
零写入拒绝。对照 b618dea，13 个 foundation 对象的 identity 和完整内容一致，四个跨域 RBAC
保留 secrets-controller owner 及内容，AppProject 权限集合一致；Root、冻结快照和密文字节未改。
这些是本地编译证据，本轮集群操作数为零。
