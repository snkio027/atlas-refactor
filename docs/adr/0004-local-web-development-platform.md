# ADR-0004：受限本地 Web/API 开发平台

- Status: Proposed
- Date: 2026-09-27
- Scope: atlas-refactor 的独立开发候选；不修改原 Atlas 规范，不批准生产替换

## 背景

最小 Go Bootstrap 已完成第四轮可丢弃集群验证。用户把近期目标调整为开发其他项目所需的
网络入口、TLS 与本地持久存储，稳定性体系后续补齐。首轮实现 GitOps 清单并本地验证，
随后用户明确授权提交审核和启动本机独立开发集群。此前更广范围部署前完成全部恢复能力的约束，需要明确限定这个新实验范围。

## 提议

允许新增一个与第四轮基线分离、可丢弃的单节点 Kind 开发 profile。它沿用原 Atlas 的
Cilium CNI、Gateway API 北南向、直接 CNI 东西向、IPv4-only 和身份选择器网络授权。
通过 cert-manager 提供 namespaced 本地 CA，通过 Kind 的 local-path 提供 Retain PVC，
以非 root Web 示例验证用途。平台自我管理、权限边界和宏观 DAG 均不改变。

Root 只管理 project-bootstrap / platform-control / workload-control。
三个 canonical AppProject 保持准确名称；workload-control 属于 platform-project，
业务叶子属于 workload-project。平台负责 namespace、网络隔离和资源预算。
Helm 是渲染器；Git 保存已渲染定义；组件/controller 拥有其运行时生成对象。
证书 Secret 运行时生成，Git 不包含 Secret material。

Cilium 是 Argo 启动前的 substrate 依赖。启动实现把 Cilium 与 Argo 纳入同一
有期限的 Seed authority，在 durable handoff latch 后统一撤销。两个 Seed 与 GitOps
叶子字节一致，GitOps 完整 SSA 接管；Go 不成为持续网络控制器，也不添加 Shell 旁路 installer。
`profiles/development.json` 以 schema 2 选择这个边界，schema 1 继续用于历史最小实验。
所有 Application 跟随审核分支；首次启动的精确 SHA、渲染指纹和私有状态独立保留。
首次部署输入摘要冻结在 `bootstrap/baseline.json`，其完整摘要必须对应已记录的初始 Identity。
该文件不是把当前期望状态写回旧身份的入口：Root、AppProject、宏观 DAG、两个 Seed、镜像锁和 substrate
必须仍与快照一致，变化时报错。普通平台/业务叶子由 GitOps 演进，其最新提交单独记录；
不因此改写初始 Identity、latch 或 Signal。首次全 bundle 身份绑定过宽的问题由这一边界收窄解决。
不授权人工跳过现有安全检查。

允许推迟完整 HA、Recovery/Drill、备份和发布 provenance，前提是范围始终为独立可丢弃开发集群，
不存放唯一业务数据，不继承原 Atlas live state，也不宣称生产就绪。
首轮部署以 Cilium-first 启动和双 Seed ownership 检查为前置条件，固定执行 commit，
目标为用户已授权的 `atlas-refactor-test-dev01`。不能沿用旧 39 项 Seed 的成功报告证明新平台交接。

## 不变的架构约束

原 Atlas `aca4ff137a1d254cfeceaec24526e0699b585e92` 的
Architecture v1.0.2、GitOps v1.0.3 §§3、7、9、11–13、18–19 和 Network v1.0 是参考权威。
此 ADR 不授予架构例外：External Root 无 parent/级联 finalizer；正常路径不覆盖 drift；
只有一个 Bootstrap engine 拥有初始 mutation authority；恢复另行操作；业务不获得平台权限。
ADR-0002 的默认语言决策、ADR-0003 的原 Atlas authority transfer 继续独立审批。

## 代价与后续决策

单节点、单副本、节点本地卷和本地 CA 仅适合开发；Retain 与删除确认都不能替代备份。
开发者需显式添加业务 namespace/repo allowlist、HTTPRoute、资源配额和网络出站白名单。
新的 namespace、域名、公开入口、存储类型或组件仍需对应控制域审查。

详细文件、波次、验证能力与尚未执行的启动顺序见
[开发平台与部署审查](../development-platform.md)。Accepted 只代表接受这个开发范围，
不代表已经部署、通过真实网络验收或批准替换原 Atlas。
