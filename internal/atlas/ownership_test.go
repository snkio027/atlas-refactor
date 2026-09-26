package atlas

import (
	"context"
	"errors"
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
