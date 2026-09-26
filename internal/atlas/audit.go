package atlas

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Metadata records request identity, verb, object and result without bodies.
// In particular, Secret and kubeconfig contents never enter the audit stream.
var auditPolicy = []byte(`apiVersion: audit.k8s.io/v1
kind: Policy
omitStages:
  - RequestReceived
rules:
  - level: Metadata
    verbs: [create, update, patch, delete, deletecollection]
  - level: None
`)

func (a *App) auditedKindConfig() ([]byte, error) {
	auditDir, e := safePath(a.Root, ".state/audit")
	if e != nil {
		return nil, e
	}
	if e = os.MkdirAll(auditDir, 0700); e != nil {
		return nil, e
	}
	patch := `apiVersion: kubeadm.k8s.io/v1beta4
kind: ClusterConfiguration
apiServer:
  extraArgs:
    - name: audit-policy-file
      value: /etc/kubernetes/atlas-audit/policy.yaml
    - name: audit-log-path
      value: /var/log/kubernetes/atlas-audit/events.jsonl
    - name: audit-log-mode
      value: blocking
    - name: audit-log-maxsize
      value: "50"
    - name: audit-log-maxbackup
      value: "5"
  extraVolumes:
    - name: atlas-audit-policy
      hostPath: /etc/kubernetes/atlas-audit
      mountPath: /etc/kubernetes/atlas-audit
      readOnly: true
      pathType: Directory
    - name: atlas-audit-log
      hostPath: /var/log/kubernetes/atlas-audit
      mountPath: /var/log/kubernetes/atlas-audit
      readOnly: false
      pathType: Directory
`
	config := Object{"kind": "Cluster", "apiVersion": "kind.x-k8s.io/v1alpha4",
		"networking": Object{"ipFamily": "ipv4", "apiServerAddress": "127.0.0.1"},
		"nodes": []Object{{"role": "control-plane", "kubeadmConfigPatches": []string{patch}, "extraMounts": []Object{
			{"hostPath": filepath.Join(a.Root, ".state/audit-policy.yaml"), "containerPath": "/etc/kubernetes/atlas-audit/policy.yaml", "readOnly": true},
			{"hostPath": auditDir, "containerPath": "/var/log/kubernetes/atlas-audit", "readOnly": false},
		}}}}
	if a.development != nil {
		config["networking"] = a.development.kind["networking"]
		nodes := a.development.kind["nodes"].([]any)
		// Copy the approved node topology; attach audit mounts only to the API node.
		var copied []Object
		for _, node := range nodes {
			copy := Object{}
			for k, v := range node.(map[string]any) {
				copy[k] = v
			}
			copied = append(copied, copy)
		}
		audit := config["nodes"].([]Object)[0]
		copied[0]["kubeadmConfigPatches"] = audit["kubeadmConfigPatches"]
		copied[0]["extraMounts"] = audit["extraMounts"]
		config["nodes"] = copied
	}
	return json.MarshalIndent(config, "", "  ")
}
