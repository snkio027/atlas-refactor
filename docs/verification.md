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

## 第一轮本地交付时尚未验证

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
  准备阶段尚未创建该集群，也没有执行 Tier-0 写入；ADR 保持 Proposed。

## 首次真实执行与修正

第一次运行在镜像预载失败，未到达 GitOps 交接，详见
[失败记录](integration-20260927-01-result.md)。修正显式平台导入和 ConfigMap
data 字段后，含真实 Helm 渲染的 `task quality` 再次通过。新增回归检查覆盖
离线导入失败时不得安装 Seed、输入文件流、临时归档清理及锁定引用验证。
这些修正在第二轮通过了真实安装；后续结果见下文。

## 第二次真实执行与 ownership 修正

第二轮完成了 Root 创建后的 SIGINT、只提交 Receipt 的续跑、重复 apply 的零
API 写入，以及三类失败的拒绝和恢复，但发现 38 个普通 Seed 对象仍未被 Argo
接管，因此整轮 Gate 失败。详见[第二轮记录](integration-20260927-02-result.md)。
ownership 修正增加了持久清单逐对象检查与精确 Git revision 检查；新的单元
回归及锁定 Helm 渲染检查通过。对第二轮真实集群执行的只读 probe 也确认新
检查能拒绝“健康全绿但 Seed 未接管”的状态，API 审计为零写入。
第三基线的实测结果见下文。

模拟器验证 CLI 的状态和请求契约，不模拟 Kubernetes/Argo CD 的全部语义；
这些结果不得用于声明生产就绪或完整恢复闭环已成立。

## 第三次真实执行与资源作用域修正

第三轮在 Root 创建中断后保留了 latch，并完成 Argo 自同步；但 observer 对
集群级 tracking namespace 和 CRD 注解的假设不符合 Argo CD 3.5.1。续跑
超时退出 1、零 API 写入、未提交 Receipt，整轮 Gate 失败。等待交接状态下
的 artifact 缺失和配置漂移均拒绝写入且恢复；ADOPTED 重复执行和 unhealthy
测试因前置状态未满足而未执行。详见[第三轮记录](integration-20260927-03-result.md)。

修正保留全部 39 个持久资源的检查，使用 36 个 tracking/SSA 证据与 3 个 CRD
的 spec SSA、当前清单和成功同步记录。包含真实 Helm 的 task quality、Kustomize
和三平台构建通过；真实第三集群只读 probe 通过且审计零写入。此 probe 不执行
apply、不提交 Receipt，也不证明新的完整 Bootstrap 闭环。

## 第四次真实执行：最小 Bootstrap Gate 通过

固定基线 `0ef918801dac84ac913416f4c1756bf08045a485` 在独立集群
`atlas-refactor-test-vs0927d` 完成完整验收。Root 仅创建一次；Root 后中断保留
latch 且无 Receipt；续跑只创建 Receipt；四个 Application 在精确 SHA 上
Synced/Healthy，39 个持久 Seed 的接管证据全部通过。重复 apply 零写入，
控制对象与全部持久资源身份、内容和 tracking 稳定。

缺失 artifact、配置 drift、controller 暂停期间的 unhealthy 注入均令正常
Bootstrap 返回非零，审计零 Bootstrap 写入；全部恢复后 status --check 返回
ADOPTED/0、controller 为 1/1。四项显式故障注入/恢复 patch 单独计入审计。
首次启动曾因准备目录权限过宽在本地被拒，收紧权限后使用原基线重试；失败
记录保留。含锁定真实 Helm 的 task quality 再次通过。

见[第四轮结果及证据](integration-20260927-04-result.md)。本轮证明约定的可丢弃
环境最小正常 Bootstrap 行为闭环，不代表完整旧 Shell CLI 对等或生产权威切换。
ADR 仍为 Proposed；所有实验集群与证据保留，原 Atlas 不变。
