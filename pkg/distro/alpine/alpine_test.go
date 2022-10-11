package alpine

import (
	"net/url"
	"testing"

	"gotest.tools/v3/assert"
)

func TestURLToFilenameWithoutProvider(t *testing.T) {
	testCases := map[string]string{
		"https://dl-cdn.alpinelinux.org/alpine/v3.16/main/x86_64/ca-certificates-bundle-20220614-r0.apk": "v3.16/main/x86_64/ca-certificates-bundle-20220614-r0.apk",
	}
	for rawURL, expected := range testCases {
		u, err := url.Parse(rawURL)
		assert.NilError(t, err)
		got, err := urlToFilenameWithoutProvider(u)
		assert.NilError(t, err)
		assert.Equal(t, expected, got)
	}
}
