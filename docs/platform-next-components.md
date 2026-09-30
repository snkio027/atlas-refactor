# Web/API 开发平台的下一批能力

本轮实际启用范围是 Sealed Secrets、指标/告警/Grafana、S3；以下是能力评估，
不是已部署清单。现有 Cilium、Gateway API/Envoy Gateway、cert-manager、开发 CA、
本地 Retain PVC、Argo CD 和命名空间网络边界继续复用。

| 优先级 | 能力 | 直接解决的问题 | 建议 |
| --- | --- | --- | --- |
| 下一批 | metrics-server | `kubectl top` 与 HPA 需要 Resource Metrics API；Prometheus 本身不提供该 API | 小规模部署，固定版本/摘要；先确认 Kind kubelet 证书策略，避免默默关闭验证 |
| 下一批 | Loki + Grafana Alloy | 查询多个 Pod 和重启前的日志，定位 Web/API 故障 | 短保留期、单实例开发规格、明确磁盘预算；通过现有 Grafana 查询 |
| 下一批 | 项目接入模板 | 新业务重复配置 namespace、配额、NetworkPolicy、Secret 引用、HTTPRoute、探针、PVC 和 ServiceMonitor | 沿用 Tier-2 workload-project，声明式生成并展示权限差异；不是新增常驻控制器 |
| 按首个业务需要 | PostgreSQL | 事务、用户/订单等关系数据，S3 不能替代数据库 | 先确定应用和数据寿命；评估 CloudNativePG，单实例开发配置；不要默认附带 Redis/Kafka |
| 按需求 | 告警外部通知 | 当前 Alertmanager 仅本地查看，尚不能通知值班人 | 由所有者选择接收渠道并提供 Secret 引用；独立授权外部发送 |
| 稳定性阶段 | 离线密钥备份、数据备份/恢复、准入策略、HA | 故障恢复与生产边界 | w1 临时备份不算离线；PVC Retain 不算备份；分别设置恢复验收 Gate |

当前宿主 OrbStack 约 16GiB 内存由所有 Kind 节点共享；四个节点不等于 64GiB。
先从真实 Prometheus 指标确认资源余量，再启用日志和数据库。此阶段不默认引入
Service Mesh、ExternalDNS、Kafka 或独立缓存集群；当前本地入口不需要外部 DNS 自动化。

普通能力继续通过 catalog → plan → select → quality → GitOps 发布；新增信任根、
跨命名空间权限或供应链来源应明确审查，不能用扩展方便代替边界检查。

## 依据

- [Kubernetes Resource metrics pipeline](https://kubernetes.io/docs/tasks/debug/debug-cluster/resource-metrics-pipeline/)：Resource Metrics API 服务于 HPA/VPA 和 kubectl top。
- [Grafana Alloy 向 Loki 收集日志](https://grafana.com/docs/loki/latest/send-data/alloy/)：沿用原 Atlas Architecture §6 的日志组件方向。
