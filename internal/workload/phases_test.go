package workload

import (
	"bytes"
	"strings"
	"testing"
)

// Reproduce the r3 race without assuming any cross-Application sync order.
// Every newly reachable source must be safe even if it reconciles first.
func TestPublicationPrerequisitesUnderAdversarialReconcileOrder(t *testing.T) {
	c, m := compileFixture(t)
	previous := c.Base
	phases := []string{"permissions", "project", "infrastructure", "consumer"}
	for _, phase := range phases {
		t.Run(phase, func(t *testing.T) {
			var a *Artifacts
			if phase == "consumer" {
				a = artifactFixture(c, m)
			}
			result, err := Compile(c, m, phase, a)
			if err != nil {
				t.Fatal(err)
			}
			again, err := Compile(c, m, phase, a)
			if err != nil || !bytes.Equal(JSON(result), JSON(again)) {
				t.Fatal("nondeterministic phase", err)
			}
			projects, err := objects(previous[ProjectsPath])
			if err != nil {
				t.Fatal(err)
			}
			platformProject, err := find(projects, "AppProject", "argocd", "platform-project")
			if err != nil {
				t.Fatal(err)
			}
			namespaceReady := previous["gitops/platform/projects/demo/resources.json"] != nil
			for _, owned := range result.Inventory.Resources {
				xs, err := objects(result.Files[owned.Path])
				if err != nil {
					t.Fatal(err)
				}
				for _, o := range xs {
					if identity(o) != owned.Identity {
						continue
					}
					// This includes the new Project Application's destination and the
					// pre-existing secrets-controller's independently watched RBAC path.
					usesNamespace := meta(o)["namespace"] == "demo" || o["kind"] == "Application" && val(o, "spec", "destination", "namespace") == "demo"
					if !usesNamespace {
						continue
					}
					permissionProbe := resource("v1", "ServiceAccount", "demo", "probe", nil)
					if err := permits(platformProject, permissionProbe); err != nil {
						t.Fatalf("%s visible before permission: %v", owned.Identity, err)
					}
					if owned.Owner != m.Intent.Project.AppName() && o["kind"] != "Application" && !namespaceReady {
						t.Fatalf("%s visible before Project namespace/baseline", owned.Identity)
					}
				}
			}
			switch phase {
			case "permissions":
				for path := range result.Inventory.Files {
					if path != ProjectsPath && !strings.HasPrefix(path, "platform/") {
						t.Fatalf("permission stage releases dependency: %s", path)
					}
				}
			case "project":
				for _, path := range []string{SealPath, SealExtraPath, EdgePath, PKIPath, MonitorPath, CredentialsPath, StoragePath, WorkloadAppsPath} {
					if !bytes.Equal(c.Base[path], result.Files[path]) {
						t.Fatalf("project stage releases platform consumer: %s", path)
					}
				}
				xs, _ := objects(result.Files["gitops/platform/projects/demo/resources.json"])
				for _, required := range [][2]string{{"Namespace", "demo"}, {"ResourceQuota", "project-budget"}, {"NetworkPolicy", "default-deny"}, {"ServiceAccount", "web-api"}} {
					found := false
					for _, o := range xs {
						found = found || o["kind"] == required[0] && meta(o)["name"] == required[1]
					}
					if !found {
						t.Fatal("missing Project prerequisite", required)
					}
				}
			case "infrastructure":
				for _, path := range []string{CredentialsPath, StoragePath, WorkloadAppsPath} {
					if !bytes.Equal(c.Base[path], result.Files[path]) {
						t.Fatalf("infrastructure stage releases consumer: %s", path)
					}
				}
			}
			// Existing identities keep their one owner in all stages.
			oldInv, err := InventoryOf(previous, c.ResourceModel)
			if err != nil {
				t.Fatal(err)
			}
			owners := map[string]string{}
			for _, v := range result.Inventory.Resources {
				owners[v.Identity] = v.Owner
			}
			for _, v := range oldInv {
				if owners[v.Identity] != v.Owner {
					t.Fatal("lost identity or owner", v.Identity)
				}
			}
			previous = result.Files
		})
	}
}
