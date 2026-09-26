package atlas

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestInterruptedHandoffOnlyResumesReceipt(t *testing.T) {
	a, s := fixture(t)
	s.converge = false
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.cancelAfterRoot = cancel
	if e := a.Apply(ctx, a.Config.Cluster, true); e == nil {
		t.Fatal("expected interruption")
	}
	if r := a.Status(context.Background()); r.State != Handoff {
		t.Fatalf("%+v", r)
	}
	n := len(s.effects)
	s.gitops()
	apply(t, a)
	if !reflect.DeepEqual(s.effects[n:], []string{"create:atlas-refactor-receipt"}) {
		t.Fatalf("unexpected resumed effects: %v", s.effects[n:])
	}
}

func TestExistingForeignAndLostClustersAreNeverCreated(t *testing.T) {
	t.Run("foreign", func(t *testing.T) {
		a, s := fixture(t)
		s.exists = true
		if e := a.Apply(context.Background(), a.Config.Cluster, true); e == nil {
			t.Fatal("foreign cluster adopted")
		}
		if len(s.effects) != 0 {
			t.Fatal("foreign cluster mutated")
		}
	})
	t.Run("lost", func(t *testing.T) {
		a, s := fixture(t)
		apply(t, a)
		s.exists = false
		n := len(s.effects)
		if e := a.Apply(context.Background(), a.Config.Cluster, true); e == nil {
			t.Fatal("lost cluster recreated")
		}
		if len(s.effects) != n {
			t.Fatal("lost cluster mutated")
		}
	})
}

func TestUnknownHealthAndActiveOperationAreNotReady(t *testing.T) {
	for _, mode := range []string{"unknown-health", "running", "pending-operation"} {
		t.Run(mode, func(t *testing.T) {
			a, s := fixture(t)
			apply(t, a)
			root := s.objects[key("Application", "argocd", "atlas-refactor-root")]
			switch mode {
			case "unknown-health":
				root["status"].(Object)["health"] = Object{"status": "FutureState"}
			case "running":
				root["status"].(Object)["operationState"] = Object{"phase": "Running"}
			case "pending-operation":
				root["operation"] = Object{"sync": Object{}}
			}
			if r := a.Status(context.Background()); r.State != Degraded {
				t.Fatalf("%+v", r)
			}
		})
	}
}

func TestLoadRejectsBadLocksAndConfiguration(t *testing.T) {
	a, _ := fixture(t)
	configPath := filepath.Join(a.Root, "config.json")
	if e := os.WriteFile(configPath, jsonBytes(a.Config), 0600); e != nil {
		t.Fatal(e)
	}
	c, l, e := Load(a.Root, "config.json")
	if e != nil || c.Cluster != a.Config.Cluster || l.Go != a.Lock.Go {
		t.Fatalf("valid load: %v", e)
	}
	invalid := []func(*Lock){func(l *Lock) { l.Go = "latest" }, func(l *Lock) { l.NodeImage = "kindest/node:latest" }, func(l *Lock) { l.ChartSHA256 = "wrong" }, func(l *Lock) { delete(l.Assets, "assets/argocd-cm.yaml") }}
	for _, change := range invalid {
		var lock Lock
		if e := strictJSON(jsonBytes(a.Lock), &lock); e != nil {
			t.Fatal(e)
		}
		change(&lock)
		if lock.Validate() == nil {
			t.Fatal("invalid lock accepted")
		}
	}
	for _, raw := range []string{`null`, `{"schema":1}`, strings.Replace(string(jsonBytes(a.Config)), `"schema": 1`, `"schema": 1, "extra": true`, 1)} {
		if e := os.WriteFile(configPath, []byte(raw), 0600); e != nil {
			t.Fatal(e)
		}
		if _, _, e = Load(a.Root, "config.json"); e == nil {
			t.Fatalf("accepted invalid config: %s", raw)
		}
	}
}

func TestStaleGeneratedTreeBlocksClusterCreation(t *testing.T) {
	a, s := fixture(t)
	p := filepath.Join(a.Root, a.Config.GitOpsPath, "root/platform.json")
	if e := os.WriteFile(p, []byte("changed"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := a.Apply(context.Background(), a.Config.Cluster, true); e == nil {
		t.Fatal("stale source accepted")
	}
	if len(s.effects) != 0 {
		t.Fatal("mutation before Git source validation")
	}
}

func TestPlaintextSecretNotRendered(t *testing.T) {
	a, _ := fixture(t)
	files, e := a.Render(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	for path, data := range files {
		var top struct {
			Kind string `json:"kind"`
		}
		_ = decode(data, &top)
		if strings.Contains("\n"+string(data), "\nkind: Secret\n") || top.Kind == "Secret" {
			t.Fatalf("Secret object in %s", path)
		}
	}
}
