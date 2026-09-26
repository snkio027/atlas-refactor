package atlas

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditConfigurationBindsOnlyPrivateTestPaths(t *testing.T) {
	a, _ := fixture(t)
	b, e := a.auditedKindConfig()
	if e != nil {
		t.Fatal(e)
	}
	var c map[string]any
	if e = json.Unmarshal(b, &c); e != nil {
		t.Fatal(e)
	}
	if c["networking"].(map[string]any)["apiServerAddress"] != "127.0.0.1" {
		t.Fatal("API listener is not loopback")
	}
	node := c["nodes"].([]any)[0].(map[string]any)
	for _, v := range node["extraMounts"].([]any) {
		p := v.(map[string]any)["hostPath"].(string)
		if !strings.HasPrefix(p, filepath.Join(a.Root, ".state")+string(filepath.Separator)) {
			t.Fatal("audit path escaped private state")
		}
	}
	if strings.Contains(string(auditPolicy), "RequestResponse") || strings.Contains(string(auditPolicy), "level: Request\n") {
		t.Fatal("request bodies must not be audited")
	}
	info, e := os.Stat(filepath.Join(a.Root, ".state/audit"))
	if e != nil || info.Mode().Perm()&0077 != 0 {
		t.Fatal("audit directory is not owner-only")
	}
}
