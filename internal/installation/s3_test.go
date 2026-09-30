package installation

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// Fixed AWS-published S3 vectors independently validate the signer; these are
// documentation example credentials, never actual AWS account material.
// https://docs.aws.amazon.com/AmazonS3/latest/developerguide/sig-v4-header-based-auth.html
func TestS3PublishedSigningVectors(t *testing.T) {
	c := Credentials{AccessKey: "AKIAIOSFODNN7EXAMPLE", SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"}
	now := time.Date(2013, 5, 24, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct{ query, want string }{{"lifecycle=", "fea454ca298b7da1c68078a5d1bdbfbbe0d65c699e0f91ac7a200a0136783543"}, {"max-keys=2&prefix=J", "34b48302e7b5fa45bde8084f4b7868a86f0a534bc59db6670ed5711ef69dc6f7"}} {
		req, e := http.NewRequest("GET", "https://examplebucket.s3.amazonaws.com/?"+test.query, nil)
		if e != nil {
			t.Fatal(e)
		}
		signed(req, nil, c, now, false)
		if !strings.HasSuffix(req.Header.Get("Authorization"), "Signature="+test.want) {
			t.Fatalf("official vector differs for %s", test.query)
		}
	}
}
func TestExternalBackupDoesNotBecomeLocalDirectory(t *testing.T) {
	c := testConfig(t)
	if e := privateDir(c.StateDirectory); e != nil {
		t.Fatal(e)
	}
	c.BackupIsolation = "external"
	w := Workflow{Config: c}
	if w.checkBackupLocation() == nil {
		t.Fatal("same filesystem claimed external backup")
	}
}
