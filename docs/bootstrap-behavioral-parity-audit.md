# Bootstrap behavioral parity audit

- Date: 2026-09-27
- Method: pinned-source inspection + existing fourth-run evidence
- Result: **BLOCKED for cutover; no Shell → Go rehearsal performed**
- Original Shell: `aca4ff137a1d254cfeceaec24526e0699b585e92`
- Go implementation: `0ef918801dac84ac913416f4c1756bf08045a485`
- Go result report: `9df41568bafcaa98d6548d2089766d83081c3374`

审计比较外部行为与权威边界，不比较代码布局。两份源代码都按完整 SHA 固定。
原仓库远端快照与本地 `427e026b109c865e20c527a7d38b7e3c58c30747` 不同，且本地有
未提交修改；后者未作为审计基线，也未被修改。此处没有执行原 Shell、访问其集群，
没有把静态推断标为运行时证明。

“原规范要求”“原 Shell 当前实现”“Go 实验证据”必须分开：Accepted ADR 可以定义
尚未实现的目标契约；已知旧实现缺口不成为新实现必须复制的行为。语言改写也不自动
批准新的状态语义。PASS 只属于相应范围，不能从第四轮成功推出整体 parity。

## 首要阻塞：adoption authority 与 readiness 的关系

[原 ADR-0002][S-ADR2]明定 Receipt 创建是 adoption 的线性化点，Application 健康
不是提交前提；valid protected Signal 存在时 Seed 已被拒绝，应进入 Receipt 提交。
原 Shell 的现行 handoff 仍以 self 是否存在及其健康状态决定停止 Seed，完整保护/
Identity v2/Receipt 流程未激活。这是原 ADR 明确记录的 INV-02 gap。

Go 则在 Root 创建前写 latch，在健康、四个精确 revision、Signal 和全体持久 Seed
ownership 通过后才提交 Receipt；Receipt 后仍持续检查这些条件。这证明了该实验
契约的单向交接，却与原 ADR 的 Receipt 时序不同。**更严格的 readiness 不能替代
对 authority 语义的批准。**默认迁移目标应遵守 Accepted ADR-0002；若另选模型，先
接受规范 amendment，再更改 candidate 并重新验证。冻结的第四轮证据不改写。

## 外部契约矩阵

状态含义：DIFFERENT 为已确认差异，GAP 为候选缺少所需能力，BOUNDED 为局部证据成立
但不覆盖迁移。所有条目当前均未取得完整跨引擎验收。实现维护者负责补证，调用方负责人
审查接口迁移，治理/供应链/Tier-0 项由 owner 与对应 CODEOWNERS 作最终决定。

