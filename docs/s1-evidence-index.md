# S1 evidence index

全部 runtime evidence 位于本机私有 `.state/authority/`，不上传公开 Git。
本页给出长期定位和摘要；不是原始证据的替代品。

## 最终干净单次 PASS

- Bundle：`.state/authority/ot1-s1-2278dbac`
- Manifest SHA256：`acbccad8e7ca47c1f2bae3fd7cca61aa3c585a077280bc15d914f9f06bec36fc`
- 366 个被索引文件，242,319,148 bytes；单次 29/29，REVERSE_VERIFIED / exit 0。
- `RETENTION.json`：目标、实现、保留边界；`REVIEW.md` 与 `setup-evidence/batch-approval.json`：整批批准。
- `plan.json` / `execution-decision.json` / `authority-executable/`：精确计划、绑定和实际 executable。
- `attempt/`：29 intent/checkpoint/snapshot、四组原始 Gate、请求结果、六次 publication receipts 与 terminal。
- `audit/`：Metadata-only 原始审计；`desired-state.bundle`：七个投影的 Git 历史。
- `setup-evidence/`：旧目标清理、新平台、密钥备份回执、三份密文摘要、工具链及实际测试日志。
- `final-checks/`：独立验证源码、29-stage 离线复核日志、最终 live/audit/timing 汇总。

归档前执行路径为 `.state/latest/s1-final`；复查保存的辅助源码时，将其 evidence 路径映射到
本 bundle。历史 runtime 目录保留在 `.state/latest/s1-final/source/.state/development/atlas-refactor-test-ot1/repo`，
其私有 kubeconfig、凭据和可写审计未迁入 bundle。私钥另存已批准的独立 w1 备份目录。
原始失败样本和旧执行结果没有替换成新结果。

2026-09-30 按所有者“删除本机旧集群”的指令，在重新核验上述 366 项证据和旧私钥备份后，
已删除此四节点 Kind 集群（UID `b80928e0-71fa-450c-b3f3-3db06c6c22a7`）。
归档、历史私有目录和外部备份仍保留；这里的 S1 PASS 是历史验收，不表示该集群仍在运行。
清理回执位于 `.state/latest/d1-cleanup/result.json`，D1 的新实例另行绑定。

## 历史 authority evidence

重建前逐一验证以下 10 份 manifest 及 1,548 个文件；旧 key backup 也完成摘要验证。
以下文件夹均以 `.state/authority/` 为前缀。

| Bundle | 历史结果 | Manifest SHA256 |
| --- | --- | --- |
| `ot1-a89a3176` | STOP；首次 ownership mutation，7/29 | `de618cc101c4e60b49441484c500c246b611ef798adeec420bd726887ba06274` |
| `ot1-b195dc63` | STOP；SOURCE_RELEASED，1/29 | `b48895b8a871c0d685b66a7bfc5fe829f7f7408d8e954542d8c2fa06bff1eb74` |
| `ot1-clean-67f00b25` | STOP；干净运行 index 12，12/29 | `7c13628c8564e236d537d240480c26ea24cac85f535bdf42c3799652f65f3618` |
| `ot1-continuation-3e8db44c` | STOP；F12，index 6 | `57a0e0f74aaa11220a0f37625da0f2b0035060b5faf6e66d6259355534a7a806` |
| `ot1-f12-8aace8f1` | STOP；index 12；F12 operation/comparison 已验证 | `3ada495c8740562e9b06fe4b855b245b6e062c2d48a9ec9c9f6c5c40d2d96f18` |
| `ot1-f13-readonly-cf093d75` | 只读 stage-12 Gate；不推进历史 checkpoint | `67e13e2606a337eb0214bbe0cad6a167c20aa3940a15bd8dbb45a56ccd2be8b0` |
| `ot1-f15-diagnosis-67f00b25` | F15 只读诊断/时序 | `0075677ceaf6a06bea9ee55f691ae25e3943688e866135431bfbe31990e4c79c` |
| `ot1-final-91dc1dfe` | 历史跨 attempt 闭环；不等于干净单次 PASS | `ee76df575e7745643f555af97f343b7bf20f3b923b77ddd7c3e09ada2a415e61` |
| `ot1-notify-fd2b9bc3` | STOP；F16，2/29 | `76a1dddadf4bfeed006b9213022321c3756c148b707d8e73e4334f5cc01b927a` |
| `ot1-stage12-619e492b` | STOP；forward 已转移，index 23 | `252baa12ed1d3b56d8c92690bf77981270e1f310c7b3ca915e3c427f62ecae68` |

最终新 bundle 不复制这些历史包；其 manifest 在 RETENTION 中引用。清理的是批准的旧集群，历史事实和 Trust Root 备份不删除。
