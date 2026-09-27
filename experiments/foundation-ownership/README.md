# OT-1 — Foundation Ownership Split Rehearsal

**NOT_READY / NOT_RUN**。本目录当前交付精确范围、阶段契约、纯本地校验与 Application
模式补丁 helper，不提供创建集群、执行迁移或捕获 live evidence 的命令。
这不是普通平台功能或通用 RolloutObservation API。

控制权、前置变更、父 App 隔离、STOP、两类 Gate 与证据要求见
[Proposed ADR-0009](../../docs/adr/0009-foundation-ownership-rehearsal.md)。ADR-0008 不变。

## 已固定的输入

- 旧布局：b618dea24b7c46cd36fd11a568a72c9a88f2097a。
- 新布局：b5d0562381f5b3989578d62e2677364d8c73710d。
- 新目标：atlas-refactor-test-ot1，四节点，与 dev02 / OT-0 隔离。
- 13 对象 3+4+6；完整对象列表、内容摘要、来源文件摘要见 [inventory.json](inventory.json)。
- [fixtures/](fixtures/) 是对应 Git 内容的副本，供本地 Kustomize 验证；没有任何 Application
  引用它们。实际阶段 Git 来源、完整平台渲染与 bootstrap snapshot 尚未准备，不能部署副本
  就声称完成 Atlas rehearsal。
- [stages.json](stages.json) 包括部分 2→1 回退、完整 1→3→1；由 contract.stages() 派生且被测试。
  `applications` 只列迁移 owner 集合，不是完整平台清单。

## 安全的本地检查

从仓库根目录执行（不连接 Kubernetes、Docker 或网络）：

```bash
PYTHONDONTWRITEBYTECODE=1 python3 experiments/foundation-ownership/contract.py
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s experiments/foundation-ownership -v
for layout in source secrets observability storage; do
  .state/tools/kubectl kustomize "experiments/foundation-ownership/fixtures/$layout" >/dev/null
done
```

contract.py 通过本地 git show 对照两个固定提交，拒绝内容、身份、domain 分配、额外类型
和 working tree 偏差，也核对未迁移的 secrets-controller payload 没有变化。输出只能是
OFFLINE_SCOPE_VERIFIED，不是 OT-1A/1B PASS。合成 observation 测试不进入 docs/evidence。

transition.mode_patch() 是纯函数：输入新读取的 Application 与已审查 expected manifest、
完整阶段 SHA/上阶段 info，输出 3 个 JSON Patch test 与单一 syncOptions replacement。
它不是集群客户端，也不是批准记录；未来 executor 仍须验证 target binding、父级移出、
唯一 window、source ref 没有变化、调用退出码，以及 patch 后的真实 observation。

## 启动仍被阻断

| 前提 | 当前状态 |
| --- | --- |
| 精确 13 对象等价与范围约束 | 离线已实现 |
| 部分/完整双向阶段契约 | 已定义，未运行 |
| Application 窄补丁与负例 | 离线已实现 |
| 同一个 Go engine 的 OT-1 独立 source/port/新快照绑定 | 未实现，当前 guards 正确拒绝新 profile |
| pre-PR#4 完整平台及各阶段 Git 投影 | 未准备 |
| 父级 detach/prune-confirm/reattach 的渲染与行为验证 | 待实现与演练 |
| 新 key 备份及三份新密文 | 同机独立 w1 例外已获用户允许；尚未生成 |
| 完整 observer、executor、create-only plan/approval/target binding | 未实现 |
| 精确实施 SHA/plan SHA 的运行批准 | 尚未请求，不能批准一个尚未完整的运行计划 |
| OT-1A / OT-1B | NOT_RUN |

当前 development profile 绑定共享 codex/development-platform、8080/8443，以及 dev02
首次实例化快照；不能修改这些输入或利用测试 hook 绕过保护。proposed-profile.json 与
proposed-kind.json 只让下一项前置审查的差异可见，并非可运行配置；普通 Go engine 应拒绝。

## 本次本地验证（2026-09-27）

- task quality：PASS，Go vet/race 与 348 个 GitOps 资源的真实 Helm/Kustomize、制品摘要、
  scope、AppProject、CRD structural schema 检查。仓库没有远端 CI Gate 证据。
- 本目录 6 个 unittest 方法：PASS，包含 20 个范围/guard 负例和双向窄补丁检查。
- source/secrets/observability/storage 四组 Kustomize：PASS。
- 从当前源码构建的 atlas doctor --config experiments/foundation-ownership/proposed-profile.json：
  预期 exit 2，报 development schema requires the reviewed development source and root。
  发生于 atlas.Load，早于构造 Runner；没有尝试创建集群或执行 Tier-0。
- 这些结果仅支撑本地准备，不支撑 OT-1A/1B、live ownership transfer 或部署就绪。
