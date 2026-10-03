package workloadrun

import (
	"atlas-refactor/internal/workload"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

func (w *Workflow) httpsClient(host string) (*http.Client, error) {
	cert, e := regular(filepath.Join(w.Install.Config.StateDirectory, "development-ca.crt"), true)
	if e != nil {
		return nil, e
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(cert) {
		return nil, errors.New("invalid local CA")
	}
	tr := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, ServerName: host, MinVersion: tls.VersionTLS12}, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{Timeout: 5 * time.Second}
		return d.DialContext(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", w.Install.Config.HTTPSPort))
	}}
	return &http.Client{Transport: tr, Timeout: 45 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, nil
}
func (w *Workflow) readyPod(ctx context.Context, v workload.Workload) (Object, error) {
	b, e := w.kube(ctx, "get", "pods", "-n", v.Project, "-l", "atlas.io/project="+v.Project+",atlas.io/workload="+v.Name, "-o", "json")
	if e != nil {
		return nil, e
	}
	var list Object
	if e = json.Unmarshal(b, &list); e != nil {
		return nil, e
	}
	var chosen Object
	count := 0
	for _, raw := range array(list["items"]) {
		o := mapping(raw)
		if at(o, "metadata", "deletionTimestamp") != nil {
			continue
		}
		count++
		ready := false
		for _, c := range array(at(o, "status", "conditions")) {
			ready = ready || at(mapping(c), "type") == "Ready" && at(mapping(c), "status") == "True"
		}
		if !ready || at(o, "status", "phase") != "Running" || at(o, "spec", "serviceAccountName") != v.Name || at(o, "spec", "automountServiceAccountToken") != false || at(o, "spec", "nodeName") != w.Install.Config.Cluster+"-worker2" {
			return nil, Pending("Workload Pod not Ready or outside compute contract")
		}
		containers := array(at(o, "spec", "containers"))
		if len(containers) != 1 || at(mapping(containers[0]), "image") != v.Image {
			return nil, errors.New("Workload Pod image/container differs")
		}
		chosen = o
	}
	if count != int(v.Replicas) {
		return nil, Pending("Workload replica inventory differs")
	}
	return chosen, nil
}
func (w *Workflow) execProbe(ctx context.Context, pod Object, verb string) error {
	if verb != "probe-network" && verb != "probe-permissions" {
		return errors.New("unreviewed probe")
	}
	ns := str(at(pod, "metadata", "namespace"))
	name := str(at(pod, "metadata", "name"))
	uid := str(at(pod, "metadata", "uid"))
	if uid == "" {
		return errors.New("probe Pod UID missing")
	}
	again, e := w.get(ctx, "pod", ns, name)
	if e != nil {
		return e
	}
	if at(again, "metadata", "uid") != uid || at(again, "metadata", "resourceVersion") != at(pod, "metadata", "resourceVersion") {
		return errors.New("probe Pod changed before execution")
	}
	if _, e = w.kube(ctx, "exec", "-n", ns, name, "-c", "web", "--", "/atlas-web", verb); e != nil {
		return e
	}
	closing, e := w.get(ctx, "pod", ns, name)
	if e != nil {
		return e
	}
	if at(closing, "metadata", "uid") != uid {
		return errors.New("probe Pod changed during execution")
	}
	return nil
}
func (w *Workflow) Probe(ctx context.Context, p Plan, approval string) error {
	if e := w.approve(p, approval); e != nil {
		return e
	}
	b, e := regular(w.publicationPath(p, "consumer"), true)
	if e != nil {
		return e
	}
	var published Publication
	if e = workload.StrictDecode(b, &published); e != nil {
		return e
	}
	if published.PlanSHA256 != approval {
		return errors.New("consumer receipt differs")
	}
	result, e := w.Compile("consumer")
	if e != nil {
		return e
	}
	before, e := w.Observe(ctx, result, published.Commit)
	if e != nil {
		return e
	}
	if e = w.VerifyMaterialized(ctx); e != nil {
		return e
	}
	var bound, unbound int
	facts := Object{}
	nonce := make([]byte, 16)
	if _, e = rand.Read(nonce); e != nil {
		return e
	}
	body := []byte("Atlas S2 " + hex.EncodeToString(nonce) + "\n")
	for _, r := range w.Model.Workloads {
		v := r.Definition
		pod, e := w.readyPod(ctx, v)
		if e != nil {
			return e
		}
		client, e := w.httpsClient(v.Exposure.Hostname)
		if e != nil {
			return e
		}
		req, e := http.NewRequestWithContext(ctx, "POST", "https://"+v.Exposure.Hostname+"/roundtrip", bytes.NewReader(body))
		if e != nil {
			return e
		}
		res, e := client.Do(req)
		if e != nil {
			return errors.New("HTTPS request failed")
		}
		raw, readErr := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		res.Body.Close()
		client.CloseIdleConnections()
		if readErr != nil {
			return readErr
		}
		if r.Binding != nil {
			bound++
			var got Object
			if json.Unmarshal(raw, &got) != nil || res.StatusCode != 200 || got["sha256"] != workload.Digest(body) || got["cleanup"] != "deleted" {
				return errors.New("HTTPS → Web → S3 roundtrip failed")
			}
			if e = w.execProbe(ctx, pod, "probe-permissions"); e != nil {
				return e
			}
			facts[v.Name] = Object{"https": "PASS", "s3PutGetDelete": "PASS", "crossBucketDenied": "PASS", "bucketConfigurationDenied": "PASS", "podUID": at(pod, "metadata", "uid")}
		} else {
			unbound++
			if res.StatusCode != 503 {
				return errors.New("unbound WebService accepted S3 operation")
			}
			if e = w.execProbe(ctx, pod, "probe-network"); e != nil {
				return e
			}
			facts[v.Name] = Object{"noBinding": "PASS", "s3NetworkDenied": "PASS", "dnsResolved": "PASS", "podUID": at(pod, "metadata", "uid")}
		}
		// SubjectAccessReview is a finite diagnostic request, not a grant or a
		// Secret read. It is included in the approved probe's API operations.
		b, e = w.kube(ctx, "auth", "can-i", "get", "secrets", "-n", "workload-web", "--as=system:serviceaccount:"+v.Project+":"+v.Name)
		// kubectl exits 1 for the expected denial; never accept another failure.
		if strings.TrimSpace(string(b)) != "no" {
			if e != nil {
				return errors.New("RBAC denial could not be proven")
			}
			return errors.New("Workload can read foreign Secrets")
		}
		if e = w.metric(ctx, v); e != nil {
			return e
		}
	}
	if bound == 0 || unbound == 0 {
		return errors.New("final acceptance requires both bound and unbound WebServices")
	}
	if e = w.verifyLegacy(ctx); e != nil {
		return e
	}
	facts["legacyD1"] = Object{"s3Credentials": "PASS", "https": "PASS"}
	after, e := w.Observe(ctx, result, published.Commit)
	if e != nil {
		return e
	}
	if !bytes.Equal(workload.JSON(before.UID), workload.JSON(after.UID)) {
		return errors.New("resource identity changed during probe")
	}
	after.Runtime = "VERIFIED"
	out := Object{"result": "PASS", "exitCode": 0, "planSHA256": approval, "deploymentCommit": published.Commit, "compilerSHA256": w.BinarySHA256, "inventory": result.Inventory, "observation": after, "functional": facts, "scope": "single-owner local development; bucket mutation denial tested only in isolated synthetic fixture"}
	if e = save(filepath.Join(w.Config.StateDirectory, "authority", approval, "final.json"), workload.JSON(out), true); e != nil {
		return e
	}
	// Only after the live provider is verified may a later create/update use it
	// as its credential predecessor. D1 credentials remain byte-for-byte intact.
	provider, e := regular(filepath.Join(w.Config.StateDirectory, "provider-prepared.json"), true)
	if e != nil {
		return e
	}
	return save(filepath.Join(w.Config.StateDirectory, "provider.json"), provider, false)
}
func (w *Workflow) VerifyMaterialized(ctx context.Context) error {
	b, e := regular(filepath.Join(w.Config.StateDirectory, "keys.json"), true)
	if e != nil {
		return e
	}
	keys := map[string]Key{}
	if e = workload.StrictDecode(b, &keys); e != nil {
		return e
	}
	for _, v := range w.Model.Intent.Bindings {
		s, e := w.get(ctx, "secret", v.Project, v.Secret())
		if e != nil {
			return e
		}
		k := keys[v.ID()]
		if str(at(s, "data", "AWS_ACCESS_KEY_ID")) != encode(k.Access) || str(at(s, "data", "AWS_SECRET_ACCESS_KEY")) != encode(k.Secret) {
			return errors.New("materialized client credential differs")
		}
	}
	s, e := w.get(ctx, "secret", "atlas-storage", "seaweedfs-auth")
	if e != nil {
		return e
	}
	want, e := regular(filepath.Join(w.Config.StateDirectory, "provider-prepared.json"), true)
	if e != nil {
		return e
	}
	if at(s, "data", "seaweedfs_s3_config") != encode(string(want)) {
		return Pending("provider credential has not materialized")
	}
	return nil
}

func encode(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

func (w *Workflow) metric(ctx context.Context, v workload.Workload) error {
	if !v.Observability.Metrics {
		return nil
	}
	address, stop, e := w.forwardService(ctx, "prometheus")
	if e != nil {
		return e
	}
	defer stop()
	client := loopbackClient()
	defer client.CloseIdleConnections()
	return queryMetric(ctx, client, address, v)
}

func queryMetric(ctx context.Context, client *http.Client, address string, v workload.Workload) error {
	q := "up{namespace=\"" + v.Project + "\",service=\"" + v.Name + "\"}"
	req, e := http.NewRequestWithContext(ctx, "GET", address+"/api/v1/query?query="+url.QueryEscape(q), nil)
	if e != nil {
		return e
	}
	response, e := client.Do(req)
	if e != nil {
		return errors.New("Prometheus query unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return errors.New("Prometheus query HTTP failure")
	}
	const limit = 4 << 20
	raw, e := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if e != nil || len(raw) > limit {
		return errors.New("Prometheus response unreadable or oversized")
	}
	var metrics Object
	if e = json.Unmarshal(raw, &metrics); e != nil {
		return e
	}
	samples := array(at(metrics, "data", "result"))
	if metrics["status"] != "success" {
		return errors.New("Prometheus query failed")
	}
	if len(samples) != int(v.Replicas) {
		return Pending("Prometheus has not discovered all Workload targets")
	}
	for _, sample := range samples {
		value := array(at(mapping(sample), "value"))
		if len(value) != 2 || value[1] != "1" {
			return Pending("Workload scrape is not up")
		}
	}

	return nil
}
func (w *Workflow) metrics(ctx context.Context) error {
	for _, v := range w.Model.Intent.Workloads {
		if e := w.metric(ctx, v); e != nil {
			return e
		}
	}
	return nil
}
