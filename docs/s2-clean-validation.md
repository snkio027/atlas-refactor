# S2 clean validation — atlas-s2-r1

2026-10-03。结果：**D1 PASS；S2 consumer Gate STOP；Runtime UNPROVEN**。
这份记录对应一次新实例验证，不能与旧实例的部分结果拼接为完整验收。
PR #9 保持 Draft；本地修复不构成再次执行批准。

## 执行绑定

| 项目 | 本次实际值 |
| --- | --- |
| Implementation | `f432cfb693cd9047c99e3406252a5fcc7226311e` |
| S2 executable SHA256 | `98e83b60e325800b9ff186f721b173581495d25995c7dd0f317250cb3553714e` |
| Review plan SHA256 | `2a7e8f12a662b8e740b2370f340f8f037651e40897aaf9e87e057dc955a47c0b` |
| Derived S2 plan SHA256 | `155003503498e2466fc24bbd547dde17a64a7968284c7728be28976b0869e3ac` |
| D1 package | `v0.1.0-d1.4`，archive checksum 与 GitHub attestation 已核验 |
| D1 plan SHA256 | `1fd5bda54cf1682617868a8df2a8f4a1200b2150e419de6cc493d3daa8c44094` |
| Installation ID | `429e00971e9cd2037e541e4d16a8efdb` |
| Cluster / deployment branch | `atlas-s2-r1` / `atlas-s2-r1` |
| Cluster UID | `218cfdc7-22d6-41ce-be53-153879729b5c` |
| Topology / loopback ports | 1 control-plane + gateway / compute / data；18080 / 18443 |
| D1 FullCommit | `9a94140c1aeecab67d144094a62f8f7775a2f2b3` |
| Infrastructure commit | `a6905a6c3b95a569569df57d4ab02b99b7b4101d` |
| Consumer commit | `b8b72de31758995895b402014647f152eeb59e13` |

冻结实现的 [Quality run 36855041208](https://github.com/snkio027/atlas-refactor/actions/runs/36855041208)
在首次外部写入前已经 success。新实例没有复用旧 kubeconfig、Trust Root、凭据、密文或运行记录。
新 Trust Root 完成独立备份与读回验证；隔离级别是这次单独批准的
`same-host-development-exception`，不是物理隔离备份或灾难恢复证明。

## 实际结果

| 阶段 | 结果 |
| --- | --- |
| 一次 D1 install | PASS / exit 0，约 15 分 24 秒；有限 Bootstrap handoff，26 个 Application 收敛 |
| 一次 D1 read-only verify | PASS / exit 0 |
| D1 基线 | 352 个持久资源身份、tracking/SSA owner，13 份冻结文件和 Bootstrap authority 记录 |
| 隔离 SeaweedFS 4.47 fixture | PASS；对象读写、跨桶与管理权限负例，清理该临时容器 |
| 唯一 S2 plan 派生 | 全部固定相等条件、新 UID / certificate / D1 parent 关系通过 |
| Infrastructure compile ×2 | 字节一致，实际文件摘要重算相符，363 个唯一 owner 资源 |
| 四节点镜像 / 独立 Binding 凭据 | 完成；没有替换原 D1 identity |
| Infrastructure publication / Gate | PASS；原 parent 的 exact lease 发布并取得 receipt |
| Consumer publication | 完成；以上一阶段为 exact parent，取得 receipt |
| Consumer Gate | **STOP**：四个 HTTPRoute 持续 OutOfSync |
| Metrics convergence / live functional probe | 未执行 |
| Repeat compile / publish / successful deploy | 未执行；只能在完整 PASS 后执行 |
| S2 final.json | 不存在；原 terminal.json 为 STOP，deploy exit 1 |

S2 deploy 只调用一次，12:06:06–12:19:08 UTC。确认持续不收敛后取消了唯一精确匹配的进程，
没有等待至 15 分钟 Gate 上限。原程序在取消等待时使用通用错误 `S2 convergence timed out`；
外层执行记录为 `timeout=false`，另有 operator-stop 原因。不能把它描述成真实超时或已通过的 Gate。

两个 WebService 的 Deployment/Service 已 Synced、Pod Healthy，但其 HTTPRoute 未 Synced，
所以不能由 Pod 健康推导 Runtime PASS。没有执行真实 HTTPS→S3 写入 probe，也没有重复部署。
危险的 CreateBucket/DeleteBucket/PutBucketCORS 负例仅在合成 fixture 执行。

## 原因与编译器修正

锁定 Gateway API v1.6.1 在 API 中补入以下字段；S2 compiler 原来省略它们：

| 位置 | API 的实际默认值 |
| --- | --- |
| `parentRefs[]` | `group: gateway.networking.k8s.io`、`kind: Gateway` |
| HTTPS `backendRefs[]` | `group: ""`、`kind: Service`、`weight: 1` |
| HTTP redirect `rules[].matches` | `path.type: PathPrefix`、`path.value: /` |

四个路由在 Argo 报告成功同步后仍为 OutOfSync；逐项 Git/live spec 比对确认这些差异。
修复只让 compiler 显式输出同一默认语义，不改变 hostname、listener、backend、端口、资源 identity
或授权。没有增加 ignoreDifferences、放宽 Gate、手工 patch live spec/tracking 或创建 continuation。

`TestConsumerRoutesMatchGatewayAPIDefaults` 使用本次 API 返回的四个路由 spec 作为非敏感 fixture，
覆盖 bound/unbound 的 HTTP/HTTPS：旧实现四项全部失败，修复后通过。
这证明生成结果与已观测 API 表示一致；**修正后的 Argo/runtime 收敛仍待新的执行决定验证**。

后续根因修复把「schema 合法」与「CRD 默认值不改变新增内容」分开检查，并在 infrastructure
编译／plan 阶段提前验证不依赖密文的 consumer。schema 绑定不可变 D1 base，历史未改动子树
保持原样；缺失默认值、版本错配或 consumer 编译错误在返回输出前拒绝。
实现边界与回归见 [编译时拒绝 CRD 默认值遗漏](s2-workloads.md#编译时拒绝-crd-默认值遗漏)。
这项源代码修复没有重跑本次执行，也不改变上表的 STOP 或任何历史 plan/证据。

辅助证据采集曾因把 JSON 多文档流作为单个 JSON 解码而退出；修正本地读取器后完成基线。
这个辅助问题没有改变产品 executable、计划或外部状态，D1 install/verify 未重跑，S2 当时尚未生成 plan。
原日志保留；不将其隐藏为无瑕疵的操作过程。

## 现场保留与未证明事项

STOP 后只读复核：四节点 Ready，352 个既有持久资源 UID/owner、13 份 D1 冻结文件以及
Identity/Latch/Receipt/External Root 不变。部署分支保留 consumer commit；原 intent、receipt、
terminal、确切二进制/plan、新实例凭据与密钥备份、元数据 audit 完整保留在私有实例目录。
原 `atlas-d1-r2` 与其 STOP/分支不变；默认 kubeconfig、hosts 未改写。
公开仓库只保存脱敏结论和非敏感路由 fixture，不附入凭据、私钥或其关联摘要。

未证明：真实 HTTPS→S3、unbound 网络拒绝、跨项目 Secret 授权拒绝、metrics discovery、S2 后 D1
功能回归，以及成功后的幂等/零写入审计 Gate。禁止把本地修复或 fixture PASS 当成这些 Gate 的替代。
本次仍是同机、有公开镜像缓存的开发实例，不构成独立干净机器、HA 或强多租户生产保证。
