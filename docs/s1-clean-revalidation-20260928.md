# S1 干净集群复验：平台通过，ceremony STOP

2026-09-28（Asia/Shanghai）。按用户要求删除旧 OT-1，并从零创建四节点环境。
**Bootstrap 与完整平台验收通过；普通 29 阶段执行通过 12/29，在 index 12 超时 STOP。**
当前平台只读验证正常，不等于完整 ceremony 或第二次 Gate-B 已通过。
本轮没有修改 Go 实现、增加 continuation、清锁、刷新 Argo、自动恢复或放宽预算。

## 精确绑定

| 项目 | 值 |
| --- | --- |
| 实现 | `13b3577759988d91595c8a6c88bc2ab8bf44be5a` |
| atlas-ot1 binary SHA256 | `a15056d0ada62d7abe9c2101b20a77888c504fa90b68cdaa86368c6ae605c34b` |
| Plan SHA256 | `67f00b253dedd8aa707a941aeb914cbb32b49da1e5c7206f2c148e291637dba7` |
| Toolchain | Go 1.27.1、CGO_ENABLED=0、-trimpath、clean VCS；锁定工具见 plan |
| 旧 UID（已删除） | `b886f730-ea3b-44b9-a904-8bd55ac345f2` |
| 新 UID | `478c4e71-fd24-4e90-bec9-e04e20af4a01` |
| 集群 | atlas-refactor-test-ot1；control-plane + gateway + compute + data |
| 完整平台基线 | `1403467e453b53180472381c11730a6d40e3901b` |
| 停止后 Git | `8809173c0206b77ab279cd7845c7e8df7052d4a0`，runtime 与远端一致 |
| 预算 | 29 阶段、每阶段 300 秒；Gate 位于 0/12/23/28 |
| 执行时间 | 2026-09-27 18:38:04–18:48:04 UTC；exit 2 |

只删除旧 OT-1 四个 Kind 节点；registry 和其他项目容器保留。七份既有 authority bundle
删除前逐文件验证，旧 Trust Root 备份保留。新环境未导入旧 identity、Receipt、密钥或凭据。
新私钥按已批准的 OT-1 同机开发例外另存 `w1/atlas-refactor-ot1/rebuild-20260928`。
三份新 strict SealedSecret 经明文扫描，追加发布至实验分支；私钥/明文/kubeconfig 未入 Git。
默认 kubeconfig 未变，SHA256 仍为 `862e14c996585bf3b2c9f4cb4026bffdd56f614faea8c0a85b099b98949c651d`。

## 实际通过

- 从零 Bootstrap：doctor、确定性 render、四节点、Cilium/Argo Seed、GitOps handoff、
  ADOPTED、HTTPS 与 PVC/PV；重复 apply exit 0、身份不变、kubectl mutation delta 0。
- 完整平台：24 Apps idle/Synced/Healthy；10 个 capability 镜像在四节点逐一验证。
- S3 签名读写、HEAD、删除、预签名、分片上传通过；匿名及跨 bucket 请求均 403。
  测试对象已删除。Grafana 登录、26 个看板与 Prometheus datasource 通过。
- Prometheus 27 targets 全 UP、31 rule groups、4 节点指标齐全；Alertmanager 本地
  告警触发/解除通过，未测试外部通知。
- 普通执行从 index 0 开始：0..11 全部通过，包含基线完整 Gate-B、source release、
  secrets/observability strict refusal → window adoption → strict restore，以及
  mixed rollback 的 source strict refusal → readoption → strict restore。
- 21 个受保护 Application 写请求、2 次 Git 发布（6 个 Git 命令）全部 exit 0。
  独立审计得到 21 条 kubectl mutation；7,217 个 baseline audit IDs 全部保留，
  post-STOP 为 10,050 IDs；受测资源写入通过既有 auditScope 检查。

## index 12 的阻塞

`MIXED_ROLLBACK_VERIFIED` 在 18:42:56.573 UTC 开始，目标 Git 为 `8809173`。
它发布 source 重新挂接定义，并等待父应用恢复 automated spec 与关键 owner exact-current。
最后 readiness precheck 的全部阻塞经离线重放确认为：

| Application | 未满足的前提 |
| --- | --- |
| platform-control | 仍报告前一版 `0099ec2` |
| capability-foundation | 仍报告 `0099ec2`，且 spec 仍只有 ceremony 手动 strict syncOptions，缺少目标 automated enabled/prune/selfHeal |

这些关键 owner 要求不能用普通 leaf 的 Desired Identity equivalence 绕过。
本次不是 F14 的多版普通 leaf 拒绝，也没有证据表明应修改 F12 operation/comparison 规则。
历史保持 **STOP / NextIndex=12**，没有 index-12 checkpoint 或 Gate artifacts；13..28 未执行。

未施加干预后，source 在 18:48:02、platform-control 在 18:48:22 记录当前 revision，
均晚于阶段 deadline。尚未独立归因 Git ref cache、controller 排队或各调谐周期的贡献，
不能仅据本次结果声称“增加超时”就是修复。

## STOP 后的当前状态

18:50:32 UTC 的新只读 capture 为 VERIFIED；既有 Assess 得出 **ownership VERIFIED、
Atlas NOT_PROVEN**。13 个资源仍归 capability-foundation，UID、semantic content、SSA
相对新基线保持，窗口为 0；Bootstrap records、Root/self 与 AppProjects 检查通过。
只读 `atlas status --check` 为 ADOPTED，`atlas-dev verify` 的四节点、Pod/放置、PVC/PV、
HTTPS 通过。最终 inventory 为 24 Apps idle/Synced/Healthy，全部报告 `8809173`。
这不补写原 checkpoint，也不构成完整 29 阶段成功。

精确 STOP lock 保留，SHA256：
`7de090b7f342b6ebdb2d2dd21f41ff9ba0a51583c468996ce222916c7a2966e5`。

## 证据与审核

私有 bundle：`.state/authority/ot1-clean-67f00b25/`，168 files、120,643,502 bytes。
Manifest SHA256：`7c13628c8564e236d537d240480c26ea24cac85f535bdf42c3799652f65f3618`。
保留实际 executable、plan、七 revision Git bundle、启动/备份 receipt、请求与 checkpoint、
失败观察、Metadata audit、STOP、只读复核及本地重放脚本。runtime checkout 未移动，
运行锁未修改；latest 保留当前 checkout/desired 与结果指针。

原 [S1 最终验证](s1-final-validation.md) 的跨 attempt 成功证据保持不变；本轮新增的限制是
单次干净执行尚未通过。PR #6 仍为 draft，ADR 仍 Proposed，不构成生产 cutover。

本次报告提交前 locked `task quality` 通过；未新增实现或修改运行时门槛。

## 手动只读检查

```bash
cd /Users/nekoreb/Workspace/03_Projects/atlas-refactor
export KUBECONFIG="$PWD/.state/latest/ot1-clean/source/.state/development/atlas-refactor-test-ot1/repo/.state/kubeconfig"
.state/tools/kubectl --context kind-atlas-refactor-test-ot1 get nodes
.state/tools/kubectl --context kind-atlas-refactor-test-ot1 get pods -A
.state/tools/kubectl --context kind-atlas-refactor-test-ot1 -n argocd get applications
```

入口仍为本机 loopback 18080/18443；以上命令不更改默认 kubeconfig。
