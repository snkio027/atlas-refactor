# ADR-0005：四节点开发环境与自动验收

- Status: Proposed
- Date: 2026-09-27
- Scope: 独立 atlas-refactor 开发候选；原 Atlas authority 不变

用户已明确授权删除本机全部既有测试集群，重建一个四节点 Kind 集群，并要求减少人工干预。
原有 dev01 是单节点；默认 kubeconfig 未设置 current-context，使裸 kubectl 请求 localhost:8080
的 Web 入口并收到 404。旧记录按原 SHA 保留，不重写为四节点成功证据。

## 新开发 profile

schema 3 固定一个控制平面和 gateway、compute、data 三个 worker。入口映射仅在 gateway
上绑定 127.0.0.1:8080/8443；API 仍绑定 loopback。data 使用 NoSchedule taint，示例应用
显式容忍该 taint，本地卷 helper 也显式容忍；平台控制器在 compute。Cilium 在全部节点运行。
这是职责验证拓扑，不是高可用：仍只有一个控制平面、本地卷和单副本应用。

新的 Identity schema/substrate 和 Bootstrap contract 摘要与 dev01 分离；不伪造或更新旧
baseline.json。摘要绑定 Root、AppProject、控制图、Kind、两份 Seed、锁文件等启动输入，
普通 GitOps 叶子变更不重写 Signal。schema 2 的历史部署须使用原提交重现，当前清单不兼容。

首次 Receipt 仍要求当前 Git SHA、15 个 Application 的 Sync/Health、两个 Seed 的完整
接管证据，CRD 必须有该精确 SHA 的成功 SSA 同步结果。有效 Receipt 之后，所有 App 仍须
在当前期望 SHA Synced/Healthy，持久 Seed 仍须 ownership 与当前资源清单有效；CRD 允许
最近成功操作来自较早 SHA，因为无资源差异的提交不会触发新同步操作。该例外不能用于
首次接管，不能恢复 Seed 权限，不能接受未知 revision、缺失 tracking/SSA 或不健康对象。

## 工具分工

`atlas-artifacts prepare` 是独立在线制品准备：只操作锁定镜像的本机 Docker cache 和
私有 archive，不接触 Kubernetes。遇到 lazy Docker store 缺层时，用仅含锁定 FROM 的
构建物化原镜像内容，再导出原引用；不执行 RUN、不替换运行镜像或 digest。
OCI 检查逐 blob 验证 SHA-256，跟随锁定 index 验证 ARM64 manifest/config/layers 闭包。
schema 3 doctor 在创建节点前检查完整离线输入，apply 本身仍不下载任何依赖。

`atlas-dev up` 只编排现有 Go Bootstrap：持久私有 checkout、准备、doctor/render、apply、
验证、再次 apply。它不实现另一套 Tier-0 installer，不删除集群、不修复 Root，也不触发
Argo refresh/sync。精确目标与 Tier-0 批准参数仍必填。仅显式传入 install-kubeconfig 时，
备份并原子合并到用户默认 kubeconfig；Bootstrap 始终使用自己的私有、摘要绑定配置。

Argo 开发配置关闭 resource.ignoreResourceUpdatesEnabled，以保留状态更新触发的重新评估。
资源 status 仍不进入 Git desired state；不跳过 CRD 健康门禁。依据
[Argo 官方 reconcile 配置](https://argo-cd.readthedocs.io/en/stable/operator-manual/reconcile/)。
代价是在这个小型开发集群上增加调谐次数，实际效果必须由全新集群验证。

验收必须证明四节点 Ready、角色调度、现有完整 ownership Gate、PVC Bound/Retain、
HTTP 重定向、用运行时公有 CA 验证的 HTTPS 200，以及重复 apply 的 UID 稳定与审计零写入。
证据位于私有持久目录，公开报告只包含筛选后的状态、摘要和退出码；不包含凭据。

自动化失败时保存现场并非零退出，不能靠隐藏补丁或清除证据变成成功。集群删除仍是独立
授权行为。ADR 保持 Proposed；此次实验授权不等于生产替换或合并批准。
