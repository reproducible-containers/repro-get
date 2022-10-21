Usage:

```bash
docker run -it --rm \
  -v $(command -v repro-get):/usr/local/bin/repro-get:ro -v $(pwd):/mnt \
  archlinux:base-20221009.0.92802@sha256:18dd035ceaa7e7296ef7c1e2cd52022ea95fbdccdc42650b13c0a995b5db3034 \
  sh -euxc 'repro-get install /mnt/SHA256SUMS-amd64 && neofetch'
```
