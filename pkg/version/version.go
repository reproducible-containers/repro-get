package version

import (
	"runtime/debug"
	"strconv"
)

// Version can be fulfilled on compilation time: -ldflags="-X main.Version=v0.1.2"
var Version string

func GetVersion() string {
	if Version != "" {
		return Version
	}
	const unknown = "(unknown)"
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return unknown
	}
	// bi.Main.Version is always "(devel)" as of Go 1.19, but will change in the future:
	// https://github.com/golang/go/issues/50603#issuecomment-1076662671
	var (
		vcsRevision string
		vcsTime     string
		vcsModified bool
	)
	for _, f := range bi.Settings {
		switch f.Key {
		case "vcs.revision":
			vcsRevision = f.Value
		case "vcs.time":
			vcsTime = f.Value
		case "vcs.modified":
			vcsModified, _ = strconv.ParseBool(f.Value)
		}
	}
	if vcsRevision == "" {
		return unknown
	}
	v := vcsRevision
	if vcsModified {
		v += ".m"
	}
	if vcsTime != "" {
		v += " [" + vcsTime + "]"
	}
	return v
}
