package version

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
)

func TestDetectDownloadable(t *testing.T) {
	if testing.Short() {
		t.Skip("slow test")
	}
	testCases := []string{
		"v0.1.2",
		Latest,
		Auto,
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc, func(t *testing.T) {
			d, err := DetectDownloadable(context.TODO(), tc)
			assert.NilError(t, err)
			t.Logf("%+v", d)
		})
	}
}
