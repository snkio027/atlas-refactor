# ADR-0018：普通业务交付与工作区

Status: Proposed — 2026-10-09；仅限单所有者本地开发，待维护者审核。

## 问题与控制边界

ADR-0015 的编译器支持合法 WebService，但 S2 执行器把 demo 镜像、unbound
陪测和 /roundtrip 当作普通部署前提。使用者还须拼接配置路径、阶段及 revision。
本变更服务 experiment-archive，新增薄的使用层，不再扩充 S2 验收范围。

复用 ADR-0016 的固定四阶段和 ADR-0017 的只读 Gate；配置/部署范围的增量由本
ADR 单独审查。Tier-1 的 Project/Binding/provider 适配、Tier-2 的 WebService
仍由现有 compiler/publisher 投影。没有 Tier-0、Bootstrap、Root、owner transfer、
新组件、恢复、解绑/退役或任意 YAML 权限。Git owns definition；Argo owns reconciliation。

## 决策

- 普通入口为 `atlas-platform app`；高级 `workload` 验收接口保留。底层 Config
  增加可选 `purpose=application`；省略时仍为原 S2 验收。字段参与 ConfigSHA，
  不能把原批准计划转成另一种执行。旧配置 canonical JSON 与原证据不重写。
- 接受一个经完整 OCI closure 校验的 linux/arm64 manifest，所有本次 Workload
  使用同一已登记镜像；tag、digest、containerd canonical name、CRI identity 一致。
  单镜像 index 拒绝额外 image alias。离线静态 Go 打包只提供 scratch executable，
  不下载、不构建任意 Dockerfile、不引入 registry 拉取或构建平台。
- 普通 Deploy 仍执行 baseline、镜像导入、必要的独立 Binding 凭据准备、固定
  publication 和每阶段 Gate。普通完成验证 materialized Secret、Ready Pod、
  HTTPS GET /readyz、声明的 metrics、closing ownership Gate；不执行 POST、
  S3 对象操作、exec、RBAC 负例或原 demo Probe。
- 普通终态为 DEPLOYED，functional=UNPROVEN。S2 的 PASS/Runtime VERIFIED 和
  probe intent 保持原义。provider 前置记录成功后才写 final；矛盾终态拒绝。
  已成功 deploy 重复时只观察。没有“默认业务测试通过”的推断。
- 初始化工作区生成既有三个 JSON 类型，绑定完成安装的 instance directory、
  匹配产品源码和制品。自动维护配置、私有 run state、内容寻址意图快照及当前计划。
  不复制安装私钥/凭据，不读取默认 context；实例含 installation.json，包在同目录或 package/，歧义布局拒绝。
  当前一个新增 Project 与 consumer-only 更新限制保持，更新不能增删 Workload。
- 人工 plan 默认展示目标、前后镜像/副本/资源、TLS/Binding、公开密文目标和外部动作。
  交互确认精确 target/project；内部绑定完整计划摘要，并在确认后重读所有输入。
  自动化必须同时给计划文件和批准摘要，不接受无条件 --yes。STOP/部分写入不能通过
  生成新计划掩盖；工作区保留原计划并指向只读诊断。
- status 从记录的输入快照及连续 receipt 链解析 phase/revision，区分实时 authority、
  readiness、历史终态/功能验收；不依据待发布编辑作实时声明。控制器原始消息不进入
  默认摘要。open 使用仅 loopback HTTP 到证书验证的 HTTPS 代理，显示实例、端口、
  TLS 与退出方式；不修改 hosts、CA 信任、默认 kubeconfig。logs 是显式读取业务日志，
  不等于保证应用自己不会记录秘密。诊断不获得修复权限。
- 同一实例的安装/部署互斥仍使用 D1 lock，status/open 使用共享锁。没有 daemon、
  stale-lock 清理、generic resume、自动 rollback 或新的 evidence 协议。

交互原则参考 [CLI Guidelines](https://clig.dev/)：人类可读默认输出、JSON 脚本输出、
stderr 进度/错误、明确确认和下一步提示。本文选择上述有限实现；补全、编辑器 schema、
完整 workload 发行包和多服务连接管理仍为后续需求。

## 验证与未证明事项

本地验证须覆盖：自有镜像/无陪测编译、固定摘要/OCI 错误拒绝、确认后输入变化、
STOP/部分发布不能替换计划、provider 失败无 final、普通 Probe 拒绝、旧 S2 路径回归，
以及真实 Helm/Kustomize 和 task quality。

用户验收仍须独立证明实际首次部署、两次正常更新、访问、业务错误定位和再次使用。
本地回归不等于这条使用闭环已通过。r7 继续绑定 d011813；本改动不重新归属该实测，
不动其集群、部署分支、凭据或历史证据。新业务目标须按精确计划另行授权。
