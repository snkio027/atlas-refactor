package atlas

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestDurableHandoffDoesNotQueryBranchHeadOrLeafRollout(t *testing.T) {
	for _, change := range []string{"unrelated commit", "unhealthy workload", "seed operation running"} {
		t.Run(change, func(t *testing.T) {
			a, s := developmentFixture(t)
			apply(t, a)
			n := len(s.effects)
			a.resolvedCommit = ""
			switch change {
			case "unhealthy workload":
				s.objects[key("Application", "argocd", "web-smoke")]["status"].(Object)["health"] = Object{"status": "Degraded"}
			case "seed operation running":
				self := s.objects[key("Application", "argocd", "argocd-self")]
				self["status"].(Object)["operationState"] = Object{"phase": "Running"}
				self["operation"] = Object{"sync": Object{"revision": strings.Repeat("b", 40)}}
			}
			original := a.Runner
			a.Runner = runnerFunc(func(ctx context.Context, q Request) ([]byte, error) {
				if q.Tool == "git" && q.Args[0] == "ls-remote" {
					return nil, errors.New("branch HEAD not available during authority check")
				}
				if q.Tool == "kubectl" && strings.Contains(strings.Join(q.Args, " "), "get application web-smoke") {
					return nil, errors.New("workload rollout is not Bootstrap authority")
				}
				return original.Run(ctx, q)
			})
			if r := a.Status(t.Context()); r.State != Adopted {
				t.Fatal(r)
			}
			// Repository validation still applies to an explicit apply; simulate the
			// later reviewed Git HEAD while leaving unchanged live leaves at the old SHA.
			a.Runner = runnerFunc(func(ctx context.Context, q Request) ([]byte, error) {
				if q.Tool == "git" && q.Args[0] == "ls-remote" {
					return []byte(strings.Repeat("b", 40) + "\trefs/heads/main\n"), nil
				}
				if q.Tool == "git" && q.Args[0] == "rev-parse" {
					return []byte(strings.Repeat("b", 40)), nil
				}
				return original.Run(ctx, q)
			})
			apply(t, a)
			if len(s.effects) != n {
				t.Fatal("post-adoption rollout caused Bootstrap writes", s.effects[n:])
			}
		})
	}
}

func TestSchema3AuthorityDamageStillFailsClosed(t *testing.T) {
	for _, damage := range []string{"root UID", "root spec", "receipt", "latch", "identity", "signal", "self missing", "seed CRD SSA", "unavailable read"} {
		t.Run(damage, func(t *testing.T) {
			a, s := developmentFixture(t)
			apply(t, a)
			n := len(s.effects)
			switch damage {
			case "root UID":
				s.objects[key("Application", "argocd", "atlas-refactor-root")]["metadata"].(Object)["uid"] = "replacement"
			case "root spec":
				s.objects[key("Application", "argocd", "atlas-refactor-root")]["spec"] = Object{"project": "foreign"}
			case "receipt", "latch", "identity":
				name := map[string]string{"receipt": "atlas-refactor-receipt", "latch": "atlas-refactor-handoff", "identity": "atlas-refactor-identity"}[damage]
				s.objects[key("ConfigMap", "kube-system", name)]["immutable"] = false
			case "signal":
				s.objects[key("ConfigMap", "argocd", "atlas-refactor-adoption-signal")]["immutable"] = false
			case "self missing":
				delete(s.objects, key("Application", "argocd", "argocd-self"))
			case "seed CRD SSA":
				s.objects[key("CustomResourceDefinition", "", "applications.argoproj.io")]["metadata"].(Object)["managedFields"] = []Object{{"manager": "foreign", "operation": "Apply"}}
			case "unavailable read":
				s.failRead = "atlas-refactor-handoff"
			}
			r := a.Status(t.Context())
			if r.State == Adopted || r.State == Fresh || r.State == Handoff {
				t.Fatal("invalid authority accepted", r)
			}
			if e := a.Apply(t.Context(), a.Config.Cluster, true); e == nil {
				t.Fatal("invalid authority allowed apply")
			}
			if len(s.effects) != n {
				t.Fatal("authority damage caused mutation")
			}
		})
	}
}

func TestFirstDevelopmentHandoffStillNeedsExactLeafRevision(t *testing.T) {
	a, s := developmentFixture(t)
	s.failCreate = "atlas-refactor-receipt"
	if e := a.Apply(t.Context(), a.Config.Cluster, true); e == nil {
		t.Fatal("missing injected interruption")
	}
	s.failCreate = ""
	n := len(s.effects)
	s.objects[key("Application", "argocd", "envoy-gateway")]["status"].(Object)["sync"].(Object)["revision"] = strings.Repeat("b", 40)
	if r := a.Status(t.Context()); r.State != Handoff {
		t.Fatal(r)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	if e := a.Apply(ctx, a.Config.Cluster, true); e == nil {
		t.Fatal("first Receipt accepted stale leaf revision")
	}
	if len(s.effects) != n || s.objects[key("ConfigMap", "kube-system", "atlas-refactor-receipt")] != nil {
		t.Fatal("first handoff wrote without exact proof")
	}
}

func TestDevelopmentWorkflowChecksRolloutSeparately(t *testing.T) {
	for _, damage := range []string{"none", "leaf revision", "leaf health", "leaf active", "CRD sync proof"} {
		t.Run(damage, func(t *testing.T) {
			a, s := developmentFixture(t)
			apply(t, a)
			n := len(s.effects)
			leaf := s.objects[key("Application", "argocd", "web-smoke")]
			switch damage {
			case "leaf revision":
				leaf["status"].(Object)["sync"].(Object)["revision"] = strings.Repeat("b", 40)
			case "leaf health":
				leaf["status"].(Object)["health"] = Object{"status": "Degraded"}
			case "leaf active":
				leaf["operation"] = Object{"sync": Object{}}
			case "CRD sync proof":
				s.objects[key("Application", "argocd", "argocd-self")]["status"].(Object)["operationState"] = Object{}
			}
			if report := a.Status(t.Context()); report.State != Adopted {
				t.Fatal(report)
			}
			if e := a.VerifyDevelopmentRollout(t.Context()); (e == nil) != (damage == "none") {
				t.Fatal("independent rollout check", damage, e)
			}
			if len(s.effects) != n {
				t.Fatal("rollout verifier wrote live state")
			}
		})
	}
}
