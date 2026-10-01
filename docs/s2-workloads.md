# S2：把 Web/API 项目编译到已安装平台

状态：实现中；真实集群验收未完成。语义与权限边界见
[S2 契约](s2-semantic-contract.md) 和 [Proposed ADR-0015](adr/0015-typed-project-workload-binding.md)。
本分支从 PR #8 合入后的 `531d234` 开始；S1 与 D1 的历史结果保持冻结。

## 用户输入与编译

用户在自己的 deployment workspace 编写三个严格 JSON 类型：

- `platform/projects/demo.json`：owner、quota、允许消费的 capability。
- `platform/workloads/demo/web-api.json`：WebService 镜像、端口、资源、TLS 和 metrics。
- `platform/bindings/demo/web-api-storage.json`：为 web-api 授权 uploads 的对象读写。

可从 [示例](../examples/s2/platform/projects/demo.json) 开始。示例镜像是本地构建的
`atlas.local/s2-web` OCI 制品，不能从公网拉取；`image` 字符串的合法性不证明制品存在。
正式部署流程校验完整 archive、manifest、config、layer 摘要并把明确镜像导入该实例的四个节点。
导入器按 manifest 验证 tag 来源，然后登记 CRI 使用的 `repo@digest` 名称；最后用 authored
`repo:tag@digest` 查询 CRI 并要求精确 repoDigest。既有相同 canonical 名称不重复登记，冲突不 force 覆盖。
纯编译器支持契约中任意合法 pinned WebService 镜像；本次本地验收执行器只接收这一个可核验的 demo 制品。

`internal/workload` 不执行外部工具、不读取集群、不生成随机数、不推送 Git。
它完成严格 parse → validate → resolve → compile → scope/schema/ownership check，输出 canonical GitOps。
Unknown 字段、重复键、大小写别名、null、错误引用、缺失授权、名称冲突与 quota 超额均拒绝。
quota 计入每个 Deployment 的一个 surge Pod。无 Binding 的 Workload 不生成凭据引用或 S3 网络许可。

生成的 Namespace/quota/policy/SA/ServiceMonitor 由新 Tier-1 Project leaf 管理；
Deployment/Service/HTTPRoute 属于 Tier-2 Workload leaf；两端 S3 NetworkPolicy 属于 Tier-1 Binding leaf。
共享 Gateway、TLS、monitoring、Sealed Secrets RBAC 和 provider auth 仍由各自既有 Application 管理。
编译器不创建 Secret 明文，不改变 External Root、Seed、handoff latch、Receipt 或 Signal。

## 构建与准备

需要已完成的 D1 `darwin/arm64` 本地安装、原始 release package、其私有 installation state，
以及相同实例的有效 sealing certificate 和已有备份 receipt。不会导出或轮换 controller 私钥。

```sh
# 在产品源码根目录；依赖 Go 1.27.1，使用标准库，没有网络构建依赖。
mkdir -p .state/s2/artifacts
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -buildvcs=false \
  -ldflags=-buildid= -o .state/s2/artifacts/atlas-web ./cmd/atlas-web
go run ./cmd/atlas-web-image --binary .state/s2/artifacts/atlas-web \
  --out .state/s2/artifacts/atlas-web.oci.tar
go build -trimpath -o .state/s2/atlas-platform ./cmd/atlas-platform
```

OCI 打包是 create-only；相同 executable 得到相同 archive，不自动覆盖已登记制品。
把输出的完整镜像引用填到 Workload；把 archive SHA256 填到私有配置。
正式执行要求编译二进制绑定一个干净的源码 commit，修改代码后需重新生成并审查计划。

私有 `s2.json` 示例，所有路径替换为当前实例的绝对路径：

```json
{
  "schema": 1,
  "installationConfig": "/absolute/instance/installation.json",
  "installationPackage": "/absolute/instance/package",
  "intentDirectory": "/absolute/deployment-workspace",
  "productSource": "/absolute/atlas-refactor",
  "stateDirectory": "/absolute/s2-state",
  "imageArchive": "/absolute/atlas-web.oci.tar",
  "imageSHA256": "<archive SHA256>"
}
```

