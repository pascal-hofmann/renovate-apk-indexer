# Renovate APK Index Server

This project aims to create a http server for [renovate](https://github.com/renovatebot/renovate) to query apk packages.

# Usage

For every release a container image is built and stored in the github container registry: `docker run -p 3000:3000 ghcr.io/pascal-hofmann/renovate-apk-indexer`

```
$ renovate-apk-indexer -help
Usage of renovate-apk-indexer:
  -apk-index-url string
        comma-separated URLs of the apk indexes to get the package information from (default "https://dl-cdn.alpinelinux.org/{alpineVersion}/main/{architecture}/APKINDEX.tar.gz")
  -log-level value
        log level (default INFO)
  -log-output string
        representation for logs (text,json) (default "text")
  -update-interval int
        update interval of the apk package index in hours (default 4)
```

You can also specify multiple comma seperate indexes for distributions with multiple repositories:

```
$ renovate-apk-indexer -apk-index-url=https://dl-cdn.alpinelinux.org/{alpineVersion}/main/{architecture}/APKINDEX.tar.gz,https://dl-cdn.alpinelinux.org/{alpineVersion}/community/{architecture}/APKINDEX.tar.gz,https://dl-cdn.alpinelinux.org/{alpineVersion}/testing/{architecture}/APKINDEX.tar.gz
```

## Renovate Gitlab example

Renovate gitlab-job (abbreviated):

```yaml
renovate:
  services:
    - name: ghcr.io/pascal-hofmann/renovate-apk-indexer:v0.1.0
      alias: wolfi-apk
  script:
    - renovate
```

Use it in your renovate.json as a custom datasource:

```json
{
  "$schema": "https://docs.renovatebot.com/renovate-schema.json",
  "customDatasources": {
    "renovate-apk-indexer": {
      "defaultRegistryUrlTemplate": "http://renovate-apk-indexer:3000/{{packageName}}"
    }
  }
}
```

Usage in code:

```bash
# renovate: datasource=custom.renovate-apk-indexer depName=argo-cd
VERSION=2.13.1-r0
apk update && apk add argo-cd=$VERSION
```

Renovate should now be able to detect updates for the specified dependency.

## API

The server provides an endpoint `/apk/<ALPINE_VERSION>/<ARCHITECTURE>/<PACKAGE_NAME>`, which returns the package information of `<PACKAGE_NAME>` in the [format that renovate custom datasources expects](https://docs.renovatebot.com/modules/datasource/custom/)

## APK Index updates

By default, the server updates it's packages every 4 hours.

## Healthchecks

Liveness and Readiness probe endpoints are available at `/livez` and `/readyz`.
