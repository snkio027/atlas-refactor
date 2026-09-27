# OT-1 同集群续行：F1–F5 完成与待批准计划

**本地验证与只读基线预检通过；29 阶段 ceremony 尚未开始。**
实施 SHA 为 `4ec600554b09e696f752435924de9df2cb69257e`，atlas-ot1 binary SHA256 为
`3d34ec736f2f8d33464057e81d94d678d8f6656500c53be44426d3a561c0963f`。
ADR-0009 保持 Proposed。

## 本轮修订

F1/F2/F3 在独立提交 `76f2cba` 中完成：仅编译后的 Step.Sync 要求 operation proof；
已验证 typed list 仅补缺失 TypeMeta；全局 deadline 按计划推导为 160 分钟。
[F123 预检报告](ot1-f123-preflight-20260927.md) 和失败快照保持历史原文。

随后用户单独授权补充两项窄兼容，提交 `4ec6005`：
- F4：只有 hook=true 且 status 字段缺失的比较条目不承担同步状态要求。显式错误状态、
  不健康资源、非 hook 缺少 status 仍阻塞；Application spec/revision/idle/conditions 检查保留。
  hook operation 的失败或未知结果不被总体 Healthy 覆盖，运行中只能等待。
  [Argo CD 3.5.1 源码依据](https://github.com/argoproj/argo-cd/blob/v3.5.1/controller/state.go#L894-L899)。
- F5：仅在基线内容对比时规范化 networking.k8s.io/v1 NetworkPolicy 已知 selector 路径内
  的空 matchLabels map；不移除 selector，不忽略非空 label、expression、端口或策略变化。
  原始 GET、冻结 inventory hash、后续 UID/semantic 不变性规则保持。
  [Kubernetes v0.36.1 编码依据](https://github.com/kubernetes/apimachinery/blob/v0.36.1/pkg/apis/meta/v1/types.go#L1209-L1222)。

定向正反向回归与完整 task quality 均通过，覆盖 Go vet/race、真实 Helm/Kustomize、
348-resource 平台检查、Python 契约及四个固定 fixture。五个 CLI 均为同一 SHA 的 clean build。
旧 a8ddf72 与 76f2cba 二进制分别留存。详见 [可机读验证记录](evidence/ot1-continuation-ready-20260927.json)。

## 新只读预检

目标仍是 `atlas-refactor-test-ot1`，cluster UID `30a366bd-41f9-48b0-9330-70dffaf62662`。
专用 kubeconfig SHA256 `0a10ec362b40f3f2b0d7e09d7bf17951955cf1fe4776040e69bfca3160b9aa0a`。
入口保留 `127.0.0.1:18080/18443`，1 control-plane + gateway/compute/data。

- 新 binary capture = VERIFIED / 0；4 Nodes、4 AppProjects、24 Applications，inventory 与首尾 fence 无错误。
- 所有 App 精确观察 `541705df77a500d784f808fe036794b5e2d551d8`，idle/Synced/Healthy。
- 离线 Assess 的 baseline ownership=VERIFIED、reasons=[]；13 对象 Git/live 内容匹配、
  tracking 仍属 capability-foundation，Argo SSA 存在。
- 普通 Bootstrap status=ADOPTED / 0。独立 Gate-B 未执行，因此此处 Atlas=NOT_PROVEN；
  这不是 29 阶段或迁移协议成功，不把只读预检当作完整演练。
- 当前快照与 F123 快照的 13 个对象 UID/semantic 和 Bootstrap identity 内容一致。

原 attempt-01 永久为 STOP，第 0 阶段、request intent=0。旧运行区间审计复核 kubectl mutation=0。
原 plan/attempt/log/锁与历史报告 hash 未变；F123 plan/capture/报告也保留。
本轮只读预检没有 cluster mutation、清锁、凭据变更、后续 Git 阶段发布或默认 kubeconfig 修改。

## 待批准执行身份与范围

新 plan SHA256：
`c26950f5c7d6920fb1f6571db504e2bcfbe32bbdf98557480307578ecdf76452`。

私有目录：`.state/ot1-continuation-20260927-f45`。
新 attempt：该目录下 `transfer-attempt-02`（尚未创建）。
runtime checkout：`.state/ot1-live-20260927/source/.state/development/atlas-refactor-test-ot1/repo`。
源分支：公开仓库 snkio027/atlas-refactor 的 `codex/ot1-desired-state`，当前须保持 541705d。

批准涵盖一次完整的 29 阶段有限执行：13 对象的部分回退与 1→3→1，
预期 strict refusal、至多一个临时窗口、strict 恢复、父级 detach/reattach 与独立 Gate-B。
七个 Git revision 的内容与原计划一致；每次只发布计划内精确 fast-forward SHA。
ceremony 只操作四个 foundation Application 的精确生命周期/模式；
13 个对象内容与 tracking 由 Argo 调谐。Root、AppProject、Seed、凭据与默认 kubeconfig 不变。

每阶段最多 300 秒，总预算 29×300+900=9600 秒（160 分钟），保留外部取消。
任一未知、漂移、意外失败或中断即 STOP，保留新锁、证据与现场；
不自动修复、重试 mutation、回滚、关闭窗口或重建集群。

## 旧锁的显式处置

这次批准还需明确包含旧锁的一次性确认处置。旧锁为 runtime checkout 内
`.state/ot1-run.lock`，SHA256：
`04f6127033fc1aff72a685b497408d23b33824e1cc86b979ff4e631f2f5d9b4e`。

1. 开始前再次核对原 STOP、零 request/审计 mutation、目标 UID、专用 kubeconfig hash、
   13 对象 owner=capability-foundation、当前来源与完整基线状态；不匹配则停止。
2. 以 create-only 方式保存原锁原字节和本次明确批准记录；只有锁的完整摘要匹配才显式移除。
3. 使用上述新 binary、plan 与新 attempt 启动 run，由 run 创建自己的锁。
   不复用旧 attempt，不更改 run 为自动 unlock，不覆盖旧 STOP。
4. 新 run 若失败，保持新锁和现场；未来续行仍需新的观察、计划与决定。

新 plan 文件及 inspect-plan 请求表只读可审查；下表给出完整对象和阶段范围。

## 13 个对象

| identity | 原 owner | 目标 owner |
| --- | --- | --- |
| networking.k8s.io/v1/NetworkPolicy/atlas-monitoring/monitoring-ingress | capability-foundation | observability-foundation |
| networking.k8s.io/v1/NetworkPolicy/atlas-storage/s3-clients | capability-foundation | storage-foundation |
| networking.k8s.io/v1/NetworkPolicy/atlas-storage/s3-default-deny | capability-foundation | storage-foundation |
| networking.k8s.io/v1/NetworkPolicy/workload-web/s3-client-egress | capability-foundation | storage-foundation |
| v1/LimitRange/atlas-monitoring/defaults | capability-foundation | observability-foundation |
| v1/LimitRange/atlas-secrets/defaults | capability-foundation | secrets-foundation |
| v1/LimitRange/atlas-storage/defaults | capability-foundation | storage-foundation |
| v1/Namespace//atlas-monitoring | capability-foundation | observability-foundation |
| v1/Namespace//atlas-secrets | capability-foundation | secrets-foundation |
| v1/Namespace//atlas-storage | capability-foundation | storage-foundation |
| v1/ResourceQuota/atlas-monitoring/platform-budget | capability-foundation | observability-foundation |
| v1/ResourceQuota/atlas-secrets/platform-budget | capability-foundation | secrets-foundation |
| v1/ResourceQuota/atlas-storage/platform-budget | capability-foundation | storage-foundation |

## 七个固定 Git revision

| 名称 | SHA |
| --- | --- |
| baseline | 541705df77a500d784f808fe036794b5e2d551d8 |
| detached-forward | 3bb0790ae33ab870cde4c189b6c8ea1af1c1a62e |
| detached-mixed | f49e902217d64eb1985c51eb82743ddfafd544c3 |
| detached-reverse | e2b84334824e4564f720b30eb963d09c12289755 |
| forward-restored | ca6071b67306c42d53752c8f2fcccf2b406182e6 |
| mixed-restored | 0bd13e5e1e26750d5689081c6faeaea3610ad5e4 |
| reverse-restored | 4185933885b150aa9175e1dccd2ef1e9cb8b9066 |

## 29 个阶段

| 阶段 | 预期结果 | Git revision |
| --- | --- | --- |
| BASELINE_ADOPTED | success | baseline |
| SOURCE_RELEASED | released | detached-mixed |
| MIXED_SECRETS_STRICT_REFUSED | blocked | detached-mixed |
| MIXED_SECRETS_ADOPTED_WINDOW | success | detached-mixed |
| MIXED_SECRETS_STRICT_RESTORED | success | detached-mixed |
| MIXED_OBSERVABILITY_STRICT_REFUSED | blocked | detached-mixed |
| MIXED_OBSERVABILITY_ADOPTED_WINDOW | success | detached-mixed |
| MIXED_OBSERVABILITY_STRICT_RESTORED | success | detached-mixed |
| MIXED_ROLLBACK_TARGETS_RELEASED | released | detached-mixed |
| MIXED_ROLLBACK_SOURCE_STRICT_REFUSED | blocked | detached-mixed |
| MIXED_ROLLBACK_SOURCE_READOPTED_WINDOW | success | detached-mixed |
| MIXED_ROLLBACK_SOURCE_STRICT_RESTORED | success | detached-mixed |
| MIXED_ROLLBACK_VERIFIED | success | mixed-restored |
| SECOND_SOURCE_RELEASED | released | detached-forward |
| FORWARD_SECRETS_STRICT_REFUSED | blocked | detached-forward |
| FORWARD_SECRETS_ADOPTED_WINDOW | success | detached-forward |
| FORWARD_SECRETS_STRICT_RESTORED | success | detached-forward |
| FORWARD_OBSERVABILITY_STRICT_REFUSED | blocked | detached-forward |
| FORWARD_OBSERVABILITY_ADOPTED_WINDOW | success | detached-forward |
| FORWARD_OBSERVABILITY_STRICT_RESTORED | success | detached-forward |
| FORWARD_STORAGE_STRICT_REFUSED | blocked | detached-forward |
| FORWARD_STORAGE_ADOPTED_WINDOW | success | detached-forward |
| FORWARD_STORAGE_STRICT_RESTORED | success | detached-forward |
| FORWARD_VERIFIED | success | forward-restored |
| REVERSE_TARGETS_RELEASED | released | detached-reverse |
| REVERSE_SOURCE_STRICT_REFUSED | blocked | detached-reverse |
| REVERSE_SOURCE_READOPTED_WINDOW | success | detached-reverse |
| REVERSE_SOURCE_STRICT_RESTORED | success | detached-reverse |
| REVERSE_VERIFIED | success | reverse-restored |
