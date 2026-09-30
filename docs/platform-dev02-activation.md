# dev02 平台能力启用记录

目标：现有 `atlas-refactor-test-dev02` 四节点 Kind 集群。
发布路径：已有 `codex/development-platform` → platform-control → 平台叶子。
原 Atlas、External Root、Bootstrap 身份、Seed 和历史验证记录保持原值。

## 所有者授权与开发例外

2026-09-27，所有者要求启用监测和 S3，指定 `w1` 作为新 Sealed Secrets
私钥备份目录。已实际执行并确认 `w1` 指向 `~/Workspace/01_Vault`，位于本机磁盘。
随后所有者明确批准：“允许 dev02 临时例外，备份到 w1 后继续启用”。

这是此可丢弃开发集群的临时备份位置例外。它不满足原 Atlas Architecture §5.3
的物理隔离要求，不改变原架构标准或生产要求，不代表恢复能力已经验证。
备份写入该目录下独立的 `atlas-refactor-dev02` 私有目录，目录 0700、文件 0600。
不得加入 Git、同步到公开位置或输出私钥；后续仍须转存离线介质并验证恢复。
原 Atlas Trust Root 不读取、不导出。

## 分阶段发布

1. 启用 capability-foundation、secrets-crds、secrets-controller。
2. 确认新控制器健康，备份新 Trust Root，提取公共证书。
3. 本地生成开发凭据；只提交严格 namespace/name 绑定的 SealedSecret 密文。
4. 启用 monitoring、object-storage、storage-monitoring 及依赖。
5. 验证 Argo、指标、看板、S3、网络边界和 Bootstrap 权限终止。

此记录区分部署授权、备份例外和实际运行结果；运行证据另行追加，不修改旧 evidence。

## 第一阶段结果

- 发布提交：`844153e7942ac0ae49d6ea4458c0e78895e16804`。
- 18 个 Application（原 15 + 3 个依赖）；新控制器 Synced/Healthy。
- AppProject 与能力清单异步更新曾触发 namespace 拒绝；权限由 GitOps 收敛后自动重试。
- CRD Established 后 Argo 应用健康暂未刷新；执行一次仅请求观察的
  `kubectl annotate application secrets-crds argocd.argoproj.io/refresh=normal` 后继续。
  未修改健康状态、放宽权限或修改 Root/Seed。首次运行不声称完全无人干预。
- 新 Trust Root 已备份到 w1 私有目录，文件 0600；重新读取校验一致，密钥与证书公钥匹配。
  只有公开证书进入封装流程；本地 receipt 记录 SHA256。物理隔离与恢复演练仍未完成。

所有者另行明确批准将本次 3 份 SealedSecret 密文提交到公开 snkio027/atlas-refactor，
并推送 codex/development-platform 触发部署；明文及私钥不上传。

## 第二阶段与运行结果

发布提交：`d1fbd2ddeda271109530f1ae3625d25bcdde61de`。
通过 Argo 子应用刷新清除旧 Git 缓存；monitoring-crds 在 Kubernetes 的 10 个 CRD
全部 Established 后请求了一次健康重新观察。没有改变健康判断或直接 apply 平台清单。
首次服务探测早于 storage-monitoring 最后一波，看板尚未出现而失败；依赖完成后重新验收通过。

- 24 个 Application 在该 SHA 上 idle / Synced / Healthy；3 个 SealedSecret Synced。
- Grafana 登录成功，26 个看板（含 Atlas），Prometheus 数据源健康。
- Prometheus 27 个目标全部 up、31 个规则组，kube_node_info 覆盖 4 节点。
- Alertmanager 本地测试告警 API 注入、查询与 resolved 通过；未发出外部通知。
- S3 签名 list/PUT/GET/HEAD/DELETE、预签名、分段上传通过；匿名和跨桶 403。
- 临时业务探测 Pod：带 s3-client 标签连通，未带标签被拒绝，Service DNS 正常；探测后已清理。
- 新增 Grafana 2Gi、Prometheus 8Gi、Alertmanager 1Gi、SeaweedFS 16Gi PVC 均 Bound。
- 原 Web HTTPS 与全部 37 个 Pod/4 节点验证通过，Go Bootstrap 状态 ADOPTED。
- 原身份、Root、self Application、Signal、Receipt 的 UID/spec/data 不变；重复 apply 退出 0，API 审计显示 kubectl 写入增量 0。

本轮未执行 Kubernetes 存储 Pod 重启持久性测试、全链路 Prometheus 告警触发、
外部通知、离线密钥恢复或生产切换。锁定 SeaweedFS 的独立容器重启测试属于此前本地证据，
不得混为本轮 Kubernetes 结果。

## 访问与凭据

[平台能力文档](platform-capabilities.md#网络和权限) 提供 Grafana、Prometheus、S3 的 loopback port-forward 命令。
Grafana 用户名 `admin`；密码在本机私有 `.state/capabilities/credentials.json` 的 `grafanaPassword` 字段。
S3 的 accessKey/secretKey 在同一私有文件；业务直接引用 workload-web/s3-client。
不把凭据复制进应用仓库。默认 bucket 为 uploads，region us-east-1，path-style。

Alertmanager 可另开终端：

```sh
kubectl --context kind-atlas-refactor-test-dev02 -n atlas-monitoring port-forward --address 127.0.0.1 svc/atlas-monitoring-alertmanager 9093:9093
```

脱敏证据：[platform-dev02-enabled-20260927.json](evidence/platform-dev02-enabled-20260927.json)。
该报告永久绑定实际部署提交 d1fbd2d；文档和证据后续提交不改写运行事实。
