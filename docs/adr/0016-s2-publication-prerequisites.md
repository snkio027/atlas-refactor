# ADR-0016：S2 前置条件先收敛，再发布依赖者

Status: Proposed — 修复 r3 暴露的发布顺序缺陷。日期：2026-10-04。

## 原因

r3 在同一个 infrastructure Git 提交中增加 AppProject destination、Project Application
和 secrets-controller 的 demo RBAC。22:29:11 UTC controller 因 demo 未获授权而同步失败；
22:31:10 project-bootstrap 才应用权限。父级 sync-wave 不约束已有子 Application 对同一
分支的独立自动同步。Git 提交原子性不等于跨 Application 的运行时就绪顺序。

[Argo sync waves](https://argo-cd.readthedocs.io/en/stable/user-guide/sync-waves/) 定义一次同步
内的排序；[自动同步](https://argo-cd.readthedocs.io/en/stable/user-guide/auto_sync/) 是各应用的
独立行为。这是 Atlas 发布协议缺少前置条件，不以升级 Argo、增加 timeout 或忽略错误解决。

## 决策提案

保留 ADR-0015 的语义、唯一 owner、权限集合和最终 consumer 输出。首次发布固定为：

1. permissions：仅扩充两个既有 AppProject 的精确 destination；没有新 leaf/namespace/RBAC。
2. project：新增 Project leaf，创建 Namespace、quota、基线 NetworkPolicy、ServiceAccount。
3. infrastructure：原 owner 下增加 controller RBAC/namespace、TLS/Gateway、monitoring 适配。
4. consumer：原来的业务/Binding/密文和 provider rollout。

每次发布之前，前一阶段必须经只读 Gate 证明完整 live content、UID、tracking/SSA、
应用状态和精确 Git parent；发布依旧使用 exact-parent lease。阶段是固定序列，不是 DAG、
resume 或新 controller。Project namespace 的 Active 与安全基线先于新增跨应用写入。
静态 consumer preflight 在 permissions 编译时已执行，不能延迟到第一次 external mutation 后。

Plan schema 2 明确绑定三个无凭据阶段的输出摘要与固定阶段序列；consumer 仍绑定登记密文。
旧 schema 1 的 STOP 与历史证据保持原字节，新执行器不接受旧计划作为写权限。
完成后的受限 update 仍只有 consumer，Project/namespace 身份固定，不涉及新的 destination 或 namespace 初始化。

观察器按名称稳定分类整个 Application 快照，fatal 优先于 Pending；未知/错误仍 fail closed，
多余应用、重复身份也失败。最新失败报告及 condition 诊断保存在 owner-only 本地证据中，
不将 controller message 输出到终端或公开 Git。STOP 的报告单独冻结。

## 边界与验证

不改 Root、Bootstrap authority、AppProject 名称/owner、D1 frozen baseline、S1 Observation
分类规则和任何组件版本。不手动 patch、refresh、清锁、续跑 r3。新集群执行需要新的精确计划。
回归必须验证：任意应用调谐顺序下只发布已满足依赖的资源；阶段跳过/乱序/receipt 漂移拒绝；
混合 Pending/fatal 永远返回 fatal；真实 Kustomize 渲染与完整 task quality。
本地检查不是新实例 Runtime PASS。

## r4 首次观察边界（2026-10-08，仍为 Proposed）

r4 的 permissions Gate 通过后，Project Application 由 Argo 创建；S2 在约一秒内读取到
其精确 spec、generation=1、tracking/SSA 和 UID，但 controller 尚未写入任何 status。
把资源创建与首次调谐观察视为原子事件，是另一项执行器时序缺口。

只在当前已发布且有精确 receipt 的未完成阶段，比较直接 Git parent，识别本阶段首次声明的
Application。该对象须无既有 baseline UID、generation=1、spec 完全一致、正确 Argo
tracking/SSA，且 status 缺省或为空对象，无 operation。此时仅返回有界 Pending，保留
原始 UNKNOWN facts；既不生成 revision/健康证明，也不开放下一次 publication。

显式 Unknown、非空/畸形 status、错误条件、缺少 UID、spec/owner 漂移仍 fatal。
已有或前一阶段的 Application 不适用；当前阶段 Gate、final 或 STOP 存在时也不适用。
全快照其他 fatal 仍优先。最终放行仍要求原完整 Gate，等待沿用现有 15 分钟上限。
不改 S1 classifier、证据 schema、compiler、输出、依赖版本或 mutation surface。
这不是 r4 continuation；r4 永久 STOP，新实现 runtime 必须独立验证。
