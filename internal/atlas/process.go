package atlas

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Request struct {
	Tool  string
	Args  []string
	Input []byte
}
type Runner interface {
	Run(context.Context, Request) ([]byte, error)
}
type ExecRunner struct {
	Root, ToolDir string
	DockerContext string
}

func (r ExecRunner) Run(ctx context.Context, q Request) ([]byte, error) {
	allowed := map[string]bool{"git": true, "helm": true, "kind": true, "kubectl": true, "docker": true}
	if !allowed[q.Tool] {
		return nil, errors.New("unapproved executable")
	}
	executable := q.Tool
	if r.ToolDir != "" && q.Tool != "git" && q.Tool != "docker" {
		executable = filepath.Join(r.ToolDir, q.Tool)
	}
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "DOCKER_") || strings.HasPrefix(key, "KIND_") {
			return nil, fmt.Errorf("ambient %s is forbidden; unset it before running Atlas", key)
		}
	}
	deadline := 2 * time.Minute
	if q.Tool == "kind" {
		deadline = 10 * time.Minute
	}
	if q.Tool == "kubectl" {
		deadline = 4 * time.Minute // Covers the explicit 180-second rollout wait.
	}
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, q.Args...)
	cmd.Dir = r.Root
	for _, key := range []string{"PATH", "HOME", "TMPDIR", "SYSTEMROOT"} {
		if v, ok := os.LookupEnv(key); ok {
			cmd.Env = append(cmd.Env, key+"="+v)
		}
	}
	cmd.Env = append(cmd.Env, "LC_ALL=C", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0", "DOCKER_CONTEXT="+r.DockerContext, "KIND_EXPERIMENTAL_PROVIDER=docker")
	cmd.Stdin = bytes.NewReader(q.Input)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if e := cmd.Run(); e != nil {
		// Subprocess output can contain kubeconfig or credentials. Never echo it.
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%s: %w", q.Tool, ctx.Err())
		}
		return nil, fmt.Errorf("%s command failed (%v); sensitive subprocess output suppressed", q.Tool, e)
	}
	return out.Bytes(), nil
}

func (a *App) run(ctx context.Context, tool string, args ...string) ([]byte, error) {
	return a.Runner.Run(ctx, Request{Tool: tool, Args: args})
}

func (a *App) verifyTools(ctx context.Context, all bool) error {
	b, e := a.run(ctx, "helm", "version", "--short")
	if e != nil {
		return e
	}
	if !strings.HasPrefix(strings.TrimSpace(string(b)), "v"+a.Lock.Helm+"+") {
		return fmt.Errorf("helm %s required", a.Lock.Helm)
	}
	if !all {
		return nil
	}
	b, e = a.run(ctx, "kind", "version")
	if e != nil {
		return e
	}
	if !strings.HasPrefix(string(b), "kind v"+a.Lock.Kind+" ") {
		return fmt.Errorf("kind %s required", a.Lock.Kind)
	}
	b, e = a.run(ctx, "kubectl", "version", "--client", "-o", "json")
	if e != nil {
		return e
	}
	var version struct {
		ClientVersion struct {
			GitVersion string `json:"gitVersion"`
		} `json:"clientVersion"`
	}
	if e = decode(b, &version); e != nil {
		return e
	}
	if version.ClientVersion.GitVersion != "v"+a.Lock.Kubectl {
		return fmt.Errorf("kubectl %s required", a.Lock.Kubectl)
	}
	b, e = a.run(ctx, "docker", "context", "inspect", a.Config.DockerContext, "--format", "{{.Endpoints.docker.Host}}")
	if e != nil {
		return e
	}
	home, e := os.UserHomeDir()
	if e != nil {
		return e
	}
	if strings.TrimSpace(string(b)) != "unix://"+filepath.Join(home, ".orbstack/run/docker.sock") {
		return errors.New("Docker context is not the owner-local OrbStack socket")
	}
	return nil
}
