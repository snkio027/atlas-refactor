// atlas-platform is a local manifest tool, never a cluster mutation engine.
package main

import (
	"atlas-refactor/internal/platform"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"
)

func main() {
	fs := flag.NewFlagSet("atlas-platform", flag.ExitOnError)
	root := fs.String("root", ".", "repository root")
	helm := fs.String("helm", "helm", "locked Helm executable")
	kubectl := fs.String("kubectl", "kubectl", "locked kubectl (client commands only)")
	yq := fs.String("yq", "yq", "locked yq executable")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: atlas-platform [flags] render|check (local files only)")
		fs.PrintDefaults()
	}
	fs.Parse(os.Args[1:])
	if fs.NArg() != 1 || (fs.Arg(0) != "render" && fs.Arg(0) != "check") {
		fs.Usage()
		os.Exit(2)
	}
	p, e := platform.Load(*root, platform.Tools{Helm: *helm, Kubectl: *kubectl, YQ: *yq})
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(signalCtx, 3*time.Minute)
	defer cancel()
	if fs.Arg(0) == "render" {
		var files map[string][]byte
		files, e = p.Render(ctx)
		if e == nil {
			e = p.Write(files)
		}
		if e == nil {
			fmt.Printf("Rendered %d files; no cluster operations.\n", len(files))
		}
	} else {
		var n int
		n, e = p.Check(ctx)
		if e == nil {
			fmt.Printf("Checked %d GitOps resources, artifact hashes, deterministic renders, project boundaries, CRD structural schemas and local Kustomize builds. No cluster operations.\n", n)
		}
	}
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
