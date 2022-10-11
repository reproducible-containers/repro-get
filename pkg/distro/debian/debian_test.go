package debian

import (
	"bytes"
	"strings"
	"testing"

	"github.com/reproducible-containers/repro-get/pkg/distro"
	"github.com/reproducible-containers/repro-get/pkg/dpkgutil"
	"gotest.tools/v3/assert"
)

func TestGenerateHash(t *testing.T) {
	// s is from `apt-cache show bash hello` on Debian 11
	const s = `Package: bash
Version: 5.1-2+deb11u1
Essential: yes
Installed-Size: 6469
Maintainer: Matthias Klose <doko@debian.org>
Architecture: amd64
Replaces: bash-completion (<< 20060301-0), bash-doc (<= 2.05-1)
Depends: base-files (>= 2.1.12), debianutils (>= 2.15)
Pre-Depends: libc6 (>= 2.25), libtinfo6 (>= 6)
Recommends: bash-completion (>= 20060301-0)
Suggests: bash-doc
Conflicts: bash-completion (<< 20060301-0)
Description: GNU Bourne Again SHell
Description-md5: 3522aa7b4374048d6450e348a5bb45d9
Multi-Arch: foreign
Homepage: http://tiswww.case.edu/php/chet/bash/bashtop.html
Tag: admin::TODO, devel::TODO, devel::interpreter, implemented-in::c,
 interface::shell, interface::text-mode, role::program,
 scope::application, suite::gnu, uitoolkit::ncurses
Section: shells
Priority: required
Filename: pool/main/b/bash/bash_5.1-2+deb11u1_amd64.deb
Size: 1416508
MD5sum: 63a8cf18a82283bb5c3138a23bfda7a3
SHA256: f702ef058e762d7208a9c83f6f6bbf02645533bfd615c54e8cdcce842cd57377

Package: hello
Version: 2.10-2
Installed-Size: 280
Maintainer: Santiago Vila <sanvila@debian.org>
Architecture: amd64
Replaces: hello-debhelper (<< 2.9), hello-traditional
Depends: libc6 (>= 2.14)
Conflicts: hello-traditional
Breaks: hello-debhelper (<< 2.9)
Description: example package based on GNU hello
Description-md5: c4a4aec43084cfb4a44c959b27e3a6d6
Homepage: http://www.gnu.org/software/hello/
Tag: devel::debian, devel::examples, devel::lang:c, devel::lang:posix-shell,
 devel::packaging, implemented-in::c, interface::commandline,
 role::documentation, role::program, scope::utility, suite::debian,
 suite::gnu
Section: devel
Priority: optional
Filename: pool/main/h/hello/hello_2.10-2_amd64.deb
Size: 56132
MD5sum: 52b0cad2e741dd722c3e2e16a0aae57e
SHA256: 35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc

`
	var b bytes.Buffer
	hw := distro.NewHashWriter(&b)
	assert.NilError(t, generateHash(hw, strings.NewReader(s)))

	const expected = `f702ef058e762d7208a9c83f6f6bbf02645533bfd615c54e8cdcce842cd57377  pool/main/b/bash/bash_5.1-2+deb11u1_amd64.deb
35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc  pool/main/h/hello/hello_2.10-2_amd64.deb
`
	assert.Equal(t, expected, b.String())
}

func TestInstalled(t *testing.T) {
	// s is from `dpkg-query -f '${Package},${Version},${Architecture}\n' -W bash hello` on Debian 11
	const s = `bash,5.1-2+deb11u1,amd64
hello,2.10-2,amd64
`
	got, err := installed(strings.NewReader(s))
	assert.NilError(t, err)
	expected := map[string]dpkgutil.Dpkg{
		"bash:amd64": {
			Package:      "bash",
			Version:      "5.1-2+deb11u1",
			Architecture: "amd64",
		},
		"hello:amd64": {
			Package:      "hello",
			Version:      "2.10-2",
			Architecture: "amd64",
		},
	}
	assert.DeepEqual(t, expected, got)
}
