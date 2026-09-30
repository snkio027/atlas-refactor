// Package developmentprofile contains the two reviewed, finite development
// source/exposure bindings. It is not an arbitrary profile/plugin mechanism.
package developmentprofile

import "errors"

const (
	Repository          = "https://github.com/snkio027/atlas-refactor.git"
	RootPath            = "gitops/root/overlays/development"
	DevelopmentRevision = "codex/development-platform"
	OT1Revision         = "codex/ot1-desired-state"
	OT1Cluster          = "atlas-refactor-test-ot1"
	OT1SourceCommit     = "65af8497c02d22a60eb8bcaecf2434790edda2df"
	OT1Projection       = "ot1-isolated-instantiation/v1"
)

type Profile struct {
	HTTPPort, HTTPSPort int
	SnapshotSHA256      string
}

func Lookup(revision string) (Profile, error) {
	switch revision {
	case DevelopmentRevision:
		return Profile{8080, 8443, "6971d4560e39e6148f6155f7f4263181e9b8df62c1c8706ba6a9f00761ae7ab9"}, nil
	case OT1Revision:
		return Profile{18080, 18443, OT1SnapshotSHA256}, nil
	default:
		return Profile{}, errors.New("unreviewed development source")
	}
}
func Validate(schema int, cluster, repository, revision, path string) error {
	if repository != Repository || path != RootPath {
		return errors.New("development profile requires the reviewed repository and root")
	}
	if _, e := Lookup(revision); e != nil {
		return e
	}
	if revision == OT1Revision {
		if schema != 3 || cluster != OT1Cluster {
			return errors.New("OT-1 source is bound only to its isolated four-node identity")
		}
	} else if cluster == OT1Cluster {
		return errors.New("OT-1 identity cannot use the development authority branch")
	}
	return nil
}
