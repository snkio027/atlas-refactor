package workloadrun

import (
	"atlas-refactor/internal/installation"
	"atlas-refactor/internal/platform"
	"atlas-refactor/internal/workload"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type Plan struct {
	Schema               int      `json:"schema"`
	Cluster              string   `json:"cluster"`
	ClusterUID           string   `json:"clusterUID"`
	InstallID            string   `json:"installID"`
	Repository           string   `json:"repository"`
	Branch               string   `json:"branch"`
	Parent               string   `json:"parent"`
	BaseCommit           string   `json:"baseCommit"`
	ProductSHA256        string   `json:"productSHA256"`
	CompilerSHA256       string   `json:"compilerSHA256"`
	ConfigSHA256         string   `json:"configSHA256"`
	IntentSHA256         string   `json:"intentSHA256"`
	InfrastructureSHA256 string   `json:"infrastructureSHA256"`
	CertificateSHA256    string   `json:"certificateSHA256"`
	Project              string   `json:"project"`
	CredentialTargets    []string `json:"credentialTargets"`
	Phases               []string `json:"phases"`
	ImplementationCommit string   `json:"implementationCommit"`
	GoVersion            string   `json:"goVersion"`
	SourceDirty          bool     `json:"sourceDirty"`
}

func (w *Workflow) Plan(ctx context.Context) (Plan, error) {
	p := Plan{Schema: 1, Cluster: w.Install.Config.Cluster, ClusterUID: w.Install.Record.ClusterUID, InstallID: w.Install.Record.InstallID, Repository: w.Context.Repository, Branch: w.Context.Branch, BaseCommit: w.Install.Record.FullCommit, ProductSHA256: w.Install.ProductDigest, CompilerSHA256: w.BinarySHA256, ConfigSHA256: workload.Digest(workload.JSON(w.Config)), IntentSHA256: workload.Digest(workload.JSON(w.Model.Intent)), CertificateSHA256: w.Install.Record.CertificateSHA256, Project: w.Model.Intent.Project.Name, CredentialTargets: []string{}, Phases: []string{"infrastructure", "consumer"}}
	if len(w.Model.Intent.Bindings) == 0 || len(w.Model.Intent.Workloads) <= len(w.Model.Intent.Bindings) {
		return p, errors.New("this S2 acceptance runner requires both bound and unbound WebServices")
	}
	if e := w.checkPublicationAccess(ctx); e != nil {
		return p, e
	}
	p.ImplementationCommit = w.BuildSource
	p.GoVersion = w.BuildGoVersion
	p.SourceDirty = w.BuildDirty
	parent, e := w.remote(ctx)
	if e != nil {
		return p, e
	}
	p.Parent = parent
	if _, _, e := w.imageArchive(); e != nil {
		return p, e
	}
	current, e := w.ReadTree(ctx, parent)
	if e != nil {
		return p, e
	}
	if parent != w.Install.Record.FullCommit {
		if e = w.validatePredecessor(current); e != nil {
			return p, e
		}
		var inv workload.Inventory
		if e = installation.Decode(current[workload.InventoryPath], &inv); e != nil {
			return p, e
		}
		if inv.Phase != "consumer" {
			return p, errors.New("an existing infrastructure publication must finish under its original plan")
		}
		p.Phases = []string{"consumer"}
	}
	r, e := w.Compile("infrastructure")
	if e != nil {
		return p, e
	}
	p.InfrastructureSHA256 = platform.BundleDigest(r.Files)
	p.CredentialTargets = append(p.CredentialTargets, "atlas-storage/seaweedfs-auth")
	for _, b := range w.Model.Intent.Bindings {
		p.CredentialTargets = append(p.CredentialTargets, b.Project+"/"+b.Secret())
	}
	return p, nil
}
func (w *Workflow) ReadPlan() (Plan, error) {
	var p Plan
	b, e := regular(filepath.Join(w.Config.StateDirectory, "plan.json"), true)
	if e == nil {
		e = workload.StrictDecode(b, &p)
	}
	return p, e
}
func (w *Workflow) approve(p Plan, digest string) error {
	if p.SourceDirty || w.BuildDirty || !fullSHA.MatchString(p.ImplementationCommit) || p.ImplementationCommit != w.BuildSource || p.GoVersion != w.BuildGoVersion {
		return errors.New("execution requires a clean, source-bound compiled binary")
	}
	if digest == "" || digest != workload.Digest(workload.JSON(p)) || p.Schema != 1 || p.CompilerSHA256 != w.BinarySHA256 || p.ProductSHA256 != w.Install.ProductDigest || p.ClusterUID != w.Install.Record.ClusterUID || p.InstallID != w.Install.Record.InstallID || p.Repository != w.Context.Repository || p.Branch != w.Context.Branch || p.BaseCommit != w.Install.Record.FullCommit || p.ConfigSHA256 != workload.Digest(workload.JSON(w.Config)) || p.IntentSHA256 != workload.Digest(workload.JSON(w.Model.Intent)) || p.CertificateSHA256 != w.Install.Record.CertificateSHA256 {
		return errors.New("exact approved S2 plan/input binding required")
	}
	r, e := w.Compile("infrastructure")
	if e != nil {
		return e
	}
	if platform.BundleDigest(r.Files) != p.InfrastructureSHA256 {
		return errors.New("compiled infrastructure changed")
	}
	return nil
}
func (w *Workflow) SavePlan(p Plan) error {
	return save(filepath.Join(w.Config.StateDirectory, "plan.json"), workload.JSON(p), false)
}
func (w *Workflow) validatePredecessor(files Files) error {
	var inv workload.Inventory
	if e := installation.Decode(files[workload.InventoryPath], &inv); e != nil {
		return e
	}
	if inv.Schema != 1 || inv.ProductSHA256 != w.Install.ProductDigest || inv.BaseSHA256 != platform.BundleDigest(w.Context.Base) {
		return errors.New("unknown S2 predecessor")
	}
	for name, digest := range inv.Files {
		if workload.Digest(files[name]) != digest {
			return fmt.Errorf("prior generated/authored file drifted: %s", name)
		}
	}
	in := workload.Intent{Workloads: []workload.Workload{}, Bindings: []workload.Binding{}}
	for name, b := range files {
		switch {
		case strings.HasPrefix(name, "platform/projects/"):
			if in.Project.Name != "" {
				return errors.New("multiple predecessor projects")
			}
			if e := workload.StrictDecode(b, &in.Project); e != nil {
				return e
			}
		case strings.HasPrefix(name, "platform/workloads/"):
			var v workload.Workload
			if e := workload.StrictDecode(b, &v); e != nil {
				return e
			}
			in.Workloads = append(in.Workloads, v)
		case strings.HasPrefix(name, "platform/bindings/"):
			var v workload.Binding
			if e := workload.StrictDecode(b, &v); e != nil {
				return e
			}
			in.Bindings = append(in.Bindings, v)
		}
	}
	old, e := workload.Resolve(in, true)
	if e != nil {
		return e
	}
	return workload.ValidateUpdate(old, w.Model)
}

type Publication struct {
	Schema     int    `json:"schema"`
	PlanSHA256 string `json:"planSHA256"`
	Phase      string `json:"phase"`
	Parent     string `json:"parent"`
	Commit     string `json:"commit"`
	TreeSHA256 string `json:"treeSHA256"`
}

func (w *Workflow) publicationPath(p Plan, phase string) string {
	return filepath.Join(w.Config.StateDirectory, "authority", workload.Digest(workload.JSON(p)), phase+".json")
}
func (w *Workflow) Publish(ctx context.Context, p Plan, approval, phase string) (Publication, error) {
	var receipt Publication
	if e := w.approve(p, approval); e != nil {
		return receipt, e
	}
	if e := w.bind(ctx); e != nil {
		return receipt, e
	}
	allowed := false
	for _, s := range p.Phases {
		allowed = allowed || s == phase
	}
	if !allowed {
		return receipt, errors.New("phase not in approved plan")
	}
	result, e := w.Compile(phase)
	if e != nil {
		return receipt, e
	}
	current, e := w.remote(ctx)
	if e != nil {
		return receipt, e
	}
	// A completed phase is an idempotent read, not another publication.
	if b, e := regular(w.publicationPath(p, phase), true); e == nil {
		if e = workload.StrictDecode(b, &receipt); e != nil {
			return receipt, e
		}
		if receipt.PlanSHA256 != approval || receipt.TreeSHA256 != platform.BundleDigest(result.Files) || receipt.Commit != current {
			return receipt, errors.New("published phase/remote drift")
		}
		return receipt, nil
	} else if !os.IsNotExist(e) {
		return receipt, e
	}
	expectedParent := p.Parent
	if phase == "consumer" && len(p.Phases) == 2 {
		b, e := regular(w.publicationPath(p, "infrastructure"), true)
		if e != nil {
			return receipt, e
		}
		var prior Publication
		if e = workload.StrictDecode(b, &prior); e != nil {
			return receipt, e
		}
		if prior.PlanSHA256 != approval || prior.Phase != "infrastructure" || !fullSHA.MatchString(prior.Commit) {
			return receipt, errors.New("infrastructure receipt mismatch")
		}
		expectedParent = prior.Commit
		infra, e := w.Compile("infrastructure")
		if e != nil {
			return receipt, e
		}
		if _, e = w.Observe(ctx, infra, expectedParent); e != nil {
			return receipt, fmt.Errorf("infrastructure gate: %w", e)
		}
	}
	if current != expectedParent {
		return receipt, errors.New("deployment branch moved outside approved sequence")
	}
	parentFiles, e := w.ReadTree(ctx, current)
	if e != nil {
		return receipt, e
	}
	if current != w.Install.Record.FullCommit {
		if e = w.validatePredecessor(parentFiles); e != nil {
			return receipt, e
		}
	}
	// Preserve every unrelated path from the complete current parent. Only the
	// compiler's declared delta can replace existing bytes.
	merged := Files{}
	for k, v := range parentFiles {
		merged[k] = v
	}
	for k, v := range result.Files {
		if bytes.Equal(v, w.Context.Base[k]) {
			if !bytes.Equal(parentFiles[k], v) {
				return receipt, fmt.Errorf("base file drift: %s", k)
			}
			continue
		}
		if old, exists := parentFiles[k]; exists && current == w.Install.Record.FullCommit && !bytes.Equal(old, w.Context.Base[k]) {
			return receipt, errors.New("new output collides with existing deployment path")
		}
		merged[k] = v
	}
	if sameFiles(parentFiles, merged) {
		receipt = Publication{1, approval, phase, current, current, platform.BundleDigest(result.Files)}
		return receipt, save(w.publicationPath(p, phase), workload.JSON(receipt), true)
	}
	if e = w.checkPublicationAccess(ctx); e != nil {
		return receipt, e
	}
	commit, e := w.commit(ctx, merged, current, "S2 "+phase+" for "+p.Project)
	if e != nil {
		return receipt, e
	}
	receipt = Publication{1, approval, phase, current, commit, platform.BundleDigest(result.Files)}
	// Retain request intent before the only remote mutation. Ambiguous outcomes
	// require inspection; they are never blindly republished or force recovered.
	if e = save(w.publicationPath(p, phase)+".intent", workload.JSON(receipt), true); e != nil {
		return receipt, e
	}
	if _, e = w.git(ctx, nil, "push", "--force-with-lease=refs/heads/"+p.Branch+":"+current, p.Repository, commit+":refs/heads/"+p.Branch); e != nil {
		return receipt, fmt.Errorf("%s Git push failed; intent retained for outcome inspection: %w", phase, e)
	}
	observed, e := w.remote(ctx)
	if e != nil || observed != commit {
		return receipt, errors.New("publication outcome unknown; preserve intent")
	}
	return receipt, save(w.publicationPath(p, phase), workload.JSON(receipt), true)
}
func sameFiles(a, b Files) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if !bytes.Equal(v, b[k]) {
			return false
		}
	}
	return true
}
func (w *Workflow) commit(ctx context.Context, files Files, parent, message string) (string, error) {
	index, e := os.CreateTemp(w.Config.StateDirectory, "git-index-")
	if e != nil {
		return "", e
	}
	index.Close()
	os.Remove(index.Name())
	defer os.Remove(index.Name())
	run := func(input []byte, args ...string) ([]byte, error) {
		c := exec.CommandContext(ctx, "git", publicationGitArgs(args...)...)
		c.Dir = w.repo()
		c.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), "GIT_INDEX_FILE=" + index.Name(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0", "GIT_AUTHOR_NAME=Atlas S2", "GIT_AUTHOR_EMAIL=atlas-s2@localhost", "GIT_COMMITTER_NAME=Atlas S2", "GIT_COMMITTER_EMAIL=atlas-s2@localhost"}
		c.Stdin = bytes.NewReader(input)
		b, e := c.Output()
		if e != nil {
			return nil, errors.New("local Git tree construction failed")
		}
		return b, nil
	}
	if _, e = run(nil, "read-tree", parent); e != nil {
		return "", e
	}
	parentTree, e := w.ReadTree(ctx, parent)
	if e != nil {
		return "", e
	}
	names := []string{}
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if bytes.Equal(files[name], parentTree[name]) {
			continue
		}
		if strings.ContainsAny(name, "\n\r\x00\t") || filepath.Clean(name) != name || filepath.IsAbs(name) || strings.HasPrefix(name, "../") {
			return "", errors.New("unsafe output path")
		}
		blob, e := run(files[name], "hash-object", "-w", "--stdin")
		if e != nil {
			return "", e
		}
		if _, e = run(nil, "update-index", "--add", "--cacheinfo", "100644", strings.TrimSpace(string(blob)), name); e != nil {
			return "", e
		}
	}
	tree, e := run(nil, "write-tree")
	if e != nil {
		return "", e
	}
	b, e := run([]byte(message+"\n"), "commit-tree", strings.TrimSpace(string(tree)), "-p", parent)
	if e != nil {
		return "", e
	}
	revision := strings.TrimSpace(string(b))
	if !fullSHA.MatchString(revision) {
		return "", errors.New("invalid generated commit")
	}
	return revision, nil
}
