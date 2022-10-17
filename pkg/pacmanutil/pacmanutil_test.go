package pacmanutil

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseFilename(t *testing.T) {
	got, err := ParseFilename("packages/c/ca-certificates/ca-certificates-20220905-1-any.pkg.tar.zst")
	assert.NilError(t, err)
	expected := &Pacman{
		Package:      "ca-certificates",
		Version:      "20220905-1",
		Architecture: "any",
	}
	assert.DeepEqual(t, expected, got)
}
