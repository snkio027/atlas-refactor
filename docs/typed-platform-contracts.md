# Typed Platform Contract 方向（Proposed）

这是供后续 owner 审查的设计方向；现有 AH-1～AH-4 见 [ADR-0007](adr/0007-platform-contract-hardening.md)，
S1 的本地实现见 [ADR-0009](adr/0009-foundation-ownership-rehearsal.md)。此文不修改原 Atlas 规范、生产 authority，也不承诺所有示例 API 已实现。

Atlas 的长期职责是把受类型约束的意图编译成可审查的 GitOps desired state，并验证权限、
依赖和生命周期。Bootstrap 保持有限实例化；Argo 持续调谐 Git 定义；Kubernetes 和领域
controller 承担运行时。新增能力不应把本地 Go 程序变成第二个 installer、scheduler 或 reconciler。

## 模型与术语

| 层 | 职责 | 当前与后续边界 |
| --- | --- | --- |
| L0 Substrate | Kubernetes、CNI、基础存储 | 当前 Kind 四节点开发 profile |
| L1 Platform Control | Root、AppProject、handoff、trust/evidence | 现有有限 Bootstrap 与 GitOps authority |
| L2 Capability | 面向使用者的接口，如 S3、TLS、Metrics | 现有目录是起点；SeaweedFS 是 Component 实现，S3 才是 Capability |
| L3 Project | tenant、owner、namespace、quota、网络与能力访问 | S2 内独立建模，不把每个项目塞进 capability catalog |
| L4 Workload | 可执行意图及 exposure/storage/observability/scaling/scheduling 需求 | S2 先支持 Web/API service；高层模型不固定等同于 Pod/Deployment |
| L5 Runtime Objects | Kubernetes / Operator 消费的对象 | Deployment、HTTPRoute、PVC 等是编译目标；未来其他目标须由真实需求驱动 |

Binding 表示 Project/Workload 对能力接口的消费关系。S2 将把当前分散的 SecretRef、
NetworkPolicy 两端、身份标签、S3 桶权限与 endpoint 约定纳入一个受检查的契约。
不要先创建空壳 CRD/operator；优先 Git-time compilation，敏感明文仍不进入 Git。

Capability 的后续模型可逐步表达 lifecycle、authority requirements、接口、credentials、
health 与 implementation。当前 catalog schema 2 只是 AH-3 的字段演进，不等于已完成这个模型。
wave 目前仍是显式 Argo 配置；将来才设计 dependsOn + phase 的编译规则，不能现在删除已有门禁。

## Observation 与 mutation

建议确立：Observation MUST NOT directly imply mutation。
观察 → 分类 → policy → 决策/门禁 → 有限行动；退化、漂移、凭据到期或 PVC 无消费者
都不能单独授权删除、恢复或重启。Agent 也应经 Proposal / Git 审查进入同一权限链。

S1 的共享 Observation 已只读记录 expected/observed revision、对象 identity/UID、
generation、源 API 提供的 observedGeneration、Sync/Health、operation phase 与 blocking resource。
不为没有 observedGeneration 的 API 伪造字段；按每类 API 的证据语义判断是否针对当前 spec。
UNKNOWN 不等于 ABSENT 或 HEALTHY。应用旧 commit 的 Healthy 不能替代目标 commit 收敛证据。

观察写入私有 .state/observations；经脱敏审查的正式报告进入 docs/evidence。
运行状态不写回 enabled/catalog desired state。S1 固定 VERIFIED/PROGRESSING/DEGRADED/DRIFTED/UNKNOWN/ABSENT 与显式
verify --revision 接口；没有观察 daemon、refresh annotation 或自动修复。

## 三个 Architecture Slice

一个 Slice 同时包含模型、实现、不变量、测试、迁移影响和证据说明；共享语义放在同一 PR，
提交仍按职责保持可审查。ADR 记录长期结构/权限/生命周期决策，不为每个 helper 新建 ADR。
新的 trust boundary、生产部署、不可逆数据生命周期和独立 recovery 仍有各自批准边界。

| Slice | 一次交付 | 当前状态 |
| --- | --- | --- |
| S1 | Observation + Evidence + 精确 OT-1 状态机、有限 executor、实例化 profile | 同一 PR #6 本地实现与验证；真实 1→3→1、部分回退和 Atlas Gate 尚未运行 |
| S2 | Project + Workload + CapabilityBinding + 首个消费 S3 的 Web/API 实例 | 路线图；没有新增 schema 或空壳 Operator |
| S3 | 确定性 scopegen、metrics-server 候选、按预算/需求选择 Loki/Alloy、CI/release hardening | 路线图；CI required check 与 remote supply-chain proof 尚未建立 |

S2 用同一个项目编译 namespace/quota/network、Deployment/Service/HTTPRoute、可选 metrics、
SecretRef 与 S3 两端网络/身份权限；不提前引入 GPU、多集群或通用 Workload DSL。
S3 的 runtime 组件仍逐项验证锁定 Kubernetes 版本、aggregation/TLS、权限和资源预算；
PostgreSQL 等数据组件在具体项目确有需求后设计数据生命周期、备份与凭据。

部署序列与本地设计序列不同：dev02 的 owner migration 仍被真实 OT-1 Gate 约束；这不会
阻止 S2 的本地契约设计。完成当前 stacked PR 链后，再回到 main 上的单个 Slice 分支，
不在这次变更中自动合并其他 PR 或改写历史 integration evidence。

Kueue、DRA、Agent Sandbox、Karmada 是未来需求的候选方向，不是当前依赖。
附带材料中的 Kubernetes 特性成熟度需要在具体采用时核对所锁版本与官方来源，不能从趋势描述
推断其已在本集群可用，或让未验证特性驱动当前 baseline 升级。

文件继续区分：AUTHORED 输入、GENERATED 清单、FROZEN 首次身份、EVIDENCE 历史事实、
PRIVATE_RUNTIME 本机状态。本轮不为了目录整洁迁移 frozen 文件，也不重命名当前环境 authority 分支。
