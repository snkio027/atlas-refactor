package atlas

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSelfSyncCannotSkipMatchingSeed(t *testing.T) {
	a, _ := fixture(t)
	self := a.application("argocd-self", "platform-project", a.Config.GitOpsPath+"/platform/argocd", "0")
	options := self["spec"].(Object)["syncPolicy"].(Object)["syncOptions"]
	if !reflect.DeepEqual(options, []string{"ServerSideApply=true"}) {
		t.Fatal("first self-sync can skip matching unowned Seed", options)
	}
}

func TestHealthySignalCannotCommitReceiptWithoutSeedOwnership(t *testing.T) {
	for _, damage := range []string{"missing", "tracking", "manager", "wrong-revision", "child-revision"} {
		t.Run(damage, func(t *testing.T) {
			a, sim := fixture(t)
			sim.failCreate = "atlas-refactor-receipt"
			if e := a.Apply(context.Background(), a.Config.Cluster, true); e == nil {
				t.Fatal("receipt failure not injected")
			}
			sim.failCreate = ""
			n := len(sim.effects)
			seed := sim.objects[key("Deployment", "argocd", "atlas-refactor-argocd-server")]
			switch damage {
			case "missing":
				delete(sim.objects, key("Deployment", "argocd", "atlas-refactor-argocd-server"))
			case "tracking":
				delete(seed["metadata"].(Object), "annotations")
			case "manager":
				seed["metadata"].(Object)["managedFields"] = []Object{{"manager": "atlas-refactor-bootstrap", "operation": "Apply"}}
			case "wrong-revision":
				sim.objects[key("Application", "argocd", "argocd-self")]["status"].(Object)["sync"].(Object)["revision"] = strings.Repeat("b", 40)
			case "child-revision":
				sim.objects[key("Application", "argocd", "project-bootstrap")]["status"].(Object)["sync"].(Object)["revision"] = strings.Repeat("b", 40)
			}
			if r := a.Status(context.Background()); r.State != Handoff {
				t.Fatalf("false adoption: %+v", r)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
			defer cancel()
			if e := a.Apply(ctx, a.Config.Cluster, true); e == nil {
				t.Fatal("unproven ownership committed Receipt")
			}
			if len(sim.effects) != n || sim.objects[key("ConfigMap", "kube-system", "atlas-refactor-receipt")] != nil {
				t.Fatal("write after incomplete ownership", sim.effects[n:])
			}
		})
	}
}

func TestReceiptCannotMaskLostSeedOwnership(t *testing.T) {
	a, sim := fixture(t)
	apply(t, a)
	n := len(sim.effects)
	delete(sim.objects[key("Deployment", "argocd", "atlas-refactor-argocd-server")]["metadata"].(Object), "managedFields")
	if r := a.Status(context.Background()); r.State != Degraded {
		t.Fatalf("ownership loss hidden by receipt: %+v", r)
	}
	if e := a.Apply(context.Background(), a.Config.Cluster, true); e == nil {
		t.Fatal("degraded ownership accepted")
	}
	if len(sim.effects) != n {
		t.Fatal("Seed authority restored")
	}
}

func TestExternalRootCannotBeTrackedByAnotherApplication(t *testing.T) {
	a, sim := fixture(t)
	apply(t, a)
	n := len(sim.effects)
	sim.objects[key("Application", "argocd", "atlas-refactor-root")]["metadata"].(Object)["annotations"] = map[string]string{"argocd.argoproj.io/tracking-id": "foreign:argoproj.io/Application:argocd/atlas-refactor-root"}
	if r := a.Status(context.Background()); r.State != Drifted {
		t.Fatalf("External Root parent accepted: %+v", r)
	}
	if len(sim.effects) != n {
		t.Fatal("Root repaired")
	}
}

func TestOwnershipProjectionUnavailableFailsClosed(t *testing.T) {
	for _, failure := range []string{"source", "projection", "read"} {
		t.Run(failure, func(t *testing.T) {
			a, sim := fixture(t)
			apply(t, a)
			n := len(sim.effects)
			a.resolvedCommit = ""
			a.Runner = runnerFunc(func(ctx context.Context, q Request) ([]byte, error) {
				joined := strings.Join(q.Args, " ")
				if (failure == "source" && q.Tool == "git" && q.Args[0] == "ls-remote") ||
					(failure == "projection" && q.Tool == "kubectl" && strings.Contains(joined, "--dry-run=client")) ||
					(failure == "read" && q.Tool == "kubectl" && strings.Contains(joined, "get -f -")) {
					return nil, errors.New("unavailable")
				}
				return sim.Run(ctx, q)
			})
			if r := a.Status(context.Background()); r.State != Unavailable {
				t.Fatalf("unknown ownership accepted: %+v", r)
			}
			if len(sim.effects) != n {
				t.Fatal("write during unavailable ownership")
			}
		})
	}
}

func TestEphemeralHooksAreNotDurableOwnershipEvidence(t *testing.T) {
	a, sim := fixture(t)
	apply(t, a)
	a.Runner = runnerFunc(func(ctx context.Context, q Request) ([]byte, error) {
		if q.Tool == "kubectl" && strings.Contains(strings.Join(q.Args, " "), "--dry-run=client") {
			hook := object("Job", "argocd", "finished-hook")
			hook["apiVersion"] = "batch/v1"
			hook["metadata"].(Object)["annotations"] = map[string]string{"helm.sh/hook": "pre-install,pre-upgrade"}
			return jsonBytes(Object{"kind": "List", "items": append(sim.seedObjects(), hook)}), nil
		}
		return sim.Run(ctx, q)
	})
	if r := a.Status(context.Background()); r.State != Adopted {
		t.Fatalf("deleted hook blocked durable ownership: %+v", r)
	}
}

func TestSeedProjectionRejectsMalformedInventory(t *testing.T) {
	for _, data := range []string{`{}`, `{"kind":"List","items":[{"apiVersion":"v1","kind":"Secret","metadata":{"name":"secret"}}]}`, `{"kind":"List","items":[{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"x"}},{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"x"}}]}`} {
		if _, e := seedList([]byte(data)); e == nil {
			t.Fatal("malformed inventory accepted", data)
		}
	}
}

func TestClientProjectionSupportsJSONObjectStream(t *testing.T) {
	one := jsonBytes(object("ConfigMap", "argocd", "one"))
	two := jsonBytes(object("ConfigMap", "argocd", "two"))
	stream := append(append([]byte{}, one...), two...)
	objects, e := seedProjection(stream)
	if e != nil || len(objects) != 2 {
		t.Fatal("kubectl client JSON stream rejected", e)
	}
	for _, bad := range [][]byte{append(append([]byte{}, one...), one...), append(append([]byte{}, one...), []byte("garbage")...), append([]byte(`{"kind":"List","items":[]}`), one...)} {
		if _, e := seedProjection(bad); e == nil {
			t.Fatal("bad projection accepted")
		}
	}
}

func TestCRDReconciliationRequiresEvidenceBeforeAndAfterReceipt(t *testing.T) {
	for _, adopted := range []bool{false, true} {
		for _, damage := range []string{"absent", "foreign-tracking", "bootstrap-only", "metadata-only", "status-manager", "not-applied", "inventory-missing", "inventory-duplicate", "inventory-version", "inventory-outofsync", "sync-missing", "sync-stale", "sync-failed", "sync-resource-failed", "sync-resource-namespace"} {
			t.Run(fmt.Sprintf("adopted=%t/%s", adopted, damage), func(t *testing.T) {
				a, sim := fixture(t)
				apply(t, a)
				if !adopted {
					delete(sim.objects, key("ConfigMap", "kube-system", "atlas-refactor-receipt"))
				}
				crd := sim.objects[key("CustomResourceDefinition", "", "applications.argoproj.io")]
				meta := crd["metadata"].(Object)
				fields := meta["managedFields"].([]Object)
				status := sim.objects[key("Application", "argocd", "argocd-self")]["status"].(Object)
				operation := status["operationState"].(Object)
				result := operation["syncResult"].(Object)
				inventory := status["resources"].([]Object)
				switch damage {
				case "absent":
					delete(sim.objects, key("CustomResourceDefinition", "", "applications.argoproj.io"))
				case "foreign-tracking":
					meta["annotations"] = map[string]string{"argocd.argoproj.io/tracking-id": "other:apiextensions.k8s.io/CustomResourceDefinition:argocd/applications.argoproj.io"}
				case "bootstrap-only":
					fields[0]["manager"] = "atlas-refactor-bootstrap"
				case "metadata-only":
					fields[0]["fieldsV1"] = Object{"f:metadata": Object{}}
				case "status-manager":
					fields[0]["subresource"] = "status"
				case "not-applied":
					fields[0]["operation"] = "Update"
				case "inventory-missing":
					status["resources"] = inventory[:1]
				case "inventory-duplicate":
					status["resources"] = append(inventory, inventory[1])
				case "inventory-version":
					inventory[1]["version"] = "v1beta1"
				case "inventory-outofsync":
					inventory[1]["status"] = "OutOfSync"
				case "sync-missing":
					delete(operation, "syncResult")
				case "sync-stale":
					result["revision"] = strings.Repeat("b", 40)
				case "sync-failed":
					operation["phase"] = "Failed"
				case "sync-resource-failed":
					result["resources"].([]Object)[0]["status"] = "SyncFailed"
				case "sync-resource-namespace":
					result["resources"].([]Object)[0]["namespace"] = ""
				}
				n := len(sim.effects)
				want := Handoff
				if adopted {
					want = Degraded
				} else if damage == "absent" {
					want = Drifted
				}
				if r := a.Status(context.Background()); r.State != want {
					t.Fatalf("unproven CRD reconciliation accepted: %+v, want %s", r, want)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
				defer cancel()
				if e := a.Apply(ctx, a.Config.Cluster, true); e == nil {
					t.Fatal("damaged CRD reconciliation accepted")
				}
				if len(sim.effects) != n {
					t.Fatal("unproven ownership permitted a mutation")
				}
			})
		}
	}
}

func TestClusterScopedTrackingUsesApplicationDestination(t *testing.T) {
	a, sim := fixture(t)
	apply(t, a)
	role := sim.objects[key("ClusterRole", "", "atlas-refactor-argocd-application-controller")]
	annotations := role["metadata"].(Object)["annotations"].(map[string]string)
	annotations["argocd.argoproj.io/tracking-id"] = "argocd-self:rbac.authorization.k8s.io/ClusterRole:argocd/atlas-refactor-argocd-application-controller"
	if r := a.Status(context.Background()); r.State != Adopted {
		t.Fatalf("Argo's actual cluster-scoped tracking rejected: %+v", r)
	}
	for _, wrong := range []string{"", "argocd-self:rbac.authorization.k8s.io/ClusterRole:/atlas-refactor-argocd-application-controller", "argocd-self:rbac.authorization.k8s.io/ClusterRole:other/atlas-refactor-argocd-application-controller"} {
		annotations["argocd.argoproj.io/tracking-id"] = wrong
		if r := a.Status(context.Background()); r.State != Degraded {
			t.Fatalf("wrong cluster tracking accepted: %+v", r)
		}
	}
}
