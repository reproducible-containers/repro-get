package arch

import (
	"strings"
	"testing"

	"github.com/reproducible-containers/repro-get/pkg/pacmanutil"
	"gotest.tools/v3/assert"
)

func TestInstalled(t *testing.T) {
	// s is from `pacman -Qi bash glibc ca-certificates`
	const s = `Name            : bash
Version         : 5.1.016-1
Description     : The GNU Bourne Again shell
Architecture    : x86_64
URL             : https://www.gnu.org/software/bash/bash.html
Licenses        : GPL
Groups          : None
Provides        : sh
Depends On      : readline  libreadline.so=8-64  glibc  ncurses
Optional Deps   : bash-completion: for tab completion
Required By     : base  bzip2  ca-certificates-utils  e2fsprogs  findutils  gawk  gdbm  gettext  gmp  gzip  icu  iptables  keyutils  libgpg-error  libksba  libpcap  npth  pacman  pcre2  systemd  xz
Optional For    : ncurses
Conflicts With  : None
Replaces        : None
Installed Size  : 8.19 MiB
Packager        : Felix Yan <felixonmars@archlinux.org>
Build Date      : Sat Jan 8 18:31:11 2022
Install Date    : Sun Oct 9 00:04:17 2022
Install Reason  : Installed as a dependency for another package
Install Script  : No
Validated By    : Signature

Name            : glibc
Version         : 2.36-6
Description     : GNU C Library
Architecture    : x86_64
URL             : https://www.gnu.org/software/libc
Licenses        : GPL  LGPL
Groups          : None
Provides        : None
Depends On      : linux-api-headers>=4.10  tzdata  filesystem
Optional Deps   : gd: for memusagestat
                  perl: for mtrace
Required By     : argon2  attr  audit  base  bash  brotli  bzip2  coreutils  device-mapper  expat  file  findutils  gawk  gcc-libs  gdbm  gnupg  grep  gzip  iproute2  json-c  kbd  keyutils  kmod  krb5  less  libassuan  libbpf  libcap  libcap-ng  libffi
                  libgpg-error  libksba  libmnl  libnfnetlink  libnghttp2  libnl  libp11-kit  libpcap  libsasl  libseccomp  libtasn1  libunistring  libverto  libxcrypt  lz4  mpfr  ncurses  npth  openssl  pacman  pam  pciutils  pinentry  popt  procps-ng  readline
                  sed  systemd-libs  tar  zlib  zstd
Optional For    : None
Conflicts With  : None
Replaces        : None
Installed Size  : 47.36 MiB
Packager        : Frederik Schwan <freswa@archlinux.org>
Build Date      : Fri Oct 7 14:17:58 2022
Install Date    : Sun Oct 9 00:04:16 2022
Install Reason  : Installed as a dependency for another package
Install Script  : Yes
Validated By    : Signature

Name            : ca-certificates
Version         : 20220905-1
Description     : Common CA certificates (default providers)
Architecture    : any
URL             : https://src.fedoraproject.org/rpms/ca-certificates
Licenses        : GPL
Groups          : None
Provides        : None
Depends On      : ca-certificates-mozilla
Optional Deps   : None
Required By     : curl
Optional For    : openssl
Conflicts With  : ca-certificates-cacert<=20140824-4
Replaces        : ca-certificates-cacert<=20140824-4
Installed Size  : 0.00 B
Packager        : Jan Alexander Steffens (heftig) <heftig@archlinux.org>
Build Date      : Mon Sep 5 21:59:24 2022
Install Date    : Sun Oct 9 00:04:18 2022
Install Reason  : Installed as a dependency for another package
Install Script  : No
Validated By    : Signature

`
	got, err := installed(strings.NewReader(s))
	assert.NilError(t, err)
	expected := map[string]pacmanutil.Pacman{
		"bash:x86_64": {
			Package:      "bash",
			Version:      "5.1.016-1",
			Architecture: "x86_64",
		},
		"glibc:x86_64": {
			Package:      "glibc",
			Version:      "2.36-6",
			Architecture: "x86_64",
		},
		"ca-certificates:any": {
			Package:      "ca-certificates",
			Version:      "20220905-1",
			Architecture: "any",
		},
	}
	assert.DeepEqual(t, expected, got)
}
