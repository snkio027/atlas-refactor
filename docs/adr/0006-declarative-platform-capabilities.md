# ADR-0006：声明式平台能力目录及首批监测 / S3

- Status: Proposed
- Date: 2026-09-27
- Scope: atlas-refactor 的独立、可丢弃开发平台；不批准生产切换或当前集群变更

## 问题与决策提议

用户需要指标、告警、Grafana 和 S3 对象存储，并要求平台能力扩展方便。
目前 schema 3 把 platform-control 的完整子应用清单和 platform-project 权限冻结进初始身份。
这种绑定把普通 Tier-1 演进误判为重新实例化，需要区分两个生命周期：

1. 初始 substrate、Root、Tier-0 project、宏观 DAG、两个 Seed 及核心控制对象固定。
2. Git 里的 Tier-1 能力清单由 Argo 持续调谐；能力添加不改变初始身份，也不恢复 Bootstrap 写权。

从已经验证的 f6d35ec44dcd12b3812145a561590a94af558ce6 精确提取启动输入快照。
生产代码固定校验快照 SHA；旧 Identity 的 platformSHA256 和 Signal 字节保持一致。
允许变化的两个文件仅为 platform-control 子应用和 canonical platform-project 的增量投影；
它们必须匹配当前能力目录生成结果，不能借此改动核心子应用或 workload-project。
其余冻结文件与快照不符仍 fail closed。历史 evidence 和旧 dev01 baseline 不变。
同包测试使用合成 archive 时，只注入与该 fixture 精确绑定的快照摘要；生产 CLI、配置、环境没有覆盖入口。

声明式目录记录 namespace、资源路径、依赖、readiness wave、Secret 引用及渲染输入。
plan 展示依赖与精确权限增量；select 只写本地 desired state；quality 校验包括全部未启用候选。
只有已有 platform-control 创建新的直接叶子 Application，不引入 umbrella Application 或并行 installer。
Bootstrap 观察器从启用的能力目录读取期望应用集合；每个新增应用仍进入状态 Gate。
缺少凭据/依赖、控制投影偏离声明、未知读取、镜像/制品不符均拒绝继续。

## 参考权威与首批范围

遵循原 Atlas Architecture v1.0.2 §5.3 Secret materialization / §6 Unified Telemetry，
GitOps v1.0.3 的 Root → Control → Leaf、平台控制器与服务波次，以及 Network v1.0 身份选择器。
这些标准的权威归属不转移到本候选仓库；ADR-0002/0003 的语言选择和 cutover 继续独立审批。

首批监测采用 kube-prometheus-stack，避免并列维护第二套指标组件。
S3 选择 SeaweedFS 单进程开发模式。选择依据包括活跃发行、多架构固定镜像及实际 S3 API 验证；
MinIO 官方仓库当前已归档且声明不再维护，不把它作为新的开发依赖。
上游状态依据：[MinIO](https://github.com/minio/minio)、[SeaweedFS](https://github.com/seaweedfs/seaweedfs)。

Sealed Secrets 物化 Grafana/S3 凭据，Git 仅含引用或严格 namespace/name 绑定的密文。
新 controller 的私钥属于新 Trust Root：首次启动及隔离备份需要显式判断门禁。
它不能读取/写入 Argo、kube-system 凭据；不导入原 Atlas 密钥。此 ADR 没有批准生成该信任根。
准备命令只使用公共证书和本地随机开发凭据；重复操作不隐式旋转密钥或凭据。

## 保持的边界与取舍

- 不变更 External Root、atlas-bootstrap、Seed、Root parent/finalizer 或正常恢复权限。
- 不用 Helm release state；图表和镜像都固定，GitOps 消费本地渲染清单。
- namespace 和精确资源类型的增量必须可审查；不引入 AppProject 通配授权。
- node-exporter 的只读宿主文件系统和 hostPID 仅限平台监测 namespace；业务继续 restricted。
- Node/API 指标抓取是平台特权路径；业务访问 S3 仍由 namespace + Pod 标签授权。
- 保留既有四节点角色；持久服务在 data，控制器在 compute，入口仍限制 loopback。
- 保留数据与删除确认不等于备份；local-path 无 HA 和目录硬配额。
- 当前告警仅本地查询，无外部通知投递。完整 logs/traces、备份与高可用后续决策。

普通能力启用无需重新批准 Tier-0 也无需改 Go Bootstrap 代码；新的信任根、宏观依赖链、
信任边界或未覆盖的部署范围仍按架构进行专门审查，不能以“易用性”为由绕过。

## 验证与发布条件

本地质量门禁、身份不变/退化不恢复权限测试、真实 kubeseal 离线测试及隔离 S3 容器 API 测试
支持实现可行性；尚无新增组件在 dev02 上的 Kubernetes/Argo 成功证据。
该 ADR 继续 Proposed，目录默认无启用项。正式接入顺序、访问方式和验收 Gate 见
[平台能力扩展](../platform-capabilities.md)。
