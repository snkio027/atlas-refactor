package ot1

import (
	"atlas-refactor/internal/observation"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func proofSnapshot(s Snapshot) Snapshot {
	s.Envelope.ProofVersion = observation.ProofVersion
	clone := func(in []observation.Object) []observation.Object {
		out := []observation.Object{}
		for _, o := range in {
			out = append(out, observation.Clone(o))
		}
		return out
	}
	s.Envelope.ClosingRaw = clone(s.Envelope.Raw)
	s.ClosingApplications = clone(s.Applications)
	s.ClosingProjects = clone(s.Projects)
	s.ClosingNodes = clone(s.Nodes)
	return s
}
func revisionFixture(t *testing.T) (Plan, map[string]observation.Object, []Snapshot) {
	p, d := syntheticPlan(t)
	p.EvidenceModel = EvidenceModel
	base := strings.Repeat("e", 40)
	p.BaselineRevision = base
	p.Phases[0].Revision = base
	dir := "gitops/test/leaf"
	hashes := map[string]string{dir + "/kustomization.yaml": strings.Repeat("1", 64), dir + "/rendered.yaml": strings.Repeat("2", 64)}
	p.Revisions = map[string]Revision{"baseline": {Commit: base, FilesSHA256: hashes}, "detached-mixed": {Commit: p.Phases[1].Revision, FilesSHA256: hashes}}
	for i := range p.Phases {
		for j := range p.Phases[i].Applications {
			a := &p.Phases[i].Applications[j]
			a.Revision = p.Phases[i].Revision
			if destination(a.Name) == "" {
				a.Spec = observation.Object{"source": observation.Object{"path": dir, "repoURL": p.Repository, "targetRevision": p.Branch}, "project": "platform-project"}
			}
		}
	}
	snapshots := syntheticSnapshots(t, p, d)
	for i := range snapshots {
		snapshots[i] = proofSnapshot(snapshots[i])
	}
	return p, d, snapshots
}
func TestTransitionalRevisionEquivalence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Plan, Snapshot)
		want   bool
	}{
		{"unchanged prior source", func(_ *Plan, _ Snapshot) {}, true},
		{"changed source", func(p *Plan, _ Snapshot) {
			v := p.Revisions["detached-mixed"]
			v.FilesSHA256 = map[string]string{"gitops/test/leaf/kustomization.yaml": strings.Repeat("1", 64), "gitops/test/leaf/rendered.yaml": strings.Repeat("3", 64)}
			p.Revisions["detached-mixed"] = v
		}, false},
		{"unrecognized older revision", func(_ *Plan, s Snapshot) {
			observation.Map(observation.At(rawIndex(s.Applications)[AppRef("argocd-self").Key()], "status", "sync"))["revision"] = strings.Repeat("a", 40)
		}, false},
		{"new uid", func(_ *Plan, s Snapshot) {
			observation.Map(rawIndex(s.Applications)[AppRef("argocd-self").Key()]["metadata"])["uid"] = "new"
		}, false},
		{"unhealthy", func(_ *Plan, s Snapshot) {
			observation.Map(observation.At(rawIndex(s.Applications)[AppRef("argocd-self").Key()], "status", "health"))["status"] = "Degraded"
		}, false},
		{"platform-control stays exact", func(p *Plan, s Snapshot) {
			observation.Map(observation.At(rawIndex(s.Applications)[AppRef("platform-control").Key()], "status", "sync"))["revision"] = p.BaselineRevision
		}, false},
		{"Atlas Gate stays exact", func(p *Plan, _ Snapshot) { p.Phases[1].Stage.AtlasGate = true }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, d, ss := revisionFixture(t)
			s := ss[1]
			app := rawIndex(s.Applications)[AppRef("argocd-self").Key()]
			observation.Map(observation.At(app, "status", "sync"))["revision"] = p.BaselineRevision
			tc.mutate(&p, s)
			s.PlanSHA256 = observation.Digest(p)
			s = proofSnapshot(s)
			got := Assess(p, 1, s, &ss[0], &ss[0], d, nil)
			if (got.Ownership == "VERIFIED") != tc.want {
				t.Fatal(got)
			}
			ready, e := applicationProgress(p, 1, s.Applications, &ss[0], "")
			if tc.want && (!ready || e != nil) {
				t.Fatal(ready, e)
			}
			if !tc.want && ready && e == nil {
				t.Fatal("progress incorrectly ready")
			}
		})
	}
}
func TestEquivalenceDoesNotExpandToOlderGitEpoch(t *testing.T) {
	p, _, ss := revisionFixture(t)
	app := rawIndex(ss[2].Applications)[AppRef("argocd-self").Key()]
	observation.Map(observation.At(app, "status", "sync"))["revision"] = p.BaselineRevision
	var expect observation.ExpectedApplication
	for _, a := range p.Phases[2].Applications {
		if a.Name == "argocd-self" {
			expect = a
		}
	}
	if transitionExpectation(p, 2, expect, app, rawIndex(ss[1].Applications)[AppRef(expect.Name).Key()]).Revision != p.BaselineRevision {
		t.Fatal("same detached epoch should keep the preceding Git revision")
	}
	// Once another Git revision is published, the older baseline is no longer eligible.
	p.Phases[2].Revision = strings.Repeat("f", 40)
	expect.Revision = p.Phases[2].Revision
	if transitionExpectation(p, 2, expect, app, app).Revision != expect.Revision {
		t.Fatal("two revisions old admitted")
	}
}
func TestStoredSemanticProofRejectsTamperedClosingRead(t *testing.T) {
	p, d, ss := revisionFixture(t)
	s := ss[1]
	observation.Map(s.ClosingApplications[0]["metadata"])["uid"] = "other"
	if Assess(p, 1, s, &ss[0], &ss[0], d, nil).Ownership == "VERIFIED" {
		t.Fatal("tampered inventory admitted")
	}
}
func TestLocalSourceClosureRejectsRemoteOrParentInput(t *testing.T) {
	for _, resource := range []string{"rendered.yaml", "../outside.yaml", "https://invalid/remote", "subdir"} {
		t.Run(resource, func(t *testing.T) {
			p, _, _ := revisionFixture(t)
			repo := privateTemp(t)
			dir := filepath.Join(repo, "gitops/test/leaf")
			if e := os.MkdirAll(dir, 0700); e != nil {
				t.Fatal(e)
			}
			k := observation.Object{"apiVersion": "kustomize.config.k8s.io/v1beta1", "kind": "Kustomization", "resources": []any{resource}}
			b := observation.Bytes(k)
			render := []byte("{}\n")
			for name, data := range map[string][]byte{"kustomization.yaml": b, "rendered.yaml": render} {
				if e := os.WriteFile(filepath.Join(dir, name), data, 0600); e != nil {
					t.Fatal(e)
				}
			}
			if _, e := git(context.Background(), repo, "init"); e != nil {
				t.Fatal(e)
			}
			sha, e := commitLocal(context.Background(), repo, "test local source closure")
			if e != nil {
				t.Fatal(e)
			}
			phase := p.Phases[0]
			phase.Revision = sha
			// Keep one unrelated app; other fixture sources are intentionally synthetic.
			phase.Applications = phase.Applications[:1]
			phase.Applications[0].Revision = sha
			p.Phases = []Phase{phase}
			p.Revisions = map[string]Revision{"baseline": {Commit: sha, FilesSHA256: map[string]string{"gitops/test/leaf/kustomization.yaml": observation.SHA(b), "gitops/test/leaf/rendered.yaml": observation.SHA(render)}}}
			e = validateSourceClosures(context.Background(), repo, p)
			if (e == nil) != (resource == "rendered.yaml") {
				t.Fatal("source closure validation", e)
			}
		})
	}
}
