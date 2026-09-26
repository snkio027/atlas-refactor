# 从 Atlas 采纳的经验

参考提交：`aca4ff137a1d254cfeceaec24526e0699b585e92`。
该提交是经验来源，不是新项目运行时的 GitOps source。

| 来源 | 采纳方式 | 当前状态 |
| --- | --- | --- |
| Operating Model 的四权分离与 External Root | 写入新架构；Root 不由父应用管理 | 已实现并有渲染测试 |
| GitOps 的 Seed/self 一致性与 Application 健康传播 | 使用同一渲染结果和同步/健康联合检查 | 已实现；真实接管待验证 |
| ADR-0002 的单调交接原则 | 用 handoff latch 提前关闭 Seed，再提交完成 Receipt | CLI 契约已测；服务端强制保护未实现 |
| ADR-0003 的恢复隔离与禁止自动跨权回滚 | 不把恢复塞进普通 apply | 边界已保留；恢复流程待独立设计 |
| ADR-0004 的身份长度失败 | 未来发证必须先验证精确字节边界和投影一致性 | 记录为未来恢复验收要求 |
| ADR-0005 的先发现、后授权循环问题 | 创建目标时明确绑定身份；不复用旧环境 Gate | 不复制 PERSONAL_LOCAL 流程；后续独立论证 |
| 锁定制品与离线准备 | 复制原始 chart 字节并校验；锁定工具/镜像/模板 | 本地运行前检查已实现；发布来源证明待建设 |
| 故障契约与 CI 分片 | 检查禁止副作用、错误分类与中断重试 | 本地测试已建立；远端 CI 待仓库建立后配置 |

原 Atlas ADR-0002 明确记录其完整 receipt-aware 正常路径尚未实现。新项目不能
把已接受文档、局部 canary 成功、或本地模拟器通过写成生产交接已验证。

新实现不复制历史阶段编号、整套 Shell 分派、未使用的 workload 树或未来 Operator
骨架。旧测试中有价值的行为被改写成 Go 测试；与 Shell 文件布局绑定的断言不沿用。

复制的素材来自同一参考提交：

- `vendor/charts/argo-cd-10.3.3.tgz`：原始第三方 chart，保留归属与随包许可。
- `assets/argocd-values.yaml`：来自原项目 argocd-self values。
- `assets/argocd-cm.yaml`：来自原项目 Application 健康规则；渲染时显式启用 annotation tracking。

后两项以及 chart 的字节摘要均记录在 `versions.lock.json`。
