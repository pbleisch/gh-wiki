package cmd

import "runtime/debug"

// version is stamped into a build with
// -ldflags "-X github.com/pbleisch/gh-wiki/cmd.version=v1.2.3". Release builds
// from cli/gh-extension-precompile are not stamped, so resolveVersion falls
// back to the information Go embeds in the binary.
var version = ""

func resolveVersion(stamped string, info *debug.BuildInfo, ok bool) string {
	if stamped != "" {
		return stamped
	}
	if !ok {
		return "unknown"
	}
	if module := info.Main.Version; module != "" && module != "(devel)" {
		return module
	}
	var revision string
	var modified bool
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	if revision == "" {
		return "unknown"
	}
	if len(revision) > 12 {
		revision = revision[:12]
	}
	if modified {
		revision += "+dirty"
	}
	return revision
}
