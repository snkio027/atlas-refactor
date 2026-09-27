package observation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

func Collect(ctx context.Context, reader Reader, expect Expectation) (Envelope, error) {
	if err := expect.Validate(); err != nil {
		return Envelope{}, err
	}
	out := Envelope{Schema: "atlas.observation/v1", Subject: expect.Subject, Target: expect.Target, ImplementationSHA: expect.ImplementationSHA, ExpectedRevision: expect.Revision, ExpectationSHA256: Digest(expect), StartedAt: time.Now().UTC(), Classification: Verified, Applications: []ApplicationFact{}, Resources: []ResourceFact{}, Raw: []Object{}}
	finish := func() Envelope { out.FinishedAt = time.Now().UTC(); return out }
	uid, err := reader.ClusterIdentity(ctx)
	if err != nil || uid != expect.Target.ClusterUID {
		out.Classification = Unknown
		out.Reasons = []string{"TARGET_BINDING_UNAVAILABLE_OR_CHANGED"}
		return finish(), nil
	}
	apps := append([]ExpectedApplication(nil), expect.Applications...)
	sort.Slice(apps, func(i, j int) bool { return apps[i].Name < apps[j].Name })
	resources := append([]ExpectedResource(nil), expect.Resources...)
	sort.Slice(resources, func(i, j int) bool { return resources[i].Ref.Key() < resources[j].Ref.Key() })
	versions := map[Ref]string{}
	for _, a := range apps {
		ref := Ref{"argoproj.io/v1alpha1", "Application", "argocd", a.Name}
		o, e := reader.Read(ctx, ref)
		fact := ClassifyApplication(a, o, e)
		out.Applications = append(out.Applications, fact)
		out.Classification = worse(out.Classification, fact.Classification)
		if o != nil && e == nil && Reference(o) == ref {
			out.Raw = append(out.Raw, o)
			versions[ref] = String(At(o, "metadata", "uid")) + "/" + String(At(o, "metadata", "resourceVersion"))
		}
	}
	for _, r := range resources {
		o, e := reader.Read(ctx, r.Ref)
		fact := ClassifyResource(r, o, e)
		out.Resources = append(out.Resources, fact)
		out.Classification = worse(out.Classification, fact.Classification)
		if o != nil && e == nil && Reference(o) == r.Ref {
			out.Raw = append(out.Raw, o)
			versions[r.Ref] = String(At(o, "metadata", "uid")) + "/" + String(At(o, "metadata", "resourceVersion"))
		}
	}
	// No Kubernetes multi-object transaction is claimed. A closing fence catches
	// object churn during collection and produces UNKNOWN, never a repaired view.
	refs := []Ref{}
	for ref := range versions {
		refs = append(refs, ref)
	}
	SortedRefs(refs)
	for _, ref := range refs {
		o, e := reader.Read(ctx, ref)
		if e != nil || o == nil || Reference(o) != ref || String(At(o, "metadata", "uid"))+"/"+String(At(o, "metadata", "resourceVersion")) != versions[ref] {
			out.Classification = Unknown
			out.Reasons = append(out.Reasons, "SNAPSHOT_CHANGED_OR_UNAVAILABLE:"+ref.Key())
		}
	}
	uid, err = reader.ClusterIdentity(ctx)
	if err != nil || uid != expect.Target.ClusterUID {
		out.Classification = Unknown
		out.Reasons = append(out.Reasons, "TARGET_BINDING_CHANGED_DURING_READ")
	}
	return finish(), nil
}

// PrivateDirectory never follows symlinks and refuses existing loose directories.
func PrivateDirectory(path string) error {
	abs, e := filepath.Abs(path)
	if e != nil {
		return e
	}
	missing := []string{}
	for p := abs; ; p = filepath.Dir(p) {
		info, e := os.Lstat(p)
		if os.IsNotExist(e) {
			missing = append(missing, p)
		} else if e != nil {
			return e
		} else if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return errors.New("non-directory/symlink in evidence path")
		}
		if p == filepath.Dir(p) {
			break
		}
	}
	for i := len(missing) - 1; i >= 0; i-- {
		if e := os.Mkdir(missing[i], 0700); e != nil {
			return e
		}
	}
	info, e := os.Stat(abs)
	if e != nil {
		return e
	}
	if info.Mode().Perm()&0077 != 0 {
		return errors.New("evidence directory must be owner-only")
	}
	return nil
}
func CreatePrivate(path string, data []byte) error {
	if e := PrivateDirectory(filepath.Dir(path)); e != nil {
		return e
	}
	file, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	_, writeErr := file.Write(data)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	dir, e := os.Open(filepath.Dir(path))
	if e != nil {
		return e
	}
	defer dir.Close()
	return dir.Sync()
}
func Save(directory, id string, envelope Envelope) (string, error) {
	if !regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,100}$`).MatchString(id) {
		return "", errors.New("invalid observation ID")
	}
	if e := PrivateDirectory(directory); e != nil {
		return "", e
	}
	path := filepath.Join(directory, id+".json")
	if e := CreatePrivate(path, Bytes(envelope)); e != nil {
		return "", e
	}
	return path, nil
}
