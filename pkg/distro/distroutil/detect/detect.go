// Forked from https://github.com/containerd/nerdctl/blob/v0.23.0/pkg/infoutil/infoutil_unix.go#L67-L110
/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package detect

import (
	"bufio"
	"errors"
	"io"
	"os"
	"regexp"

	"strings"

	"github.com/sirupsen/logrus"
)

func DistroID() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		logrus.WithError(err).Warn("failed to open /etc/os-release")
		return ""
	}
	defer f.Close()
	id, err := distroID(f)
	if err != nil {
		logrus.WithError(err).Warn("failed to get ID from /etc/os-release")
		return ""
	}
	return id
}

func distroID(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		k, v := getOSReleaseAttrib(line)
		switch k {
		case "ID":
			return v, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", errors.New("no ID was found")
}

var osReleaseAttribRegex = regexp.MustCompile(`([^\s=]+)\s*=\s*("{0,1})([^"]*)("{0,1})`)

func getOSReleaseAttrib(line string) (string, string) {
	splitBySlash := strings.SplitN(line, "#", 2)
	l := strings.TrimSpace(splitBySlash[0])
	x := osReleaseAttribRegex.FindAllStringSubmatch(l, -1)
	if len(x) >= 1 && len(x[0]) > 3 {
		return x[0][1], x[0][3]
	}
	return "", ""
}
