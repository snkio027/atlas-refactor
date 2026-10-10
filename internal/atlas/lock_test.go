package atlas

import (
	"os"
	"testing"
)

func TestChartLockRequiresVersionedLocalArgoArchive(t *testing.T) {
	b, err := os.ReadFile("../../versions.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	var lock Lock
	if err := strictJSON(b, &lock); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		lock.Chart,
		"vendor/charts/argo-cd-10.3.3.tgz", // Historical chart lock paths remain valid.
	} {
		candidate := lock
		candidate.Chart = path
		if err := candidate.Validate(); err != nil {
			t.Fatalf("versioned chart %q: %v", path, err)
		}
	}
	for _, path := range []string{
		"vendor/charts/argo-cd-latest.tgz",
		"vendor/charts/argo-cd-10.10.tgz",
		"vendor/charts/argo-cd-10.10.2-rc1.tgz",
		"vendor/charts/../argo-cd-10.10.2.tgz",
		"/tmp/argo-cd-10.10.2.tgz",
		"https://example.test/argo-cd-10.10.2.tgz",
		"vendor/charts/other-10.10.2.tgz",
	} {
		candidate := lock
		candidate.Chart = path
		if candidate.Validate() == nil {
			t.Fatalf("unsafe or mutable chart accepted: %q", path)
		}
	}
	lock.ChartSHA256 = ""
	if lock.Validate() == nil {
		t.Fatal("chart without checksum accepted")
	}
}
