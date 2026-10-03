package workloadrun

import (
	"atlas-refactor/internal/workload"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sequenceFixture(t *testing.T) (*Workflow, Plan) {
	t.Helper()
	d, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(d, 0700); err != nil {
		t.Fatal(err)
	}
	w := &Workflow{Config: Config{StateDirectory: d}}
	p := Plan{Schema: 2, Parent: strings.Repeat("a", 40), BaseCommit: strings.Repeat("a", 40), Phases: publicationPhases(true), PhaseSHA256: map[string]string{}}
	for _, phase := range p.Phases[:3] {
		p.PhaseSHA256[phase] = workload.Digest([]byte(phase))
	}
	return w, p
}
func seedReceipts(t *testing.T, w *Workflow, p Plan, change func(string, *Publication)) {
	t.Helper()
	parent := p.Parent
	for _, phase := range p.Phases[:3] {
		receipt := Publication{1, workload.Digest(workload.JSON(p)), phase, parent, workload.Digest([]byte(phase))[:40], p.PhaseSHA256[phase]}
		parent = receipt.Commit
		if change != nil {
			change(phase, &receipt)
		}
		if err := save(w.publicationPath(p, phase), workload.JSON(receipt), true); err != nil {
			t.Fatal(err)
		}
	}
}
func TestPublicationCannotSkipPrerequisiteReceipts(t *testing.T) {
	w, p := sequenceFixture(t)
	if prior, err := w.precedingPublication(p, "permissions"); err != nil || prior != nil {
		t.Fatal(prior, err)
	}
	for _, phase := range []string{"project", "infrastructure", "consumer", "unknown"} {
		if _, err := w.precedingPublication(p, phase); err == nil {
			t.Fatal("accepted skip", phase)
		}
	}
	seedReceipts(t, w, p, nil)
	for i, phase := range p.Phases[1:] {
		prior, err := w.precedingPublication(p, phase)
		if err != nil || prior.Phase != p.Phases[i] {
			t.Fatal(phase, prior, err)
		}
	}
}
func TestPublicationRejectsBrokenReceiptChain(t *testing.T) {
	for _, phase := range []string{"permissions", "project", "infrastructure"} {
		for _, field := range []string{"schema", "plan", "phase", "parent", "commit", "tree"} {
			t.Run(phase+"/"+field, func(t *testing.T) {
				w, p := sequenceFixture(t)
				seedReceipts(t, w, p, func(step string, v *Publication) {
					if step != phase {
						return
					}
					switch field {
					case "schema":
						v.Schema = 0
					case "plan":
						v.PlanSHA256 = "other"
					case "phase":
						v.Phase = "consumer"
					case "parent":
						v.Parent = strings.Repeat("f", 40)
					case "commit":
						v.Commit = "HEAD"
					case "tree":
						v.TreeSHA256 = strings.Repeat("f", 64)
					}
				})
				if _, err := w.precedingPublication(p, "consumer"); err == nil {
					t.Fatal("accepted broken chain")
				}
			})
		}
	}
}
func TestFixedSequenceRejectsReorderingAndArbitraryResume(t *testing.T) {
	for _, phases := range [][]string{{"consumer"}, {"project", "permissions", "infrastructure", "consumer"}, {"permissions", "infrastructure", "consumer"}, {"permissions", "project", "infrastructure", "consumer", "consumer"}} {
		w, p := sequenceFixture(t)
		p.Phases = phases
		if _, err := w.precedingPublication(p, phases[0]); err == nil {
			t.Fatal("accepted custom sequence", phases)
		}
	}
	w, p := sequenceFixture(t)
	p.Parent = strings.Repeat("b", 40)
	p.Phases = publicationPhases(false)
	if prior, err := w.precedingPublication(p, "consumer"); err != nil || prior != nil {
		t.Fatal("completed update requires one consumer", err)
	}
}
func TestStoppedOrAmbiguousPublicationCannotBeRetried(t *testing.T) {
	for _, state := range []string{"intent", "stop", "receipt-and-stop"} {
		t.Run(state, func(t *testing.T) {
			w, p := sequenceFixture(t)
			target := w.publicationPath(p, "permissions")
			if state == "intent" {
				target += ".intent"
			} else {
				if state == "receipt-and-stop" {
					seedReceipts(t, w, p, nil)
				}
				target = filepath.Join(filepath.Dir(target), "terminal.json")
			}
			if err := save(target, []byte("{}"), true); err != nil {
				t.Fatal(err)
			}
			if err := w.publicationWritable(p, "permissions"); err == nil {
				t.Fatal("accepted mutation after STOP/unknown outcome")
			}
		})
	}
}
