# S2 r7 最终验收与收口边界

2026-10-09，**完整干净实例 Runtime PASS**。S2 停止功能扩展，PR #9 进入最终代码/架构审核。
同一 implementation、唯一派生计划、同一新 target、单次 attempt 完成 D1 → S2 → 无副作用重复验收。
本页仅为脱敏摘要；原始私有证据保留本地，未随代码上传。

## 不可变实测绑定

| 项目 | 值 |
| --- | --- |
| Implementation | `d011813ac94559e623f29d54efbde231cab261fc` |
| 总审核计划 SHA256 | `49a19382d346c2cf6f1df0efe4feaa6fcaaa2a7a21ba43ca78756734abe23cfe` |
| 唯一实际 S2 plan SHA256 | `07f2657da81c69f35bb48881c6e857f5f7140b269489bd1f46e3bbcccd4f1160` |
| S2 binary SHA256 | `fdfc4115528b684c030d106ea193836a3df4d3dcb6dd35f5424806965436a092` |
| D1 package | `v0.1.0-d1.4`，原包 checksum / provenance 核验通过 |
| Target / public deployment branch | `atlas-s2-r7` |
| Final deployment commit | `285b7e04a152ab17827d788d8c7fccee16dc1d44` |
| Final evidence manifest SHA256 | `75d94d543b38a8784ec4757b465d3f044c91e4e44d6fc365f1371fc20ff9ae0a` |

该候选的完整本地 `task quality` 与 [exact-head Quality CI](https://github.com/snkio027/atlas-refactor/actions/runs/37838999892)
均 PASS，外部执行发生于 CI SUCCESS 之后。70 项审核输入保持原字节；最终 manifest 绑定
645 份本地文件及两份独立备份文件摘要。此处不公开私钥/凭据的单项关联摘要、UID、私有路径或日志。

## 已通过的 Gate

| 验收 | 结果 |
| --- | --- |
| D1 clean install / independent verify | PASS；四节点，有限 Bootstrap → durable GitOps handoff |
| 编译与计划 | 唯一派生条件、consumer preflight / CRD default-stability、阶段双编译通过 |
| Permissions → Project → Infrastructure → Consumer | 四个 publication 与只读 Gate 全部 PASS |
| TLS 顺序 | 两张新 Certificate 当前 generation Ready 时 Gateway 尚无 S2 listeners；随后才发布引用它们的 listeners |
| 功能 | HTTPS → Web → S3 Put/Get/Delete；独立 Binding identity；跨 bucket / bucket 管理权限拒绝 |
| 隔离 | unbound 无 Binding、DNS 可解析而 S3 网络拒绝；跨项目 Secret 权限拒绝 |
| Metrics | 声明开启 metrics 的 workload 全部副本 up=1 |
| 兼容性 | 原 D1 Web/S3 通过；352 项旧持久资源 identity/owner、13 项冻结文件和 Bootstrap authority 不变 |
| 原始 final | Project / Workload / Binding / Runtime 全部 VERIFIED |
| 幂等 | 双 compile 字节一致；repeat publish 无新 commit；repeat deploy 仅观察、不轮换凭据、不重复 probe |

最终四节点 Ready，30 个 Application 纳入验收，378 个生成资源 identity。r6 节点按精确授权
删除；两次原 STOP、备份与公开历史保留。没有 r7 continuation、live patch、降低 Gate 或延长预算。

## 私有证据索引

以下路径相对于私有 r7 验收目录，仅列出用途，不附原始文件：

| 相对位置 | 证明内容 |
| --- | --- |
| `REVIEW.md`、`RUNBOOK.md`、`evidence/input-sha256.json` | 审核范围、唯一派生规则、固定输入 |
| `evidence/derived-s2-plan.json` | 实际 plan 与目标绑定 |
| `s2/state/authority/<planSHA>/` | 每阶段 intent/receipt/Gate，以及原始功能 `final.json` |
| `evidence/tls-before-listeners.json` | Certificate Ready 先于新增 listener 的辅助快照 |
| `evidence/execution-result.json` | 唯一执行 PASS |
| `evidence/repeat-verification.json` | 编译/发布/部署重复验证与证据不变 |
| `evidence/final-validation-summary.json` | 原始结果、Git、旧证据及审计范围交叉核验 |
| `evidence/final-evidence-manifest.json` 与 `.sha256` | 最终证据绑定；未替代仍追加的 live audit |

HTTPS/S3/网络与 provider 权限结果直接记录在原 final.functional。跨项目 Secret 拒绝及 metrics
全副本 up=1 是该成功源码绑定 Probe 的强制断言，目前没有额外独立序列化的 raw metrics / RBAC
报告。成功后的只读 `latest-observation.json` 仍标记 Runtime=UNPROVEN；它不替代原功能 final。

## 证据限制与审查修复

Metadata audit 仅覆盖 create/update/patch/delete/deletecollection，不覆盖 GET/streaming。
重复验收窗口的受覆盖管理写入为零；**没有 exec 审计事件不能证明没有执行 probe**。
受覆盖的零管理写入、功能执行 intent/result、源码中的跨调用防重放，是三种不同证据。

r7 的 `d011813` 已证明重复成功 deploy 只观察，但当时独立 probe 尚缺跨调用防重放 intent，
final 也先于本地 provider 收尾写入。最终静态审查发现这些可达错误路径，不表示 r7 实际发生过。
本次有限修复添加 exclusive `probe-started.json`，禁止 final/STOP/既有 intent 后重放；
provider 更新完成后才提交 final；PASS+STOP 明确拒绝；STOP 证据写入错误与原错误合并返回。
定向故障注入覆盖重复调用、并发争用、intent 写入失败、provider 失败、终态矛盾和证据保存失败；
正常前后观察重试仍只执行一次功能操作。

这次修复只验证增量源码与完整质量检查，不重建 r8、不更换 r7 binary/plan、不重复功能 probe。
**r7 Runtime PASS 永久绑定 d011813，不等于修复后提交已经实测。** 编译器、清单、凭据策略、
四阶段发布、功能探测内容保持冻结。ADR-0015/0016/0017 继续 Proposed，等待维护者审核。

本验收是有校验缓存的同机、单所有者本地开发实例；Trust Root 备份使用独立批准的同机开发例外。
不覆盖空缓存新机器、物理隔离备份/DR、生产就绪、HA、强多租户或原 Atlas cutover。
