# OT-0：两对象 Argo ownership transfer probe

状态：首次运行在 SETUP 停止，A1–A6 未执行，尚无 ownership runtime proof。控制范围见 [Proposed ADR-0008](../../docs/adr/0008-ownership-transfer-probe.md)。
这是一次性实验材料，不是普通 Bootstrap、recovery 或 capability migration 命令。

## 精确目标与启动门禁

新建 `atlas-refactor-test-ot0`，仅一个 Kind control-plane，Docker context `orbstack`，API loopback。
安装锁定 Argo CD 3.5.1/Redis；Helm 仅本地渲染，server/ApplicationSet replicas=0，不开放 ingress。
使用固定 node image 自带的 Kind 网络。迁移资源恰好是 fixture 中的 Namespace/ConfigMap。
保留 dev02、所有旧状态和默认 kubeconfig；仅在新集群创建/修改 AppProject、Application 与实验 Seed。

先固定并推送完整 Git SHA，完成本地检查，再审查输出 plan。创建集群与执行 A1–A6 需要用户对
该 SHA、目标和双向 transfer window 的明确批准（仓库 AGENTS.md）。本材料不代表已获得批准。

```bash
# 在独立 atlas-refactor 仓库，clean checkout 且 HEAD=已审查 SHA
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly \
  go run ./experiments/ownership-transfer/check_images.go versions.lock.json .state/images
python3 -B experiments/ownership-transfer/prepare.py \
  --revision "$(git rev-parse HEAD)" \
  --output "$PWD/.state/ownership-transfer/ot0" --tool-dir "$PWD/.state/tools"
python3 -B -m unittest discover -s experiments/ownership-transfer -p 'test_*.py' -v
```

prepare 只写新私有目录，不联网、不接触集群。plan.json 绑定 commit、工具版本、镜像 digest、
fixture 和全部生成文件的 SHA。Argo 匿名从公开 repo 读取该 commit 下的 fixture；没有 live 分支移动。
`check_images.go` 复用 internal/oci 验证缓存中 node/Argo/Redis 的 digest 与完整 linux/arm64 closure。

## 经批准后的操作顺序

此版用受审查清单和显式命令执行，不实现通用 migration CLI。所有 shell 命令必须在独立 repo
执行，设置 `DOCKER_CONTEXT=orbstack`，清除外部 DOCKER_HOST/KUBECONFIG 干扰，并显式指定专用
`--kubeconfig .state/ownership-transfer/ot0/kubeconfig --context kind-atlas-refactor-test-ot0`。
任何命令非零退出或阶段断言失败立即停止；不得删除失败现场或自动 rollback。

1. 检查 plan 所列 SHA；确认目标集群、目标 node 容器和专用 kubeconfig 均不存在。
   以 plan/kind.json 和锁定 nodeImage 创建新集群，`kind create cluster --name atlas-refactor-test-ot0
   --config <plan>/kind.json --kubeconfig <plan>/kubeconfig`；不导出到默认 kubeconfig。
2. 仅向新节点导入已验证的 Argo/Redis OCI archive（ctr --platform linux/arm64 --digests）；
   为锁定的完整 repo:tag@digest 建立 containerd alias，并验证该 alias 的 index digest。
   不执行 docker pull、不操作 dev02 容器；Kind node image 必须已在 host cache。
3. 在新集群 create argocd namespace / configmap，使用 SSA 应用私有 seed.json，等待 CRD Established、
   application-controller/repo-server/redis 就绪。记录 Pod spec image 与运行 imageID、版本及 command exit codes。
   不输出或收集任何 Secret。记录 kube-system UID、私有 kubeconfig SHA 到 binding.json，字段是
   `cluster`、`clusterUID`、`kubeconfigSHA256`，文件权限 0600。所有后续观察核对此绑定。
4. create project.json。对下表逐阶段执行；每次 sync 等到当前 `ot0-stage` 的 operation 终止且
   Application 没有活动 operation，最大 5 分钟。只做 GET 观察，不执行 refresh/restart。
   超时保存当前快照并停止。删除 App 前要求无 finalizer、operation idle、UID 与上一阶段相符。
5. 每个阶段调用下方 observe；任何非零结果保留现场且停止。**A3/A6-strict 的同步失败是预期结果，
   但 observe 必须返回 0，确认失败原因确为共享资源且两个对象完全未变。**

| 阶段 | Application 操作 | 显式同步 patch |
| --- | --- | --- |
| A1 | create A1-app.json | A1-sync.json |
| A2 | delete owner-a --cascade=orphan --wait=true | 无 |
| A3 | create A3-app.json | A3-sync.json |
| A4 | SSA apply A4-app.json（同一个 owner-b） | A4-sync.json |
| A5 | SSA apply A5-app.json | A5-sync.json |
| A6-release | delete owner-b --cascade=orphan --wait=true | 无 |
| A6-strict | create A6-strict-app.json | A6-strict-sync.json |
| A6-transfer | SSA apply A6-transfer-app.json（同一个 owner-a） | A6-transfer-sync.json |
| A6-restored | SSA apply A6-restored-app.json | A6-restored-sync.json |

Application 写入使用同一 `--field-manager=ot0-ceremony`；同步用
`kubectl patch application <owner> -n argocd --type=merge --patch-file <stage>-sync.json`。
禁止对 Namespace/ConfigMap 执行 apply、patch、annotate、delete；Argo 始终是其 tracking writer。
人工门禁只授权这两个已审查资源的窗口，并非降低 AppProject 或扩大资源集合。

```bash
python3 -B experiments/ownership-transfer/observe.py \
  --stage A1 --plan-dir "$PWD/.state/ownership-transfer/ot0" --tool-dir "$PWD/.state/tools"
```

逐阶段将 `A1` 替换为表中阶段。observe 只执行 GET/config-view，保存 create-only 原始快照和结果。
比较资源 UID、全部内容、RV/managedFields、精确 tracking、AppProject、操作 info/SHA/模式/结果、
当前 Sync/Health 和 source owner 缺席。不会以旧成功 operation 或旧 Healthy 放行。

实验结束只允许报告 OT0_TRANSFER_WINDOW_VERIFIED；13 对象和 Bootstrap authority 不在本实验内。
关闭窗口是 A5/A6-restored 的明确同步阶段，没有自动计时恢复；中断时报告遗留的 strict/transfer
状态并保持停止。保留新集群和证据，后续清理须另行授权。

## 首次运行与受审查续行

ac90898 的首次运行已创建独立 OT-0 集群并安装 Seed，但生成的 argocd-cm 缺少
`app.kubernetes.io/part-of=argocd`，被 settings informer 过滤，controller 因此退出。
A1 尚未开始，无 probe Application、无两个 probe 对象、无 transfer window。
原计划/日志留在 `.state/ownership-transfer/ot0`；失败证据见
[attempt01](../../docs/evidence/ownership-transfer-ot0-20260927-attempt01.json)。

修正仅增加 argocd-cm 的 Argo 识别标签。新的 immutable plan 写入独立目录 `ot0-retry1`，
不覆盖首次计划。续行需要新的明确批准：核对原 binding 的 cluster UID、kubeconfig SHA、
argocd-cm UID、原始 data 与标签为空，确认仍无 probe Application/Namespace，随后仅更新
argocd-cm 的两个 labels，等待 Kubernetes 正常重试拉起 controller。禁止重建集群、替换
Seed、主动 restart 或修改 Secret。完成 readiness 后再创建 AppProject，并按原 A1–A6
序列执行，使用新计划的统一 commit SHA。任何再次异常仍停止；OT-1/dev02 继续不获批准。
