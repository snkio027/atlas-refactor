# 架构与生命周期

此文档描述当前实验性实现。适用范围是单个所有者、单工作树、独立可丢弃的
OrbStack/Kind 测试环境。初始架构决策见 Proposed ADR-0001。

## 权责

Git 保存定义；Go Bootstrap 建立最小控制面；Argo CD 持续同步 Git 定义；
Kubernetes 与领域 Operator 承担运行时调谐。Bootstrap 不成为常驻 controller。

```text
Go Bootstrap
  ├─ Kind test substrate + local image loading
  ├─ Argo CD Seed
  ├─ atlas-bootstrap AppProject
  └─ External Root (never a child of another Application)
       ├─ project-bootstrap (-20) → platform-project
       └─ platform-control (-10) → argocd-self → same rendered Seed + signal
```

第一阶段没有 workload 应用，因此没有预先建立空的 Workload 控制树。
Root 与子 Application 不带级联删除 finalizer；对子 Application 的自动删除
使用确认保护。平台资源的删除语义仍须逐项审查，不能从控制图保护推断数据保护。

Helm 仅在本地渲染，没有 Helm Release 状态。GitOps 消费已渲染的本地清单，
无需 Argo CD 在线获取 Helm chart。`argocd-cm` 的 Application 健康检查要求
子应用没有活动操作且为 Synced，避免将目录顺序误当成就绪依赖。

## 状态与写权限

| 状态 | 正常 Bootstrap 权限 |
| --- | --- |
| ABSENT | 经双重显式批准且不存在旧目标证据时，创建专属测试集群 |
| FRESH | 验证身份和节点后，安装 Seed；初次建立 Root |
| HANDOFF_PENDING | 只读观察；有效接管证据就绪后创建 Receipt |
| ADOPTED | 只读；再次 apply 不产生写操作 |
| ADOPTED_DEGRADED | 报告退化；拒绝正常修复 |
| DRIFTED / UNAVAILABLE | 拒绝写入 |

Identity 绑定配置与制品锁摘要。Root 创建前先以 create-only 请求写入 Handoff
latch，绑定 Identity UID 与配置指纹。此后永远不能由正常流程重新获取 Seed
权限，即使后续步骤失败。Receipt 绑定 Identity、Root、argocd-self 与 Signal
的 UID。Signal 必须来自 argocd-self 的资源清单并带有对应 Argo tracking ID。

latch 才是正常路径撤销 Seed 权限的边界；Receipt 记录观察到的接管完成。
健康状态不决定是否恢复 Seed 权限。旧 Atlas 的状态格式或凭据不会被导入。

第二轮真实测试表明，Synced/Healthy 与有效 Signal 仍不足以证明 Seed 接管。
`argocd-self` 必须完整同步；观察器核对四个 Application 的实际 commit，并逐个
读取锁定渲染中的全部 39 个持久 Seed 对象，检查与其资源类型相符的接管证据。
36 个普通对象要求 tracking ID 与 Argo SSA；集群级对象的 tracking namespace
使用 Application 的目标 namespace。Argo CD 3.5.1 不给 CRD 注入 tracking，
因此 3 个 CRD 另需当前 Synced 清单、精确 commit 的成功同步结果，以及 Argo
SSA 对 spec 的字段记录。CRD 仍在 Gate 内，不能仅凭对象存在或 Healthy 放行。
这些条件在 Receipt 前后都必须成立。Helm hook 的短暂 Job 和辅助 RBAC/SA
不承担持久接管证据。清单解码使用 client dry-run，集群访问只读；无法解析 Git
revision 或读取 ownership 时失败关闭。详见第二、三轮失败记录。

管理员可删除这些记录，所以本实现不宣称具备服务端强制的永久降权。
更广的共享或生产部署仍须先完成独立 Recovery、证据保护、跨执行器互斥以及
恢复演练。用户另行提出先构建独立可丢弃的 Web/API 开发环境，范围与延期项由
[Proposed ADR-0004](adr/0004-local-web-development-platform.md) 单独定义；schema 2 实现 Cilium-first 启动与双 Seed 检查。
用户已授权启动独立开发目标；其运行时结论单独记录。

## 实现边界

`cmd/atlas` 只负责参数、输出、信号和总 deadline。`internal/atlas` 的文件分别
承担严格配置、确定性渲染、进程适配、状态观测与执行流程。通过 Runner 接口
注入外部工具；模拟器记录完整副作用序列。后续扩大职责时再拆分独立包。

配置采用严格 JSON，拒绝重复键、未知字段、尾随数据、路径逃逸和符号链接。
镜像要求版本和 digest，模板与 chart 要求 SHA-256。Go 无外部模块依赖。
配置和制品与身份绑定后，修改它们会产生漂移；本阶段不提供自动升级或迁移。

工具执行使用参数数组与有限环境，不通过 shell。Docker 绑定 owner-local
OrbStack socket。Kubernetes 操作必须使用本程序创建且摘要未变化的私有
kubeconfig 和精确 Kind context；不继承默认 kubeconfig。子进程失败时不回显
可能包含凭据的 stdout/stderr。

## 失败处置

不自动回滚集群写入，不删除失败集群，不跨信任边界恢复。中断后保留 latch、
Receipt 和本地状态；重新检查后，只续行当前权限允许的步骤。
Root 创建失败可能留下无法继续的测试环境，这种情况必须由所有者独立处置。
当前缺少恢复命令是明确的发布限制，不由模拟契约测试替代。

## 集成审计

新测试节点使用 `kind-ipv4-audit/v1` substrate profile，作为 Identity 的一部分。
API 监听 loopback。Metadata-only 审计只记录写请求，不记录正文；私有日志
保存在 `.state/audit/`。审计用于区分 Bootstrap、Kubernetes 与 Argo 的实际写入，
并不授予恢复或 Admission 权限。

## 独立开发平台候选

`gitops/{root,platform,workloads}/` 的 development overlays 实现网络、TLS、本地存储与 Web 示例。
它们没有替换 `gitops/test`。schema 2 在同一正常 Bootstrap engine 中把初始 Cilium Seed
纳入既有 durable latch 边界；交接后没有新的持续 mutation authority。
`cmd/atlas-platform` 只渲染和校验本地文件；它不是第二个 Bootstrap engine。
完整控制图、当前验证边界和 Cilium-first 启动前置工作见
[开发平台与部署审查](development-platform.md)。

## 四节点开发 profile 的增量边界

schema 3 的配置、离线制品准备、首次接管与持续观察的区别见
[Proposed ADR-0005](adr/0005-four-node-development-workflow.md)。上述 schema 1/2 历史 Gate
保持原含义；schema 3 在有效 Receipt 后允许 CRD 最近成功同步 SHA 早于当前无资源差异的
Git SHA，但仍检查当前 Sync/Health、资源清单与 SSA ownership，首次接管要求不变。
