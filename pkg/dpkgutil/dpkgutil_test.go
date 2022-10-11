package dpkgutil

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseFilename(t *testing.T) {
	got, err := ParseFilename("pool/main/h/hello/hello_2.10-2_amd64.deb")
	assert.NilError(t, err)
	expected := &Dpkg{
		Package:      "hello",
		Version:      "2.10-2",
		Architecture: "amd64",
	}
	assert.DeepEqual(t, expected, got)

	got, err = ParseFilename("pool/main/c/ca-certificates/ca-certificates_20210119_all.deb")
	assert.NilError(t, err)
	expected = &Dpkg{
		Package:      "ca-certificates",
		Version:      "20210119",
		Architecture: "all",
	}
	assert.DeepEqual(t, expected, got)
}
