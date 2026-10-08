# ADR-0017：S2 发布契约与只读收敛 Gate

Status: Proposed。日期：2026-10-08。替代 ADR-0016 中 r4 的首次观察特判；保留四阶段发布。
本轮用户授权重新设计和本地验证，不授权续跑 r4 或新集群 mutation。

## 问题与工程依据

创建对象不等于 controller 已观察对象。r4 在创建后一秒读取到空 status 后 STOP；
fc8ce63 用 generation=1、空 status 和一组三个 reason 字符串识别初始化，仍绑定某一瞬间的
序列化形状。原观察还未把新资源 UID 从一个阶段传到下一个阶段，旧 spec 的正常应用过程
与未知 spec 漂移也混在一起。必须明确收敛协议，不能继续积累表示特判。

[Kubernetes API conventions](https://github.com/kubernetes/community/blob/main/contributors/devel/sig-architecture/api-conventions.md)
要求 level-based：spec 先变，status 异步跟随；不保证观察到所有中间状态。
[Argo health](https://argo-cd.readthedocs.io/en/stable/operator-manual/health/) 只聚合其定义的直接资源，
[Sync waves](https://argo-cd.readthedocs.io/en/stable/user-guide/sync-waves/) 是一次同步的顺序，不能代替跨应用发布前置条件。
[锁定 Argo 3.5.1 controller](https://github.com/argoproj/argo-cd/blob/v3.5.1/controller/appcontroller.go)
将 comparison 结果与 status 持久化分开，并用 comparedTo 判断 source/destination 是否变化。
工程上将不可变约束、暂未收敛和已就绪分离；用可重复的快照/时序测试验证，限制重试在只读观察内。

## 决策提案

1. 每阶段从 exact plan、完整 publication receipt 链、直接 Git parent、当前 compiler 输出
   建立不可变观察契约。区分本阶段新增、已存在且修改、保持不变的 Application。
   引入 Ready / Waiting / Rejected 三种 Gate 决策；S1 原始 facts 与 UNKNOWN 分类不改。
2. 先检查 identity、UID、完整 spec、删除边界、tracking/SSA、错误条件和 malformed status。
   漂移、未知 Git、不可用读取及明确失败立即 Rejected，整个快照 fatal 优先。
   新声明对象尚未出现、或在首次 comparison 前只具有受支持的初始化 status，可以 Waiting；
   没有条件/错误也不等于 Ready。显式未知状态不被全局吞掉；只有当前发布的新对象在严格
   有界初始化契约内可等待。已有对象丢失状态、新对象完成首次 comparison 后回退未知均拒绝。
3. 不要求经过 generation=1/空 status/OutOfSync 等固定序列；允许直接 Healthy。
   仅支持 Argo 3.5.1 的已知初始化字段，错误条件、终止/失败 operation、畸形字段不等待。
   发布中的既有 Application 可以暂处于 exact previous spec；第三种 spec 或观察到 target
   后回退 previous 均拒绝。source/destination 改变还必须等 `status.sync.comparedTo` 指向
   新目标；旧 comparison 不能证明新 spec 已收敛。阶段完成/STOP/final 后关闭这些过渡许可。
4. 一次只读等待保留首次观察到的 UID 和 comparison 事实；对象消失/换 UID 不能伪装成重建。
   已通过阶段的 UID 证据合并进入后续 Gate 和重复验收，冲突立即拒绝。
   standalone publish 同样保留前置 Gate，CLI observe --wait 与 deploy 共用会话循环。结束时仍做双读闭合，
   UID 改变立即拒绝，其他并发变化重新观察。无新恢复、resume 或 controller 接口。
5. 资源内容必须是 target，或当前阶段 exact predecessor 的已知旧内容；第三种内容是 drift。
   ownership 与 UID 优先核对。只有全部目标资源和完整 Application 事实通过才 Ready。
   Missing/Waiting 不开放 publish 或 probe。
6. 等待只调用读适配器；总 deadline 不延长、不因状态变化重置。取消、超时或 Rejected 即退出，
   进度只在决策变化时输出。publish/credentials/image/probe 都不进入重试闭包。
   新诊断字段仅为 S2 报告的可选 Gate decision，不改 S1 evidence 或计划 schema。

## 保持与验收

保持 permissions → project → infrastructure → consumer、exact-parent lease、独立凭据、
Bootstrap durable handoff、S1/D1 冻结输入和全部 compiler 输出。没有依赖升级、手动 refresh、
ignoreDifferences、放宽终态 proof、延长 timeout、自动回滚或 STOP 续跑。

用纯 evaluator 与只读 poller 测试合法时序的跳步/重复/乱序、r4 空状态、部分初始化、旧 spec、
错误优先、UID 替换、状态回退、receipt/gate 漂移、取消与超时；变形测试保证 Ready 不可由
缺失终态证明得到。再运行完整 task quality。历史 r4 永久 STOP；新 Runtime PASS 需要
新实现和精确执行计划，不能用本地测试或 r4 后续自然 Healthy 替代。


## r5 后的 Probe 读边界修正

r5 已完整通过四阶段发布 Gate，但功能 Probe 前的单次 Observe 遇到 closing proof 变化，
Pending 被直接作为失败返回。具体变化字段未保存，不能把它归因于 RV 或无害时间戳；
S1 semantic proof 保持不变。该历史 attempt 为 STOP，不能续跑或拼接成 Runtime PASS。

Probe 复用发布 Gate/CLI 的同一只读轮询与错误快照记录器。前后两次观察共享一个
receipt-bound contract 和单调 UID 会话；功能操作位于两个观察窗口之间，至多执行一次。
整个前观察 → 功能操作 → 后观察共用一个 15 分钟 deadline，且受父命令更早的 deadline
限制；不在后观察或 Pending 时重置预算。只有 Pending 的读可重试；功能结果即使是
Pending 也立即失败，不重试 POST、exec、SubjectAccessReview 或任一凭据操作。
终态仍需完整 closing proof、前后 UID 一致及功能事实；失败读保存当前 Rejected 报告。

没有新配置、计划/evidence schema、依赖、controller 或恢复接口。保持 compiler、四阶段
publication、D1 包和所有已冻结 runtime 证据。局部修正和本地回归不授权 r5 continuation，
新干净验收仍需新实现和精确计划。回归覆盖双侧暂态、双侧 fatal/UID/spec/SSA/Git drift、
功能 Pending/未知结果不重试、共享 deadline、取消和迟到成功。
