package pacmanutil

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Pacman struct {
	// ca-certificates-20220905-1-any.pkg.tar.zst
	Package      string `json:"Package"`      // "ca-certificates-bundle"
	Version      string `json:"Version"`      // "20220905-1"
	Architecture string `json:"Architecture"` // "any"
}

func ParseFilename(filename string) (*Pacman, error) {
	if !strings.HasSuffix(filename, ".pkg.tar.zst") {
		return nil, fmt.Errorf("expected *.pkg.tar.zst, got %q", filename)
	}
	base := filepath.Base(filename)
	trimmed := strings.TrimSuffix(base, ".pkg.tar.zst")
	return Split(trimmed)
}

func Split(trimmed string) (*Pacman, error) {
	lastDash := strings.LastIndex(trimmed, "-")
	if lastDash < 0 {
		return nil, fmt.Errorf("unexpected package string: %q", trimmed)
	}
	pkgVerRel, arch := trimmed[:lastDash], trimmed[lastDash+1:]
	nextLastDash := strings.LastIndex(pkgVerRel, "-")
	if nextLastDash < 0 {
		return nil, fmt.Errorf("unexpected package string: %q", trimmed)
	}
	pkgVer, rel := pkgVerRel[:nextLastDash], pkgVerRel[nextLastDash+1:]
	nextNextLastDash := strings.LastIndex(pkgVer, "-")
	if nextNextLastDash < 0 {
		return nil, fmt.Errorf("unexpected package string: %q", trimmed)
	}
	pkg, ver := pkgVer[:nextNextLastDash], pkgVer[nextNextLastDash+1:]
	return &Pacman{
		Package:      pkg,
		Version:      ver + "-" + rel,
		Architecture: arch,
	}, nil
}
