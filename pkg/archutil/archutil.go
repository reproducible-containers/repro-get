package archutil

import "runtime"

// OCIArchDashVariant returns a string like "amd64", "arm64", "arm-v7".
func OCIArchDashVariant() string {
	s := runtime.GOARCH
	if s == "arm" {
		// TODO: support v6
		s += "-v7"
	}
	return s
}
