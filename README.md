# atlas-refactor

从 Atlas 的架构和失败经验出发，用 Go 独立实现 Bootstrap。

当前版本包含本地可构建的 `doctor`、`render`、`status`、`apply`，以及完整流程的
模拟契约测试。公开远端为 `snkio027/atlas-refactor`；实际集群创建和 GitOps 交接尚未执行。
第一阶段只支持独立的、可丢弃的 OrbStack / 单节点 Kind 测试集群。

## 构建与检查

需要 Go **1.27.1**。运行时实现只使用标准库，没有第三方 Go 模块。

```sh
task quality
task build
./bin/atlas --help
```

`task quality` 检查格式、运行 `go vet` 和 race-enabled 契约测试；不访问集群。
Go 自动工具链获取与模块网络访问被关闭。若本机没有 Task，也可运行：

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -race ./...
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go vet ./...
GOTOOLCHAIN=local CGO_ENABLED=0 go build -trimpath -o bin/atlas ./cmd/atlas
```

真实 Helm 渲染校验使用预装的 **4.2.3** 可执行文件：

```sh
ATLAS_TEST_HELM=/absolute/path/to/helm task quality
```

该校验渲染两次并比较结果，同时检查固定镜像、CRD、Secret 边界及 Seed/接管一致性。
未设置路径时，该项明确标记为跳过。`task build:matrix` 交叉构建三种目标；
交叉构建不等于已支持或已验证 Linux 运行环境。

## 首次集成验证

固定测试配置见 `profiles/integration.json`，完整目标、故障注入和证据要求见
[首次集成 Gate](docs/integration-first-slice.md)。`status --check` 是严格退出码的显式写法。
新建测试节点启用 Metadata-only API 写入审计，日志位于私有 `.state/audit/`；
记录中不包含请求/响应正文。ADR 保持 Proposed。

## 配置与渲染

```sh
cp config.example.json config.local.json
```

编辑 `repositoryURL`，填入 Argo CD 可匿名读取的 HTTPS Git URL。
本阶段没有私有仓库凭据配置功能。`config.local.json` 被 Git 忽略。

```sh
./bin/atlas doctor --config config.local.json --tool-dir /absolute/path/to/locked-tools
./bin/atlas render --config config.local.json --tool-dir /absolute/path/to/locked-tools
./bin/atlas render --config config.local.json --tool-dir /absolute/path/to/locked-tools --output gitops/test
```

`--tool-dir` 可指向预装的 Helm、Kind、kubectl 所在目录；未指定时从 PATH 查找。
工具版本必须与 `versions.lock.json` 一致。Docker 与 Git 从 PATH 查找。
编译和运行不自动安装工具，也不拉取镜像。

`doctor` 检查制品摘要、工具版本、OrbStack 上下文绑定和本地镜像存在性。
它不证明远端 Git、真实集群或接管已准备好。
`render` 默认只写 `.state/rendered/`；指定 `gitops/test` 时输出可提交的 GitOps 树。
配置、资源和镜像改变后必须重新渲染、审查和提交。

## 后续真实闭环

以下是操作说明，本地测试不会执行它们。

1. 准备锁定的工具和三种镜像；核对 `doctor`。
2. 渲染 `gitops/test`，审查并提交新仓库，将提交发布到配置中的远端分支。
3. 单独确认集群名、Docker 上下文、Git 提交和 Tier-0 目标。
4. 使用两个显式批准参数执行：

```sh
./bin/atlas apply --config config.local.json \
  --tool-dir /absolute/path/to/locked-tools \
  --approve-cluster atlas-refactor-test --approve-tier0
./bin/atlas status --config config.local.json --tool-dir /absolute/path/to/locked-tools
```

`apply` 要求干净的 Git 工作树、已提交且与当前渲染一致的产物，以及远端 revision
恰好指向本地 HEAD。首次安装创建独立 Kind 集群并绑定本地私有 kubeconfig；
随后预载镜像、安装 Seed、创建最小 AppProject 与 External Root，等待 GitOps
接管并创建一次性 Receipt。它不会删除或替换集群。

默认集群名为 `atlas-refactor-test`，也可使用该名字加短后缀。旧 Atlas 名称被拒绝。
继承的 `DOCKER_*` / `KIND_*` 环境变量被拒绝，避免命令目标被环境覆盖。

## 状态、重试与限制

`status` 输出 JSON：成功接管返回 `0`；缺失、接管中或退化返回 `1`；
漂移、不可读或不确定状态返回 `2`。其他命令的失败返回非零。

`HANDOFF_PENDING` 开始于 Root 创建前写入的不可变 latch。此后只允许观察和
一次性 Receipt 提交，不再安装 Seed、修复 Root 或修改 Bootstrap AppProject。
Root 创建结果不确定或交接后 Root 丢失会拒绝正常重试。Receipt 存在时的健康
退化也不会恢复 Bootstrap 权限。

`.state/apply.lock` 防止同一工作树并发执行。进程被强制终止后可能保留该锁；
必须先确认旧进程结束并检查集群状态，再由所有者处置。不要盲删 `.state/`：
其 kubeconfig 与摘要是目标绑定证据，丢失后本版本不会重新绑定已有集群。

这个版本不实现跨主机锁、独立恢复命令或对管理员删除证据的 Admission 防护。
`immutable` 防止内容修改，不能防止删除重建。因此它是可丢弃环境中的验证实现，
不能作为生产恢复或安全隔离保证。真实集群测试、私有 Git 支持、发布二进制的
签名/SBOM/来源证明，以及工具分发摘要验证仍待后续工作。

## 文档

- [架构与生命周期](docs/architecture.md)
- [新项目 ADR-0001](docs/adr/0001-go-bootstrap-foundation.md)
- [Atlas 经验采纳清单](docs/atlas-lessons.md)
- [测试与验证记录](docs/verification.md)
