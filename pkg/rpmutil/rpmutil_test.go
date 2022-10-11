package rpmutil

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseFilename(t *testing.T) {
	got, err := ParseFilename("ca-certificates/2022.2.54/5.fc37/noarch/ca-certificates-2022.2.54-5.fc37.noarch.rpm")
	assert.NilError(t, err)
	expected := &RPM{
		Package:      "ca-certificates",
		Version:      "2022.2.54",
		Release:      "5.fc37",
		Architecture: "noarch",
	}
	assert.DeepEqual(t, expected, got)
}

func TestSplit(t *testing.T) {
	got, err := Split("gpg-pubkey-38ab71f4-60242b08")
	assert.NilError(t, err)
	expected := &RPM{
		Package:      "gpg-pubkey",
		Version:      "38ab71f4",
		Release:      "60242b08",
		Architecture: "", // (none)
	}
	assert.DeepEqual(t, expected, got)
}
