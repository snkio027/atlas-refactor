package atlas

import (
	"context"
	"strings"
	"testing"
)

func (s *simulator) fourNodes() []byte {
	items := []Object{}
	for i, suffix := range []string{"-control-plane", "-worker", "-worker2", "-worker3"} {
		labels := Object{}
		spec := Object{}
		if i > 0 {
			labels["node-role.local/"+[]string{"gateway", "compute", "data"}[i-1]] = "true"
		}
		if i == 3 {
			labels["topology.kubernetes.io/zone"] = "data-zone-1"
			spec["taints"] = []Object{{"key": "node-role.local/data", "value": "true", "effect": "NoSchedule"}}
		}
		ready := "True"
		if s.objects[key("DaemonSet", "kube-system", "cilium")] == nil {
			ready = "False"
		}
		items = append(items, Object{"metadata": Object{"name": s.app.Config.Cluster + suffix, "labels": labels}, "spec": spec, "status": Object{"nodeInfo": Object{"architecture": "arm64", "operatingSystem": "linux", "kubeletVersion": "v" + s.app.Lock.Kubernetes}, "conditions": []Object{{"type": "Ready", "status": ready}}}})
	}
	return jsonBytes(Object{"items": items})
}
func TestFourNodeImportsAndAudit(t *testing.T) {
	a, _ := developmentFixture(t)
	counts := map[string]int{}
	original := a.Runner
	a.Runner = runnerFunc(func(ctx context.Context, q Request) ([]byte, error) {
		if q.Tool == "docker" && strings.Contains(strings.Join(q.Args, " "), "images import") {
			counts[q.Args[3]]++
		}
		return original.Run(ctx, q)
	})
	apply(t, a)
	for _, suffix := range []string{"-control-plane", "-worker", "-worker2", "-worker3"} {
		if counts[a.Config.Cluster+suffix] != len(a.lockedImages())-1 {
			t.Fatal(counts)
		}
	}
	b, e := a.auditedKindConfig()
	if e != nil {
		t.Fatal(e)
	}
	var k Object
	_ = decode(b, &k)
	nodes := k["nodes"].([]any)
	if len(nodes) != 4 || nodes[1].(map[string]any)["extraPortMappings"] == nil || nodes[0].(map[string]any)["extraMounts"] == nil {
		t.Fatal(string(b))
	}
}
func TestReceiptAllowsUnchangedCRDAtNewGitCommit(t *testing.T) {
	a, s := developmentFixture(t)
	apply(t, a)
	for _, o := range s.objects {
		if o["kind"] == "Application" {
			o["status"].(Object)["sync"].(Object)["revision"] = strings.Repeat("b", 40)
		}
	}
	a.resolvedCommit = strings.Repeat("b", 40)
	if r := a.Status(context.Background()); r.State != Adopted {
		t.Fatal(r)
	}
	delete(s.objects, key("ConfigMap", "kube-system", "atlas-refactor-receipt"))
	if r := a.Status(context.Background()); r.State != Handoff {
		t.Fatal("first adoption accepted stale CRD proof", r)
	}
}

func TestFourNodeRuntimeDriftDoesNotChangeDurableAuthority(t *testing.T) {
	for _, damage := range []string{"missing-worker", "wrong-role", "unready-data", "wrong-worker-image", "missing-data-taint"} {
		t.Run(damage, func(t *testing.T) {
			a, s := developmentFixture(t)
			apply(t, a)
			n := len(s.effects)
			original := a.Runner
			a.Runner = runnerFunc(func(ctx context.Context, q Request) ([]byte, error) {
				if damage == "wrong-worker-image" && q.Tool == "docker" && q.Args[0] == "inspect" && strings.HasSuffix(q.Args[1], "-worker2") {
					return []byte("sha256:wrong true"), nil
				}
				b, e := original.Run(ctx, q)
				if e != nil {
					return b, e
				}
				if q.Tool == "kubectl" && len(q.Args) > 6 && q.Args[5] == "get" && q.Args[6] == "nodes" {
					var list Object
					_ = decode(b, &list)
					items := list["items"].([]any)
					switch damage {
					case "missing-worker":
						list["items"] = items[:3]
					case "wrong-role":
						items[1].(map[string]any)["metadata"].(map[string]any)["labels"].(map[string]any)["node-role.local/gateway"] = "false"
					case "unready-data":
						items[3].(map[string]any)["status"].(map[string]any)["conditions"] = []Object{{"type": "Ready", "status": "False"}}
					case "missing-data-taint":
						items[3].(map[string]any)["spec"].(map[string]any)["taints"] = []Object{}
					}
					return jsonBytes(list), nil
				}
				return b, nil
			})
			if r := a.Status(t.Context()); r.State != Adopted {
				t.Fatal("runtime drift changed handoff state", r)
			}
			if e := a.VerifyNodes(t.Context()); e == nil {
				t.Fatal("runtime verifier accepted damaged substrate")
			}
			apply(t, a) // Durable adoption is still a no-op, not runtime repair.
			if len(s.effects) != n {
				t.Fatal("mutated damaged substrate")
			}
		})
	}
}
