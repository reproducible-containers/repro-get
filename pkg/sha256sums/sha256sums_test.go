package sha256sums

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseLine(t *testing.T) {
	type testCase struct {
		line     string
		sum      string
		filename string
		err      string
	}
	testCases := []testCase{
		{
			line: "",
			err:  ErrEmptyLine.Error(),
		},
		{
			line: " ",
			err:  ErrEmptyLine.Error(),
		},
		{
			line: "# foo",
			err:  ErrCommentLine.Error(),
		},
		{
			line: " # foo",
			err:  ErrCommentLine.Error(),
		},
		{
			line: "foo",
			err:  "invalid line",
		},
		{
			line: "foo bar",
			err:  "invalid sha256",
		},
		{
			// text mode (on non-Unix)
			line:     "35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc  pool/main/h/hello/hello_2.10-2_amd64.deb",
			sum:      "35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc",
			filename: "pool/main/h/hello/hello_2.10-2_amd64.deb",
		},
		{
			// binary mode (on non-Unix)
			line:     "35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc *pool/main/h/hello/hello_2.10-2_amd64.deb",
			sum:      "35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc",
			filename: "pool/main/h/hello/hello_2.10-2_amd64.deb",
		},
		{
			// no mode identifier
			line:     "35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc pool/main/h/hello/hello_2.10-2_amd64.deb",
			sum:      "35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc",
			filename: "pool/main/h/hello/hello_2.10-2_amd64.deb",
		},
		{
			// Filename starts with a space
			line:     "35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc   pool/main/h/hello/hello_2.10-2_amd64.deb",
			sum:      "35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc",
			filename: " pool/main/h/hello/hello_2.10-2_amd64.deb",
		},
		{
			// Filename ends with a space
			line:     "35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc  pool/main/h/hello/hello_2.10-2_amd64.deb ",
			sum:      "35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc",
			filename: "pool/main/h/hello/hello_2.10-2_amd64.deb ",
		},
		{
			// Extra space before the hash (ignored)
			line:     " 35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc  pool/main/h/hello/hello_2.10-2_amd64.deb",
			sum:      "35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc",
			filename: "pool/main/h/hello/hello_2.10-2_amd64.deb",
		},
		{
			// Absolute path
			line:     " 35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc  /ipfs/QmRY19HEWeTJtRC6vAdz7rDfX3PjSMgXmd1KYi9guAACUj",
			sum:      "35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc",
			filename: "/ipfs/QmRY19HEWeTJtRC6vAdz7rDfX3PjSMgXmd1KYi9guAACUj",
		},
	}

	for _, tc := range testCases {
		sum, filename, err := ParseLine(tc.line)
		if tc.err == "" {
			assert.NilError(t, err)
			assert.Equal(t, tc.sum, sum)
			assert.Equal(t, tc.filename, filename)
		} else {
			assert.ErrorContains(t, err, tc.err)
		}
	}
}
