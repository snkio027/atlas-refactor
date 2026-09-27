# Typed Platform Contract 方向（Proposed）

这是供后续 owner 审查的设计方向；当前实现范围只有 [ADR-0007](adr/0007-platform-contract-hardening.md)
的 AH-1～AH-4。此文不修改原 Atlas 规范、生产 authority，也不承诺所有示例 API 已实现。

Atlas 的长期职责是把受类型约束的意图编译成可审查的 GitOps desired state，并验证权限、
依赖和生命周期。Bootstrap 保持有限实例化；Argo 持续调谐 Git 定义；Kubernetes 和领域
controller 承担运行时。新增能力不应把本地 Go 程序变成第二个 installer、scheduler 或 reconciler。

## 模型与术语

| 层 | 职责 | 当前与后续边界 |
| --- | --- | --- |
| L0 Substrate | Kubernetes、CNI、基础存储 | 当前 Kind 四节点开发 profile |
| L1 Platform Control | Root、AppProject、handoff、trust/evidence | 现有有限 Bootstrap 与 GitOps authority |
| L2 Capability | 面向使用者的接口，如 S3、TLS、Metrics | 现有目录是起点；SeaweedFS 是 Component 实现，S3 才是 Capability |
| L3 Project | tenant、owner、namespace、quota、网络与能力访问 | P1 独立建模，不把每个项目塞进 capability catalog |
| L4 Workload | 可执行意图及 exposure/storage/observability/scaling/scheduling 需求 | W1 先支持 Web/API service；高层模型不固定等同于 Pod/Deployment |
| L5 Runtime Objects | Kubernetes / Operator 消费的对象 | Deployment、HTTPRoute、PVC 等是编译目标；未来其他目标须由真实需求驱动 |

Binding 表示 Project/Workload 对能力接口的消费关系。B1 将把当前分散的 SecretRef、
NetworkPolicy 两端、身份标签、S3 桶权限与 endpoint 约定纳入一个受检查的契约。
不要先创建空壳 CRD/operator；优先 Git-time compilation，敏感明文仍不进入 Git。

Capability 的后续模型可逐步表达 lifecycle、authority requirements、接口、credentials、
health 与 implementation。当前 catalog schema 2 只是 AH-3 的字段演进，不等于已完成这个模型。
wave 目前仍是显式 Argo 配置；将来才设计 dependsOn + phase 的编译规则，不能现在删除已有门禁。

## Observation 与 mutation

建议确立：Observation MUST NOT directly imply mutation。
观察 → 分类 → policy → 决策/门禁 → 有限行动；退化、漂移、凭据到期或 PVC 无消费者
都不能单独授权删除、恢复或重启。Agent 也应经 Proposal / Git 审查进入同一权限链。

A5 的 RolloutObservation 将只读记录 expected/observed revision、对象 identity/UID、
generation、源 API 提供的 observedGeneration、Sync/Health、operation phase 与 blocking resource。
不为没有 observedGeneration 的 API 伪造字段；按每类 API 的证据语义判断是否针对当前 spec。
UNKNOWN 不等于 ABSENT 或 HEALTHY。应用旧 commit 的 Healthy 不能替代目标 commit 收敛证据。

观察写入私有 .state/observations；经脱敏审查的正式报告进入 docs/evidence。
运行状态不写回 enabled/catalog desired state。建议的状态名和 verify --revision 接口尚未冻结，
本轮不新增观察 daemon，不自动写 refresh annotation，不自动修复失败。

## 实施 Gate

| Gate | 交付 | 进入下一阶段所需结论 |
| --- | --- | --- |
| A1–A4 | 单调启用、版本绑定 GVK、permissionDomain、foundation 分域 | 本轮实现与本地测试；现有集群 owner 迁移独立审查 |
| A5 | Rollout Observation | 当前 revision 的就绪证据可解释；只读验证不取得调谐权 |
| P1 | Project Contract v1 | tenant/namespace/quota/network 与 capability access 边界明确 |
| W1 | Workload Contract v1 | 首个 Web/API 意图可重复编译，归属与产物分工可审查 |
| B1 | Provider / Consumer Binding | S3 消费者凭据、网络和服务接口来自同一契约 |
| X1 | metrics-server 候选 | 先完成离线 scopegen 的源 SHA 校验与 registry 字节一致性；再验证 aggregation/TLS/权限，可保持未启用 |
| X2 | Loki / Alloy | 在 Project/Workload ownership 后定义日志归属与访问 |
| X3 | PostgreSQL | 第一个业务需要时再设计数据生命周期、备份和凭据 |

Kueue、DRA、Agent Sandbox、Karmada 是未来需求的候选方向，不是当前依赖。
附带材料中的 Kubernetes 特性成熟度需要在具体采用时核对所锁版本与官方来源，不能从趋势描述
推断其已在本集群可用，或让未验证特性驱动当前 baseline 升级。

文件继续区分：AUTHORED 输入、GENERATED 清单、FROZEN 首次身份、EVIDENCE 历史事实、
PRIVATE_RUNTIME 本机状态。本轮不为了目录整洁迁移 frozen 文件，也不重命名当前环境 authority 分支。
