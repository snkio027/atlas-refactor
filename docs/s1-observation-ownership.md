# S1：Observation、Evidence 与 Ownership Rehearsal

本次是同一 PR #6 的完整工程 Slice。首次干净重建和完整平台验收已通过；OT-1 在基线阶段 STOP，尚未发生所有权转移。见 [现场结果](ot1-clean-rebuild-20260927.md)；ADR-0009 保持 Proposed。
现有 Bootstrap integration 基线、OT-0 成功/失败证据与 dev02 desired 分支没有改写。

## 实现与目的

| 实现 | 为什么需要 |
| --- | --- |
| internal/observation | 通用观察与 OT-1 共用 identity、revision、scope、operation 和 semantic 解释 |
| atlas-platform observe / verify | 显式 expectation + SHA 的只读检查；旧 commit 的 Healthy 不能冒充本次成功 |
| EvidenceEnvelope 与私有原始 GET | 保留事实、分类和来源，Markdown/终端只是投影；不能用一句 PASS 替代证据 |
| 首尾 UID/RV、inventory、cluster/kubeconfig fence | 检出不可用读取和采集期间的并发变化；不声称 Kubernetes 多对象读取具有原子性 |
| internal/developmentprofile | 在同一 engine 中隔离 OT-1 的 source、端口、substrate identity 和冻结快照 |
| atlas-ot1 prepare-profile / plan | 从固定历史输入产生可复核旧布局、完整图和 7 个不可变阶段提交，避免手工拼装环境 |
| 29 阶段 ownership checker | 固定部分回退与完整 1→3→1，分别验证预期 strict 拒绝、窗口接管和 strict 恢复 |
| guarded create / patch / orphan delete | Application 模式与生命周期是有限 mutation 面；13 对象内容/tracking 只由 Argo 写入 |
| run 与 create-only attempt | 同一批准计划内正常续行；每个请求先写 intent，失败停止、不能重试覆盖历史 |
| 独立 OT-1B | 正常 engine ADOPTED、重复 apply、runner deny、audit、nodes/Pod/PVC/PV/HTTPS，避免把 OT-1A 当作完整平台证明 |

Observation 为 `VERIFIED / PROGRESSING / DEGRADED / DRIFTED / UNKNOWN / ABSENT`。
exit 0 只表示本次 expectation 验证成功；普通未就绪/退化/缺失为 1，未知/漂移/输入错误为 2。
它不授权修复，不修改 refresh annotation，不写 catalog/enabled，不成为 daemon。

## 状态证据与操作证据

校验器依据编译后的 `Steps(plan)[index].Sync` 判断是否需要本阶段 operation proof。
未提交 sync 的 BASELINE_ADOPTED 验证当前完整 spec、精确 observed revision、Synced/Healthy、
idle、identity、tracking 与 inventory；最近成功 operation 可以属于更早的无内容差异提交，
无需虚构 ceremony marker。实际提交 sync 的阶段仍要求精确 ot1-stage、operation revision、
outcome 和 syncOptions；同样的旧 operation 在 STRICT_RESTORED 必须失败。

共享 APIReader 先验证 discovery、锁定 GVK/scope、typed list header、resourceVersion 和无分页，
然后仅补全成员中缺失的 apiVersion/kind。显式空值、null、类型错误和冲突值仍拒绝；后续
namespace/name、scope 与重复 identity 校验不变。规范化不能隐藏语义分歧。

