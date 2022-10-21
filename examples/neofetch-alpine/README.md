Usage:

```bash
docker run -it --rm \
  -v $(command -v repro-get):/usr/local/bin/repro-get:ro -v $(pwd):/mnt \
  alpine:3.16.2@sha256:bc41182d7ef5ffc53a40b044e725193bc10142a1243f395ee852a8d9730fc2ad \
  sh -euxc 'repro-get install /mnt/SHA256SUMS-amd64 && neofetch'
```

:warning: Alpine requires specifying a custom `--provider=<YOUR_OWN_PROVIDER>` for long-term persistence of old packages.
