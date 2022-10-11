package apkutil

import (
	"fmt"
	"path/filepath"
	"strings"
)

type APK struct {
	Package string `json:"Package"` // "ca-certificates-bundle"
	Version string `json:"Version"` // "20220614-r0"
}

func ParseFilename(filename string) (*APK, error) {
	if !strings.HasSuffix(filename, ".apk") {
		return nil, fmt.Errorf("expected *.apk, got %q", filename)
	}
	base := filepath.Base(filename)
	pkgDashVer := strings.TrimSuffix(base, ".apk")
	return Split(pkgDashVer)
}

func Split(pkgDashVer string) (*APK, error) {
	sp := strings.Split(pkgDashVer, "-")
	for i, f := range sp {
		if i >= 1 && '0' <= f[0] && f[0] <= '9' {
			return &APK{
				Package: strings.Join(sp[0:i], "-"),
				Version: strings.Join(sp[i:], "-"),
			}, nil
		}
	}
	return nil, fmt.Errorf("failed to split %q into the package name and the version string", pkgDashVer)
}
