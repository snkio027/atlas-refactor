# 本地验证记录

日期：2026-09-27。环境：macOS ARM64，Go 1.27.1。

## 已通过

- `task quality`：格式检查、`go vet ./...`、`go test -race ./...`。
- 严格 JSON 解析器短时模糊测试：执行 208,421 次，无失败。
- 具名测试函数及其子案例：完整模拟交接、副作用顺序、重复执行、Root
  创建中断、Receipt 失败重试、健康退化、UID/配置漂移、不可用读取、严格配置、
  路径和凭据边界、锁及批准参数。模糊测试种子也在常规 Go 测试中执行。
- 真实 Helm **4.2.3** 渲染：两次输出一致；Seed 与 GitOps 使用同一清单；
  镜像固定，生成结果没有 Secret 对象。
- 使用本次构建的 CLI 执行离线 `render`，再用 kubectl **1.36.3** 的本地
  `kustomize` 检查所有生成目录：Root 2 个资源、Projects 1 个、Platform
  Applications 1 个、Argo CD 44 个。此检查没有连接 Kubernetes API。
- `task build:matrix` 和 `task build`：darwin/arm64、linux/amd64、linux/arm64
  构建成功；本机 CLI 帮助可运行。Linux 二进制尚未在对应系统执行。

真实 Helm 验证通过 `ATLAS_TEST_HELM` 指定本地预装的锁定可执行文件。
编译和测试期间使用 `GOTOOLCHAIN=local`、`GOPROXY=off`、`GOSUMDB=off`，
没有获取 Go 工具链或模块。

## 尚未验证

- 真实 Docker/Kind 集群创建、镜像预载、Argo CD 安装和 GitOps 接管。
- Argo CD 从集群内读取远端 Git、精确 revision 和匿名运行时认证。
- 实际服务端默认字段、Admission、网络和 controller 行为下的故障恢复。
- 跨主机互斥、管理员删除证据的保护、独立恢复路径和生产环境运行。
- 发布制品签名、SBOM、来源证明、工具分发摘要和复现构建。

第一轮按所有者选择只完成本地实现与测试，没有执行真实 `apply`。

## 首次集成准备（同日追加）

- 按所有者后续授权创建公开仓库 `snkio027/atlas-refactor`。
- 使用锁定工具对 `profiles/integration.json` 执行只读 `doctor`：通过；
  三个固定 digest 的镜像均已存在于 owner-local OrbStack。
- 加入 Metadata-only API 审计及 `status --check` 后，再次通过含真实 Helm
  校验的 `task quality`；四层 Kustomize 本地渲染与 Secret 边界检查通过。
- Kind 0.32.0 的 darwin/arm64 二进制从官方 GitHub release 获取，并与
  release asset digest 及随附校验文件核对 SHA-256：
  `dca67911095a110c2b5c36e26df6cac860c602033e456c0db47be498cdef1ebb`。
- 测试目标及完整验收步骤见 [首次集成 Gate](integration-first-slice.md)。
  尚未创建该集群，也没有执行 Tier-0 写入；ADR 保持 Proposed。

模拟器验证 CLI 的状态和请求契约，不模拟 Kubernetes/Argo CD 的全部语义；
这些结果不得用于声明生产就绪或完整恢复闭环已成立。
