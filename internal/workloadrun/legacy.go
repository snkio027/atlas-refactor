package workloadrun

import (
	"atlas-refactor/internal/installation"
	"atlas-refactor/internal/webslice"
	"atlas-refactor/internal/workload"
	"bufio"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"
)

func (w *Workflow) verifyLegacy(ctx context.Context) error {
	raw, e := regular(filepath.Join(w.Install.Config.StateDirectory, "credentials.json"), true)
	if e != nil {
		return e
	}
	if workload.Digest(raw) != w.Install.Record.CredentialsSHA256 {
		return errors.New("D1 credentials changed")
	}
	var c installation.Credentials
	if e = installation.Decode(raw, &c); e != nil {
		return e
	}
	k := filepath.Join(installation.ToolDirectory(w.Install.Config.StateDirectory, w.Install.Product.Tools), "kubectl")
	forwardCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(forwardCtx, k, "--kubeconfig", filepath.Join(w.Install.Config.StateDirectory, "runtime/.state/kubeconfig"), "--context", "kind-"+w.Install.Config.Cluster, "-n", "atlas-storage", "port-forward", "--address=127.0.0.1", "--pod-running-timeout=30s", "service/seaweedfs", ":8333")
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME")}
	stdout, e := cmd.StdoutPipe()
	if e != nil {
		return e
	}
	if e = cmd.Start(); e != nil {
		return e
	}
	defer func() { cancel(); _ = cmd.Wait() }()
	addresses := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		re := regexp.MustCompile(`^Forwarding from 127\.0\.0\.1:([0-9]+) -> 8333$`)
		for scanner.Scan() {
			if m := re.FindStringSubmatch(scanner.Text()); len(m) == 2 {
				select {
				case addresses <- m[1]:
				default:
				}
				return
			}
		}
		close(addresses)
	}()
	var port string
	select {
	case port = <-addresses:
		if port == "" {
			return errors.New("S3 forwarding unavailable")
		}
	case <-time.After(30 * time.Second):
		return errors.New("S3 forwarding timed out")
	case <-ctx.Done():
		return ctx.Err()
	}
	// Dial remains loopback and there is no credential-provider/environment chain.
	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	s3 := webslice.Client{Endpoint: "http://127.0.0.1:" + port, Bucket: "uploads", Access: c.AccessKey, Secret: c.SecretKey, HTTP: client}
	body, code, e := s3.Request(ctx, "GET", "/uploads?list-type=2", nil)
	if e != nil || code != 200 {
		return errors.New("existing D1 S3 identity stopped working")
	}
	var list struct {
		XMLName xml.Name
		Name    string
	}
	if xml.Unmarshal(body, &list) != nil || list.XMLName.Local != "ListBucketResult" || list.Name != "uploads" {
		return errors.New("legacy S3 bucket identity differs")
	}
	web, e := w.httpsClient("web.atlas.test")
	if e != nil {
		return e
	}
	defer web.CloseIdleConnections()
	req, e := http.NewRequestWithContext(ctx, "GET", "https://web.atlas.test/", nil)
	if e != nil {
		return e
	}
	response, e := web.Do(req)
	if e != nil {
		return errors.New("legacy D1 HTTPS endpoint unavailable")
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		return fmt.Errorf("legacy D1 HTTPS status %d", response.StatusCode)
	}
	return nil
}