在随后单独授权的兼容修订中，hook 比较仅允许 `hook=true` 且同步 `status` 字段缺失；
显式空/null/Unknown/OutOfSync 和不健康资源仍阻塞。Application 的精确 revision、健康、idle、
condition、完整 spec 检查保留；operationState 中已识别 hook 的执行结果另行分类：运行中等待、
失败停止、未知类型/结果失败关闭。普通资源的 hookPhase 字段不会被误当 lifecycle hook。
依据是 [Argo CD 3.5.1 的比较实现](https://github.com/argoproj/argo-cd/blob/v3.5.1/controller/state.go#L894-L899)。

基线 Git/live 内容比较仅规范化 networking.k8s.io/v1 NetworkPolicy 已知 selector 路径内的
空 matchLabels map。依据 [锁定 Kubernetes LabelSelector 编码](https://github.com/kubernetes/apimachinery/blob/v0.36.1/pkg/apis/meta/v1/types.go#L1209-L1222)，
移除空 map 后保留 selector 本身；缺失/null selector、非空 label、expression、端口与策略差异
仍可见。规范化只用于首次基线内容对比；原始 GET、冻结 scope hash 和后续 UID/semantic
不变性检查保持原规则。

## 入口

在仓库根目录使用锁定工具构建。`observe/capture/run` 要求 `go build` 的 clean VCS-stamped
binary；不会接受没有实现 SHA 的 `go run` 或 dirty build 充当现场证据。

```sh
PATH="$PWD/.state/tools:$PATH" task quality
PATH="$PWD/.state/tools:$PATH" task build

# 纯本地：新目录必须不存在；不 fetch/push、不生成凭据、不连接集群。
bin/atlas-ot1 prepare-profile --root "$PWD" \
  --output "$PWD/.state/ot1-candidate/repo" \
  --helm "$PWD/.state/tools/helm" --kubectl "$PWD/.state/tools/kubectl" \
  --yq "$PWD/.state/tools/yq"
```

`prepare-profile` 清空旧 SealedSecret 密文，生成新的 profiles/ot1.json、core 渲染与本地提交。
它不是完整平台启动：仍需按具体 setup plan 发布独立 desired branch、用普通 `atlas-dev up`
创建新目标、启用独立 Sealed Secrets、备份新 key、准备三份新密文并启用完整旧 catalog。
这些操作与密文公开发布须有精确计划。首次执行已使用独立新 trust root，未复用旧私钥；实际结果与 STOP 见 [现场记录](ot1-clean-rebuild-20260927.md)。

完整旧平台准备好后，target-binding JSON 需要 `cluster/context/clusterUID/kubeconfigSHA256`。
`plan --root ... --repo <已提交的完整旧布局> --output-repo <新私有目录>
--target-binding <0600 文件> --output <新 plan.json>` 生成规范 JSON；其文件字节 SHA256 就是
批准摘要。plan 含实现 SHA/binary hash、目标、锁定工具及 supply-chain 输入摘要、13 对象、
29 阶段、7 个 Git SHA、所有 render 文件 hash、每阶段完整 Application spec 与 300 秒阶段上限。
plan 不包含凭据明文或 kubeconfig 内容。

`inspect-plan --root ... --repo <plan clone> --plan <plan.json>` 输出完整阶段请求表，仍无副作用。
`prepare-action` 可生成精确 create、mode、sync、release 请求供审查；输出不等于执行批准。
DELETE 使用 Kubernetes DeleteOptions 的 UID/RV preconditions + Orphan，普通 `kubectl delete`
不具备同等 RV 保护。运行适配使用锁定 kubectl 的 `delete --raw=... -f -` 发送该正文。

只读入口：

```text
atlas-platform observe|verify --root ... --expectation ... --revision <40-char SHA>
  --kubeconfig <private> --kubectl <locked> --id <unique>

atlas-ot1 capture --root ... --repo <plan clone> --plan ... --stage <exact stage>
  --kubeconfig <private> --baseline-snapshot <baseline, after stage 0> --output <new private file>

atlas-ot1 verify-attempt --root ... --repo <plan clone> --plan ...
  --input <private captured bundle> --output <new attempt directory>
```

`capture` 返回普通 observation 分类；strict refusal 时非零不意味着迁移状态机失败。
`verify-attempt` 重新解释完整 raw facts、expected operation 和独立 Gate-B 文件，不能跳阶段。
自填 `GateProof` 布尔值不生效；status/repeat/audit/runtime 文件均严格解析并绑定 snapshot。
这些本地文件不是签名或防管理员篡改的证据，运行结论依赖被审查的 collector 与完整私有记录。

## 有限 run

现场批准的是完整 plan SHA，不是每一个正常命令。唯一运行入口：

```text
atlas-ot1 run --root <implementation> --repo <seven-revision clone> --plan <plan.json>
  --runtime-repo <actually created OT-1; retains its .state> --tool-dir <locked tools>
  --output <new private attempt directory> --approve-plan <exact plan file SHA256>
```

这是带副作用的模板，不能把它作为本地常规检查。运行前必须已批准这份精确 plan，且目标
已完成独立的 setup。实现 binary 与 plan 必须完全一致；runtime checkout 必须 clean、位于
baseline SHA，并持有创建该集群时的专用 kubeconfig、offline archives 与 audit 目录。
不允许复制 dev02 state 来满足条件。Git 发布使用已登录的 GitHub CLI credential helper，
凭据只经 helper 的管道交给 Git，不写入 plan、参数值或证据。运行适配显式禁用 Git hooks，
不执行 repository-local hook；这里使用宿主已有 Git/gh，不下载或安装它们。

执行器在同一 `runtime-repo/.state/ot1-run.lock` 下串行运行。所有 Kubernetes 请求显式绑定
kubeconfig/context；Source 分支只允许计划内 fast-forward。每步前重新观察前驱、目标、
对象与来源，每次 API mutation 检查 13 对象及审计；patch/delete 使用服务器原子前置条件。
父级先移出 foundation Apps，比较新 SHA 后才 release；全部 strict 恢复后按 Git 重接 parent。

总 deadline 从命令准备开始计时，按 `阶段数 × MaxStageSeconds + 15 分钟` 推导；
当前为 `29 × 300 秒 + 15 分钟 = 160 分钟`。计划验证阶段仍限 10 分钟，运行阶段不继承
该较短 timer；调用者的取消和更短 deadline 仍有效。不会新增可调超时配置。

一个阶段最多 300 秒；先以单次 Application list 检查已提交操作的可解释收敛，待就绪后
才采集完整多对象证据。新建 owner 的 UID 绑定 create/sync 返回的身份；完整快照仍要求
首尾 UID/RV 与 inventory 一致。Argo `omitempty` 省略的 `prune=false` 可被识别，但每次
operation 的 source/manifests/resources 等覆盖参数会被拒绝。等待只读，不重试 mutation。任何未知、
意外条件、scope/spec/UID drift、额外对象、超时、中断或证据写失败都会停止。失败时 run lock、
intent、响应摘要与现场先保留；不会自动关闭窗口、恢复 controller、rollback、删除集群或续跑。
窗口可能仍开放；续行必须先审查现场与新计划，不能简单删锁后重放同一 attempt。

## 本地验证与现场 Gate

`task quality` 强制真实 Helm/Kustomize fixture 测试、普通 348-resource 平台检查、Go vet/race、
固定 Python contract 交叉检查；`task build` 产出 atlas-platform 与 atlas-ot1。
合成测试验证 29 阶段、预期失败、部分/完整回退、父级图恢复、错误计划、不同目标、额外对象、
UID/semantic/权限漂移、陈旧 operation、UNKNOWN、只读 HTTP、写请求拒绝、audit 丢失/旁路写入、
STOP 后零续行和历史不可覆盖。它们不读取现存集群，不生成可解密凭据，也不是 runtime proof。

真实 Gate 仍包括：fresh OT-1、完整旧平台 ADOPTED、parent detach/reattach 的 Argo 行为、
13 UID/semantic/tracking/SSA、独立 OT-1B、partial rollback、full reverse、最终现场审查。
第一次执行可能暴露 API 默认值、controller 状态时序或 parent prune 行为与计划不同；届时
保留 STOP，按真实证据修正，不能放宽 classifier 来制造 PASS。

S2 Project/Workload/Binding 与 S3 scopegen/runtime/CI 只更新路线图，未混入本次实现。

锁定的 [Argo CD 3.5.1 server](https://github.com/argoproj/argo-cd/blob/v3.5.1/server/server.go#L276-L300)
会初始化 `default` AppProject。因此完整项目 inventory 固定为 Atlas 的 atlas-bootstrap /
platform-project / workload-project，加这个 upstream 内建项目。校验器对内建 spec 使用
该版本的精确投影，之后绑定 UID/完整 semantic；不会删除它、放宽它或把 Atlas App 指向它。

## 开发证据与 authority evidence

日常开发只验证最新 implementation、plan、cluster 与 verification，固定使用
`.state/latest/ot1/`；失败沉淀为 [Failure Journal](s1-failures.md) 与回归测试。
本地编译、只读 capture/preflight、质量日志可在确认不再被运行引用后替换。
CLI 的 create-only 检查仍保留；清理由调用者在重新生成前显式执行，不添加 force/overwrite。

| 情况 | 保留与续行 |
| --- | --- |
| 本地/只读失败 | 确定原因、补测试、更新 journal；替换临时开发输出 |
| ceremony 在外部 mutation 前 STOP | 先核对 request intent、Git source、审计和当前目标；明确零影响后可清理旧 binary/plan/快照/锁，在既有授权范围内用最新绑定重验 |
| 已发布后续 Git、已写 live resource、数据或 Trust Root，或影响不明 | 冻结完整 plan/binary/intent/响应/前后事实；停止并审查恢复决定 |
| 正式 Gate PASS | 保存完整不可变最终 bundle |

run lock 阻止并发与未经审查的重入，terminal 记录运行终态。两者用途分开。
失败时执行器仍保留锁并停止，不自行判断可恢复；经上述零影响核对及用户授权后，
调用者可显式移除摘要匹配的 stale lock。重要 mutation 后 STOP 的锁和证据继续保留，
直到恢复/完成决定；禁止仅删锁后重放。此规则不改变任何 Kubernetes/Git 写权限、
13 对象作用域、failure classifier 或批准过的目标。

当前完整 ceremony 尚未通过；最新零写入失败及修复见 Failure Journal。真实 Bootstrap 与平台创建证据继续保留，最终结果仅在完整 Gate 后记录。
