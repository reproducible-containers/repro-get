Usage:

```bash
docker run -it --rm \
  -v $(command -v repro-get):/usr/local/bin/repro-get:ro -v $(pwd):/mnt \
  ubuntu:jammy-20221003@sha256:35fb073f9e56eb84041b0745cb714eff0f7b225ea9e024f703cab56aaa5c7720 \
  sh -euxc 'repro-get install /mnt/SHA256SUMS-amd64 && neofetch'
```
:warning: Ubuntu requires specifying a custom `--provider=<YOUR_OWN_PROVIDER>` for long-term persistence of old packages.
