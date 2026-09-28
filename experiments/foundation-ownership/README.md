# OT-1 — Foundation Ownership Split Rehearsal

当前：`dff5a20` 在干净新四节点集群通过普通入口单次 **29/29**，终态
REVERSE_VERIFIED / exit 0，S1 功能冻结并进入收口审核。
见 [最终验证](../../docs/s1-final-validation.md)、[证据索引](../../docs/s1-evidence-index.md)
和 [Failure Journal](../../docs/s1-failures.md)。历史 STOP 与首次记录保持历史含义。

- 旧布局：b618dea24b7c46cd36fd11a568a72c9a88f2097a。
- 新布局：b5d0562381f5b3989578d62e2677364d8c73710d。
- 唯一目标：atlas-refactor-test-ot1，四节点、18080/18443、codex/ot1-desired-state。
- [inventory.json](inventory.json) 固定 13 个对象及内容摘要；[stages.json](stages.json) 固定 29 个阶段。
- [profile-baseline.json](profile-baseline.json) 是新目标的独立首次实例化快照；dev02 的历史快照不变。

Go 运行入口、操作范围、plan/target/evidence 绑定和限制见 [S1 实现说明](../../docs/s1-observation-ownership.md)。
控制 authority、parent detach/reattach、两个 Gate 见 [Proposed ADR-0009](../../docs/adr/0009-foundation-ownership-rehearsal.md)。

## 本地验证

```sh
PATH="$PWD/.state/tools:$PATH" task quality
PATH="$PWD/.state/tools:$PATH" task build
```

quality 包含 Go vet/race、真实 Helm/Kustomize、348 资源的普通平台回归、独立 OT-1 profile、
24→26→24 完整图投影、合成 29 阶段链及 STOP/guard 负例；不会创建或删除集群。

原 Python contract/transition 只保留为固定历史输入与窄补丁的离线交叉检查，不承担 runtime
观察或迁移。新的运行路径只使用 Go 的统一 GVK model 与 Observation；没有第二套 Python observer。
合成测试不进入 docs/evidence，也不伪装成 OT-1A/OT-1B PASS。

## 首次真实执行状态（历史记录，非当前状态）

| 条件 | 状态 |
| --- | --- |
| 独立 source/port/snapshot，旧布局与阶段投影 | 本地实现与真实渲染验证 |
| 13-object 语义检查、父级重接、29 阶段与 create-only evidence | 本地实现与合成验证 |
| 有限 run、单次请求、批准 plan SHA、超时/STOP | 本地实现与合成验证 |
| 四节点 fresh Bootstrap 与完整旧平台 ADOPTED | 已验收；首次 core 中断记录保留 |
| 新私钥、w1/atlas-refactor-ot1 独立备份、新的三份密文 | 已生成、备份并按精确批准发布；同机开发例外 |
| 带实际 cluster UID/kubeconfig hash 的 transfer plan 与 7 个 Git SHA | 已编译并批准；后续阶段提交未发布 |
| 精确 setup/transfer plan 的真实操作批准 | 已取得；本次 attempt 在 BASELINE_ADOPTED STOP |
| OT-1A / OT-1B、partial rollback、full reverse | 尚未证明；迁移请求数为 0，保留失败证据与运行锁 |

`proposed-profile.json` / `proposed-kind.json` 保留为审查差异示例。应使用 `prepare-profile`
生成完整私有 checkout；把示例直接套在现有 dev02 输入上不能形成有效 OT-1 快照。
