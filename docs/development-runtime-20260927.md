# 本机 Web/API 开发集群启动记录

2026-09-27（Asia/Shanghai），`atlas-refactor-test-dev01` 已启动并通过严格交接检查。
审核入口：[PR #1](https://github.com/snkio027/atlas-refactor/pull/1)。本次为有人工介入的独立开发验证，
不是原 Atlas cutover、生产验收或最终实现的无人值守全新安装证明。ADR-0004 保持 Proposed。

## 精确基线与结果

- 初次实例化：`9557cca9c95a1aafcc9b7d476a018fcc6d5967e4`。
- 修正及交接完成：`83b997bc5a8b603f751c068851360532149edd80`。
- OrbStack，单 control-plane，Kubernetes 1.36.1 / linux arm64；全部 15 个锁定镜像预先准备。
- 15 个 Application 在修正 SHA 上全部 Synced / Healthy；`status --check` 为 `ADOPTED / 0`。
- 52 项持久 Seed 接管：Argo 39 项（含 3 个 CRD）、Cilium 13 项。
  普通对象核对 tracking ID 与 Argo SSA；CRD 核对当前 inventory、成功同步 SHA 和 spec SSA。
- Metadata-only 审计：Bootstrap 对 Root 只有一次成功 create；中断后续行只有一次 Receipt create；
  20:27:59–20:28:02 UTC 的重复 apply 退出 0，Bootstrap 写请求数为 0。
- HTTP 8080 返回 301，Location 为 `https://web.atlas.test:8443/`；HTTPS 8443 使用公开开发 CA 验证后返回 200。
- `web-data` PVC 为 Bound，StorageClass 为 `atlas-local-retain`；Envoy bootstrap 实际 DNS 为 `V4_ONLY`。

精确 UID、工具版本、二进制摘要、47 项当前输入摘要、初始完整快照、Application revision、双 Seed ownership、
审计时间窗与退出码见[本轮证据](evidence/development-platform-runtime-20260927.json)。
原 Atlas、四轮旧集群、旧 tag 和历史报告均未修改。

## 实际遇到的问题和处理

1. **Docker 缺层导出。** `doctor` 的本地 image inspect 通过，但 cert-manager 的 `docker image save`
   只有约 13 KiB 元数据，containerd 导入报缺少 config digest。首次 apply 退出 1，未安装 Seed 或创建 Root。
   独立制品准备阶段对每个固定 digest 执行仅含 `FROM <locked-image>`、无 RUN 指令的本地 build，
   让 Docker 补齐原镜像的层内容，再验证原始 index、arm64 manifest/config/layers 的 SHA-256。
   没有部署 build 产物，没有改锁，也没有把在线下载放进 Bootstrap。
   该现象与 [Moby #52193](https://github.com/moby/moby/issues/52193) 的报告相符。
2. **CRD 初始健康缓存。** Kubernetes 中 Gateway CRD 已 Established，Argo 的资源树仍保留安装中状态。
   对 `gateway-api` 做一次正常 hard-refresh 后 Healthy；没有删除 CRD、重启 controller、忽略差异或降低健康条件。
3. **HTTPRoute 默认值 diff。** API 自动补全 parent group/kind、PathPrefix 和 backend group/kind/weight。
   Git 未显式声明导致应用 Healthy 但 OutOfSync。`83b997b` 将相同默认值写入 Git，Argo 随后收敛。
4. **启动身份与业务变更。** 初版把整个开发 bundle 当作持续身份，业务修正也会改写 Signal。
   `bootstrap/baseline.json` 保存 `9557cca` 的完整输入摘要，重算结果与已存在的 Identity 完全一致。
   Root、AppProject、控制图、Kind、镜像锁和两个 Seed 继续逐字节绑定，普通平台/业务叶子由 GitOps 演进。
   不改写 Identity/latch/Signal，不允许通过修改 Seed 继续正常 apply。对应单元测试与 task quality 已通过。
5. **当前 SHA 的 CRD 同步证据。** Argo 可把未变化应用标为新 commit 的 Synced，同时保留旧成功操作记录。
   正常同步一次 `argocd-self` 的既有审核分支后，3 个 CRD 的完整 SSA 证据对齐修正 SHA。
   Bootstrap 的续行始终只观察；审计确认它仅补了 Receipt。

## 本机查看与访问

凭据和完整审计保存在私有执行目录，不进入 Git。原始 9557cca 源码另有归档，执行 checkout
在保留同一 `.state` 的情况下推进到修正提交。以下命令不使用默认 kubeconfig：

```sh
export KUBECONFIG=/private/tmp/atlas-development-start-20260927/repo/.state/kubeconfig
kubectl --context kind-atlas-refactor-test-dev01 get nodes
kubectl --context kind-atlas-refactor-test-dev01 get applications -n argocd
kubectl --context kind-atlas-refactor-test-dev01 get pods -A
kubectl --context kind-atlas-refactor-test-dev01 get pvc -n workload-web

curl --noproxy '*' \
  --cacert /private/tmp/atlas-development-start-20260927/development-ca.crt \
  --resolve web.atlas.test:8443:127.0.0.1 \
  https://web.atlas.test:8443/
```

浏览器需自行配置 `web.atlas.test` 的本机解析及开发 CA 信任；本次没有修改系统 DNS/hosts 或信任库。
私有执行目录目前位于 `/private/tmp`，不要在保留该集群期间清理它；其中还有 Kind 节点的审计 bind mount。
工作仓库 `.state/` 中已有旧实验状态，因此没有把新目标塞进旧状态目录。

## 仍然保留的限制

这里证明入口、证书初次签发、PVC provision、完整 GitOps 图、52 项 Seed ownership 和重复 apply。
没有执行 Pod 重建/PVC rebind、未授权网络与 HTTPRoute 探针、证书轮换或故障注入；它们不在本次精确批准范围。
这次不是最终代码从空集群的一次无人值守成功，Docker 制品准备和 Argo 健康缓存仍需后续改善。

严格 `status` 目前还要求 CRD 最后成功 SSA 操作的 SHA 与当前 GitOps revision 一致。
审核分支推进后，即便 Seed 字节未变，也可能先报 HANDOFF_PENDING/ADOPTED_DEGRADED；
通过 Argo 对既有分支正常同步 `argocd-self` 后再检查，不能由 Bootstrap 重装 Seed。
后续可单独设计首次 adoption 证据与持续 desired revision 的更完整状态模型。

单节点、Retain 本地卷和开发 CA 不提供 HA、备份或恢复承诺。原 Atlas authority transfer 继续 NO_GO。
