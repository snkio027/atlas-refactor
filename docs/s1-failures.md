# S1 Failure Journal

失败知识保留在回归测试和本表中。这里的“零 mutation”指失败的 transfer/preflight
未执行 ceremony 写请求、未发布后续 Git 阶段；不否认先前真实 Bootstrap 和平台创建。
真实创建、凭据/Trust Root 的证据仍见 [首次现场记录](ot1-clean-rebuild-20260927.md)。
最终完整 29 阶段及独立 Gate-B 尚未通过，ADR-0009 保持 Proposed。

| ID | 症状与原因 | 外部影响 | 修复 | 回归 |
| --- | --- | --- | --- | --- |
| F1 | BASELINE_ADOPTED 错要求当前 ceremony operation；纯观察应允许最近成功 operation 早于已观察 Git SHA | 原始 plan `3b3d89475fac4d4d977aca05a0bb4a9598b24f8eb7a709ab369a65ac262adbfb` 在第 0 阶段 STOP；0 请求 intent，审计区间 64 事件、0 kubectl mutation，未发布后续 Git | `76f2cba`：operation proof iff compiled Step.Sync | `TestObservedStateDoesNotInventCeremonyOperation` |
| F2 | verified typed List 的成员可能省略 TypeMeta；逐项要求字段导致 NodeList 无法读取 | 只读诊断失败，0 mutation | `76f2cba`：在 discovery/scope/list header 验证后仅补缺失字段，显式冲突仍拒绝 | `TestTypedInventoryNormalization`、`TestTypedInventoryCannotBypassDiscoveryOrScope` |
| F3 | 有效 run 错继承准备阶段的 10 分钟 deadline，完整计划可能提前中断 | 本地审查发现；未在现场触发 | `76f2cba`：从原始 parent 与开始时间推导 29×300 秒 + 15 分钟；保留调用者取消 | `TestRunDeadlineCoversPlanAndPreservesCallerControl` |
| F4 | Argo CD 3.5.1 的 hook 比较条目可能省略普通 sync status；普通资源也可能带 hookPhase | 只读预检失败，0 mutation | `4ec6005`：仅接受 hook=true 且 status 缺失；有 HookType 的实际 hook outcome 独立分类 | `TestHookComparisonOmissionDoesNotHideFailure`、`TestHookOutcomeRequiresRecognizedType`、`TestRunningHookRemainsNonReadyDuringConvergence` |
| F5 | API 编码省略空 LabelSelector.matchLabels map，Git/live 基线误报内容漂移 | 只读预检失败，0 mutation | `4ec6005`：仅在 NetworkPolicy 已知 selector 路径的首次基线比较中规范化空 map；selector 本身及后续 live semantic 不变 | `TestBaselineSelectorWireOmissionIsNarrow`、`TestNetworkPolicyEmptySelectorsKeepPeerAndPlacement` |
| F6 | Gate-B 包装器拒绝正常 Seed inventory 的 stdin manifest GET，普通 CLI ADOPTED 而包装后 UNAVAILABLE | plan `c26950f5c7d6920fb1f6571db504e2bcfbe32bbdf98557480307578ecdf76452` 第 0 阶段 STOP；2026-09-27 12:25:43–12:26:12 UTC 审计 78 事件、0 kubectl mutation，0 请求 intent，未发布后续 Git | 本次修订：仅放行精确 `get -f - --ignore-not-found=true --show-managed-fields -o json`；Gate 错误携带实际 state/detail | `TestReadOnlySeedInventoryWithStdin`；修复后同集群只读 Status=ADOPTED，47 请求、0 denied |

F1–F5 定向回归与完整 `task quality` 已通过。F6 定向回归及同集群只读复核已通过；
完整质量检查和新的真实 ceremony 结果单独更新，不把预检成功等同于运行成功。

## 保留规则

- 本地开发和只读预检使用 `.state/latest/ot1/`，失败明确且进入回归后可替换其临时输出。
- mutation 前 STOP 经请求 intent、Git source 与审计确认后，只保留原因、影响、plan SHA、
  fix 与测试。旧 plan、binary、快照、日志和已核对的 stale lock 可清理。
- 任何重要 live mutation、后续 Git 阶段发布、凭据/Trust Root 改动、影响不明或未解决事故，
  都冻结完整 authority evidence；不能以“开发模式”自动续跑、回滚或清理。
- 最终正式 Gate PASS 保存完整不可变 bundle。中间报告不再成为长期独立文档。
