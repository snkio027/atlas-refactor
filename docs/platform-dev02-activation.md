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
