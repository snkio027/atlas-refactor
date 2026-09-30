# S2 前操作与 Go 工程审查

2026-09-30。审查起点 `f4de94b0bedd70da476c65aa5490cf901353c507`；
修复在 `codex/pre-s2-quality` 中形成，随后按 PR 流程归入主线。范围是日常 CLI、验收逻辑、进程/文件边界和本地质量门。
本轮不启动 S2，不操作集群、凭据或 GitOps 分支，不更改 S1 的冻结运行结论。

## 判断

控制权模型目前有清晰的实现边界和针对性测试；主要新发现集中在外围验收与操作入口。
修复这些问题比继续扩展 OT-1 状态机更有价值。本轮没有增加配置/证据 schema、恢复接口、
controller 或新的第三方 Go 依赖，也没有调整 Desired Identity 等价规则或 mutation fences。

本地工程检查通过后，可将这组修复交给 S2 前代码审查。该结论不等于新版本通过真实集群验收，
也不等于已完成远端 CI、release 或 production Gate。

## 已修复的具体问题

| 问题与触发条件 | 后果 | 本轮处理与回归 |
| --- | --- | --- |
| `atlas-dev` defer 忽略 `latest-run.json` 写入错误，PASS 提前打印 | 磁盘/文件异常时仍退出 0；旧证据容易被误读为当前成功 | 保存成功后才输出 PASS；保存失败非零；正常失败写 FAIL 和原因；保留原始错误。`TestFinish*` |
| 私有 JSON 直接 `WriteFile` 到现有路径 | 可能跟随 symlink、保留过宽权限，写入中断可破坏旧文件 | owner-only regular file 检查、同目录临时文件、fsync/rename；失败保留既有文件。`TestPrivateJSONRejectsUnsafeTargetsAndPreservesExistingData` |
| 多副本中只有部分 web/proxy Pod 在正确节点 | dev 验收受遍历顺序影响；OT-1 Gate 只要求存在一个正确副本，均可能漏报错误放置 | 所有匹配副本逐个检查，仍要求 web 和真实 Envoy data plane 存在；dev 同时核对 namespace/labels。两侧多副本/顺序回归 |
| 重复 apply 仅比较 kubectl mutation 数量 | 一条旧日志被轮转掉、同时新增一条写入时，数量相等却并非零写入 | 比较 audit ID 集合；重复记录去重，缺失/空/损坏审计拒绝；UID 比较保留。`TestRepeatApplyDetectsSameCountDifferentRequests` |
| `--tool-dir` 相对路径随子进程 cwd 变化 | 进入私有 runtime checkout 后找不到原工具，或解析到错误目录 | 在执行前按调用者 cwd 转绝对路径。`TestRelativeToolDirectoryResolvedBeforeChildChangesDirectory` |

Pod 和工具路径回归已用 Go overlay 对照原始实现，均明确失败；修复后通过。
新增测试还覆盖未 Ready、缺失 gateway、错误 namespace/labels、证据失败与原错误并存、
symlink/文件权限、CLI 未知参数和多余参数。未用新增超时或宽松状态接受来让测试通过。

## 操作复杂度的收缩

| 操作 | 现在的入口与边界 |
| --- | --- |
| 本地完整质量门 | `task quality`；自动优先已存在的 `.state/tools`，其次 PATH，显式参数仍可覆盖；不下载工具、不访问集群 |
| 本地能力规划 | `task platform:plan` 或 `atlas-platform plan [flags]`；旧 `[flags] plan` 脚本仍可用 |
| 日常 runtime 检查 | `atlas-dev verify` 一次完整只读验收；`up` 在重复 apply/访问安装后仍进行最终复核 |
| 日常证据 | 同一个 `latest-run.json`，原子替换、明确 PASS/FAIL；保存失败时通过退出码识别，不能依赖旧文件 |
| S1 历史演练 | 冻结专用入口与不可变 authority bundle；不进入日常操作路径，不新增 continuation |
| 冻结入口的维护 | `task experiments:check` 编译/vet 三个带 build tag 的入口；已纳入 quality，不执行其 main |