S2 state 必须与 D1 state 独立。archive 与私有 state 文件使用 owner-only 权限。
首次验收意图额外包含同 namespace 的第二个 WebService `unbound`，使用同一镜像、不同 hostname，
不声明 Binding；它提供网络隔离负例。该负例不能通过一个根本不存在的 Pod 来证明。

## 计划、发布与观察

```sh
atlas-platform workload plan --config /absolute/s2.json
atlas-platform workload compile --config /absolute/s2.json --phase infrastructure
# 审查生成资源、namespace/RBAC、公开密文目标、镜像与 probe 范围以后：
atlas-platform workload deploy --config /absolute/s2.json --approve-plan <SHA256>
```

`plan` 只下载指定公开 deployment Git 分支到独立 bare repository，读取本地安装元数据并编译计划。
`compile` 只使用已准备的本地 Git 对象与文件，不联网、不碰集群。
二者都不读取 Secret 或私钥，不启动集群。正式部署不创建/删除集群。

一次 `deploy` 有限完成以下步骤，异常即停止：

1. 核对 cluster UID、D1 authority、四节点和当前 Git；拒绝接管已占用的 namespace；记录相关旧资源 UID。
2. 校验并导入已登记 OCI；校验现有 public certificate/backup receipt，保留 D1 旧 identity，为每个新 Binding 生成独立凭据。
3. 发布 infrastructure：精确 destination、namespace/quota/SA/policy、controller RBAC、TLS、monitoring selector。等待 Argo 与 live content/ownership 验证。
4. 发布 consumer：新 client SealedSecret、聚合 provider SealedSecret、Binding、Workload/Route 和受控 provider rollout。再次验证。
5. 执行 HTTPS→Web→S3 put/get/delete、无 Binding 网络拒绝、跨桶/管理读取拒绝、SA 跨 namespace Secret 拒绝、metrics 发现、旧 D1 HTTPS/S3 凭据验证。
6. 保存绑定 implementation/binary/plan/Git/cluster/resource UID 的 final evidence。

所有 Git 发布都保留完整 parent tree，仅替换编译器声明的 delta，并用 exact-parent lease 拒绝并发更新。
已有无关文件、可执行位及 Git 历史不重建。重复成功 `deploy` 只观察；重复发布相同 consumer 不产生新 commit。
首次随机密文保存在独立注册文件中，重复编译使用原字节；编译本身不重新加密。
已有对象 identity、hostname、port、Binding 关系与授权不允许通过 update 隐式删除/退役。

`prepare-credentials`、`baseline`、`publish`、`probe` 是同一精确计划的显式分步入口，便于审查诊断；
不构成 recovery/resume 接口。外部状态变更后失败保留 intent/receipt/terminal evidence，不自动回滚或清除 STOP。

日常只读观察：

```sh
atlas-platform workload observe --config /absolute/s2.json \
  --phase consumer --revision <published commit> --wait 10m
```

观察核对 Git 输出、Bootstrap authority、App spec/Sync/Health/idle、live identity/字段/SSA/tracking、
相关旧 UID，并做 closing proof。普通未变 leaf 只允许该有限发布序列中 source closure 相同的 revision；
控制 Application 和所有 mutation fence 保持 exact-current。观察不 refresh、不 sync、不执行 Pod 命令。
其 `Runtime: UNPROVEN` 明确表示没有重做写入型功能探测；完整 Runtime VERIFIED 来自独立 probe 的最终证据。

## 权限与验证边界

