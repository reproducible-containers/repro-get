package apkutil

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseFilename(t *testing.T) {
	got, err := ParseFilename("v3.16/main/x86_64/ca-certificates-bundle-20220614-r0.apk")
	assert.NilError(t, err)
	expected := &APK{
		Package: "ca-certificates-bundle",
		Version: "20220614-r0",
	}
	assert.DeepEqual(t, expected, got)
}
