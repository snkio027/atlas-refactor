# OT-1 首次干净重建与验证（2026-09-27）

四节点和完整开发平台已经完成真实验收；OT-1 所有权演练在第 0 阶段 STOP，尚未发生资源转移。结构化结果见 [证据摘要](evidence/ot1-clean-rebuild-20260927.json)。ADR-0009 保持 Proposed。

## 目标与结果

- 删除旧 `atlas-refactor-test-dev02`、`atlas-refactor-test-ot0`，确认旧 Kind 集群清空后创建新的 `atlas-refactor-test-ot1`。历史证据、备份和无关容器保留。
- 新集群为 control-plane、gateway、compute、data 四节点，全部 Ready；HTTP/HTTPS 仅绑定 `127.0.0.1:18080/18443`。
- 完整平台基线 `541705df77a500d784f808fe036794b5e2d551d8` 已发布到独立 OT-1 分支，24 个 Application 均在该 SHA 上 idle、Synced/Healthy；普通 Bootstrap `ADOPTED / 0`。
- Core 重复 apply exit 0，审计 kubectl mutation delta=0；扩展完整平台后 Root、Identity、Latch、Receipt、Signal 和 Argo self 的 UID 与 core 验收记录一致。
- PVC Bound、PV Retain、节点放置、HTTP 301 和验证证书的 HTTPS 200 均通过。
- Grafana 登录、Prometheus 数据源及 26 个看板通过；Prometheus 27 个目标全部 up，31 个规则组、四节点指标可用。
- S3 签名 CRUD、预签名 GET、分段上传、匿名与跨桶 403 通过；Alertmanager 本地 API 测试告警 firing/resolved 通过。未验证外部通知或完整 Prometheus 规则触发链。
- 新 Sealed Secrets key/cert 配对与独立备份通过，三份新密文经明确批准公开。使用已批准的 OT-1 同机 w1 开发备份例外，物理隔离要求未满足。

## 保留的启动记录

首次 core 等待被本地中断（exit 1），保留为独立 INTERRUPTED。实际 CRD 已 Established，Argo 资源树曾缓存旧 resourceVersion；随后自行收敛。正常 Bootstrap 后续调用补齐 Receipt 并完成验证，未手工刷新、重启或补丁修复。不能把这次结果记成首次无中断启动成功。
能力 Application 曾早于 AppProject 获取新提交，受到 namespace 权限拒绝；AppProject 经 GitOps 更新后，既定重试收敛。此时序问题被记录，未手工扩大权限。

## OT-1 STOP：基线错误要求本阶段操作记录

批准 plan SHA256：`3b3d89475fac4d4d977aca05a0bb4a9598b24f8eb7a709ab369a65ac262adbfb`。执行实现为 clean `a8ddf72bc86478f6c6a321c63417a29ad3f17d6c`，run exit 2，terminal=STOP，nextIndex=0。

`capability-foundation.status.sync.revision` 已为 `541705d`，最近成功 operation 仍为 `83521c8`。两次提交中 foundation 的期望内容摘要完全相同。该 App 没有 `ot1-stage=BASELINE_ADOPTED` 请求标记。

[Steps](../internal/ot1/actions.go) 对 index 0 不生成同步请求，但 [applicationProgress](../internal/ot1/executor.go) 仍把基线 activeOwner 送入 [operationMatches](../internal/ot1/evidence.go)，要求本阶段操作 SHA 和标记；后续 Assess 也有相同操作门禁。这使正常自动同步形成的基线被拒绝。

修正应区分基线观察与实际提交的阶段操作，继续保留当前 spec/revision、identity、semantic、tracking、SSA 及独立 Bootstrap Gate；不能全局放宽操作标记或使用陈旧成功掩盖实际阶段失败。需补真实自动同步形态的回归样例。

## 只读诊断：内建 NodeList 成员缺少 TypeMeta

停止后用同一 Go observer 做独立只读 capture，Application inventory=24、AppProject inventory=4，随后 `INVENTORY_UNAVAILABLE`。原始 API 返回顶层 `apiVersion=v1, kind=NodeList`，四个成员均省略 `apiVersion/kind`。

[APIReader.List](../internal/observation/reader.go) 要求每个成员的 TypeMeta 显式等于目标 GVK，因此拒绝真实内建列表。修正应仅从已验证的 typed-list 外层补齐缺省 TypeMeta，并继续拒绝冲突值、身份/作用域错误、不完整分页和缺失版本证据；Secret 读取边界不变。该诊断没有形成 OT-1 Gate PASS。

## 停止后的边界

本次 transfer attempt 没有 `request-*-intent.json`；执行时间窗的 Metadata-only 审计中 kubectl mutation=0。没有发布七阶段计划中的后续提交，没有 release owner、打开窗口或修改 tracking。原运行锁、intent、失败观察和 terminal 均保留，未自动续跑、修复或覆盖。

下一次演练须先修正这两处实现并验证，再以新的 clean binary 和新 plan 获得批准。当前完整平台可继续用于已验证的开发用途；ownership partial rollback、full forward/reverse、OT-1A/OT-1B 尚未证明。

## 本地访问与证据

原始证据位于私有 `.state/ot1-live-20260927/`；公开摘要只包含事实、版本和哈希，不含凭据明文、私钥、kubeconfig 或原始 Secret。历史验证报告保持原字节。

默认 kubeconfig 未切换；在 atlas-refactor 根目录可用：

```sh
export KUBECONFIG="$PWD/.state/ot1-live-20260927/source/.state/development/atlas-refactor-test-ot1/repo/.state/kubeconfig"
.state/tools/kubectl get nodes -o wide
.state/tools/kubectl get pods -A
.state/tools/kubectl get applications -n argocd
```
