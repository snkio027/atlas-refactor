// Package kubeconfig validates a local, minified Kind credential projection.
// It performs no reads, requests, discovery, or Kubernetes mutations.
package kubeconfig

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
)

type Projection struct {
	Clusters []struct {
		Name    string
		Cluster map[string]any
	}
	Contexts []struct {
		Name    string
		Context map[string]any
	}
	Users []struct {
		Name string
		User map[string]any
	}
}

func stringValue(v any) string { s, _ := v.(string); return s }

// Transport permits only explicit loopback HTTPS and inline client certificates.
// Callers must independently verify their private file digest and cluster UID.
func Transport(config Projection, contextName string) (string, *http.Transport, error) {
	if len(config.Clusters) != 1 || len(config.Contexts) != 1 || len(config.Users) != 1 || config.Contexts[0].Name != contextName || stringValue(config.Contexts[0].Context["cluster"]) != config.Clusters[0].Name || stringValue(config.Contexts[0].Context["user"]) != config.Users[0].Name {
		return "", nil, errors.New("ambiguous kubeconfig target")
	}
	c, u := config.Clusters[0].Cluster, config.Users[0].User
	for key := range c {
		if key != "server" && key != "certificate-authority-data" {
			return "", nil, errors.New("kubeconfig cluster has unsupported transport options")
		}
	}
	for key := range u {
		if key != "client-certificate-data" && key != "client-key-data" {
			return "", nil, errors.New("only explicit Kind client certificates are supported; exec/auth plugins forbidden")
		}
	}
	server := stringValue(c["server"])
	parsed, e := url.Parse(server)
	if e != nil || parsed.Scheme != "https" || parsed.Hostname() != "127.0.0.1" || parsed.Port() == "" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return "", nil, errors.New("explicit loopback HTTPS API is required")
	}
	decode := func(v any) ([]byte, error) { return base64.StdEncoding.DecodeString(stringValue(v)) }
	ca, e := decode(c["certificate-authority-data"])
	if e != nil {
		return "", nil, errors.New("invalid CA encoding")
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return "", nil, errors.New("invalid API CA")
	}
	cert, e := decode(u["client-certificate-data"])
	if e != nil {
		return "", nil, errors.New("invalid client certificate encoding")
	}
	key, e := decode(u["client-key-data"])
	if e != nil {
		return "", nil, errors.New("invalid client key encoding")
	}
	pair, e := tls.X509KeyPair(cert, key)
	if e != nil {
		return "", nil, errors.New("invalid API client key pair")
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool, Certificates: []tls.Certificate{pair}}}
	return server, transport, nil
}
