// Package version reports build identity from Go's embedded build info.
package version

import "runtime/debug"

// String reports the module path and best available version from
// embedded build info: VCS revision for source builds, module version for
// proxy installs, "unknown" when neither exists (e.g. tarball builds).
func String() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "frame unknown"
	}
	commit, date, dirty := "", "", false
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			commit = s.Value
		case "vcs.time":
			date = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	version := "unknown"
	switch {
	case commit != "":
		version = commit
		if date != "" {
			version += " (committed " + date + ")"
		}
		if dirty {
			version += " [dirty]"
		}
	case bi.Main.Version != "" && bi.Main.Version != "(devel)":
		version = bi.Main.Version
	}
	return bi.Main.Path + " " + version
}