| ID / surface | 原 Shell 当前行为 | Go `0ef9188` 行为 | 结论与切换前验收要求 |
| --- | --- | --- | --- |
| P01 CLI | `bootstrap/atlas doctor\|render\|apply\|status --env <profile>`；apply 要求 `--approve-tier0`；支持 `-V/--version`、status `--check` [S-CLI] | `--config`、`--root`、`--tool-dir`；apply 另要求精确 `--approve-cluster`；`--version` 报 dev；命令专属 flags [G-CLI] | **DIFFERENT**：列出 CI、Task、runbook、操作员调用；批准版本化参数/help/version/默认值映射，测试无效及错配参数；不能直接覆盖旧入口 |
| P02 配置与身份 | 严格 `KEY=value` env + `versions.lock`；Identity v1；Kind 配置与 repo 绑定；ADR-0002 要求不兼容 v2 和独立迁移 [S-CONFIG]、[S-ADR2] | 严格 JSON + JSON lock；`atlas-refactor/identity/v1` 与配置/制品绑定；拒绝旧集群名和已有集群重绑 [G-CONFIG]、[G-STATE] | **GAP**：schema 映射、默认值、路径/重复键/未知字段与语义摘要测试；专用目标迁移器必须遵循规范，不能复制 `.state`/凭据或只增加 v1 忽略的字段 |
| P03 status / exit | TSV 五组件；普通 status 观察到非 ready 状态不据此失败；`--check` 汇总 ready=0、已知不就绪/cluster drift=1、未知/不可用=2；部分参数错误返回1，Bash版本/缺命令返回2 [S-CLI]、[S-STATUS] | JSON；status 总是严格，`--check` 是显式别名；ADOPTED=0，未完成/退化=1，DRIFTED/UNAVAILABLE=2；参数/配置错误2，一般操作失败1 [G-CLI] | **DIFFERENT**：raw stdout/stderr/schema/exit 分类矩阵与调用方迁移；不能只比较两个 `status --check` 都返回0 |
| P04 中断 / deadline | INT=130、TERM=143（EXIT cleanup 失败可能覆盖）；组件 timeout/wait；锁检查贯穿 mutation [S-CLI]、[S-LOCK] | context cancellation + 总 deadline；第四轮 Root 后 SIGINT 返回1，保留 latch；恢复仅提交 Receipt [G-CLI]、[E04] | **DIFFERENT / BOUNDED**：决定退出码兼容约定，验证 subprocess 结束与在途请求、锁/Fence 归属；超时/信号非零不代表 API 未提交 |
| P05 render / 资源身份 | Helm release `atlas-argocd`；Root `atlas-root`；`.state/rendered` Seed/Root；GitOps 用 vendored chart tree 与共享 values [S-RENDER]、[S-PROFILE] | release `atlas-refactor-argocd`；Root `atlas-refactor-root`；输出13文件，GitOps 消费渲染清单 [G-RENDER]、[E04] | **DIFFERENT**：相同已批准输入的语义 inventory/diff、intra-engine deterministic hash；保留目标名称/UID/chart/values/release 或单独批准迁移；不能以 JSON/YAML 格式差异掩盖资源替换 |
| P06 Git / 文件副作用 | render 写本地状态；GitOps 源由 env 指定（示例为 main）；现有目录/权限/锁规则 [S-CLI]、[S-RENDER]、[S-PROFILE] | apply 检查 clean worktree、已提交 render、远端 ref 精确等于 local HEAD；render 可写 `gitops/test`；私有 state、kubeconfig hash 绑定 [G-APPLY] | **DIFFERENT**：固定 refs 与 Git 可用性契约、Git reads 与显式发布动作分开；验证无隐式 commit/push/ref 修改及未经计划的本地文件写入；发布 docs 提交不能改变固定运行 ref |
| P07 Docker / Registry / 网络 | owner-local OrbStack；local profile 为多节点池；normal apply 先确保 cluster/Registry，之后 handoff；adoption 后仍允许 substrate Registry ensure [S-CLI]、[S-PROFILE]、[S-ADR2] | 单节点 ARM64 Kind；本地 archive 导入 image；无 local Registry 管理；拒绝环境变量改向，私有 kubeconfig [G-APPLY]、[G-CONFIG] | **GAP**：原拓扑、节点集合、Registry、端口、socket 和离线导入行为逐项验收或明确删减支持范围；网络规范保持；Kubernetes 零写入不证明 Docker/Registry 无副作用 |
| P08 Seed → adoption | 现行 self 启发式；Root create-only、漂移拒绝；规范目标为 protected Signal → Receipt，不以 health 阻塞 [S-HANDOFF]、[S-ADR2] | Root 前 latch；四 App 健康/精确 commit、Signal 与39 durable ownership 后 Receipt；36 tracking/SSA + 3 CRD spec SSA/inventory/sync [G-STATE]、[G-OWNERSHIP]、[E04] | **GAP / 规范差异**：先决定并实现 authority 时序；测试有效 Signal + unhealthy 仍提交规范 Receipt；Root已存在的迁移零创建；不把实验39项当所有目标 inventory |
| P09 idempotency / drift | normal Root 不覆盖；self 存在但不健康拒绝重 Seed；self 丢失可能暴露已记录 INV-02 gap [S-HANDOFF]、[S-ADR2] | 重复 apply API零写入；Root后中断只补 Receipt；drift/unhealthy拒绝并恢复；但约定尚无服务端删除保护 [G-APPLY]、[G-STATE]、[E04] | **BOUNDED / GAP**：迁移目标重复执行、proof缺失/UID冲突/读不可用、旧引擎重入，逐类记录 authority/readiness/exit；不得复制原实现缺口或删除证据以恢复 Seed |
| P10 offline / supply chain | vendored chart archive+tree、锁定工具/镜像，本地制品预先到位；Helm仅渲染 [S-README]、[S-RENDER] | stdlib-only，自动 Go/module 获取关闭；chart/hash/image 本地检查；apply/status 仍依赖 Git 和 API 可达 [G-TASK]、[G-APPLY]、[G-STATE] | **BOUNDED / GAP**：离线不是断开 GitOps source；分别验证 artifact acquisition 禁止、允许网络端点、断网构建、missing/corrupt artifact；完成 compiler/module/binary G3 发布证明 |
| P11 并发 / proof protection | 工作树锁有 PID/custody 检查；ADR-0003 要求目标 Operation Fence，Phase-0 canary 不能证明生产保护已激活 [S-LOCK]、[S-ADR3]、[S-PHASE0] | `.state/apply.lock` 仅同工作树；immutable 防更新不防删除；实验 admin 审计不是 capability 隔离 [G-APPLY]、[E04] | **GAP**：服务端 evidence protection、principal/capability fencing、跨执行器互斥和 lost-holder 处置；同时阻断旧 Shell 与普通管理员路径须符合已接受保护模型，不能以两个本地锁推导唯一 authority |
| P12 recovery / rollback / scope | Accepted ADR-0003、0004、0005；已实现 Phase-0 canary，不代表 full Seed recovery、legacy migration 或 production readiness [S-ADR3]、[S-ADR4]、[S-ADR5]、[S-PHASE0] | 没有 recovery/drill/migration；不重绑现存集群；当前支持独立 disposable profile [G-ARCH] | **GAP**：完成规范 rollout 与专用迁移，验证合法 pre-transition rollback、兼容 fallback、旧版本 downgrade fence、退役；更换可执行文件不是完整回退路径 |

