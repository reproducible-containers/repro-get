package sha256sums

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
)

var (
	ErrEmptyLine   = errors.New("empty line")
	ErrCommentLine = errors.New("comment line")
)

func ParseLine(origLine string) (sum, filename string, err error) {
	if strings.TrimSpace(origLine) == "" {
		return "", "", ErrEmptyLine
	}
	line := strings.TrimLeftFunc(origLine, unicode.IsSpace)
	if strings.HasPrefix(line, "#") {
		return "", "", ErrCommentLine
	}
	sp := strings.SplitN(line, " ", 2)
	if len(sp) != 2 {
		return "", "", fmt.Errorf("invalid line %q", origLine)
	}
	sum = sp[0]
	if len(sum) != 64 {
		return "", "", fmt.Errorf("invalid sha256 sum %q", sum)
	}
	filenameWithModePrefix := sp[1]
	filename = filenameWithModePrefix
	switch string(filenameWithModePrefix[0]) {
	case " ", "*": // " ": text mode, "*": binary mode (on non-Unix)
		filename = filenameWithModePrefix[1:]
	default: // no mode prefix
	}
	return sum, filename, nil
}

func Parse(r io.Reader) (mapByFilename map[string]string, err error) {
	sc := bufio.NewScanner(r)
	mapByFilename = make(map[string]string)
	for i := 0; sc.Scan(); i++ {
		line := sc.Text()
		var sum, filename string
		sum, filename, err = ParseLine(line)
		if err != nil {
			if errors.Is(err, ErrEmptyLine) || errors.Is(err, ErrCommentLine) {
				continue
			}
			err = fmt.Errorf("line %d: %w", i+1, err)
			return
		}
		mapByFilename[filename] = sum
	}
	err = sc.Err()
	return
}
