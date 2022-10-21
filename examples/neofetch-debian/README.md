Usage:

```bash
docker run -it --rm \
  -v $(command -v repro-get):/usr/local/bin/repro-get:ro -v $(pwd):/mnt \
  debian:bullseye-20221004@sha256:e538a2f0566efc44db21503277c7312a142f4d0dedc5d2886932b92626104bff \
  sh -euxc 'repro-get install /mnt/SHA256SUMS-amd64 && neofetch'
```
