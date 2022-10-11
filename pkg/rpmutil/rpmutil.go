package rpmutil

import (
	"fmt"
	"path/filepath"
	"strings"
)

type RPM struct {
	// ca-certificates-2022.2.54-5.fc37.noarch.rpm
	Package      string `json:"Package"`      // "ca-certificates-bundle"
	Version      string `json:"Version"`      // "2022.2.54"
	Release      string `json:"Release"`      // "5.fc37"
	Architecture string `json:"Architecture"` // "noarch"
}

func ParseFilename(filename string) (*RPM, error) {
	if !strings.HasSuffix(filename, ".rpm") {
		return nil, fmt.Errorf("expected *.rpm, got %q", filename)
	}
	base := filepath.Base(filename)
	trimmed := strings.TrimSuffix(base, ".rpm")
	return Split(trimmed)
}

func Split(trimmed string) (*RPM, error) {
	var pkgVerRel, arch string
	lastDot := strings.LastIndex(trimmed, ".")
	if lastDot >= 0 {
		pkgVerRel, arch = trimmed[:lastDot], trimmed[lastDot+1:]
	} else {
		pkgVerRel, arch = trimmed, ""
	}
	lastDash := strings.LastIndex(pkgVerRel, "-")
	if lastDash < 0 {
		return nil, fmt.Errorf("unexpected package string: %q", trimmed)
	}
	pkgVer, rel := pkgVerRel[:lastDash], pkgVerRel[lastDash+1:]
	nextLastDash := strings.LastIndex(pkgVer, "-")
	if nextLastDash < 0 {
		return nil, fmt.Errorf("unexpected package string: %q", trimmed)
	}
	pkg, ver := pkgVer[:nextLastDash], pkgVer[nextLastDash+1:]
	return &RPM{
		Package:      pkg,
		Version:      ver,
		Release:      rel,
		Architecture: arch,
	}, nil
}
