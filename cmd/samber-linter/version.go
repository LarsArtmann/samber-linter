package main

import "runtime/debug"

// fallbackVersion marks binaries built without module or VCS identity (e.g.
// hermetic nix builds). It is deliberately not a release number: a
// hand-pinned constant that outlives its release made analyzer version-
// gating impossible for consumers (the CV healthwash-gate instrument
// incident, 2026-09-20).
const fallbackVersion = "devel"

// version is the tool version reported by --version and in findings/SARIF
// exports. Releases can pin it with -ldflags "-X main.version=vX.Y.Z";
// unpinned builds resolve it from the build itself (see resolveVersion).
var version string

// resolveVersion derives the tool version from the build: the module proxy
// version for `go install module@vX` builds (the consumption path consumers
// version-gate on), otherwise the VCS revision for source builds. The build
// cannot go stale; a hand-maintained constant can.
func resolveVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return fallbackVersion
	}

	return buildVersion(bi)
}

// shortRevisionLength keeps version strings scannable; full SHAs are noise
// in a SARIF tool field.
const shortRevisionLength = 7

// buildVersion maps build info to a version string. Pure so the version
// contract is testable without building a release.
func buildVersion(info *debug.BuildInfo) string {
	if info == nil {
		return fallbackVersion
	}

	if v := info.Main.Version; v != "" && v != "(devel)" && v != "devel" {
		return v
	}

	var (
		revision string
		dirty    bool
	)

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			dirty = setting.Value == "true"
		}
	}

	if revision == "" {
		return fallbackVersion
	}

	if len(revision) > shortRevisionLength {
		revision = revision[:shortRevisionLength]
	}

	if dirty {
		return "devel+" + revision + ".dirty"
	}

	return "devel+" + revision
}
