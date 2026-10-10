package platform

import "testing"

func TestLockedArtifactSelectionRejectsMissingAndAmbiguousVersions(t *testing.T) {
	for _, version := range []string{"10.3.3", "10.10.2"} {
		path := "vendor/charts/argo-cd-" + version + ".tgz"
		p := &Project{Lock: Lock{Artifacts: map[string]Artifact{path: {}}}}
		got, err := p.artifactPath("vendor/charts/argo-cd-", ".tgz")
		if err != nil || got != path {
			t.Fatalf("locked version %s: %s / %v", version, got, err)
		}
		p.Lock.Artifacts["vendor/charts/argo-cd-99.0.0.tgz"] = Artifact{}
		if _, err = p.artifactPath("vendor/charts/argo-cd-", ".tgz"); err == nil {
			t.Fatal("ambiguous lock silently selected a version")
		}
	}
	for _, path := range []string{
		"vendor/charts/argo-cd-latest.tgz",
		"vendor/charts/argo-cd-10.10.2-rc1.tgz",
		"vendor/charts/argo-cd-../10.10.2.tgz",
		"vendor/charts/another-10.10.2.tgz",
	} {
		p := &Project{Lock: Lock{Artifacts: map[string]Artifact{path: {}}}}
		if _, err := p.artifactPath("vendor/charts/argo-cd-", ".tgz"); err == nil {
			t.Fatalf("accepted missing reviewed chart: %s", path)
		}
	}
}