## 证据强度与下一步

第四轮足以证明当前 Go disposable profile 的技术闭环；其 source、tools、render、
UID、audit、退出码均由[冻结清单](evidence/bootstrap-baseline-20260927.json)绑定。
本轮重新校验了183份已索引私有证据和保留的测试 binary，摘要均匹配。没有重新运行
第四轮或验证当前集群仍保持当时状态；保存的 PASS 是当时的历史观察。

本次是静态 baseline audit，不是完整执行式 parity PASS。先审查 P08 的规范选择、
P02/P12 的迁移方案及 P11 的隔离边界，再围绕明确支持的迁移 profile 补实现和差分
fixtures。P01/P03/P04 的接口差异可以通过显式版本化契约处理，不能静默等同。
后续按[Cutover Contract](go-bootstrap-cutover-contract.md) G1–G6 与 R1–R7 验证；
全新 Go 集群的第五次成功不会关闭这些迁移缺口。

## 本次文档变更的验证

`task quality` 已通过：Go 1.27.1、格式、`go vet`、race-enabled 测试，并通过
`ATLAS_TEST_HELM` 显式启用了锁定 Helm 4.2.3 的真实渲染测试。另检查文档链接、
固定 source SHA 的文件路径、JSON 绑定、12 项历史文档摘要及工作树差异。
这些检查不运行集群，不构成 P01–P12 的执行式 parity 或 R1–R7 的迁移证据。

## 固定来源

以下 source links 固定到审计 SHA；它们是行为判断的依据，不是对 main 的持续跟踪。
原规范仍按 [Architecture][S-ARCH] > [GitOps][S-GITOPS] > [Network][S-NETWORK] >
Accepted ADR 的顺序控制。当前语言权威见[原 ADR-0001][S-ADR1]。