工具路径只做选择，原有版本、制品与供应链校验仍执行。Task 默认值使用其标准
[动态变量机制](https://taskfile.dev/docs/guide#dynamic-variables)。
临时测试日志放 `/tmp`，经验进入测试与本文；没有新建另一套 `.state` 历史目录。

## 关键逻辑核对

| 不变量 | 检查对象与现有回归证据 | 判断 |
| --- | --- | --- |
| 正常 Bootstrap authority 单调减少 | `internal/atlas` latch/Receipt 路径；`TestFailureBeforeRootLeavesDurableLatch`、`TestDevelopmentInterruptionOnlyCompletesReceipt` | Root 前 latch 阻止重新获得 Seed 权限；中断续行只完成被允许的部分 |
| ADOPTED 与 runtime 分开 | `authority_test.go`；durable handoff 不查询 branch HEAD/leaf rollout，损坏 authority 仍拒绝 | 不把平台 Healthy 当作 authority 证明，也不由平台退化触发 Bootstrap 修复 |
| Desired Identity 只用于读取 | `internal/ot1/revision.go` 与 `revision_test.go` | 仅已发布的 plan 连续内容/spec 等价类；变化后恢复、未知/未来 SHA、UID 变化均拒绝；关键 owner exact-current |
| 操作完成与比较完成分开 | `publication.go`、`run_test.go`、`temporal_test.go` | 发布与受保护通知后等待比较；operation success 不替代新鲜 comparison proof；本轮未放宽 |
| 观察不能授权写入 | `internal/observation/reader.go`、`reader_test.go`、`proof_test.go` | 显式目标与凭据绑定，只读 API；不一致/未知不会成为 VERIFIED |
| Mutation 和 STOP 边界 | OT-1 exact plan/UID/RV/spec/Git fencing 与 audit/continuation 测试 | 本轮仅加强 runtime Gate，未新增 mutation、自动清锁或自动恢复 |

这是源码与回归测试的审查结果，不是形式化证明。已有 S1 live 证明仍只绑定
[冻结实现与原始运行](s1-final-validation.md)；不能把它转记到本轮新代码。

## 测试与覆盖范围

- 修改前：锁定 Go、离线模块、真实本地工具下 `go test -coverprofile=... ./...` 全部通过。
- 修改后：新定向回归、旧实现反例、冻结入口编译/vet、CLI coverage 已执行。
- 最终 `task quality` exit 0：gofmt、vet、Go race、真实 Helm/Kustomize、348 个 GitOps 资源校验、
  8 项 Python 测试、四个 ownership layout 和三个 tagged 历史入口均通过。
- `task build` exit 0：五个 CLI 使用 `CGO_ENABLED=0`、`-trimpath` 构建成功。
- 新旧 CLI 语法生成的 capability plan 字节一致；显式工具覆盖、默认本地工具选择已检查。
- `git diff --check` 通过。无 live cluster、网络部署、authority bundle 或 Trust Root 操作。

起点覆盖率：`internal/atlas` 80.0%、`developmentprofile` 92.9%、`observation` 69.5%、
`oci` 81.3%、`ot1` 69.3%、`platform` 61.3%。原 `atlas-dev` 与 `atlas-platform` CLI 均无包内测试。
新增针对性测试后，这两个 CLI 分别为 21.2% 与 8.9%；**未覆盖整个 CLI orchestration**。
覆盖率用于定位空白，不作为“逻辑正确”的替代指标；`cmd/atlas-artifacts` 仍无包内测试。

## 保留的工程债与 S2 约束

1. 仓库尚无 `.github` 自动质量工作流。本轮没有建立或确认远端 required checks，
   因此本地 PASS 不能替代合并/发布时的独立 CI。
2. `atlas-dev` 仍组合进程、文件、网络与流程；本轮只分离了 Pod 检查、审计和证据保存。
   后续新增行为应先给可失败边界补测试，不以大规模重写或覆盖率数字为目标。
3. `internal/ot1` 保留较多专用历史入口；继续冻结维护，不把它变成 S2 的恢复或部署引擎。
4. JSON object 与部分 runtime 判定存在跨包重复。暂不强行合并：严格解码、Kubernetes
   语义投影和 immutable evidence 的契约不同；S2 应复用 observation 的公共能力，
   新 Project/Workload/Binding 输入用明确类型和边界校验。
5. S2 的资源规划与本地编译可以从这条质量门进入；真实部署仍需具体目标与计划。
   不借“降低操作复杂度”取消权限、身份、Secret、供应链或恢复边界。
