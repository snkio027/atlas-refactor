# 四节点开发环境

本轮全新创建及自动验收已通过，实例化提交为 `f6d35ec44dcd12b3812145a561590a94af558ce6`。
实际执行 6 分 35 秒，最终四节点 Ready、26 个 Pod Ready、15 个 Application Synced/Healthy，
52 项持久 Seed ownership 成立，External Root 只创建一次，重复 apply 零 Kubernetes 写入。
HTTP 301 跳转 HTTPS，使用本地公有 CA 校验 HTTPS 200，PVC 为 Bound / Retain。
完整摘要、SHA、工具链、UID 与审计见[运行时证据](evidence/development-four-node-20260927.json)。

初次 `13ea2bb` 的 OCI 目录条目预检失败发生在创建集群之前；修复后才从零执行上述成功运行。
创建后的 Gateway API 曾显示 Degraded，随后由 Argo 自行收敛；未手工 refresh/sync 或 patch。
原来的 5 个 Kind 集群已按用户要求删除，本机仅保留这个四节点集群。旧 Git 证据未改写。

目标为 `atlas-refactor-test-dev02`，配置见 `profiles/development.json`（schema 3）。
固定角色：control-plane；worker=gateway；worker2=compute；worker3=data。
API 与 Web 入口均只暴露到 loopback。原有 dev01/四轮测试记录保留历史含义。
设计变化见 [ADR-0005](adr/0005-four-node-development-workflow.md)，状态 Proposed。

## 一次性准备

预装锁定工具：Go 1.27.1、Helm 4.2.3、Kind 0.32.0、kubectl 1.36.3、yq 4.53.6、Lua 5.5.1，
以及 Git、Docker / OrbStack。工具安装不由 Bootstrap 隐式执行。
`--tool-dir` 包含 Helm、Kind、kubectl；Docker/Git 从 PATH 获取。
先完成 `task quality`，提交并发布到 `codex/development-platform`，远端 SHA 必须等于本地 HEAD。

```sh
task build
go build -trimpath -o bin/atlas-dev ./cmd/atlas-dev
./bin/atlas-dev up --tool-dir /absolute/path/to/locked-tools \
  --approve-cluster atlas-refactor-test-dev02 --approve-tier0 --install-kubeconfig
```

此命令会准备镜像、验证离线闭包、创建四节点、导入各节点镜像、Cilium-first Seed、
一次性实例化 Root、等待 GitOps 接管、验证网络/TLS/PVC、重复 apply 并检查审计。
只有准备阶段可获取缺失镜像；正常 Bootstrap 仍是离线制品消费者。GitOps 匿名 HTTPS
读取 GitHub，所以整个工作流不宣称断网可运行。

执行仓库和私有状态固定保存在 `.state/development/atlas-refactor-test-dev02/repo`；
结果为同级 `latest-run.json`，公有 CA 为 `development-ca.crt`。不会再依赖临时目录。
同名活跃任务有 run.lock；异常结束时先确认旧进程和证据，不能盲删锁或 `.state`。
集群丢失而本地绑定仍存在时，正常流程拒绝重新初始化；删除/新建目标仍需独立授权。

## 日常检查

```sh
kubectl config current-context
kubectl get nodes -L node-role.local/gateway,node-role.local/compute,node-role.local/data
kubectl get pods -A -o wide
kubectl -n argocd get applications
kubectl -n workload-web get pvc
./bin/atlas-dev verify --tool-dir /absolute/path/to/locked-tools
curl --resolve web.atlas.test:8443:127.0.0.1 \
  --cacert .state/development/atlas-refactor-test-dev02/development-ca.crt \
  https://web.atlas.test:8443/
```

`--install-kubeconfig` 会先备份 `~/.kube/config`，合并新 context 并将其设为 current-context。
其他 context 保留。若 shell 设置了 KUBECONFIG，应先 unset；不修改你的 shell 启动文件。
CA 只导出公有证书，不自动加入操作系统信任。默认 kubectl 的 context 修改不改变 Bootstrap
自己的凭据绑定。verify 只读集群，并在节点、GitOps、调度、PVC 或 HTTPS 不满足时非零退出。

四节点仍不是 HA，Retain 不保证删除 Kind 容器后数据保留。此平台用于开发；不要保存唯一数据。