[S-ARCH]: https://github.com/snkio027/atlas/blob/aca4ff137a1d254cfeceaec24526e0699b585e92/docs/architecture/operating-model.md
[S-GITOPS]: https://github.com/snkio027/atlas/blob/aca4ff137a1d254cfeceaec24526e0699b585e92/docs/standards/gitops.md
[S-NETWORK]: https://github.com/snkio027/atlas/blob/aca4ff137a1d254cfeceaec24526e0699b585e92/docs/standards/network.md
[S-ADR1]: https://github.com/snkio027/atlas/blob/aca4ff137a1d254cfeceaec24526e0699b585e92/docs/adr/0001-retain-shell-bootstrap.md
[S-ADR2]: https://github.com/snkio027/atlas/blob/aca4ff137a1d254cfeceaec24526e0699b585e92/docs/adr/0002-monotonic-bootstrap-adoption-proof.md
[S-ADR3]: https://github.com/snkio027/atlas/blob/aca4ff137a1d254cfeceaec24526e0699b585e92/docs/adr/0003-bootstrap-break-glass-recovery.md
[S-ADR4]: https://github.com/snkio027/atlas/blob/aca4ff137a1d254cfeceaec24526e0699b585e92/docs/adr/0004-length-bounded-recovery-principal-identities.md
[S-ADR5]: https://github.com/snkio027/atlas/blob/aca4ff137a1d254cfeceaec24526e0699b585e92/docs/adr/0005-personal-local-target-materialization.md
[S-CLI]: https://github.com/snkio027/atlas/blob/aca4ff137a1d254cfeceaec24526e0699b585e92/bootstrap/atlas
[S-STATUS]: https://github.com/snkio027/atlas/blob/aca4ff137a1d254cfeceaec24526e0699b585e92/bootstrap/status/report.sh
[S-CONFIG]: https://github.com/snkio027/atlas/blob/aca4ff137a1d254cfeceaec24526e0699b585e92/bootstrap/lib/config.sh
[S-LOCK]: https://github.com/snkio027/atlas/blob/aca4ff137a1d254cfeceaec24526e0699b585e92/bootstrap/lib/lock.sh
[S-RENDER]: https://github.com/snkio027/atlas/blob/aca4ff137a1d254cfeceaec24526e0699b585e92/bootstrap/argocd/render.sh
[S-HANDOFF]: https://github.com/snkio027/atlas/blob/aca4ff137a1d254cfeceaec24526e0699b585e92/bootstrap/argocd/handoff.sh
[S-PROFILE]: https://github.com/snkio027/atlas/blob/aca4ff137a1d254cfeceaec24526e0699b585e92/env/local-orbstack.env
[S-PHASE0]: https://github.com/snkio027/atlas/blob/aca4ff137a1d254cfeceaec24526e0699b585e92/docs/runbooks/recovery-phase0.md
[S-README]: https://github.com/snkio027/atlas/blob/aca4ff137a1d254cfeceaec24526e0699b585e92/bootstrap/README.md
[G-CLI]: https://github.com/snkio027/atlas-refactor/blob/0ef918801dac84ac913416f4c1756bf08045a485/cmd/atlas/main.go
[G-CONFIG]: https://github.com/snkio027/atlas-refactor/blob/0ef918801dac84ac913416f4c1756bf08045a485/internal/atlas/config.go
[G-APPLY]: https://github.com/snkio027/atlas-refactor/blob/0ef918801dac84ac913416f4c1756bf08045a485/internal/atlas/apply.go
[G-STATE]: https://github.com/snkio027/atlas-refactor/blob/0ef918801dac84ac913416f4c1756bf08045a485/internal/atlas/state.go
[G-RENDER]: https://github.com/snkio027/atlas-refactor/blob/0ef918801dac84ac913416f4c1756bf08045a485/internal/atlas/render.go
[G-OWNERSHIP]: https://github.com/snkio027/atlas-refactor/blob/0ef918801dac84ac913416f4c1756bf08045a485/internal/atlas/ownership.go
[G-TASK]: https://github.com/snkio027/atlas-refactor/blob/0ef918801dac84ac913416f4c1756bf08045a485/Taskfile.yaml
[G-ARCH]: https://github.com/snkio027/atlas-refactor/blob/0ef918801dac84ac913416f4c1756bf08045a485/docs/architecture.md
[E04]: https://github.com/snkio027/atlas-refactor/blob/9df41568bafcaa98d6548d2089766d83081c3374/docs/integration-20260927-04-result.md
