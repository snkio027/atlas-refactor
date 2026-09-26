// atlas-platform is a local manifest tool, never a cluster mutation engine.
package main

import (
	"atlas-refactor/internal/platform"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"
)

func main() {
	fs := flag.NewFlagSet("atlas-platform", flag.ExitOnError)
	root := fs.String("root", ".", "repository root")
	helm := fs.String("helm", "helm", "locked Helm executable")
	kubectl := fs.String("kubectl", "kubectl", "locked kubectl (client commands only)")
	cert := fs.String("cert", "", "reviewed public Sealed Secrets certificate (offline)")
	kubeseal := fs.String("kubeseal", "kubeseal", "checksum-locked kubeseal executable")
	capabilities := fs.String("capabilities", "monitoring,object-storage,storage-monitoring", "complete desired capability set for plan/select; removal is unsupported")
	yq := fs.String("yq", "yq", "locked yq executable")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: atlas-platform [flags] render|check|plan|select|prepare-credentials (local files only)")
		fs.PrintDefaults()
	}
	fs.Parse(os.Args[1:])
	if fs.NArg() != 1 || (fs.Arg(0) != "render" && fs.Arg(0) != "check" && fs.Arg(0) != "plan" && fs.Arg(0) != "select" && fs.Arg(0) != "prepare-credentials") {
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

	requested := []string{}
	if *capabilities != "" {
		requested = strings.Split(*capabilities, ",")
	}
	if fs.Arg(0) == "prepare-credentials" {
		e = p.PrepareCredentials(ctx, *cert, *kubeseal)
		if e == nil {
			fmt.Println("Prepared sealed manifests; plaintext remains in private .state/capabilities/credentials.json. No cluster operations.")
		}
	} else if fs.Arg(0) == "select" {
		e = p.SelectCapabilities(ctx, requested)
		if e == nil {
			fmt.Println("Updated local capability selection and GitOps projection. Review the diff before publishing.")
		}
	} else if fs.Arg(0) == "plan" {
		var plan platform.Object
		plan, e = p.CapabilityPlan(requested)
		if e == nil {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			e = encoder.Encode(plan)
		}
	} else if fs.Arg(0) == "render" {
		plan, err := p.CapabilityPlan(p.Capabilities.Enabled.Capabilities)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err = p.ValidateCapabilitySelection(p.Capabilities.Active); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if plan["readyToEnable"] != true {
			fmt.Fprintln(os.Stderr, "enabled capabilities are missing sealed credentials; run plan")
			os.Exit(1)
		}
		var files map[string][]byte
		files, e = p.Render(ctx)
		if e == nil {
			e = p.Write(files)
		}

		if e == nil {
			files, e = p.CapabilityActivation(p.Capabilities.Active)
			if e == nil {
				e = p.Write(files)
			}
		}
		if e == nil {
			e = p.ValidateCapabilityActivation()
		}
		if e == nil {
			fmt.Println("Rendered core, capability candidates and active catalog; no cluster operations.")
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
