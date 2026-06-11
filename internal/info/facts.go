package info

import (
	"runtime/debug"
	"strings"
)

// devVersion is reported for development builds (`go build`/`go test` from a
// checkout, where the embedded version is "(devel)", VCS-stamped, or "+dirty").
// Keeping it constant makes generated output deterministic for tests.
const devVersion = "v1.31.1"

// Version is the running version. When the binary was installed from a clean
// module release (`go install github.com/rlpeck/crdb-sqlc/...@vX.Y.Z`), it is
// that module version so `crdb-sqlc version` and generated files reflect the
// installed release; otherwise it falls back to devVersion.
var Version = readVersion()

func readVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		if v := bi.Main.Version; isReleaseVersion(v) {
			return v
		}
	}
	return devVersion
}

// isReleaseVersion reports whether v is a clean release tag (vMAJOR.MINOR.PATCH).
// It rejects "(devel)", "+dirty" builds, and pseudo-versions so development and
// test builds stay on devVersion and produce deterministic output.
func isReleaseVersion(v string) bool {
	if !strings.HasPrefix(v, "v") {
		return false
	}
	parts := strings.Split(v[1:], ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}
