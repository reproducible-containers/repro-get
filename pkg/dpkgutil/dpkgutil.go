package dpkgutil

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Dpkg struct {
	Package      string `json:"Package"`      // "hello"
	Version      string `json:"Version"`      // "2.10-2"
	Architecture string `json:"Architecture"` // "amd64"
}

func ParseFilename(filename string) (*Dpkg, error) {
	if !strings.HasSuffix(filename, ".deb") {
		return nil, fmt.Errorf("expected *.deb, got %q", filename)
	}
	base := filepath.Base(filename)
	return Split(strings.TrimSuffix(base, ".deb"))
}

func Split(trimmed string) (*Dpkg, error) {
	sp := strings.SplitN(trimmed, "_", 3)
	if len(sp) != 3 {
		return nil, fmt.Errorf("expected <PACKAGE>_<VERSION>_<ARCHITECTURE>, got %q", trimmed)
	}
	return &Dpkg{
		Package:      sp[0],
		Version:      sp[1],
		Architecture: sp[2],
	}, nil
}