新 Binding 使用 SeaweedFS 4.47 的 policy-only identity，授予 uploads 对象 CRUD、Tagging、multipart 和 List，
不使用包含 bucket 配置权限的 legacy `Write:uploads`。原 D1 identity 保持原样。
依据是锁定 [4.47 的 IAM 实现](https://github.com/seaweedfs/seaweedfs/blob/4.47/weed/s3api/auth_credentials.go)
及本仓库的隔离运行回归，而非仅比对权限字符串。

```sh
task quality
# 使用固定镜像、合成凭据、tmpfs 数据和 loopback 随机端口；结束清理本测试容器。
ATLAS_S2_PROVIDER_FIXTURE=1 go test ./internal/workloadrun \
  -run '^TestSeaweed447PolicyFixture$' -count=1 -v
```

`task quality` 包括 S2 两种 phase 的真实 Kustomize 构建；不会启动 Docker/Kind 或使用实例凭据。
隔离 provider fixture 已验证对象读写、Head/List/Tagging/multipart，以及跨桶、CreateBucket、DeleteBucket、
PutBucketCORS 的 403。危险的 bucket mutation 负例仅作用于这个空的合成 fixture，绝不在真实 uploads 上试删。

本机首次执行在只读 baseline 阶段 STOP，尚未导入镜像、生成凭据或发布 Git；真实 slice 尚未通过。
原因是 Helm 的空 annotations、core ServiceAccount 空 apiGroup、API 省略的探针零延迟与 false 字段
被当作 drift。修正只覆盖这些已知 Kubernetes 字段；非默认值、额外 RBAC subject 与网络权限仍拒绝。
回归见 `TestAbsentAuthoredAnnotationsPermitArgoTracking`、`TestKnownAPIOmissionsPreserveAuthoredValues`、
`TestBindingSubjectDefaultDoesNotBroadenPermissions`。这次失败没有外部 mutation，保留本说明与回归，
本地只保留 latest 工作材料。基线也独立核对每个既有对象的 owner tracking 与 Argo SSA。

第二次获批执行 `a6aa7cf` / plan `0e64333d` 的远端 Quality 为 PASS，baseline 通过后在首节点
CRI image inspection STOP。控制平面已导入正确锁定镜像并登记 literal `repo:tag@digest`，
但 CRI 按 `repo@digest` 查找而失败。三个 worker 未导入；没有准备凭据、发布 Git 或执行 probe。
四节点仍 Ready，Git parent 仍为 `70147b2`。这次有外部镜像写入，原 plan、decision、baseline、
terminal 和 post-stop 诊断保留在对应 authority bundle；不清除 STOP，不自动重试。

根因依据锁定 [containerd 2.3.1 LocalResolve](https://github.com/containerd/containerd/blob/v2.3.1/internal/cri/server/images/service.go#L147-L174)
与 [ParseDockerRef](https://github.com/distribution/reference/blob/v0.6.0/normalize.go#L80-L112)。修正不改镜像制品、
compiler、密文、GitOps 清单或 D1 engine；只修正 S2 的 canonical alias，新增精确 manifest 来源检查和
分阶段错误定位。回归 `TestImageImportRegistersCRINameAndChecksAuthoredReference` 覆盖首次及已有同 digest，
`TestImageImportFailsClosedAtEveryBoundary` 覆盖错误 repo/digest 和每步立即停止。
回归是本地过程模拟，修正后的真实 CRI 导入尚未执行，不能记为 runtime PASS。

修正后，同一个 Go 基线函数已只读验证当前实例 145 个资源的内容、UID 与 ownership；显式复核命令：

```sh
ATLAS_S2_BASELINE_CONFIG=/absolute/path/to/s2.json go test ./internal/workloadrun \
  -run '^TestS2ReadOnlyBaseline$' -count=1 -v
```

该 opt-in 测试只做实例绑定检查、GET、Git read 和本地编译，不导入镜像、不使用凭据、不写 authority evidence。
上述单元/隔离/只读测试不能替代 TLS、Cilium、Argo、Sealed Secrets、Prometheus 的实际组合验证。
S2 不是生产多租户、安全隔离 admission、端到端 mTLS 或 HA 声明。发布与最终验收保持同一个 PR。
