# ADR-0015：Typed Project / WebService / S3 Binding

Status: Accepted — 仅限单所有者本地开发。提出：2026-10-01；维护者复审接受：2026-10-09。

本记录采纳独立 atlas-refactor 的 S2 配置与权限投影，不修改原 Atlas 的 normative 文档，
不自动接受其他 Proposed ADR，不批准具体集群或公开密文发布。

## 问题

D1 能安装平台，但新增业务仍需理解 namespace、HTTPRoute、网络策略、S3 SecretRef、
provider auth 配置和监测选择器。单独包装三个 JSON 文件不能消除这些跨域关系，也不能
把共享平台对象拆成多个 Application owner。

## 决策

采纳 [S2-0 语义契约](../s2-semantic-contract.md)：三个严格类型 authored 对象 Project、Workload(WebService)、
CapabilityBinding。首版一个新增 Project/namespace，绑定现有 object-storage/uploads，
有界 Create/Update。配置经显式 resolved semantic model 编译为 GitOps runtime objects。
不接受任意 Kubernetes escape hatch；不创建 Atlas CRD、Operator、controller 或公共 SDK。

纯编译不访问集群、不生成随机凭据、不发布 Git；凭据准备、发布与有限功能探测分别有显式边界。
确定性绑定完整产品、部署、意图与已登记密文输入。观察保持只读，写入型探测有独立证据。

Namespace/quota/policy/SA/监测/凭据由 Tier-1 持有；Deployment/Service/HTTPRoute 属于
Tier-2 workload leaf。Root macro DAG、三个 canonical AppProject、Bootstrap durable handoff
保持不变；destination 精确扩充，权限集合不转化成多租户强隔离承诺。
共享 Gateway、Sealed Secrets controller、provider auth 和 Prometheus 配置经原 owner 聚合增量，
不进行 owner transfer，不复用 S1 ceremony 作为普通业务生命周期。

每 Binding 独立凭据；同实例现有证书/备份事实先验证。保留 D1 的既有 provider identity，
向聚合认证配置添加有界权限的新 identity；密文严格绑定实际 namespace/name。
controller 的 namespace/RBAC 扩展、Secret 物化、provider 认证生效都有真实门禁，
不把 SecretRef 存在或 sync-wave 当作能力可用的充分证明。

S2 保留 D1 安装记录并建立新的扩展验收。旧 D1.4 的 exact FullCommit 检查仍安全拒绝
扩展后的 commit；不篡改原安装证据来保持旧 CLI 返回成功。未扩展 D1 的安装与重复验收不回归。
publisher 保留完整 deployment parent tree，只发布受审查 authored/generated 路径增量，
不把 D1 gitops-only 安装发布器当成任意用户仓库发布器。

不支持 identity 改名、移除对象、解绑、撤权、退役、自动 rollback 或恢复；update 中也拒绝
隐含这些操作。独立诊断资源清理仅按该次 probe 的明确清单，不形成产品删除协议。

## 代价与范围

这是单所有者本地 Web/API 开发能力，不是生产多租户平台。S3 使用已有 HTTP endpoint，
标签策略不等于密码学身份。SeaweedFS mini 与原领域 Operator 要求的差距不由本 ADR 自动豁免。
现有组件的固定 selector/namespace、聚合 Secret、D1 发布接口都需要窄集成修改；
“不增加组件”不意味着不用修改平台定义，所有修改必须保持唯一 owner 和权限可审查。

不把 read-write 只映射成 provider 字符串就宣称最小权限；实际拒绝跨桶/桶管理权限属于 Gate。
provider 若不能实现承诺的权限，停止启用并回来审查契约，不追加隐含管理权限。

## 接受与实施

PR #8 已合入 `531d234`，S2 从该 main 独立分支，一个 PR 完成语义模型、编译、静态契约与真实 slice。
维护者完成增量复审后，PR #9 以 merge commit `6dcc6d36642df56eca9e5bfd72b93793ee2c3356` 合入 main。
接受范围仅为本文已实现的单所有者本地开发模型，不包含生产、多租户、删除/退役或恢复能力。
[r7 验收](../s2-r7-validation.md) 的 Runtime PASS 绑定 `d011813`；`fc809be` 的入口/终态修复
由增量回归与 [Quality CI](https://github.com/snkio027/atlas-refactor/actions/runs/37851541576) 验证。
契约接受、代码审核、具体执行授权和 runtime 证据仍相互独立；合并不部署或改绑 r7 实测。
S1/D1 frozen evidence 不修改；S2 不以继续扩张 OT-1 或安装器功能来满足验收。
