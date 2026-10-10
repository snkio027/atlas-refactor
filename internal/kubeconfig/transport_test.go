package kubeconfig

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testProjection(t *testing.T) Projection {
	t.Helper()
	key, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	der, e := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if e != nil {
		t.Fatal(e)
	}
	keyDER, e := x509.MarshalECPrivateKey(key)
	if e != nil {
		t.Fatal(e)
	}
	enc := func(typ string, b []byte) string {
		return base64.StdEncoding.EncodeToString(pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: b}))
	}
	certPEM, keyPEM := enc("CERTIFICATE", der), enc("EC PRIVATE KEY", keyDER)
	v := map[string]any{"clusters": []any{map[string]any{"name": "kind-test", "cluster": map[string]any{"server": "https://127.0.0.1:6443", "certificate-authority-data": certPEM}}}, "contexts": []any{map[string]any{"name": "kind-test", "context": map[string]any{"cluster": "kind-test", "user": "kind-test"}}}, "users": []any{map[string]any{"name": "kind-test", "user": map[string]any{"client-certificate-data": certPEM, "client-key-data": keyPEM}}}}
	b, _ := json.Marshal(v)
	var p Projection
	if e = json.Unmarshal(b, &p); e != nil {
		t.Fatal(e)
	}
	return p
}
func TestTransportProjectionBoundaries(t *testing.T) {
	for _, mode := range []string{"valid", "context", "reference", "http", "remote", "no port", "query", "path", "insecure", "exec", "bad CA", "bad key", "ambiguous"} {
		t.Run(mode, func(t *testing.T) {
			p := testProjection(t)
			c, u := p.Clusters[0].Cluster, p.Users[0].User
			switch mode {
			case "context":
				p.Contexts[0].Name = "other"
			case "reference":
				p.Contexts[0].Context["user"] = "other"
			case "http":
				c["server"] = "http://127.0.0.1:6443"
			case "remote":
				c["server"] = "https://example.com:6443"
			case "no port":
				c["server"] = "https://127.0.0.1"
			case "query":
				c["server"] = "https://127.0.0.1:6443?x=y"
			case "path":
				c["server"] = "https://127.0.0.1:6443/extra"
			case "insecure":
				c["insecure-skip-tls-verify"] = true
			case "exec":
				u["exec"] = map[string]any{"command": "forbidden"}
			case "bad CA":
				c["certificate-authority-data"] = "broken"
			case "bad key":
				u["client-key-data"] = "broken"
			case "ambiguous":
				p.Users = append(p.Users, p.Users[0])
			}
			_, tr, e := Transport(p, "kind-test")
			if (e == nil) != (mode == "valid") {
				t.Fatalf("unexpected validation: %v", e)
			}
			if tr != nil {
				tr.CloseIdleConnections()
				if tr.Proxy != nil || tr.TLSClientConfig.InsecureSkipVerify {
					t.Fatal("unsafe transport")
				}
			}
		})
	}
}
func TestTransportRejectsUntrustedServer(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("untrusted API received request") }))
	defer s.Close()
	p := testProjection(t)
	p.Clusters[0].Cluster["server"] = s.URL
	_, tr, e := Transport(p, "kind-test")
	if e != nil {
		t.Fatal(e)
	}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, Timeout: time.Second}
	if res, e := client.Get(s.URL); e == nil {
		res.Body.Close()
		t.Fatal("untrusted API accepted")
	}
}
