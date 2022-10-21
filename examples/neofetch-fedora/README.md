Usage:

```bash
docker run -it --rm \
  -v $(command -v repro-get):/usr/local/bin/repro-get:ro -v $(pwd):/mnt \
  fedora:36@sha256:2c5b21348e9b2a0b4c49bd5013be6d406be8594831aba21043393fcfba7252e0 \
  sh -euxc 'repro-get install /mnt/SHA256SUMS-amd64 && neofetch'
```
