package developmentprofile

import "testing"

func TestExactOT1Binding(t *testing.T) {
	if e := Validate(3, OT1Cluster, Repository, OT1Revision, RootPath); e != nil {
		t.Fatal(e)
	}
	for _, c := range []struct {
		schema            int
		cluster, revision string
	}{{1, OT1Cluster, OT1Revision}, {2, OT1Cluster, OT1Revision}, {3, "atlas-refactor-test-dev02", OT1Revision}, {3, OT1Cluster, DevelopmentRevision}, {3, "atlas-refactor-test-ot2", OT1Revision}, {3, OT1Cluster, "main"}} {
		if Validate(c.schema, c.cluster, Repository, c.revision, RootPath) == nil {
			t.Fatal("unreviewed binding accepted", c)
		}
	}
	for _, cluster := range []string{"atlas-refactor-test-dev01", "atlas-refactor-test-dev02"} {
		if e := Validate(3, cluster, Repository, DevelopmentRevision, RootPath); e != nil {
			t.Fatal(e)
		}
	}
	normal, _ := Lookup(DevelopmentRevision)
	probe, _ := Lookup(OT1Revision)
	if normal.HTTPPort != 8080 || normal.HTTPSPort != 8443 || probe.HTTPPort != 18080 || probe.HTTPSPort != 18443 || normal.SnapshotSHA256 == probe.SnapshotSHA256 {
		t.Fatal("profile collision")
	}
}
