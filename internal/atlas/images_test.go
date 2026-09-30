package atlas

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type runnerFunc func(context.Context, Request) ([]byte, error)

func (r runnerFunc) Run(ctx context.Context, q Request) ([]byte, error) { return r(ctx, q) }

func TestOfflineImageImportStreamsOnlyNodePlatform(t *testing.T) {
	a, sim := fixture(t)
	var archives []string
	a.Runner = runnerFunc(func(ctx context.Context, q Request) ([]byte, error) {
		joined := strings.Join(q.Args, " ")
		if q.Tool == "kind" && q.Args[0] == "load" {
			t.Fatal("kind convenience loader cannot preserve this offline import contract")
		}
		if q.Tool == "docker" && strings.Contains(joined, "images import") {
			if strings.Contains(joined, "--all-platforms") || argValue(q.Args, "--platform") != "linux/arm64" || len(q.Input) != 0 || q.InputPath == "" {
				t.Fatalf("unexpected import request: %+v", q)
			}
			path := filepath.Join(a.Root, q.InputPath)
			b, e := os.ReadFile(path)
			if e != nil || string(b) != "archive" {
				t.Fatal("archive unavailable to streaming process adapter", e)
			}
			archives = append(archives, path)
		}
		return sim.Run(ctx, q)
	})
	apply(t, a)
	if len(archives) != 2 {
		t.Fatalf("expected two offline image imports, got %d", len(archives))
	}
	for _, path := range archives {
		if _, e := os.Stat(path); !errors.Is(e, os.ErrNotExist) {
			t.Fatal("temporary archive retained after import", path, e)
		}
	}
}

func TestImageImportFailureCannotInstallSeed(t *testing.T) {
	for _, failure := range []string{"missing-platform", "import-fails", "digest-absent"} {
		t.Run(failure, func(t *testing.T) {
			a, sim := fixture(t)
			a.Runner = runnerFunc(func(ctx context.Context, q Request) ([]byte, error) {
				joined := strings.Join(q.Args, " ")
				if q.Tool == "docker" {
					switch {
					case failure == "missing-platform" && strings.Contains(joined, "image inspect --platform"):
						return []byte("linux/amd64"), nil
					case failure == "import-fails" && strings.Contains(joined, "images import"):
						return nil, errors.New("incomplete archive")
					case failure == "digest-absent" && strings.Contains(joined, "images list"):
						return nil, nil
					}
				}
				return sim.Run(ctx, q)
			})
			if e := a.Apply(context.Background(), a.Config.Cluster, true); e == nil {
				t.Fatal("expected offline import failure")
			}
			for _, effect := range sim.effects {
				if effect == "seed:apply" || effect == "create:atlas-refactor-handoff" || effect == "create:atlas-refactor-root" || (failure == "missing-platform" && effect == "cluster:create") {
					t.Fatalf("effect after failed precondition: %s", effect)
				}
			}
			archives, e := filepath.Glob(filepath.Join(a.Root, ".state", "image-*.tar"))
			if e != nil || len(archives) != 0 {
				t.Fatal("failed import leaked private archive", archives, e)
			}
		})
	}
}

func TestNodeImageReferenceMustBeBoundToLockedDigest(t *testing.T) {
	a, sim := fixture(t)
	if e := os.MkdirAll(filepath.Join(a.Root, ".state"), 0700); e != nil {
		t.Fatal(e)
	}
	_, digest, _ := strings.Cut(a.Lock.ArgoImage, "@")
	source := "imported.example/argocd@" + digest
	tagged := false
	a.Runner = runnerFunc(func(ctx context.Context, q Request) ([]byte, error) {
		joined := strings.Join(q.Args, " ")
		if q.Tool == "docker" && strings.Contains(joined, "images list") {
			if !strings.HasSuffix(joined, "target.digest=="+digest) {
				t.Fatal("digest filter missing")
			}
			if tagged {
				return []byte(a.Lock.ArgoImage), nil
			}
			return []byte(source), nil
		}
		if q.Tool == "docker" && strings.Contains(joined, "images tag") {
			if q.Args[len(q.Args)-2] != source || q.Args[len(q.Args)-1] != a.Lock.ArgoImage || strings.Contains(joined, "--force") {
				t.Fatal("unbound image tag")
			}
			tagged = true
			return nil, nil
		}
		return sim.Run(ctx, q)
	})
	if e := a.loadNodeImage(context.Background(), a.Lock.ArgoImage); e != nil {
		t.Fatal(e)
	}
	if !tagged {
		t.Fatal("exact locked reference not restored")
	}
}

func TestTrackingMethodIsConfigMapData(t *testing.T) {
	a, _ := fixture(t)
	files, e := a.Render(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	seed := string(files["bootstrap/seed.yaml"])
	if !strings.Contains(seed, "\ndata:\n  application.resourceTrackingMethod: annotation\n") || strings.Contains(seed, "metadata:\n  application.resourceTrackingMethod:") {
		t.Fatal("tracking configuration belongs in ConfigMap data, not metadata")
	}
}

func TestProcessStreamsPrivateInputAndRejectsEscape(t *testing.T) {
	root := t.TempDir()
	if e := os.Mkdir(filepath.Join(root, ".state"), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(root, ".state/input"), []byte("streamed archive bytes"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(root, "helm"), []byte("#!/bin/sh\nexec cat\n"), 0700); e != nil {
		t.Fatal(e)
	}
	runner := ExecRunner{Root: root, ToolDir: root, DockerContext: "orbstack"}
	output, e := runner.Run(context.Background(), Request{Tool: "helm", InputPath: ".state/input"})
	if e != nil || string(output) != "streamed archive bytes" {
		t.Fatal("file input did not reach process", e)
	}
	for _, path := range []string{"input", ".state/../helm", ".state"} {
		if _, e := runner.Run(context.Background(), Request{Tool: "helm", InputPath: path}); e == nil {
			t.Fatal("private file input escape accepted", path)
		}
	}
}

func TestRelativeToolDirectoryResolvedBeforeChildChangesDirectory(t *testing.T) {
	caller := t.TempDir()
	t.Chdir(caller)
	if err := os.Mkdir("tools", 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("tools/helm", []byte("#!/bin/sh\nprintf '%s' caller-tool\n"), 0700); err != nil {
		t.Fatal(err)
	}
	runner := ExecRunner{Root: t.TempDir(), ToolDir: "tools", DockerContext: "orbstack"}
	output, err := runner.Run(context.Background(), Request{Tool: "helm", Args: []string{"version"}})
	if err != nil || string(output) != "caller-tool" {
		t.Fatalf("relative tool resolved in child cwd: output=%q err=%v", output, err)
	}
}
