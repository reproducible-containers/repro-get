package filespec

import (
	"strings"
	"testing"

	"github.com/reproducible-containers/repro-get/pkg/dpkgutil"
	"github.com/reproducible-containers/repro-get/pkg/sha256sums"
	"gotest.tools/v3/assert"
)

func TestNewFromSHA256SUMS(t *testing.T) {
	type testCase struct {
		sums     string
		expected map[string]*FileSpec
	}
	testCases := []testCase{
		{
			sums: `
# Simple
35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc  pool/main/h/hello/hello_2.10-2_amd64.deb
`,
			expected: map[string]*FileSpec{
				"pool/main/h/hello/hello_2.10-2_amd64.deb": &FileSpec{
					Name:     "pool/main/h/hello/hello_2.10-2_amd64.deb",
					Basename: "hello_2.10-2_amd64.deb",
					SHA256:   "35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc",
					Dpkg: &dpkgutil.Dpkg{
						Package:      "hello",
						Version:      "2.10-2",
						Architecture: "amd64",
					},
				},
			},
		},

		{
			sums: `
# With CID
35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc  pool/main/h/hello/hello_2.10-2_amd64.deb
35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc  /ipfs/QmRY19HEWeTJtRC6vAdz7rDfX3PjSMgXmd1KYi9guAACU
`,
			expected: map[string]*FileSpec{
				"pool/main/h/hello/hello_2.10-2_amd64.deb": &FileSpec{
					Name:     "pool/main/h/hello/hello_2.10-2_amd64.deb",
					Basename: "hello_2.10-2_amd64.deb",
					SHA256:   "35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc",
					CID:      "QmRY19HEWeTJtRC6vAdz7rDfX3PjSMgXmd1KYi9guAACU",
					Dpkg: &dpkgutil.Dpkg{
						Package:      "hello",
						Version:      "2.10-2",
						Architecture: "amd64",
					},
				},
			},
		},
		{
			sums: `
# With multiple CIDs
35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc  pool/main/h/hello/hello_2.10-2_amd64.deb
35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc  /ipfs/QmRY19HEWeTJtRC6vAdz7rDfX3PjSMgXmd1KYi9guAACU
35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc  /ipfs/QmTsD9EfB3Zu7DtLGWwDAkmnuhfjea5KyhXzNjd41LW35i
`,
			expected: map[string]*FileSpec{
				"pool/main/h/hello/hello_2.10-2_amd64.deb": &FileSpec{
					Name:     "pool/main/h/hello/hello_2.10-2_amd64.deb",
					Basename: "hello_2.10-2_amd64.deb",
					SHA256:   "35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc",
					CID:      "QmTsD9EfB3Zu7DtLGWwDAkmnuhfjea5KyhXzNjd41LW35i",
					Dpkg: &dpkgutil.Dpkg{
						Package:      "hello",
						Version:      "2.10-2",
						Architecture: "amd64",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		sums, err := sha256sums.Parse(strings.NewReader(tc.sums))
		assert.NilError(t, err)
		got, err := NewFromSHA256SUMS(sums)
		assert.NilError(t, err)
		assert.DeepEqual(t, tc.expected, got)
	}
}
