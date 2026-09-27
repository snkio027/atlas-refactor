# ADR-0008：受限 Ownership Transfer Window 探针（OT-0）

- Status: Proposed
- Date: 2026-09-27
- Parent: [ADR-0007](0007-platform-contract-hardening.md)
- Scope: 独立一次性 Argo 实验；不批准 dev02、OT-1、生产迁移或修改普通 Bootstrap/select

## 控制权与问题

沿用原 Atlas Architecture v1.0.2、GitOps v1.0.3 的 Git authority、有限实例化与 Argo 调谐边界。
这是独立 atlas-refactor 的实验提案，不是原 Atlas 规范的替代记录。
Ownership Migration 与 enable/retire 不同：资源 identity、期望内容保持，GitOps owner 改变。

锁定 Argo CD v3.5.1 在 [annotation tracking](https://github.com/argoproj/argo-cd/blob/v3.5.1/util/argo/resource_tracking.go#L48-L84)
中从 live 对象解析 ApplicationName；[共享判断](https://github.com/argoproj/argo-cd/blob/v3.5.1/controller/state.go#L734-L743)
不在该判断中查询旧 Application 是否仍存在。[同步入口](https://github.com/argoproj/argo-cd/blob/v3.5.1/controller/sync.go#L140-L145)
在 FailOnSharedResource=true 时拒绝共享资源。由此预期：非级联删除旧 App 留下 stale tracking，
strict 新 App 接管失败。运行时必须验证这个反例；若直接成功，停止并重新分析。

## 候选机制与边界

仅对本次精确目标，提出一次性显式 transfer window：旧 App 退出后，新 App 的
FailOnSharedResource=false，ServerSideApply=true；由 Argo 正常同步相同 Git 对象并写 tracking，
之后恢复 FailOnSharedResource=true 并再做一次显式同步。反向转移必须走同样的窗口。
不手工修改/删除 tracking annotation，不使用 force apply、资源删除重建或自动 rollback。

这是一项待批准、待实测的局部例外；不撤销 ADR-0007 对普通路径的 shared-resource 保护，
不在 atlas-platform select/render 或 Bootstrap 中增加迁移入口。窗口不是服务端自动过期租约；
超过单阶段 5 分钟或任一断言失败都停止，保留现场。此时窗口可能仍开启，报告必须明确指出，
不能为了输出“成功”而自动恢复/继续。后续人工处置需要针对实际状态重新审查。

状态：PREPARED → SOURCE_RELEASED → TRANSFER_ALLOWED → TARGET_ADOPTED →
STRICT_RESTORED → VERIFIED。VERIFIED 仅表示 OT-0 两对象实验，不等于 OWNER_TRANSFER_SUPPORTED。

## 精确实验对象

- 新 Kind 集群 atlas-refactor-test-ot0；OrbStack、一个 control-plane、API 只监听 127.0.0.1。
- 复用 versions.lock.json 的 Kind 0.32.0 / Kubernetes 1.36.1 / Argo CD 3.5.1 / Redis digest 与 chart 校验。
- Kind 默认网络由固定 node image 提供；这是最小 Argo substrate，不是 Atlas Cilium profile 等价验证。
- Application owner-a / owner-b，没有自动同步、没有 finalizer；显式 sync 固定到同一完整 Git SHA。
- 仅迁移 Namespace/ot0-probe 与 ConfigMap/ot0-probe/probe。
- 实验 AppProject ot0-probe 只允许此 repo、ot0-probe destination、Namespace/ConfigMap kinds。
  AppProject 不能按 cluster object 名称隔离；精确对象白名单由已审查 fixture 和只读验证约束。
  这个独立 Argo 实验的 AppProject 名不改变 Atlas canonical Projects。
- 专用私有 kubeconfig 和 kube-system UID 绑定；不合并默认 context，不读取 dev02 凭据。
- 不创建 Root/Identity/Latch/Receipt/Signal，不测试 Bootstrap，也不删除任何既有集群。
- 试验后保留新集群及证据；删除是另一个明确授权动作。

## 正反向 Gates

| 阶段 | 动作 | 必须观察到 |
| --- | --- | --- |
| A1 | owner-a strict 显式同步 | 精确 SHA、idle/Synced/Healthy；记录全部对象与 UID/RV/managedFields/tracking |
| A2 | 非级联删除 owner-a | App 消失；两个对象完整不变，tracking 仍为 A |
| A3 | owner-b strict 显式同步 | 必须因 SharedResourceWarning 失败；对象完整不变 |
| A4 | 同一 owner-b 打开窗口并显式 SSA 同步 | UID/内容不变；tracking=B；精确 SHA、idle/Synced/Healthy |
| A5 | 同一 owner-b 恢复 strict 并再同步 | 成功且无 shared warning；App UID 与资源 UID 不变 |
| A6-release | 非级联删除 owner-b | 对象完整不变，tracking=B |
| A6-strict | owner-a strict 同步 | 同样必须共享冲突失败，不能用 git revert 冒充 rollback |
| A6-transfer | 同一 owner-a 打开窗口并同步 | 同资源 identity/内容回到 A |
| A6-restored | owner-a 恢复 strict 再同步 | 精确 SHA、idle/Synced/Healthy，无共享警告 |

资源内容比较排除 API 的 UID/RV/creationTimestamp/generation/managedFields/status 和唯一的
argocd.argoproj.io/tracking-id；其他 label、annotation、data、spec、finalizer、ownerReference 都参与比较。
所有字段的原始值另外保存，managedFields 必须有 Argo SSA。A2/A3 及反向同阶段使用完整原始
对象相等断言；不是仅比较 spec。每次 sync 用独立 operation info 标识，避免接受前一阶段结果。

任意阶段不得同时存在两个可同步 owner；AppProject UID/内容和 kube-system UID 全程保持。
没有 refresh/restart、tracking patch、降级后跳过核验或隐式恢复。读失败不能视为 App 不存在。

## 证据和后续门禁

实验清单与只读验证器见 [OT-0 runbook](../../experiments/ownership-transfer/README.md)。原始观察、
专用 kubeconfig、命令记录与 SHA 摘要保存在 .state/ownership-transfer/ot0/；kubeconfig 和 Secrets
不进入 Git。公开报告只包含审查后的非敏感结果，明确源 commit、工具/镜像、render hash 和退出码。

OT-0 通过后仍须独立提出 OT-1：精确 13 对象、UID/内容保持、三个新 owner、正反向转移及保护恢复，
并验证 AppProject、Bootstrap Identity、Root/Latch/Receipt/Signal 和重复 apply 零写入。
本次不执行 OT-1，不扩展 A5/P1/W1/X1，不把 probe 成功外推为 dev02 迁移批准。
